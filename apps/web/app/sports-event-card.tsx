"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import type { SportsEvent } from "./lib";

const API = (typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081");

// LiveEventRefresh keeps a live match page current without a manual reload. For
// an in-progress (or imminent) event it re-renders the server component every
// 30s via router.refresh(), which re-fetches the low-revalidate event query and
// updates the scoreboard, status and provenance in place. It renders a small
// "live" pulse; for finished/scheduled events it does nothing.
export function LiveEventRefresh({ status }: { status: SportsEvent["status"] }) {
  const router = useRouter();
  const isLive = status === "live";
  useEffect(() => {
    if (!isLive) return;
    const timer = window.setInterval(() => router.refresh(), 30000);
    const onVisible = () => {
      if (document.visibilityState === "visible") router.refresh();
    };
    document.addEventListener("visibilitychange", onVisible);
    return () => {
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", onVisible);
    };
  }, [isLive, router]);
  if (!isLive) return null;
  return (
    <span className="live-refresh" role="status" aria-label="Đang cập nhật trực tiếp">
      <i /> TRỰC TIẾP · tự động cập nhật
    </span>
  );
}

export function EventStatus({ event }: { event: SportsEvent }) {
  const labels: Record<string, string> = {
    live: "Đang diễn ra",
    finished: "Kết thúc",
    scheduled: "Lịch dự kiến",
    postponed: "Tạm hoãn",
    cancelled: "Đã hủy",
  };
  const delayed = event.freshness === "delayed" && event.status !== "scheduled";
  const stale = event.freshness === "stale";
  return (
    <span className={`event-status ${event.status}`}>
      {labels[event.status] || event.status}
      {delayed ? " · cập nhật chậm" : stale ? " · dữ liệu cũ" : ""}
    </span>
  );
}

export function SportsEventCard({ event }: { event: SportsEvent }) {
  const hasScore = event.home_score != null || event.away_score != null;
  return (
    <article className="event-card">
      <div className="event-card-head">
        <span>{event.competition || event.sport_name}</span>
        <EventStatus event={event} />
      </div>
      <Link href={`/tran-dau/${event.id}`} className="event-matchup">
        <div>
          <strong>{event.home_name || event.title}</strong>
          <b>{hasScore ? (event.home_score ?? "–") : ""}</b>
        </div>
        {event.away_name ? (
          <div>
            <strong>{event.away_name}</strong>
            <b>{hasScore ? (event.away_score ?? "–") : ""}</b>
          </div>
        ) : null}
      </Link>
      <div className="event-card-foot">
        <time dateTime={event.starts_at}>
          {new Intl.DateTimeFormat("vi-VN", {
            hour: "2-digit",
            minute: "2-digit",
            day: "2-digit",
            month: "2-digit",
          }).format(new Date(event.starts_at))}
        </time>
        <span>{event.is_manual ? "BaoTheX cập nhật" : `Nguồn: ${event.data_source}`}</span>
        <Link href={`/tran-dau/${event.id}`}>Chi tiết →</Link>
      </div>
    </article>
  );
}

export function EventFollowButton({
  eventId,
  initial = false,
}: {
  eventId: number;
  initial?: boolean;
}) {
  const [following, setFollowing] = useState(initial);
  const [message, setMessage] = useState("");
  async function toggle() {
    const response = await fetch(`${API}/api/v1/events/${eventId}/follow`, {
      method: following ? "DELETE" : "POST",
      credentials: "include",
    });
    if (response.ok) {
      setFollowing(!following);
      setMessage("");
    } else {
      setMessage("Đăng nhập để theo dõi sự kiện");
    }
  }
  return (
    <>
      <button type="button" className="btn ember" onClick={toggle}>
        {following ? "Đang theo dõi" : "Theo dõi sự kiện"}
      </button>
      {message ? <small className="inline-message">{message}</small> : null}
    </>
  );
}
