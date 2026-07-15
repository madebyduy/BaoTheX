"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { type Item, typeLabel } from "./lib";

export function SiteHeader() {
  const path = usePathname();
  const links = [
    ["/", "Dòng tin"],
    ["/goc-nhin", "Góc nhìn"],
    ["/nghien-cuu", "Nghiên cứu"],
    ["/video", "Video"],
    ["/podcast", "Podcast"],
    ["/chu-de", "Chủ đề"],
    ["/nhan-vat", "Nhân vật"],
    ["/nguon", "Nguồn"],
  ];
  return (
    <header className="header">
      <div className="wrap topbar">
        <Link className="brand" href="/">
          Bao<i>TheX</i>
        </Link>
        <nav className="nav">
          {links.map(([href, label]) => (
            <Link className={path === href ? "active" : ""} href={href} key={href}>
              {label}
            </Link>
          ))}
        </nav>
        <form className="search" action="/tim-kiem">
          <span>⌕</span>
          <input name="q" placeholder="Tìm kiếm nội dung…" />
        </form>
        <Link className="premium-link" href="/premium">
          Premium
        </Link>
        <Link className="btn ember" href="/dang-nhap">
          Đăng nhập
        </Link>
      </div>
      <div className="ticker">
        <div className="wrap">
          <b>● ĐANG CẬP NHẬT</b> Nghiên cứu, video và kiến thức tập luyện được chọn lọc mỗi ngày
        </div>
      </div>
      <div className="subbar">
        <div className="wrap">
          Tin mới mỗi ngày <span>·</span> Nội dung có nguồn cho người Việt
        </div>
      </div>
    </header>
  );
}
export function Card({ item }: { item: Item }) {
  return (
    <Link className="card" href={`/noi-dung/${item.id}`}>
      {item.image_url ? <img className="card-image" src={item.image_url} alt="" /> : null}
      <span className="tag">{typeLabel(item.type)}</span>
      <h3>{item.title}</h3>
      <p>{item.summary || item.excerpt || "Nội dung đang được biên tập và tóm tắt."}</p>
      <div className="meta">{item.source_name || "BaoTheX"} · Đọc bài →</div>
    </Link>
  );
}
export function ItemGrid({ items }: { items: Item[] }) {
  return (
    <div className="grid">
      {items.map((x) => (
        <Card item={x} key={x.id} />
      ))}
    </div>
  );
}
export function PageTitle({
  eyebrow,
  title,
  description,
}: {
  eyebrow: string;
  title: string;
  description?: string;
}) {
  return (
    <div className="section" style={{ paddingBottom: 10 }}>
      <span className="tag">{eyebrow}</span>
      <h1
        style={{
          fontSize: "clamp(36px,5vw,64px)",
          lineHeight: 1.05,
          letterSpacing: "-.055em",
          marginTop: 8,
        }}
      >
        {title}
      </h1>
      {description && (
        <p style={{ maxWidth: 680, color: "var(--muted)", marginTop: 14, fontSize: 16 }}>
          {description}
        </p>
      )}
    </div>
  );
}
export function Footer() {
  return (
    <footer className="footer">
      <div className="wrap">
        <div className="brand">
          Bao<i>TheX</i>
        </div>
        <small>
          Nền tảng tổng hợp bài viết, nghiên cứu và kiến thức tập luyện có dẫn nguồn cho người Việt.
        </small>
      </div>
    </footer>
  );
}
