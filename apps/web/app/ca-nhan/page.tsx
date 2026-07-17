import Link from "next/link";
import { Footer, PageTitle } from "../ui";
import { AccountOverview } from "../account-panels";
import { FanPassportPanel } from "../fan-passport";
export default function Page() {
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Tài khoản"
          title="Không gian của bạn"
          description="Quản lý dòng tin, chủ đề theo dõi và các nội dung đã lưu."
        />
        <AccountOverview />
        <FanPassportPanel />
        <div className="topic-grid section">
          <Link href="/luu" className="topic">
            <strong>Thư viện đã lưu</strong>
            <div className="meta">Xem nội dung đã đánh dấu →</div>
          </Link>
          <Link href="/cai-dat" className="topic">
            <strong>Cài đặt thông báo</strong>
            <div className="meta">Telegram và bản tin →</div>
          </Link>
          <Link href="/chu-de" className="topic">
            <strong>Chủ đề theo dõi</strong>
            <div className="meta">Chọn lại mối quan tâm →</div>
          </Link>
        </div>
      </main>
      <Footer />
    </>
  );
}
