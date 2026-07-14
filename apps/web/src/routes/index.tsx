import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/")({
  component: Index,
});

type Kind = "ARTICLE" | "RESEARCH" | "VIDEO" | "PODCAST" | "EVENT";

const kindStyle: Record<Kind, string> = {
  ARTICLE: "bg-ink text-paper",
  RESEARCH: "bg-lime text-ink",
  VIDEO: "bg-ember text-ember-foreground",
  PODCAST: "bg-paper text-ink border border-ink",
  EVENT: "bg-ink text-lime",
};

const kindGlyph: Record<Kind, string> = {
  ARTICLE: "§",
  RESEARCH: "∎",
  VIDEO: "▶",
  PODCAST: "◉",
  EVENT: "✦",
};

// Nhãn loại nội dung hiển thị bằng tiếng Việt.
const kindLabel: Record<Kind, string> = {
  ARTICLE: "BÀI VIẾT",
  RESEARCH: "NGHIÊN CỨU",
  VIDEO: "VIDEO",
  PODCAST: "PODCAST",
  EVENT: "SỰ KIỆN",
};

const topics = [
  { name: "Phì đại cơ", count: 128 },
  { name: "Powerlifting", count: 74 },
  { name: "Dinh dưỡng", count: 96 },
  { name: "Creatine", count: 41 },
  { name: "Phục hồi", count: 52 },
  { name: "Phòng chấn thương", count: 33 },
  { name: "Cardio", count: 28 },
  { name: "Giấc ngủ", count: 19 },
];

const feed: Array<{
  kind: Kind;
  title: string;
  source: string;
  author?: string;
  time: string;
  read: string;
  excerpt: string;
  tags: string[];
  points?: string[];
  metric?: string;
}> = [
  {
    kind: "RESEARCH",
    title:
      "Tăng volume tập giúp cơ phát triển tới ~20 set mỗi nhóm cơ mỗi tuần — phân tích gộp cho thấy lợi ích giảm dần sau ngưỡng này.",
    source: "Journal of Strength & Conditioning · Phân tích gộp",
    author: "Schoenfeld và cộng sự",
    time: "2 giờ trước",
    read: "6 phút đọc",
    excerpt:
      "Dữ liệu gộp từ 34 thử nghiệm đối chứng ngẫu nhiên cho thấy đường cong liều lượng đi ngang khi vượt khoảng 20 set nặng mỗi nhóm cơ mỗi tuần với người tập lâu năm — và phần mệt mỏi tăng thêm còn có thể ăn vào hiệu suất buổi kế tiếp.",
    tags: ["Phì đại cơ", "Volume", "Phân tích gộp"],
    metric: "n = 1.842",
    points: [
      "Hiệu quả đạt đỉnh quanh 16–20 set/nhóm cơ/tuần",
      "Vượt 22 set, chi phí phục hồi lớn hơn phần cơ tăng thêm",
      "Tần suất (2 buổi/tuần) vẫn tốt hơn 1 buổi/tuần khi cùng volume",
    ],
  },
  {
    kind: "VIDEO",
    title: "Jeff Nippard phân tích nghiên cứu điện cơ (EMG) mới về bài đẩy ngực",
    source: "Jeff Nippard · YouTube",
    time: "5 giờ trước",
    read: "18 phút xem",
    excerpt:
      "Đi qua các biến số về độ rộng nắm, độ ưỡn lưng và nhịp độ, cùng ý nghĩa thực sự của các con số EMG đối với sự phát triển cơ ngực.",
    tags: ["Đẩy ngực", "EMG", "Kỹ thuật"],
  },
  {
    kind: "ARTICLE",
    title: "Vì sao lầm tưởng 'gây sốc cho cơ' vẫn chưa chết — mổ xẻ bằng ngôn ngữ dễ hiểu",
    source: "Stronger by Science",
    author: "Greg Nuckols",
    time: "8 giờ trước",
    read: "9 phút đọc",
    excerpt:
      "Quan niệm rằng cơ cần bị 'bất ngờ' liên tục để phát triển cứ trồi lên mãi. Đây là điều thực sự thúc đẩy thích nghi và nguồn gốc của lầm tưởng này.",
    tags: ["Lập trình tập", "Lầm tưởng"],
  },
  {
    kind: "PODCAST",
    title: "TS. Layne Norton bàn về thời điểm nạp protein, ngưỡng leucine và 'cửa sổ đồng hóa'",
    source: "Iron Culture · Tập 241",
    time: "1 ngày trước",
    read: "1 giờ 42 phút",
    excerpt:
      "Cuộc trò chuyện dài về cách phân bổ protein trong ngày, casein trước khi ngủ có quan trọng không, và tình trạng nghiên cứu về 'cửa sổ đồng hóa' năm 2026.",
    tags: ["Protein", "Dinh dưỡng"],
  },
  {
    kind: "RESEARCH",
    title: "Creatine monohydrate 5 g/ngày cho thấy lợi ích nhận thức ở người thiếu ngủ",
    source: "EuropePMC · Thử nghiệm đối chứng ngẫu nhiên",
    author: "Gordji-Nejad và cộng sự",
    time: "1 ngày trước",
    read: "4 phút đọc",
    excerpt:
      "Thử nghiệm mù đôi (n=15) ghi nhận cải thiện trí nhớ làm việc và tốc độ xử lý sau 24 giờ thiếu ngủ, với một liều duy nhất 0,35 g/kg.",
    tags: ["Creatine", "Nhận thức", "RCT"],
    metric: "n = 15",
  },
  {
    kind: "ARTICLE",
    title: "Các vận động viên powerlifting đỉnh cao thực sự sắp xếp giai đoạn peak thế nào",
    source: "Barbell Medicine",
    author: "Austin Baraki",
    time: "2 ngày trước",
    read: "12 phút đọc",
    excerpt:
      "Phân tích các chiến lược taper dùng ở ba kỳ IPF World Classic gần nhất và điều mà người tập nghiệp dư có thể học theo.",
    tags: ["Powerlifting", "Peak"],
  },
  {
    kind: "EVENT",
    title: "Giải IPF World Classic 2026 — cổng đăng ký mở vào thứ Sáu",
    source: "IPF · Thông báo",
    time: "3 ngày trước",
    read: "2 phút đọc",
    excerpt: "Sundsvall, Thụy Điển. Tổng mức đạt chuẩn được cập nhật cho hạng -74 kg và -83 kg.",
    tags: ["Powerlifting", "Thi đấu"],
  },
];

