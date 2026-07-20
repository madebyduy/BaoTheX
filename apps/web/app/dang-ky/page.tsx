import Link from "next/link";
import { Footer } from "../ui";
import { pageMetadata } from "../lib";

export const metadata = pageMetadata({
  title: "Đăng ký",
  description: "Tạo tài khoản BaoTheX để theo dõi chủ đề yêu thích và nhận bản tin cá nhân hoá.",
  path: "/dang-ky",
  index: false,
});
import { AuthForm } from "../auth-form";
export default function Page() {
  return (
    <>
      <main className="auth">
        <div className="auth-box">
          <span className="tag">Bắt đầu miễn phí</span>
          <h1 style={{ fontSize: 32, marginTop: 8 }}>Tạo tài khoản</h1>
          <p style={{ color: "var(--muted)", marginTop: 8 }}>
            Chọn nội dung phù hợp với mục tiêu tập luyện của bạn.
          </p>
          <AuthForm mode="register" />
          <p style={{ fontSize: 13, marginTop: 20 }}>
            Đã có tài khoản?{" "}
            <Link href="/dang-nhap" style={{ color: "var(--ember)" }}>
              Đăng nhập
            </Link>
          </p>
        </div>
      </main>
      <Footer />
    </>
  );
}
