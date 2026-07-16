import type { MetadataRoute } from "next";
import { api, type Item } from "./lib";

const SITE = process.env.NEXT_PUBLIC_SITE_URL || "https://baothex.vn";

// Rebuild the sitemap hourly so newly published articles get discovered fast.
export const revalidate = 3600;

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const items = await api<Item[]>("/content?per_page=500", []);

  const staticRoutes: MetadataRoute.Sitemap = [
    { path: "", priority: 1, freq: "hourly" as const },
    { path: "/chu-de", priority: 0.7, freq: "daily" as const },
    { path: "/nguon", priority: 0.5, freq: "weekly" as const },
    { path: "/video", priority: 0.6, freq: "daily" as const },
    { path: "/premium", priority: 0.5, freq: "monthly" as const },
  ].map((r) => ({
    url: `${SITE}${r.path}`,
    lastModified: new Date(),
    changeFrequency: r.freq,
    priority: r.priority,
  }));

  const articles: MetadataRoute.Sitemap = items.map((it) => ({
    url: `${SITE}/noi-dung/${it.id}`,
    lastModified: it.published_at ? new Date(it.published_at) : new Date(),
    changeFrequency: "weekly",
    priority: 0.8,
  }));

  return [...staticRoutes, ...articles];
}
