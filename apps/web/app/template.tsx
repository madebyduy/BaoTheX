// App Router re-mounts this template on every client navigation, so the CSS
// entrance animation on `.page-turn` fires each time the reader opens a new
// page — a subtle "lật trang báo" (newspaper page turn) that makes navigation
// feel deliberate instead of snapping. The sticky header and back button live
// in layout.tsx (outside this wrapper), so only the page body turns.
//
// Kept as a Server Component: no client JS needed, the effect is pure CSS.
export default function Template({ children }: { children: React.ReactNode }) {
  return <div className="page-turn">{children}</div>;
}
