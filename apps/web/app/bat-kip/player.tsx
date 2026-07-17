"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import type { Item } from "../lib";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

export function CatchUpPlayer({ items, duration }: { items: Item[]; duration: number }) {
  const storageKey = `baothex-catchup-${duration}`;
  const [index, setIndex] = useState(0);
  const [known, setKnown] = useState<number[]>([]);
  const [speaking, setSpeaking] = useState(false);
  const [actionMessage, setActionMessage] = useState("");
  const current = items[index];
  const progress = items.length ? Math.round(((index + 1) / items.length) * 100) : 0;
  const text = useMemo(
    () => (current ? `${current.title}. ${current.summary || current.excerpt || ""}` : ""),
    [current],
  );

  useEffect(() => {
    const saved = Number(localStorage.getItem(storageKey) || 0);
    const hidden = readHidden();
    const firstVisible = items.findIndex(
      (item, itemIndex) => itemIndex >= saved && !hidden.includes(item.id),
    );
    if (firstVisible >= 0) setIndex(firstVisible);
    track("catch_up_started", { duration });
  }, [duration, items, storageKey]);

  function move(next: number) {
    const safe = Math.max(0, Math.min(items.length - 1, next));
    setIndex(safe);
    localStorage.setItem(storageKey, String(safe));
    if (safe === items.length - 1) track("catch_up_completed", { duration, count: items.length });
  }
  function markKnown() {
    if (!current) return;
    setKnown((values) => [...values, current.id]);
    move(index + 1);
  }
  async function hide() {
    if (!current) return;
    const hidden = readHidden();
    localStorage.setItem(
      "btx-hidden-content",
      JSON.stringify([...new Set([...hidden, current.id])]),
    );
    await fetch(`${API}/api/v1/hidden/${current.id}`, {
      method: "POST",
      credentials: "include",
    }).catch(() => null);
    markKnown();
  }
  async function followStory() {
    if (!current?.story_cluster_id) {
      setActionMessage("Câu chuyện đơn lẻ chưa có cụm để theo dõi.");
      return;
    }
    const response = await fetch(`${API}/api/v1/clusters/${current.story_cluster_id}/follow`, {
      method: "POST",
      credentials: "include",
    });
    setActionMessage(response.ok ? "Đang theo dõi câu chuyện này" : "Đăng nhập để theo dõi tiếp");
  }
  function speak() {
    if (!("speechSynthesis" in window) || !text) return;
    window.speechSynthesis.cancel();
    if (speaking) {
      setSpeaking(false);
      return;
    }
    const utterance = new SpeechSynthesisUtterance(text);
    utterance.lang = "vi-VN";
    utterance.rate = 1;
    utterance.onend = () => setSpeaking(false);
    setSpeaking(true);
    window.speechSynthesis.speak(utterance);
  }

  if (!items.length)
    return (
      <div className="empty-state">
        <h2>Chưa có bản bắt kịp hôm nay</h2>
        <p>
          Khi nội dung đã duyệt xuất hiện, hệ thống sẽ tự gom theo câu chuyện mà không cần gọi AI
          theo từng lượt xem.
        </p>
      </div>
    );
  if (!current) return null;
  return (
    <section className="catch-up-player">
      <div className="catch-up-progress">
        <span style={{ width: `${progress}%` }} />
        <b>
          {index + 1}/{items.length}
        </b>
      </div>
      <div className="catch-up-card">
        <div className="catch-up-kicker">
          <span>
            {current.verification_status === "confirmed" ? "ĐÃ XÁC NHẬN" : "ĐANG THEO DÕI"}
          </span>
          <small>{current.source_name || "BaoTheX"}</small>
        </div>
        <h2>{current.title}</h2>
        <p>{current.summary || current.excerpt || "Nội dung đang được biên tập."}</p>
        <div className="catch-up-actions">
          <button className="btn light" type="button" onClick={speak}>
            {speaking ? "Dừng nghe" : "Nghe thẻ này"}
          </button>
          <button className="btn light" type="button" onClick={markKnown}>
            Đã biết
          </button>
          <button className="btn light" type="button" onClick={followStory}>
            Theo dõi tiếp
          </button>
          <Link className="btn ember" href={`/noi-dung/${current.id}`}>
            Đọc sâu
          </Link>
          <button className="text-action" type="button" onClick={hide}>
            Không quan tâm
          </button>
        </div>
        {actionMessage ? <small className="inline-message">{actionMessage}</small> : null}
      </div>
      <div className="catch-up-nav">
        <button type="button" onClick={() => move(index - 1)} disabled={index === 0}>
          ← Trước
        </button>
        <span>
          {known.length ? `Đã xử lý ${known.length} câu chuyện` : "Tiến độ được lưu trên thiết bị"}
        </span>
        <button type="button" onClick={() => move(index + 1)} disabled={index === items.length - 1}>
          Tiếp →
        </button>
      </div>
    </section>
  );
}

function track(event_name: string, properties: Record<string, unknown>) {
  let clientId = localStorage.getItem("btx-cid");
  if (!clientId) {
    clientId = crypto.randomUUID?.() || `${Date.now()}`;
    localStorage.setItem("btx-cid", clientId);
  }
  fetch(`${API}/api/v1/product-events`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ client_id: clientId, event_name, properties }),
  }).catch(() => null);
}

function readHidden(): number[] {
  try {
    const value = JSON.parse(localStorage.getItem("btx-hidden-content") || "[]");
    return Array.isArray(value) ? value.filter((id) => Number.isInteger(id)) : [];
  } catch {
    return [];
  }
}
