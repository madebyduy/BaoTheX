import Link from "next/link";
import { api, demoItems, demoTopics, type Item, type Topic } from "./lib";
import { Footer, PageTitle } from "./ui";

const hotNews = [
  ["Nghiên cứu mới", "Volume tập luyện và ngưỡng tăng cơ", "12 phút trước"],
  ["Dinh dưỡng", "Creatine có thật sự cần dùng mỗi ngày?", "28 phút trước"],
  ["Sức mạnh", "Kỹ thuật squat cho người tập lâu năm", "1 giờ trước"],
  ["Phục hồi", "Giấc ngủ ảnh hưởng đến hiệu suất thế nào?", "2 giờ trước"],
];

function FeedStory({
  item,
  image = "",
  featured = false,
}: {
  item: Item;
  image?: string;
  featured?: boolean;
}) {
  if (featured)
    return (
      <article className="featured">
        <div className="featured-meta">
          ▣ {item.source_name || "BaoTheX"} · 1 giờ trước · 🔥 Nổi bật
        </div>
        <h2>{item.title}</h2>
        <p>
          {item.summary ||
            item.excerpt ||
            "Nội dung được tóm tắt từ nguồn đáng tin cậy và trình bày ngắn gọn."}
        </p>
        <div className="featured-footer">
          <div className="source-stack">
            <div className="source-dots">
              <span className="avatar">B</span>
              <span className="avatar">R</span>
              <span className="avatar">N</span>
            </div>
            <span>3 nguồn đưa tin</span>
          </div>
          <Link className="arrow-btn" href={`/noi-dung/${item.id}`}>
            Xem tin →
          </Link>
        </div>
      </article>
    );
  return (
    <article className="feed-card">
      <div className="feed-type">
        {item.type === "research"
          ? "NGHIÊN CỨU"
          : item.type === "video"
            ? "VIDEO"
            : item.type === "podcast"
              ? "PODCAST"
              : "BÀI VIẾT"}
      </div>
      <div className="feed-meta">
        <span>◈</span>
        <strong>{item.source_name || "BaoTheX"}</strong>
        <span>·</span>
        <span>3 giờ trước</span>
      </div>
      <Link className="feed-card-title" href={`/noi-dung/${item.id}`}>
        <h2>{item.title}</h2>
      </Link>
      <p>{item.summary || item.excerpt || "Bài viết được BaoTheX chọn lọc và tóm tắt."}</p>
      {image && (
        <Link className={`feed-image ${image}`} href={`/noi-dung/${item.id}`}>
          {item.type === "video" ? "▶" : "BX"}
        </Link>
      )}
      <div className="feed-actions">
        <span>♡ 1,2k</span>
        <span>⇄ 184</span>
        <span>◌ 96</span>
        <Link href={`/noi-dung/${item.id}`}>↗ Mở bài →</Link>
      </div>
    </article>
  );
}

