import Link from "next/link";
import { Footer } from "../ui";
import { AuthForm } from "../auth-form";
export default function Page() {
  return (
    <>
      <main className="auth">
        <div className="auth-box">
          <span className="tag">BaoTheX</span>
          <h1 style={{ fontSize: 32, marginTop: 8 }}>Chào mừng trở lại</h1>
          <p style={{ color: "var(--muted)", marginTop: 8 }}>
            Đăng nhập để theo dõi chủ đề, lưu bài và nhận bản tin cá nhân.
          </p>
          <AuthForm mode="login" />
          <p style={{ fontSize: 13, marginTop: 20 }}>
            Chưa có tài khoản?{" "}
            <Link href="/dang-ky" style={{ color: "var(--ember)" }}>
              Đăng ký ngay
            </Link>
          </p>
        </div>
      </main>
      <Footer />
    </>
  );
}
