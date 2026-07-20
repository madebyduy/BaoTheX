import type { Metadata } from "next";
import Link from "next/link";
import { api, articleHref, pageMetadata, type StoryCluster } from "../../lib";
import { Footer, RemoteImage } from "../../ui";
import { TruthCenterActions } from "./truth-actions";

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  const cluster = await api<StoryCluster | null>(`/clusters/${id}`, null);
  if (!cluster) return { title: "Không tìm thấy sự kiện" };
  return pageMetadata({
    title: cluster.representative_title,
    description: `Toàn cảnh sự kiện qua ${cluster.source_count} nguồn: ${cluster.representative_title}. Đối chiếu và kiểm chứng trên BaoTheX.`,
    path: `/su-kien/${id}`,
  });
}

export default async function StoryPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const cluster = await api<StoryCluster | null>(`/clusters/${id}`, null);

  if (!cluster) {
    return (
      <>
        <main className="wrap cluster-page cluster-empty">
          <span className="tag">MỘT SỰ KIỆN · MỌI NGUỒN</span>
          <h1>Chưa tìm thấy cụm tin này</h1>
          <Link href="/">Trở về trang chủ →</Link>
        </main>
        <Footer />
      </>
    );
  }

  const primary =
    cluster.items?.find((item) => item.id === cluster.primary_content_id) || cluster.items?.[0];
  const timeline = [...(cluster.items || [])].sort(
    (a, b) => new Date(a.published_at || 0).getTime() - new Date(b.published_at || 0).getTime(),
  );

  return (
    <>
      <main className="wrap cluster-page">
        <header className="cluster-hero">
          <div>
            <span className="tag">MỘT SỰ KIỆN · MỌI NGUỒN</span>
            <h1>{cluster.representative_title}</h1>
            <p>
              BaoTheX gom các bài cùng nói về một sự kiện để bạn đọc nhanh, đối chiếu nguồn và tránh
              bị dẫn dắt bởi một tiêu đề duy nhất.
            </p>
          </div>
          <div className="cluster-meter">
            <strong>{cluster.source_count.toString().padStart(2, "0")}</strong>
            <span>nguồn đưa tin</span>
            <b className={`signal ${cluster.verification_status}`}>
              {verificationLabel(cluster.verification_status)}
            </b>
          </div>
        </header>

        <TruthCenterActions clusterId={cluster.id} updatedAt={cluster.updated_at} />

        <section className="truth-status-grid">
          <article className={cluster.verification_status === "confirmed" ? "active" : ""}>
            <span>01</span>
            <h3>Đã xác nhận</h3>
            <p>
              {cluster.verification_status === "confirmed"
                ? "Câu chuyện đã có đủ tín hiệu xác nhận theo quy tắc biên tập."
                : "Chưa đạt ngưỡng xác nhận."}
            </p>
          </article>
          <article className={cluster.source_count >= 2 ? "active" : ""}>
            <span>02</span>
            <h3>Các nguồn đồng thuận</h3>
            <p>
              {cluster.source_count >= 2
                ? `${cluster.source_count} nguồn đang cùng đưa tin về diễn biến cốt lõi.`
                : "Hiện mới có một nguồn trong cụm."}
            </p>
          </article>
          <article className={cluster.verification_status === "verifying" ? "active warning" : ""}>
            <span>03</span>
            <h3>Còn mâu thuẫn</h3>
            <p>
              {cluster.verification_status === "verifying"
                ? "Một số chi tiết vẫn cần biên tập viên đối chiếu thêm."
                : "Metadata hiện chưa phát hiện tín hiệu mâu thuẫn nổi bật."}
            </p>
          </article>
          <article className={cluster.verification_status === "rumor" ? "active warning" : ""}>
            <span>04</span>
            <h3>Chưa đủ dữ kiện</h3>
            <p>
              {cluster.verification_status === "rumor"
                ? "Không nên xem đây là thông tin đã được xác nhận."
                : "Đã có nhiều hơn tín hiệu ban đầu."}
            </p>
          </article>
        </section>

        {primary ? (
          <section className="cluster-summary">
            <span>TÓM TẮT TRUNG LẬP</span>
            <h2>{primary.title}</h2>
            <p>{primary.summary || primary.excerpt || "Nội dung đang được tổng hợp."}</p>
            <Link href={articleHref(primary)}>Đọc bản chi tiết →</Link>
          </section>
        ) : null}

        <section className="truth-timeline">
          <div className="section-heading">
            <div>
              <span className="tag">DÒNG THỜI GIAN</span>
              <h2>Câu chuyện đã thay đổi thế nào</h2>
            </div>
          </div>
          <div>
            {timeline.map((item, index) => (
              <article key={item.id}>
                <time>
                  {item.published_at
                    ? new Intl.DateTimeFormat("vi-VN", {
                        hour: "2-digit",
                        minute: "2-digit",
                        day: "2-digit",
                        month: "2-digit",
                      }).format(new Date(item.published_at))
                    : "Chưa rõ giờ"}
                </time>
                <i />
                <section>
                  <small>
                    {index === 0 ? "Nguồn đầu tiên trong cụm" : `Cập nhật thứ ${index + 1}`}
                  </small>
                  <strong>{item.source_name || "BaoTheX"}</strong>
                  <p>{item.title}</p>
                </section>
              </article>
            ))}
          </div>
        </section>

        <section className="truth-compare">
          <div className="section-heading">
            <div>
              <span className="tag">SO SÁNH NHANH</span>
              <h2>Mỗi nguồn nhấn vào điều gì</h2>
            </div>
          </div>
          <div className="truth-table">
            <div className="truth-table-head">
              <b>Nguồn</b>
              <b>Tiêu đề / góc tiếp cận</b>
              <b>Thời gian</b>
              <b>Chất lượng</b>
            </div>
            {(cluster.items || []).map((item) => (
              <div className="truth-table-row" key={item.id}>
                <strong>{item.source_name || "BaoTheX"}</strong>
                <Link href={articleHref(item)}>{item.title}</Link>
                <span>
                  {item.published_at
                    ? new Intl.DateTimeFormat("vi-VN", {
                        hour: "2-digit",
                        minute: "2-digit",
                      }).format(new Date(item.published_at))
                    : "—"}
                </span>
                <span>{item.source_quality ? `${item.source_quality}/5` : "Chưa chấm"}</span>
              </div>
            ))}
          </div>
          <p className="truth-method">
            Phân nhóm trên dùng metadata, số nguồn, chất lượng nguồn và quy tắc biên tập. BaoTheX
            không dùng câu chữ “đã xác nhận” khi dữ kiện chưa đạt ngưỡng.
          </p>
        </section>

        <section className="cluster-sources">
          <div className="section-heading">
            <div>
              <span className="tag">ĐỐI CHIẾU NGUỒN</span>
              <h2>Các bài viết về sự kiện này</h2>
            </div>
          </div>
          <div className="cluster-source-grid">
            {(cluster.items || []).map((item) => (
              <article className="cluster-source-card" key={item.id}>
                {item.image_url ? (
                  <RemoteImage
                    src={item.image_url}
                    alt=""
                    loading="lazy"
                    decoding="async"
                    referrerPolicy="no-referrer"
                  />
                ) : (
                  <div className="cluster-image">BX</div>
                )}
                <div>
                  <div className="source-heading">
                    <span>{item.source_name || "BaoTheX"}</span>
                    {(item.source_quality || 0) >= 4 ? (
                      <b>Nguồn uy tín · {item.source_quality}/5</b>
                    ) : null}
                  </div>
                  <h3>{item.title}</h3>
                  <p>{item.summary || item.excerpt || "Mở bài để xem nội dung đầy đủ."}</p>
                  <Link href={articleHref(item)}>Đọc bài →</Link>
                </div>
              </article>
            ))}
          </div>
        </section>
      </main>
      <Footer />
    </>
  );
}

function verificationLabel(value: string) {
  if (value === "confirmed") return "Đã xác nhận";
  if (value === "verifying") return "Đang xác minh";
  return "Tin đồn / một nguồn";
}
