"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import type { Topic } from "./lib";

const API =
  typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

type Follows = { topics?: Topic[] };
type FeedPreferences = { feed_following_only?: boolean };
type PremiumStatus = { active?: boolean };

export function FeedCustomizationSettings() {
  const router = useRouter();
  const [topics, setTopics] = useState<Topic[]>([]);
  const [followed, setFollowed] = useState<Set<number>>(new Set());
  const [followingOnly, setFollowingOnly] = useState(false);
  const [premiumActive, setPremiumActive] = useState(false);
  const [message, setMessage] = useState("Đang tải dòng tin của bạn…");
  const [busy, setBusy] = useState<number | "mode" | null>(null);

  useEffect(() => {
    Promise.all([
      fetch(`${API}/api/v1/topics`, { credentials: "include" }),
      fetch(`${API}/api/v1/follows`, { credentials: "include" }),
      fetch(`${API}/api/v1/notifications/prefs`, { credentials: "include" }),
      fetch(`${API}/api/v1/premium/status`, { credentials: "include" }),
    ])
      .then(async ([topicResponse, followResponse, prefsResponse, premiumResponse]) => {
        if (!followResponse.ok || !prefsResponse.ok || !premiumResponse.ok) throw new Error("auth");
        const [topicJSON, followJSON, prefsJSON, premiumJSON] = await Promise.all([
          topicResponse.json(),
          followResponse.json(),
          prefsResponse.json(),
          premiumResponse.json(),
        ]);
        const allTopics = (topicJSON.data ?? topicJSON) as Topic[];
        const follows = (followJSON.data ?? followJSON) as Follows;
        const prefs = (prefsJSON.data ?? prefsJSON) as FeedPreferences;
        const premium = (premiumJSON.data ?? premiumJSON) as PremiumStatus;
        setTopics(allTopics);
        setFollowed(new Set((follows.topics || []).map((topic) => topic.id)));
        setPremiumActive(Boolean(premium.active));
        setFollowingOnly(Boolean(premium.active && prefs.feed_following_only));
        setMessage("");
      })
      .catch(() => setMessage("Đăng nhập để tạo dòng tin riêng theo sở thích."));
  }, []);

  const visibleTopics = useMemo(
    () =>
      topics
        .filter(
          (topic) =>
            topic.category === "sport" ||
            /bong|tennis|cau-long|the-thao|the-hinh|motor|esport|khac/.test(topic.slug),
        )
        .sort((a, b) => a.name.localeCompare(b.name, "vi")),
    [topics],
  );

  async function toggleTopic(topic: Topic) {
    const isFollowing = followed.has(topic.id);
    if (isFollowing && followingOnly && followed.size === 1) {
      setMessage("Hãy chuyển về chế độ cân bằng trước khi bỏ chủ đề cuối cùng.");
      return;
    }
    setBusy(topic.id);
    const response = await fetch(`${API}/api/v1/follows/topics/${topic.id}`, {
      method: isFollowing ? "DELETE" : "POST",
      credentials: "include",
    });
    if (response.ok) {
      setFollowed((current) => {
        const next = new Set(current);
        if (isFollowing) next.delete(topic.id);
        else next.add(topic.id);
        return next;
      });
      setMessage(isFollowing ? `Đã bỏ ${topic.name}.` : `Đang theo dõi ${topic.name}.`);
    } else {
      setMessage("Không thể cập nhật. Hãy đăng nhập lại.");
    }
    setBusy(null);
    router.refresh();
  }

  async function toggleMode() {
    const next = !followingOnly;
    if (next && !premiumActive) {
      setMessage("Chế độ chỉ hiện chủ đề theo dõi thuộc Premium 10.000đ/tháng.");
      return;
    }
    if (next && followed.size === 0) {
      setMessage("Hãy chọn ít nhất một chủ đề trước khi bật chế độ chỉ theo dõi.");
      return;
    }
    setBusy("mode");
    const response = await fetch(`${API}/api/v1/notifications/prefs`, {
      method: "PATCH",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ feed_following_only: next }),
    });
    if (response.ok) {
      setFollowingOnly(next);
      setMessage(
        next
          ? "Đã bật dòng tin theo dõi. Trang chủ sẽ chỉ hiện các chủ đề bạn chọn."
          : "Đã trở lại trang chủ tổng hợp mọi môn.",
      );
    } else {
      setMessage("Không thể lưu chế độ dòng tin.");
    }
    setBusy(null);
    router.refresh();
  }

  return (
    <section className="settings-card feed-customization">
      <div className="settings-card-head">
        <div>
          <span className="tag">DÒNG TIN CỦA BẠN</span>
          <h2>Chọn môn bạn thực sự quan tâm</h2>
          <p>
            Chế độ cân bằng ưu tiên mạnh các môn bạn chọn nhưng vẫn dành một phần nhỏ để khám phá
            tin quan trọng. Chỉ bật chế độ theo dõi khi bạn muốn lọc tuyệt đối.
          </p>
        </div>
        <button
          className={`feed-mode-switch ${followingOnly ? "active" : ""}`}
          type="button"
          role="switch"
          aria-checked={followingOnly}
          disabled={busy === "mode"}
          onClick={toggleMode}
        >
          <i />
          <span>{followingOnly ? "Chỉ chủ đề theo dõi" : "Cân bằng & khám phá"}</span>
        </button>
      </div>

      <div className="feed-preference-summary">
        <div>
          <strong>{followed.size}</strong>
          <span>chủ đề đang theo dõi</span>
        </div>
        <p>
          Chủ đề đã chọn được ưu tiên trên trang chủ và bản tin Telegram. Bạn có thể thay đổi bất cứ
          lúc nào.
        </p>
        <Link href={premiumActive ? "/" : "/premium"}>
          {premiumActive ? "Xem dòng tin của tôi →" : "Mở chế độ chỉ theo dõi · 10.000đ/tháng →"}
        </Link>
      </div>

      {visibleTopics.length ? (
        <div className="topic-choice-grid">
          {visibleTopics.map((topic) => {
            const active = followed.has(topic.id);
            return (
              <button
                className={active ? "active" : ""}
                type="button"
                key={topic.id}
                disabled={busy === topic.id}
                onClick={() => toggleTopic(topic)}
              >
                <span>{active ? "✓" : "+"}</span>
                <div>
                  <strong>{topic.name}</strong>
                  <small>{active ? "Đang có trong dòng tin" : "Thêm vào dòng tin"}</small>
                </div>
              </button>
            );
          })}
        </div>
      ) : null}
      {message ? <p className="settings-message">{message}</p> : null}
    </section>
  );
}
