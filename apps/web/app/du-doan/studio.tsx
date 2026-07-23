"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import type { Prediction } from "../lib";

const API = (typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081");

export function PredictionStudio() {
  const [items, setItems] = useState<Prediction[]>([]);
  const [loading, setLoading] = useState(true);
  const [login, setLogin] = useState(false);
  useEffect(() => {
    fetch(`${API}/api/v1/predictions`, { credentials: "include", cache: "no-store" })
      .then((response) => {
        setLogin(response.ok);
        return response.ok ? response.json() : Promise.reject();
      })
      .then((json) => setItems(json.data ?? json))
      .catch(() => setItems([]))
      .finally(() => setLoading(false));
  }, []);
  async function answer(id: number, value: string) {
    const response = await fetch(`${API}/api/v1/predictions/${id}/answer`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ answer: value }),
    });
    if (!response.ok) return;
    setItems((values) =>
      values.map((item) =>
        item.id === id
          ? { ...item, user_answer: value, answer_count: item.answer_count + 1 }
          : item,
      ),
    );
  }
  if (loading) return <div className="empty-state">Đang mở studio…</div>;
  if (!login)
    return (
      <div className="prediction-login">
        <span>HỒ SƠ RIÊNG TƯ</span>
        <h2>Đăng nhập để câu trả lời được khóa và tính điểm đúng một lần.</h2>
        <p>Không có bảng xếp hạng công khai trong phiên bản này.</p>
        <Link className="btn ember" href="/dang-nhap">
          Đăng nhập
        </Link>
      </div>
    );
  if (!items.length)
    return (
      <div className="empty-state">
        <h2>Chưa có câu hỏi đang mở</h2>
        <p>Quản trị viên có thể tạo dự đoán hoặc quiz từ nội dung đã kiểm duyệt.</p>
      </div>
    );
  return (
    <div className="prediction-grid">
      {items.map((item) => {
        const open = item.status === "open" && new Date(item.deadline) > new Date();
        return (
          <article className="prediction-card" key={item.id}>
            <header>
              <span>{item.kind === "quiz" ? "QUIZ" : "DỰ ĐOÁN"}</span>
              <b>+{item.points} điểm</b>
            </header>
            <h2>{item.question}</h2>
            <div className="prediction-options">
              {item.options.map((option) => (
                <button
                  type="button"
                  key={option}
                  disabled={!open || Boolean(item.user_answer)}
                  className={item.user_answer === option ? "selected" : ""}
                  onClick={() => answer(item.id, option)}
                >
                  {option}
                </button>
              ))}
            </div>
            <footer>
              <span>
                {open
                  ? `Khóa lúc ${new Intl.DateTimeFormat("vi-VN", { hour: "2-digit", minute: "2-digit", day: "2-digit", month: "2-digit" }).format(new Date(item.deadline))}`
                  : "Đã khóa"}
              </span>
              <span>{item.answer_count} lượt trả lời</span>
            </footer>
            {item.status === "settled" ? (
              <p className={item.is_correct ? "answer-good" : "answer-result"}>
                {item.is_correct
                  ? "Bạn trả lời đúng"
                  : `Đáp án: ${item.correct_option || "đang cập nhật"}`}
              </p>
            ) : null}
          </article>
        );
      })}
    </div>
  );
}
