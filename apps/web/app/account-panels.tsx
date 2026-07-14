"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

export function AccountOverview() {
  const [user, setUser] = useState<{
    email: string;
    display_name?: string;
    timezone?: string;
  } | null>(null);
  const [loading, setLoading] = useState(true);
  useEffect(() => {
    fetch(`${API}/api/v1/auth/me`, { credentials: "include" })
      .then((response) => (response.ok ? response.json() : Promise.reject()))
      .then((json) => setUser(json.data ?? json))
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
  }, []);
  if (loading) return <div className="empty-state">Đang tải tài khoản…</div>;
  if (!user) {
    return (
      <div className="empty-state">
        <strong>Bạn chưa đăng nhập</strong>
        <p>Đăng nhập để cá nhân hóa dòng tin và theo dõi chủ đề.</p>
        <Link className="btn ember" href="/dang-nhap">
          Đăng nhập
        </Link>
      </div>
    );
  }
  return (
    <div className="account-summary">
      <span className="avatar">{(user.display_name || user.email)[0].toUpperCase()}</span>
      <div>
        <strong>{user.display_name || "Thành viên BaoTheX"}</strong>
        <small>
          {user.email} · {user.timezone || "Asia/Ho_Chi_Minh"}
        </small>
      </div>
    </div>
  );
}

type Preferences = {
  daily_enabled: boolean;
  weekly_research: boolean;
  follow_alerts: boolean;
  highlights_only: boolean;
  daily_max_items: number;
};
const defaults: Preferences = {
  daily_enabled: true,
  weekly_research: true,
  follow_alerts: true,
  highlights_only: false,
  daily_max_items: 5,
};

export function NotificationSettings() {
  const [prefs, setPrefs] = useState<Preferences>(defaults);
  const [state, setState] = useState("Đang tải cài đặt…");
  useEffect(() => {
    fetch(`${API}/api/v1/notifications/prefs`, { credentials: "include" })
      .then((response) => (response.ok ? response.json() : Promise.reject()))
      .then((json) => {
        setPrefs({ ...defaults, ...(json.data ?? json) });
        setState("");
      })
      .catch(() => setState("Đăng nhập để tải cài đặt thông báo."));
  }, []);
  function toggle(key: Exclude<keyof Preferences, "daily_max_items">) {
    setPrefs((current) => ({ ...current, [key]: !current[key] }));
  }
  async function save() {
    setState("Đang lưu…");
    const response = await fetch(`${API}/api/v1/notifications/prefs`, {
      method: "PATCH",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(prefs),
    });
    setState(response.ok ? "Đã lưu cài đặt" : "Không thể lưu. Hãy đăng nhập trước.");
  }
  return (
    <div className="auth-box" style={{ margin: "20px 0 60px", width: "100%", maxWidth: 700 }}>
      <div className="form">
        {state && <p className="meta">{state}</p>}
        <label>
          <input
            type="checkbox"
            checked={prefs.daily_enabled}
            onChange={() => toggle("daily_enabled")}
          />{" "}
          Nhận bản tin buổi sáng
        </label>
        <label>
          <input
            type="checkbox"
            checked={prefs.weekly_research}
            onChange={() => toggle("weekly_research")}
          />{" "}
          Nhận nghiên cứu nổi bật hàng tuần
        </label>
        <label>
          <input
            type="checkbox"
            checked={prefs.follow_alerts}
            onChange={() => toggle("follow_alerts")}
          />{" "}
          Thông báo khi chủ đề theo dõi có bài mới
        </label>
        <label>
          <input
            type="checkbox"
            checked={prefs.highlights_only}
            onChange={() => toggle("highlights_only")}
          />{" "}
          Chỉ nhận nội dung nổi bật
        </label>
        <label>
          Số bài tối đa mỗi bản tin
          <select
            value={prefs.daily_max_items}
            onChange={(event) =>
              setPrefs({ ...prefs, daily_max_items: Number(event.target.value) })
            }
          >
            <option value={3}>3 bài</option>
            <option value={5}>5 bài</option>
            <option value={7}>7 bài</option>
          </select>
        </label>
        <button className="btn ember" onClick={save}>
          Lưu cài đặt
        </button>
      </div>
    </div>
  );
}
