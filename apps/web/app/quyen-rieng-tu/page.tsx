import { Footer, PageTitle } from "../ui";
import { pageMetadata } from "../lib";

export const metadata = pageMetadata({
  title: "Chính sách quyền riêng tư",
  description: "BaoTheX thu thập và sử dụng dữ liệu của bạn như thế nào.",
  path: "/quyen-rieng-tu",
});

export default function Page() {
  return (
    <>
      <main className="wrap static-page">
        <PageTitle
          eyebrow="Pháp lý"
          title="Chính sách quyền riêng tư"
          description="Chúng tôi thu thập tối thiểu dữ liệu cần thiết để vận hành dịch vụ."
        />
        <div className="static-prose">
          <h2>Dữ liệu chúng tôi thu thập</h2>
          <ul>
            <li>Thông tin tài khoản bạn cung cấp (email, tên hiển thị) khi đăng ký.</li>
            <li>Sở thích theo dõi (chủ đề, nhân vật, nguồn) để cá nhân hoá dòng tin.</li>
            <li>Dữ liệu sử dụng ẩn danh (bài đã đọc, tương tác) để cải thiện chất lượng gợi ý.</li>
          </ul>
          <h2>Mục đích sử dụng</h2>
          <p>
            Dữ liệu được dùng để cung cấp dòng tin cá nhân hoá, gửi bản tin bạn đăng ký (ví dụ qua
            Telegram) và cải thiện sản phẩm. Chúng tôi không bán dữ liệu cá nhân của bạn.
          </p>
          <h2>Cookie</h2>
          <p>
            Chúng tôi sử dụng cookie phiên để duy trì đăng nhập. Bạn có thể xoá cookie trong trình
            duyệt, nhưng một số tính năng cá nhân hoá có thể ngừng hoạt động.
          </p>
          <h2>Thông báo đẩy</h2>
          <p>
            Nếu bạn bật thông báo (Web Push hoặc Telegram), chúng tôi lưu thông tin cần thiết để gửi
            thông báo và sẽ ngừng ngay khi bạn tắt.
          </p>
          <h2>Quyền của bạn</h2>
          <p>
            Bạn có thể yêu cầu xem, chỉnh sửa hoặc xoá dữ liệu tài khoản của mình bằng cách liên hệ
            qua trang <a href="/lien-he">Liên hệ</a>.
          </p>
        </div>
      </main>
      <Footer />
    </>
  );
}
