"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import type { FanPassport } from "./lib";

const API = (typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081");

export function FanPassportPanel() {
  const [passport, setPassport] = useState<FanPassport | null>(null);
  const [unauthorized, setUnauthorized] = useState(false);
  useEffect(() => {
    fetch(`${API}/api/v1/me/fan-passport`, { credentials: "include", cache: "no-store" })
      .then((response) => {
        if (!response.ok) {
          setUnauthorized(response.status === 401);
          return Promise.reject();
        }
        return response.json();
      })
      .then((json) => setPassport(json.data ?? json))
      .catch(() => null);
  }, []);
  if (unauthorized)
    return (
      <div className="passport-locked">
        <p>Fan Passport là hồ sơ riêng tư của bạn.</p>
        <Link href="/dang-nhap">Đăng nhập để xem →</Link>
      </div>
    );
  if (!passport) return <div className="passport-locked">Đang tổng hợp hành trình fan…</div>;
  return (
    <section className="fan-passport">
      <header>
        <div>
          <span>FAN PASSPORT · PRIVATE</span>
          <h2>Hành trình của bạn</h2>
        </div>
        <b>
          {passport.points}
          <small>điểm</small>
        </b>
      </header>
      <div className="passport-stats">
        <div>
          <b>{passport.active_days}</b>
          <span>Ngày ghé</span>
        </div>
        <div>
          <b>{passport.current_streak}</b>
          <span>Chuỗi ngày</span>
        </div>
        <div>
          <b>{passport.articles_read}</b>
          <span>Bài đã đọc</span>
        </div>
        <div>
          <b>{passport.events_followed}</b>
          <span>Sự kiện theo dõi</span>
        </div>
        <div>
          <b>{passport.predictions_correct}</b>
          <span>Dự đoán đúng</span>
        </div>
      </div>
      <div className="passport-badges">
        <strong>Huy hiệu</strong>
        {passport.badges.length ? (
          passport.badges.map((badge) => <span key={badge}>◆ {badge}</span>)
        ) : (
          <p>Đọc bài, theo dõi sự kiện và thử quiz để nhận huy hiệu đầu tiên.</p>
        )}
      </div>
    </section>
  );
}
