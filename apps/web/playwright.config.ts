import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  reporter: "list",
  use: {
    baseURL: "http://localhost:4123",
  },
  webServer: {
    command: "bun run build && bunx serve out -l 4123",
    url: "http://localhost:4123",
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
});
