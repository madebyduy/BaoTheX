package process

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"repwire/internal/domain"
	"repwire/internal/postgres"
	"repwire/internal/ratelimit"
)

// Summarizer calls an LLM API to produce paraphrased summaries. It talks the
// Anthropic Messages API shape by default (configurable via base URL/model).
//
// It holds a pool of API keys and rotates to the next one when the active key
// is rate-limited or quota-exhausted, so a single key running out does not stop
// generation while another key still has budget.
type Summarizer struct {
	client  *http.Client
	baseURL string
	model   string
	llm     *postgres.LLMRepo

	// pacer spaces requests so concurrent workers cannot burst past the
	// provider's per-minute limit. The hourly cap below bounds the total; this
	// bounds the rate, and they are not the same thing.
	pacer *ratelimit.Pacer

	pricing         Pricing
	dailyBudgetUSD  float64
	maxCallsPerHour int

	mu       sync.Mutex
	apiKeys  []string
	keyIndex int
}

// Pricing is what one million tokens costs the account behind LLM_API_KEY. It
// exists so the daily budget meter can describe the provider actually in use.
//
// These rates must match the provider named by LLM_BASE_URL, and nothing checks
// that for you. Both zero is the correct setting for a free tier, where calls
// genuinely cost nothing: the budget then never intervenes and the provider's
// own quota is the real limit — which is the honest arrangement, because a
// meter that invents spending would stop work that is not costing anything.
type Pricing struct {
	InputUSDPerMTok  float64
	OutputUSDPerMTok float64
}

// NewSummarizer constructs a Summarizer. apiKeys is the rotation pool; empty and
// blank entries are dropped. The keys are tried in order, advancing to the next
// on quota exhaustion.
//
// pacer may be nil to disable rate pacing. Pass the same Pacer given to the TTS
// renderer: both call the same Gemini project and therefore share one
// per-minute allowance, so pacing them separately would let the two of them
// stampede each other while each believed it was behaving.
func NewSummarizer(apiKeys []string, baseURL, model string, pricing Pricing, dailyBudgetUSD float64, maxCallsPerHour int, llm *postgres.LLMRepo, pacer *ratelimit.Pacer) *Summarizer {
	keys := make([]string, 0, len(apiKeys))
	for _, k := range apiKeys {
		if t := strings.TrimSpace(k); t != "" {
			keys = append(keys, t)
		}
	}
	return &Summarizer{
		client:          &http.Client{Timeout: 60 * time.Second},
		apiKeys:         keys,
		baseURL:         baseURL,
		model:           model,
		llm:             llm,
		pacer:           pacer,
		pricing:         pricing,
		dailyBudgetUSD:  dailyBudgetUSD,
		maxCallsPerHour: maxCallsPerHour,
	}
}

// Enabled reports whether at least one API key is configured.
func (s *Summarizer) Enabled() bool { return len(s.apiKeys) > 0 }

// currentKey returns the active key and its index.
func (s *Summarizer) currentKey() (int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.keyIndex, s.apiKeys[s.keyIndex]
}

// rotateFrom advances to the next key, but only if the active key is still the
// one that failed — so concurrent callers that already rotated don't get bumped
// backwards onto a key another goroutine just moved past.
func (s *Summarizer) rotateFrom(failed int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.keyIndex == failed {
		s.keyIndex = (s.keyIndex + 1) % len(s.apiKeys)
	}
}

// isQuotaErr reports whether an HTTP status/message indicates the key is
// rate-limited or out of quota.
func isQuotaErr(status int, msg string) bool {
	if status == http.StatusTooManyRequests {
		return true
	}
	m := strings.ToLower(msg)
	return strings.Contains(m, "resource_exhausted") ||
		strings.Contains(m, "quota") ||
		strings.Contains(m, "rate limit") ||
		strings.Contains(m, "rate_limit")
}

// retryPolicy decides what to do after a failed LLM request.
type retryPolicy int

const (
	policyFatal  retryPolicy = iota // don't retry or rotate — the request itself is bad
	policyRetry                     // retry the SAME key after a backoff, then rotate
	policyRotate                    // skip straight to the next key (this key is bad/blocked)
)

// maxAttemptsPerKey is how many times one key is tried (with backoff) before we
// give up on it and rotate to the next key in the pool. A single transient
// hiccup therefore never bounces us to another key.
const maxAttemptsPerKey = 3

// classify maps an HTTP status + message to a retry policy.
func classify(status int, msg string) retryPolicy {
	switch {
	case isQuotaErr(status, msg):
		return policyRetry // rate-limited / quota: back off, retry, then rotate
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return policyRotate // key invalid or blocked — another key may work
	case status >= 500:
		return policyRetry // transient server error
	case status >= 400:
		return policyFatal // bad request (prompt/model) — same for every key
	default:
		return policyRetry
	}
}

// Retry pacing lives in internal/ratelimit, shared with the TTS renderer: both
// talk to the same rate-limited Gemini endpoints and both once mistook a
// "slow down for 49s" for a dead key.

// ErrBudgetExceeded means today's LLM *spend* hit LLM_DAILY_BUDGET_USD.
var ErrBudgetExceeded = fmt.Errorf("llm daily budget exceeded")

// ErrHourlyCapReached means the LLM_MAX_CALLS_PER_HOUR throttle is full. It is
// a pacing limit, not a money limit, and it clears on its own within the hour —
// which is exactly why it must never be reported as a budget problem.
var ErrHourlyCapReached = fmt.Errorf("llm hourly call cap reached")

// BudgetOK reports whether background work may spend a call right now, i.e.
// both the hourly throughput cap and the daily spend ceiling allow it.
//
// Prefer BudgetStatus when the answer will reach a human: this bool cannot say
// which of the two ceilings said no, and they are fixed in completely different
// places.
func (s *Summarizer) BudgetOK(ctx context.Context) (bool, error) {
	err, checkErr := s.BudgetStatus(ctx)
	if checkErr != nil {
		return false, checkErr
	}
	return err == nil, nil
}

