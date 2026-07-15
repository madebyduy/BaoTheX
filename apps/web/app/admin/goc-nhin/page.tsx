import { Footer, PageTitle } from "../../ui";
import { AnalysisDesk } from "./analysis-desk";

export default function Page() {
  return (
    <>
      <main className="wrap analysis-admin-page">
        <PageTitle
          eyebrow="Tòa soạn"
          title="Bàn phân tích"
          description="Hệ thống đề cử sự kiện đã xác nhận. Bài AI chỉ là bản nháp và không thể xuất bản nếu chưa có biên tập viên duyệt."
        />
        <AnalysisDesk />
      </main>
      <Footer />
    </>
  );
}