export default async function Home() {
  const home = await api<{ items?: Item[]; topics?: Topic[] }>("/home", {});
  const items = home.items?.length
    ? home.items
    : await api<Item[]>("/content?per_page=8&sort=recent", demoItems);
  const topics = home.topics?.length ? home.topics : demoTopics;
  const lead = items[0] || demoItems[0];
  const latestItems = items.slice(5, 8).length ? items.slice(5, 8) : items.slice(1, 4);
  return (
    <>
      <main className="wrap dashboard">
        <aside className="sidebar">
          <div className="side-section">
            <div className="side-link active">
              🏠 <span>Trang chủ</span>
            </div>
            <Link className="side-link" href="/danh-muc">
              ◷ <span>Mới nhất</span>
            </Link>
            <Link className="side-link" href="/nghien-cuu">
              🎁 <span>Nghiên cứu nổi bật</span>
            </Link>
          </div>
          <div className="side-section">
            <div className="side-title">Lọc nguồn</div>
            <Link className="side-link" href="/nguon">
              ◉ <span>Tất cả nguồn</span>
              <span className="count">24</span>
            </Link>
            <Link className="side-link" href="/danh-muc">
              ▤ <span>Bài viết</span>
            </Link>
            <Link className="side-link" href="/video">
              ▶ <span>Video</span>
            </Link>
            <Link className="side-link" href="/podcast">
              ◈ <span>Podcast</span>
            </Link>
          </div>
          <div className="side-section">
            <div className="side-title">Theo dõi</div>
            <div className="follow-list">
              {[
                "Stronger by Science",
                "Journal of Strength",
                "Barbell Medicine",
                "Europe PMC",
                "Iron Culture",
              ].map((source) => (
                <div className="follow-item" key={source}>
                  <span className="avatar">{source[0]}</span>
                  {source}
                </div>
              ))}
            </div>
          </div>
        </aside>
        <section className="main-feed">
          <div className="feed-heading">
            <h1>Dòng tin</h1>
            <div className="feed-tabs">
              <span className="active">Dành cho bạn</span>
              <span>Mới nhất</span>
            </div>
          </div>
          <FeedStory item={lead} featured />
          {items.slice(1, 5).map((item, index) => (
            <FeedStory
              key={item.id}
              item={item}
              image={index === 0 ? "blue" : index === 1 ? "orange" : ""}
            />
          ))}
          <section className="newsroom-detail">
            <div className="section-head detail-head">
              <div>
                <span className="tag">Cập nhật liên tục</span>
                <h2>Mới trong ngày</h2>
              </div>
              <Link className="read-link" href="/danh-muc">
                Xem dòng tin →
              </Link>
            </div>
            <div className="latest-list">
              {latestItems.map((item, index) => (
                <Link className="latest-row" href={`/noi-dung/${item.id}`} key={item.id}>
                  <span className="latest-number">0{index + 1}</span>
                  <span className="latest-copy">
                    <strong>{item.title}</strong>
                    <small>
                      {item.source_name || "BaoTheX"} · {index + 1} giờ trước · 6 phút đọc
                    </small>
                  </span>
                  <span className="latest-arrow">↗</span>
                </Link>
              ))}
            </div>
            <div className="editor-note">
              <span className="editor-note-mark">BX</span>
              <div>
                <strong>Biên tập có chọn lọc</strong>
                <p>
                  Mỗi nội dung đều được gắn nguồn, phân loại và tóm tắt để bạn nắm ý chính trước khi
                  đọc sâu.
                </p>
              </div>
            </div>
          </section>
          <div className="section">
            <div className="section-head">
              <div>
                <span className="tag">Khám phá</span>
                <h2>Chủ đề đang được quan tâm</h2>
              </div>
              <Link className="read-link" href="/chu-de">
                Xem tất cả →
              </Link>
            </div>
            <div className="topic-links">
              {topics.slice(0, 8).map((topic) => (
                <Link href={`/chu-de/${topic.slug}`} key={topic.id}>
                  {topic.name}
                  <span>→</span>
                </Link>
              ))}
            </div>
          </div>
        </section>
        <aside className="rightbar">
          <div className="right-card">
            <div className="right-head">
              <span>🔥 Tin nóng</span>
              <span className="live">● LIVE</span>
            </div>
            {hotNews.map(([category, title, time]) => (
              <div className="hot-item" key={title}>
                <span className="hot-icon">✦</span>
                <div>
                  {title}
                  <small>
                    {category} · {time}
                  </small>
                </div>
              </div>
            ))}
          </div>
          <div className="right-card">
            <div className="right-head">
              <span>📝 Tin hôm nay</span>
            </div>
            {items.slice(1, 5).map((item) => (
              <Link className="hot-item" href={`/noi-dung/${item.id}`} key={item.id}>
                <span className="hot-icon">▣</span>
                <div>
                  {item.title}
                  <small>{item.source_name || "BaoTheX"} · mới cập nhật</small>
                </div>
              </Link>
            ))}
          </div>
        </aside>
      </main>
      <Footer />
    </>
  );
}
