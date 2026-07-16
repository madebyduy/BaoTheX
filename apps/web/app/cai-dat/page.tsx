import { Footer, PageTitle } from "../ui";
import { NotificationSettings, TelegramSettings } from "../account-panels";
import { PWAControls } from "../pwa-controls";
import { FeedCustomizationSettings } from "../feed-customization";
export default function Page() {
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Cài đặt"
          title="Cài đặt tài khoản"
          description="Điều chỉnh cách BaoTheX gửi nội dung đến bạn."
        />
        <FeedCustomizationSettings />
        <TelegramSettings />
        <PWAControls />
        <NotificationSettings />
      </main>
      <Footer />
    </>
  );
}
