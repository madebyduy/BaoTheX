import type { NextRequest } from "next/server";

// Same-origin API proxy.
//
// The browser talks to /api/* on the frontend origin; this handler forwards to
// the real backend server-side. That removes three deploy-only failures at once:
//
//   1. The free ngrok tunnel answers every *browser* request with an HTML
//      interstitial that carries no CORS headers, so a direct cross-origin fetch
//      dies as "Failed to fetch" and login never reaches the API. A server-side
//      hop is not a browser, and we also send ngrok-skip-browser-warning, so it
//      passes straight through.
//   2. No cross-origin request means no CORS preflight — the earlier attempt to
//      send the skip header from the browser failed because the OPTIONS preflight
//      cannot carry it and was itself interstitialed.
//   3. The session cookie is now first-party on the frontend origin, so it is
//      stored and sent without any SameSite gymnastics.
//
// Server-rendered code in lib.ts keeps calling the backend directly; only the
// browser goes through here.
const BASE = (
  process.env.API_INTERNAL_URL ||
  process.env.NEXT_PUBLIC_API_URL ||
  "http://localhost:8081"
).replace(/\/+$/, "");

// Hop-by-hop and length/encoding headers must not be copied verbatim: fetch has
// already decoded the upstream body, so a leftover content-encoding/length would
// describe bytes that no longer exist.
const STRIP_RESPONSE = new Set([
  "content-encoding",
  "content-length",
  "transfer-encoding",
  "connection",
  "keep-alive",
]);

async function proxy(req: NextRequest, path: string[]): Promise<Response> {
  const target = `${BASE}/api/${path.map(encodeURIComponent).join("/")}${req.nextUrl.search}`;

  const headers = new Headers(req.headers);
  headers.set("ngrok-skip-browser-warning", "true");
  // Let fetch set these for the upstream connection.
  headers.delete("host");
  headers.delete("connection");
  headers.delete("content-length");

  const init: RequestInit & { duplex?: "half" } = {
    method: req.method,
    headers,
    redirect: "manual",
  };
  if (req.method !== "GET" && req.method !== "HEAD") {
    // Buffer the body so the request can be replayed to the upstream. The API
    // payloads here are small JSON documents.
    init.body = await req.arrayBuffer();
  }

  let upstream: Response;
  try {
    upstream = await fetch(target, init);
  } catch {
    return Response.json({ error: "upstream_unreachable" }, { status: 502 });
  }

  const resHeaders = new Headers();
  upstream.headers.forEach((value, key) => {
    if (STRIP_RESPONSE.has(key.toLowerCase()) || key.toLowerCase() === "set-cookie") return;
    resHeaders.set(key, value);
  });
  // Set-Cookie must survive as separate headers; Headers.get() would fold them.
  const setCookies =
    (upstream.headers as Headers & { getSetCookie?: () => string[] }).getSetCookie?.() ?? [];
  for (const cookie of setCookies) resHeaders.append("set-cookie", cookie);

  const body = await upstream.arrayBuffer();
  return new Response(body, { status: upstream.status, headers: resHeaders });
}

type Ctx = { params: Promise<{ path: string[] }> };
const handler = async (req: NextRequest, ctx: Ctx) => proxy(req, (await ctx.params).path);

export const GET = handler;
export const POST = handler;
export const PUT = handler;
export const PATCH = handler;
export const DELETE = handler;
export const OPTIONS = handler;

export const dynamic = "force-dynamic";
