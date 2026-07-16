"use client";

import { useCallback, useEffect, useState } from "react";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";
type Candidate = {
  id: number;
  cluster_id: number;
  representative_title: string;
  score: number;
  source_count: number;
  high_quality_sources: number;
  velocity_24h: number;
  status: string;
  draft_content_id?: number;
  conflicts: string[];
  open_questions: string[];
  last_error?: string;
};

export function AnalysisDesk() {
  const [items, setItems] = useState<Candidate[]>([]);
  const [message, setMessage] = useState("");
  const [editor, setEditor] = useState<{
    id: number;
    clusterID: number;
    title: string;
    summary: string;
    body: string;
  } | null>(null);
  const load = useCallback(async () => {
    const response = await fetch(`${API}/api/v1/admin/analysis-candidates`, {
      credentials: "include",
    });
    if (!response.ok) throw new Error("Bạn cần đăng nhập bằng tài khoản quản trị.");
    const json = await response.json();
    setItems(json.data ?? json);
  }, []);
  useEffect(() => {
    void load().catch((error: Error) => setMessage(error.message));
  }, [load]);

  async function action(clusterID: number, name: "generate" | "dismiss") {
    setMessage("Đang xử lý…");
    const response = await fetch(`${API}/api/v1/admin/analysis-candidates/${clusterID}/${name}`, {
      method: "POST",
      credentials: "include",
    });
    if (!response.ok) {
      setMessage("Không thể thực hiện. Kiểm tra trạng thái job hoặc quyền quản trị.");
      return;
    }
    setMessage(name === "generate" ? "Đã đưa vào hàng chờ viết nháp." : "Đã bỏ qua đề cử.");
    await load();
  }

  // publishCandidate flips the reviewed draft to ready AND marks the candidate
  // published, so the piece appears in the public "Góc nhìn" section.
  async function publishCandidate(clusterID: number): Promise<boolean> {
    const response = await fetch(`${API}/api/v1/admin/analysis-candidates/${clusterID}/publish`, {
      method: "POST",
      credentials: "include",
    });
    if (!response.ok) {
      setMessage("Không thể xuất bản. Bản nháp phải đang ở trạng thái chờ duyệt.");
      return false;
    }
    return true;
  }

  async function publish(clusterID: number) {
    if (!window.confirm("Xác nhận bạn đã đọc, sửa và chịu trách nhiệm xuất bản bài này?")) return;
    if (await publishCandidate(clusterID)) {
      setMessage("Bài đã lên mục Góc nhìn.");
      await load();
    }
  }

  async function openEditor(contentID: number, clusterID: number) {
    const response = await fetch(`${API}/api/v1/admin/content/${contentID}`, {
      credentials: "include",
    });
    if (!response.ok) return setMessage("Không thể mở bản nháp.");
    const json = await response.json();
    const data = json.data ?? json;
    setEditor({
      id: contentID,
      clusterID,
      title: data.item.title || "",
      summary: data.item.summary || "",
      body: data.body?.vietnamese_body || data.body?.original_body || "",
    });
  }

  async function saveEditor(publishNow: boolean) {
    if (!editor) return;
    // Always persist edits as a draft first, so the published version reflects
    // the editor's changes.
    const response = await fetch(`${API}/api/v1/admin/content/${editor.id}`, {
      method: "PATCH",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        title: editor.title,
        summary: editor.summary,
        body: editor.body,
        status: "needs_review",
      }),
    });
    if (!response.ok) {
      setMessage("Không thể lưu bài.");
      return;
    }
    if (publishNow && !(await publishCandidate(editor.clusterID))) return;
    setMessage(publishNow ? "Bài đã lên mục Góc nhìn." : "Đã lưu bản biên tập.");
    setEditor(null);
    await load();
  }

  return (
    <section className="analysis-desk">
      <div className="analysis-desk-note">
        <b>Cổng biên tập bắt buộc</b>
        <span>Đề cử → Trích xuất luận điểm → Viết nháp → Biên tập viên duyệt → Xuất bản</span>
      </div>
      {message ? <p className="analysis-message">{message}</p> : null}
      {editor ? (
        <div className="editor-panel">
          <div>
            <b>Biên tập bài Góc nhìn</b>
            <button className="quiet" onClick={() => setEditor(null)}>
              Đóng
            </button>
          </div>
          <label>
            Tiêu đề
            <input
              value={editor.title}
              onChange={(event) => setEditor({ ...editor, title: event.target.value })}
            />
          </label>
          <label>
            Tóm tắt
            <textarea
              rows={4}
              value={editor.summary}
              onChange={(event) => setEditor({ ...editor, summary: event.target.value })}
            />
          </label>
          <label>
            Nội dung
            <textarea
              rows={22}
              value={editor.body}
              onChange={(event) => setEditor({ ...editor, body: event.target.value })}
            />
          </label>
          <div className="editor-actions">
            <button className="quiet" onClick={() => void saveEditor(false)}>
              Lưu bản nháp
            </button>
            <button onClick={() => void saveEditor(true)}>Ký duyệt & xuất bản</button>
          </div>
        </div>
      ) : null}
      {items.map((item) => (
        <article className="candidate-card" key={item.id}>
          <div className="candidate-score">
            <strong>{Math.round(item.score)}</strong>
            <span>ĐIỂM</span>
          </div>
          <div className="candidate-copy">
            <small>{item.status.replaceAll("_", " ").toUpperCase()}</small>
            <h2>{item.representative_title}</h2>
            <div className="candidate-metrics">
              <span>{item.source_count} nguồn độc lập</span>
              <span>{item.high_quality_sources} nguồn uy tín</span>
              <span>{item.velocity_24h} bài/24h</span>
            </div>
            {item.conflicts?.length ? (
              <p>
                <b>Vênh nguồn:</b> {item.conflicts[0]}
              </p>
            ) : null}
            {item.open_questions?.length ? (
              <p>
                <b>Còn bỏ ngỏ:</b> {item.open_questions[0]}
              </p>
            ) : null}
            {item.last_error ? <p className="candidate-error">{item.last_error}</p> : null}
          </div>
          <div className="candidate-actions">
            {item.draft_content_id ? (
              <>
                <button
                  className="quiet"
                  onClick={() => void openEditor(item.draft_content_id!, item.cluster_id)}
                >
                  Đọc & sửa bản nháp
                </button>
                {item.status === "needs_review" ? (
                  <button
                    className="quiet"
                    onClick={() => void action(item.cluster_id, "generate")}
                  >
                    Tạo lại bản đầy đủ
                  </button>
                ) : null}
                {item.status === "needs_review" ? (
                  <button onClick={() => void publish(item.cluster_id)}>Duyệt nhanh</button>
                ) : null}
              </>
            ) : (
              <button onClick={() => void action(item.cluster_id, "generate")}>Tạo bản nháp</button>
            )}
            {item.status !== "published" ? (
              <button className="quiet" onClick={() => void action(item.cluster_id, "dismiss")}>
                Bỏ qua
              </button>
            ) : null}
          </div>
        </article>
      ))}
      {!items.length && !message ? <p>Chưa có cluster đạt ngưỡng đề cử.</p> : null}
    </section>
  );
}
