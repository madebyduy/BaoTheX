import type { Metadata } from "next";

// pageMetadata builds a consistent Metadata block (title, description, canonical,
// Open Graph, Twitter) for a route. metadataBase in the root layout resolves the
// relative canonical/OG url to an absolute one. Pass index:false for member-only
// or utility pages that should stay out of the index.
export function pageMetadata(opts: {
  title: string;
  description: string;
  path: string;
  index?: boolean;
}): Metadata {
  return {
    title: opts.title,
    description: opts.description,
    alternates: { canonical: opts.path },
    robots: opts.index === false ? { index: false, follow: false } : undefined,
    openGraph: {
      type: "website",
      siteName: "BaoTheX",
      locale: "vi_VN",
      title: opts.title,
      description: opts.description,
      url: opts.path,
    },
    twitter: {
      card: "summary_large_image",
      title: opts.title,
      description: opts.description,
    },
  };
}

export type Item = {
  id: number;
  source_id?: number;
  title: string;
  type: string;
  summary?: string;
  excerpt?: string;
  source_name?: string;
  published_at?: string;
  canonical_url?: string;
  image_url?: string;
  key_points?: string[];
  status?: string;
  updated_at?: string;
  view_count?: number;
  save_count?: number;
  final_score?: number;
  language?: string;
  story_cluster_id?: number;
  cluster_source_count?: number;
  verification_status?: "rumor" | "verifying" | "confirmed";
  source_quality?: number;
  quality_state?: "pending" | "passed" | "review";
  quality_flags?: string[];
  quality_checked_at?: string;
};
export type StoryCluster = {
  id: number;
  representative_title: string;
  primary_content_id?: number;
  verification_status: "rumor" | "verifying" | "confirmed";
  source_count: number;
  created_at: string;
  updated_at: string;
  items: Item[];
};
export type ContentBody = {
  content_id: number;
  original_language: string;
  original_body: string;
  vietnamese_body?: string;
  translation_status: string;
};
export type Topic = {
  id: number;
  slug: string;
  name: string;
  description?: string;
  category?: string;
  follower_count?: number;
};
export type Source = {
  id: number;
  kind?: string;
  name: string;
  homepage_url?: string;
  quality?: number;
  default_lang?: string;
  enabled?: boolean;
};
export type Entity = {
  id: number;
  slug: string;
  name: string;
  kind?: string;
  bio?: string;
  avatar_url?: string;
  expertise?: string[];
  follower_count?: number;
};
export type Sport = {
  id: number;
  slug: string;
  name: string;
  enabled: boolean;
};
export type Competition = {
  id: number;
  sport_id?: number;
  sport_slug?: string;
  slug: string;
  name: string;
  country?: string;
  data_source?: string;
  coverage?: string;
};
export type SportsEvent = {
  id: number;
  sport_id: number;
  sport_slug: string;
  sport_name: string;
  competition_id?: number;
  competition?: string;
  title: string;
  home_name?: string;
  away_name?: string;
  starts_at: string;
  status: "scheduled" | "live" | "finished" | "postponed" | "cancelled";
  home_score?: string;
  away_score?: string;
  data_source: string;
  data_updated_at: string;
  freshness: "live" | "delayed" | "scheduled" | "manual" | string;
  is_manual: boolean;
  manual_locked: boolean;
  following?: boolean;
  related_content?: Item[];
};
export type Prediction = {
  id: number;
  event_id?: number;
  kind: "winner" | "score" | "player" | "quiz" | "poll";
  question: string;
  options: string[];
  correct_option?: string;
  deadline: string;
  status: string;
  points: number;
  user_answer?: string;
  is_correct?: boolean;
  answer_count: number;
};
export type FanPassport = {
  active_days: number;
  current_streak: number;
  articles_read: number;
  events_followed: number;
  predictions: number;
  predictions_correct: number;
  points: number;
  badges: string[];
};
// Server-rendered requests use Docker's private service address when present;
// browser components continue to use the public URL baked into the bundle.
const API =
  (typeof window === "undefined" && process.env.API_INTERNAL_URL) ||
  process.env.NEXT_PUBLIC_API_URL ||
  "http://localhost:8081";
const IS_PRODUCTION = process.env.NODE_ENV === "production";

function safeFallback<T>(fallback: T): T {
  // Seed/demo records are useful for local UI work, but publishing them during
  // an outage is editorially misleading. Array fallbacks therefore become an
  // honest empty state in production; object fallbacks retain only their shape.
  return (IS_PRODUCTION && Array.isArray(fallback) ? [] : fallback) as T;
}

