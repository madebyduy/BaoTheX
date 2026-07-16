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
  velocity_6h: number;
  controversy_score: number;
  action_score: number;
  heat_terms: string[];
  picked_for_date?: string;
  status: string;
  draft_content_id?: number;
  conflicts: string[];
  open_questions: string[];
  last_error?: string;
};

// Vietnam time, which is where the newsroom day starts and ends regardless of
// the reader's browser timezone.
function newsroomToday(): string {
  return new Date().toLocaleDateString("en-CA", { timeZone: "Asia/Ho_Chi_Minh" });
}

function isTodaysPick(item: Candidate): boolean {
  return Boolean(item.picked_for_date?.startsWith(newsroomToday()));
}

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

  const pick = items.find(isTodaysPick);
  const rest = items.filter((item) => !isTodaysPick(item));

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
      {pick ? (
        <div className="daily-pick">
          <div className="daily-pick-head">
            <b>Chủ đề nóng nhất hôm nay</b>
            <span>Một chủ đề mỗi ngày · toàn bộ quota LLM dồn vào đây</span>
          </div>
          <CandidateCard
            item={pick}
            featured
            onEdit={openEditor}
            onAction={action}
            onPublish={publish}
          />
        </div>
      ) : (
        <p className="daily-pick-empty">
          Hôm nay chưa chốt chủ đề. Hệ thống quét và chọn vào cuối ngày; nếu không chủ đề nào
          đủ nóng thì sẽ không viết bài — đó là chủ ý, không phải lỗi.
        </p>
      )}

      {rest.length ? <h3 className="candidate-rest-head">Các chủ đề khác đang theo dõi</h3> : null}
      {rest.map((item) => (
        <CandidateCard
          key={item.id}
          item={item}
          onEdit={openEditor}
          onAction={action}
          onPublish={publish}
        />
      ))}
      {!items.length && !message ? <p>Chưa có cluster đạt ngưỡng đề cử.</p> : null}
    </section>
  );
}

function CandidateCard({
  item,
  featured = false,
  onEdit,
  onAction,
  onPublish,
}: {
  item: Candidate;
  featured?: boolean;
  onEdit: (contentID: number, clusterID: number) => Promise<void>;
  onAction: (clusterID: number, name: "generate" | "dismiss") => Promise<void>;
  onPublish: (clusterID: number) => Promise<void>;
}) {
  return (
    <article className={featured ? "candidate-card featured" : "candidate-card"}>
      <div className="candidate-score">
        <strong>{Math.round(item.score)}</strong>
        <span>ĐỘ NÓNG</span>
      </div>
      <div className="candidate-copy">
        <small>{item.status.replaceAll("_", " ").toUpperCase()}</small>
        <h2>{item.representative_title}</h2>
        <div className="candidate-metrics">
          <span>{item.source_count} nguồn độc lập</span>
          <span>{item.high_quality_sources} nguồn uy tín</span>
          <span>{item.velocity_6h} bài/6h</span>
          <span>{item.velocity_24h} bài/24h</span>
        </div>
        {/* Show the words that made this rank, so an editor can judge the pick
            instead of trusting an unexplained score. */}
        {item.heat_terms?.length ? (
          <div className="heat-terms">
            <b>{item.controversy_score > 0 ? "Dấu hiệu tranh cãi:" : "Dấu hiệu diễn biến:"}</b>
            {item.heat_terms.map((term) => (
              <span className="heat-term" key={term}>
                {term}
              </span>
            ))}
          </div>
        ) : null}
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
            <button className="quiet" onClick={() => void onEdit(item.draft_content_id!, item.cluster_id)}>
              Đọc & sửa bản nháp
            </button>
            {item.status === "needs_review" ? (
              <>
                <button className="quiet" onClick={() => void onAction(item.cluster_id, "generate")}>
                  Tạo lại bản đầy đủ
                </button>
                <button onClick={() => void onPublish(item.cluster_id)}>Duyệt nhanh</button>
              </>
            ) : null}
          </>
        ) : (
          <button onClick={() => void onAction(item.cluster_id, "generate")}>Tạo bản nháp</button>
        )}
        {item.status !== "published" ? (
          <button className="quiet" onClick={() => void onAction(item.cluster_id, "dismiss")}>
            Bỏ qua
          </button>
        ) : null}
      </div>
    </article>
  );
}
