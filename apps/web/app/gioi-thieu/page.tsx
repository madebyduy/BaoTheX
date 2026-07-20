import { Footer, PageTitle } from "../ui";
import { pageMetadata } from "../lib";

export const metadata = pageMetadata({
  title: "Giới thiệu",
  description:
    "BaoTheX là báo thể thao đa nguồn cho người Việt: tổng hợp, kiểm chứng nguồn và biên tập tin thể thao trong ngày.",
  path: "/gioi-thieu",
});

export default function Page() {
  return (
    <>
      <main className="wrap static-page">
        <PageTitle
          eyebrow="Về chúng tôi"
          title="Giới thiệu BaoTheX"
          description="Báo thể thao đa nguồn, đối chiếu thông tin và biên tập bằng tiếng Việt."
        />
        <div className="static-prose">
          <p>
            BaoTheX là một sản phẩm báo chí thể thao tổng hợp dành cho người Việt. Mỗi ngày, hệ
            thống của chúng tôi theo dõi các nguồn báo chí và kênh video thể thao uy tín trong nước
            và quốc tế, chọn lọc những tin đáng chú ý nhất, tóm tắt bằng tiếng Việt và luôn dẫn về
            nguồn gốc để bạn kiểm chứng.
          </p>
          <h2>Chúng tôi làm gì</h2>
          <ul>
            <li>Tổng hợp tin thể thao nổi bật từ nhiều nguồn, phân loại theo môn và giải đấu.</li>
            <li>Tóm tắt, kiểm chứng và đánh dấu mức độ xác thực của từng thông tin.</li>
            <li>
              Xuất bản “Góc nhìn” — bài phân tích đối chiếu nhiều nguồn, do biên tập viên đọc và
              chịu trách nhiệm.
            </li>
            <li>Cung cấp lịch thi đấu, kết quả có nguồn và bản tin cá nhân hoá qua Telegram.</li>
          </ul>
          <h2>Nguyên tắc</h2>
          <p>
            Chúng tôi không sao chép toàn văn bài của nguồn khác. Với tin quốc tế, BaoTheX viết tiêu
            đề, tóm tắt và điểm chính của riêng mình, đồng thời đặt liên kết nổi bật tới bài gốc —
            bản quyền nội dung gốc thuộc về đơn vị phát hành. Xem thêm{" "}
            <a href="/chinh-sach-bien-tap">Chính sách biên tập</a> và{" "}
            <a href="/nguyen-tac-kiem-chung">Nguyên tắc kiểm chứng</a>.
          </p>
          <p>
            Mọi góp ý, phản hồi hoặc yêu cầu gỡ nội dung, vui lòng gửi qua trang{" "}
            <a href="/lien-he">Liên hệ</a>.
          </p>
        </div>
      </main>
      <Footer />
    </>
  );
}
