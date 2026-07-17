import Link from "next/link";
import { api, type Sport, type SportsEvent } from "../../lib";
import { AdminSportsStudio } from "./studio";

export default async function AdminSportsPage() {
  const [sports, events] = await Promise.all([
    api<Sport[]>("/sports", []),
    api<SportsEvent[]>("/events?limit=20", [], 10),
  ]);
  return (
    <main className="wrap admin-sports-page">
      <div className="admin-page-head">
        <div>
          <span>BAOTHEX ADMIN</span>
          <h1>Event & Prediction Studio</h1>
          <p>
            Bổ sung giải Việt Nam hoặc môn chưa có nguồn miễn phí. Dữ liệu thủ công luôn được gắn
            nhãn và có thể khóa để provider không ghi đè.
          </p>
        </div>
        <Link href="/admin">← Admin</Link>
      </div>
      <AdminSportsStudio sports={sports} initialEvents={events} />
    </main>
  );
}
