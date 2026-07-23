"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { type AudioTrack, usePersistentAudio } from "./persistent-audio-player";

const API =
  typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";
type Edition = "morning" | "evening";
type Brief = {
  title: string;
  edition: Edition;
  audio_url?: string;
  duration_seconds?: number;
  brief_date: string;
};

export function DailyBriefPlayer() {
  const [briefs, setBriefs] = useState<Partial<Record<Edition, Brief>>>({});
  useEffect(() => {
    let active = true;
    const loadBriefs = async () => {
      try {
        const values = await Promise.all(
          (["morning", "evening"] as Edition[]).map(async (edition) => {
            const response = await fetch(`${API}/api/v1/audio-briefs/latest?edition=${edition}`, {
              cache: "no-store",
            });
            if (!response.ok) return null;
            const json = await response.json();
            return (json.data ?? json) as Brief;
          }),
        );
        if (!active) return;
        const today = localDateKey(new Date());
        const next: Partial<Record<Edition, Brief>> = {};
        values.forEach((brief) => {
          if (brief && localDateKey(new Date(brief.brief_date)) === today) {
            next[brief.edition] = brief;
          }
        });
        setBriefs(next);
      } catch {
        if (active) setBriefs({});
      }
    };
    void loadBriefs();
    const timer = window.setInterval(() => void loadBriefs(), 60_000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, []);

  return (
    <section className="brief-appointment">
      <div className="brief-appointment-title">
        <span>NGHE BÁO MỖI NGÀY</span>
        <h3>Hai cuộc hẹn thể thao</h3>
        <p>Bản tin đã đối chiếu nhiều nguồn, chỉ sử dụng nội dung tiếng Việt đã biên tập.</p>
      </div>
      <BriefCard edition="morning" time="06:00" brief={briefs.morning} />
      <BriefCard edition="evening" time="20:00" brief={briefs.evening} />
      <Link className="brief-personalize" href="/premium">
        Cá nhân hóa theo đội bạn theo dõi →
      </Link>
    </section>
  );
}

function localDateKey(date: Date) {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Ho_Chi_Minh",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(date);
}

function BriefCard({ edition, time, brief }: { edition: Edition; time: string; brief?: Brief }) {
  const { track, playing, playTrack } = usePersistentAudio();
  const audioTrack: AudioTrack | null = brief?.audio_url
    ? {
        id: `daily-brief:${edition}:${brief.brief_date}`,
        src: brief.audio_url,
        title: brief.title,
        subtitle: `${edition === "morning" ? "Bản tin 6h" : "Bản tin 20h"} · Báo Thể Ích`,
        href: "/",
      }
    : null;
  const isCurrent = !!audioTrack && track?.id === audioTrack.id;
  return (
    <div className={`brief-edition ${edition}`}>
      <div className="brief-time">
        <strong>{time}</strong>
        <span>{edition === "morning" ? "BẢN TIN SÁNG" : "BẢN TIN TỐI"}</span>
      </div>
      <div className="brief-player-copy">
        <b>{brief?.title || (edition === "morning" ? "Thể thao 6h" : "Thể thao 20h")}</b>
        <small>
          {brief?.duration_seconds
            ? `${Math.max(1, Math.round(brief.duration_seconds / 60))} phút`
            : "Đang chuẩn bị ấn bản mới"}
        </small>
      </div>
      {audioTrack ? (
        <button
          className={`brief-play-button ${isCurrent ? "active" : ""}`}
          type="button"
          onClick={() => playTrack(audioTrack)}
        >
          <span>{isCurrent && playing ? "Ⅱ" : "▶"}</span>
          {isCurrent && playing ? "Đang nghe" : isCurrent ? "Tiếp tục" : "Nghe ngay"}
        </button>
      ) : (
        <i>▶</i>
      )}
    </div>
  );
}
