"use client";

import { useEffect, useState } from "react";

const API = (typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081");

// PerspectiveAction is an admin-only control on the article page. It asks the
// newsroom to draft a "Góc nhìn" — an evaluative take on the event — from THIS
// single article. The draft is not published here; it lands in the analysis desk
// for review, so this button only kicks off the job.
export function PerspectiveAction({ contentId }: { contentId: number }) {
  const [isAdmin, setIsAdmin] = useState(false);
  const [status, setStatus] = useState<"idle" | "loading" | "done" | "error">("idle");
  const [message, setMessage] = useState("");

  useEffect(() => {
    let active = true;
    fetch(`${API}/api/v1/auth/me`, { credentials: "include", cache: "no-store" })
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((j) => {
        if (active) setIsAdmin((j.data ?? j)?.role === "admin");
      })
      .catch(() => null);
    return () => {
      active = false;
    };
  }, []);

  if (!isAdmin) return null;

  async function generate() {
    if (status === "loading") return;
    setStatus("loading");
    setMessage("");
    try {
      const res = await fetch(`${API}/api/v1/admin/content/${contentId}/perspective`, {
        method: "POST",
        credentials: "include",
      });
      if (res.ok) {
        setStatus("done");
        setMessage(
          "Đã gửi yêu cầu — AI đang viết bản nháp Góc nhìn từ bài này. Vào Bàn phân tích để duyệt và xuất bản.",
        );
      } else {
        const j = await res.json().catch(() => null);
        setStatus("error");
        setMessage(
          j?.error?.message || j?.message || "Chưa tạo được Góc nhìn cho bài này. Thử lại sau.",
        );
      }
    } catch {
      setStatus("error");
      setMessage("Lỗi kết nối máy chủ. Thử lại sau.");
    }
  }

  return (
    <div className="perspective-action">
      <span className="perspective-tag">Công cụ tòa soạn</span>
      <button
        className="perspective-btn"
        type="button"
        onClick={generate}
        disabled={status === "loading" || status === "done"}
      >
        {status === "loading"
          ? "Đang tạo Góc nhìn…"
          : status === "done"
            ? "Đã gửi ✓"
            : "✦ Tạo Góc nhìn từ bài này"}
      </button>
      <a className="perspective-desk-link" href="/admin/goc-nhin">
        Bàn phân tích →
      </a>
      {message ? (
        <small className={status === "error" ? "perspective-msg error" : "perspective-msg"}>
          {message}
        </small>
      ) : null}
    </div>
  );
}
