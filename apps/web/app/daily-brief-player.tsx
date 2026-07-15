"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";
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
    void Promise.all(
      (["morning", "evening"] as Edition[]).map(async (edition) => {
        const response = await fetch(`${API}/api/v1/audio-briefs/latest?edition=${edition}`);
        if (!response.ok) return null;
        const json = await response.json();
        return (json.data ?? json) as Brief;
      }),
    )
      .then((values) => {
        const next: Partial<Record<Edition, Brief>> = {};
        values.forEach((brief) => {
          if (brief) next[brief.edition] = brief;
        });
        setBriefs(next);
      })
      .catch(() => setBriefs({}));
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

function BriefCard({ edition, time, brief }: { edition: Edition; time: string; brief?: Brief }) {
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
      {brief?.audio_url ? <audio controls preload="metadata" src={brief.audio_url} /> : <i>▶</i>}
    </div>
  );
}
