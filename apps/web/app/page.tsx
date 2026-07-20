import Link from "next/link";
import { cookies } from "next/headers";
import {
  api,
  apiWithCookie,
  articleHref,
  demoItems,
  demoTopics,
  safeJsonLd,
  type Item,
  type SportsEvent,
  type Source,
  type Topic,
} from "./lib";
import { Footer, RemoteImage } from "./ui";
import { DailyBriefPlayer } from "./daily-brief-player";

type HomeData = { today?: Item[]; sports?: Item[]; videos?: Item[] };
type FeedPreferences = { feed_following_only?: boolean };

const SITE = process.env.NEXT_PUBLIC_SITE_URL || "https://baothex.vn";
const HOME_JSON_LD = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "WebSite",
      "@id": `${SITE}/#website`,
      url: SITE,
      name: "BaoTheX",
      inLanguage: "vi",
      description:
        "Tin thể thao nổi bật trong ngày, được tổng hợp, kiểm chứng nguồn và biên tập bằng tiếng Việt.",
      potentialAction: {
        "@type": "SearchAction",
        target: { "@type": "EntryPoint", urlTemplate: `${SITE}/tim-kiem?q={search_term_string}` },
        "query-input": "required name=search_term_string",
      },
    },
    {
      "@type": "Organization",
      "@id": `${SITE}/#organization`,
      name: "BaoTheX",
      url: SITE,
      logo: { "@type": "ImageObject", url: `${SITE}/icon.png` },
    },
  ],
};

