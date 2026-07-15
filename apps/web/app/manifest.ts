import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "BaoTheX — Bản tin thể thao của bạn",
    short_name: "BaoTheX",
    description: "Đọc, nghe và nhận tin thể thao từ những đội bạn theo dõi.",
    start_url: "/",
    display: "standalone",
    background_color: "#080a0e",
    theme_color: "#ff6b4a",
    lang: "vi",
    icons: [
      { src: "/icon-192.png", sizes: "192x192", type: "image/png" },
      { src: "/icon-512.png", sizes: "512x512", type: "image/png" },
    ],
  };
}
