package ingest

import (
	"strings"
	"testing"
)

func TestCleanReadableTextRemovesTrailingReadMore(t *testing.T) {
	input := "Đây là nội dung bài viết đủ dài để được giữ lại sau khi làm sạch.\n\nContinue reading..."
	got := cleanReadableText(input)
	if strings.Contains(strings.ToLower(got), "continue reading") {
		t.Fatalf("trailing read-more marker was not removed: %q", got)
	}
	if !strings.Contains(got, "Đây là nội dung bài viết") {
		t.Fatalf("article content was removed unexpectedly: %q", got)
	}
}

func TestCleanReadableTextRemovesVietnameseTrailingReadMore(t *testing.T) {
	input := "Phần nội dung tiếng Việt này đủ dài để được giữ lại nguyên vẹn.\n\nTiếp tục đọc…"
	got := cleanReadableText(input)
	if strings.Contains(strings.ToLower(got), "tiếp tục đọc") {
		t.Fatalf("Vietnamese read-more marker was not removed: %q", got)
	}
}