export default async function Home() {
  const cookieHeader = (await cookies()).toString();
  const preferences = await apiWithCookie<FeedPreferences | null>(
    "/notifications/prefs",
    null,
    cookieHeader,
  );
  const isPersonalized = preferences !== null;
  const followingOnly = Boolean(preferences?.feed_following_only);
  // The ranked feed carries the personalization. Supporting newsroom blocks
  // stay public, and both requests run together to keep the homepage fast.
  const [personalizedFeed, followedFeed, home] = await Promise.all([
    isPersonalized
      ? apiWithCookie<Item[]>(
          `/feed?${followingOnly ? "strict=1&" : ""}per_page=50`,
          [],
          cookieHeader,
        )
      : Promise.resolve([] as Item[]),
    isPersonalized && !followingOnly
      ? apiWithCookie<Item[]>("/feed?strict=1&per_page=50", [], cookieHeader)
      : Promise.resolve([] as Item[]),
    followingOnly ? Promise.resolve({} as HomeData) : api<HomeData>("/home", {}),
  ]);
  const personalizedArticles = personalizedFeed.filter((item) => item.type === "article");
  const followedArticles = (followingOnly ? personalizedFeed : followedFeed).filter(
    (item) => item.type === "article",
  );
  // Fetch well past what the page shows, because uniqueStories collapses every
  // article sharing a story_cluster_id down to one. That ratio is brutal and it
  // gets worse the bigger the news is: 30 recent articles are 10 distinct
  // stories right now, because a World Cup semi-final arrives as one story told
  // by fifteen mastheads. The homepage asks for 30 stories and every section
  // from latest.slice(18) down was being handed an empty array — the front page
  // looked short of news while the database held 766 ready articles. 100 in
  // yields ~62 distinct stories out, which fills the page with room to spare.
  const publicArticles = followingOnly
    ? []
    : await api<Item[]>(
        "/content?type=article&per_page=100&sort=recent",
        demoItems.filter((item) => item.type === "article"),
      );
  // A personalized feed may temporarily contain only videos while newly
  // followed topics are still being translated. Keep the newsroom populated
  // with recent public articles unless the user explicitly selected strict mode.
  const preferredArticles = followedArticles.length ? followedArticles : personalizedArticles;
  const feed = followingOnly
    ? followedArticles
    : preferredArticles.length
      ? preferredArticles
      : publicArticles;
  const [videoFeed, sources, analyses] = await Promise.all([
    isPersonalized && personalizedFeed.length
      ? Promise.resolve(personalizedFeed.filter((item) => item.type === "video"))
      : api<Item[]>("/videos?per_page=50&sort=recent", []),
    api<Source[]>("/sources", []),
    followingOnly ? Promise.resolve([] as Item[]) : api<Item[]>("/analyses?limit=4", []),
  ]);
  const fitnessSources = (followingOnly ? [] : sources).filter(
    (source) =>
      source.kind === "youtube" &&
      /jeff nippard|renaissance|athlean|jeremy ethier|squat university|picturefit|hypertrophy/i.test(
        source.name,
      ),
  );
  const fitnessBatches = await Promise.all(
    fitnessSources.map((source) =>
      api<Item[]>(`/content?type=video&source=${source.id}&per_page=4&sort=recent`, []),
    ),
  );
  const fitnessVideos = diversifyVideos(uniqueItems(fitnessBatches.flat()), 8, 1);
  const generalVideos = diversifyVideos(uniqueItems(videoFeed), 9);
  const latest = uniqueStories(
    Array.from(
      new Map(
        [
          ...feed,
          ...(!followingOnly && !followedArticles.length
            ? [...(home.today || []), ...(home.sports || [])]
            : []),
        ]
          .filter((item) => item.type === "article")
          .map((item) => [item.id, item]),
      ).values(),
    ),
  ).slice(0, 30);
  const latestKeys = new Set(latest.map(storyKey));
  const truthAnalyses = analyses.filter((item) => !latestKeys.has(storyKey(item)));
  const topics = await api<Topic[]>("/topics", demoTopics);
  const sportsTopics = topics.filter(
    (topic) =>
      topic.category === "sport" ||
      /bong|tennis|cau-long|the-thao|the-hinh|motor|esport|khac/.test(topic.slug),
  );
  const today = new Date().toISOString().slice(0, 10);
  const events = await api<SportsEvent[]>(`/events?date=${today}&limit=6`, [], 20);
  const lead = latest[0];
  const secondary = latest.slice(1, 5);
  const dateLabel = new Intl.DateTimeFormat("vi-VN", {
    weekday: "long",
    day: "2-digit",
    month: "long",
    year: "numeric",
  }).format(new Date());

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: safeJsonLd(HOME_JSON_LD) }}
      />
      <main className="wrap sports-home">
        <section className="newsroom-masthead">
          <div className="issue-line">
            <span>{dateLabel}</span>
            <b>
              <i /> Tin mới cập nhật liên tục
            </b>
          </div>
          <div className="masthead-grid">
            <div className="masthead-copy">
              <span className="tag">Báo thể thao đa nguồn</span>
              <h1>
                Bản tin thể thao <em>24h</em>
              </h1>
              <p>
                Tin đáng chú ý, kết quả mới và những câu chuyện thể thao được tuyển chọn từ các
                nguồn uy tín.
              </p>
            </div>
            <div className="masthead-actions" aria-label="Tác vụ nhanh">
              <Link className="masthead-primary-action" href="#nghe-bao">
                <span>Nghe bản tin</span>
                <small>Audio 6h và 20h</small>
              </Link>
              <Link className="masthead-secondary-action" href="/bat-kip">
                Bắt kịp 3 phút
              </Link>
              <details className="topic-dropdown">
                <summary>Chọn chuyên mục</summary>
                <div>
                  <Link href="/danh-muc">Mới nhất</Link>
                  {sportsTopics.slice(0, 10).map((topic) => (
                    <Link href={`/chu-de/${topic.slug}`} key={topic.id}>
                      {topic.name}
                    </Link>
                  ))}
                </div>
              </details>
            </div>
          </div>
        </section>

        {isPersonalized ? (
          <section className={`personal-feed-mode ${followingOnly ? "strict" : "balanced"}`}>
            <div>
              <span>DÒNG TIN CỦA BẠN</span>
              <strong>
                {followingOnly
                  ? "Chỉ hiển thị các chủ đề đang theo dõi"
                  : "Đang ưu tiên sở thích và giữ một phần tin khám phá"}
              </strong>
            </div>
            <Link href="/cai-dat">Tùy chỉnh dòng tin →</Link>
          </section>
        ) : null}

        <div className="sports-grid">
          <section className="sports-main">
            <div className="section-heading">
              <div>
                <span className="tag">01 · TIN NÓNG</span>
                <h2>Đáng chú ý trong ngày</h2>
              </div>
              <Link href="/danh-muc">Xem tất cả →</Link>
            </div>
            {lead ? (
              <Link className="sports-lead" href={articleHref(lead)}>
                {lead.image_url ? (
                  <RemoteImage
                    src={lead.image_url}
                    alt=""
                    fetchPriority="high"
                    decoding="async"
                    referrerPolicy="no-referrer"
                  />
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
            ) : followingOnly ? (
              <div className="personal-feed-empty">
                <span>Chưa có bài phù hợp</span>
                <h3>Chọn ít nhất một chủ đề để tạo dòng tin riêng.</h3>
                <p>
                  Bạn có thể theo dõi Thể hình, Thể thao điện tử, Bóng rổ hoặc bất kỳ môn nào mình
                  quan tâm.
                </p>
                <Link className="btn ember" href="/cai-dat">
                  Chọn chủ đề theo dõi
                </Link>
              </div>
            ) : null}
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
          <aside className="sports-rail" id="nghe-bao">
            <DailyBriefPlayer />
            <div className="rail-separator">
              <span>KHÁM PHÁ THEO MÔN</span>
            </div>
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

        {generalVideos.length ? (
          <section className="home-video-zone">
            <div className="section-heading">
              <div>
                <span className="tag">02 · VIDEO</span>
                <h2>Video thể thao chọn lọc</h2>
              </div>
              <Link href="/video">Xem thư viện video →</Link>
            </div>
            <div className="video-desk">
              <VideoFeature item={generalVideos[0]} />
              <div className="video-desk-list">
                {generalVideos.slice(1, 5).map((item) => (
                  <VideoRow item={item} key={item.id} />
                ))}
              </div>
            </div>
            {fitnessVideos.length ? (
              <div className="fitness-video-block">
                <div className="video-subheading">
                  <div>
                    <span>THỂ HÌNH & TẬP LUYỆN</span>
                    <h3>Kỹ thuật, dinh dưỡng và phát triển cơ bắp</h3>
                  </div>
                  <Link href="/video">Xem thêm →</Link>
                </div>
                <div className="fitness-video-grid">
                  {fitnessVideos.slice(0, 4).map((item) => (
                    <VideoCard item={item} key={item.id} />
                  ))}
                </div>
              </div>
            ) : null}
          </section>
        ) : null}

        {events.length ? (
          <section className="sports-results">
            <div className="section-heading">
              <div>
                <span className="tag">03 · EVENT HUB</span>
                <h2>Lịch & kết quả có nguồn</h2>
              </div>
              <Link href="/lich-the-thao">Xem lịch đầy đủ →</Link>
            </div>
            <div className="score-grid">
              {events.map((event) => (
                <Link className="score-tile" href={`/tran-dau/${event.id}`} key={event.id}>
                  <div className="score-head">
                    <span>
                      {event.status === "live"
                        ? "Đang diễn ra"
                        : event.status === "finished"
                          ? "Kết thúc"
                          : "Lịch dự kiến"}
                    </span>
                    <small>{event.is_manual ? "BaoTheX cập nhật" : event.data_source}</small>
                  </div>
                  <div className="score-team">
                    <span className="team-mark">
                      {(event.home_name || event.title).slice(0, 2).toUpperCase()}
                    </span>
                    <b>{event.home_name || event.title}</b>
                    <strong>{event.home_score ?? ""}</strong>
                  </div>
                  {event.away_name ? (
                    <div className="score-team">
                      <span className="team-mark">{event.away_name.slice(0, 2).toUpperCase()}</span>
                      <b>{event.away_name}</b>
                      <strong>{event.away_score ?? ""}</strong>
                    </div>
                  ) : null}
                  <div className="score-foot">
                    <span>{shortDate(event.starts_at)}</span>
                    <b>{event.freshness === "delayed" ? "Cập nhật chậm · " : ""}Chi tiết →</b>
                  </div>
                </Link>
              ))}
            </div>
          </section>
        ) : null}
        {truthAnalyses.length ? (
          <section className="sports-section analysis-home-section">
            <div className="section-heading">
              <div>
                <span className="tag">GÓC NHÌN BAOTHEX</span>
                <h2>Một sự kiện, nhiều nguồn</h2>
              </div>
              <Link href="/goc-nhin">Xem toàn bộ phân tích →</Link>
            </div>
            <div className="analysis-home-grid">
              {truthAnalyses.map((item) => (
                <MosaicCard item={item} key={item.id} />
              ))}
            </div>
          </section>
        ) : null}
        {/* Guarded like every other block on this page. A fixed slice into a
            variable-length list is a promise the data cannot always keep, and
            when it came up empty this section still rendered its heading, its
            "Khám phá chuyên mục" link and an empty grid — 188px of furniture
            announcing news that was not there. A section with nothing in it
            should not be on the page at all. */}
        {latest.length > 18 ? (
          <section className="sports-section">
            <div className="section-heading">
              <div>
                <span className="tag">04 · TOÀN CẢNH</span>
                <h2>Nhiều góc nhìn thể thao</h2>
              </div>
              <Link href="/danh-muc">Khám phá chuyên mục →</Link>
            </div>
            <div className="news-mosaic">
              {latest.slice(18, 30).map((item) => (
                <MosaicCard item={item} key={item.id} />
              ))}
            </div>
          </section>
        ) : null}
        <section className="sports-section">
          <div className="section-heading">
            <div>
              <span className="tag">05 · THEO DÕI</span>
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

function uniqueItems(items: Item[]) {
  return Array.from(new Map(items.map((item) => [item.id, item])).values());
}

function uniqueStories(items: Item[]) {
  return Array.from(new Map(items.map((item) => [storyKey(item), item])).values());
}

function storyKey(item: Item) {
  return item.story_cluster_id ? `cluster:${item.story_cluster_id}` : `content:${item.id}`;
}

function diversifyVideos(items: Item[], limit: number, maxPerSource = 2) {
  const selected: Item[] = [];
  const sourceCounts = new Map<string, number>();

  for (const item of items) {
    const source = item.source_name || "Nguồn chưa xác định";
    const count = sourceCounts.get(source) || 0;
    if (count >= maxPerSource) continue;
    selected.push(item);
    sourceCounts.set(source, count + 1);
    if (selected.length === limit) return selected;
  }

  for (const item of items) {
    if (selected.some((candidate) => candidate.id === item.id)) continue;
    selected.push(item);
    if (selected.length === limit) break;
  }

  return selected;
}

function VideoFeature({ item }: { item: Item }) {
  return (
    <Link className="video-feature" href={articleHref(item)}>
      <div className="video-feature-media">
        {item.image_url ? (
          <RemoteImage
            src={item.image_url}
            alt=""
            loading="lazy"
            decoding="async"
            referrerPolicy="no-referrer"
          />
        ) : (
          <div className="video-placeholder">▶</div>
        )}
        <span className="video-play">▶</span>
      </div>
      <div className="video-feature-copy">
        <span>{item.source_name || "Kênh thể thao"}</span>
        <h3>{item.title}</h3>
        <p>{item.summary || item.excerpt || "Xem video từ kênh chính thức."}</p>
        <b>Xem video →</b>
      </div>
    </Link>
  );
}

function VideoRow({ item }: { item: Item }) {
  return (
    <Link className="video-desk-row" href={articleHref(item)}>
      <div>
        {item.image_url ? (
          <RemoteImage
            src={item.image_url}
            alt=""
            loading="lazy"
            decoding="async"
            referrerPolicy="no-referrer"
          />
        ) : (
          <span>▶</span>
        )}
        <i>▶</i>
      </div>
      <section>
        <small>{item.source_name || "YouTube"}</small>
        <strong>{item.title}</strong>
      </section>
    </Link>
  );
}

function VideoCard({ item }: { item: Item }) {
  return (
    <Link className="fitness-video-card" href={articleHref(item)}>
      <div>
        {item.image_url ? (
          <RemoteImage
            src={item.image_url}
            alt=""
            loading="lazy"
            decoding="async"
            referrerPolicy="no-referrer"
          />
        ) : (
          <span>▶</span>
        )}
        <i>▶</i>
      </div>
      <small>{item.source_name || "Kênh thể hình"}</small>
      <strong>{item.title}</strong>
    </Link>
  );
}

function NewsTile({ item }: { item: Item }) {
  return (
    <Link className="news-tile" href={articleHref(item)}>
      {item.image_url ? (
        <RemoteImage
          src={item.image_url}
          alt=""
          loading="lazy"
          decoding="async"
          referrerPolicy="no-referrer"
        />
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
    <Link className="news-mosaic-card" href={articleHref(item)}>
      {item.image_url ? (
        <RemoteImage
          src={item.image_url}
          alt=""
          loading="lazy"
          decoding="async"
          referrerPolicy="no-referrer"
        />
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
    <Link className={`sports-row ${compact ? "compact" : ""}`} href={articleHref(item)}>
      {item.image_url ? (
        <RemoteImage
          src={item.image_url}
          alt=""
          loading="lazy"
          decoding="async"
          referrerPolicy="no-referrer"
        />
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
  if (item.type !== "article") return "";
  const text = item.title;
  const match = text.match(/\b(\d{1,2})\s*[-–:]\s*(\d{1,2})\b/);
  return match ? `${match[1]}-${match[2]}` : "";
}

type Team = { name: string; aliases: string[]; mark: string; code?: string; kind?: "club" };
type Fixture = {
  key: string;
  home: Team;
  away: Team;
  homeScore: string;
  awayScore: string;
};

const teams: Team[] = [
  { name: "Tây Ban Nha", aliases: ["tây ban nha", "spain"], mark: "ES", code: "es" },
  { name: "Việt Nam", aliases: ["việt nam", "vietnam"], mark: "VN", code: "vn" },
  { name: "Hàn Quốc", aliases: ["hàn quốc", "south korea", "korea"], mark: "KR", code: "kr" },
  { name: "Argentina", aliases: ["argentina"], mark: "AR", code: "ar" },
  { name: "Bồ Đào Nha", aliases: ["bồ đào nha", "portugal"], mark: "PT", code: "pt" },
  { name: "Hà Lan", aliases: ["hà lan", "netherlands"], mark: "NL", code: "nl" },
  { name: "Ma-rốc", aliases: ["ma-rốc", "morocco"], mark: "MA", code: "ma" },
  { name: "Nhật Bản", aliases: ["nhật bản", "japan"], mark: "JP", code: "jp" },
  { name: "Hoa Kỳ", aliases: ["hoa kỳ", "mỹ", "united states", "usa"], mark: "US", code: "us" },
  { name: "Pháp", aliases: ["pháp", "france"], mark: "FR", code: "fr" },
  { name: "Anh", aliases: ["đội tuyển anh", "england"], mark: "GB", code: "gb-eng" },
  { name: "Brazil", aliases: ["brazil", "brasil"], mark: "BR", code: "br" },
  { name: "Đức", aliases: ["đội tuyển đức", "germany"], mark: "DE", code: "de" },
  { name: "Italy", aliases: ["italy", "ý"], mark: "IT", code: "it" },
  { name: "Real Madrid", aliases: ["real madrid"], mark: "RM", kind: "club" },
  { name: "Barcelona", aliases: ["barcelona", "barca"], mark: "FCB", kind: "club" },
  { name: "Man City", aliases: ["man city", "manchester city"], mark: "MCI", kind: "club" },
  {
    name: "Man Utd",
    aliases: ["man utd", "man united", "manchester united"],
    mark: "MUN",
    kind: "club",
  },
  { name: "Liverpool", aliases: ["liverpool"], mark: "LIV", kind: "club" },
  { name: "Arsenal", aliases: ["arsenal"], mark: "ARS", kind: "club" },
  { name: "Chelsea", aliases: ["chelsea"], mark: "CHE", kind: "club" },
  { name: "Bayern Munich", aliases: ["bayern munich", "bayern"], mark: "FCB", kind: "club" },
  { name: "PSG", aliases: ["paris saint-germain", "psg"], mark: "PSG", kind: "club" },
  { name: "Inter Milan", aliases: ["inter milan", "inter"], mark: "INT", kind: "club" },
  { name: "AC Milan", aliases: ["ac milan"], mark: "MIL", kind: "club" },
  { name: "Juventus", aliases: ["juventus"], mark: "JUV", kind: "club" },
  { name: "Dortmund", aliases: ["dortmund"], mark: "BVB", kind: "club" },
  { name: "Tottenham", aliases: ["tottenham", "spurs"], mark: "TOT", kind: "club" },
];

function fixtureFrom(item: Item): Fixture | null {
  const text = [item.title, item.summary, item.excerpt]
    .filter(Boolean)
    .join(" ")
    .toLocaleLowerCase("vi");
  const score = /\b(\d{1,2})\s*[-–:]\s*(\d{1,2})\b/.exec(text);
  if (!score || score.index < 0) return null;

  const found = teams
    .map((team) => {
      const positions = team.aliases
        .map((alias) => text.indexOf(alias))
        .filter((index) => index >= 0);
      return positions.length ? { team, index: Math.min(...positions) } : null;
    })
    .filter((entry): entry is { team: Team; index: number } => Boolean(entry))
    .sort((a, b) => a.index - b.index);
  if (found.length < 2) return null;

  const beforeScore = found.filter((entry) => entry.index < score.index);
  const pair = beforeScore.length >= 2 ? beforeScore.slice(-2) : found.slice(0, 2);
  if (pair[0].team.name === pair[1].team.name) return null;

  const homeScore = score[1];
  const awayScore = score[2];
  const canonical = [
    { name: pair[0].team.name, score: homeScore },
    { name: pair[1].team.name, score: awayScore },
  ].sort((a, b) => a.name.localeCompare(b.name, "vi"));
  return {
    key: `${canonical[0].name}-${canonical[0].score}-${canonical[1].name}-${canonical[1].score}`,
    home: pair[0].team,
    away: pair[1].team,
    homeScore,
    awayScore,
  };
}

function TeamMark({ team }: { team: Team }) {
  return (
    <i className={`team-mark ${team.kind || "country"}`}>
      {team.code ? (
        <RemoteImage src={`https://flagcdn.com/w80/${team.code}.png`} alt={`Cờ ${team.name}`} />
      ) : (
        team.mark
      )}
    </i>
  );
}

function shortDate(value?: string) {
  if (!value) return "Mới cập nhật";
  return new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit" }).format(
    new Date(value),
  );
}
