/* eslint-disable react-refresh/only-export-components */
import type { Metadata } from "next";
import "./globals.css";
import { SiteHeader } from "./ui";
import { SiteBackButton } from "./action-buttons";

export const metadata: Metadata = {
  title: "BaoTheX — Báo thể thao chọn lọc",
  description:
    "Tin thể thao nổi bật trong ngày, được tổng hợp, kiểm chứng nguồn và biên tập bằng tiếng Việt.",
  manifest: "/manifest.webmanifest",
  applicationName: "BaoTheX",
  appleWebApp: { capable: true, title: "BaoTheX", statusBarStyle: "black-translucent" },
};

export const viewport = { themeColor: "#ff6b4a", colorScheme: "dark" };

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="vi">
      <body>
        <SiteHeader />
        <SiteBackButton />
        {children}
      </body>
    </html>
  );
}
