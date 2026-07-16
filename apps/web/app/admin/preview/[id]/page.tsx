import { cookies } from "next/headers";
import Link from "next/link";
import { notFound } from "next/navigation";
import { apiWithCookie, type ContentBody, type Item, typeLabel } from "../../../lib";

type AdminDetail = {
  item?: Item;
  body?: ContentBody;
};

function paragraphs(text: string) {
  return text
    .split(/\n{2,}/)
    .map((part) => part.trim())
    .filter(Boolean);
}

export default async function AdminPreviewPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const cookieHeader = (await cookies()).toString();
  const data = await apiWithCookie<AdminDetail>(`/admin/content/${id}`, {}, cookieHeader);
  if (!data.item) notFound();

  const item = data.item;
  const body = data.body;
  const readableBody =
    body?.vietnamese_body?.trim() ||
    body?.original_body?.trim() ||
    item.summary?.trim() ||
    item.excerpt?.trim() ||
    "";
  const sourceURL =
    item.canonical_url && /^https?:\/\//i.test(item.canonical_url) ? item.canonical_url : "";

  return (
    <main className="admin-preview-page">
      <div className="admin-preview-shell">
        <div className="admin-preview-bar">
          <Link href="/admin">← Quay lại admin</Link>
          <span>{item.status || "draft"}</span>
        </div>

        <article className="admin-preview-article">
          <div className="admin-preview-kicker">
            <span>{typeLabel(item.type)}</span>
            <span>{item.source_name || "BaoTheX"}</span>
            {item.language ? <span>{item.language.toUpperCase()}</span> : null}
          </div>

          <h1>{item.title}</h1>

          {item.summary || item.excerpt ? (
            <p className="admin-preview-lede">{item.summary || item.excerpt}</p>
          ) : null}

          {item.image_url ? (
            <img className="admin-preview-image" src={item.image_url} alt="" />
          ) : null}

          {item.key_points?.length ? (
            <section className="admin-preview-box">
              <h2>Điểm chính</h2>
              <ol>
                {item.key_points.map((point) => (
                  <li key={point}>{point}</li>
                ))}
              </ol>
            </section>
          ) : null}

          <section className="admin-preview-body">
            <h2>Nội dung xem trước</h2>
            {readableBody ? (
              paragraphs(readableBody).map((part) => <p key={part}>{part}</p>)
            ) : (
              <p>Bài này chưa có thân bài. Nên giữ ở hàng chờ và chạy lại bước biên tập.</p>
            )}
          </section>

          <div className="admin-preview-footer">
            {sourceURL ? (
              <a href={sourceURL} target="_blank" rel="noreferrer">
                Mở nguồn gốc ↗
              </a>
            ) : null}
            <Link href={`/noi-dung/${item.id}`}>Trang public</Link>
          </div>
        </article>
      </div>
    </main>
  );
}
