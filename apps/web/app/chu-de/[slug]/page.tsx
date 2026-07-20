import type { Metadata } from "next";
import { api, demoItems, demoTopics, pageMetadata, type Item } from "../../lib";
import { Footer, PageTitle } from "../../ui";
import { FollowButton } from "../../action-buttons";
import { LoadMore } from "../../load-more";

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const data = await api<{ topic?: { name: string; description?: string } }>(`/topics/${slug}`, {});
  const demo = demoTopics.find((t) => t.slug === slug);
  const name = data.topic?.name || demo?.name || slug.replaceAll("-", " ");
  return pageMetadata({
    title: `${name} — Tin thể thao`,
    description:
      data.topic?.description ||
      demo?.description ||
      `Tin mới nhất, video và phân tích về ${name.toLowerCase()} trên BaoTheX.`,
    path: `/chu-de/${slug}`,
  });
}

export default async function Page({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const demo = demoTopics.find((t) => t.slug === slug) || demoTopics[0];
  const listPath = `/content?topic=${encodeURIComponent(slug)}&sort=recent`;
  const [data, items] = await Promise.all([
    api<{ topic?: { name: string; description?: string }; featured?: Item[] }>(
      `/topics/${slug}`,
      {},
    ),
    api<Item[]>(`${listPath}&per_page=20`, demoItems),
  ]);
  const seed = items.length ? items : demoItems;
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Chủ đề"
          title={data.topic?.name || demo.name}
          description={data.topic?.description || demo.description}
        />
        <section className="section">
          <div className="section-head">
            <h2>Nội dung nổi bật</h2>
            <FollowButton topicId={demo.id} />
          </div>
          <LoadMore initial={seed} path={listPath} perPage={20} />
        </section>
      </main>
      <Footer />
    </>
  );
}
