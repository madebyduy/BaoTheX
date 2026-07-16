import { Footer } from "../ui";
import { CheckoutButton } from "./checkout-button";
import { PremiumStatus } from "./premium-status";

export default function PremiumPage() {
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
            Nhận tin đã xác minh, bản audio 6–8 phút và cảnh báo từ những đội, giải đấu, cầu thủ bạn
            thật sự quan tâm.
          </p>
          <CheckoutButton />
          <PremiumStatus />
        </section>
        <section className="premium-grid">
          <article>
            <b>01</b>
            <h2>Hai cuộc hẹn mỗi ngày</h2>
            <p>
              Thể thao 6h và 20h chỉ dùng tin đã biên tập tiếng Việt, phát trên web và gửi thẳng qua
              Telegram.
            </p>
          </article>
          <article>
            <b>02</b>
            <h2>Tin đúng đội bạn</h2>
            <p>
              Telegram và Web Push cho tin xác nhận, tỷ số, đội hình, chấn thương và chuyển nhượng
              của đội bạn theo dõi.
            </p>
          </article>
          <article>
            <b>03</b>
            <h2>Theo dõi thật sự</h2>
            <p>
              Cá nhân hóa theo đội bóng, giải đấu, vận động viên và nguồn bạn tin tưởng — không cần
              doomscroll.
            </p>
          </article>
          <article>
            <b>04</b>
            <h2>Góc Nhìn BaoTheX</h2>
            <p>
              Bài phân tích đa nguồn: điều đã xác nhận, điểm các nguồn còn vênh nhau và điều cần
              theo dõi tiếp.
            </p>
          </article>
        </section>
      </main>
      <Footer />
    </>
  );
}
