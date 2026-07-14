"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import type { Item } from "./lib";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

export function SavedContent() {
  const [items, setItems] = useState<Item[]>([]);
  const [state, setState] = useState<"loading" | "ready" | "login" | "error">("loading");

  useEffect(() => {
    fetch(`${API}/api/v1/saved?per_page=20`, { credentials: "include" })
      .then(async (response) => {
        if (response.status === 401) {
          setState("login");
          return;
        }
        if (!response.ok) throw new Error("saved request failed");
        const json = await response.json();
        setItems(json.data ?? json.items ?? json ?? []);
        setState("ready");
      })
      .catch(() => setState("error"));
  }, []);

  if (state === "loading") return <div className="empty-state">Đang tải thư viện của bạn…</div>;
  if (state === "login") {
    return (
      <div className="empty-state">
        <strong>Đăng nhập để xem nội dung đã lưu</strong>
        <p>Những bài bạn đánh dấu sẽ được đồng bộ giữa các thiết bị.</p>
        <Link className="btn ember" href="/dang-nhap">
          Đăng nhập
        </Link>
      </div>
    );
  }
  if (state === "error")
    return <div className="empty-state">Không thể tải thư viện lúc này. Hãy thử lại sau.</div>;
  if (!items.length) return <div className="empty-state">Bạn chưa lưu bài viết nào.</div>;

  return (
    <div className="content-list">
      {items.map((item) => (
        <Link className="card" href={`/noi-dung/${item.id}`} key={item.id}>
          <span className="tag">Đã lưu · {item.type}</span>
          <h3>{item.title}</h3>
          <p>{item.summary || item.excerpt || "Nội dung đã lưu để đọc lại."}</p>
          <div className="meta">{item.source_name || "BaoTheX"} · Mở bài →</div>
        </Link>
      ))}
    </div>
  );
}
