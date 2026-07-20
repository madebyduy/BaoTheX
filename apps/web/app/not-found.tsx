import Link from "next/link";
import { Footer } from "./ui";

export default function NotFound() {
  return (
    <>
      <main className="wrap error-page">
        <span className="error-code">404</span>
        <h1>Không tìm thấy trang</h1>
        <p>Trang bạn tìm có thể đã được chuyển, đổi đường dẫn hoặc không còn tồn tại.</p>
        <div className="error-actions">
          <Link className="btn ember" href="/">
            Về trang chủ
          </Link>
          <Link className="btn light" href="/danh-muc">
            Xem tin mới nhất
          </Link>
        </div>
      </main>
      <Footer />
    </>
  );
}
