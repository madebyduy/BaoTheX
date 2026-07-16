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

	dailyBudgetUSD  float64
	maxCallsPerHour int

	mu       sync.Mutex
	apiKeys  []string
	keyIndex int
}

// NewSummarizer constructs a Summarizer. apiKeys is the rotation pool; empty and
// blank entries are dropped. The keys are tried in order, advancing to the next
// on quota exhaustion.
func NewSummarizer(apiKeys []string, baseURL, model string, dailyBudgetUSD float64, maxCallsPerHour int, llm *postgres.LLMRepo) *Summarizer {
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

// retryBackoff returns the delay before the next attempt on the same key
// (0.5s, 1s, 1.5s, ...).
func retryBackoff(attempt int) time.Duration {
	return time.Duration(attempt) * 500 * time.Millisecond
}

// ErrBudgetExceeded is returned when today's LLM spend has hit the cap.
var ErrBudgetExceeded = fmt.Errorf("llm daily budget exceeded")

// BudgetOK reports whether there is remaining daily budget.
func (s *Summarizer) BudgetOK(ctx context.Context) (bool, error) {
	if s.maxCallsPerHour > 0 {
		calls, err := s.llm.CallsLastHour(ctx)
		if err != nil {
			return false, err
		}
		if calls >= s.maxCallsPerHour {
			return false, nil
		}
	}
	return s.dailyBudgetOK(ctx)
}

// dailyBudgetOK checks the hard spend ceiling without applying the shared
// hourly throughput cap. Admin-selected editorial analysis uses this check so
// bulk translation cannot starve a deliberate newsroom action.
func (s *Summarizer) dailyBudgetOK(ctx context.Context) (bool, error) {
	if s.dailyBudgetUSD <= 0 {
		return true, nil
	}
	spent, err := s.llm.SpendToday(ctx)
	if err != nil {
		return false, err
	}
	return spent < s.dailyBudgetUSD, nil
}

// EditorialBudgetOK is the budget gate for an admin-selected analysis. It
// keeps the hard daily spend limit while leaving the shared hourly throughput
// cap to background translation jobs.
func (s *Summarizer) EditorialBudgetOK(ctx context.Context) (bool, error) {
	return s.dailyBudgetOK(ctx)
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

// ExtractAnalysisClaims is stage one of the analysis pipeline. It does not
// write prose; it maps agreement, disagreement and source-exclusive claims.
func (s *Summarizer) ExtractAnalysisClaims(ctx context.Context, title string, materials []domain.AnalysisMaterial) (*domain.AnalysisClaims, error) {
	prompt := fmt.Sprintf(`Bạn là trưởng ban dữ liệu của một tòa soạn thể thao Việt Nam.
Từ hồ sơ nhiều nguồn dưới đây, chỉ trích xuất những điều có bằng chứng trong nguyên liệu.
Phân biệt rõ: điều các nguồn đồng thuận; điểm các nguồn vênh/mâu thuẫn; thông tin chỉ một nguồn nêu; câu hỏi chưa thể kết luận.
Mỗi ý phải ghi tên nguồn trong ngoặc. Không suy đoán và không biến tin đồn thành sự thật.
Chỉ trả JSON đúng schema:
{"consensus":["..."],"conflicts":["..."],"unique_claims":["..."],"open_questions":["..."]}

SỰ KIỆN: %s
NGUYÊN LIỆU:
%s`, title, clip(formatAnalysisMaterials(materials), 26000))
	_ = prompt // Kept for backward-compatible source context; use the repaired UTF-8 prompt below.
	raw, err := s.completeEditorial(ctx, analysisClaimsPrompt(title, materials), 2200)
	if err != nil {
		return nil, err
	}
	var out domain.AnalysisClaims
	if err := json.Unmarshal([]byte(extractJSON(raw)), &out); err != nil {
		return nil, fmt.Errorf("parse analysis claims: %w", err)
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
	if strings.TrimSpace(out.Title) == "" || len(strings.Fields(out.Body)) < 800 {
		return nil, fmt.Errorf("analysis draft is incomplete")
	}
	if out.KeyPoints == nil {
		out.KeyPoints = []string{}
	}
	return &out, nil
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
- Mọi dữ kiện, con số, tỷ số, thời gian và phát biểu phải gắn tên nguồn, ví dụ [Nguồn: ESPN].
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

// TranslationSummary contains one-pass translation output. Keeping this in a
// single request is materially cheaper than translating and summarizing in two
// separate calls.
type TranslationSummary struct {
	VietnameseTitle string   `json:"vietnamese_title"`
	VietnameseBody  string   `json:"vietnamese_body"`
	Summary         *string  `json:"summary"`
	KeyPoints       []string `json:"key_points"`
}

// TranslateAndSummarize performs the two reader-facing transformations in one
// model call and returns structured JSON for reliable storage.
func (s *Summarizer) TranslateAndSummarize(ctx context.Context, title, body string) (*TranslationSummary, error) {
	prompt := fmt.Sprintf(`Bạn là biên tập viên báo thể thao Việt Nam. Hãy dịch tiêu đề và toàn bộ nội dung tiếng Anh sang tiếng Việt tự nhiên, chính xác; giữ nguyên tên vận động viên, câu lạc bộ, giải đấu, tỷ số, số liệu và thời gian. Sau đó tóm tắt đúng nội dung, không thêm suy đoán.
Chỉ trả về JSON hợp lệ theo schema:
{"vietnamese_title":"tiêu đề tiếng Việt","vietnamese_body":"bản dịch đầy đủ","summary":"3-4 câu tóm tắt tiếng Việt","key_points":["3-5 ý chính"]}
Nếu nội dung không đủ, vẫn dịch phần có sẵn và để summary là null. Không dùng markdown fence.

TIÊU ĐỀ: %s
NỘI DUNG:
%s`, title, clip(body, 18000))
	raw, err := s.complete(ctx, prompt, 6500)
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
	budgetCheck := s.BudgetOK
	if editorial {
		budgetCheck = s.dailyBudgetOK
	}
	if ok, err := budgetCheck(ctx); err != nil {
		return "", err
	} else if !ok {
		return "", ErrBudgetExceeded
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
				slog.Warn("llm request failed, retrying same key",
					"key_index", idx, "attempt", attempt, "err", err)
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(retryBackoff(attempt)):
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

	// Record usage (best-effort). Rough Haiku-class pricing estimate.
	cost := estimateCost(out.Usage.InputTokens, out.Usage.OutputTokens)
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
		out.UsageMetadata.CandidatesTokenCount, 0)
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

// estimateCost is a coarse USD estimate ($1/Mtok in, $5/Mtok out).
func estimateCost(inTok, outTok int) float64 {
	return float64(inTok)/1_000_000*1.0 + float64(outTok)/1_000_000*5.0
}

// extractJSON returns the substring from the first '{' to the last '}' so we
// tolerate models that wrap JSON in prose or code fences.
func extractJSON(s string) string {
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
