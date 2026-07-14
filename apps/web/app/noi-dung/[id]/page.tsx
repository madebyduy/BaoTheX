import Link from "next/link";
import { api, demoItems, type Item, type ContentBody, typeLabel } from "../../lib";
import { Footer } from "../../ui";
import { SaveButton, TranslateButton } from "../../action-buttons";

type ResearchDetail = {
  abstract?: string;
  journal?: string;
  authors?: string[];
  study_type?: string;
  full_text_url?: string;
  breakdown?: {
    question?: string;
    participants?: string;
    intervention?: string;
    findings?: string[];
    not_proven?: string;
    limitations?: string[];
    practical?: string;
  };
};

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const data = await api<{
    item?: Item;
    body?: ContentBody;
    research?: ResearchDetail;
    article?: { author?: string; word_count?: number };
  }>(`/content/${id}`, {});
  const item = data.item || demoItems.find((x) => x.id === Number(id)) || demoItems[0];
  const body = data.body;
  const research = data.research;
  const translated = body?.vietnamese_body?.trim();
  const sourceText = translated || body?.original_body || research?.abstract || item.excerpt || "";
  const related = await api<Item[]>(`/content/${id}/related`, []);
  const isEnglish = Boolean(body?.original_body && !translated && body.original_language !== "vi");

  return (
    <>
      <main className="wrap article-page">
        <div className="article-kicker">
          <span className="tag">{typeLabel(item.type)}</span>
          <span>{item.source_name || "BaoTheX"}</span>
          <span>{formatDate(item.published_at)}</span>
        </div>
        <h1 className="article-title">{item.title}</h1>
        <p className="article-lede">
          {item.summary || item.excerpt || "Bài viết đang được biên tập và tóm tắt."}
        </p>
        {item.image_url ? (
          <div className="article-hero">
            <img src={item.image_url} alt="" />
          </div>
        ) : null}

        <div className="article-layout">
          <aside className="article-aside">
            <span className="eyebrow">THÔNG TIN BÀI</span>
            <div className="article-aside-line">
              <b>Nguồn</b>
              <span>{item.source_name || "BaoTheX"}</span>
            </div>
            {research?.journal ? (
              <div className="article-aside-line">
                <b>Tạp chí</b>
                <span>{research.journal}</span>
              </div>
            ) : null}
            {research?.study_type ? (
              <div className="article-aside-line">
                <b>Loại nghiên cứu</b>
                <span>{research.study_type}</span>
              </div>
            ) : null}
            {item.canonical_url ? (
              <a
                className="article-source"
                href={item.canonical_url}
                target="_blank"
                rel="noreferrer"
              >
                Mở bài gốc ↗
              </a>
            ) : null}
            <div id="luu">
              <SaveButton contentId={item.id} />
            </div>
          </aside>

          <div className="article-body">
            {isEnglish ? (
              <div className="translation-banner">
                <b>Bài gốc đang ở tiếng Anh</b>
                <span>
                  Bấm dịch để đọc bản tiếng Việt. BaoTheX chỉ hiển thị bản dịch sau khi xử lý xong.
                </span>
                <TranslateButton contentId={item.id} />
              </div>
            ) : null}
            {translated ? (
              <div className="translation-badge">BẢN TIẾNG VIỆT ĐÃ ĐƯỢC BIÊN TẬP</div>
            ) : null}
            <section className="article-content">
              <div className="article-section article-main-copy">
                <h2>{translated ? "Nội dung bài viết" : "Nội dung nguồn"}</h2>
                <ReadableText text={sourceText} />
              </div>
              {item.key_points?.length ? (
                <ArticleList title="Điểm chính" items={item.key_points} />
              ) : null}
              {research?.breakdown ? (
                <>
                  <ArticleSection title="Câu hỏi nghiên cứu" text={research.breakdown.question} />
                  <ArticleSection
                    title="Đối tượng và phương pháp"
                    text={[research.breakdown.participants, research.breakdown.intervention]
                      .filter(Boolean)
                      .join("\n\n")}
                  />
                  <ArticleList title="Phát hiện chính" items={research.breakdown.findings} />
                  <ArticleSection
                    title="Giới hạn cần biết"
                    text={
                      research.breakdown.not_proven ||
                      (research.breakdown.limitations || []).join("\n\n")
                    }
                  />
                </>
              ) : null}
            </section>
            {related.length ? (
              <section className="related-section">
                <h2>Bài liên quan</h2>
                {related.map((x) => (
                  <Link className="related-item" href={`/noi-dung/${x.id}`} key={x.id}>
                    <strong>{x.title}</strong>
                    <small>{x.source_name || "BaoTheX"}</small>
                  </Link>
                ))}
              </section>
            ) : null}
          </div>
        </div>
      </main>
      <Footer />
    </>
  );
}

function ReadableText({ text }: { text: string }) {
  const paragraphs = text
    .split(/\n\s*\n/)
    .map((p) => p.trim())
    .filter(Boolean);
  return (
    <div className="article-prose">
      {paragraphs.map((p, i) => (
        <p key={`${i}-${p.slice(0, 12)}`}>{p}</p>
      ))}
    </div>
  );
}
function ArticleSection({ title, text }: { title: string; text?: string }) {
  return text ? (
    <section className="article-section">
      <h2>{title}</h2>
      <ReadableText text={text} />
    </section>
  ) : null;
}
function ArticleList({ title, items }: { title: string; items?: string[] }) {
  return items?.length ? (
    <section className="article-section">
      <h2>{title}</h2>
      <ul>
        {items.map((x) => (
          <li key={x}>{x}</li>
        ))}
      </ul>
    </section>
  ) : null;
}
function formatDate(value?: string) {
  if (!value) return "";
  try {
    return new Intl.DateTimeFormat("vi-VN", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
    }).format(new Date(value));
  } catch {
    return "";
  }
}
