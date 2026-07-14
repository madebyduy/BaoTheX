"use client";

import { useState } from "react";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

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

export function TranslateButton({ contentId }: { contentId: number }) {
  const [status, setStatus] = useState("");
  async function translate() {
    setStatus("Đang dịch…");
    const response = await fetch(`${API}/api/v1/content/${contentId}/translate`, {
      method: "POST",
    });
    setStatus(
      response.ok
        ? "Đã xếp hàng dịch — tải lại sau ít phút"
        : "Chưa thể dịch: hãy cấu hình LLM_API_KEY",
    );
  }
  return (
    <button className="btn light translate-button" onClick={translate}>
      {status || "Dịch bài này sang tiếng Việt"}
    </button>
  );
}
