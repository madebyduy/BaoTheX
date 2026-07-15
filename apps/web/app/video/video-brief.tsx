"use client";

import { useEffect, useState } from "react";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

type VideoBrief = {
  title: string;
  video_url?: string;
  thumbnail_url?: string;
  duration_seconds?: number;
};

export function VideoBriefPlayer() {
  const [brief, setBrief] = useState<VideoBrief | null>(null);
  useEffect(() => {
    void fetch(`${API}/api/v1/video-briefs/latest`)
      .then((response) => (response.ok ? response.json() : Promise.reject()))
      .then((json) => setBrief(json.data ?? json))
      .catch(() => setBrief(null));
  }, []);
  if (!brief?.video_url) return null;
  return (
    <section className="video-brief-feature">
      <div className="video-brief-copy">
        <span className="tag">TIN NHANH BẰNG VIDEO</span>
        <h2>{brief.title}</h2>
        <p>Năm câu chuyện quan trọng được biên tập, đọc và dựng tự động thành một video ngắn.</p>
        <small>
          {brief.duration_seconds
            ? `${Math.max(1, Math.round(brief.duration_seconds / 60))} phút`
            : "Khoảng 1 phút"}
        </small>
      </div>
      <video controls preload="metadata" poster={brief.thumbnail_url} src={brief.video_url} />
    </section>
  );
}
