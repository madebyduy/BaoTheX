import type { NextConfig } from "next";
import { initOpenNextCloudflareForDev } from "@opennextjs/cloudflare";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: "standalone",
  // Keep standalone tracing inside this app when unrelated lockfiles exist in
  // a developer's home directory.
  outputFileTracingRoot: process.cwd(),
  // Keep lint as an explicit CI step. OpenNext invokes Next through Bun on
  // Windows, where CRLF files are incorrectly reported by eslint-plugin-prettier.
  eslint: { ignoreDuringBuilds: true },
};

export default nextConfig;

initOpenNextCloudflareForDev();
