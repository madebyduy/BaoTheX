import { Footer, PageTitle } from "../ui";
import { SavedContent } from "../saved-content";
export default function Page() {
  return (
    <>
      <main className="wrap">
        <PageTitle
          eyebrow="Thư viện cá nhân"
          title="Nội dung đã lưu"
          description="Đăng nhập để xem và sắp xếp các bài viết bạn muốn đọc lại."
        />
        <section className="section">
          <SavedContent />
        </section>
      </main>
      <Footer />
    </>
  );
}
