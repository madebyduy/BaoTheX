export type Item = {
  id: number;
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
  view_count?: number;
  save_count?: number;
  language?: string;
  story_cluster_id?: number;
  cluster_source_count?: number;
  verification_status?: "rumor" | "verifying" | "confirmed";
  source_quality?: number;
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
const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";
export async function api<T>(path: string, fallback: T): Promise<T> {
  try {
    const r = await fetch(`${API}/api/v1${path}`, { cache: "no-store" });
    if (!r.ok) return fallback;
    const json = await r.json();
    return (json.data ?? json) as T;
  } catch {
    return fallback;
  }
}
export const demoItems: Item[] = [
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
    title: "Bản tin nhanh: Những trận đấu không thể bỏ lỡ",
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
export const demoTopics: Topic[] = [
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