// BudgetStatus returns the specific reason background work must wait, or nil
// when it may proceed. The second error is for a failed check, not a refusal.
//
// Two ceilings guard the LLM and they are unrelated: LLM_MAX_CALLS_PER_HOUR
// paces throughput, LLM_DAILY_BUDGET_USD caps money. Reporting both as
// "daily budget exceeded" sent an operator to stare at a spend meter reading
// $0.00 of $0.50 while the real blocker — a 10-calls-per-hour throttle hit in
// the first few seconds — went unmentioned. An error that names the wrong
// ceiling is worse than no error: it actively misdirects.
func (s *Summarizer) BudgetStatus(ctx context.Context) (error, error) {
	if s.maxCallsPerHour > 0 {
		calls, err := s.llm.CallsLastHour(ctx)
		if err != nil {
			return nil, err
		}
		if calls >= s.maxCallsPerHour {
			return fmt.Errorf("%w: %d/%d calls used this hour (raise LLM_MAX_CALLS_PER_HOUR)",
				ErrHourlyCapReached, calls, s.maxCallsPerHour), nil
		}
	}
	if s.dailyBudgetUSD > 0 {
		spent, err := s.llm.SpendToday(ctx)
		if err != nil {
			return nil, err
		}
		if spent >= s.dailyBudgetUSD {
			return fmt.Errorf("%w: $%.4f/$%.2f spent today (raise LLM_DAILY_BUDGET_USD)",
				ErrBudgetExceeded, spent, s.dailyBudgetUSD), nil
		}
	}
	return nil, nil
}

// editorialBudgetStatus is the spend ceiling alone, skipping the hourly pacing
// throttle. A deliberate newsroom decision answers to money, not to the queue
// depth of background translation.
func (s *Summarizer) editorialBudgetStatus(ctx context.Context) (error, error) {
	if s.dailyBudgetUSD <= 0 {
		return nil, nil
	}
	spent, err := s.llm.SpendToday(ctx)
	if err != nil {
		return nil, err
	}
	if spent >= s.dailyBudgetUSD {
		return fmt.Errorf("%w: $%.4f/$%.2f spent today (raise LLM_DAILY_BUDGET_USD)",
			ErrBudgetExceeded, spent, s.dailyBudgetUSD), nil
	}
	return nil, nil
}

// EditorialBudgetOK is the budget gate for editorial analysis: the hard daily
// spend limit, without the hourly throughput cap that paces background jobs.
func (s *Summarizer) EditorialBudgetOK(ctx context.Context) (bool, error) {
	refusal, err := s.editorialBudgetStatus(ctx)
	if err != nil {
		return false, err
	}
	return refusal == nil, nil
}

// ArticleSummary is the expected JSON shape from the article prompt.
type ArticleSummary struct {
	Summary   *string  `json:"summary"`
	KeyPoints []string `json:"key_points"`
}

const articlePrompt = `Bạn là biên tập viên báo thể thao. Tóm tắt chính xác nội dung bài báo thể thao, giữ nguyên tên vận động viên, đội bóng, tỷ số, thời gian và giải đấu. Trả về JSON đúng schema, không thêm text nào khác.
{
  "summary": "3-4 câu, ngôn ngữ đơn giản, không phóng đại, viết lại hoàn toàn bằng lời của bạn (KHÔNG trích nguyên văn)",
  "key_points": ["ý 1", "ý 2", "ý 3"]
}
Nếu nội dung đầu vào không đủ để tóm tắt, trả {"summary": null, "key_points": []}.
Không dùng ngôn ngữ tuyệt đối ("chứng minh", "phải", "luôn luôn"). Không đưa liều lượng như lời khuyên cá nhân.

TIÊU ĐỀ: %s
NỘI DUNG: %s`

const articlePromptVI = `Bạn là biên tập viên báo thể thao Việt Nam. Tóm tắt chính xác nội dung, giữ nguyên tên vận động viên, câu lạc bộ, giải đấu, tỷ số, số liệu và thời gian. Không thêm suy đoán.
Chỉ trả về JSON hợp lệ theo schema:
{"summary":"3-4 câu tiếng Việt rõ ràng, trung lập","key_points":["3-5 ý chính"]}
Nếu dữ liệu không đủ, trả {"summary":null,"key_points":[]}.

TIÊU ĐỀ: %s
NỘI DUNG: %s`

// SummarizeArticle produces a paraphrased summary + key points for an article.
func (s *Summarizer) SummarizeArticle(ctx context.Context, title, body string) (*ArticleSummary, error) {
	prompt := fmt.Sprintf(articlePromptVI, title, clip(body, 8000))
	raw, err := s.complete(ctx, prompt, 700)
	if err != nil {
		return nil, err
	}
	var out ArticleSummary
	if err := json.Unmarshal([]byte(extractJSON(raw)), &out); err != nil {
		return nil, fmt.Errorf("parse article summary: %w", err)
	}
	if out.KeyPoints == nil {
		out.KeyPoints = []string{}
	}
	return &out, nil
}

// claimsMaxTokens must fit four arrays of sourced claims drawn from up to six
// materials. At 2,200 the model ran out mid-sentence and the response failed to
// parse, which threw away the call, the claims and the day's piece — the desk's
// single most common hard failure. Truncation is cheap to prevent and expensive
// to hit.
const claimsMaxTokens = 4000

// ExtractAnalysisClaims is stage one of the analysis pipeline. It does not
// write prose; it maps agreement, disagreement and source-exclusive claims.
func (s *Summarizer) ExtractAnalysisClaims(ctx context.Context, title string, materials []domain.AnalysisMaterial) (*domain.AnalysisClaims, error) {
	raw, err := s.completeEditorial(ctx, analysisClaimsPrompt(title, materials), claimsMaxTokens)
	if err != nil {
		return nil, err
	}
	var out domain.AnalysisClaims
	if err := json.Unmarshal([]byte(extractJSON(raw)), &out); err != nil {
		// A response cut off at the token ceiling is still mostly good evidence.
		// Salvage the complete entries rather than lose the call: the alternative
		// is no piece at all, and stage two only needs a map of what the sources
		// agree and disagree on, not every last item.
		repaired := repairTruncatedJSON(extractJSON(raw))
		if repairErr := json.Unmarshal([]byte(repaired), &out); repairErr != nil {
			return nil, fmt.Errorf("parse analysis claims: %w", err)
		}
		slog.Warn("analysis claims were truncated; recovered the complete entries",
			"consensus", len(out.Consensus), "conflicts", len(out.Conflicts))
	}
	if out.Consensus == nil {
		out.Consensus = []string{}
	}
	if out.Conflicts == nil {
		out.Conflicts = []string{}
	}
	if out.UniqueClaims == nil {
		out.UniqueClaims = []string{}
	}
	if out.OpenQuestions == nil {
		out.OpenQuestions = []string{}
	}
	return &out, nil
}

