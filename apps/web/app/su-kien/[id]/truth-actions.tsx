"use client";

import { useEffect, useState } from "react";

const API =
  typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

export function TruthCenterActions({
  clusterId,
  updatedAt,
}: {
  clusterId: number;
  updatedAt: string;
}) {
  const key = `baothex-cluster-read-${clusterId}`;
  const [following, setFollowing] = useState(false);
  const [newSince, setNewSince] = useState(false);
  const [message, setMessage] = useState("");
  useEffect(() => {
    const last = localStorage.getItem(key);
    setNewSince(Boolean(last && new Date(updatedAt) > new Date(last)));
    localStorage.setItem(key, new Date().toISOString());
    fetch(`${API}/api/v1/clusters/${clusterId}/read`, {
      method: "POST",
      credentials: "include",
    }).catch(() => null);
  }, [clusterId, key, updatedAt]);
  async function toggle() {
    const response = await fetch(`${API}/api/v1/clusters/${clusterId}/follow`, {
      method: following ? "DELETE" : "POST",
      credentials: "include",
    });
    if (response.ok) {
      setFollowing(!following);
      setMessage("");
    } else setMessage("Đăng nhập để theo dõi cả câu chuyện");
  }
  return (
    <div className="truth-actions">
      <button className="btn ember" type="button" onClick={toggle}>
        {following ? "Đang theo dõi câu chuyện" : "Theo dõi câu chuyện"}
      </button>
      {newSince ? (
        <span>● Có nội dung mới từ lần đọc trước</span>
      ) : (
        <span>Đã ghi nhớ lần đọc này</span>
      )}
      {message ? <small>{message}</small> : null}
    </div>
  );
}
