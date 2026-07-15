import type { Metadata } from "next";
import "./globals.css";
import { SiteHeader } from "./ui";

export const metadata: Metadata = {
  title: "BaoTheX — Báo thể thao chọn lọc",
  description:
    "Tin thể thao nổi bật trong ngày, được tổng hợp, kiểm chứng nguồn và biên tập bằng tiếng Việt.",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="vi">
      <body>
        <SiteHeader />
        {children}
      </body>
    </html>
  );
}