// WriteClusterAnalysis is stage two. It receives the evidence map from stage
// one and drafts a sourced article for human review; it never publishes.
func (s *Summarizer) WriteClusterAnalysis(ctx context.Context, title string, materials []domain.AnalysisMaterial, claims domain.AnalysisClaims) (*domain.AnalysisDraft, error) {
	claimJSON, _ := json.Marshal(claims)
	prompt := fmt.Sprintf(`Bạn là cây bút của Góc nhìn BaoTheX. Hãy viết một bản NHÁP báo chí tiếng Việt để biên tập viên người duyệt.

MỤC TIÊU BIÊN TẬP:
- Truyền tải rõ điều vừa xảy ra, vì sao người hâm mộ nên quan tâm và thông điệp BaoTheX muốn để lại sau câu chuyện.
- Đưa ra góc nhìn BaoTheX như một cách soi sáng sự kiện, KHÔNG biến bài thành bài nghị luận, KHÔNG dựng "luận điểm", KHÔNG ép chọn phe và KHÔNG trình bày kiểu dẫn chứng → suy luận → kết luận.
- Bài phải đọc tự nhiên như một cây bút có cá tính đang kể chuyện cho độc giả, không giống báo cáo tổng hợp hay dàn ý do máy tạo.

MẠCH BÀI GỢI Ý, KHÔNG PHẢI DANH SÁCH CỨNG:
1. Mở bằng chi tiết đáng nhớ hoặc một câu dẫn có duyên, đi thẳng vào chuyện.
2. Kể diễn biến và bối cảnh theo nhịp báo chí: rõ người, rõ việc, rõ điều mới; ưu tiên chi tiết giúp độc giả hình dung được câu chuyện.
3. Khi nhiều nguồn cùng đưa tin, nói gọn phần họ thống nhất; chỉ nêu điểm vênh khi điểm vênh đó thực sự làm thay đổi cách hiểu sự kiện.
4. Lồng góc nhìn BaoTheX vào đúng chỗ: chỉ ra nghịch lý, ý nghĩa, hệ quả hoặc điều đáng để người hâm mộ nhớ. Góc nhìn phải có căn cứ nhưng được viết thành lời bình tự nhiên, không gắn nhãn "luận điểm" và không lên lớp độc giả.
5. Kết bằng điều còn chờ đợi, hệ quả tiếp theo hoặc một hình ảnh/câu chữ đọng lại.

CÂU HỎI CHO ĐỘC GIẢ — CHỈ DÙNG KHI THẬT SỰ CẦN:
- Chỉ thêm TỐI ĐA MỘT câu hỏi khi nguyên liệu cho thấy có tranh cãi thật, lựa chọn khó, thông tin vênh nhau hoặc một quyết định chia rẽ người hâm mộ.
- Câu hỏi phải sắc, cụ thể và bám đúng dữ kiện trong bài; có thể đặt ở cuối đoạn liên quan hoặc cuối bài.
- Không tự tạo tranh cãi để câu tương tác. Với tin kết quả, lịch thi đấu, thông báo thông thường hoặc sự kiện không có mâu thuẫn đáng kể, TUYỆT ĐỐI không chèn câu hỏi.
- Tránh câu rỗng như "Bạn nghĩ sao?", tránh dẫn dắt độc giả công kích cá nhân và tránh câu hỏi có sẵn đáp án.

GIỌNG VĂN BAOTHEX:
- Hài hước, dí dỏm, duyên dáng và dễ tiếp cận; ưu tiên quan sát thông minh, ví von gọn hoặc một cú chơi chữ đúng lúc.
- Sự hài hước phải phục vụ thông điệp: đọc xong vừa thấy thú vị vừa hiểu thêm về trận đấu, con người hoặc xu thế. Không rải câu đùa theo định mức, không cố bắt trend và không biến bài thành tấu hài.
- Có thể "cà khịa" nhẹ màn trình diễn, chiến thuật, kỳ vọng hoặc nghịch lý của sự kiện; giọng điệu như người am hiểu thể thao trò chuyện với độc giả, không cay nghiệt và không phán xét từ trên xuống.
- Không dùng teencode, sáo ngữ mạng, câu cảm thán liên tục hoặc tiêu đề giật gân. Duyên dáng quan trọng hơn ồn ào.
- Không bôi nhọ, xúc phạm, chế giễu ngoại hình/đời tư/dân tộc/tôn giáo; không bịa phát ngôn hay giai thoại.

QUY TẮC BẤT DI BẤT DỊCH (giữ uy tín):
- Mọi khẳng định thực tế phải gắn tên nguồn; không bịa số liệu, tỷ số, thời gian; không phóng đại; không biến tin đồn thành sự thật.
- Tách bạch SỰ THẬT với GÓC NHÌN: lời bình, ví von hoặc đánh giá không được viết như một dữ kiện đã được xác nhận. Có thể dùng tự nhiên các cụm như "theo Góc nhìn BaoTheX", "nói vui thì", "công bằng mà nói" nhưng không lặp máy móc.
- Nói thẳng điều chưa chắc; không dùng từ tuyệt đối; không chép nguyên văn dài. Ký tên do hệ thống thêm sau.
Chỉ trả JSON đúng schema:
{"title":"tiêu đề rõ nội dung, có duyên nhưng không giật gân","summary":"3 câu tóm tắt giàu thông tin, dễ đọc","body":"bài 800-1300 từ, chia đoạn tự nhiên; chỉ dùng tiêu đề phụ khi giúp bài dễ đọc","key_points":["4-6 điểm đáng nhớ viết ngắn gọn"]}

SỰ KIỆN: %s
BẢN ĐỒ DỮ KIỆN, ĐIỂM ĐỒNG THUẬN VÀ ĐIỂM VÊNH: %s
NGUYÊN LIỆU ĐÃ DẪN NGUỒN:
%s`, title, string(claimJSON), clip(formatAnalysisMaterials(materials), 26000))
	_ = prompt // Kept for backward-compatible source context; use the repaired UTF-8 prompt below.
	raw, err := s.completeEditorial(ctx, analysisDraftPrompt(title, materials, claims), 7000)
	if err != nil {
		return nil, err
	}
	var out domain.AnalysisDraft
	if err := json.Unmarshal([]byte(extractJSON(raw)), &out); err != nil {
		return nil, fmt.Errorf("parse analysis draft: %w", err)
	}
	if err := validateAnalysisDraft(out, materials); err != nil {
		return nil, err
	}
	if out.KeyPoints == nil {
		out.KeyPoints = []string{}
	}
	return &out, nil
}

