import { api, demoItems, pageMetadata, type Item } from "../lib";
import { Footer, PageTitle } from "../ui";
import { LoadMore } from "../load-more";
export const metadata = pageMetadata({
  title: "Video thể thao",
  description:
    "Tuyển tập video thể thao chọn lọc: highlight, phân tích chiến thuật và câu chuyện sau trận.",
  path: "/video",
});

export default async function Page() {
  const fallback = demoItems.filter((x) => x.type === "video");
  const items = await api<Item[]>("/videos?per_page=20", fallback);
  const listPath = "/videos?sort=recent";
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Video"
          title="Xem có chọn lọc"
          description="Video từ các kênh chuyên môn, có tóm tắt và mốc nội dung để bạn xem nhanh hơn."
        />
        <section className="section">
          <div className="video-page-intro">
            <span className="tag">KÊNH CHÍNH THỨC</span>
            <p>
              Video mới từ các kênh thể thao đã chọn lọc. Mỗi thẻ dẫn thẳng đến video YouTube gốc,
              không tạo thêm một bản “tin nhanh” trùng với Thể thao 6h.
            </p>
          </div>
          <LoadMore initial={items} path={listPath} perPage={20} />
        </section>
      </main>
      <Footer />
    </>
  );
}
