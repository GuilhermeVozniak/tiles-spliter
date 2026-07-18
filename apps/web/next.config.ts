import type { NextConfig } from "next";

// Static export for GitHub Pages. The project site is served under
// /tiles-spliter, so the deploy workflow sets NEXT_PUBLIC_BASE_PATH=/tiles-spliter;
// a custom domain would leave it unset (empty basePath).
const nextConfig: NextConfig = {
  output: "export",
  transpilePackages: ["@tiles-spliter/shared"],
  images: { unoptimized: true },
  basePath: process.env.NEXT_PUBLIC_BASE_PATH ?? "",
};

export default nextConfig;
