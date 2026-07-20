import { Footer, PageTitle } from "../ui";
import { pageMetadata } from "../lib";

export const metadata = pageMetadata({
  title: "Điều khoản sử dụng",
  description: "Điều khoản sử dụng dịch vụ và nội dung trên BaoTheX.",
  path: "/dieu-khoan",
});

export default function Page() {
  return (
    <>
      <main className="wrap static-page">
        <PageTitle
          eyebrow="Pháp lý"
          title="Điều khoản sử dụng"
          description="Khi sử dụng BaoTheX, bạn đồng ý với các điều khoản dưới đây."
        />
        <div className="static-prose">
          <h2>1. Nội dung tham khảo</h2>
          <p>
            Nội dung trên BaoTheX nhằm cung cấp thông tin thể thao tổng hợp và mang tính tham khảo.
            Chúng tôi nỗ lực bảo đảm tính chính xác nhưng không cam kết mọi thông tin đều hoàn toàn
            không có sai sót; hãy đối chiếu với nguồn gốc khi cần.
          </p>
          <h2>2. Tài khoản</h2>
          <p>
            Bạn chịu trách nhiệm bảo mật thông tin đăng nhập của mình và mọi hoạt động diễn ra dưới
            tài khoản đó. Vui lòng thông báo cho chúng tôi nếu phát hiện việc sử dụng trái phép.
          </p>
          <h2>3. Sử dụng hợp lệ</h2>
          <p>
            Bạn đồng ý không sử dụng dịch vụ để phát tán nội dung vi phạm pháp luật, quấy rối người
            khác, hoặc can thiệp vào hoạt động kỹ thuật của hệ thống.
          </p>
          <h2>4. Dịch vụ Premium</h2>
          <p>
            Một số tính năng yêu cầu gói Premium. Điều kiện thanh toán và quyền lợi được nêu tại
            trang <a href="/premium">Premium</a>.
          </p>
          <h2>5. Thay đổi</h2>
          <p>
            Chúng tôi có thể cập nhật điều khoản này theo thời gian. Việc tiếp tục sử dụng dịch vụ
            đồng nghĩa với việc bạn chấp nhận các thay đổi.
          </p>
        </div>
      </main>
      <Footer />
    </>
  );
}
