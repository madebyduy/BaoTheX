"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { Item, Source } from "../lib";
import { AnalysisDesk } from "./goc-nhin/analysis-desk";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

type User = { email: string; display_name?: string; role: string };
type Job = {
  id: number;
  kind: string;
  status: string;
  attempts: number;
  max_attempts: number;
  last_error?: string;
  created_at: string;
};
type Stat = { kind: string; status: string; count: number };
type LLMUsage = {
  spend_today_usd: number;
  daily_budget_usd: number;
  calls_today: number;
  calls_last_hour: number;
  max_calls_per_hour: number;
  input_tokens_today: number;
  output_tokens_today: number;
  model: string;
};
type View = "analysis" | "overview" | "content" | "sources" | "jobs";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API}/api/v1${path}`, {
    credentials: "include",
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers || {}) },
  });
  if (!response.ok) {
    const body = await response.json().catch(() => null);
    throw new Error(body?.error?.message || `Yêu cầu thất bại (${response.status})`);
  }
  const json = await response.json();
  return (json.data ?? json) as T;
}

export function AdminConsole({ initialView = "analysis" }: { initialView?: View }) {
  const [view, setView] = useState<View>(initialView);
  const [me, setMe] = useState<User | null | undefined>(undefined);
  const [content, setContent] = useState<Item[]>([]);
  const [sources, setSources] = useState<Source[]>([]);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [stats, setStats] = useState<Stat[]>([]);
  const [usage, setUsage] = useState<LLMUsage | null>(null);
  const [review, setReview] = useState<Item[]>([]);
  const [notableOnly, setNotableOnly] = useState(false);
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState<string | null>(null);

  const load = useCallback(async () => {
    setMessage("");
    try {
      const user = await request<User | null>("/auth/me");
      setMe(user);
      if (!user || user.role !== "admin") return;
      // The review queue is fetched separately: only needs_review, ordered by
      // notability server-side, and deep enough that older items never scroll off.
      const reviewQuery = notableOnly ? "&min_score=40" : "";
      const [items, reviewItems, sourceItems, jobItems, statItems, usageItem] = await Promise.all([
        request<Item[]>("/admin/content?per_page=30"),
        request<Item[]>(`/admin/content?needs_review=true&per_page=100${reviewQuery}`),
        request<Source[]>("/admin/sources"),
        request<Job[]>("/admin/jobs?per_page=40"),
        request<Stat[]>("/admin/jobs/stats"),
        request<LLMUsage>("/admin/llm-usage"),
      ]);
      setContent(items);
      setReview(reviewItems);
      setSources(sourceItems);
      setJobs(jobItems);
      setStats(statItems);
      setUsage(usageItem);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Không thể tải dữ liệu quản trị.");
    }
  }, [notableOnly]);

  useEffect(() => {
    void load();
  }, [load]);

  async function act(key: string, path: string, init: RequestInit) {
    setBusy(key);
    setMessage("");
    try {
      await request(path, init);
      setMessage("Đã cập nhật thành công.");
      await load();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Không thể cập nhật.");
    } finally {
      setBusy(null);
    }
  }

  const totals = useMemo(() => {
    const total = (status: string) =>
      stats.filter((row) => row.status === status).reduce((sum, row) => sum + row.count, 0);
    return {
      review: review.length,
      pending: total("pending"),
      running: total("running"),
    };
  }, [review, stats]);

  const recentFailedJobs = useMemo(() => {
    const since = Date.now() - 24 * 60 * 60 * 1000;
    return jobs.filter(
      (job) =>
        ["dead", "failed"].includes(job.status) && new Date(job.created_at).getTime() >= since,
    );
  }, [jobs]);

  if (me === undefined) return <div className="admin-state">Đang kiểm tra phiên quản trị…</div>;
  if (!me) {
    return (
      <div className="admin-state">
        <span>YÊU CẦU ĐĂNG NHẬP</span>
        <h2>Đăng nhập để mở phòng biên tập</h2>
        <p>Tài khoản phải được cấp vai trò quản trị viên.</p>
        <Link className="btn ember" href="/dang-nhap">
          Đăng nhập
        </Link>
      </div>
    );
  }
  if (me.role !== "admin") {
    return (
      <div className="admin-state danger">
        <span>KHÔNG ĐỦ QUYỀN</span>
        <h2>Tài khoản này chưa phải quản trị viên</h2>
        <p>Dùng công cụ seed admin để cấp quyền cho {me.email}.</p>
      </div>
    );
  }

  return (
    <section className="admin-shell">
      <header className="admin-toolbar">
        <div>
          <span>PHÒNG BIÊN TẬP BAOTHEX</span>
          <strong>{me.display_name || me.email}</strong>
        </div>
        <button onClick={() => void load()}>Làm mới dữ liệu</button>
      </header>

      <nav className="admin-tabs" aria-label="Khu vực quản trị">
        {(
          [
            ["analysis", "Góc nhìn chờ duyệt"],
            ["content", "Nội dung"],
            ["sources", "Nguồn tin"],
            ["jobs", "Tác vụ nền"],
            ["overview", "Hệ thống"],
          ] as const
        ).map(([key, label]) => (
          <button className={view === key ? "active" : ""} onClick={() => setView(key)} key={key}>
            {label}
          </button>
        ))}
        <Link className="admin-tab-cta" href="/admin/goc-nhin">
          Mở bàn phân tích đầy đủ →
        </Link>
      </nav>

      {message ? <div className="admin-message">{message}</div> : null}

      {view === "analysis" ? (
        <div className="admin-editorial-view">
          <AnalysisDesk focus="review" />
        </div>
      ) : null}

      {view === "overview" ? (
        <>
          <div className="admin-metrics">
            <Metric value={totals.review} label="Tin thường chờ duyệt" tone="orange" />
            <Metric value={totals.pending} label="Đang chờ chạy" tone="blue" />
            <Metric value={totals.running} label="Đang xử lý" tone="green" />
            <Metric value={recentFailedJobs.length} label="Lỗi mới trong 24 giờ" tone="red" />
          </div>
          <div className="admin-overview-grid">
            <div className="admin-panel">
              <PanelTitle eyebrow="HÀNG CHỜ" title="Nội dung cần quyết định" />
              <ContentRows items={review.slice(0, 6)} busy={busy} act={act} />
            </div>
            <div className="admin-panel">
              <PanelTitle eyebrow="SỨC KHỎE HỆ THỐNG" title="Tác vụ cần chú ý" />
              <JobRows jobs={recentFailedJobs.slice(0, 7)} busy={busy} act={act} />
            </div>
          </div>
          {usage ? <LLMUsagePanel usage={usage} /> : null}
        </>
      ) : null}

      {view === "content" ? (
        <div className="admin-panel">
          <PanelTitle eyebrow="KIỂM DUYỆT" title="Danh sách nội dung" />
          <p className="admin-queue-note">
            {review.length} bai chua duyet · sap theo do nong va do tin cay. Bai khong bi xoa khi co
            dot cao moi.
          </p>
          <div className="admin-queue-filter">
            <button
              className={notableOnly ? "active" : ""}
              onClick={() => setNotableOnly((value) => !value)}
              type="button"
            >
              {notableOnly ? "Äang hiá»‡n tin ná»•i báº­t" : "Chá»‰ hiá»‡n tin ná»•i báº­t"}
            </button>
          </div>
          <ContentRows items={review} busy={busy} act={act} />
        </div>
      ) : null}

      {view === "sources" ? (
        <div className="admin-panel">
          <PanelTitle eyebrow="INGESTION" title={`${sources.length} nguồn dữ liệu`} />
          <div className="admin-source-grid">
            {sources.map((source) => (
              <article className="admin-source" key={source.id}>
                <div>
                  <span>
                    {source.kind || "nguồn"} · {source.default_lang || "—"}
                  </span>
                  <h3>{source.name}</h3>
                  <p>
                    Uy tín {source.quality || 0}/5 · {source.enabled ? "Đang hoạt động" : "Đã tắt"}
                  </p>
                </div>
                <div className="admin-actions">
                  <button
                    disabled={busy === `source-${source.id}`}
                    onClick={() =>
                      void act(`source-${source.id}`, `/admin/sources/${source.id}/fetch`, {
                        method: "POST",
                      })
                    }
                  >
                    Quét ngay
                  </button>
                  <button
                    className={source.enabled ? "danger" : "success"}
                    disabled={busy === `toggle-${source.id}`}
                    onClick={() =>
                      void act(`toggle-${source.id}`, `/admin/sources/${source.id}`, {
                        method: "PATCH",
                        body: JSON.stringify({ enabled: !source.enabled }),
                      })
                    }
                  >
                    {source.enabled ? "Tạm dừng" : "Bật nguồn"}
                  </button>
                </div>
              </article>
            ))}
          </div>
        </div>
      ) : null}

      {view === "jobs" ? (
        <div className="admin-panel">
          <PanelTitle eyebrow="WORKER" title="Nhật ký tác vụ nền" />
          <JobRows jobs={jobs} busy={busy} act={act} />
        </div>
      ) : null}
    </section>
  );
}

function LLMUsagePanel({ usage }: { usage: LLMUsage }) {
  const spendPct =
    usage.daily_budget_usd > 0
      ? Math.min(100, (usage.spend_today_usd / usage.daily_budget_usd) * 100)
      : 0;
  const callPct =
    usage.max_calls_per_hour > 0
      ? Math.min(100, (usage.calls_last_hour / usage.max_calls_per_hour) * 100)
      : 0;
  const color = (pct: number) => (pct >= 90 ? "#e0453a" : pct >= 70 ? "#e0a100" : "#12915f");
  return (
    <div className="admin-panel" style={{ marginTop: 20 }}>
      <PanelTitle eyebrow="QUOTA & CHI PHÍ LLM" title={`Hôm nay · ${usage.model}`} />
      <div style={{ display: "grid", gap: 18 }}>
        <Meter
          label="Lượt gọi trong giờ (trần tự đặt)"
          value={`${usage.calls_last_hour} / ${usage.max_calls_per_hour}`}
          pct={callPct}
          barColor={color(callPct)}
          note="Chạm trần này là job báo “budget exceeded”. Nới trong .env: LLM_MAX_CALLS_PER_HOUR."
        />
        <Meter
          label="Chi phí hôm nay"
          value={`$${usage.spend_today_usd.toFixed(4)} / $${usage.daily_budget_usd.toFixed(2)}`}
          pct={spendPct}
          barColor={color(spendPct)}
        />
        <div style={{ display: "flex", gap: 24, flexWrap: "wrap", fontSize: 13, opacity: 0.8 }}>
          <span>
            Tổng lượt gọi hôm nay: <b>{usage.calls_today}</b>
          </span>
          <span>
            Token vào: <b>{usage.input_tokens_today.toLocaleString()}</b>
          </span>
          <span>
            Token ra: <b>{usage.output_tokens_today.toLocaleString()}</b>
          </span>
        </div>
      </div>
    </div>
  );
}

function Meter({
  label,
  value,
  pct,
  barColor,
  note,
}: {
  label: string;
  value: string;
  pct: number;
  barColor: string;
  note?: string;
}) {
  return (
    <div>
      <div
        style={{ display: "flex", justifyContent: "space-between", fontSize: 13, marginBottom: 6 }}
      >
        <span>{label}</span>
        <b style={{ fontVariantNumeric: "tabular-nums" }}>{value}</b>
      </div>
      <div
        style={{
          height: 8,
          borderRadius: 999,
          background: "rgba(128,128,128,.2)",
          overflow: "hidden",
        }}
      >
        <div style={{ width: `${pct}%`, height: "100%", background: barColor }} />
      </div>
      {note ? <p style={{ fontSize: 12, opacity: 0.6, margin: "6px 0 0" }}>{note}</p> : null}
    </div>
  );
}

function Metric({ value, label, tone }: { value: number; label: string; tone: string }) {
  return (
    <div className={`admin-metric ${tone}`}>
      <strong>{String(value).padStart(2, "0")}</strong>
      <span>{label}</span>
    </div>
  );
}

function PanelTitle({ eyebrow, title }: { eyebrow: string; title: string }) {
  return (
    <div className="admin-panel-title">
      <span>{eyebrow}</span>
      <h2>{title}</h2>
    </div>
  );
}

function ContentRows({
  items,
  busy,
  act,
}: {
  items: Item[];
  busy: string | null;
  act: (key: string, path: string, init: RequestInit) => Promise<void>;
}) {
  if (!items.length) return <p className="admin-empty">Không có nội dung trong hàng chờ.</p>;
  return (
    <div className="admin-list">
      {items.map((item) => (
        <article className="admin-row" key={item.id}>
          <div className="admin-row-copy">
            <span>
              {item.status || "—"} · {item.source_name || "BaoTheX"}
            </span>
            <Link href={`/admin/preview/${item.id}`} target="_blank">
              {item.title}
            </Link>
            <p>{item.summary || item.excerpt || "Chưa có tóm tắt."}</p>
          </div>
          <div className="admin-actions">
            <Link
              className="admin-preview-button"
              href={`/admin/preview/${item.id}`}
              target="_blank"
            >
              Xem trước
            </Link>
            <button
              className="success"
              disabled={busy === `publish-${item.id}`}
              onClick={() =>
                void act(`publish-${item.id}`, `/admin/content/${item.id}`, {
                  method: "PATCH",
                  body: JSON.stringify({ status: "ready" }),
                })
              }
            >
              Xuất bản
            </button>
            <button
              disabled={busy === `feature-${item.id}`}
              onClick={() =>
                void act(`feature-${item.id}`, `/admin/content/${item.id}/highlight`, {
                  method: "POST",
                  body: JSON.stringify({ boost: 30 }),
                })
              }
            >
              Đẩy nổi bật
            </button>
            <button
              className="danger"
              disabled={busy === `hide-${item.id}`}
              onClick={() =>
                void act(`hide-${item.id}`, `/admin/content/${item.id}/hide`, { method: "POST" })
              }
            >
              Ẩn
            </button>
          </div>
        </article>
      ))}
    </div>
  );
}

function JobRows({
  jobs,
  busy,
  act,
}: {
  jobs: Job[];
  busy: string | null;
  act: (key: string, path: string, init: RequestInit) => Promise<void>;
}) {
  if (!jobs.length) return <p className="admin-empty">Không có tác vụ cần hiển thị.</p>;
  return (
    <div className="admin-list compact">
      {jobs.map((job) => (
        <article className="admin-row" key={job.id}>
          <div className="admin-row-copy">
            <span>
              #{job.id} · {job.status}
            </span>
            <strong>{job.kind}</strong>
            <p>{job.last_error || `Lần chạy ${job.attempts}/${job.max_attempts}`}</p>
          </div>
          {["dead", "failed"].includes(job.status) ? (
            <div className="admin-actions">
              <button
                disabled={busy === `retry-${job.id}`}
                onClick={() =>
                  void act(`retry-${job.id}`, `/admin/jobs/${job.id}/retry`, { method: "POST" })
                }
              >
                Chạy lại
              </button>
            </div>
          ) : null}
        </article>
      ))}
    </div>
  );
}
