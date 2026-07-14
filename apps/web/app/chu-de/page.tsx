import Link from "next/link";
import { api, demoTopics, type Topic } from "../lib";
import { Footer, PageTitle } from "../ui";
export default async function Page() {
  const topics = await api<Topic[]>("/topics", demoTopics);
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Khám phá"
          title="Chủ đề bạn quan tâm"
          description="Theo dõi chủ đề để nhận dòng tin phù hợp với mục tiêu của bạn."
        />
        <div className="topic-grid section">
          {topics.map((t) => (
            <Link href={`/chu-de/${t.slug}`} className="topic" key={t.id}>
              <strong>{t.name}</strong>
              <p style={{ color: "var(--muted)", fontSize: 13, marginTop: 8 }}>
                {t.description || "Nội dung chọn lọc theo chủ đề."}
              </p>
              <div className="meta">{t.follower_count || 0} người theo dõi →</div>
            </Link>
          ))}
        </div>
      </main>
      <Footer />
    </>
  );
}
