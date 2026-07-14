import { api, demoItems, type Item } from "../../lib";
import { Footer, ItemGrid, PageTitle } from "../../ui";
export default async function Page() {
  const items = await api<Item[]>("/admin/content?per_page=20", demoItems);
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Quản trị nội dung"
          title="Hàng chờ biên tập"
          description="Danh sách nội dung để quản trị viên kiểm tra trước khi hiển thị."
        />
        <section className="section">
          <ItemGrid items={items} />
        </section>
      </main>
      <Footer />
    </>
  );
}
