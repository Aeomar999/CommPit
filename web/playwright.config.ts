import { defineConfig } from "@playwright/test";

const baseURL = process.env.MOCKSMS_URL ?? "http://127.0.0.1:4010";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  retries: process.env.CI ? 1 : 0,
  // Serial: every spec shares one server and its default project, so
  // parallel workers trade speed for cross-test interference.
  workers: 1,
  use: {
    baseURL,
  },
});
