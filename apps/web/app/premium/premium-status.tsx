"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

type Premium = {
  subscription?: {
    status?: string;
    current_period_end?: string;
  };
};

export function PremiumStatus() {
  const [status, setStatus] = useState<"loading" | "guest" | "active" | "inactive">("loading");
  const [expiresAt, setExpiresAt] = useState("");

  useEffect(() => {
    fetch(`${API}/api/v1/premium/status`, { credentials: "include" })
      .then((response) => {
        if (response.status === 401) throw new Error("guest");
        if (!response.ok) throw new Error("inactive");
        return response.json();
      })
      .then((json) => {
        const data: Premium = json.data ?? json;
        const sub = data.subscription;
        const valid =
          sub?.status === "active" &&
          !!sub.current_period_end &&
          new Date(sub.current_period_end) > new Date();
        setStatus(valid ? "active" : "inactive");
        if (sub?.current_period_end)
          setExpiresAt(new Date(sub.current_period_end).toLocaleDateString("vi-VN"));
      })
      .catch((error: Error) => setStatus(error.message === "guest" ? "guest" : "inactive"));
  }, []);

  if (status === "loading") return null;
  if (status === "active") {
    return (
      <div className="premium-status active">
        <b>Premium đang hoạt động</b>
        <span>
          Audio 6h/20h và cảnh báo theo dõi đã sẵn sàng{expiresAt ? ` đến ${expiresAt}` : ""}.
        </span>
        <Link href="/cai-dat">Chỉnh lịch Telegram →</Link>
      </div>
    );
  }
  if (status === "guest") {
    return (
      <p className="premium-note">Đăng nhập để kích hoạt Premium và liên kết Telegram cá nhân.</p>
    );
  }
  return (
    <p className="premium-note">
      Premium không khóa tin thường; chỉ mở các trải nghiệm theo dõi và bản tin riêng.
    </p>
  );
}
