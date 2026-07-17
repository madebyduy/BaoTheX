import type { Metadata } from "next";
import "./globals.css";
import { SiteHeader } from "./ui";
import { SiteBackButton } from "./action-buttons";
import { PersistentAudioProvider } from "./persistent-audio-player";
import { ProductAnalytics } from "./product-analytics";

const SITE = process.env.NEXT_PUBLIC_SITE_URL || "https://baothex.vn";
const DESCRIPTION =
  "Tin thể thao nổi bật trong ngày, được tổng hợp, kiểm chứng nguồn và biên tập bằng tiếng Việt.";

export const metadata: Metadata = {
  metadataBase: new URL(SITE),
  title: {
    default: "BaoTheX — Báo thể thao chọn lọc",
    template: "%s | BaoTheX",
  },
  description: DESCRIPTION,
  applicationName: "BaoTheX",
  manifest: "/manifest.webmanifest",
  appleWebApp: { capable: true, title: "BaoTheX", statusBarStyle: "black-translucent" },
  alternates: { canonical: "/" },
  robots: { index: true, follow: true },
  openGraph: {
    type: "website",
    siteName: "BaoTheX",
    locale: "vi_VN",
    url: SITE,
    title: "BaoTheX — Báo thể thao chọn lọc",
    description: DESCRIPTION,
  },
  twitter: {
    card: "summary_large_image",
    title: "BaoTheX — Báo thể thao chọn lọc",
    description: DESCRIPTION,
  },
};

export const viewport = { themeColor: "#ff6b4a", colorScheme: "dark" };

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="vi">
      <body>
        <PersistentAudioProvider>
          <ProductAnalytics />
          <SiteHeader />
          <SiteBackButton />
          {children}
        </PersistentAudioProvider>
      </body>
    </html>
  );
}
