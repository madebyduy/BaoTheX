import type { NextRequest } from "next/server";

// Proxy media through the frontend origin so ngrok's browser-warning HTML is
// never handed to the browser's audio decoder.
const BASE = (
  process.env.API_INTERNAL_URL ||
  process.env.NEXT_PUBLIC_API_URL ||
  "http://localhost:8081"
).replace(/\/+$/, "");

const RESPONSE_HEADERS = [
  "accept-ranges",
  "cache-control",
  "content-length",
  "content-range",
  "content-type",
  "etag",
  "last-modified",
];

type Ctx = { params: Promise<{ path: string[] }> };

async function handler(req: NextRequest, ctx: Ctx): Promise<Response> {
  const { path } = await ctx.params;
  const target = `${BASE}/media/${path.map(encodeURIComponent).join("/")}${req.nextUrl.search}`;
  const headers = new Headers();
  for (const name of ["accept", "if-none-match", "if-modified-since", "range"]) {
    const value = req.headers.get(name);
    if (value) headers.set(name, value);
  }
  headers.set("ngrok-skip-browser-warning", "true");

  let upstream: Response;
  try {
    upstream = await fetch(target, { method: req.method, headers, redirect: "manual" });
  } catch {
    return Response.json({ error: "media_upstream_unreachable" }, { status: 502 });
  }

  const responseHeaders = new Headers();
  for (const name of RESPONSE_HEADERS) {
    const value = upstream.headers.get(name);
    if (value) responseHeaders.set(name, value);
  }
  return new Response(req.method === "HEAD" ? null : await upstream.arrayBuffer(), {
    status: upstream.status,
    headers: responseHeaders,
  });
}

export const GET = handler;
export const HEAD = handler;
export const dynamic = "force-dynamic";