function reportAPIFailure(path: string, detail: string) {
  // Warn, not error: every caller of api() already degrades gracefully to a
  // fallback, so an upstream hiccup (a 503 while the audio brief is still being
  // generated, a background ISR revalidation that missed) is a handled
  // condition, not a crash. Logging it at error level made Next.js 15 dev
  // promote it into a full-screen error overlay for something the page had
  // already recovered from.
  console.warn(`[BaoTheX API] ${path}: ${detail}`);
}
// Public content is cached and revalidated in the background (ISR). Without a
// default the whole site refetched every request, adding a Tokyo round-trip per
// call. Pass an explicit revalidate to tune, or 0 to always hit the API.
export async function api<T>(path: string, fallback: T, revalidate = 60): Promise<T> {
  try {
    const r = await fetch(
      `${API}/api/v1${path}`,
      revalidate > 0 ? { next: { revalidate } } : { cache: "no-store" },
    );
    if (!r.ok) {
      reportAPIFailure(path, `HTTP ${r.status}`);
      return safeFallback(fallback);
    }
    const json = await r.json();
    return (json.data ?? json) as T;
  } catch (error) {
    reportAPIFailure(path, error instanceof Error ? error.message : "request failed");
    return safeFallback(fallback);
  }
}

export async function apiWithCookie<T>(path: string, fallback: T, cookie: string): Promise<T> {
  if (!cookie) return fallback;
  try {
    const response = await fetch(`${API}/api/v1${path}`, {
      cache: "no-store",
      headers: { cookie },
    });
    if (!response.ok) {
      reportAPIFailure(path, `HTTP ${response.status}`);
      return safeFallback(fallback);
    }
    const json = await response.json();
    return (json.data ?? json) as T;
  } catch (error) {
    reportAPIFailure(path, error instanceof Error ? error.message : "request failed");
    return safeFallback(fallback);
  }
}
const demoItemSeeds: Item[] = [
  {
    id: 1,
    type: "article",
    title: "Tin thể thao nổi bật trong ngày",
    summary: "Những diễn biến đáng chú ý nhất từ sân cỏ và các giải đấu lớn.",
    source_name: "BaoTheX",
  },
  {
    id: 2,
    type: "video",
    title: "Video đáng xem: Những trận đấu không thể bỏ lỡ",
    summary: "Tóm tắt lịch thi đấu, kết quả và câu chuyện sau trận.",
    source_name: "BaoTheX",
  },
  {
    id: 3,
    type: "article",
    title: "Thể thao Việt Nam hướng đến mục tiêu mới",
    summary: "Các vận động viên và đội tuyển đang chuẩn bị cho những giải đấu quan trọng.",
    source_name: "BaoTheX",
  },
];
export const demoItems: Item[] = IS_PRODUCTION ? [] : demoItemSeeds;
const demoTopicSeeds: Topic[] = [
  "Bóng đá Việt Nam",
  "Bóng đá quốc tế",
  "Bóng rổ",
  "Tennis",
  "Thể thao Việt Nam",
  "F1 & thể thao motor",
  "Esports",
  "Các môn khác",
].map((name, i) => ({
  id: i + 1,
  name,
  slug: name
    .toLowerCase()
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/đ/g, "d")
    .replace(/[^a-z0-9]+/g, "-"),
  follower_count: 20 + i * 13,
  description: `Tin mới nhất về ${name.toLowerCase()}.`,
}));
export const demoTopics: Topic[] = IS_PRODUCTION ? [] : demoTopicSeeds;
// slugify turns a Vietnamese headline into an ASCII, hyphenated URL slug.
export function slugify(text: string): string {
  return text
    .toLowerCase()
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .replace(/đ/g, "d")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 70)
    .replace(/-+$/g, "");
}

// articleHref builds the canonical, SEO-friendly path for a content item:
// /noi-dung/<slug>-<id>. The numeric id is always appended so it stays
// parseable from the URL tail no matter how the title slugifies (or if the
// title later changes). Callers that only have an id still get a working link.
export function articleHref(item: { id: number | string; title?: string }): string {
  const slug = item.title ? slugify(item.title) : "";
  return slug ? `/noi-dung/${slug}-${item.id}` : `/noi-dung/${item.id}`;
}

// idFromSlug extracts the trailing numeric id from a /noi-dung/<slug>-<id>
// segment. Legacy id-only URLs (/noi-dung/123) parse unchanged.
export function idFromSlug(param: string): string {
  const match = /(\d+)$/.exec(param);
  return match ? match[1] : param;
}

// JSON-LD lives in a script element. JSON.stringify alone does not neutralise
// a source headline containing </script>; escaping '<' prevents that sequence
// from terminating the element while preserving the exact JSON value.
export function safeJsonLd(value: unknown): string {
  return JSON.stringify(value).replace(/</g, "\\u003c");
}

export function typeLabel(type: string) {
  return (
    (
      {
        research: "Nghiên cứu",
        video: "Video",
        podcast: "Podcast",
        article: "Bài viết",
        event: "Sự kiện",
      } as Record<string, string>
    )[type] || "Nội dung"
  );
}
