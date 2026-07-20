import type { Metadata } from "next";
import { api, demoItems, type Entity, type Item, pageMetadata } from "../../lib";
import { Footer, ItemGrid, PageTitle } from "../../ui";
import { FollowButton } from "../../action-buttons";

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const data = await api<{ entity?: Entity }>(`/entities/${slug}`, {});
  const name = data.entity?.name || slug.replaceAll("-", " ");
  return pageMetadata({
    title: `${name} — Hồ sơ & tin mới`,
    description:
      data.entity?.bio ||
      `Tiểu sử, thành tích và tin tức mới nhất về ${name} được BaoTheX tổng hợp từ nguồn chính thức.`,
    path: `/nhan-vat/${slug}`,
  });
}

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
