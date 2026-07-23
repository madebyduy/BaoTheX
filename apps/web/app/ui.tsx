"use client";
import Link from "next/link";
import Image, { type ImageProps } from "next/image";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { articleHref, type Item, typeLabel } from "./lib";

// Publisher images come from a large, changing set of hosts. Keeping them
// unoptimised avoids turning Next's image endpoint into an open remote fetcher,
// while explicit intrinsic dimensions still prevent layout shift.
export function RemoteImage({ alt, ...props }: Omit<ImageProps, "width" | "height">) {
  return <Image {...props} alt={alt} width={1200} height={675} unoptimized />;
}

const API =
  typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

type HeaderUser = {
  email: string;
  display_name?: string;
  role?: string;
};

export function SiteHeader() {
  const path = usePathname();
  const links = [
    ["/", "Dòng tin"],
    ["/ban-the-thao", "Bàn thể thao"],
    ["/bat-kip", "Bắt kịp"],
    ["/lich-the-thao", "Lịch"],
    ["/goc-nhin", "Góc nhìn"],
    ["/video", "Video"],
    ["/chu-de", "Chủ đề"],
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
        <SearchBox />
        <Link className="premium-link" href="/premium">
          Premium
        </Link>
        <AccountNav />
      </div>
      <div className="ticker">
        <div className="wrap">
          <b>● ĐANG CẬP NHẬT</b> Tin nóng, video và góc nhìn thể thao được chọn lọc mỗi ngày
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

function SearchBox() {
  const [query, setQuery] = useState("");
  const [suggestions, setSuggestions] = useState<string[]>([]);
  useEffect(() => {
    if (query.trim().length < 2) {
      setSuggestions([]);
      return;
    }
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      fetch(`${API}/api/v1/search/suggest?q=${encodeURIComponent(query)}`, {
        signal: controller.signal,
      })
        .then((response) => (response.ok ? response.json() : Promise.reject()))
        .then((json) => {
          const data = json.data ?? json;
          const values = Array.isArray(data) ? data : data.suggestions || [];
          setSuggestions(
            values
              .map((value: unknown) =>
                typeof value === "string"
                  ? value
                  : String(
                      (value as { name?: string; text?: string; title?: string }).name ||
                        (value as { text?: string }).text ||
                        (value as { title?: string }).title ||
                        "",
                    ),
              )
              .filter(Boolean)
              .slice(0, 6),
          );
        })
        .catch(() => null);
    }, 180);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [query]);
  return (
    <div className="search-wrap">
      <form className="search" action="/tim-kiem">
        <span>⌕</span>
        <input
          name="q"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Tìm đội, VĐV, nội dung…"
          autoComplete="off"
        />
      </form>
      {suggestions.length ? (
        <div className="search-suggestions">
          {suggestions.map((value) => (
            <Link
              href={`/tim-kiem?q=${encodeURIComponent(value)}`}
              key={value}
              onClick={() => setSuggestions([])}
            >
              {value}
            </Link>
          ))}
        </div>
      ) : null}
    </div>
  );
}

function AccountNav() {
  const [user, setUser] = useState<HeaderUser | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    function loadUser() {
      setLoading(true);
      fetch(`${API}/api/v1/auth/me`, { credentials: "include", cache: "no-store" })
        .then((response) => (response.ok ? response.json() : Promise.reject()))
        .then((json) => {
          if (!cancelled) setUser(json.data ?? json);
        })
        .catch(() => {
          if (!cancelled) setUser(null);
        })
        .finally(() => {
          if (!cancelled) setLoading(false);
        });
    }
    loadUser();
    window.addEventListener("baothex:auth-changed", loadUser);
    return () => {
      cancelled = true;
      window.removeEventListener("baothex:auth-changed", loadUser);
    };
  }, []);

  async function logout() {
    await fetch(`${API}/api/v1/auth/logout`, {
      method: "POST",
      credentials: "include",
    }).catch(() => null);
    setUser(null);
    window.location.href = "/";
  }

  if (loading) {
    return <span className="account-pill skeleton-pill" aria-label="Dang kiem tra dang nhap" />;
  }

  if (!user) {
    return (
      <Link className="btn ember header-login" href="/dang-nhap">
        Đăng nhập
      </Link>
    );
  }

  const name = user.display_name || user.email.split("@")[0] || "Tài khoản";

  return (
    <div className="header-account" aria-label="Tai khoan">
      {user.role === "admin" ? (
        <Link className="account-admin" href="/admin">
          Admin
        </Link>
      ) : null}
      <Link className="account-pill" href="/cai-dat" title={user.email}>
        <span className="avatar">{name[0]?.toUpperCase()}</span>
        <span>{name}</span>
      </Link>
      <button className="logout-btn" type="button" onClick={logout}>
        Thoát
      </button>
    </div>
  );
}
export function Card({ item }: { item: Item }) {
  return (
    <Link className="card" href={articleHref(item)}>
      {item.image_url ? (
        <RemoteImage
          className="card-image"
          src={item.image_url}
          alt=""
          loading="lazy"
          decoding="async"
          referrerPolicy="no-referrer"
        />
      ) : null}
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
  const year = new Date().getFullYear();
  return (
    <footer className="footer">
      <div className="wrap footer-grid">
        <div className="footer-brand">
          <div className="brand">
            Bao<i>TheX</i>
          </div>
          <small>
            Báo thể thao đa nguồn, đối chiếu thông tin và biên tập bằng tiếng Việt. Nội dung được
            tổng hợp từ các nguồn uy tín và dẫn nguồn gốc rõ ràng.
          </small>
        </div>
        <nav className="footer-col" aria-label="Chuyên mục">
          <b>Chuyên mục</b>
          <Link href="/danh-muc">Tin mới nhất</Link>
          <Link href="/ban-the-thao">Bàn thể thao</Link>
          <Link href="/video">Video</Link>
          <Link href="/lich-the-thao">Lịch & kết quả</Link>
          <Link href="/goc-nhin">Góc nhìn</Link>
        </nav>
        <nav className="footer-col" aria-label="Tòa soạn">
          <b>Tòa soạn</b>
          <Link href="/gioi-thieu">Giới thiệu</Link>
          <Link href="/chinh-sach-bien-tap">Chính sách biên tập</Link>
          <Link href="/nguyen-tac-kiem-chung">Nguyên tắc kiểm chứng</Link>
          <Link href="/lien-he">Liên hệ</Link>
        </nav>
        <nav className="footer-col" aria-label="Pháp lý">
          <b>Pháp lý</b>
          <Link href="/dieu-khoan">Điều khoản sử dụng</Link>
          <Link href="/quyen-rieng-tu">Quyền riêng tư</Link>
          <Link href="/ban-quyen">Bản quyền</Link>
        </nav>
        <nav className="footer-col" aria-label="Theo dõi">
          <b>Theo dõi</b>
          <a href="/feed.xml">RSS</a>
          <Link href="/cai-dat">Nhận bản tin</Link>
          <Link href="/premium">Premium</Link>
        </nav>
      </div>
      <div className="wrap footer-legal">
        <small>© {year} BaoTheX. Mọi quyền được bảo lưu.</small>
      </div>
    </footer>
  );
}
