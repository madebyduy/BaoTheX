"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { articleHref, type Item, type SportsEvent } from "../lib";
import { SportsEventCard } from "../sports-event-card";

const API =
  typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";
const DEFAULT = [
  "today",
  "catch_up",
  "schedule",
  "favorites",
  "following",
  "read_later",
  "listen_later",
  "predictions",
];
const LABELS: Record<string, string> = {
  today: "Hôm nay",
  catch_up: "Bắt kịp nhanh",
  schedule: "Lịch của tôi",
  favorites: "Đội/VĐV yêu thích",
  following: "Đang theo dõi",
  read_later: "Đọc sau",
  listen_later: "Nghe sau",
  predictions: "Dự đoán",
};

export function SportsDesk({ events, catchUp }: { events: SportsEvent[]; catchUp: Item[] }) {
  const [layout, setLayout] = useState(DEFAULT);
  const [dragging, setDragging] = useState<string | null>(null);
  const [sync, setSync] = useState("Đã lưu trên thiết bị");

  useEffect(() => {
    const rawLocal = localStorage.getItem("baothex-dashboard");
    const local = parseLayout(rawLocal);
    if (rawLocal && local.length) setLayout(local);
    fetch(`${API}/api/v1/me/dashboard`, { credentials: "include", cache: "no-store" })
      .then((response) => (response.ok ? response.json() : Promise.reject()))
      .then((json) => {
        const server = parseLayout(JSON.stringify((json.data ?? json).layout));
        const merged = rawLocal && local.length ? local : server;
        if (merged.length) {
          setLayout(merged);
          localStorage.setItem("baothex-dashboard", JSON.stringify(merged));
          setSync("Đã đồng bộ tài khoản");
          if (rawLocal) {
            fetch(`${API}/api/v1/me/dashboard`, {
              method: "PATCH",
              credentials: "include",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ layout: merged }),
            }).catch(() => null);
          }
        }
      })
      .catch(() => setSync("Đã lưu trên thiết bị"));
  }, []);

  const hidden = DEFAULT.filter((id) => !layout.includes(id));
  function persist(next: string[]) {
    setLayout(next);
    localStorage.setItem("baothex-dashboard", JSON.stringify(next));
    fetch(`${API}/api/v1/me/dashboard`, {
      method: "PATCH",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ layout: next }),
    })
      .then((response) => {
        if (response.ok) setSync("Đã đồng bộ tài khoản");
      })
      .catch(() => null);
  }
  function drop(target: string) {
    if (!dragging || dragging === target) return;
    const next = layout.filter((id) => id !== dragging);
    next.splice(next.indexOf(target), 0, dragging);
    persist(next);
    setDragging(null);
  }
  function restore() {
    persist(DEFAULT);
  }

  return (
    <>
      <div className="desk-toolbar">
        <span>{sync}</span>
        <button type="button" onClick={restore}>
          Khôi phục bố cục
        </button>
      </div>
      <div className="sports-desk-grid">
        {layout.map((id) => (
          <section
            className="desk-widget"
            key={id}
            draggable
            onDragStart={() => setDragging(id)}
            onDragOver={(event) => event.preventDefault()}
            onDrop={() => drop(id)}
          >
            <header>
              <span className="drag-handle" title="Kéo để sắp xếp">
                ⠿
              </span>
              <div>
                <small>Vì đây là lựa chọn của bạn</small>
                <h2>{LABELS[id] || id}</h2>
              </div>
              <button
                type="button"
                aria-label={`Ẩn ${LABELS[id]}`}
                onClick={() => layout.length > 1 && persist(layout.filter((value) => value !== id))}
              >
                ×
              </button>
            </header>
            <Widget id={id} events={events} catchUp={catchUp} />
          </section>
        ))}
      </div>
      {hidden.length ? (
        <div className="hidden-widgets">
          <span>Đang ẩn: {hidden.map((id) => LABELS[id]).join(", ")}</span>
          <button type="button" onClick={() => persist(DEFAULT)}>
            Hiện lại
          </button>
        </div>
      ) : null}
    </>
  );
}

function Widget({ id, events, catchUp }: { id: string; events: SportsEvent[]; catchUp: Item[] }) {
  if (id === "today" || id === "schedule")
    return events.length ? (
      <div className="desk-event-list">
        {events.slice(0, id === "today" ? 2 : 3).map((event) => (
          <SportsEventCard event={event} key={event.id} />
        ))}
      </div>
    ) : (
      <Empty
        href="/lich-the-thao"
        text="Chưa có lịch hôm nay. Xem Event Hub hoặc chờ nguồn dữ liệu đồng bộ."
      />
    );
  if (id === "catch_up")
    return catchUp.length ? (
      <div className="desk-story-list">
        {catchUp.slice(0, 3).map((item) => (
          <Link href={articleHref(item)} key={item.id}>
            <span>
              {item.verification_status === "confirmed" ? "Đã xác nhận" : "Đang theo dõi"}
            </span>
            <strong>{item.title}</strong>
          </Link>
        ))}
        <Link className="widget-more" href="/bat-kip">
          Mở chế độ Bắt kịp →
        </Link>
      </div>
    ) : (
      <Empty href="/bat-kip" text="Bản bắt kịp sẽ xuất hiện khi có nội dung mới." />
    );
  if (id === "read_later")
    return <Empty href="/luu" text="Mở thư viện bài đã lưu và bộ sưu tập của bạn." />;
  if (id === "listen_later")
    return (
      <Empty href="/luu" text="Danh sách nghe sau dùng chung hàng đợi âm thanh của BaoTheX." />
    );
  if (id === "predictions")
    return <Empty href="/du-doan" text="Trả lời dự đoán và quiz để mở khóa huy hiệu cá nhân." />;
  if (id === "favorites" || id === "following")
    return (
      <Empty href="/cai-dat" text="Chọn đội, vận động viên và chủ đề để cá nhân hóa phần này." />
    );
  return null;
}

function Empty({ href, text }: { href: string; text: string }) {
  return (
    <div className="desk-empty">
      <p>{text}</p>
      <Link href={href}>Thiết lập ngay →</Link>
    </div>
  );
}
function parseLayout(raw: string | null): string[] {
  try {
    const value = JSON.parse(raw || "[]");
    return Array.isArray(value) ? value.filter((id) => typeof id === "string") : [];
  } catch {
    return [];
  }
}
