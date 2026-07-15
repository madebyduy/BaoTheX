import { api, demoItems, type Item } from "../lib";
import { Footer, ItemGrid, PageTitle } from "../ui";
export default async function Page() {
  const items = await api<Item[]>("/content?per_page=20&sort=recent", demoItems);
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
            <a href="/danh-muc">Mới nhất</a>
            <a href="/danh-muc?sort=top">Nổi bật</a>
            <a href="/video">Video</a>
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
