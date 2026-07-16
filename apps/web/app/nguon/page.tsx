import Link from "next/link";
import { api, type Source } from "../lib";
import { Footer, PageTitle } from "../ui";

const fallbackSources: Source[] = [
  { id: 1, name: "Journal of Strength & Conditioning", kind: "Nghiên cứu", quality: 5 },
  { id: 2, name: "Stronger by Science", kind: "Trang tin", quality: 5 },
  { id: 3, name: "Europe PMC", kind: "Cơ sở dữ liệu", quality: 5 },
  { id: 4, name: "Jeff Nippard", kind: "YouTube", quality: 4 },
  { id: 5, name: "Iron Culture", kind: "Podcast", quality: 4 },
  { id: 6, name: "Barbell Medicine", kind: "Trang tin", quality: 5 },
];

export default async function Page() {
  const sources = await api<Source[]>("/sources", fallbackSources);
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Minh bạch nguồn"
          title="Nguồn dữ liệu"
          description="BaoTheX ưu tiên nguồn gốc rõ ràng, có phương pháp và được cập nhật thường xuyên."
        />
        <div className="grid section">
          {sources.map((source) => (
            <Link className="card" href={`/danh-muc?source=${source.id}`} key={source.id}>
              <span className="tag">{source.kind || "Nguồn nội dung"}</span>
              <h3>{source.name}</h3>
              <p>
                {source.homepage_url
                  ? source.homepage_url.replace(/^https?:\/\//, "").replace(/\/.*$/, "")
                  : "Nguồn nội dung"}
                {" · "}
                {source.default_lang === "vi" ? "Tiếng Việt" : "Quốc tế"}
              </p>
              <div className="meta">
                Độ tin cậy {"★".repeat(Math.max(1, Math.min(5, source.quality || 3)))} · Xem bài →
              </div>
            </Link>
          ))}
        </div>
      </main>
      <Footer />
    </>
  );
}
