import type { MetadataRoute } from "next";
import { api, articleHref, type Competition, type Item } from "./lib";

const SITE = process.env.NEXT_PUBLIC_SITE_URL || "https://baothex.vn";

// Rebuild the sitemap hourly so newly published articles get discovered fast.
export const revalidate = 3600;

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const [items, competitions] = await Promise.all([
    api<Item[]>("/content?per_page=500", []),
    api<Competition[]>("/competitions", []),
  ]);

  const staticRoutes: MetadataRoute.Sitemap = [
    { path: "", priority: 1, freq: "hourly" as const },
    { path: "/chu-de", priority: 0.7, freq: "daily" as const },
    { path: "/nguon", priority: 0.5, freq: "weekly" as const },
    { path: "/video", priority: 0.6, freq: "daily" as const },
    { path: "/lich-the-thao", priority: 0.9, freq: "hourly" as const },
    { path: "/bat-kip", priority: 0.8, freq: "hourly" as const },
    { path: "/ban-the-thao", priority: 0.8, freq: "daily" as const },
    { path: "/du-doan", priority: 0.6, freq: "daily" as const },
    { path: "/premium", priority: 0.5, freq: "monthly" as const },
    { path: "/gioi-thieu", priority: 0.4, freq: "monthly" as const },
    { path: "/chinh-sach-bien-tap", priority: 0.4, freq: "monthly" as const },
    { path: "/nguyen-tac-kiem-chung", priority: 0.4, freq: "monthly" as const },
    { path: "/lien-he", priority: 0.3, freq: "monthly" as const },
    { path: "/ban-quyen", priority: 0.3, freq: "monthly" as const },
    { path: "/dieu-khoan", priority: 0.3, freq: "monthly" as const },
    { path: "/quyen-rieng-tu", priority: 0.3, freq: "monthly" as const },
  ].map((r) => ({
    url: `${SITE}${r.path}`,
    lastModified: new Date(),
    changeFrequency: r.freq,
    priority: r.priority,
  }));

  const articles: MetadataRoute.Sitemap = items.map((it) => ({
    url: `${SITE}${articleHref(it)}`,
    lastModified: it.published_at ? new Date(it.published_at) : new Date(),
    changeFrequency: "weekly",
    priority: 0.8,
  }));

  const leagues: MetadataRoute.Sitemap = competitions.map((c) => ({
    url: `${SITE}/giai-dau/${c.slug}`,
    lastModified: new Date(),
    changeFrequency: "daily",
    priority: 0.6,
  }));

  return [...staticRoutes, ...articles, ...leagues];
}
