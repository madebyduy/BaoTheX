"use client";

import { useEffect, useState } from "react";

const API = (typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081");

type InstallPrompt = Event & {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: string }>;
};
type Capabilities = { web_push_enabled: boolean; web_push_public_key?: string };
type PremiumStatus = { active?: boolean };

export function PWAControls() {
  const [installPrompt, setInstallPrompt] = useState<InstallPrompt | null>(null);
  const [capabilities, setCapabilities] = useState<Capabilities | null>(null);
  const [state, setState] = useState("Đang kiểm tra thiết bị…");
  const [subscribed, setSubscribed] = useState(false);
  const [premiumActive, setPremiumActive] = useState(false);

  useEffect(() => {
    const onInstall = (event: Event) => {
      event.preventDefault();
      setInstallPrompt(event as InstallPrompt);
    };
    window.addEventListener("beforeinstallprompt", onInstall);
    void fetch(`${API}/api/v1/capabilities`)
      .then((r) => r.json())
      .then((json) => setCapabilities(json.data ?? json));
    void fetch(`${API}/api/v1/premium/status`, { credentials: "include" })
      .then((response) => (response.ok ? response.json() : Promise.reject()))
      .then((json) => {
        const premium = (json.data ?? json) as PremiumStatus;
        setPremiumActive(Boolean(premium.active));
      })
      .catch(() => setPremiumActive(false));
    if ("serviceWorker" in navigator) {
      void navigator.serviceWorker
        .register("/sw.js")
        .then(async (registration) => {
          setSubscribed(Boolean(await registration.pushManager.getSubscription()));
          setState("");
        })
        .catch(() => setState("Trình duyệt chưa cho phép cài ứng dụng."));
    } else {
      setState("Trình duyệt này chưa hỗ trợ PWA.");
    }
    return () => window.removeEventListener("beforeinstallprompt", onInstall);
  }, []);

  async function install() {
    if (!installPrompt) {
      setState("Trong Chrome/Edge, mở menu trình duyệt và chọn “Cài đặt BaoTheX”.");
      return;
    }
    await installPrompt.prompt();
    await installPrompt.userChoice;
    setInstallPrompt(null);
  }

  async function enablePush() {
    if (!premiumActive) {
      setState(
        "Web Push tức thời thuộc Premium 10.000đ/tháng; ứng dụng cài trên máy vẫn miễn phí.",
      );
      return;
    }
    if (!capabilities?.web_push_enabled || !capabilities.web_push_public_key) {
      setState("Máy chủ chưa bật Web Push.");
      return;
    }
    const permission = await Notification.requestPermission();
    if (permission !== "granted") {
      setState("Bạn chưa cho phép thông báo trên thiết bị này.");
      return;
    }
    const registration = await navigator.serviceWorker.ready;
    let subscription = await registration.pushManager.getSubscription();
    if (!subscription) {
      subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(capabilities.web_push_public_key),
      });
    }
    const response = await fetch(`${API}/api/v1/push/subscribe`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(subscription.toJSON()),
    });
    setSubscribed(response.ok);
    setState(
      response.ok ? "Đã bật thông báo cho thiết bị này." : "Hãy đăng nhập trước khi bật thông báo.",
    );
  }

  async function testPush() {
    const response = await fetch(`${API}/api/v1/push/test`, {
      method: "POST",
      credentials: "include",
    });
    setState(
      response.ok ? "Đã gửi thông báo thử." : "Chưa thể gửi; hãy bật thông báo và đăng nhập.",
    );
  }

  return (
    <section className="settings-card">
      <div className="settings-card-head">
        <div>
          <span className="tag">ỨNG DỤNG MÁY TÍNH</span>
          <h2>Cài BaoTheX và nhận tin đúng lúc</h2>
          <p>Mở như một ứng dụng riêng trên Windows và nhận tin từ đội, giải đấu bạn theo dõi.</p>
        </div>
        <span className={`connection-dot ${subscribed ? "online" : ""}`}>
          {subscribed ? "Đã bật thông báo" : "Chưa bật"}
        </span>
      </div>
      <div className="telegram-actions">
        <button className="btn light" type="button" onClick={install}>
          Cài lên máy tính
        </button>
        <button className="btn ember" type="button" onClick={enablePush}>
          {premiumActive ? "Bật thông báo" : "Mở với Premium"}
        </button>
        {subscribed && premiumActive ? (
          <button className="btn light" type="button" onClick={testPush}>
            Gửi thử
          </button>
        ) : null}
      </div>
      {state ? <p className="settings-message">{state}</p> : null}
    </section>
  );
}

function urlBase64ToUint8Array(value: string) {
  const padding = "=".repeat((4 - (value.length % 4)) % 4);
  const base64 = (value + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = window.atob(base64);
  return Uint8Array.from([...raw].map((char) => char.charCodeAt(0)));
}
