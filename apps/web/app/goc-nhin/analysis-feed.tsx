"use client";

import { useState } from "react";
import Link from "next/link";
import { articleHref, type Item } from "../lib";

// How many analyses to reveal per "Xem thêm" click. The page fetches a large
// window server-side, so revealing is instant and needs no extra request — good
// enough until the archive grows large enough to warrant true pagination.
const CHUNK = 9;

function formatDay(value?: string): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat("vi-VN", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  }).format(date);
}

export function AnalysisFeed({ items }: { items: Item[] }) {
  const [visible, setVisible] = useState(Math.min(CHUNK, items.length));
  const shown = items.slice(0, visible);
  const done = visible >= items.length;

  return (
    <>
      <div className="analysis-tiles">
        {shown.map((item) => {
          const day = formatDay(item.published_at);
          const sources = item.cluster_source_count ?? 0;
          return (
            <Link href={articleHref(item)} className="analysis-tile" key={item.id}>
              <span className="analysis-tile-kicker">Góc nhìn</span>
              <h3>{item.title}</h3>
              <p>{item.summary || item.excerpt || "Bài phân tích đang được biên tập."}</p>
              <div className="analysis-tile-meta">
                {day ? <span>{day}</span> : null}
                {sources > 1 ? <span>{sources} nguồn</span> : null}
                <i>Đọc →</i>
              </div>
            </Link>
          );
        })}
      </div>
      {!done ? (
        <div className="load-more-wrap">
          <button className="btn light" type="button" onClick={() => setVisible((v) => v + CHUNK)}>
            Xem thêm góc nhìn
          </button>
        </div>
      ) : null}
    </>
  );
}
