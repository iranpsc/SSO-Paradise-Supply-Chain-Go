import type { NextConfig } from "next";
const config: NextConfig = {
  distDir: process.env.NEXT_DIST_DIR ?? ".next",
  output: "standalone",
  async rewrites() {
    return [
      {source:"/email/verify/:id/:hash",destination:`${process.env.API_ORIGIN ?? "http://127.0.0.1:8080"}/email/verify/:id/:hash`},
      {source:"/password/reset/:token",destination:"/password/reset?token=:token"},
      {source: "/oauth/:path*",destination: `${process.env.API_ORIGIN ?? "http://127.0.0.1:8080"}/oauth/:path*`},
      {
        source: "/api/:path*",
        destination: `${process.env.API_ORIGIN ?? "http://127.0.0.1:8080"}/api/:path*`,
      },
    ];
  },
  async redirects() { return [{source:"/account/edit",destination:"/account",permanent:false},{source:"/personal-info/edit",destination:"/personal-info",permanent:false}]; },
  async headers() {
    return [
      {
        source: "/:path*",
        headers: [
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "X-Frame-Options", value: "DENY" },
          { key: "Referrer-Policy", value: "no-referrer" },
        ],
      },
    ];
  },
};
export default config;
