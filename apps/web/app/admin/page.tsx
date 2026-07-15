import Link from "next/link";
import { Footer, PageTitle } from "../ui";
export default function Page() {
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Quản trị"
          title="Bảng điều khiển"
          description="Các màn hình quản trị kết nối với API admin của BaoTheX."
        />
        <div className="topic-grid section">
          <Link href="/admin/noi-dung" className="topic">
            <strong>Duyệt nội dung</strong>
            <div className="meta">Sửa, ẩn, gắn chủ đề →</div>
          </Link>
          <Link href="/admin/goc-nhin" className="topic">
            <strong>Bàn phân tích</strong>
            <div className="meta">Chọn cluster, tạo nháp và duyệt Góc nhìn →</div>
          </Link>
          <div className="topic">
            <strong>Nguồn dữ liệu</strong>
            <div className="meta">RSS, YouTube, Europe PMC</div>
          </div>
          <div className="topic">
            <strong>Tác vụ nền</strong>
            <div className="meta">Theo dõi fetch và retry</div>
          </div>
        </div>
      </main>
      <Footer />
    </>
  );
}
