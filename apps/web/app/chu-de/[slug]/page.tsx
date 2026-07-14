import { api, demoItems, demoTopics, type Item } from "../../lib";
import { Footer, ItemGrid, PageTitle } from "../../ui";
import { FollowButton } from "../../action-buttons";
export default async function Page({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const demo = demoTopics.find((t) => t.slug === slug) || demoTopics[0];
  const data = await api<{ topic?: { name: string; description?: string }; featured?: Item[] }>(
    `/topics/${slug}`,
    {},
  );
  const items = data.featured?.length ? data.featured : demoItems;
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
          <ItemGrid items={items} />
        </section>
      </main>
      <Footer />
    </>
  );
}
