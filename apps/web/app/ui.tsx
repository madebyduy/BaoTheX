"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { type Item, typeLabel } from "./lib";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

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
        <AccountNav path={path} />
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

function AccountNav({ path }: { path: string }) {
  const [user, setUser] = useState<HeaderUser | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
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
    return () => {
      cancelled = true;
    };
  }, [path]);

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
        <small>Báo thể thao đa nguồn, đối chiếu thông tin và biên tập bằng tiếng Việt.</small>
      </div>
    </footer>
  );
}
