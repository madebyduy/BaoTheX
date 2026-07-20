import Link from "next/link";
import { api, pageMetadata, type Entity } from "../lib";
import { Footer, PageTitle } from "../ui";

const fallbackPeople: Entity[] =
  process.env.NODE_ENV === "production"
    ? []
    : [
        { id: 1, slug: "jeff-nippard", name: "Jeff Nippard", kind: "Nhà sáng tạo nội dung" },
        {
          id: 2,
          slug: "greg-nuckols",
          name: "Greg Nuckols",
          kind: "Nhà nghiên cứu & huấn luyện viên",
        },
        { id: 3, slug: "layne-norton", name: "Layne Norton", kind: "Nhà nghiên cứu dinh dưỡng" },
      ];

export const metadata = pageMetadata({
  title: "Nhân vật thể thao",
  description:
    "Hồ sơ cầu thủ, vận động viên và huấn luyện viên: tiểu sử, thành tích và tin mới nhất.",
  path: "/nhan-vat",
});

export default async function Page() {
  const people = await api<Entity[]>("/entities", fallbackPeople);
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Nhân vật & nguồn"
          title="Những người đáng theo dõi"
          description="Hồ sơ, nội dung mới và các nguồn chính thức được liên kết trong một nơi."
        />
        <div className="grid section">
          {people.map((person) => (
            <Link className="card" href={`/nhan-vat/${person.slug}`} key={person.id}>
              <span className="tag">{person.kind || "Nhân vật"}</span>
              <h3>{person.name}</h3>
              <p>
                {person.bio || "Các nội dung liên quan được BaoTheX tổng hợp và kiểm tra nguồn."}
              </p>
              <div className="meta">{person.follower_count || 0} người theo dõi · Xem hồ sơ →</div>
            </Link>
          ))}
        </div>
      </main>
      <Footer />
    </>
  );
}