const followedSources = [
  { name: "Stronger by Science", kind: "Trang tin" },
  { name: "Jeff Nippard", kind: "YouTube" },
  { name: "Iron Culture", kind: "Podcast" },
  { name: "Barbell Medicine", kind: "Trang tin" },
  { name: "EuropePMC · sức mạnh", kind: "Nghiên cứu" },
];

const tickerItems = [
  "PHÂN TÍCH GỘP · Volume đi ngang ở 20 set/tuần",
  "RCT · Creatine và tình trạng thiếu ngủ",
  "VIDEO · Nippard nói về EMG đẩy ngực",
  "PODCAST · Layne Norton — thời điểm nạp protein",
  "SỰ KIỆN · IPF World Classic — Sundsvall",
  "NGHIÊN CỨU · Ashwagandha và testosterone nhìn lại",
  "BÀI VIẾT · Giai đoạn peak của VĐV powerlifting đỉnh cao",
];

function SectionHeader({ n, kicker, title }: { n: string; kicker: string; title: string }) {
  return (
    <div className="mb-5 flex items-end justify-between rule-double-b pb-2">
      <div className="flex items-baseline gap-3">
        <span className="font-mono text-[11px] text-ember">{n}</span>
        <span className="label-eyebrow text-muted-foreground">{kicker}</span>
        <span className="font-display text-xl italic tracking-tight">{title}</span>
      </div>
      <span className="font-mono text-[10px] text-muted-foreground">// cập nhật 2 phút trước</span>
    </div>
  );
}

