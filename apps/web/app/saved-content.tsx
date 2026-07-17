"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import type { Item } from "./lib";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";
type Collection = { id: number; name: string; created_at: string };

export function SavedContent() {
  const [items, setItems] = useState<Item[]>([]);
  const [collections, setCollections] = useState<Collection[]>([]);
  const [activeCollection, setActiveCollection] = useState<number | null>(null);
  const [newName, setNewName] = useState("");
  const [state, setState] = useState<"loading" | "ready" | "login" | "error">("loading");

  useEffect(() => {
    fetch(
      `${API}/api/v1/saved?per_page=20${activeCollection ? `&collection=${activeCollection}` : ""}`,
      { credentials: "include" },
    )
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
  }, [activeCollection]);

  useEffect(() => {
    fetch(`${API}/api/v1/collections`, { credentials: "include" })
      .then((response) => (response.ok ? response.json() : Promise.reject()))
      .then((json) => setCollections(json.data ?? json ?? []))
      .catch(() => null);
  }, []);

  async function createCollection(event: FormEvent) {
    event.preventDefault();
    if (!newName.trim()) return;
    const response = await fetch(`${API}/api/v1/collections`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: newName.trim() }),
    });
    if (!response.ok) return;
    const json = await response.json();
    const collection = json.data ?? json;
    setCollections((values) => [...values, collection]);
    setNewName("");
  }

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
  return (
    <div className="saved-library">
      <aside className="collection-panel">
        <strong>Bộ sưu tập</strong>
        <button
          className={!activeCollection ? "active" : ""}
          type="button"
          onClick={() => setActiveCollection(null)}
        >
          Tất cả bài đã lưu
        </button>
        {collections.map((collection) => (
          <button
            className={activeCollection === collection.id ? "active" : ""}
            type="button"
            onClick={() => setActiveCollection(collection.id)}
            key={collection.id}
          >
            {collection.name}
          </button>
        ))}
        <form onSubmit={createCollection}>
          <input
            value={newName}
            onChange={(event) => setNewName(event.target.value)}
            maxLength={80}
            placeholder="Tên bộ sưu tập mới"
          />
          <button type="submit">+ Tạo</button>
        </form>
      </aside>
      <div className="content-list">
        {items.length ? (
          items.map((item) => (
            <Link className="card" href={`/noi-dung/${item.id}`} key={item.id}>
              <span className="tag">Đã lưu · {item.type}</span>
              <h3>{item.title}</h3>
              <p>{item.summary || item.excerpt || "Nội dung đã lưu để đọc lại."}</p>
              <div className="meta">{item.source_name || "BaoTheX"} · Mở bài →</div>
            </Link>
          ))
        ) : (
          <div className="empty-state">Bộ sưu tập này chưa có bài viết.</div>
        )}
      </div>
    </div>
  );
}
