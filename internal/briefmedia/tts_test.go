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
