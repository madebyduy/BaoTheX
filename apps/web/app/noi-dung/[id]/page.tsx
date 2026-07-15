import Link from "next/link";
import { notFound } from "next/navigation";
import { api, type ContentBody, type Item, typeLabel } from "../../lib";
import { Footer } from "../../ui";
import { SaveButton, TranslateButton } from "../../action-buttons";

type Detail = {
  item?: Item;
  body?: ContentBody;
  article?: { author?: string; word_count?: number };
  video?: {
    youtube_id: string;
    channel_title?: string;
    thumbnail_url?: string;
    description?: string;
    duration_sec?: number;
    yt_views?: number;
  };
  research?: {
    abstract?: string;
    journal?: string;
    breakdown?: {
      question?: string;
      participants?: string;
      intervention?: string;
      findings?: string[];
      not_proven?: string;
      limitations?: string[];
    };
  };
};

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const [data, related, newsroom] = await Promise.all([
    api<Detail>(`/content/${id}`, {}, 30),
    api<Item[]>(`/content/${id}/related`, [], 30),
    api<Item[]>("/content?per_page=8&sort=top", [], 30),
  ]);
  if (!data.item) notFound();
  const item = data.item;
  const body = data.body;
  const translated = body?.vietnamese_body?.trim();
  const sourceText =
    translated ||
    body?.original_body ||
    data.research?.abstract ||
    data.video?.description ||
    item.excerpt ||
    "";
  const isEnglish = Boolean(body?.original_body && !translated && body.original_language !== "vi");
  // Chỉ bài báo kết quả có tỷ số ngay trong tiêu đề mới được gắn badge.
  // Video thường chứa timecode như 0:00, 1:35 nên tuyệt đối không suy diễn
  // các con số trong mô tả thành tỷ số trận đấu.
  const score = item.type === "article" ? scorelineFrom(item.title) : "";
  return (
    <>
      <main className="wrap article-page">
        <div className="article-kicker">
          <span className="tag">{typeLabel(item.type)}</span>
          <span>{item.source_name || "BaoTheX"}</span>
          <span>{formatDate(item.published_at)}</span>
          {score ? <strong className="score-badge">TỶ SỐ {score}</strong> : null}
        </div>
        <h1 className="article-title">{item.title}</h1>
        <p className="article-lede">
          {item.summary || item.excerpt || "Bài viết đang được biên tập và tóm tắt."}
        </p>
        {item.story_cluster_id && (item.cluster_source_count || 0) > 1 ? (
          <Link className="cluster-callout" href={`/su-kien/${item.story_cluster_id}`}>
            <span>MỘT SỰ KIỆN · MỌI NGUỒN</span>
            <strong>{item.cluster_source_count} nguồn đang cùng đưa tin</strong>
            <b>Xem bản tổng hợp trung lập →</b>
          </Link>
        ) : null}
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
            <div className="article-aside-line">
              <b>Xác minh</b>
              <span>{verificationLabel(item.verification_status)}</span>
            </div>
            {(item.source_quality || 0) >= 4 ? (
              <div className="article-aside-line">
                <b>Uy tín nguồn</b>
                <span>{item.source_quality}/5 · Nguồn uy tín</span>
              </div>
            ) : null}
            {data.article?.author ? (
              <div className="article-aside-line">
                <b>Tác giả</b>
                <span>{data.article.author}</span>
              </div>
            ) : null}
            {data.article?.word_count ? (
              <div className="article-aside-line">
                <b>Độ dài</b>
                <span>{data.article.word_count} từ</span>
              </div>
            ) : null}
            {data.research?.journal ? (
              <div className="article-aside-line">
                <b>Tạp chí</b>
                <span>{data.research.journal}</span>
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
            {data.video?.youtube_id ? (
              <a
                className="youtube-watch-card"
                href={
                  item.canonical_url || `https://www.youtube.com/watch?v=${data.video.youtube_id}`
                }
                target="_blank"
                rel="noreferrer"
              >
                {data.video.thumbnail_url || item.image_url ? (
                  <img src={data.video.thumbnail_url || item.image_url} alt="" />
                ) : null}
                <div>
                  <span>VIDEO YOUTUBE · {data.video.channel_title || item.source_name}</span>
                  <strong>▶ Xem video trên YouTube</strong>
                  <small>
                    {data.video.duration_sec
                      ? formatDuration(data.video.duration_sec)
                      : "Video mới"}
                    {data.video.yt_views ? ` · ${formatViews(data.video.yt_views)} lượt xem` : ""}
                  </small>
                </div>
              </a>
            ) : null}
            {isEnglish ? (
              <div className="translation-banner">
                <b>Bài gốc đang ở tiếng Anh</b>
                <span>
                  BaoTheX chỉ dịch nguồn quốc tế; nguồn tiếng Việt được giữ nguyên để tiết kiệm
                  quota.
                </span>
                <TranslateButton contentId={item.id} />
              </div>
            ) : null}
            {translated ? (
              <div className="translation-badge">BẢN TIẾNG VIỆT ĐÃ ĐƯỢC BIÊN TẬP</div>
            ) : null}
            <section className="article-content">
              {item.summary ? (
                <div className="article-summary">
                  <span>TÓM TẮT NHANH</span>
                  <p>{item.summary}</p>
                </div>
              ) : null}
              <section className="article-section article-main-copy">
                <h2>
                  {item.type === "video"
                    ? "Giới thiệu video"
                    : translated
                      ? "Nội dung bài viết"
                      : "Nội dung nguồn"}
                </h2>
                <ReadableText text={sourceText} />
                {item.type === "article" && !body?.original_body && !data.research?.abstract ? (
                  <div className="notice">
                    Nguồn RSS hiện chỉ cung cấp phần tóm tắt. Mở bài gốc để đọc toàn văn; hệ thống
                    sẽ hiển thị toàn văn ngay khi nguồn cho phép cung cấp nội dung đầy đủ.
                  </div>
                ) : null}
              </section>
              {item.key_points?.length ? (
                <ArticleList title="Điểm chính" items={item.key_points} />
              ) : null}
              {data.research?.breakdown ? (
                <>
                  <ArticleSection
                    title="Câu hỏi nghiên cứu"
                    text={data.research.breakdown.question}
                  />
                  <ArticleSection
                    title="Đối tượng và phương pháp"
                    text={[
                      data.research.breakdown.participants,
                      data.research.breakdown.intervention,
                    ]
                      .filter(Boolean)
                      .join("\n\n")}
                  />
                  <ArticleList title="Phát hiện chính" items={data.research.breakdown.findings} />
                  <ArticleSection
                    title="Giới hạn cần biết"
                    text={
                      data.research.breakdown.not_proven ||
                      (data.research.breakdown.limitations || []).join("\n\n")
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
          <aside className="article-rail">
            <div className="article-rail-card">
              <span className="tag">TIN NÓNG</span>
              <h3>Đang được quan tâm</h3>
              {newsroom.slice(0, 5).map((x) => (
                <Link href={`/noi-dung/${x.id}`} key={x.id}>
                  <small>{x.source_name || "BaoTheX"}</small>
                  <strong>{x.title}</strong>
                </Link>
              ))}
            </div>
            <div className="article-rail-card">
              <span className="tag">ĐỌC TIẾP</span>
              <h3>Cùng chủ đề</h3>
              {related.slice(0, 4).map((x) => (
                <Link href={`/noi-dung/${x.id}`} key={x.id}>
                  <strong>{x.title}</strong>
                </Link>
              ))}
            </div>
          </aside>
        </div>
      </main>
      <Footer />
    </>
  );
}

function ReadableText({ text }: { text: string }) {
  const paragraphs = sanitizeArticleText(text);
  return (
    <div className="article-prose">
      {paragraphs.map((p, i) => (
        <p key={`${i}-${p.slice(0, 12)}`}>{p}</p>
      ))}
    </div>
  );
}

function sanitizeArticleText(text: string) {
  const withoutConsent = text.replace(
    /(?:Để hiển thị nội dung này từ YouTube|To display this content from YouTube)[\s\S]*?(?:Thử lại|Try again)/gi,
    "\n",
  );
  const lines = withoutConsent
    .replace(
      /(Chấp nhận|Quản lý lựa chọn của tôi|Ảnh bìa:|Phát sóng ngày:|Chia sẻ|Video thực hiện bởi:|Từ khóa cho bài viết này|Đọc thêm|Đọc ít lại)/gi,
      "\n$1",
    )
    .split(/\n+/)
    .map((line) => line.trim())
    .filter(Boolean);
  const paragraphs: string[] = [];
  let skipNext = false;
  for (const line of lines) {
    if (skipNext) {
      skipNext = false;
      continue;
    }
    if (/^(?:đọc thêm|read more)(?:\b|\s|\.{3}|…)/i.test(line)) break;
    if (/^(?:video thực hiện bởi|từ khóa cho bài viết này)/i.test(line)) {
      skipNext = true;
      continue;
    }
    if (
      /^(?:chấp nhận|accept|quản lý lựa chọn của tôi|manage my choices|chia sẻ|share|đọc ít lại|read less|thử lại|try again|thể thao|\d{1,2}:\d{2})$/i.test(
        line,
      ) ||
      /^(?:ảnh bìa:|phát sóng ngày:)/i.test(line)
    ) {
      continue;
    }
    paragraphs.push(line);
  }
  return paragraphs;
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
function scorelineFrom(text: string) {
  const match = text.match(/\b(\d{1,2})\s*[-–:]\s*(\d{1,2})\b/);
  return match ? `${match[1]}-${match[2]}` : "";
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
function verificationLabel(value?: string) {
  if (value === "confirmed") return "Đã xác nhận";
  if (value === "verifying") return "Đang xác minh";
  return "Tin đồn / một nguồn";
}
function formatDuration(seconds: number) {
  const minutes = Math.floor(seconds / 60);
  const rest = seconds % 60;
  return `${minutes}:${rest.toString().padStart(2, "0")}`;
}
function formatViews(value: number) {
  return new Intl.NumberFormat("vi-VN", { notation: "compact", maximumFractionDigits: 1 }).format(
    value,
  );
}
