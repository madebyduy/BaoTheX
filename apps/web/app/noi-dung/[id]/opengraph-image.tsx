import { ImageResponse } from "next/og";
import { api, idFromSlug, type Item } from "../../lib";

export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
export const alt = "BaoTheX — Báo thể thao chọn lọc";

type Detail = { item?: Item };

// Be Vietnam Pro TTFs (Latin + Vietnamese subsets) so the card renders diacritics
// correctly. Fetched from the Fontsource CDN and cached; if the fetch fails the
// card still renders with the default font rather than erroring.
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

function scorelineFrom(text: string) {
  const match = text.match(/\b(\d{1,2})\s*[-–:]\s*(\d{1,2})\b/);
  return match ? `${match[1]}-${match[2]}` : "";
}

export default async function Image({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const data = await api<Detail>(`/content/${idFromSlug(id)}`, {}, 300);
  const item = data.item;
  const title = (item?.title || "Tin thể thao nổi bật").slice(0, 140);
  const source = item?.source_name || "BaoTheX";
  const score = item ? scorelineFrom(item.title) : "";

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
        justifyContent: "space-between",
        background: "linear-gradient(135deg, #0b1018 0%, #12161d 100%)",
        padding: "64px 72px",
        fontFamily: fonts.length ? "BeVietnam" : "sans-serif",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
        <div style={{ display: "flex", alignItems: "center", fontSize: 40, color: "#fbfcff" }}>
          <span style={{ fontWeight: 800 }}>Bao</span>
          <span style={{ fontWeight: 800, color: "#ff6b4a" }}>TheX</span>
        </div>
        {score ? (
          <div
            style={{
              display: "flex",
              background: "#ff6b4a",
              color: "#0b1018",
              fontSize: 34,
              fontWeight: 800,
              padding: "8px 22px",
              borderRadius: 12,
            }}
          >
            TỶ SỐ {score}
          </div>
        ) : (
          <div style={{ display: "flex", color: "#69a7ff", fontSize: 26, fontWeight: 700 }}>
            Báo thể thao chọn lọc
          </div>
        )}
      </div>
      <div
        style={{
          display: "flex",
          fontSize: title.length > 90 ? 52 : 64,
          fontWeight: 800,
          color: "#fbfcff",
          lineHeight: 1.15,
          letterSpacing: "-0.02em",
        }}
      >
        {title}
      </div>
      <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
        <div
          style={{ display: "flex", width: 44, height: 6, background: "#ff6b4a", borderRadius: 4 }}
        />
        <div style={{ display: "flex", fontSize: 28, color: "#aeb5c2" }}>{source}</div>
      </div>
    </div>,
    { ...size, fonts: fonts.length ? fonts : undefined },
  );
}
