import Link from "next/link";
import { api, demoItems, demoTopics, type Item, type Topic } from "./lib";
import { Footer } from "./ui";

type HomeData = { today?: Item[]; sports?: Item[]; videos?: Item[] };

export default async function Home() {
  const home = await api<HomeData>("/home", {});
  const feed = await api<Item[]>("/content?per_page=30&sort=recent", demoItems);
  const latest = Array.from(
    new Map(
      [...(home.today || []), ...(home.sports || []), ...feed].map((item) => [item.id, item]),
    ).values(),
  ).slice(0, 30);
  const topics = await api<Topic[]>("/topics", demoTopics);
  const sportsTopics = topics.filter((topic) =>
    /bong|tennis|the-thao|motor|esport|khac/.test(topic.slug),
  );
  const lead = latest[0] || demoItems[0];
  const secondary = latest.slice(1, 5);
  const scored = latest.filter((item) => scorelineFrom(item)).slice(0, 6);
  const dateLabel = new Intl.DateTimeFormat("vi-VN", {
    weekday: "long",
    day: "2-digit",
    month: "long",
    year: "numeric",
  }).format(new Date());

  return (
    <>
      <main className="wrap sports-home">
        <section className="newsroom-masthead">
          <div className="issue-line">
            <span>{dateLabel}</span>
            <b>
              <i /> Tin mới cập nhật liên tục
            </b>
            <span>Ấn bản số {new Date().getDate().toString().padStart(2, "0")}</span>
          </div>
          <div className="masthead-grid">
            <div className="masthead-copy">
              <span className="tag">BÁO THỂ THAO ĐA NGUỒN</span>
              <h1>
                Bản tin thể thao <em>24h</em>
              </h1>
              <p>
                Tin đáng chú ý, kết quả mới và những câu chuyện thể thao được tuyển chọn từ các
                nguồn uy tín.
              </p>
            </div>
          </div>
          <nav className="category-strip">
            <Link href="/danh-muc">Mới nhất</Link>
            {sportsTopics.slice(0, 7).map((topic) => (
              <Link href={`/chu-de/${topic.slug}`} key={topic.id}>
                {topic.name}
              </Link>
            ))}
          </nav>
        </section>

        <div className="sports-grid">
          <section className="sports-main">
            <div className="section-heading">
              <div>
                <span className="tag">01 · TIN NÓNG</span>
                <h2>Đáng chú ý trong ngày</h2>
              </div>
              <Link href="/danh-muc">Xem tất cả →</Link>
            </div>
            <Link className="sports-lead" href={`/noi-dung/${lead.id}`}>
              {lead.image_url ? (
                <img src={lead.image_url} alt="" />
              ) : (
                <div className="lead-placeholder">BX</div>
              )}
              <div className="lead-copy">
                <span className="tag">
                  {lead.source_name || "BaoTheX"}
                  {scorelineFrom(lead) ? ` · TỶ SỐ ${scorelineFrom(lead)}` : ""}
                </span>
                <StorySignals item={lead} />
                <h3>{lead.title}</h3>
                <p>{lead.summary || lead.excerpt || "Tin thể thao đang được biên tập."}</p>
                <b>Đọc bài →</b>
              </div>
            </Link>
            <div className="secondary-leads">
              {secondary.map((item) => (
                <NewsTile item={item} key={item.id} />
              ))}
            </div>
            <div className="sports-list">
              {latest.slice(5, 18).map((item) => (
                <NewsRow item={item} key={item.id} />
              ))}
            </div>
          </section>
          <aside className="sports-rail">
            <div className="rail-card">
              <h3>Chủ đề thể thao</h3>
              {sportsTopics.slice(0, 10).map((topic) => (
                <Link href={`/chu-de/${topic.slug}`} key={topic.id}>
                  {topic.name}
                  <span>→</span>
                </Link>
              ))}
            </div>
          </aside>
        </div>

        {scored.length ? (
          <section className="sports-results">
            <div className="section-heading">
              <div>
                <span className="tag">02 · KẾT QUẢ</span>
                <h2>Tỷ số đáng chú ý</h2>
              </div>
            </div>
            <div className="score-grid">
              {scored.map((item) => (
                <Link className="score-tile" href={`/noi-dung/${item.id}`} key={item.id}>
                  <span>{item.source_name || "Thể thao"}</span>
                  <strong>{scorelineFrom(item)}</strong>
                  <b>{item.title}</b>
                </Link>
              ))}
            </div>
          </section>
        ) : null}
        <section className="sports-section">
          <div className="section-heading">
            <div>
              <span className="tag">03 · TOÀN CẢNH</span>
              <h2>Nhiều góc nhìn thể thao</h2>
            </div>
            <Link href="/danh-muc">Khám phá chuyên mục →</Link>
          </div>
          <div className="news-mosaic">
            {latest.slice(8, 26).map((item) => (
              <MosaicCard item={item} key={item.id} />
            ))}
          </div>
        </section>
        <section className="sports-section">
          <div className="section-heading">
            <div>
              <span className="tag">04 · THEO DÕI</span>
              <h2>Các mảng thể thao</h2>
            </div>
          </div>
          <div className="sport-pills">
            {sportsTopics.slice(0, 8).map((topic) => (
              <Link href={`/chu-de/${topic.slug}`} key={topic.id}>
                {topic.name}
                <span>→</span>
              </Link>
            ))}
          </div>
        </section>
      </main>
      <Footer />
    </>
  );
}

