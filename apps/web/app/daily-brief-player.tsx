"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";
type Brief = { title: string; audio_url?: string; duration_seconds?: number; brief_date: string };

export function DailyBriefPlayer() {
  const [brief, setBrief] = useState<Brief | null>(null);
  useEffect(() => {
    void fetch(`${API}/api/v1/audio-briefs/latest`)
      .then((response) => (response.ok ? response.json() : Promise.reject()))
      .then((json) => setBrief(json.data ?? json))
      .catch(() => setBrief(null));
  }, []);
  return (
    <section className="morning-player">
      <div className="morning-player-head">
        <span className="audio-orb">▶</span>
        <div>
          <small>NGHE BÁO · BẢN TIN BUỔI SÁNG</small>
          <h3>{brief?.title || "Thể thao 6h sáng nay"}</h3>
        </div>
      </div>
      {brief?.audio_url ? (
        <audio controls preload="metadata" src={brief.audio_url} />
      ) : (
        <p>Bản tin mở rộng đang được tổng hợp từ nhiều nguồn và nhiều môn thể thao.</p>
      )}
      <div className="morning-player-foot">
        <span>
          {brief?.duration_seconds
            ? `${Math.max(1, Math.round(brief.duration_seconds / 60))} phút`
            : "Khoảng 6–8 phút"}
        </span>
        <Link href="/premium">Cá nhân hóa bản tin →</Link>
      </div>
    </section>
  );
}
