import type { Metadata } from "next";
import "./globals.css";
import { SiteHeader } from "./ui";

export const metadata: Metadata = {
  title: "BaoTheX — Trung tâm thông tin fitness",
  description: "Tin tức, nghiên cứu và kiến thức tập luyện có dẫn nguồn cho người Việt.",
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
