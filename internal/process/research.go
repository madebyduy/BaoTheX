package process

import (
	"context"
	"encoding/json"
	"fmt"

	"repwire/internal/domain"
)

const researchPrompt = `Bạn là trợ lý khoa học thể thao. Đọc abstract/full text và trả về JSON đúng schema, KHÔNG thêm text nào khác.
Chỉ dùng thông tin CÓ trong nội dung. KHÔNG suy diễn. Nếu không nêu → trả null (hoặc mảng rỗng).

{
  "question":     "Nghiên cứu muốn trả lời điều gì (ngôn ngữ đơn giản)",
  "participants": "Số người, tuổi, giới tính, kinh nghiệm tập, tình trạng sức khỏe (null nếu không rõ)",
  "intervention": "Chương trình tập, nhóm can thiệp, nhóm so sánh, thời lượng (null nếu không rõ)",
  "findings":     ["3-5 kết quả chính"],
  "not_proven":   "Nghiên cứu này CHƯA chứng minh được điều gì (BẮT BUỘC, không để trống)",
  "limitations":  ["cỡ mẫu", "thời gian", "đối tượng", "khả năng áp dụng"],
  "practical":    "Áp dụng thực tế, viết thận trọng, KHÔNG phải tư vấn y tế",
  "funding_note": "Funding / conflict of interest nếu có, null nếu không rõ"
}

Quy tắc: không dùng "chứng minh/phải/luôn luôn"; không đưa liều lượng như lời khuyên cá nhân;
"not_proven" là mục bắt buộc chống phóng đại.

TIÊU ĐỀ: %s
ABSTRACT/NỘI DUNG: %s`

// researchJSON mirrors the prompt schema for decoding.
type researchJSON struct {
	Question     *string  `json:"question"`
	Participants *string  `json:"participants"`
	Intervention *string  `json:"intervention"`
	Findings     []string `json:"findings"`
	NotProven    *string  `json:"not_proven"`
	Limitations  []string `json:"limitations"`
	Practical    *string  `json:"practical"`
	FundingNote  *string  `json:"funding_note"`
}

// SummarizeResearch produces the fixed 8-section research breakdown.
func (s *Summarizer) SummarizeResearch(ctx context.Context, title, abstract string) (*domain.ResearchBreakdown, error) {
	prompt := fmt.Sprintf(researchPrompt, title, clip(abstract, 12000))
	raw, err := s.complete(ctx, prompt, 1200)
	if err != nil {
		return nil, err
	}
	var j researchJSON
	if err := json.Unmarshal([]byte(extractJSON(raw)), &j); err != nil {
		return nil, fmt.Errorf("parse research breakdown: %w", err)
	}
	bd := &domain.ResearchBreakdown{
		Question:     j.Question,
		Participants: j.Participants,
		Intervention: j.Intervention,
		Findings:     nonNilSlice(j.Findings),
		NotProven:    j.NotProven,
		Limitations:  nonNilSlice(j.Limitations),
		Practical:    j.Practical,
		FundingNote:  j.FundingNote,
	}
	return bd, nil
}

func nonNilSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
