import { defineConfig, devices } from "@playwright/test";

// Smoke tests run against a production build with no API backend. Public pages
// must render honest empty states rather than demo/editorial content. Point the
// API URL at a closed port so upstream failures are deterministic and fast.
const PORT = 3000;
const BASE_URL = `http://127.0.0.1:${PORT}`;

export default defineConfig({
  testDir: "./tests",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["html", { open: "never" }], ["list"]] : "list",
  use: {
    baseURL: BASE_URL,
    trace: "on-first-retry",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: {
    command: "npm run start",
    url: BASE_URL,
    timeout: 120_000,
    reuseExistingServer: !process.env.CI,
    env: {
      NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || "http://127.0.0.1:9",
      NEXT_PUBLIC_SITE_URL: BASE_URL,
    },
  },
});
