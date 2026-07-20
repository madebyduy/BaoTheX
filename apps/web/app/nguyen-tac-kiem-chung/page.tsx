import { Footer, PageTitle } from "../ui";
import { pageMetadata } from "../lib";

export const metadata = pageMetadata({
  title: "Nguyên tắc kiểm chứng",
  description: "Cách BaoTheX xác minh thông tin và gắn nhãn mức độ xác thực cho từng tin thể thao.",
  path: "/nguyen-tac-kiem-chung",
});

export default function Page() {
  return (
    <>
      <main className="wrap static-page">
        <PageTitle
          eyebrow="Tòa soạn"
          title="Nguyên tắc kiểm chứng"
          description="Chúng tôi phân biệt rõ tin đồn, tin đang xác minh và tin đã xác nhận."
        />
        <div className="static-prose">
          <p>
            Thể thao đầy tin đồn chuyển nhượng và thông tin chưa được xác nhận. Thay vì đăng tất cả
            như nhau, BaoTheX gắn nhãn mức độ xác thực để bạn biết mình đang đọc gì.
          </p>
          <h2>Ba mức độ</h2>
          <ul>
            <li>
              <b>Tin đồn / một nguồn</b> — mới xuất hiện ở một nguồn, chưa được kiểm chứng chéo.
            </li>
            <li>
              <b>Đang xác minh</b> — có dấu hiệu đáng tin nhưng chưa đủ nguồn độc lập để khẳng định.
            </li>
            <li>
              <b>Đã xác nhận</b> — được nhiều nguồn độc lập, uy tín cùng xác nhận.
            </li>
          </ul>
          <h2>Đối chiếu nhiều nguồn</h2>
          <p>
            Khi nhiều nguồn cùng đưa một sự kiện, chúng tôi gom lại thành một câu chuyện và hiển thị
            số nguồn. Một bài “Góc nhìn” phân tích chỉ được đưa vào bàn biên tập khi sự kiện đã được
            xác nhận và có ít nhất ba nguồn độc lập.
          </p>
          <h2>Con người quyết định</h2>
          <p>
            Hệ thống hỗ trợ việc tổng hợp và xếp hạng, nhưng quyết định xuất bản các bài phân tích
            luôn thuộc về biên tập viên. Chúng tôi ưu tiên đúng hơn nhanh.
          </p>
        </div>
      </main>
      <Footer />
    </>
  );
}
