import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "export", // static SPA, no Node server
  images: { unoptimized: true },
  turbopack: { root: import.meta.dirname }, // pin workspace root (a lockfile exists in the home dir)
};

export default nextConfig;
