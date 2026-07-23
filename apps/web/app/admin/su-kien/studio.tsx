"use client";

import { useState } from "react";
import type { FormEvent } from "react";
import type { Sport, SportsEvent } from "../../lib";

const API = (typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081");

export function AdminSportsStudio({
  sports,
  initialEvents,
}: {
  sports: Sport[];
  initialEvents: SportsEvent[];
}) {
  const [message, setMessage] = useState("");
  const [events, setEvents] = useState(initialEvents);
  const [options, setOptions] = useState("Đội A\nĐội B");
  async function createEvent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("Đang lưu…");
    const form = new FormData(event.currentTarget);
    const payload = {
      sport_slug: form.get("sport_slug"),
      title: form.get("title"),
      home_name: nullable(form.get("home_name")),
      away_name: nullable(form.get("away_name")),
      starts_at: new Date(String(form.get("starts_at"))).toISOString(),
      status: form.get("status"),
      home_score: nullable(form.get("home_score")),
      away_score: nullable(form.get("away_score")),
      manual_locked: form.get("manual_locked") === "on",
    };
    const response = await fetch(`${API}/api/v1/admin/events`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    setMessage(
      response.ok
        ? "Đã tạo sự kiện và ghi nhãn BaoTheX cập nhật."
        : "Không thể tạo. Hãy kiểm tra quyền admin và dữ liệu.",
    );
    if (response.ok) {
      const json = await response.json();
      setEvents((values) => [json.data ?? json, ...values]);
      event.currentTarget.reset();
    }
  }
  async function updateResult(eventId: number, form: HTMLFormElement) {
    const data = new FormData(form);
    const payload = {
      status: data.get("status"),
      home_score: nullable(data.get("home_score")),
      away_score: nullable(data.get("away_score")),
      manual_locked: true,
    };
    const response = await fetch(`${API}/api/v1/admin/events/${eventId}/result`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (!response.ok) {
      setMessage("Không thể cập nhật kết quả.");
      return;
    }
    const json = await response.json();
    const updated = json.data ?? json;
    setEvents((values) => values.map((item) => (item.id === eventId ? updated : item)));
    setMessage("Đã cập nhật và khóa kết quả thủ công.");
  }
  async function createPrediction(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("Đang lưu…");
    const form = new FormData(event.currentTarget);
    const payload = {
      kind: form.get("kind"),
      question: form.get("question"),
      options: options
        .split("\n")
        .map((value) => value.trim())
        .filter(Boolean),
      deadline: new Date(String(form.get("deadline"))).toISOString(),
      points: Number(form.get("points")) || 10,
      status: "open",
      answer_count: 0,
    };
    const response = await fetch(`${API}/api/v1/admin/predictions`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    setMessage(
      response.ok ? "Đã lưu câu hỏi cố định cho người chơi." : "Không thể tạo dự đoán/quiz.",
    );
  }
  return (
    <>
      <div className="admin-studio-grid">
        <form className="admin-studio-form" onSubmit={createEvent}>
          <header>
            <span>01</span>
            <h2>Tạo sự kiện thủ công</h2>
          </header>
          <label>
            Môn
            <select name="sport_slug" required>
              {sports.map((sport) => (
                <option value={sport.slug} key={sport.id}>
                  {sport.name}
                </option>
              ))}
            </select>
          </label>
          <label>
            Tiêu đề
            <input name="title" required placeholder="Việt Nam vs Thái Lan" />
          </label>
          <div className="form-pair">
            <label>
              Đội/VĐV 1<input name="home_name" />
            </label>
            <label>
              Đội/VĐV 2<input name="away_name" />
            </label>
          </div>
          <label>
            Thời gian
            <input name="starts_at" type="datetime-local" required />
          </label>
          <label>
            Trạng thái
            <select name="status">
              <option value="scheduled">Lịch dự kiến</option>
              <option value="live">Đang diễn ra</option>
              <option value="finished">Kết thúc</option>
              <option value="postponed">Tạm hoãn</option>
              <option value="cancelled">Đã hủy</option>
            </select>
          </label>
          <div className="form-pair">
            <label>
              Điểm 1<input name="home_score" />
            </label>
            <label>
              Điểm 2<input name="away_score" />
            </label>
          </div>
          <label className="check-line">
            <input type="checkbox" name="manual_locked" defaultChecked />
            Khóa kết quả, không cho provider ghi đè
          </label>
          <button className="btn ember" type="submit">
            Tạo sự kiện
          </button>
        </form>
        <form className="admin-studio-form" onSubmit={createPrediction}>
          <header>
            <span>02</span>
            <h2>Tạo prediction / quiz</h2>
          </header>
          <label>
            Loại
            <select name="kind">
              <option value="winner">Người thắng</option>
              <option value="score">Tỷ số</option>
              <option value="player">VĐV nổi bật</option>
              <option value="quiz">Quiz kiến thức</option>
              <option value="poll">Bình chọn</option>
            </select>
          </label>
          <label>
            Câu hỏi
            <input name="question" required />
          </label>
          <label>
            Các lựa chọn (mỗi dòng một đáp án)
            <textarea
              value={options}
              onChange={(event) => setOptions(event.target.value)}
              rows={6}
              required
            />
          </label>
          <label>
            Khóa trả lời lúc
            <input name="deadline" type="datetime-local" required />
          </label>
          <label>
            Điểm
            <input name="points" type="number" min="0" max="1000" defaultValue="10" />
          </label>
          <button className="btn ember" type="submit">
            Lưu câu hỏi cố định
          </button>
        </form>
      </div>
      {message ? <div className="admin-studio-message">{message}</div> : null}
      <section className="admin-event-results">
        <header>
          <span>03</span>
          <h2>Sửa trạng thái & kết quả gần đây</h2>
        </header>
        {events.length ? (
          events.map((item) => (
            <form
              key={item.id}
              onSubmit={(event) => {
                event.preventDefault();
                updateResult(item.id, event.currentTarget);
              }}
            >
              <strong>{item.title}</strong>
              <select name="status" defaultValue={item.status}>
                <option value="scheduled">Lịch dự kiến</option>
                <option value="live">Đang diễn ra</option>
                <option value="finished">Kết thúc</option>
                <option value="postponed">Tạm hoãn</option>
                <option value="cancelled">Đã hủy</option>
              </select>
              <input name="home_score" defaultValue={item.home_score || ""} placeholder="Điểm 1" />
              <input name="away_score" defaultValue={item.away_score || ""} placeholder="Điểm 2" />
              <button type="submit">Lưu & khóa</button>
            </form>
          ))
        ) : (
          <p>Chưa có sự kiện.</p>
        )}
      </section>
    </>
  );
}
function nullable(value: FormDataEntryValue | null) {
  const text = String(value || "").trim();
  return text || null;
}
