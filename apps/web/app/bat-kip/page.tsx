import { api, type Item } from "../lib";
import { Footer, PageTitle } from "../ui";
import { CatchUpPlayer } from "./player";

type Search = Promise<{ duration?: string }>;
type CatchUpResponse = { duration: number; items: Item[] };

export default async function CatchUpPage({ searchParams }: { searchParams: Search }) {
  const { duration: raw } = await searchParams;
  const duration = raw === "1" || raw === "10" ? Number(raw) : 3;
  const result = await api<CatchUpResponse>(
    `/catch-up?duration=${duration}`,
    { duration, items: [] },
    30,
  );
  return (
    <>
      <main className="wrap catch-up-page">
        <PageTitle
          eyebrow="CATCH-UP MODE"
          title="Bắt kịp thể thao, theo thời gian bạn có"
          description="Mỗi câu chuyện chỉ xuất hiện một lần. Chọn 1, 3 hoặc 10 phút rồi đọc, nghe và tiếp tục ở lần ghé sau."
        />
        <nav className="duration-switch" aria-label="Thời lượng bắt kịp">
          {[1, 3, 10].map((value) => (
            <a
              className={duration === value ? "active" : ""}
              href={`/bat-kip?duration=${value}`}
              key={value}
            >
              <b>{value}</b> phút
            </a>
          ))}
        </nav>
        <CatchUpPlayer items={result.items} duration={duration} />
      </main>
      <Footer />
    </>
  );
}
