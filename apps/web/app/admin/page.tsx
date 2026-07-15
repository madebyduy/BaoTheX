import { Footer, PageTitle } from "../ui";
import { AdminConsole } from "./admin-console";
export default function Page() {
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Quản trị"
          title="Bảng điều khiển"
          description="Kiểm duyệt tin, quản lý nguồn và theo dõi toàn bộ dây chuyền xuất bản."
        />
        <AdminConsole />
      </main>
      <Footer />
    </>
  );
}
