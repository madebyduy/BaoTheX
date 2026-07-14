import { api, demoItems, type Item } from "../lib";
import { Footer, ItemGrid, PageTitle } from "../ui";
export default async function Page() {
  const fallback = demoItems.filter((x) => x.type === "research");
  const items = await api<Item[]>("/research?per_page=20", fallback);
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Nghiên cứu"
          title="Khoa học, nói dễ hiểu"
          description="Các nghiên cứu được tóm tắt theo câu hỏi, đối tượng, phát hiện, giới hạn và ứng dụng thực tế."
        />
        <div className="layout">
          <aside className="side">
            <span className="eyebrow">Lọc nghiên cứu</span>
            <a href="/nghien-cuu">Tất cả</a>
            <a href="/nghien-cuu?tab=notable">Đáng chú ý</a>
            <a href="/nghien-cuu?tab=open_access">Truy cập mở</a>
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
