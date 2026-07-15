import Link from "next/link";
import { api, type StoryCluster } from "../../lib";
import { Footer } from "../../ui";

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

        {primary ? (
          <section className="cluster-summary">
            <span>TÓM TẮT TRUNG LẬP</span>
            <h2>{primary.title}</h2>
            <p>{primary.summary || primary.excerpt || "Nội dung đang được tổng hợp."}</p>
            <Link href={`/noi-dung/${primary.id}`}>Đọc bản chi tiết →</Link>
          </section>
        ) : null}

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
                  <img src={item.image_url} alt="" />
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
                  <Link href={`/noi-dung/${item.id}`}>Đọc bài →</Link>
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
