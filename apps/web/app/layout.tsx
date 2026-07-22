import type { Metadata } from "next";
import { Archivo, Be_Vietnam_Pro, JetBrains_Mono } from "next/font/google";
import { Suspense } from "react";
import "./globals.css";
import { SiteHeader } from "./ui";
import { NavigationProgress } from "./navigation-progress";

// Fonts are self-hosted and preloaded by next/font — no render-blocking round
// trip to fonts.googleapis.com, and zero layout shift because the fallback
// metrics are matched. The CSS variables are consumed in globals.css.
//
// Three faces, three jobs. Be Vietnam Pro stays on body text because it was
// drawn for Vietnamese: its diacritics sit correctly at reading sizes, which
// most Latin-first faces only approximate. Archivo takes the display role — a
// wide, flat-sided grotesque that gives headlines the weight of a sports front
// page, where the previous rounded geometric read friendly and generic. And the
// mono is promoted from an afterthought to the paper's utility voice: source
// names, corroboration counts, kick-off times and scores all set in it, so the
// evidence around a story reads like wire copy rather than like more prose.
const bodyFont = Be_Vietnam_Pro({
  subsets: ["latin", "vietnamese"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-body",
  display: "swap",
});
const displayFont = Archivo({
  subsets: ["latin", "vietnamese"],
  weight: ["600", "700", "800"],
  variable: "--font-display",
  display: "swap",
});
const monoFont = JetBrains_Mono({
  subsets: ["latin", "vietnamese"],
  weight: ["400", "500", "700"],
  variable: "--font-mono",
  display: "swap",
});
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
  alternates: {
    canonical: "/",
    types: {
      "application/rss+xml": [{ url: "/feed.xml", title: "BaoTheX — Tin thể thao mới nhất" }],
    },
  },
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
    <html
      lang="vi"
      suppressHydrationWarning
      className={`${bodyFont.variable} ${displayFont.variable} ${monoFont.variable}`}
    >
      <body>
        <PersistentAudioProvider>
          <Suspense fallback={null}>
            <NavigationProgress />
          </Suspense>
          <ProductAnalytics />
          <SiteHeader />
          <SiteBackButton />
          {children}
        </PersistentAudioProvider>
      </body>
    </html>
  );
}
