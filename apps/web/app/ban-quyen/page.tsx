import { Footer, PageTitle } from "../ui";
import { pageMetadata } from "../lib";

export const metadata = pageMetadata({
  title: "Chính sách bản quyền",
  description:
    "Cách BaoTheX tôn trọng bản quyền nội dung gốc: tóm tắt, dẫn nguồn và quy trình xử lý khiếu nại.",
  path: "/ban-quyen",
});

export default function Page() {
  return (
    <>
      <main className="wrap static-page">
        <PageTitle
          eyebrow="Pháp lý"
          title="Chính sách bản quyền"
          description="BaoTheX tổng hợp có trách nhiệm và tôn trọng quyền của đơn vị phát hành gốc."
        />
        <div className="static-prose">
          <h2>Nội dung tổng hợp</h2>
          <p>
            BaoTheX là sản phẩm tổng hợp tin tức. Chúng tôi tóm tắt và biên tập lại thông tin bằng
            tiếng Việt, không đăng lại toàn văn bài viết của đơn vị khác. Với mỗi tin có nguồn bên
            ngoài, chúng tôi đặt liên kết tới bài gốc; bản quyền nội dung gốc thuộc về đơn vị phát
            hành.
          </p>
          <h2>Hình ảnh</h2>
          <p>
            Ảnh minh họa (nếu có) được dẫn từ nguồn gốc kèm liên kết. Nếu bạn là chủ sở hữu và không
            muốn hình ảnh của mình được sử dụng, chúng tôi sẽ gỡ theo yêu cầu.
          </p>
          <h2>Nội dung của BaoTheX</h2>
          <p>
            Phần tiêu đề, tóm tắt, điểm chính và các bài “Góc nhìn” do BaoTheX biên soạn thuộc bản
            quyền của BaoTheX. Vui lòng không sao chép khi chưa được phép.
          </p>
          <h2>Khiếu nại bản quyền</h2>
          <p>
            Nếu bạn cho rằng nội dung trên BaoTheX vi phạm quyền của mình, vui lòng liên hệ qua
            trang <a href="/lien-he">Liên hệ</a> với thông tin cụ thể về nội dung liên quan. Chúng
            tôi sẽ xem xét và phản hồi trong thời gian sớm nhất.
          </p>
        </div>
      </main>
      <Footer />
    </>
  );
}