function Index() {
  return (
    <div className="min-h-screen text-foreground">
      {/* Thanh trên cùng */}
      <header className="sticky top-0 z-30 bg-background/90 backdrop-blur-md rule-b">
        <div className="mx-auto flex max-w-[1400px] items-center gap-6 px-6 py-3">
          <a href="/" className="flex items-baseline gap-2">
            <span className="font-display text-3xl font-black tracking-[-0.02em]">
              Bao<span className="italic text-ember">TheX</span>
            </span>
            <span className="label-eyebrow text-muted-foreground">/ TỪ 2026</span>
          </a>
          <nav className="ml-8 hidden items-center gap-6 md:flex">
            {["Dòng tin", "Nghiên cứu", "Video", "Podcast", "Nhân vật", "Nguồn"].map((n, i) => (
              <a
                key={n}
                href="#"
                className={`label-eyebrow transition relative ${
                  i === 0
                    ? "text-foreground after:absolute after:-bottom-1 after:left-0 after:h-[2px] after:w-full after:bg-ember"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {n}
              </a>
            ))}
          </nav>
          <div className="ml-auto flex items-center gap-3">
            <div className="hidden w-72 items-center gap-2 rounded-sm border border-input bg-card px-3 py-1.5 text-sm md:flex">
              <span className="font-mono text-muted-foreground">⌘K</span>
              <input
                placeholder="Tìm chủ đề, người, nghiên cứu…"
                className="w-full bg-transparent outline-none placeholder:text-muted-foreground"
              />
            </div>
            <button className="label-eyebrow rounded-sm bg-ember px-3 py-2 text-ember-foreground shadow-[3px_3px_0_var(--color-ink)] transition hover:-translate-x-[1px] hover:-translate-y-[1px] hover:shadow-[4px_4px_0_var(--color-ink)]">
              Kết nối Telegram
            </button>
          </div>
        </div>

        {/* Dải tin chạy */}
        <div className="rule-t rule-b bg-ink text-paper overflow-hidden">
          <div className="mx-auto flex max-w-[1400px] items-center gap-4 px-6 py-2 font-mono text-[11px] uppercase tracking-widest">
            <span className="shrink-0 rounded-sm bg-lime px-1.5 py-0.5 text-ink">● TRỰC TIẾP</span>
            <span className="shrink-0 text-paper/70">THỨ TƯ · 15.07.26</span>
            <span className="shrink-0 text-paper/40">/</span>
            <div className="relative flex-1 overflow-hidden">
              <div className="marquee-track">
                {[...tickerItems, ...tickerItems].map((t, i) => (
                  <span key={i} className="shrink-0 text-paper/85">
                    <span className="text-ember">✦</span> {t}
                  </span>
                ))}
              </div>
            </div>
            <span className="shrink-0 text-paper/60">Bản tin · 07:00 sáng</span>
          </div>
        </div>
      </header>

      {/* Dải măng-sét */}
      <div className="rule-b bg-cream">
        <div className="mx-auto flex max-w-[1400px] flex-wrap items-center justify-between gap-4 px-6 py-3">
          <div className="flex items-center gap-4 font-mono text-[11px] uppercase tracking-widest text-muted-foreground">
            <span>Số IV</span>
            <span className="text-ink/30">·</span>
            <span>Kỳ 128</span>
            <span className="text-ink/30">·</span>
            <span className="text-ink">Thứ Tư, 15 tháng 7, 2026</span>
          </div>
          <div className="flex items-center gap-5 font-mono text-[11px] uppercase tracking-widest">
            <span><b className="text-ember">47</b> mới</span>
            <span><b>12</b> nghiên cứu</span>
            <span><b>18</b> video</span>
            <span><b>9</b> podcast</span>
            <span className="text-muted-foreground">3.2k lượt đọc · 24 giờ qua</span>
          </div>
        </div>
      </div>

      {/* Dải bộ lọc */}
      <div className="rule-b">
        <div className="mx-auto flex max-w-[1400px] items-center gap-3 overflow-x-auto px-6 py-3">
          <span className="label-eyebrow text-muted-foreground">LỌC —</span>
          {(["Tất cả", "Bài viết", "Nghiên cứu", "Video", "Podcast", "Sự kiện"] as const).map((f, i) => (
            <button
              key={f}
              className={`label-eyebrow shrink-0 rounded-sm px-3 py-1.5 transition ${
                i === 0
                  ? "bg-ink text-paper shadow-[2px_2px_0_var(--color-ember)]"
                  : "border border-rule text-muted-foreground hover:border-ink hover:text-foreground"
              }`}
            >
              {f}
            </button>
          ))}
          <span className="mx-3 h-5 w-px bg-rule" />
          <span className="label-eyebrow text-muted-foreground">SẮP XẾP —</span>
          <button className="label-eyebrow shrink-0 rounded-sm border border-ink px-3 py-1.5">
            Mới nhất ↓
          </button>
          <button className="label-eyebrow shrink-0 rounded-sm px-3 py-1.5 text-muted-foreground hover:text-foreground">
            Nổi bật hôm nay
          </button>
          <button className="label-eyebrow shrink-0 rounded-sm px-3 py-1.5 text-muted-foreground hover:text-foreground">
            Được trích nhiều
          </button>
          <span className="ml-auto label-eyebrow text-muted-foreground hidden md:inline">
            ⌥ ⇧ D — nền tối
          </span>
        </div>
      </div>

      {/* Lưới nội dung */}
      <main className="mx-auto grid max-w-[1400px] grid-cols-12 gap-8 px-6 py-10">
        {/* Cột trái */}
        <aside className="col-span-12 lg:col-span-2">
          <div className="sticky top-40 space-y-8">
            <div>
              <div className="mb-3 flex items-baseline justify-between rule-b pb-2">
                <span className="label-eyebrow">Chủ đề</span>
                <span className="font-mono text-[10px] text-muted-foreground">8</span>
              </div>
              <ul className="space-y-0">
                {topics.map((t, i) => (
                  <li key={t.name}>
                    <a href="#" className="group flex items-center justify-between py-2 rule-b">
                      <span className="flex items-baseline gap-2">
                        <span className="font-mono text-[10px] text-muted-foreground">
                          {String(i + 1).padStart(2, "0")}
                        </span>
                        <span className="font-heading text-[13px] font-semibold group-hover:text-ember">
                          {t.name}
                        </span>
                      </span>
                      <span className="font-mono text-[11px] text-muted-foreground group-hover:text-ember">
                        {t.count}
                      </span>
                    </a>
                  </li>
                ))}
              </ul>
              <button className="label-eyebrow mt-4 text-ember hover:underline">
                + Theo dõi chủ đề
              </button>
            </div>

            {/* Thẻ chỉ số trang trí */}
            <div className="grain relative overflow-hidden rounded-sm border border-ink bg-cream p-4">
              <div className="grain-overlay" />
              <div className="label-eyebrow text-ember">Chỉ số</div>
              <div className="mt-2 font-display text-4xl font-black leading-none tracking-tight">
                74<span className="text-ember">.</span>2
              </div>
              <div className="mt-1 font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
                Tỷ lệ tín hiệu / nhiễu
              </div>
              <div className="mt-3 h-1 w-full bg-ink/10">
                <div className="h-1 w-[74%] bg-ember" />
              </div>
            </div>
          </div>
        </aside>

        {/* Giữa — Dòng tin */}
        <section className="col-span-12 lg:col-span-7">
          <SectionHeader n="§ 01" kicker="Dòng tin" title="tin chọn hôm nay" />

          {/* Tin nổi bật */}
          <article className="group grain relative overflow-hidden rule-b pb-8">
            <div className="flex flex-wrap items-center gap-3">
              <span className={`label-eyebrow rounded-sm px-2 py-1 ${kindStyle[feed[0].kind]}`}>
                <span className="mr-1">{kindGlyph[feed[0].kind]}</span>
                {kindLabel[feed[0].kind]}
              </span>
              <span className="font-mono text-[11px] uppercase tracking-widest text-muted-foreground">
                {feed[0].source}
              </span>
              {feed[0].metric && (
                <span className="label-eyebrow rounded-sm bg-secondary px-2 py-1 text-secondary-foreground">
                  {feed[0].metric}
                </span>
              )}
              <span className="ml-auto font-mono text-[11px] text-muted-foreground">
                {feed[0].time} · {feed[0].read}
              </span>
            </div>
            <h2 className="mt-5 font-display text-[2.6rem] font-black leading-[1.05] tracking-[-0.02em]">
              {feed[0].title.split(" — ")[0]}
              <span className="italic font-normal text-ember"> — {feed[0].title.split(" — ")[1]}</span>
            </h2>
            <p className="drop-cap mt-5 max-w-3xl font-sans text-[16px] leading-[1.65] text-foreground/90">
              {feed[0].excerpt}
            </p>
            {feed[0].points && (
              <ul className="mt-5 grid gap-2 md:grid-cols-3">
                {feed[0].points.map((p, i) => (
                  <li
                    key={p}
                    className="relative border-l-2 border-lime bg-secondary/60 px-3 py-2.5 text-[13px] leading-snug"
                  >
                    <span className="mr-1 font-mono text-[10px] text-ember">
                      0{i + 1}·
                    </span>
                    {p}
                  </li>
                ))}
              </ul>
            )}
            <div className="mt-5 flex flex-wrap items-center gap-2">
              {feed[0].tags.map((t) => (
                <span
                  key={t}
                  className="label-eyebrow rounded-sm border border-rule px-2 py-1 text-muted-foreground hover:border-ink hover:text-foreground"
                >
                  #{t}
                </span>
              ))}
              <div className="ml-auto flex items-center gap-4">
                <button className="label-eyebrow text-muted-foreground hover:text-ember">
                  ↗ Lưu
                </button>
                <button className="label-eyebrow text-muted-foreground hover:text-ember">
                  ✦ Tóm tắt
                </button>
                <button className="label-eyebrow text-muted-foreground hover:text-ember">
                  ⎘ Nguồn
                </button>
              </div>
            </div>
          </article>

          {/* Vạch ngăn trang trí */}
          <div className="my-8 flex items-center justify-center gap-4 text-ember">
            <span className="h-px w-full bg-rule" />
            <span className="font-display text-lg">✦ ✦ ✦</span>
            <span className="h-px w-full bg-rule" />
          </div>

          <SectionHeader n="§ 02" kicker="Luồng" title="thêm từ dòng tin" />

          {/* Danh sách dày */}
          <ul className="divide-y divide-rule">
            {feed.slice(1).map((item, idx) => (
              <li key={item.title} className="group grid grid-cols-12 gap-4 py-6 transition hover:bg-cream/60">
                <div className="col-span-1 font-mono text-xs text-muted-foreground">
                  <span className="text-ember">§</span>
                  <br />
                  {String(idx + 2).padStart(2, "0")}
                </div>
                <div className="col-span-11">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className={`label-eyebrow rounded-sm px-2 py-1 ${kindStyle[item.kind]}`}>
                      <span className="mr-1">{kindGlyph[item.kind]}</span>
                      {kindLabel[item.kind]}
                    </span>
                    <span className="font-mono text-[11px] uppercase tracking-widest text-muted-foreground">
                      {item.source}
                    </span>
                    {item.author && (
                      <span className="font-display italic text-[13px] text-foreground/70">
                        — {item.author}
                      </span>
                    )}
                    {item.metric && (
                      <span className="label-eyebrow rounded-sm bg-lime px-2 py-1 text-ink">
                        {item.metric}
                      </span>
                    )}
                    <span className="ml-auto font-mono text-[11px] text-muted-foreground">
                      {item.time} · {item.read}
                    </span>
                  </div>
                  <h3 className="mt-3 font-display text-[1.55rem] font-bold leading-[1.2] tracking-[-0.01em] group-hover:text-ember">
                    {item.title}
                  </h3>
                  <p className="mt-2 max-w-2xl text-[14.5px] leading-relaxed text-foreground/80">
                    {item.excerpt}
                  </p>
                  <div className="mt-3 flex flex-wrap items-center gap-2">
                    {item.tags.map((t) => (
                      <span
                        key={t}
                        className="label-eyebrow rounded-sm border border-rule px-2 py-1 text-muted-foreground"
                      >
                        #{t}
                      </span>
                    ))}
                    <div className="ml-auto flex items-center gap-3 text-muted-foreground">
                      <button className="label-eyebrow hover:text-ember">Lưu</button>
                      <button className="label-eyebrow hover:text-ember">Tóm tắt</button>
                    </div>
                  </div>
                </div>
              </li>
            ))}
          </ul>

          <div className="mt-10 flex items-center justify-between rule-t pt-6">
            <span className="font-mono text-xs text-muted-foreground">
              Hiển thị 7 / 47 · làm mới 2 phút trước
            </span>
            <button className="label-eyebrow rounded-sm border border-ink bg-paper px-5 py-2.5 shadow-[3px_3px_0_var(--color-ember)] transition hover:-translate-x-[1px] hover:-translate-y-[1px] hover:shadow-[4px_4px_0_var(--color-ember)]">
              Tải thêm 20 →
            </button>
          </div>
        </section>

        {/* Cột phải */}
        <aside className="col-span-12 space-y-8 lg:col-span-3">
          {/* Thẻ bản tin */}
          <div className="relative overflow-hidden rounded-sm bg-ink p-5 text-paper shadow-[6px_6px_0_var(--color-ember)]">
            <div className="absolute -right-6 -top-6 h-24 w-24 rounded-full bg-ember/20 blur-2xl" />
            <div className="flex items-center gap-2">
              <span className="label-eyebrow text-lime">Bản tin hằng ngày</span>
              <span className="ornament flex-1 text-lime/40" />
            </div>
            <p className="mt-3 font-display text-[1.35rem] leading-[1.2] tracking-tight">
              Bảy nội dung chọn lọc theo mục tiêu của bạn — <span className="italic text-lime">gửi mỗi sáng sớm</span>.
            </p>
            <div className="mt-4 flex flex-wrap gap-1.5">
              {["Phì đại cơ", "Powerlifting", "Dinh dưỡng"].map((t) => (
                <span key={t} className="rounded-full bg-lime/15 px-2.5 py-1 font-mono text-[10px] uppercase tracking-widest text-lime">
                  {t}
                </span>
              ))}
            </div>
            <button className="label-eyebrow mt-5 w-full rounded-sm bg-ember py-2.5 text-ember-foreground hover:opacity-90">
              Thiết lập bản tin của tôi →
            </button>
          </div>

          {/* Đang thịnh hành */}
          <div>
            <div className="mb-3 flex items-baseline justify-between rule-double-b pb-2">
              <span className="label-eyebrow">Đang thịnh hành</span>
              <span className="font-mono text-[10px] text-muted-foreground">24 GIỜ</span>
            </div>
            <ol className="space-y-4">
              {[
                "Rest-pause so với set truyền thống cho phát triển tay",
                "Nghiên cứu: ngủ đủ giúp tăng 4% mức 1RM squat",
                "TS. Mike Israetel ra mắt nền tảng giáo án mới",
                "Ashwagandha và testosterone — dữ liệu nói gì",
              ].map((t, i) => (
                <li key={t} className="flex gap-3 rule-b pb-3">
                  <span className="font-display text-4xl font-black leading-[0.85] text-ember">
                    {String(i + 1).padStart(2, "0")}
                  </span>
                  <a href="#" className="font-heading text-[14px] font-semibold leading-snug hover:text-ember">
                    {t}
                  </a>
                </li>
              ))}
            </ol>
          </div>

          {/* Trích dẫn */}
          <div className="relative bg-cream p-5">
            <span className="absolute -top-4 left-4 font-display text-6xl leading-none text-ember">
              &ldquo;
            </span>
            <p className="pt-2 font-display text-lg italic leading-snug tracking-tight">
              Set quan trọng nhất là set mà bạn suýt bỏ qua.
            </p>
            <div className="mt-3 font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
              — Góc huấn luyện viên · Số IV
            </div>
          </div>

          {/* Đang theo dõi */}
          <div>
            <div className="mb-3 flex items-baseline justify-between rule-double-b pb-2">
              <span className="label-eyebrow">Đang theo dõi</span>
              <a href="#" className="label-eyebrow text-ember">Quản lý</a>
            </div>
            <ul className="space-y-0">
              {followedSources.map((s) => (
                <li key={s.name} className="flex items-center justify-between rule-b py-3">
                  <div className="flex items-center gap-3">
                    <span className="grid h-9 w-9 place-items-center rounded-sm bg-ink font-display text-xs font-black text-paper">
                      {s.name.slice(0, 2).toUpperCase()}
                    </span>
                    <div>
                      <div className="font-heading text-[13px] font-semibold leading-tight">
                        {s.name}
                      </div>
                      <div className="font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
                        {s.kind}
                      </div>
                    </div>
                  </div>
                  <span className="flex items-center gap-1">
                    <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-lime" />
                    <span className="font-mono text-[9px] uppercase tracking-widest text-muted-foreground">
                      trực tiếp
                    </span>
                  </span>
                </li>
              ))}
            </ul>
          </div>

          {/* Bàn nghiên cứu */}
          <div className="relative overflow-hidden rounded-sm border-2 border-ink bg-paper p-4">
            <div className="absolute right-0 top-0 h-full w-1/3 dotgrid opacity-10" />
            <div className="label-eyebrow text-ember">Bàn nghiên cứu</div>
            <p className="mt-2 font-display text-[1.05rem] leading-snug">
              Tuần này: <span className="italic">4 phân tích gộp, 11 RCT, 6 bài tổng quan</span> lập chỉ mục từ EuropePMC.
            </p>
            <div className="mt-4 grid grid-cols-3 gap-2 font-mono text-[11px]">
              {[
                { n: "21", l: "Nghiên cứu" },
                { n: "04", l: "Gộp" },
                { n: "11", l: "RCT" },
              ].map((s) => (
                <div key={s.l} className="rule-t pt-2">
                  <div className="font-display text-2xl font-black text-ink">{s.n}</div>
                  <div className="text-muted-foreground uppercase tracking-widest text-[10px]">
                    {s.l}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </aside>
      </main>

      {/* Dải chân bài */}
      <section className="rule-t rule-b bg-cream">
        <div className="mx-auto grid max-w-[1400px] grid-cols-2 gap-6 px-6 py-6 md:grid-cols-4">
          {[
            { k: "Biên tập", v: "Biên tập viên + LLM sàng lọc" },
            { k: "Nguồn", v: "142 kênh · cập nhật mỗi giờ" },
            { k: "Phương pháp", v: "Trọng số theo bình duyệt" },
            { k: "Nhịp gửi", v: "Bản tin sáng · 07:00" },
          ].map((c) => (
            <div key={c.k}>
              <div className="label-eyebrow text-ember">{c.k}</div>
              <div className="mt-2 font-display text-[15px] italic leading-snug">{c.v}</div>
            </div>
          ))}
        </div>
      </section>

      <footer className="bg-ink text-paper">
        <div className="mx-auto max-w-[1400px] px-6 py-10">
          <div className="flex flex-wrap items-end justify-between gap-6 rule-b border-paper/15 pb-6">
            <div>
              <div className="font-display text-5xl font-black tracking-[-0.02em]">
                Bao<span className="italic text-ember">TheX</span>
              </div>
              <div className="mt-1 font-mono text-[11px] uppercase tracking-widest text-paper/60">
                Dòng tin gọn cho người tập nghiêm túc · v0.4
              </div>
            </div>
            <div className="flex gap-6 font-mono text-[11px] uppercase tracking-widest text-paper/70">
              <a href="#" className="hover:text-lime">Giới thiệu</a>
              <a href="#" className="hover:text-lime">Nguồn</a>
              <a href="#" className="hover:text-lime">Phương pháp</a>
              <a href="#" className="hover:text-lime">Bot Telegram</a>
              <a href="#" className="hover:text-lime">RSS</a>
            </div>
          </div>
          <div className="mt-4 flex items-center justify-between font-mono text-[10px] uppercase tracking-widest text-paper/40">
            <span>© 2026 BaoTheX</span>
            <span>Phông chữ Be Vietnam Pro · tối ưu tiếng Việt</span>
          </div>
        </div>
      </footer>
    </div>
  );
}
