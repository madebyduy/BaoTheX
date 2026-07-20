import { Footer, PageTitle } from "../ui";
import { pageMetadata } from "../lib";
import { NotificationSettings, TelegramSettings } from "../account-panels";

export const metadata = pageMetadata({
  title: "Cài đặt",
  description: "Tùy chỉnh dòng tin, thông báo và tài khoản BaoTheX của bạn.",
  path: "/cai-dat",
  index: false,
});
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