// analysisMinWords is the floor below which a draft is a stub rather than an
// article. The prompt asks for 1,200-1,800 words; this leaves room for a model
// that runs short without letting through something no editor could work with.
const analysisMinWords = 800

// footnoteCitation matches the bracketed source note — "[Nguồn: ESPN]",
// "(Nguồn: VnExpress)" — that the draft prompt used to ask for by example.
var footnoteCitation = regexp.MustCompile(`(?i)[\[(]\s*nguồn\s*:`)

// analysisDeskSuffixes are the desk labels a masthead carries in the sources
// table but never in prose: a writer says "theo VnExpress", not "theo VnExpress
// Thể thao". Checked longest-first so "Thể thao Việt Nam" is stripped before
// "Thể thao" can take a bite out of it.
var analysisDeskSuffixes = []string{
	"thể thao việt nam", "bóng đá quốc tế", "thể thao", "bóng đá",
	"sports", "sport", "football", "news", "daily",
}

// validateAnalysisDraft holds a drafted article to the two things that make it
// publishable-in-principle before a human ever opens it: it has to be a finished
// piece, and every fact in it has to be traceable to a publication by name.
//
// The attribution rule has two sides and both come from the same lesson. Bracket
// footnotes are a research artefact rather than journalism, and readers skip
// them — so they are banned. But banning them alone is what taught the model to
// stop naming anyone at all and reach for "theo ghi nhận nội bộ" instead, which
// is strictly worse: an unsourced claim wearing the costume of a sourced one. So
// a draft must also name, in prose, at least one of the publications it was
// built from. Ban one shape of attribution and you must require the other, or
// you have simply traded a visible citation for an invisible fabrication.
func validateAnalysisDraft(d domain.AnalysisDraft, materials []domain.AnalysisMaterial) error {
	if strings.TrimSpace(d.Title) == "" {
		return fmt.Errorf("analysis draft has no title")
	}
	if n := len(strings.Fields(d.Body)); n < analysisMinWords {
		return fmt.Errorf("analysis draft is too short: %d words, want at least %d", n, analysisMinWords)
	}
	if m := footnoteCitation.FindString(d.Body); m != "" {
		return fmt.Errorf("analysis draft attributes with a footnote (%q): name the publication in the sentence instead", strings.TrimSpace(m))
	}
	if !namesAnySource(d.Body, materials) {
		return fmt.Errorf("analysis draft names none of its %d sources: every fact must be traceable to a masthead", len(materials))
	}
	return nil
}

// namesAnySource reports whether the body credits at least one of the
// publications it was drafted from, accepting the masthead alone.
func namesAnySource(body string, materials []domain.AnalysisMaterial) bool {
	b := strings.ToLower(body)
	for _, m := range materials {
		name := strings.ToLower(strings.TrimSpace(m.SourceName))
		if name == "" {
			continue
		}
		if strings.Contains(b, name) {
			return true
		}
		if masthead := stripDeskSuffix(name); masthead != name && strings.Contains(b, masthead) {
			return true
		}
	}
	return false
}

// stripDeskSuffix reduces a stored source name to the masthead a writer would
// actually type. It returns name unchanged when nothing matches, so a source
// with no suffix is never shortened into something that could match by accident.
func stripDeskSuffix(name string) string {
	for _, suffix := range analysisDeskSuffixes {
		if trimmed := strings.TrimSuffix(name, " "+suffix); trimmed != name {
			return strings.TrimSpace(trimmed)
		}
	}
	return name
}

func formatAnalysisMaterials(materials []domain.AnalysisMaterial) string {
	var b strings.Builder
	for i, material := range materials {
		fmt.Fprintf(&b, "\n--- NGUỒN %d: %s (uy tín %d/5, xuất bản %v) ---\n", i+1, material.SourceName, material.SourceQuality, material.PublishedAt)
		fmt.Fprintf(&b, "TIÊU ĐỀ: %s\nTÓM TẮT: %s\nÝ CHÍNH: %s\nNỘI DUNG: %s\n",
			material.Title, material.Summary, strings.Join(material.KeyPoints, " | "), clip(material.Body, 7000))
	}
	return b.String()
}

func analysisClaimsPrompt(title string, materials []domain.AnalysisMaterial) string {
	return fmt.Sprintf(`Bạn là trưởng ban dữ liệu của một tòa soạn thể thao Việt Nam.
Đọc toàn bộ hồ sơ nhiều nguồn dưới đây. Chỉ trích xuất điều có căn cứ và ghi tên nguồn trong ngoặc.
Hãy tách rõ: các nguồn đồng thuận; các nguồn mâu thuẫn; thông tin chỉ một nguồn nêu; câu hỏi chưa thể kết luận.
Không suy đoán và không biến tin đồn thành sự thật. Chỉ trả JSON hợp lệ, không markdown:
{"consensus":["..."],"conflicts":["..."],"unique_claims":["..."],"open_questions":["..."]}

SỰ KIỆN: %s
HỒ SƠ NGUỒN:
%s`, title, formatAnalysisMaterialsClean(materials, 52000))
}

