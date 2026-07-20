import { ImageResponse } from "next/og";

export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
export const alt = "BaoTheX — Báo thể thao chọn lọc";

async function loadFont(subset: string): Promise<ArrayBuffer | null> {
  try {
    const res = await fetch(
      `https://cdn.jsdelivr.net/fontsource/fonts/be-vietnam-pro@latest/${subset}-700-normal.ttf`,
      { next: { revalidate: 86400 } },
    );
    if (!res.ok) return null;
    return await res.arrayBuffer();
  } catch {
    return null;
  }
}

export default async function Image() {
  const fonts = [];
  for (const subset of ["latin", "vietnamese"]) {
    const data = await loadFont(subset);
    if (data)
      fonts.push({ name: "BeVietnam", data, weight: 700 as const, style: "normal" as const });
  }
  return new ImageResponse(
    <div
      style={{
        width: "100%",
        height: "100%",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        gap: 24,
        background: "linear-gradient(135deg, #0b1018 0%, #12161d 100%)",
        fontFamily: fonts.length ? "BeVietnam" : "sans-serif",
      }}
    >
      <div style={{ display: "flex", fontSize: 96, fontWeight: 800, color: "#fbfcff" }}>
        <span>Bao</span>
        <span style={{ color: "#ff6b4a" }}>TheX</span>
      </div>
      <div style={{ display: "flex", fontSize: 34, color: "#aeb5c2" }}>
        Báo thể thao chọn lọc cho người Việt
      </div>
    </div>,
    { ...size, fonts: fonts.length ? fonts : undefined },
  );
}
