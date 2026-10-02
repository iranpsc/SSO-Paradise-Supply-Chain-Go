import { defineConfig } from "@playwright/test";
import path from "node:path";
const run = (process.env.TEST_RUN_ID ??= Date.now().toString());
process.env.MYSQL_DATABASE = `paradise_e2e_${run}_${process.pid}`;
process.env.MYSQL_HOST ??= "127.0.0.1";
const mailDirectory = path.resolve("test-results", `mail-${run}`);
process.env.TEST_MAIL_DIRECTORY = mailDirectory;
export default defineConfig({
  testDir: "./tests",
  globalSetup: "./tests/global-setup.ts",
  fullyParallel: false,
  workers: 1,
  timeout: 60_000,
  expect: { timeout: 15_000 },
  use: {
    baseURL: "http://localhost:3100",
    channel: process.env.PLAYWRIGHT_CHANNEL || undefined,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  webServer: [
    {
      command: "go run ../cmd/testdb && go run ../cmd/api",
      url: "http://127.0.0.1:8180/healthz",
      timeout: 180_000,
      env: {
        HTTP_ADDR: "127.0.0.1:8180",
        PUBLIC_URL: "http://localhost:3100",
        MYSQL_DATABASE: process.env.MYSQL_DATABASE,
        MYSQL_HOST: process.env.MYSQL_HOST,
        TRUSTED_PROXY_CIDRS: "127.0.0.1/32,::1/128",
        OAUTH_PRIVATE_KEY_PATH: path.resolve("test-results", `oauth-${run}.key`),
        MAIL_DIRECTORY: mailDirectory,
      },
    },
    {
      command: "npm run dev -- --port 3100",
      url: "http://localhost:3100/login",
      timeout: 180_000,
      env: {
        API_ORIGIN: "http://127.0.0.1:8180",
        NEXT_TELEMETRY_DISABLED: "1",
        NEXT_DIST_DIR: ".next/e2e",
      },
    },
  ],
});
