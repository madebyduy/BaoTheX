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

type TelegramStatus = {
  configured: boolean;
  linked: boolean;
  bot_username?: string;
  username?: string;
  linked_at?: string;
};

export function TelegramSettings() {
  const [status, setStatus] = useState<TelegramStatus | null>(null);
  const [message, setMessage] = useState("Đang kiểm tra Telegram…");

  async function refresh() {
    const response = await fetch(`${API}/api/v1/telegram/status`, { credentials: "include" });
    if (!response.ok) {
      setStatus(null);
      setMessage(
        response.status === 401 ? "Đăng nhập để kết nối Telegram." : "Chưa thể kiểm tra Telegram.",
      );
      return;
    }
    const json = await response.json();
    setStatus(json.data ?? json);
    setMessage("");
  }

  useEffect(() => {
    void refresh();
  }, []);

  async function connect() {
    setMessage("Đang tạo liên kết bảo mật…");
    const response = await fetch(`${API}/api/v1/telegram/link`, { credentials: "include" });
    if (!response.ok) {
      setMessage(
        response.status === 401 ? "Bạn cần đăng nhập trước." : "Bot Telegram chưa sẵn sàng.",
      );
      return;
    }
    const json = await response.json();
    const link = (json.data ?? json).deep_link as string;
    window.open(link, "_blank", "noopener,noreferrer");
    setMessage("Trong Telegram, bấm Start rồi quay lại đây kiểm tra kết nối.");
  }

  async function unlink() {
    const response = await fetch(`${API}/api/v1/telegram`, {
      method: "DELETE",
      credentials: "include",
    });
    if (response.ok) await refresh();
  }

  async function test() {
    setMessage("Đang xếp hàng gửi bản tin thử…");
    const response = await fetch(`${API}/api/v1/notifications/test`, {
      method: "POST",
      credentials: "include",
    });
    setMessage(response.ok ? "Đã xếp hàng. Kiểm tra Telegram trong ít phút." : "Chưa thể gửi thử.");
  }

  return (
    <section className="settings-card telegram-settings">
      <div className="settings-card-head">
        <div>
          <span className="tag">BẢN TIN CÁ NHÂN</span>
          <h2>Nhận báo qua Telegram</h2>
          <p>Bản tin sáng, tin đã xác nhận và cập nhật từ đội bạn theo dõi.</p>
        </div>
        <span className={`connection-dot ${status?.linked ? "online" : ""}`}>
          {status?.linked ? "Đã kết nối" : "Chưa kết nối"}
        </span>
      </div>
      {status?.linked ? (
        <div className="telegram-actions">
          <strong>@{status.username || status.bot_username || "telegram"}</strong>
          <button className="btn ember" type="button" onClick={test}>
            Gửi bản tin thử
          </button>
          <button className="btn light" type="button" onClick={unlink}>
            Ngắt kết nối
          </button>
        </div>
      ) : (
        <div className="telegram-actions">
          <button className="btn ember" type="button" onClick={connect}>
            Mở @{status?.bot_username || "baothexbot"}
          </button>
          <button className="btn light" type="button" onClick={refresh}>
            Tôi đã bấm Start
          </button>
        </div>
      )}
      {message ? <p className="settings-message">{message}</p> : null}
    </section>
  );
}
