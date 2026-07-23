"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
export function AuthForm({ mode }: { mode: "login" | "register" }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const router = useRouter();
  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (submitting) return;
    setError("");
    setSubmitting(true);
    try {
      const response = await fetch(
        `${typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081"}/api/v1/auth/${mode}`,
        {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(
            mode === "login" ? { email, password } : { email, password, display_name: name },
          ),
        },
      );
      if (!response.ok) {
        setError("Email hoặc mật khẩu chưa đúng. Hãy kiểm tra lại và thử lần nữa.");
        return;
      }
      window.dispatchEvent(new Event("baothex:auth-changed"));
      router.push("/");
      router.refresh();
    } catch {
      setError("Không thể kết nối máy chủ. Vui lòng kiểm tra API rồi thử lại.");
    } finally {
      setSubmitting(false);
    }
  }
  return (
    <form className="form" onSubmit={submit}>
      {mode === "register" && (
        <label>
          Tên hiển thị
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Nguyễn Văn A"
          />
        </label>
      )}
      <label>
        Email
        <input
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="ban@example.com"
        />
      </label>
      <label>
        Mật khẩu
        <input
          type="password"
          required
          minLength={8}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="Tối thiểu 8 ký tự"
        />
      </label>
      {error && <div className="notice">{error}</div>}
      <button className="btn ember" type="submit" disabled={submitting}>
        {submitting ? "Đang xử lý…" : mode === "login" ? "Đăng nhập" : "Tạo tài khoản"}
      </button>
    </form>
  );
}
