import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: "standalone",
  // Keep standalone tracing inside this app when unrelated lockfiles exist in
  // a developer's home directory.
  outputFileTracingRoot: process.cwd(),
};

export default nextConfig;
