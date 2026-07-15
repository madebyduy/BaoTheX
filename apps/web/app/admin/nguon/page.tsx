import { Footer, PageTitle } from "../../ui";
import { AdminConsole } from "../admin-console";

export default function Page() {
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Quản trị nguồn"
          title="Nguồn dữ liệu"
          description="Bật, tắt và chủ động quét từng nguồn tin."
        />
        <AdminConsole initialView="sources" />
      </main>
      <Footer />
    </>
  );
}
