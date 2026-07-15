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

func TestCleanReadableTextRemovesYouTubeConsentAndSourceControls(t *testing.T) {
	input := `Để hiển thị nội dung này từ YouTube, bạn phải bật tính năng theo dõi quảng cáo.
Chấp nhận
Quản lý lựa chọn của tôi
Một tiện ích mở rộng đang chặn trình phát video tải.
Thử lại
Đây là phần nội dung thể thao thật sự, đủ dài để hệ thống giữ lại và xuất bản cho độc giả.
Video thực hiện bởi:
Antoine BESSE
Đọc thêm
Một bài liên quan không được lẫn vào nội dung.`
	got := cleanReadableText(input)
	for _, unwanted := range []string{"theo dõi quảng cáo", "Chấp nhận", "Antoine BESSE", "Một bài liên quan"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("reader boilerplate remained in article: %q", got)
		}
	}
	if !strings.Contains(got, "phần nội dung thể thao thật sự") {
		t.Fatalf("real article text was removed: %q", got)
	}
}

func TestCleanReadableTextStopsBeforePublisherHotlineAndErrorForm(t *testing.T) {
	input := `Báo lỗi cho Soha
Thông báo bán vé cho trận đấu của đội tuyển Việt Nam. Đây là nội dung chính cần được giữ lại cho độc giả.

Tags
Việt Nam
bán vé
Copy link
https://example.com/article?tracking=` + strings.Repeat("x", 180) + `
Đường dây nóng:
0943 113 999
Báo lỗi cho Soha
*Vui lòng nhập đủ thông tin email hoặc số điện thoại
Gửi báo lỗi
Đóng`

	got := cleanReadableText(input)
	if !strings.Contains(got, "nội dung chính cần được giữ lại") {
		t.Fatalf("real article text was removed: %q", got)
	}
	for _, unwanted := range []string{"Báo lỗi cho Soha", "Copy link", "tracking=", "Đường dây nóng", "Gửi báo lỗi", "Đóng"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("publisher boilerplate remained in article: %q", got)
		}
	}
}
