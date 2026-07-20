import { Footer, PageTitle } from "../ui";
import { pageMetadata } from "../lib";

// NOTE: cập nhật địa chỉ email/kênh liên hệ thật của tòa soạn trước khi phát hành.
const CONTACT_EMAIL = "lienhe@baothex.vn";

export const metadata = pageMetadata({
  title: "Liên hệ",
  description: "Liên hệ với tòa soạn BaoTheX: góp ý nội dung, hợp tác và yêu cầu bản quyền.",
  path: "/lien-he",
});

export default function Page() {
  return (
    <>
      <main className="wrap static-page">
        <PageTitle
          eyebrow="Tòa soạn"
          title="Liên hệ"
          description="Chúng tôi luôn lắng nghe góp ý và phản hồi từ bạn đọc."
        />
        <div className="static-prose">
          <h2>Email</h2>
          <p>
            Góp ý nội dung, báo lỗi thông tin, hợp tác hoặc yêu cầu về bản quyền, vui lòng gửi tới{" "}
            <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a>.
          </p>
          <h2>Báo lỗi hoặc yêu cầu đính chính</h2>
          <p>
            Nếu bạn thấy một tin chưa chính xác, hãy gửi kèm đường dẫn bài viết và mô tả ngắn gọn
            vấn đề. Chúng tôi ưu tiên xử lý các yêu cầu đính chính.
          </p>
          <h2>Yêu cầu bản quyền</h2>
          <p>
            BaoTheX tôn trọng quyền của các đơn vị phát hành. Nếu bạn là chủ sở hữu nội dung và muốn
            điều chỉnh cách trích dẫn hoặc gỡ liên kết, vui lòng nêu rõ nội dung liên quan; xem thêm{" "}
            <a href="/ban-quyen">Chính sách bản quyền</a>.
          </p>
        </div>
      </main>
      <Footer />
    </>
  );
}
