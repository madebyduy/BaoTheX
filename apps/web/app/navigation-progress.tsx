"use client";

import { usePathname, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";

type ProgressPhase = "idle" | "loading" | "completing";

// App Router intentionally has no global router-events API. This component
// starts feedback from the user's navigation intent (internal link or GET
// form), then completes it when the pathname/query actually changes.
export function NavigationProgress() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const routeKey = `${pathname}?${searchParams.toString()}`;
  const [phase, setPhase] = useState<ProgressPhase>("idle");
  const phaseRef = useRef<ProgressPhase>("idle");
  const safetyTimer = useRef<number | null>(null);

  const changePhase = useCallback((next: ProgressPhase) => {
    phaseRef.current = next;
    setPhase(next);
  }, []);

  const finish = useCallback(() => {
    if (phaseRef.current !== "loading") return;
    if (safetyTimer.current !== null) window.clearTimeout(safetyTimer.current);
    changePhase("completing");
    window.setTimeout(() => changePhase("idle"), 220);
  }, [changePhase]);

  const start = useCallback(() => {
    if (phaseRef.current === "loading") return;
    if (safetyTimer.current !== null) window.clearTimeout(safetyTimer.current);
    changePhase("loading");
    // Never leave stale progress on screen when navigation is cancelled or a
    // route errors before its URL changes.
    safetyTimer.current = window.setTimeout(finish, 10000);
  }, [changePhase, finish]);

  useEffect(() => {
    finish();
  }, [routeKey, finish]);

  useEffect(() => {
    function onClick(event: MouseEvent) {
      if (
        event.defaultPrevented ||
        event.button !== 0 ||
        event.metaKey ||
        event.ctrlKey ||
        event.shiftKey ||
        event.altKey
      ) {
        return;
      }
      const element = event.target instanceof Element ? event.target : null;
      const anchor = element?.closest("a[href]");
      if (!(anchor instanceof HTMLAnchorElement)) return;
      if (anchor.target && anchor.target !== "_self") return;
      if (anchor.hasAttribute("download")) return;

      const destination = new URL(anchor.href, window.location.href);
      if (destination.origin !== window.location.origin) return;
      const current = new URL(window.location.href);
      if (destination.pathname === current.pathname && destination.search === current.search) {
        return;
      }
      start();
    }

    function onSubmit(event: SubmitEvent) {
      const form = event.target;
      if (!(form instanceof HTMLFormElement)) return;
      const explicitAction = form.getAttribute("action");
      if (!explicitAction || (form.method && form.method.toLowerCase() !== "get")) return;
      const destination = new URL(explicitAction, window.location.href);
      if (destination.origin === window.location.origin) start();
    }

    document.addEventListener("click", onClick, true);
    document.addEventListener("submit", onSubmit, true);
    window.addEventListener("popstate", start);
    return () => {
      document.removeEventListener("click", onClick, true);
      document.removeEventListener("submit", onSubmit, true);
      window.removeEventListener("popstate", start);
      if (safetyTimer.current !== null) window.clearTimeout(safetyTimer.current);
    };
  }, [start]);

  return (
    <div
      className={`route-progress route-progress-${phase}`}
      role="progressbar"
      aria-label="Đang chuyển trang"
      aria-hidden={phase === "idle"}
    >
      <span />
    </div>
  );
}
