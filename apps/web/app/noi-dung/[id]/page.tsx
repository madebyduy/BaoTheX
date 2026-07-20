import type { Metadata } from "next";
import Link from "next/link";
import { notFound, permanentRedirect } from "next/navigation";
import {
  api,
  articleHref,
  idFromSlug,
  safeJsonLd,
  type ContentBody,
  type Item,
  type Topic,
  typeLabel,
} from "../../lib";
import { Footer, RemoteImage } from "../../ui";
import { LikeButton, SaveButton, ShareBar, ReadingProgress } from "../../action-buttons";
import { ReadingTracker } from "../../product-analytics";
import { ContentFeedback } from "../../content-feedback";
import { PerspectiveAction } from "./perspective-action";

const SITE = process.env.NEXT_PUBLIC_SITE_URL || "https://baothex.vn";
// generateMetadata and the page both need the article. Using the SAME revalidate
// lets Next dedupe them into a single upstream request instead of two.
const ARTICLE_REVALIDATE = 60;

type Detail = {
  item?: Item;
  topics?: Topic[];
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

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}): Promise<Metadata> {
  const { id: param } = await params;
  const id = idFromSlug(param);
  const data = await api<Detail>(`/content/${id}`, {}, ARTICLE_REVALIDATE);
  const item = data.item;
  if (!item) return { title: "Không tìm thấy bài viết" };
  const description = (
    item.summary ||
    item.excerpt ||
    "Tin thể thao chọn lọc, kiểm chứng nguồn trên BaoTheX."
  )
    .replace(/\s+/g, " ")
    .trim()
    .slice(0, 300);
  const url = `${SITE}${articleHref(item)}`;
  const images = item.image_url ? [item.image_url] : undefined;
  return {
    title: item.title,
    description,
    alternates: { canonical: url },
    openGraph: {
      type: "article",
      siteName: "BaoTheX",
      locale: "vi_VN",
      title: item.title,
      description,
      url,
      images,
      publishedTime: item.published_at,
    },
    twitter: {
      card: "summary_large_image",
      title: item.title,
      description,
      images,
    },
  };
}

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id: param } = await params;
  const id = idFromSlug(param);
  const [data, related, newsroom] = await Promise.all([
    api<Detail>(`/content/${id}`, {}, ARTICLE_REVALIDATE),
    api<Item[]>(`/content/${id}/related`, [], ARTICLE_REVALIDATE),
    api<Item[]>("/content?per_page=8&sort=top", [], ARTICLE_REVALIDATE),
  ]);
  if (!data.item) notFound();
  const item = data.item;
  // Enforce the canonical slug URL. A legacy /noi-dung/123 link, or a stale slug
  // from before an editor retitled the piece, gets a permanent (308) redirect to
  // /noi-dung/<slug>-<id> so search engines consolidate on one address.
  const canonical = articleHref(item);
  if (`/noi-dung/${param}` !== canonical) {
    permanentRedirect(canonical);
  }
  const tracker = <ReadingTracker contentId={item.id} />;
  const body = data.body;
  // A foreign article is summarised, never reproduced.
  //
  // This page used to fall back to body.original_body when no translation
  // existed, which served readers the verbatim English article from Reuters or
  // the Guardian; when a translation did exist it served a full Vietnamese copy
  // instead. Both are republishing someone else's reporting. What we publish now
  // is our own headline, key points and summary, plus a prominent link to the
  // original — the standard aggregation posture, and the only one that is ours
  // to publish.
  //
  // The winning cluster of the day still gets a full translation stored, purely
  // as raw material for the Góc nhìn analysis. isForeign keeps it off the page
  // regardless.
  const isForeign =
    (item.language && item.language !== "vi") ||
    Boolean(body?.original_language && body.original_language !== "vi");
  const ownText = data.research?.abstract || data.video?.description || "";
  const sourceText = isForeign
    ? ownText
    : ownText || body?.vietnamese_body?.trim() || body?.original_body || item.excerpt || "";
  // Chỉ bài báo kết quả có tỷ số ngay trong tiêu đề mới được gắn badge.
  // Video thường chứa timecode như 0:00, 1:35 nên tuyệt đối không suy diễn
  // các con số trong mô tả thành tỷ số trận đấu.
  const score = item.type === "article" ? scorelineFrom(item.title) : "";
  // Reading time from the stored word count (~200 wpm). Foreign digests have no
  // word_count, so it simply falls away rather than showing a wrong number.
  const readingMinutes = data.article?.word_count
    ? Math.max(1, Math.round(data.article.word_count / 200))
    : 0;
  const updatedLabel =
    item.updated_at &&
    item.published_at &&
    item.updated_at.slice(0, 16) > item.published_at.slice(0, 16)
      ? formatDateTime(item.updated_at)
      : "";
  const pageUrl = `${SITE}${canonical}`;
  // Góc nhìn / editorial pieces store an internal canonical (/goc-nhin/...) and
  // have no external "bài gốc"; only treat http(s) URLs as a real source link.
  const externalSource =
    item.canonical_url && /^https?:\/\//i.test(item.canonical_url) ? item.canonical_url : "";
  const section = data.topics?.[0];
  const dateModified =
    item.updated_at && item.published_at && item.updated_at > item.published_at
      ? item.updated_at
      : item.published_at;
  const breadcrumb = {
    "@type": "BreadcrumbList",
    itemListElement: [
      { "@type": "ListItem", position: 1, name: "Trang chủ", item: SITE },
      section
        ? {
            "@type": "ListItem",
            position: 2,
            name: section.name,
            item: `${SITE}/chu-de/${section.slug}`,
          }
        : {
            "@type": "ListItem",
            position: 2,
            name: typeLabel(item.type),
            item: `${SITE}/danh-muc`,
          },
      { "@type": "ListItem", position: 3, name: item.title, item: pageUrl },
    ],
  };
  const jsonLd = {
    "@context": "https://schema.org",
    "@graph": [
      {
        "@type": "NewsArticle",
        headline: item.title,
        description: (item.summary || item.excerpt || "").slice(0, 300) || undefined,
        image: item.image_url ? [item.image_url] : [`${SITE}${articleHref(item)}/opengraph-image`],
        datePublished: item.published_at,
        dateModified,
        inLanguage: "vi",
        articleSection: section?.name || typeLabel(item.type),
        author: { "@type": "Organization", name: item.source_name || "BaoTheX" },
        publisher: {
          "@type": "Organization",
          name: "BaoTheX",
          logo: { "@type": "ImageObject", url: `${SITE}/icon.png` },
        },
        mainEntityOfPage: { "@type": "WebPage", "@id": pageUrl },
        isBasedOn: externalSource || undefined,
      },
      breadcrumb,
    ],
  };
  return (
    <>
      {tracker}
      <ReadingProgress />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: safeJsonLd(jsonLd) }} />
      <main className="wrap article-page">
        <nav className="breadcrumbs" aria-label="Đường dẫn">
          <Link href="/">Trang chủ</Link>
          <span aria-hidden>›</span>
          {section ? (
            <Link href={`/chu-de/${section.slug}`}>{section.name}</Link>
          ) : (
            <Link href="/danh-muc">{typeLabel(item.type)}</Link>
          )}
          <span aria-hidden>›</span>
          <b>{item.title}</b>
        </nav>
        <div className="article-kicker">
          <span className="tag">{typeLabel(item.type)}</span>
          <span>{item.source_name || "BaoTheX"}</span>
          <span>{formatDate(item.published_at)}</span>
          {readingMinutes ? <span>{readingMinutes} phút đọc</span> : null}
          {updatedLabel ? <span className="article-updated">Cập nhật {updatedLabel}</span> : null}
          {score ? <strong className="score-badge">TỶ SỐ {score}</strong> : null}
        </div>
        <h1 className="article-title">{item.title}</h1>
        <p className="article-lede">
          {item.summary || item.excerpt || "Bài viết đang được biên tập và tóm tắt."}
        </p>
        {item.image_url ? (
          <div className="article-hero">
            <RemoteImage
              src={item.image_url}
              alt=""
              fetchPriority="high"
              decoding="async"
              referrerPolicy="no-referrer"
            />
          </div>
        ) : null}
        <div className="article-actions">
          <LikeButton contentId={item.id} />
          <SaveButton contentId={item.id} />
          <ShareBar title={item.title} />
        </div>
        <ContentFeedback
          contentId={item.id}
          sourceId={item.source_id}
          topicId={data.topics?.[0]?.id}
        />
        <PerspectiveAction contentId={item.id} />
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
            {externalSource ? (
              <a className="article-source" href={externalSource} target="_blank" rel="noreferrer">
                Mở bài gốc ↗
              </a>
            ) : null}
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
                  <RemoteImage
                    src={(data.video.thumbnail_url || item.image_url)!}
                    alt=""
                    loading="lazy"
                    decoding="async"
                    referrerPolicy="no-referrer"
                  />
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
            <section className="article-content">
              {/* Only foreign stories get a summary box here: their digest is
                  several paragraphs and IS the article. For a native article the
                  summary already appears as the lede under the title, so a second
                  copy here was pure duplication — omit it. */}
              {item.summary && isForeign ? (
                <div className="article-summary">
                  <span>TÓM TẮT CỦA BAOTHEX</span>
                  {splitParagraphs(item.summary).map((p, i) => (
                    <p key={`${i}-${p.slice(0, 12)}`}>{p}</p>
                  ))}
                </div>
              ) : null}
              {/* Key points come first for a foreign story: with no body to
                  follow, they are the article rather than a sidebar to it. */}
              {isForeign && item.key_points?.length ? (
                <ArticleList title="Những điểm chính" items={item.key_points} featured />
              ) : null}
              {sourceText ? (
                <section className="article-section article-main-copy">
                  <h2>{item.type === "video" ? "Giới thiệu video" : "Nội dung bài viết"}</h2>
                  <ReadableText text={sourceText} />
                </section>
              ) : null}
              {isForeign && externalSource ? (
                <a
                  className="origin-card"
                  href={externalSource}
                  target="_blank"
                  rel="noreferrer nofollow"
                >
                  <span>BÀI GỐC</span>
                  <strong>Đọc toàn văn trên {item.source_name || "nguồn gốc"} ↗</strong>
                  <small>
                    BaoTheX tóm tắt và biên tập bằng tiếng Việt. Bản quyền nội dung gốc thuộc về{" "}
                    {item.source_name || "nguồn phát hành"}.
                  </small>
                </a>
              ) : null}
              {!isForeign && item.key_points?.length ? (
                <ArticleList title="Điểm chính" items={item.key_points} featured />
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
                  <Link className="related-item" href={articleHref(x)} key={x.id}>
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
                <Link href={articleHref(x)} key={x.id}>
                  <small>{x.source_name || "BaoTheX"}</small>
                  <strong>{x.title}</strong>
                </Link>
              ))}
            </div>
            <div className="article-rail-card">
              <span className="tag">ĐỌC TIẾP</span>
              <h3>Cùng chủ đề</h3>
              {related.slice(0, 4).map((x) => (
                <Link href={articleHref(x)} key={x.id}>
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

// splitParagraphs breaks digest prose on blank lines. Unlike sanitizeArticleText
// it strips nothing: this text is ours, so there is no publisher boilerplate to
// filter out of it.
function splitParagraphs(text: string) {
  return text
    .split(/\n{2,}|\r\n\r\n/)
    .map((p) => p.trim())
    .filter(Boolean);
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
    // Everything after these controls belongs to the publisher UI, not the article.
    // Stop before tracking URLs, hotline forms and error-report dialogs can leak in.
    if (
      /^(?:tags?|copy link|link bài gốc|lấy link|đường dây nóng\s*:|hotline\s*:|gửi báo lỗi|report (?:an )?error)$/i.test(
        line,
      )
    ) {
      break;
    }
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
      /^(?:ảnh bìa:|phát sóng ngày:|báo lỗi cho\b|\*?vui lòng nhập đủ thông tin|đóng$)/i.test(
        line,
      ) ||
      /^(?:vff\s*\||soha)$/i.test(line) ||
      /^\d{1,2}\/\d{1,2}\/\d{4}\s+\d{1,2}:\d{2}$/i.test(line)
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
function ArticleList({
  title,
  items,
  featured = false,
}: {
  title: string;
  items?: string[];
  featured?: boolean;
}) {
  return items?.length ? (
    <section className={`article-section ${featured ? "article-key-points" : ""}`}>
      <h2>{title}</h2>
      <ul>
        {items.map((x, index) => (
          <li key={x}>
            {featured ? <span>{String(index + 1).padStart(2, "0")}</span> : null}
            <p>{x}</p>
          </li>
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
function formatDateTime(value?: string) {
  if (!value) return "";
  try {
    return new Intl.DateTimeFormat("vi-VN", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
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
