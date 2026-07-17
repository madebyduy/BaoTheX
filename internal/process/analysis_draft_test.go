package process

import (
	"strings"
	"testing"

	"repwire/internal/domain"
)

var testMaterials = []domain.AnalysisMaterial{
	{SourceName: "Tuổi Trẻ Thể thao"},
	{SourceName: "VnExpress Thể thao"},
}

func draftWithBody(body string) domain.AnalysisDraft {
	return domain.AnalysisDraft{Title: "Một tiêu đề có duyên", Body: body}
}

// longBody clears the word floor so each test isolates the rule it is about.
// It names a source, since that is now required on its own.
func longBody(extra string) string {
	return strings.Repeat("bóng đá Việt Nam đang chuẩn bị cho giải đấu lớn sắp tới ", 200) +
		" Theo Tuổi Trẻ Thể thao, đội tuyển đã trở về nước. " + extra
}

func TestValidateAnalysisDraftAcceptsProseAttribution(t *testing.T) {
	d := draftWithBody(longBody("VnExpress cho hay đội tuyển còn 25 cầu thủ."))
	if err := validateAnalysisDraft(d, testMaterials); err != nil {
		t.Fatalf("well-formed draft rejected: %v", err)
	}
}

func TestValidateAnalysisDraftRejectsBracketCitations(t *testing.T) {
	for _, bad := range []string{
		"Đội tuyển còn 25 cầu thủ [Nguồn: Tuổi Trẻ Thể thao].",
		"Trận đấu diễn ra ngày 24/7 (Nguồn: VnExpress).",
		"Số liệu này gây tranh cãi [nguồn: Soha].",
	} {
		if err := validateAnalysisDraft(draftWithBody(longBody(bad)), testMaterials); err == nil {
			t.Fatalf("footnote-style citation was accepted: %q", bad)
		}
	}
}

// The regression that prompted this rule: banning footnotes made the model stop
// naming anyone and invent "theo ghi nhận nội bộ" instead.
func TestValidateAnalysisDraftRejectsDraftThatNamesNoSource(t *testing.T) {
	body := strings.Repeat("Theo những ghi nhận từ nội bộ đội tuyển, mọi thứ vẫn ổn định. ", 160)
	if err := validateAnalysisDraft(draftWithBody(body), testMaterials); err == nil {
		t.Fatal("draft with no named publication was accepted")
	}
}

// A writer naturally drops the section suffix: "VnExpress" must satisfy a
// source stored as "VnExpress Thể thao".
func TestValidateAnalysisDraftAcceptsMastheadWithoutSuffix(t *testing.T) {
	body := strings.Repeat("đội tuyển Việt Nam chuẩn bị cho giải đấu lớn sắp tới ", 200) +
		" VnExpress cho hay danh sách đã chốt."
	if err := validateAnalysisDraft(draftWithBody(body), testMaterials); err != nil {
		t.Fatalf("masthead-only attribution rejected: %v", err)
	}
}

func TestValidateAnalysisDraftRejectsShortBody(t *testing.T) {
	if err := validateAnalysisDraft(draftWithBody("Quá ngắn để gọi là bài phân tích."), testMaterials); err == nil {
		t.Fatal("short draft was accepted")
	}
}

func TestValidateAnalysisDraftRejectsMissingTitle(t *testing.T) {
	d := domain.AnalysisDraft{Title: "   ", Body: longBody("")}
	if err := validateAnalysisDraft(d, testMaterials); err == nil {
		t.Fatal("draft without a title was accepted")
	}
}
