import { api, demoItems, type Item } from "../lib";
import { Footer, ItemGrid, PageTitle } from "../ui";
import { VideoBriefPlayer } from "./video-brief";
export default async function Page() {
  const fallback = demoItems.filter((x) => x.type === "video");
  const items = await api<Item[]>("/videos?per_page=20", fallback);
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Video"
          title="Xem có chọn lọc"
          description="Video từ các kênh chuyên môn, có tóm tắt và mốc nội dung để bạn xem nhanh hơn."
        />
        <VideoBriefPlayer />
        <section className="section">
          <ItemGrid items={items} />
        </section>
      </main>
      <Footer />
    </>
  );
}
