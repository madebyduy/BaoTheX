"use client";

import { useEffect } from "react";
import { usePathname } from "next/navigation";

const API =
  typeof window !== "undefined" ? "" : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

export function ProductAnalytics() {
  const pathname = usePathname();
  useEffect(() => {
    const day = new Date().toISOString().slice(0, 10);
    if (localStorage.getItem("btx-last-visit") === day) return;
    localStorage.setItem("btx-last-visit", day);
    sendProductEvent("visit", { path: pathname });
  }, [pathname]);
  return null;
}

export function ReadingTracker({ contentId }: { contentId: number }) {
  useEffect(() => {
    const timer = window.setTimeout(() => {
      fetch(`${API}/api/v1/history/${contentId}`, { method: "POST", credentials: "include" }).catch(
        () => null,
      );
      sendProductEvent("content_read", { content_id: contentId });
    }, 10000);
    return () => window.clearTimeout(timer);
  }, [contentId]);
  return null;
}

function sendProductEvent(event_name: string, properties: Record<string, unknown>) {
  let clientId = localStorage.getItem("btx-cid");
  if (!clientId) {
    clientId = crypto.randomUUID?.() || `${Date.now()}`;
    localStorage.setItem("btx-cid", clientId);
  }
  fetch(`${API}/api/v1/product-events`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ client_id: clientId, event_name, properties }),
  }).catch(() => null);
}
