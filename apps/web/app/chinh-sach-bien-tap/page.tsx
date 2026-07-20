import { Footer, PageTitle } from "../ui";
import { pageMetadata } from "../lib";

export const metadata = pageMetadata({
  title: "Chính sách biên tập",
  description:
    "Cách BaoTheX chọn lọc, tóm tắt, dẫn nguồn và biên tập tin thể thao — từ khâu thu thập đến khi xuất bản.",
  path: "/chinh-sach-bien-tap",
});

export default function Page() {
  return (
    <>
      <main className="wrap static-page">
        <PageTitle
          eyebrow="Tòa soạn"
          title="Chính sách biên tập"
          description="Quy trình từ khi một tin xuất hiện đến khi được xuất bản trên BaoTheX."
        />
        <div className="static-prose">
          <h2>1. Nguồn tin</h2>
          <p>
            BaoTheX chỉ thu thập từ những nguồn báo chí và kênh chính thức đã được tòa soạn chọn
            lọc. Mỗi nguồn được gắn một mức độ uy tín; nguồn uy tín cao được ưu tiên hiển thị và đối
            chiếu.
          </p>
          <h2>2. Tóm tắt, không sao chép</h2>
          <p>
            Chúng tôi tóm tắt và biên tập lại thông tin bằng tiếng Việt. Với bài của nguồn nước
            ngoài, BaoTheX xuất bản tiêu đề, tóm tắt và các điểm chính của riêng mình, kèm liên kết
            nổi bật tới bài gốc. Chúng tôi không đăng lại toàn văn bài viết của đơn vị khác.
          </p>
          <h2>3. Đánh dấu mức độ xác thực</h2>
          <p>
            Mỗi tin được gắn nhãn “Tin đồn / một nguồn”, “Đang xác minh” hoặc “Đã xác nhận” tùy theo
            số nguồn độc lập cùng đưa tin. Xem chi tiết tại{" "}
            <a href="/nguyen-tac-kiem-chung">Nguyên tắc kiểm chứng</a>.
          </p>
          <h2>4. Bàn phân tích “Góc nhìn”</h2>
          <p>
            Những sự kiện lớn được đối chiếu qua nhiều nguồn và dựng thành bản thảo phân tích. Không
            bản thảo nào tự động xuất bản: một biên tập viên phải đọc, sửa và chịu trách nhiệm trước
            khi bài “Góc nhìn” lên trang.
          </p>
          <h2>5. Đính chính</h2>
          <p>
            Khi phát hiện sai sót, chúng tôi cập nhật bài viết và ghi rõ thời điểm “Cập nhật”. Nếu
            bạn thấy một thông tin chưa chính xác, vui lòng báo cho chúng tôi qua trang{" "}
            <a href="/lien-he">Liên hệ</a>.
          </p>
        </div>
      </main>
      <Footer />
    </>
  );
}
