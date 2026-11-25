import { defineConfig } from "@playwright/test";

const port = 8091;

// The smoke tests run against the real binary: frontend built, embedded, served by Go.
export default defineConfig({
  testDir: "e2e",
  use: { baseURL: `http://localhost:${port}` },
  webServer: {
    command: "pnpm build:bin && ./bin/zoop",
    url: `http://localhost:${port}/api/healthz`,
    timeout: 120_000,
    reuseExistingServer: false,
    env: {
      PORT: String(port),
      DATABASE_URL:
        process.env.ZOOP_E2E_DATABASE_URL ??
        "postgres://zoop:zoop@localhost:5433/zoop?sslmode=disable",
    },
  },
});
