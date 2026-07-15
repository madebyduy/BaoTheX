import Link from "next/link";
import { api, demoItems, demoTopics, type Item, type Source, type Topic } from "./lib";
import { Footer } from "./ui";
import { DailyBriefPlayer } from "./daily-brief-player";

type HomeData = { today?: Item[]; sports?: Item[]; videos?: Item[] };

export default async function Home() {
  const home = await api<HomeData>("/home", {});
  const feed = await api<Item[]>(
    "/content?type=article&per_page=30&sort=recent",
    demoItems.filter((item) => item.type === "article"),
  );
  const [videoFeed, sources] = await Promise.all([
    api<Item[]>("/videos?per_page=50&sort=recent", []),
    api<Source[]>("/sources", []),
  ]);
  const fitnessSources = sources.filter(
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
  const latest = Array.from(
    new Map(
      [...(home.today || []), ...(home.sports || []), ...feed]
        .filter((item) => item.type === "article")
        .map((item) => [item.id, item]),
    ).values(),
  ).slice(0, 30);
  const topics = await api<Topic[]>("/topics", demoTopics);
  const sportsTopics = topics.filter((topic) =>
    /bong|tennis|the-thao|motor|esport|khac/.test(topic.slug),
  );
  const lead = latest[0] || demoItems[0];
  const secondary = latest.slice(1, 5);
  const scored = Array.from(
    new Map(
      latest
        .map((item) => ({ item, fixture: fixtureFrom(item) }))
        .filter((entry): entry is { item: Item; fixture: Fixture } => Boolean(entry.fixture))
        .map((entry) => [entry.fixture.key, entry]),
    ).values(),
  ).slice(0, 6);
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
            <DailyBriefPlayer />
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

        {scored.length ? (
          <section className="sports-results">
            <div className="section-heading">
              <div>
                <span className="tag">03 · KẾT QUẢ</span>
                <h2>Tỷ số đáng chú ý</h2>
              </div>
            </div>
            <div className="score-grid">
              {scored.map(({ item, fixture }) => (
                <Link className="score-tile" href={`/noi-dung/${item.id}`} key={fixture.key}>
                  <div className="score-head">
                    <span>Kết thúc</span>
                    <small>{item.source_name || "BaoTheX"}</small>
                  </div>
                  <div className="score-team">
                    <TeamMark team={fixture.home} />
                    <b>{fixture.home.name}</b>
                    <strong>{fixture.homeScore}</strong>
                  </div>
                  <div className="score-team">
                    <TeamMark team={fixture.away} />
                    <b>{fixture.away.name}</b>
                    <strong>{fixture.awayScore}</strong>
                  </div>
                  <div className="score-foot">
                    <span>{shortDate(item.published_at)}</span>
                    <b>Xem diễn biến →</b>
                  </div>
                </Link>
              ))}
            </div>
          </section>
        ) : null}
        <section className="sports-section">
          <div className="section-heading">
            <div>
              <span className="tag">04 · TOÀN CẢNH</span>
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
    <Link className="video-feature" href={`/noi-dung/${item.id}`}>
      <div className="video-feature-media">
        {item.image_url ? (
          <img src={item.image_url} alt="" />
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
    <Link className="video-desk-row" href={`/noi-dung/${item.id}`}>
      <div>
        {item.image_url ? <img src={item.image_url} alt="" /> : <span>▶</span>}
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
    <Link className="fitness-video-card" href={`/noi-dung/${item.id}`}>
      <div>
        {item.image_url ? <img src={item.image_url} alt="" /> : <span>▶</span>}
        <i>▶</i>
      </div>
      <small>{item.source_name || "Kênh thể hình"}</small>
      <strong>{item.title}</strong>
    </Link>
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
        <img src={`https://flagcdn.com/w80/${team.code}.png`} alt={`Cờ ${team.name}`} />
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
