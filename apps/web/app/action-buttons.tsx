"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

export function BackButton() {
  const router = useRouter();
  function goBack() {
    if (window.history.length > 1) router.back();
    else router.push("/");
  }
  return (
    <button
      className="article-back"
      type="button"
      onClick={goBack}
      aria-label="Quay lại trang trước"
    >
      <span>←</span> Quay lại
    </button>
  );
}

export function SiteBackButton() {
  const pathname = usePathname();
  const router = useRouter();
  if (pathname === "/") return null;
  return (
    <div className="site-back-wrap">
      <button
        className="article-back site-back"
        type="button"
        onClick={() => (window.history.length > 1 ? router.back() : router.push("/"))}
      >
        <span>←</span> Quay lại
      </button>
    </div>
  );
}

export function SaveButton({ contentId }: { contentId: number }) {
  const [saved, setSaved] = useState(false);
  const [message, setMessage] = useState("");
  async function toggle() {
    const response = await fetch(`${API}/api/v1/saved/${contentId}`, {
      method: saved ? "DELETE" : "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: saved ? undefined : JSON.stringify({}),
    });
    if (response.ok) {
      setSaved(!saved);
      setMessage(saved ? "Đã bỏ lưu" : "Đã lưu vào thư viện");
    } else {
      setMessage("Bạn cần đăng nhập để lưu nội dung");
    }
  }
  return (
    <button className="btn ember" onClick={toggle}>
      {message || (saved ? "Bỏ lưu" : "Lưu bài viết")}
    </button>
  );
}

export function FollowButton({ topicId, entityId }: { topicId?: number; entityId?: number }) {
  const [following, setFollowing] = useState(false);
  const [message, setMessage] = useState("");
  async function toggle() {
    const target = entityId ? `entities/${entityId}` : `topics/${topicId}`;
    const response = await fetch(`${API}/api/v1/follows/${target}`, {
      method: following ? "DELETE" : "POST",
      credentials: "include",
    });
    if (response.ok) {
      setFollowing(!following);
      setMessage(following ? "Đã bỏ theo dõi" : "Đang theo dõi");
    } else {
      setMessage(`Đăng nhập để theo dõi ${entityId ? "nhân vật" : "chủ đề"}`);
    }
  }
  return (
    <button className="btn light" onClick={toggle}>
      {message || (following ? "Bỏ theo dõi" : entityId ? "Theo dõi nhân vật" : "Theo dõi chủ đề")}
    </button>
  );
}

// getClientId returns a stable random per-device id used to dedup anonymous
// likes without requiring login.
function getClientId(): string {
  try {
    let id = localStorage.getItem("btx-cid");
    if (!id) {
      id =
        (typeof crypto !== "undefined" && crypto.randomUUID?.()) ||
        Math.random().toString(36).slice(2) + Date.now().toString(36);
      localStorage.setItem("btx-cid", id);
    }
    return id;
  } catch {
    return "";
  }
}

// LikeButton is a server-backed "thích" with a real, per-device-deduped count.
export function LikeButton({ contentId }: { contentId: number }) {
  const [liked, setLiked] = useState(false);
  const [count, setCount] = useState(0);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const cid = getClientId();
    fetch(`${API}/api/v1/content/${contentId}/reactions?client_id=${encodeURIComponent(cid)}`)
      .then((r) => (r.ok ? r.json() : null))
      .then((j) => {
        if (!j) return;
        const d = j.data ?? j;
        setLiked(Boolean(d.liked));
        setCount(Number(d.count) || 0);
      })
      .catch(() => {});
  }, [contentId]);

  async function toggle() {
    if (busy) return;
    setBusy(true);
    const cid = getClientId();
    const next = !liked;
    // Optimistic update; reconcile with the server's authoritative count.
    setLiked(next);
    setCount((c) => Math.max(0, c + (next ? 1 : -1)));
    try {
      const response = await fetch(`${API}/api/v1/content/${contentId}/like`, {
        method: next ? "POST" : "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ client_id: cid }),
      });
      if (response.ok) {
        const j = await response.json();
        const d = j.data ?? j;
        setLiked(Boolean(d.liked));
        setCount(Number(d.count) || 0);
      }
    } catch {
      /* keep optimistic value */
    } finally {
      setBusy(false);
    }
  }

  return (
    <button
      className={`like-btn ${liked ? "liked" : ""}`}
      onClick={toggle}
      aria-pressed={liked}
      type="button"
    >
      <span className="heart" aria-hidden>
        {liked ? "♥" : "♡"}
      </span>
      {liked ? "Đã thích" : "Thích"}
      {count > 0 ? <span className="like-count">{count}</span> : null}
    </button>
  );
}

// ShareBar shares the current article via the native share sheet (mobile),
// social networks, or a copy-link fallback. All client-side, no backend needed.
export function ShareBar({ title }: { title: string }) {
  const [url, setUrl] = useState("");
  const [copied, setCopied] = useState(false);
  useEffect(() => setUrl(window.location.href), []);

  function openShare(target: "facebook" | "x" | "telegram") {
    const u = encodeURIComponent(url);
    const t = encodeURIComponent(title);
    const links = {
      facebook: `https://www.facebook.com/sharer/sharer.php?u=${u}`,
      x: `https://twitter.com/intent/tweet?url=${u}&text=${t}`,
      telegram: `https://t.me/share/url?url=${u}&text=${t}`,
    };
    window.open(links[target], "_blank", "noopener,noreferrer,width=620,height=520");
  }
  async function copy() {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    } catch {
      /* clipboard unavailable */
    }
  }
  async function nativeShare() {
    if (typeof navigator !== "undefined" && navigator.share) {
      try {
        await navigator.share({ title, url });
      } catch {
        /* user cancelled */
      }
    } else {
      void copy();
    }
  }

  return (
    <div className="share-bar">
      <span className="share-label">Chia sẻ</span>
      <button className="share-btn fb" type="button" onClick={() => openShare("facebook")}>
        Facebook
      </button>
      <button className="share-btn x" type="button" onClick={() => openShare("x")}>
        X
      </button>
      <button className="share-btn tg" type="button" onClick={() => openShare("telegram")}>
        Telegram
      </button>
      <button
        className={`share-btn copy ${copied ? "copied" : ""}`}
        type="button"
        onClick={() => void nativeShare()}
      >
        {copied ? "Đã chép link ✓" : "Chép link"}
      </button>
    </div>
  );
}

// TranslateButton is gone: foreign articles are digested into Vietnamese on
// ingest and never rendered as a full translation, so there is nothing for a
// reader to request. The POST /content/{id}/translate endpoint it called still
// exists and still works — it now queues a digest — which is worth keeping for
// re-processing a story by hand.
