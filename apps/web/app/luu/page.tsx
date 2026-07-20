import { Footer, PageTitle } from "../ui";
import { pageMetadata } from "../lib";
import { SavedContent } from "../saved-content";

export const metadata = pageMetadata({
  title: "Nội dung đã lưu",
  description: "Danh sách bài viết và video bạn đã lưu trên BaoTheX.",
  path: "/luu",
  index: false,
});
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
