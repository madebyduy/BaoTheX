import { api, articleHref, type Item } from "../lib";

const SITE = process.env.NEXT_PUBLIC_SITE_URL || "https://baothex.vn";
const TITLE = "BaoTheX — Báo thể thao chọn lọc";
const DESCRIPTION =
  "Tin thể thao nổi bật trong ngày, được tổng hợp, kiểm chứng nguồn và biên tập bằng tiếng Việt.";

// The site's own RSS 2.0 feed. Aggregators, Google/Bing and Zalo use it to
// discover new articles quickly; it mirrors the newest ready articles.
export const revalidate = 900;

function xmlEscape(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&apos;");
}

export async function GET() {
  const items = await api<Item[]>("/content?type=article&per_page=40&sort=recent", []);
  const now = new Date().toUTCString();

  const entries = items
    .map((item) => {
      const link = xmlEscape(`${SITE}${articleHref(item)}`);
      const title = xmlEscape(item.title);
      const description = xmlEscape(
        (item.summary || item.excerpt || "").replace(/\s+/g, " ").trim().slice(0, 500),
      );
      const pubDate = item.published_at ? new Date(item.published_at).toUTCString() : now;
      const source = xmlEscape(item.source_name || "BaoTheX");
      return `    <item>
      <title>${title}</title>
      <link>${link}</link>
      <guid isPermaLink="true">${link}</guid>
      <description>${description}</description>
      <category>${source}</category>
      <pubDate>${pubDate}</pubDate>
    </item>`;
    })
    .join("\n");

  const body = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>${xmlEscape(TITLE)}</title>
    <link>${SITE}</link>
    <description>${xmlEscape(DESCRIPTION)}</description>
    <language>vi</language>
    <lastBuildDate>${now}</lastBuildDate>
    <atom:link href="${SITE}/feed.xml" rel="self" type="application/rss+xml" />
${entries}
  </channel>
</rss>`;

  return new Response(body, {
    headers: {
      "Content-Type": "application/rss+xml; charset=utf-8",
      "Cache-Control": "public, max-age=600, s-maxage=900, stale-while-revalidate=1800",
    },
  });
}
