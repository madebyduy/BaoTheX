import { Footer, PageTitle } from "../ui";
import { AdminConsole } from "./admin-console";
import Link from "next/link";
export default function Page() {
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Tòa soạn"
          title="Hàng duyệt Góc nhìn"
          description="Ưu tiên đọc, sửa và ký duyệt các bản phân tích trước khi xuất bản. Công cụ vận hành vẫn nằm trong các tab bên dưới."
        />
        <div className="admin-quick-links">
          <Link className="btn ember" href="/admin/su-kien">
            Event & Prediction Studio →
          </Link>
        </div>
        <AdminConsole />
      </main>
      <Footer />
    </>
  );
}
