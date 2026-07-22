package briefmedia

import (
	"strings"
	"testing"
)

func TestSplitTranscriptNeverExceedsLimit(t *testing.T) {
	text := strings.Repeat("Một câu thể thao có đủ thông tin để đọc rõ ràng. ", 80)
	chunks := splitTranscript(text, 180)
	if len(chunks) < 2 {
		t.Fatalf("expected long paragraph to be split, got %d chunk", len(chunks))
	}
	for _, chunk := range chunks {
		if len([]rune(chunk)) > 180 {
			t.Fatalf("chunk exceeded limit: %d", len([]rune(chunk)))
		}
	}
}

func TestNormalizeSpeechTextCleansEditorialNoise(t *testing.T) {
	input := "HLV &amp; ĐT Việt Nam dự FIFA World Cup. &amp;apos;Trận đấu&amp;apos;. <b>BaoTheX</b> cập nhật…"
	got := normalizeSpeechText(input)
	for _, expected := range []string{
		"huấn luyện viên & đội tuyển Việt Nam",
		"Phi-pha Uôn Cúp",
		"Báo Thể Ích",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("normalized speech missing %q: %s", expected, got)
		}
	}
	if strings.Contains(got, "<b>") || strings.Contains(got, "&amp;") || strings.Contains(got, "&apos;") || strings.Contains(got, "…") {
		t.Fatalf("speech still contains markup noise: %s", got)
	}
}

func TestSplitTranscriptPrefersSentenceBoundaries(t *testing.T) {
	text := strings.Repeat("Đây là một câu hoàn chỉnh về thể thao. ", 24)
	chunks := splitTranscript(text, 170)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	for _, chunk := range chunks[:len(chunks)-1] {
		if !strings.HasSuffix(chunk, ".") {
			t.Fatalf("chunk ended mid-sentence: %q", chunk)
		}
	}
}

func TestValidateNarrationChunkRejectsSilentTruncation(t *testing.T) {
	text := strings.Repeat("nội dung thể thao rõ ràng ", 30)
	if err := validateNarrationChunk(text, 5); err == nil {
		t.Fatal("very short audio should be rejected for a long transcript")
	}
	if err := validateNarrationChunk(text, 40); err != nil {
		t.Fatalf("plausible narration was rejected: %v", err)
	}
}
