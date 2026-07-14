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
    type: "research",
    title: "Tăng volume tập giúp cơ phát triển tới khoảng 20 set mỗi nhóm cơ mỗi tuần",
    summary: "Phân tích gộp mới cho thấy lợi ích giảm dần sau ngưỡng này.",
    source_name: "Journal of Strength & Conditioning",
  },
  {
    id: 2,
    type: "video",
    title: "Cách hiểu đúng về EMG và bài đẩy ngực",
    summary: "Giải thích các biến số kỹ thuật và ý nghĩa thực tế của chỉ số EMG.",
    source_name: "Kênh khoa học tập luyện",
  },
  {
    id: 3,
    type: "article",
    title: "Protein, leucine và thời điểm nạp trong ngày",
    summary: "Những điều nghiên cứu hiện tại thực sự cho chúng ta biết.",
    source_name: "BaoTheX biên tập",
  },
  {
    id: 4,
    type: "podcast",
    title: "Phục hồi, giấc ngủ và hiệu suất tập luyện",
    summary: "Cuộc trò chuyện dài về cách xây dựng lịch tập bền vững.",
    source_name: "Iron Culture",
  },
];
export const demoTopics: Topic[] = [
  "Phì đại cơ",
  "Sức mạnh",
  "Dinh dưỡng",
  "Creatine",
  "Phục hồi",
  "Giấc ngủ",
  "Cardio",
  "Chấn thương",
].map((name, i) => ({
  id: i + 1,
  slug: name
    .toLowerCase()
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/đ/g, "d")
    .replace(/\s+/g, "-"),
  name,
  follower_count: 20 + i * 13,
  description: `Kiến thức và nội dung chọn lọc về ${name.toLowerCase()}.`,
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
