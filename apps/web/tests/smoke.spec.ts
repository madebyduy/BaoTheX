import { test, expect } from "@playwright/test";

// These pages must render without a backend (demo-data fallbacks or fully
// static). They catch the most common production breakages: a build/runtime
// crash, a broken layout import, or a missing route.

test("homepage renders masthead and header", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("link", { name: /BaoTheX|BaoTheX/i }).first()).toBeVisible();
  await expect(page.locator(".header")).toBeVisible();
  await expect(page.locator(".footer")).toBeVisible();
});

test("latest news list page renders", async ({ page }) => {
  await page.goto("/danh-muc");
  await expect(page.locator("main")).toBeVisible();
  await expect(page.locator(".footer")).toBeVisible();
});

test("video page renders", async ({ page }) => {
  await page.goto("/video");
  await expect(page.locator("main")).toBeVisible();
});

test("search page renders", async ({ page }) => {
  await page.goto("/tim-kiem?q=bong-da");
  await expect(page.locator("main")).toBeVisible();
});

test("about page renders trust content", async ({ page }) => {
  await page.goto("/gioi-thieu");
  await expect(page.getByRole("heading", { name: /Giới thiệu/i })).toBeVisible();
});

test("login page renders", async ({ page }) => {
  await page.goto("/dang-nhap");
  await expect(page.locator("main")).toBeVisible();
});

test("unknown route shows branded 404", async ({ page }) => {
  const res = await page.goto("/khong-ton-tai-abc-123");
  expect(res?.status()).toBe(404);
  await expect(page.getByText("404")).toBeVisible();
});

test("RSS feed is served as XML", async ({ request }) => {
  const res = await request.get("/feed.xml");
  expect(res.ok()).toBeTruthy();
  expect(res.headers()["content-type"]).toContain("xml");
});

test("homepage has usable landmarks and no unnamed controls", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator("html")).toHaveAttribute("lang", "vi");
  await expect(page.locator("main")).toHaveCount(1);

  const unnamed = await page.locator("button, a[href], input, select, textarea").evaluateAll(
    (nodes) =>
      nodes.filter((node) => {
        const element = node as HTMLElement;
        const label =
          element.getAttribute("aria-label") ||
          element.getAttribute("title") ||
          element.textContent?.trim() ||
          (element instanceof HTMLInputElement ? element.placeholder : "");
        return !label;
      }).length,
  );
  expect(unnamed).toBe(0);
});

test("mobile homepage does not overflow horizontally", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
  );
  expect(overflow).toBeLessThanOrEqual(1);
});
