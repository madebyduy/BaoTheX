import { api, demoItems, type Item } from "../lib";
import { Footer, ItemGrid, PageTitle } from "../ui";
export default async function Page({ searchParams }: { searchParams: Promise<{ q?: string }> }) {
  const { q = "" } = await searchParams;
  const data = await api<{ research?: Item[]; articles?: Item[]; videos?: Item[] }>(
    `/search?q=${encodeURIComponent(q)}`,
    {},
  );
  const items = [...(data.research || []), ...(data.articles || []), ...(data.videos || [])];
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Tìm kiếm"
          title={q ? `Kết quả cho “${q}”` : "Tìm nội dung bạn cần"}
          description="Tìm theo chủ đề, người, nghiên cứu hoặc nguồn."
        />
        {q && (
          <section className="section">
            <ItemGrid items={items.length ? items : demoItems} />
          </section>
        )}
      </main>
      <Footer />
    </>
  );
}
