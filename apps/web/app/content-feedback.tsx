"use client";

import { useState } from "react";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

export function ContentFeedback({
  contentId,
  sourceId,
  topicId,
}: {
  contentId: number;
  sourceId?: number;
  topicId?: number;
}) {
  const [message, setMessage] = useState("");
  async function act(path: string, success: string) {
    const response = await fetch(`${API}/api/v1${path}`, {
      method: "POST",
      credentials: "include",
    });
    setMessage(response.ok ? success : "Đăng nhập để lựa chọn này áp dụng cho dòng tin của bạn");
  }
  return (
    <div className="content-feedback" aria-label="Điều chỉnh dòng tin">
      <span>Tùy chỉnh:</span>
      <button type="button" onClick={() => act(`/history/${contentId}`, "Đã đánh dấu đã đọc")}>
        Đã đọc
      </button>
      <button
        type="button"
        onClick={() => act(`/hidden/${contentId}`, "Bài này sẽ không xuất hiện lại")}
      >
        Ẩn bài này
      </button>
      {topicId ? (
        <button
          type="button"
          onClick={() => act(`/mutes/topics/${topicId}`, "Sẽ hiển thị ít nội dung tương tự")}
        >
          Ít nội dung tương tự
        </button>
      ) : null}
      {sourceId ? (
        <button
          type="button"
          onClick={() => act(`/mutes/sources/${sourceId}`, "Đã ẩn nguồn khỏi dòng tin cá nhân")}
        >
          Ẩn nguồn này
        </button>
      ) : null}
      {message ? <small>{message}</small> : null}
    </div>
  );
}