func analysisDraftPrompt(title string, materials []domain.AnalysisMaterial, claims domain.AnalysisClaims) string {
	claimJSON, _ := json.Marshal(claims)
	return fmt.Sprintf(`Bạn là cây bút của chuyên mục Góc nhìn BaoTheX. Hãy viết một bài báo tiếng Việt dài, giàu chi tiết để biên tập viên đọc và duyệt.

Độ dài bắt buộc: 1.200-1.800 từ, ít nhất 9 đoạn có thông tin thực chất. Không lặp ý và không kéo dài bằng câu rỗng.
Mở bài bằng một chi tiết đáng nhớ hoặc một câu dẫn hài hước có duyên, đi thẳng vào chuyện.
Kể rõ diễn biến và bối cảnh: ai, làm gì, ở đâu, khi nào, điều gì mới; giải thích vì sao người hâm mộ nên quan tâm.
Có một phần so sánh các nguồn: nguồn nào đồng thuận, nguồn nào nói khác, khác ở đâu và sự khác biệt làm thay đổi cách hiểu thế nào.
Đưa góc nhìn BaoTheX tự nhiên, hài hước, dí dỏm, duyên dáng và dễ tiếp cận. Có thể châm biếm nhẹ một nghịch lý thể thao, nhưng không xúc phạm, không bịa phát ngôn và không biến bài thành tấu hài.
Kết bằng điều còn bỏ ngỏ hoặc diễn biến tiếp theo. Chỉ đặt tối đa một câu hỏi sắc nếu hồ sơ thật sự có tranh cãi; tin kết quả hoặc lịch thi đấu bình thường thì không đặt câu hỏi.

QUY TẮC NGUỒN:
- Mọi dữ kiện, con số, tỷ số, thời gian và phát biểu phải gắn tên nguồn, viết thẳng trong câu như một cây bút thật: "Theo ESPN, ...", "VnExpress cho hay ...", "Tuổi Trẻ Thể thao dẫn lời ...".
- TUYỆT ĐỐI không dùng chú thích trong ngoặc kiểu [Nguồn: ESPN] hay (Nguồn: VnExpress). Đó là cách ghi của bài nghiên cứu, không phải của báo, và độc giả không đọc chúng.
- Phải gọi thẳng tên tờ báo. Không được viết chung chung như "theo ghi nhận nội bộ", "theo một nguồn tin" hay "theo truyền thông" khi hồ sơ đã cho biết đích danh tờ nào đưa tin.
- Tách dữ kiện với nhận xét. Nếu chưa chắc, phải nói rõ là chưa được xác nhận.
- Không suy đoán thành sự thật, không phóng đại, không sao chép nguyên văn dài.

Chỉ trả JSON hợp lệ, không markdown fence:
{"title":"tiêu đề rõ nội dung, có duyên nhưng không giật gân","summary":"3-4 câu tóm tắt giàu thông tin","body":"bài báo 1.200-1.800 từ, chia đoạn tự nhiên","key_points":["5-7 điểm đáng nhớ"]}

SỰ KIỆN: %s
BẢN ĐỒ ĐỒNG THUẬN VÀ ĐIỂM VÊNH: %s
HỒ SƠ ĐÃ DẪN NGUỒN:
%s`, title, string(claimJSON), formatAnalysisMaterialsClean(materials, 52000))
}

func formatAnalysisMaterialsClean(materials []domain.AnalysisMaterial, max int) string {
	var b strings.Builder
	for i, material := range materials {
		fmt.Fprintf(&b, "\n--- NGUỒN %d: %s (uy tín %d/5, xuất bản %v) ---\n", i+1, material.SourceName, material.SourceQuality, material.PublishedAt)
		fmt.Fprintf(&b, "TIÊU ĐỀ: %s\nTÓM TẮT: %s\nÝ CHÍNH: %s\nNỘI DUNG: %s\n",
			material.Title, material.Summary, strings.Join(material.KeyPoints, " | "), clip(material.Body, 10000))
	}
	return clip(b.String(), max)
}

// TranslateToVietnamese translates source text while preserving meaning and paragraph structure.
func (s *Summarizer) TranslateToVietnamese(ctx context.Context, body string) (string, error) {
	prompt := fmt.Sprintf(`Dịch nội dung sau sang tiếng Việt tự nhiên, rõ ràng. Giữ nguyên ý nghĩa, số liệu, tên riêng và cấu trúc đoạn. Chỉ trả về bản dịch, không thêm lời dẫn.\n\nNỘI DUNG:\n%s`, clip(body, 18000))
	return s.complete(ctx, prompt, 6000)
}

// ForeignDigest is what a foreign article becomes for readers: a Vietnamese
// headline, the points that matter, and a few sentences of context. No body.
//
// This is deliberately not a translation. Republishing a full Vietnamese copy of
// a Reuters or Guardian article is a derivative work of someone else's
// reporting, and it costs roughly seven times as much to produce: measured on
// live traffic, a full translation ran ~1,240 output tokens to make something
// averaging 0.67 views. A digest is a few hundred tokens, it is the standard
// defensible aggregation pattern, and — because the reader who does not speak
// English cannot use a link to the original — it has to stand on its own. The
// summary is the product, not a teaser for one.
type ForeignDigest struct {
	VietnameseTitle string   `json:"vietnamese_title"`
	Summary         string   `json:"summary"`
	KeyPoints       []string `json:"key_points"`
}

// digestPrompt asks for reader-facing Vietnamese without reproducing the source.
//
// The length here is a judgement call that only survived contact with the live
// page: at 3-4 sentences the article read as a stub — you learned the news and
// hit a link, with half the screen empty. Eight to ten sentences fills the page
// and carries the detail a reader who cannot open an English original actually
// needs, while still being our writing rather than a copy of theirs.
const digestPrompt = `Bạn là biên tập viên báo thể thao Việt Nam. Đọc bài báo tiếng nước ngoài dưới đây rồi viết lại bằng tiếng Việt cho độc giả KHÔNG đọc được tiếng Anh.

YÊU CẦU:
- Người đọc chỉ xem phần bạn viết, không mở bài gốc. Vì vậy phần viết phải ĐỦ Ý và tự đứng được: đọc xong là nắm trọn câu chuyện, không thấy thiếu.
- Có bối cảnh: chuyện gì xảy ra, ai liên quan, vì sao đáng chú ý, ảnh hưởng ra sao.
- Giữ nguyên tên vận động viên, câu lạc bộ, giải đấu, tỷ số, số liệu, thời gian.
- Viết lại bằng lời của bạn, mạch lạc như một bài báo ngắn. TUYỆT ĐỐI không dịch nguyên văn từng câu và không sao chép đoạn dài.
- Trung lập, không phóng đại, không suy đoán. Nếu bài gốc có phát biểu đáng chú ý, hãy thuật lại ý thay vì trích dài.

Chỉ trả JSON hợp lệ, không markdown fence:
{"vietnamese_title":"tiêu đề tiếng Việt rõ nội dung","summary":"8-10 câu tiếng Việt, chia 2-3 đoạn ngăn cách bằng \n\n, đủ để hiểu trọn tin mà không cần bài gốc","key_points":["5-7 ý chính, mỗi ý một câu ngắn"]}

TIÊU ĐỀ GỐC: %s
NỘI DUNG GỐC:
%s`

