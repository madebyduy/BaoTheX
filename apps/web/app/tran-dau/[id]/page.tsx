import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { api, pageMetadata, safeJsonLd, type Item, type SportsEvent } from "../../lib";
import { Footer, ItemGrid } from "../../ui";
import { EventFollowButton, EventStatus, LiveEventRefresh } from "../../sports-event-card";

const SITE = process.env.NEXT_PUBLIC_SITE_URL || "https://baothex.vn";

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  const event = await api<SportsEvent | null>(`/events/${id}`, null, 10);
  if (!event) return { title: "Không tìm thấy trận đấu" };
  const vs = event.away_name ? `${event.home_name} vs ${event.away_name}` : event.title;
  const comp = event.competition || event.sport_name;
  return pageMetadata({
    title: `${vs} — ${comp}`,
    description: `Lịch thi đấu, tỷ số và tin liên quan: ${vs}${comp ? ` (${comp})` : ""} trên BaoTheX.`,
    path: `/tran-dau/${id}`,
  });
}

function eventJsonLd(event: SportsEvent, id: string) {
  const statusMap: Record<string, string> = {
    scheduled: "https://schema.org/EventScheduled",
    live: "https://schema.org/EventScheduled",
    postponed: "https://schema.org/EventPostponed",
    cancelled: "https://schema.org/EventCancelled",
    finished: "https://schema.org/EventScheduled",
  };
  const competitors = [event.home_name, event.away_name].filter(Boolean).map((name) => ({
    "@type": "SportsTeam",
    name,
  }));
  return {
    "@context": "https://schema.org",
    "@type": "SportsEvent",
    name: event.title,
    startDate: event.starts_at,
    eventStatus: statusMap[event.status] || "https://schema.org/EventScheduled",
    sport: event.sport_name,
    ...(event.competition
      ? { superEvent: { "@type": "SportsEvent", name: event.competition } }
      : {}),
    ...(competitors.length ? { competitor: competitors } : {}),
    url: `${SITE}/tran-dau/${id}`,
    location: {
      "@type": "Place",
      name: event.competition || event.sport_name || "Sự kiện thể thao",
    },
  };
}

export default async function EventDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const event = await api<SportsEvent | null>(`/events/${id}`, null, 10);
  if (!event) notFound();
  const content = event.related_content?.length
    ? event.related_content
    : await api<Item[]>(`/events/${id}/content`, [], 30);
  const hasScore = event.home_score != null || event.away_score != null;
  const API =
    typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: safeJsonLd(eventJsonLd(event, id)) }}
      />
      <main className="wrap event-detail">
        <div className="event-detail-top">
          <span className="tag">{event.competition || event.sport_name}</span>
          <EventStatus event={event} />
          <LiveEventRefresh status={event.status} />
        </div>
        <h1>{event.title}</h1>
        <div className="event-scoreboard">
          <div>
            <span>Chủ nhà</span>
            <strong>{event.home_name || event.title}</strong>
            <b>{hasScore ? (event.home_score ?? "–") : ""}</b>
          </div>
          <i>
            {hasScore
              ? ""
              : new Intl.DateTimeFormat("vi-VN", {
                  hour: "2-digit",
                  minute: "2-digit",
                  day: "2-digit",
                  month: "2-digit",
                  year: "numeric",
                }).format(new Date(event.starts_at))}
          </i>
          {event.away_name ? (
            <div>
              <span>Đội khách</span>
              <strong>{event.away_name}</strong>
              <b>{hasScore ? (event.away_score ?? "–") : ""}</b>
            </div>
          ) : null}
        </div>
        <div className="event-actions">
          <EventFollowButton eventId={event.id} initial={event.following} />
          <a className="btn light" href={`${API}/api/v1/events/${event.id}/calendar.ics`}>
            Thêm vào lịch (.ics)
          </a>
        </div>
        <div className="event-provenance">
          <div>
            <small>Nguồn dữ liệu</small>
            <strong>{event.is_manual ? "BaoTheX cập nhật" : event.data_source}</strong>
          </div>
          <div>
            <small>Độ mới</small>
            <strong>{freshnessLabel(event.freshness)}</strong>
          </div>
          <div>
            <small>Cập nhật lúc</small>
            <strong>
              {new Intl.DateTimeFormat("vi-VN", {
                hour: "2-digit",
                minute: "2-digit",
                day: "2-digit",
                month: "2-digit",
              }).format(new Date(event.data_updated_at))}
            </strong>
          </div>
        </div>
        <section className="section">
          <div className="section-heading">
            <div>
              <span className="tag">TIN LIÊN QUAN</span>
              <h2>Diễn biến & phân tích</h2>
            </div>
            <Link href="/lich-the-thao">Tất cả sự kiện →</Link>
          </div>
          {content.length ? (
            <ItemGrid items={content} />
          ) : (
            <div className="empty-state">
              <p>Chưa có bài viết được liên kết với sự kiện này.</p>
            </div>
          )}
        </section>
      </main>
      <Footer />
    </>
  );
}

function freshnessLabel(value: string) {
  return (
    (
      {
        live: "Theo thời gian thực",
        delayed: "Cập nhật chậm",
        scheduled: "Lịch dự kiến",
        manual: "Biên tập thủ công",
        stale: "Đang dùng dữ liệu lưu gần nhất",
      } as Record<string, string>
    )[value] || value
  );
}
