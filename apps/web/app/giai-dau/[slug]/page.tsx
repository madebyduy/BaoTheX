import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { api, pageMetadata, type Competition, type SportsEvent } from "../../lib";
import { Footer } from "../../ui";

type HubData = {
  competition?: Competition;
  live?: SportsEvent[];
  upcoming?: SportsEvent[];
  results?: SportsEvent[];
};

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const data = await api<HubData>(`/competitions/${slug}`, {}, 300);
  const comp = data.competition;
  if (!comp) return { title: "Không tìm thấy giải đấu" };
  return pageMetadata({
    title: `${comp.name} — Lịch thi đấu & kết quả`,
    description: `Lịch thi đấu, kết quả và tin tức mới nhất của ${comp.name}${comp.country ? ` (${comp.country})` : ""} trên BaoTheX.`,
    path: `/giai-dau/${slug}`,
  });
}

function fmt(value: string) {
  return new Intl.DateTimeFormat("vi-VN", {
    weekday: "short",
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function EventRow({ event }: { event: SportsEvent }) {
  const hasScore = event.home_score != null || event.away_score != null;
  return (
    <Link className="league-row" href={`/tran-dau/${event.id}`}>
      <span className="league-row-time">
        {event.status === "live" ? "TRỰC TIẾP" : fmt(event.starts_at)}
      </span>
      <span className="league-row-teams">
        <b>{event.home_name || event.title}</b>
        {event.away_name ? (
          <>
            <i>{hasScore ? `${event.home_score ?? "–"} : ${event.away_score ?? "–"}` : "vs"}</i>
            <b>{event.away_name}</b>
          </>
        ) : null}
      </span>
    </Link>
  );
}

function Section({ title, events }: { title: string; events?: SportsEvent[] }) {
  if (!events?.length) return null;
  return (
    <section className="league-section">
      <h2>{title}</h2>
      <div className="league-list">
        {events.map((event) => (
          <EventRow event={event} key={event.id} />
        ))}
      </div>
    </section>
  );
}

export default async function Page({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const data = await api<HubData>(`/competitions/${slug}`, {}, 60);
  const comp = data.competition;
  if (!comp) notFound();
  const empty = !data.live?.length && !data.upcoming?.length && !data.results?.length;
  return (
    <>
      <main className="wrap league-hub">
        <div className="league-head">
          <span className="tag">Giải đấu</span>
          <h1>{comp.name}</h1>
          {comp.country ? <p>{comp.country}</p> : null}
        </div>
        <Section title="Đang diễn ra" events={data.live} />
        <Section title="Lịch thi đấu" events={data.upcoming} />
        <Section title="Kết quả gần đây" events={data.results} />
        {empty ? (
          <div className="empty-state">
            <p>Chưa có lịch hoặc kết quả cho giải đấu này. Vui lòng quay lại sau.</p>
            <Link className="btn light" href="/lich-the-thao">
              Xem tất cả lịch thi đấu
            </Link>
          </div>
        ) : null}
      </main>
      <Footer />
    </>
  );
}
