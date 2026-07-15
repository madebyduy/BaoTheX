import { Footer } from "../ui";
import { CheckoutButton } from "./checkout-button";

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
        </section>
        <section className="premium-grid">
          <article>
            <b>01</b>
            <h2>Nghe báo mỗi sáng</h2>
            <p>Bản audio 6–8 phút, phát trên web và gửi thẳng qua Telegram.</p>
          </article>
          <article>
            <b>02</b>
            <h2>Tin đúng lúc</h2>
            <p>Web Push và Telegram cho tin xác nhận, tỷ số, đội hình và chấn thương.</p>
          </article>
          <article>
            <b>03</b>
            <h2>Theo dõi thật sự</h2>
            <p>Cá nhân hóa theo đội bóng, giải đấu, vận động viên và nguồn bạn tin tưởng.</p>
          </article>
          <article>
            <b>04</b>
            <h2>Một sự kiện, mọi nguồn</h2>
            <p>Xem nhiều góc nhìn trong một cụm tin, có nhãn xác minh và độ uy tín.</p>
          </article>
        </section>
      </main>
      <Footer />
    </>
  );
}
