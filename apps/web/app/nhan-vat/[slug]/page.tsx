import { api, demoItems, type Entity, type Item } from "../../lib";
import { Footer, ItemGrid, PageTitle } from "../../ui";
import { FollowButton } from "../../action-buttons";

export default async function Page({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const data = await api<{ entity?: Entity; recent?: Item[] }>(`/entities/${slug}`, {});
  const entity = data.entity;
  const items = data.recent?.length ? data.recent : demoItems;
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow={entity?.kind || "Nhân vật"}
          title={entity?.name || slug.replaceAll("-", " ")}
          description={
            entity?.bio ||
            "Hồ sơ và các nội dung liên quan được BaoTheX tổng hợp từ nguồn chính thức."
          }
        />
        <section className="section">
          <div className="section-head">
            <div>
              <span className="tag">Hồ sơ nguồn</span>
              <h2>Nội dung liên quan</h2>
            </div>
            {entity?.id ? <FollowButton entityId={entity.id} /> : null}
          </div>
          <ItemGrid items={items} />
        </section>
      </main>
      <Footer />
    </>
  );
}
