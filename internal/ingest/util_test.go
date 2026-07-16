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

func TestCleanReadableTextCutsPublisherTrailingBlocks(t *testing.T) {
	input := `Argentina đã giành chiến thắng 2-1 trước Anh trong trận bán kết World Cup 2026. Đây là nội dung bài viết thật sự cần được giữ lại cho độc giả.

Tags: Argentina Tây Ban Nha Chung kết World Cup

Đọc nhiều trong World Cup
Hơn 5 triệu người ký đơn đề nghị FIFA loại tuyển Argentina

Thông tin doanh nghiệp - sản phẩm
Đồng hồ thông minh Garmin Forerunner 265S giá rẻ nhất thị trường
Tủ lạnh Xiaomi MRS72HMPAVN gây chú ý với giá rẻ`

	got := cleanReadableText(input)
	if !strings.Contains(got, "nội dung bài viết thật sự") {
		t.Fatalf("real article text was removed: %q", got)
	}
	for _, unwanted := range []string{"Tags:", "Đọc nhiều", "Garmin", "Tủ lạnh Xiaomi", "Thông tin doanh nghiệp"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("trailing publisher block remained: %q", got)
		}
	}
}

func TestCleanReadableTextRejectsSkyCookieWall(t *testing.T) {
	input := `Latest West Ham transfer news and rumours.
Sorry, this blog is currently unavailable. Please try again later.
This content is provided by Sky, who may be using cookies and other technologies.
We need your permission to use cookies. We are unable to verify if you have consented to cookies.
Enable Cookies
Allow Cookies Once
You can change your settings through Privacy Options.`

	if got := cleanReadableText(input); got != "" {
		t.Fatalf("cookie wall should not become article text: %q", got)
	}
	if !BlockedArticleText(input) {
		t.Fatal("Sky cookie wall was not detected")
	}
}

func TestBlockedArticleTextRejectsVietnameseTranslation(t *testing.T) {
	input := `Rất tiếc, blog này hiện không khả dụng. Vui lòng thử lại sau.
Để hiển thị nội dung này, chúng tôi cần sự cho phép của bạn để sử dụng cookie.
Rất tiếc, chúng tôi không thể xác minh xem bạn đã đồng ý với cookie hay chưa.
Bật Cookie
Cho phép Cookie một lần
Bạn có thể thay đổi cài đặt qua Tùy chọn quyền riêng tư.`

	if !BlockedArticleText(input) {
		t.Fatal("Vietnamese cookie wall was not detected")
	}
}

func TestBlockedArticleTextKeepsRealCookieReporting(t *testing.T) {
	input := strings.Repeat("Đây là nội dung phân tích thể thao thực tế, có dữ kiện và phát biểu từ các bên liên quan. ", 80) +
		"Bài viết cũng nhắc đến cookie settings và privacy options như một chi tiết phụ."
	if BlockedArticleText(input) {
		t.Fatal("long real article was incorrectly treated as a cookie wall")
	}
}

func TestCleanReadableTextRemovesPublisherWidgetsAfterBody(t *testing.T) {
	input := strings.Repeat("Đây là nội dung thể thao có thông tin về trận đấu, cầu thủ và diễn biến đáng chú ý. ", 12) + `

Trở lại chủ đề
Tặng sao cho bài viết hay
Chủ đề:
Giải bóng đá công nhân
Tuổi Trẻ Online Newsletters
Đăng ký ngay để nhận gói tin tức mới`

	got := cleanReadableText(input)
	if strings.Contains(got, "Tặng sao") || strings.Contains(got, "Chủ đề:") || strings.Contains(got, "Newsletters") {
		t.Fatalf("publisher widgets remained: %q", got)
	}
	if !strings.Contains(got, "Đây là nội dung thể thao") {
		t.Fatal("real article body was removed")
	}
}
