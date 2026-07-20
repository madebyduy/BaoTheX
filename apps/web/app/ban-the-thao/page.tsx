import { api, pageMetadata, type Item, type SportsEvent } from "../lib";
import { Footer, PageTitle } from "../ui";
import { SportsDesk } from "./sports-desk";

type CatchUp = { duration: number; items: Item[] };

export const metadata = pageMetadata({
  title: "Bàn thể thao",
  description: "Tổng hợp tin nóng, lịch thi đấu và video thể thao theo từng môn.",
  path: "/ban-the-thao",
});

export default async function SportsDeskPage() {
  const today = new Date().toISOString().slice(0, 10);
  const [events, catchUp] = await Promise.all([
    api<SportsEvent[]>(`/events?date=${today}&limit=12`, [], 20),
    api<CatchUp>("/catch-up?duration=3", { duration: 3, items: [] }, 30),
  ]);
  return (
    <>
      <main className="wrap sports-desk-page">
        <PageTitle
          eyebrow="MY SPORTS DESK"
          title="Bàn thể thao của riêng bạn"
          description="Kéo thả để sắp xếp, ẩn những phần không cần. Khách được lưu trên máy; khi đăng nhập, bố cục được đồng bộ với tài khoản."
        />
        <SportsDesk events={events} catchUp={catchUp.items} />
      </main>
      <Footer />
    </>
  );
}