function NewsTile({ item }: { item: Item }) {
  return (
    <Link className="news-tile" href={`/noi-dung/${item.id}`}>
      {item.image_url ? (
        <img src={item.image_url} alt="" />
      ) : (
        <div className="tile-placeholder">BX</div>
      )}
      <div className="news-tile-copy">
        <span>{item.source_name || "BaoTheX"}</span>
        <StorySignals item={item} />
        <strong>{item.title}</strong>
        <small>{item.summary || item.excerpt || "Đọc nội dung đầy đủ."}</small>
      </div>
    </Link>
  );
}
function MosaicCard({ item }: { item: Item }) {
  return (
    <Link className="news-mosaic-card" href={`/noi-dung/${item.id}`}>
      {item.image_url ? (
        <img src={item.image_url} alt="" />
      ) : (
        <div className="mosaic-placeholder">BX</div>
      )}
      <div>
        <span>
          {item.source_name || "BaoTheX"} · {item.type === "video" ? "VIDEO" : "TIN MỚI"}
        </span>
        <StorySignals item={item} />
        <h3>{item.title}</h3>
        <p>{item.summary || item.excerpt || "Xem nội dung đầy đủ."}</p>
      </div>
    </Link>
  );
}
function NewsRow({ item, compact = false }: { item: Item; compact?: boolean }) {
  const score = scorelineFrom(item);
  return (
    <Link className={`sports-row ${compact ? "compact" : ""}`} href={`/noi-dung/${item.id}`}>
      {item.image_url ? (
        <img src={item.image_url} alt="" />
      ) : (
        <div className="row-placeholder">{item.type === "video" ? "▶" : "BX"}</div>
      )}
      <div>
        <span>
          {item.source_name || "BaoTheX"} · {item.type === "video" ? "VIDEO" : "TIN THỂ THAO"}
          {score ? ` · TỶ SỐ ${score}` : ""}
        </span>
        <StorySignals item={item} compact />
        <strong>{item.title}</strong>
        {!compact && (
          <small>{item.summary || item.excerpt || "Đọc bản tin đầy đủ tại BaoTheX."}</small>
        )}
      </div>
    </Link>
  );
}
function StorySignals({ item, compact = false }: { item: Item; compact?: boolean }) {
  const status = item.verification_status || "rumor";
  const labels = {
    rumor: "Tin đồn",
    verifying: "Đang xác minh",
    confirmed: "Đã xác nhận",
  } as const;
  return (
    <div className={`story-signals ${compact ? "compact" : ""}`}>
      {(item.cluster_source_count || 0) > 1 ? (
        <span className="signal multi-source">{item.cluster_source_count} nguồn</span>
      ) : null}
      <span className={`signal ${status}`}>{labels[status]}</span>
      {(item.source_quality || 0) >= 4 ? (
        <span className="signal trusted">Nguồn uy tín</span>
      ) : null}
    </div>
  );
}
function scorelineFrom(item: Item) {
  const text = [item.title, item.summary, item.excerpt].filter(Boolean).join(" ");
  const match = text.match(/\b(\d{1,2})\s*[-–:]\s*(\d{1,2})\b/);
  return match ? `${match[1]}-${match[2]}` : "";
}
