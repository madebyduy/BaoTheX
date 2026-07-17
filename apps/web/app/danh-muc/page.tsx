import { api, demoItems, type Item, type Source } from "../lib";
import { Footer, ItemGrid, PageTitle } from "../ui";
type Search = Promise<{ sort?: string; source?: string }>;
export default async function Page({ searchParams }: { searchParams: Search }) {
  const params = await searchParams;
  const sort = params.sort === "top" ? "top" : "recent";
  const query = new URLSearchParams({ per_page: "20", sort });
  if (/^\d+$/.test(params.source || "")) query.set("source", params.source!);
  const [items, sources] = await Promise.all([
    api<Item[]>(`/content?${query}`, demoItems),
    api<Source[]>("/sources", []),
  ]);
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Dòng tin"
          title="Tất cả nội dung"
          description="Tin thể thao và video mới nhất được cập nhật từ các nguồn đã kiểm duyệt."
        />
        <div className="layout">
          <aside className="side">
            <span className="eyebrow">Bộ lọc</span>
            <a
              className={sort === "recent" ? "active" : ""}
              href={`/danh-muc?sort=recent${params.source ? `&source=${params.source}` : ""}`}
            >
              Mới nhất
            </a>
            <a
              className={sort === "top" ? "active" : ""}
              href={`/danh-muc?sort=top${params.source ? `&source=${params.source}` : ""}`}
            >
              Nổi bật
            </a>
            <a href="/video">Video</a>
            <form action="/danh-muc" className="source-filter">
              <input type="hidden" name="sort" value={sort} />
              <label htmlFor="source">Nguồn</label>
              <select id="source" name="source" defaultValue={params.source || ""}>
                <option value="">Tất cả nguồn</option>
                {sources.map((source) => (
                  <option value={source.id} key={source.id}>
                    {source.name}
                  </option>
                ))}
              </select>
              <button type="submit">Áp dụng</button>
            </form>
          </aside>
          <section>
            <ItemGrid items={items} />
          </section>
        </div>
      </main>
      <Footer />
    </>
  );
}
