import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "export",
  transpilePackages: ["@tiles-spliter/shared"],
  images: { unoptimized: true },
};

export default nextConfig;
