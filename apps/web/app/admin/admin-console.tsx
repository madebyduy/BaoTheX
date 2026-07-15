"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { Item, Source } from "../lib";

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
type View = "overview" | "content" | "sources" | "jobs";

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

export function AdminConsole({ initialView = "overview" }: { initialView?: View }) {
  const [view, setView] = useState<View>(initialView);
  const [me, setMe] = useState<User | null | undefined>(undefined);
  const [content, setContent] = useState<Item[]>([]);
  const [sources, setSources] = useState<Source[]>([]);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [stats, setStats] = useState<Stat[]>([]);
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState<string | null>(null);

  const load = useCallback(async () => {
    setMessage("");
    try {
      const user = await request<User | null>("/auth/me");
      setMe(user);
      if (!user || user.role !== "admin") return;
      const [items, sourceItems, jobItems, statItems] = await Promise.all([
        request<Item[]>("/admin/content?per_page=30"),
        request<Source[]>("/admin/sources"),
        request<Job[]>("/admin/jobs?per_page=40"),
        request<Stat[]>("/admin/jobs/stats"),
      ]);
      setContent(items);
      setSources(sourceItems);
      setJobs(jobItems);
      setStats(statItems);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Không thể tải dữ liệu quản trị.");
    }
  }, []);

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
      review: content.filter((item) => item.status === "needs_review").length,
      pending: total("pending"),
      running: total("running"),
      dead: total("dead") + total("failed"),
    };
  }, [content, stats]);

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
            ["overview", "Tổng quan"],
            ["content", "Nội dung"],
            ["sources", "Nguồn tin"],
            ["jobs", "Tác vụ nền"],
          ] as const
        ).map(([key, label]) => (
          <button className={view === key ? "active" : ""} onClick={() => setView(key)} key={key}>
            {label}
          </button>
        ))}
        <Link href="/admin/goc-nhin">Bàn phân tích →</Link>
      </nav>

      {message ? <div className="admin-message">{message}</div> : null}

      {view === "overview" ? (
        <>
          <div className="admin-metrics">
            <Metric value={totals.review} label="Chờ biên tập" tone="orange" />
            <Metric value={totals.pending} label="Đang chờ chạy" tone="blue" />
            <Metric value={totals.running} label="Đang xử lý" tone="green" />
            <Metric value={totals.dead} label="Cần can thiệp" tone="red" />
          </div>
          <div className="admin-overview-grid">
            <div className="admin-panel">
              <PanelTitle eyebrow="HÀNG CHỜ" title="Nội dung cần quyết định" />
              <ContentRows items={content.slice(0, 6)} busy={busy} act={act} />
            </div>
            <div className="admin-panel">
              <PanelTitle eyebrow="SỨC KHỎE HỆ THỐNG" title="Tác vụ cần chú ý" />
              <JobRows
                jobs={jobs.filter((job) => ["dead", "failed"].includes(job.status)).slice(0, 7)}
                busy={busy}
                act={act}
              />
            </div>
          </div>
        </>
      ) : null}

      {view === "content" ? (
        <div className="admin-panel">
          <PanelTitle eyebrow="KIỂM DUYỆT" title="Danh sách nội dung" />
          <ContentRows items={content} busy={busy} act={act} />
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
            <Link href={`/noi-dung/${item.id}`}>{item.title}</Link>
            <p>{item.summary || item.excerpt || "Chưa có tóm tắt."}</p>
          </div>
          <div className="admin-actions">
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
