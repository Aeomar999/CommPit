import { defineConfig } from "@playwright/test";

const baseURL = process.env.MOCKSMS_URL ?? "http://127.0.0.1:4010";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  retries: process.env.CI ? 1 : 0,
  use: {
    baseURL,
  },
});
