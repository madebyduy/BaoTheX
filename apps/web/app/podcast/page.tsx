import { api, demoItems, type Item } from "../lib";
import { Footer, ItemGrid, PageTitle } from "../ui";
export default async function Page() {
  const fallback = demoItems.filter((x) => x.type === "podcast");
  const items = await api<Item[]>("/content?type=podcast&per_page=20&sort=recent", fallback);
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Podcast"
          title="Nghe để hiểu sâu hơn"
          description="Các cuộc trò chuyện dài về tập luyện, dinh dưỡng và phục hồi được tóm tắt bằng tiếng Việt."
        />
        <section className="section">
          <ItemGrid items={items} />
        </section>
      </main>
      <Footer />
    </>
  );
}
