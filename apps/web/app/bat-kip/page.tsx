import Link from "next/link";
import { cookies } from "next/headers";
import { api, apiWithCookie, pageMetadata, type Entity, type Item, type Topic } from "../lib";
import { Footer } from "../ui";
import { CatchUpPlayer } from "./player";

export const metadata = pageMetadata({
  title: "Bắt kịp — Tin thể thao trong 3 phút",
  description:
    "Bản tóm tắt nhanh những tin thể thao cần biết trong ngày: 1, 3 hoặc 10 phút, đọc hoặc nghe.",
  path: "/bat-kip",
});

type Search = Promise<{ duration?: string }>;
type CatchUpResponse = { duration: number; items: Item[] };
type Follows = { topics?: Topic[]; entities?: Entity[]; Topics?: Topic[]; Entities?: Entity[] };

const durations = [
  { value: 1, label: "1 phút", note: "3 tin cần biết" },
  { value: 3, label: "3 phút", note: "5 tin quan trọng" },
  { value: 10, label: "10 phút", note: "Đọc sâu hơn" },
];

export default async function CatchUpPage({ searchParams }: { searchParams: Search }) {
  const { duration: raw } = await searchParams;
  const duration = raw === "1" || raw === "10" ? Number(raw) : 3;
  const cookie = (await cookies()).toString();
  const fallback = { duration, items: [] };
  const result = cookie
    ? await apiWithCookie<CatchUpResponse>(`/catch-up?duration=${duration}`, fallback, cookie)
    : await api<CatchUpResponse>(`/catch-up?duration=${duration}`, fallback, 30);
  const follows = cookie
    ? await apiWithCookie<Follows>("/follows", { topics: [], entities: [] }, cookie)
    : { topics: [], entities: [] };
  const topics = Array.isArray(follows.topics) ? follows.topics : (follows.Topics ?? []);
  const entities = Array.isArray(follows.entities) ? follows.entities : (follows.Entities ?? []);
  const interests = [...topics, ...entities].map((entry) => entry.name).slice(0, 3);
  return (
    <>
      <main className="wrap catch-up-page">
        <header className="catch-up-masthead">
          <div>
            <span className="eyebrow">BAOTHEX BRIEFING</span>
            <h1>Bắt kịp</h1>
            <p>Những diễn biến đáng đọc, đã được chọn lọc, Việt hóa và đối chiếu nguồn.</p>
          </div>
          <div className="catch-up-editions">
            <Link href="/#nghe-bao">Bản tin 6h/20h</Link>
            <Link href="/cai-dat">Theo đội và môn</Link>
          </div>
        </header>
        <nav className="duration-switch" aria-label="Thời lượng bắt kịp">
          {durations.map(({ value, label, note }) => (
            <a
              className={duration === value ? "active" : ""}
              href={`/bat-kip?duration=${value}`}
              key={value}
            >
              <b>{label}</b>
              <small>{note}</small>
            </a>
          ))}
        </nav>
        <CatchUpPlayer items={result.items} duration={duration} interests={interests} />
      </main>
      <Footer />
    </>
  );
}