// DigestForeign turns a foreign article into a Vietnamese headline, key points
// and a self-contained summary — the reader-facing form — without translating
// the body.
//
// The source is clipped harder than the old translation path (6k vs 18k chars):
// the tail of a match report rarely changes the summary, and input is the cheap
// half of a call we are trying to make cheap. The output ceiling has room for
// ten sentences plus seven key points; too tight and the model truncates
// mid-JSON, which fails the parse and wastes the whole call.
func (s *Summarizer) DigestForeign(ctx context.Context, title, body string) (*ForeignDigest, error) {
	prompt := fmt.Sprintf(digestPrompt, title, clip(body, 6000))
	raw, err := s.completeWithBudget(ctx, prompt, 2000, false)
	if err != nil {
		return nil, err
	}
	var out ForeignDigest
	if err := json.Unmarshal([]byte(extractJSON(raw)), &out); err != nil {
		return nil, fmt.Errorf("parse foreign digest: %w", err)
	}
	out.VietnameseTitle = strings.TrimSpace(out.VietnameseTitle)
	out.Summary = strings.TrimSpace(out.Summary)
	if out.VietnameseTitle == "" {
		out.VietnameseTitle = title
	}
	if out.KeyPoints == nil {
		out.KeyPoints = []string{}
	}
	if out.Summary == "" {
		return nil, fmt.Errorf("foreign digest returned no summary")
	}
	return &out, nil
}

// TranslationSummary contains one-pass translation output. Keeping this in a
// single request is materially cheaper than translating and summarizing in two
// separate calls.
//
// Only the editorial path still uses this. A full translation is expensive and
// legally awkward to publish, so it is now produced for one cluster a day and
// never shown to readers — it exists purely as raw material for the analysis
// prompt, which turns several sources into an original, sourced piece.
type TranslationSummary struct {
	VietnameseTitle string   `json:"vietnamese_title"`
	VietnameseBody  string   `json:"vietnamese_body"`
	Summary         *string  `json:"summary"`
	KeyPoints       []string `json:"key_points"`
}

// TranslateAndSummarize performs the two reader-facing transformations in one
// model call and returns structured JSON for reliable storage. It runs on the
// background budget: the shared hourly cap paces it alongside routine wire work.
func (s *Summarizer) TranslateAndSummarize(ctx context.Context, title, body string) (*TranslationSummary, error) {
	return s.translateAndSummarize(ctx, title, body, false)
}

// TranslateAndSummarizeEditorial is the same call on the editorial budget path:
// it skips the shared hourly throughput cap while still honouring the hard daily
// spend ceiling. The daily pick uses it so that once the newsroom has committed
// to a story, translating its materials cannot be starved by a backlog of
// routine articles queued ahead of it.
func (s *Summarizer) TranslateAndSummarizeEditorial(ctx context.Context, title, body string) (*TranslationSummary, error) {
	return s.translateAndSummarize(ctx, title, body, true)
}

func (s *Summarizer) translateAndSummarize(ctx context.Context, title, body string, editorial bool) (*TranslationSummary, error) {
	prompt := fmt.Sprintf(`Bạn là biên tập viên báo thể thao Việt Nam. Hãy dịch tiêu đề và toàn bộ nội dung tiếng Anh sang tiếng Việt tự nhiên, chính xác; giữ nguyên tên vận động viên, câu lạc bộ, giải đấu, tỷ số, số liệu và thời gian. Sau đó tóm tắt đúng nội dung, không thêm suy đoán.
Chỉ trả về JSON hợp lệ theo schema:
{"vietnamese_title":"tiêu đề tiếng Việt","vietnamese_body":"bản dịch đầy đủ","summary":"3-4 câu tóm tắt tiếng Việt","key_points":["3-5 ý chính"]}
Nếu nội dung không đủ, vẫn dịch phần có sẵn và để summary là null. Không dùng markdown fence.

TIÊU ĐỀ: %s
NỘI DUNG:
%s`, title, clip(body, 18000))
	raw, err := s.completeWithBudget(ctx, prompt, 6500, editorial)
	if err != nil {
		return nil, err
	}
	var out TranslationSummary
	if err := json.Unmarshal([]byte(extractJSON(raw)), &out); err != nil {
		if partial, ok := parsePartialTranslation(raw); ok {
			return unwrapTranslationSummary(partial), nil
		}
		text := strings.TrimSpace(raw)
		if text == "" {
			return nil, fmt.Errorf("parse translation summary: %w", err)
		}
		out.VietnameseBody = text
		out.KeyPoints = []string{}
	}
	if out.KeyPoints == nil {
		out.KeyPoints = []string{}
	}
	if strings.TrimSpace(out.VietnameseBody) == "" {
		return nil, fmt.Errorf("translation returned empty body")
	}
	if strings.TrimSpace(out.VietnameseTitle) == "" {
		out.VietnameseTitle = title
	}
	return unwrapTranslationSummary(&out), nil
}

func unwrapTranslationSummary(out *TranslationSummary) *TranslationSummary {
	text := strings.TrimSpace(out.VietnameseBody)
	if !strings.HasPrefix(text, "{") || !strings.Contains(text, "vietnamese_body") {
		return out
	}
	var nested TranslationSummary
	if err := json.Unmarshal([]byte(extractJSON(text)), &nested); err == nil && strings.TrimSpace(nested.VietnameseBody) != "" {
		return unwrapTranslationSummary(&nested)
	}
	if partial, ok := parsePartialTranslation(text); ok && partial.VietnameseBody != text {
		return unwrapTranslationSummary(partial)
	}
	return out
}

var partialTranslationBody = regexp.MustCompile(`(?s)"vietnamese_body"\s*:\s*"((?:\\.|[^"\\])*)"`)

