"use client";

import { useState } from "react";
import { type Item } from "./lib";
import { Card } from "./ui";

const API =
  typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

// LoadMore renders a content grid that grows in place. It seeds from the
// server-rendered first page (good for SEO and first paint) and fetches further
// pages from the same API path on demand, stopping when a short page arrives.
export function LoadMore({
  initial,
  path,
  perPage = 20,
}: {
  initial: Item[];
  path: string;
  perPage?: number;
}) {
  const [items, setItems] = useState<Item[]>(initial);
  const [page, setPage] = useState(1);
  const [done, setDone] = useState(initial.length < perPage);
  const [loading, setLoading] = useState(false);

  async function loadMore() {
    if (loading || done) return;
    setLoading(true);
    const nextPage = page + 1;
    const sep = path.includes("?") ? "&" : "?";
    try {
      const res = await fetch(`${API}/api/v1${path}${sep}per_page=${perPage}&page=${nextPage}`);
      const json = await res.json();
      const data: Item[] = Array.isArray(json.data) ? json.data : Array.isArray(json) ? json : [];
      setItems((prev) => {
        const seen = new Set(prev.map((i) => i.id));
        return [...prev, ...data.filter((i) => !seen.has(i.id))];
      });
      setPage(nextPage);
      if (data.length < perPage) setDone(true);
    } catch {
      setDone(true);
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <div className="grid">
        {items.map((item) => (
          <Card item={item} key={item.id} />
        ))}
      </div>
      {!done ? (
        <div className="load-more-wrap">
          <button className="btn light" type="button" onClick={loadMore} disabled={loading}>
            {loading ? "Đang tải…" : "Xem thêm"}
          </button>
        </div>
      ) : null}
    </>
  );
}
