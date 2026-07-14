import { Footer, PageTitle } from "../ui";
import { NotificationSettings } from "../account-panels";
export default function Page() {
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Cài đặt"
          title="Cài đặt tài khoản"
          description="Điều chỉnh cách BaoTheX gửi nội dung đến bạn."
        />
        <NotificationSettings />
      </main>
      <Footer />
    </>
  );
}
