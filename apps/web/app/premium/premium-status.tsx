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
          Dòng tin riêng và Web Push đang hoạt động{expiresAt ? ` đến ${expiresAt}` : ""}; audio
          6h/20h vẫn miễn phí qua Telegram.
        </span>
        <Link href="/cai-dat">Chỉnh lịch Telegram →</Link>
      </div>
    );
  }
  if (status === "guest") {
    return (
      <p className="premium-note">
        Đăng nhập để liên kết Telegram và nhận miễn phí bản tin 6h/20h.
      </p>
    );
  }
  return (
    <p className="premium-note">
      Premium mở dòng tin chỉ theo dõi và Web Push; bản tin 6h/20h qua Telegram vẫn miễn phí.
    </p>
  );
}
