import Link from "next/link";
import { api, pageMetadata, type Sport, type SportsEvent } from "../lib";
import { Footer, PageTitle } from "../ui";
import { SportsEventCard } from "../sports-event-card";

type Search = Promise<{ sport?: string; date?: string; status?: string }>;

export const metadata = pageMetadata({
  title: "Lịch thi đấu & kết quả",
  description:
    "Lịch thi đấu, tỷ số trực tiếp và kết quả các giải thể thao lớn, cập nhật liên tục có nguồn.",
  path: "/lich-the-thao",
});

export default async function SportsCalendarPage({ searchParams }: { searchParams: Search }) {
  const params = await searchParams;
  const today = new Date().toISOString().slice(0, 10);
  const date = /^\d{4}-\d{2}-\d{2}$/.test(params.date || "") ? params.date! : today;
  const query = new URLSearchParams({ date });
  if (params.sport) query.set("sport", params.sport);
  if (params.status) query.set("status", params.status);
  const [sports, events] = await Promise.all([
    api<Sport[]>("/sports", []),
    api<SportsEvent[]>(`/events?${query}`, [], 15),
  ]);

  return (
    <>
      <main className="wrap event-hub">
        <PageTitle
          eyebrow="EVENT HUB"
          title="Lịch & kết quả trung thực"
          description="Xem lịch theo ngày, biết rõ nguồn và độ mới của dữ liệu. BaoTheX chỉ gắn nhãn đang diễn ra khi nguồn thực sự hỗ trợ."
        />
        <form className="event-filters" action="/lich-the-thao">
          <label>
            Ngày
            <input type="date" name="date" defaultValue={date} />
          </label>
          <label>
            Môn
            <select name="sport" defaultValue={params.sport || ""}>
              <option value="">Tất cả môn</option>
              {sports.map((s) => (
                <option value={s.slug} key={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          </label>
          <label>
            Trạng thái
            <select name="status" defaultValue={params.status || ""}>
              <option value="">Tất cả</option>
              <option value="scheduled">Sắp diễn ra</option>
              <option value="live">Đang diễn ra</option>
              <option value="finished">Đã kết thúc</option>
              <option value="postponed">Tạm hoãn</option>
            </select>
          </label>
          <button className="btn ember" type="submit">
            Xem lịch
          </button>
        </form>
        <div className="date-shortcuts">
          <Link href={`/lich-the-thao?date=${shiftDate(date, -1)}&sport=${params.sport || ""}`}>
            ← Hôm trước
          </Link>
          <strong>
            {new Intl.DateTimeFormat("vi-VN", {
              weekday: "long",
              day: "2-digit",
              month: "long",
            }).format(new Date(`${date}T12:00:00`))}
          </strong>
          <Link href={`/lich-the-thao?date=${shiftDate(date, 1)}&sport=${params.sport || ""}`}>
            Hôm sau →
          </Link>
        </div>
        {events.length ? (
          <div className="event-list">
            {events.map((event) => (
              <SportsEventCard event={event} key={event.id} />
            ))}
          </div>
        ) : (
          <div className="empty-state">
            <h2>Chưa có sự kiện cho bộ lọc này</h2>
            <p>
              Website vẫn hoạt động bình thường khi chưa cấu hình nguồn miễn phí. Quản trị viên có
              thể bổ sung lịch Việt Nam thủ công.
            </p>
          </div>
        )}
        <aside className="data-promise">
          <strong>Cam kết dữ liệu</strong>
          <span>
            “Lịch dự kiến” không phải live. Dữ liệu chậm luôn có nhãn. Sự kiện nhập tay luôn ghi
            BaoTheX cập nhật.
          </span>
        </aside>
      </main>
      <Footer />
    </>
  );
}

function shiftDate(value: string, days: number) {
  const date = new Date(`${value}T12:00:00`);
  date.setDate(date.getDate() + days);
  return date.toISOString().slice(0, 10);
}
