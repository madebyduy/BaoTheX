package process

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"repwire/internal/postgres"
)

// Summarizer calls an LLM API to produce paraphrased summaries. It talks the
// Anthropic Messages API shape by default (configurable via base URL/model).
type Summarizer struct {
	client  *http.Client
	apiKey  string
	baseURL string
	model   string
	llm     *postgres.LLMRepo

	dailyBudgetUSD  float64
	maxCallsPerHour int
}

// NewSummarizer constructs a Summarizer.
func NewSummarizer(apiKey, baseURL, model string, dailyBudgetUSD float64, maxCallsPerHour int, llm *postgres.LLMRepo) *Summarizer {
	return &Summarizer{
		client:          &http.Client{Timeout: 60 * time.Second},
		apiKey:          apiKey,
		baseURL:         baseURL,
		model:           model,
		llm:             llm,
		dailyBudgetUSD:  dailyBudgetUSD,
		maxCallsPerHour: maxCallsPerHour,
	}
}

// Enabled reports whether an API key is configured.
func (s *Summarizer) Enabled() bool { return s.apiKey != "" }

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
	if s.dailyBudgetUSD <= 0 {
		return true, nil
	}
	spent, err := s.llm.SpendToday(ctx)
	if err != nil {
		return false, err
	}
	return spent < s.dailyBudgetUSD, nil
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
		// Some Gemini aliases ignore JSON mode for long prompts. Preserve the
		// model output instead of dropping a useful summary; callers can still
		// display it and retry with a stricter prompt later.
		text := strings.TrimSpace(raw)
		if text == "" {
			return nil, fmt.Errorf("parse article summary: %w", err)
		}
		out.Summary = &text
		out.KeyPoints = []string{}
	}
	if out.KeyPoints == nil {
		out.KeyPoints = []string{}
	}
	return &out, nil
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
// recording token usage/cost for budget tracking.
func (s *Summarizer) complete(ctx context.Context, prompt string, maxTokens int) (string, error) {
	if !s.Enabled() {
		return "", fmt.Errorf("summarizer: LLM_API_KEY not configured")
	}
	if ok, err := s.BudgetOK(ctx); err != nil {
		return "", err
	} else if !ok {
		return "", ErrBudgetExceeded
	}
	if err := s.llm.RecordAttempt(ctx, s.model); err != nil {
		return "", err
	}
	if strings.Contains(s.baseURL, "generativelanguage.googleapis.com") {
		return s.completeGemini(ctx, prompt, maxTokens)
	}
	body, _ := json.Marshal(anthropicRequest{
		Model:     s.model,
		MaxTokens: maxTokens,
		Messages:  []anthropicMessage{{Role: "user", Content: prompt}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 || out.Error != nil {
		msg := fmt.Sprintf("llm http %d", resp.StatusCode)
		if out.Error != nil {
			msg = out.Error.Message
		}
		return "", fmt.Errorf("summarizer: %s", msg)
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
	return sb.String(), nil
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

func (s *Summarizer) completeGemini(ctx context.Context, prompt string, maxTokens int) (string, error) {
	body, _ := json.Marshal(geminiRequest{
		Contents:         []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
		GenerationConfig: geminiGenerationConfig{MaxOutputTokens: maxTokens, Temperature: 0.1, ResponseMimeType: "application/json"},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", s.apiKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 || out.Error != nil {
		msg := fmt.Sprintf("gemini http %d", resp.StatusCode)
		if out.Error != nil {
			msg = out.Error.Message
		}
		return "", fmt.Errorf("gemini: %s", msg)
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
		return "", fmt.Errorf("gemini returned empty response")
	}
	return sb.String(), nil
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
