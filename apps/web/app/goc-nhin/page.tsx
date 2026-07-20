import Link from "next/link";
import { api, articleHref, pageMetadata, type Item } from "../lib";
import { Footer, PageTitle } from "../ui";

export const metadata = pageMetadata({
  title: "Góc nhìn — Phân tích đa nguồn",
  description:
    "Bài phân tích đối chiếu nhiều nguồn, được biên tập viên đọc, sửa và chịu trách nhiệm trước khi xuất bản.",
  path: "/goc-nhin",
});

export default async function AnalysisPage() {
  const items = await api<Item[]>("/analyses?limit=30", []);
  return (
    <>
      <main className="wrap analysis-page">
        <PageTitle
          eyebrow="Góc nhìn BaoTheX"
          title="Một sự kiện, mọi nguồn"
          description="Bài phân tích do hệ thống đối chiếu nhiều nguồn và chỉ xuất bản sau khi biên tập viên đọc, sửa và chịu trách nhiệm."
        />
        {items.length ? (
          <div className="analysis-list">
            {items.map((item, index) => (
              <Link href={articleHref(item)} className="analysis-card" key={item.id}>
                <span>{String(index + 1).padStart(2, "0")}</span>
                <div>
                  <small>PHÂN TÍCH ĐA NGUỒN · TÒA SOẠN BAOTHEX</small>
                  <h2>{item.title}</h2>
                  <p>{item.summary || item.excerpt}</p>
                  <b>Đọc góc nhìn →</b>
                </div>
              </Link>
            ))}
          </div>
        ) : (
          <div className="analysis-empty">
            <b>Tòa soạn đang chuẩn bị bài phân tích đầu tiên.</b>
            <p>
              Chỉ sự kiện đã xác nhận và có ít nhất ba nguồn độc lập mới được đưa vào bàn biên tập.
            </p>
          </div>
        )}
      </main>
      <Footer />
    </>
  );
}
