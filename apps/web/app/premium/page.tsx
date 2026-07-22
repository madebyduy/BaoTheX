import { Footer } from "../ui";
import { pageMetadata } from "../lib";
import { CheckoutButton } from "./checkout-button";

export const metadata = pageMetadata({
  title: "BaoTheX Premium",
  description: "BaoTheX Premium 10.000đ/tháng: dòng tin theo dõi tuyệt đối và Web Push tức thời.",
  path: "/premium",
});
import { PremiumStatus } from "./premium-status";
import { api } from "../lib";

export default async function PremiumPage() {
  const capabilities = await api<{ sepay_enabled?: boolean; premium_monthly_price?: number }>(
    "/capabilities",
    {},
    // Price and checkout availability must never be served from an older ISR
    // snapshot: the number on this button has to match the SePay order exactly.
    0,
  );
  const price = capabilities.premium_monthly_price || 10000;
  return (
    <>
      <main className="wrap premium-page">
        <section className="premium-hero">
          <span className="tag">BAOTHEX PREMIUM</span>
          <h1>
            Ít nhiễu hơn.
            <br />
            Đúng đội bạn hơn.
          </h1>
          <p>
            Giữ những gì cần thiết trong tầm mắt: dòng tin không nhiễu và thông báo tức thời từ đội,
            giải đấu, vận động viên bạn thật sự quan tâm.
          </p>
          {capabilities.sepay_enabled ? (
            <CheckoutButton price={price} />
          ) : (
            <p className="settings-message">
              Premium đang ở chế độ giới thiệu; mọi tính năng cốt lõi vẫn miễn phí.
            </p>
          )}
          <PremiumStatus />
        </section>
        <section className="premium-grid">
          <article>
            <b>01</b>
            <h2>Dòng tin chỉ dành cho bạn</h2>
            <p>
              Bật chế độ “Chỉ chủ đề theo dõi” để loại khỏi trang chủ những môn, đội và câu chuyện
              bạn không quan tâm.
            </p>
          </article>
          <article>
            <b>02</b>
            <h2>Web Push tức thời</h2>
            <p>
              Nhận thông báo trực tiếp trên máy tính về tin quan trọng, không cần mở Telegram hay
              giữ trang báo luôn bật.
            </p>
          </article>
          <article>
            <b>03</b>
            <h2>Vẫn miễn phí mỗi ngày</h2>
            <p>
              Theo dõi chủ đề cơ bản, đọc toàn bộ bài báo và nhận bản tin Telegram 6h/20h vẫn miễn
              phí cho mọi tài khoản.
            </p>
          </article>
          <article>
            <b>04</b>
            <h2>10.000đ cho 30 ngày</h2>
            <p>
              Một mức giá nhỏ, không quảng cáo quyền lợi chưa tồn tại và có thể gia hạn thêm 30 ngày
              bất cứ lúc nào.
            </p>
          </article>
        </section>
      </main>
      <Footer />
    </>
  );
}