// parsePartialTranslation handles a truncated JSON response from a long
// generation. It extracts the complete body field when the closing JSON
// object was cut off by the provider's output limit.
func parsePartialTranslation(raw string) (*TranslationSummary, bool) {
	match := partialTranslationBody.FindStringSubmatch(raw)
	if len(match) != 2 {
		return nil, false
	}
	body, err := strconv.Unquote(`"` + match[1] + `"`)
	if err != nil || strings.TrimSpace(body) == "" {
		return nil, false
	}
	return &TranslationSummary{VietnameseBody: body, KeyPoints: []string{}}, true
}

// ---- LLM transport (Anthropic Messages API shape) ----

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// complete sends a single-turn prompt and returns the model's text output,
// recording token usage/cost for budget tracking. It rotates through the API
// key pool, advancing to the next key when the active one is quota-exhausted.
func (s *Summarizer) complete(ctx context.Context, prompt string, maxTokens int) (string, error) {
	return s.completeWithBudget(ctx, prompt, maxTokens, false)
}

// completeEditorial reserves a narrow priority lane for an analysis that an
// editor explicitly selected. It may exceed the hourly call-count throttle,
// but never the configured daily spend ceiling or the provider's real quota.
func (s *Summarizer) completeEditorial(ctx context.Context, prompt string, maxTokens int) (string, error) {
	return s.completeWithBudget(ctx, prompt, maxTokens, true)
}

func (s *Summarizer) completeWithBudget(ctx context.Context, prompt string, maxTokens int, editorial bool) (string, error) {
	if !s.Enabled() {
		return "", fmt.Errorf("summarizer: LLM_API_KEY not configured")
	}
	// Editorial work answers only to the money ceiling; background work also has
	// to respect the hourly pacing throttle. Either way, surface the ceiling that
	// actually said no.
	var refusal error
	var checkErr error
	if editorial {
		refusal, checkErr = s.editorialBudgetStatus(ctx)
	} else {
		refusal, checkErr = s.BudgetStatus(ctx)
	}
	if checkErr != nil {
		return "", checkErr
	}
	if refusal != nil {
		return "", refusal
	}
	if err := s.llm.RecordAttempt(ctx, s.model); err != nil {
		return "", err
	}

	var lastErr error
	// Outer loop walks the key pool; inner loop retries one key a few times
	// (with backoff) before we give up on it and rotate. A key is only
	// abandoned after it keeps failing — so a brief hiccup won't cause churn.
	for k := 0; k < len(s.apiKeys); k++ {
		idx, key := s.currentKey()
		for attempt := 1; attempt <= maxAttemptsPerKey; attempt++ {
			// Wait for a slot before every request, retries included. Retries are
			// what turn a busy minute into a stampede: each of four worker slots
			// re-firing three times is twelve requests aimed at a five-per-minute
			// door, and each rejection burns an attempt the job cannot get back.
			if err := s.pacer.Wait(ctx); err != nil {
				return "", err
			}
			text, policy, err := s.completeOnce(ctx, key, prompt, maxTokens)
			if err == nil {
				return text, nil
			}
			lastErr = err
			if policy == policyFatal {
				return "", err // bad request — retrying or rotating won't help
			}
			if policy == policyRotate {
				break // key is bad/blocked; go straight to the next one
			}
			// policyRetry: back off and try the SAME key again, unless this key
			// has used up its attempts.
			if attempt < maxAttemptsPerKey {
				wait := ratelimit.Wait(attempt, err.Error())
				slog.Warn("llm request failed, retrying same key",
					"key_index", idx, "attempt", attempt, "wait", wait, "err", err)
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(wait):
				}
			}
		}
		slog.Warn("llm key exhausted after retries, rotating to next key",
			"key_index", idx, "keys", len(s.apiKeys), "err", lastErr)
		s.rotateFrom(idx)
	}
	return "", fmt.Errorf("summarizer: all %d API key(s) failed after retries: %w", len(s.apiKeys), lastErr)
}

// completeOnce performs a single request with the given key, returning the
// output text, the retry policy to apply on failure, and the error itself.
//
// Note which field actually selects the model on each provider. Anthropic takes
// it in the request body, so LLM_MODEL decides. Gemini puts it in the URL path
// (".../models/<model>:generateContent"), so LLM_BASE_URL decides and LLM_MODEL
// is only the label written to llm_usage. Changing LLM_MODEL alone against
// Gemini therefore bills you for the old model while your records name the new
// one — change both, or the numbers you reason about are fiction.
func (s *Summarizer) completeOnce(ctx context.Context, key, prompt string, maxTokens int) (string, retryPolicy, error) {
	if strings.Contains(s.baseURL, "generativelanguage.googleapis.com") {
		return s.completeGemini(ctx, key, prompt, maxTokens)
	}
	return s.completeAnthropic(ctx, key, prompt, maxTokens)
}

