import { Footer, PageTitle } from "../../ui";
import { AdminConsole } from "../admin-console";

export default function Page() {
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Vận hành"
          title="Tác vụ nền"
          description="Theo dõi hàng đợi, lỗi và chạy lại tác vụ cần thiết."
        />
        <AdminConsole initialView="jobs" />
      </main>
      <Footer />
    </>
  );
}
