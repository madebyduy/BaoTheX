import { api, articleHref, type Item } from "../lib";

const SITE = process.env.NEXT_PUBLIC_SITE_URL || "https://baothex.vn";
const PUBLICATION = "BaoTheX";

// Google News only considers articles from roughly the last two days, so the
// news sitemap is deliberately small and refreshed often. It is separate from
// the main sitemap.xml, which lists the whole archive for ordinary crawling.
export const revalidate = 600;

function xmlEscape(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&apos;");
}

export async function GET() {
  const items = await api<Item[]>("/content?type=article&per_page=100&sort=recent", []);
  const cutoff = Date.now() - 48 * 60 * 60 * 1000;
  const fresh = items.filter((item) => {
    if (!item.published_at) return false;
    return new Date(item.published_at).getTime() >= cutoff;
  });

  const urls = fresh
    .map((item) => {
      const loc = xmlEscape(`${SITE}${articleHref(item)}`);
      const published = new Date(item.published_at as string).toISOString();
      const title = xmlEscape(item.title);
      return `  <url>
    <loc>${loc}</loc>
    <news:news>
      <news:publication>
        <news:name>${PUBLICATION}</news:name>
        <news:language>vi</news:language>
      </news:publication>
      <news:publication_date>${published}</news:publication_date>
      <news:title>${title}</news:title>
    </news:news>
  </url>`;
    })
    .join("\n");

  const body = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
        xmlns:news="http://www.google.com/schemas/sitemap-news/0.9">
${urls}
</urlset>`;

  return new Response(body, {
    headers: {
      "Content-Type": "application/xml; charset=utf-8",
      "Cache-Control": "public, max-age=300, s-maxage=600, stale-while-revalidate=1800",
    },
  });
}
