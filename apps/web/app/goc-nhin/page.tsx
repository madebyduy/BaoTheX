import Link from "next/link";
import { api, articleHref, pageMetadata, type Item } from "../lib";
import { Footer, PageTitle, RemoteImage } from "../ui";
import { AnalysisFeed } from "./analysis-feed";

export const metadata = pageMetadata({
  title: "Góc nhìn — Phân tích đa nguồn",
  description:
    "Bài phân tích đối chiếu nhiều nguồn, được biên tập viên đọc, sửa và chịu trách nhiệm trước khi xuất bản.",
  path: "/goc-nhin",
});

function formatDay(value?: string): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat("vi-VN", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  }).format(date);
}

export default async function AnalysisPage() {
  // Pull a generous window and let the client reveal the tail on demand: the
  // section reads as a curated op-ed page (one lead + a scannable grid) instead
  // of an endless stack of identical hero blocks.
  const items = await api<Item[]>("/analyses?limit=60", []);
  const [lead, ...rest] = items;
  const leadDate = lead ? formatDay(lead.published_at) : "";
  const leadSources = lead?.cluster_source_count ?? 0;

  return (
    <>
      <main className="wrap analysis-page">
        <PageTitle
          eyebrow="Góc nhìn BaoTheX"
          title="Một sự kiện, mọi nguồn"
          description="Bài phân tích do hệ thống đối chiếu nhiều nguồn và chỉ xuất bản sau khi biên tập viên đọc, sửa và chịu trách nhiệm."
        />
        {lead ? (
          <>
            <Link
              href={articleHref(lead)}
              className={`analysis-lead${lead.image_url ? "" : " analysis-lead--text"}`}
            >
              <div className="analysis-lead-body">
                <span className="analysis-lead-kicker">Góc nhìn nổi bật</span>
                <h2>{lead.title}</h2>
                <p>{lead.summary || lead.excerpt}</p>
                <div className="analysis-lead-meta">
                  {leadDate ? <span>{leadDate}</span> : null}
                  {leadSources > 1 ? <span>Đối chiếu {leadSources} nguồn</span> : null}
                  <b>Đọc góc nhìn →</b>
                </div>
              </div>
              {/* Only reserve the media column when there is an image; a text-only
                  analysis fills the full width instead of leaving an empty panel. */}
              {lead.image_url ? (
                <div className="analysis-lead-media">
                  <RemoteImage
                    src={lead.image_url}
                    alt=""
                    loading="eager"
                    decoding="async"
                    referrerPolicy="no-referrer"
                  />
                </div>
              ) : null}
            </Link>
            {rest.length ? <AnalysisFeed items={rest} /> : null}
          </>
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