func (s *Summarizer) completeAnthropic(ctx context.Context, key, prompt string, maxTokens int) (string, retryPolicy, error) {
	body, _ := json.Marshal(anthropicRequest{
		Model:     s.model,
		MaxTokens: maxTokens,
		Messages:  []anthropicMessage{{Role: "user", Content: prompt}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewReader(body))
	if err != nil {
		return "", policyFatal, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", policyRetry, err // network/transport error — worth retrying
	}
	defer resp.Body.Close()

	var out anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", policyRetry, err
	}
	if resp.StatusCode >= 400 || out.Error != nil {
		msg := fmt.Sprintf("llm http %d", resp.StatusCode)
		if out.Error != nil {
			msg = out.Error.Message
		}
		return "", classify(resp.StatusCode, msg), fmt.Errorf("summarizer: %s", msg)
	}

	// Record usage (best-effort), priced at the configured provider's rates.
	cost := s.estimateCost(out.Usage.InputTokens, out.Usage.OutputTokens)
	_ = s.llm.RecordUsage(ctx, s.model, out.Usage.InputTokens, out.Usage.OutputTokens, cost)

	var sb strings.Builder
	for _, c := range out.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	return sb.String(), policyFatal, nil // policy ignored on success (err == nil)
}

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens  int     `json:"maxOutputTokens"`
	Temperature      float64 `json:"temperature"`
	ResponseMimeType string  `json:"responseMimeType,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (s *Summarizer) completeGemini(ctx context.Context, key, prompt string, maxTokens int) (string, retryPolicy, error) {
	body, _ := json.Marshal(geminiRequest{
		Contents:         []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
		GenerationConfig: geminiGenerationConfig{MaxOutputTokens: maxTokens, Temperature: 0.1, ResponseMimeType: "application/json"},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewReader(body))
	if err != nil {
		return "", policyFatal, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", key)
	resp, err := s.client.Do(req)
	if err != nil {
		return "", policyRetry, err // network/transport error — worth retrying
	}
	defer resp.Body.Close()
	var out geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", policyRetry, err
	}
	if resp.StatusCode >= 400 || out.Error != nil {
		msg := fmt.Sprintf("gemini http %d", resp.StatusCode)
		if out.Error != nil {
			msg = out.Error.Message
		}
		return "", classify(resp.StatusCode, msg), fmt.Errorf("gemini: %s", msg)
	}
	_ = s.llm.RecordUsage(ctx, s.model, out.UsageMetadata.PromptTokenCount,
		out.UsageMetadata.CandidatesTokenCount,
		s.estimateCost(out.UsageMetadata.PromptTokenCount, out.UsageMetadata.CandidatesTokenCount))
	var sb strings.Builder
	for _, candidate := range out.Candidates {
		for _, part := range candidate.Content.Parts {
			sb.WriteString(part.Text)
		}
	}
	if strings.TrimSpace(sb.String()) == "" {
		// Empty output can be a transient content-filter/streaming blip; retry.
		return "", policyRetry, fmt.Errorf("gemini returned empty response")
	}
	return sb.String(), policyFatal, nil // policy ignored on success (err == nil)
}

// estimateCost converts a call's token counts into the USD figure the daily
// budget meter accumulates.
//
// This used to be a package-level function hardcoded at $1/$5 per million
// tokens — Anthropic Haiku-class rates — with no way to say otherwise. That was
// tolerable only because the one path that called it was the Anthropic path;
// the Gemini path recorded a flat zero and LLM_DAILY_BUDGET_USD quietly meant
// nothing at all on the provider this deployment actually uses. Both halves of
// that were wrong in the same way: the meter described a provider rather than
// the provider, and neither could be corrected without editing code.
func (s *Summarizer) estimateCost(inTok, outTok int) float64 {
	return float64(inTok)/1_000_000*s.pricing.InputUSDPerMTok +
		float64(outTok)/1_000_000*s.pricing.OutputUSDPerMTok
}

// repairTruncatedJSON closes a JSON object that a model left open when it hit
// the token ceiling, discarding the entry it was midway through.
//
// It is a salvage, not a parser: it walks the text tracking string state and an
// open-bracket stack, cuts back past any half-written value, drops a dangling
// comma, and closes what is still open. Everything already complete survives.
// Returns the input unchanged when nothing is open, so a healthy response is
// untouched.
//
// It finds its own starting brace rather than trusting extractJSON, which gives
// up and returns the whole raw string when there is no closing '}' — precisely
// the truncated case this handles. Without that, any prose the model wrote
// before the JSON would be walked as if it were structure.
// It works by proposal and verification rather than by reasoning about JSON
// grammar. Deciding analytically whether a given cut is legal means tracking
// key-versus-value position at every nesting level — easy to get subtly wrong,
// and a subtly wrong repair corrupts evidence instead of dropping it. Instead it
// collects every plausible cut point, tries them longest-first, and returns the
// first candidate the standard library agrees is valid JSON.
func repairTruncatedJSON(s string) string {
	if start := strings.IndexByte(s, '{'); start > 0 {
		s = s[start:]
	}
	if json.Valid([]byte(s)) {
		return s
	}

	type frame struct {
		open byte
		at   int
	}
	var stack []frame
	// cut point -> the bracket stack open at that point, so each candidate knows
	// what it must close.
	type candidate struct {
		end   int
		stack []byte
	}
	var candidates []candidate
	snapshot := func(end int) {
		open := make([]byte, len(stack))
		for i, f := range stack {
			open[i] = f.open
		}
		candidates = append(candidates, candidate{end: end, stack: open})
	}

	inString, escaped := false, false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case ch == '\\':
				escaped = true
			case ch == '"':
				inString = false
				snapshot(i + 1) // a completed string: possibly a value, possibly a key
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{', '[':
			stack = append(stack, frame{open: ch, at: i})
			snapshot(i + 1)
		case '}', ']':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			snapshot(i + 1)
		case ',':
			snapshot(i + 1)
		}
	}

	for i := len(candidates) - 1; i >= 0; i-- {
		c := candidates[i]
		out := strings.TrimRight(s[:c.end], " \t\r\n")
		out = strings.TrimRight(out, ",")
		var b strings.Builder
		b.WriteString(out)
		for j := len(c.stack) - 1; j >= 0; j-- {
			if c.stack[j] == '{' {
				b.WriteByte('}')
			} else {
				b.WriteByte(']')
			}
		}
		if repaired := b.String(); json.Valid([]byte(repaired)) {
			return repaired
		}
	}
	return s // nothing salvageable; the caller reports the original parse error
}

// extractJSON returns the substring from the first '{' to the last '}' so we
// tolerate models that wrap JSON in prose or code fences.
// extractJSON returns the first complete JSON object in s, tolerating whatever
// a model wraps around it: a markdown fence, a lead-in sentence, a friendly
// sign-off, or a second object it decided to volunteer.
//
// It used to cut from the first '{' to the LAST '}', which is right only when
// nothing at all follows the payload. A stray closing brace, a "Hy vọng giúp
// ích!" sign-off, or a second object were each swept inside the slice, and
// translate jobs died five attempts deep on "invalid character '}' after
// top-level value". Counting depth is what fixes it, and the count has to
// respect string values: the '}' in "Tỷ số {2-1}" closes nothing, and an
// escaped quote inside a string does not end the string.
//
// When the object never closes — a response truncated at the token ceiling —
// everything from the opening brace is returned, because repairTruncatedJSON
// downstream can still salvage that, and returning the raw string would deny it
// the chance.
func extractJSON(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return s
	}
	var depth int
	var inString, escaped bool
	for i := start; i < len(s); i++ {
		switch c := s[i]; {
		case escaped:
			escaped = false
		case inString && c == '\\':
			escaped = true
		case c == '"':
			inString = !inString
		case inString:
			// Structural characters inside a string value are data.
		case c == '{':
			depth++
		case c == '}':
			if depth--; depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return s[start:]
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
