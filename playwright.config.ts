import { defineConfig } from "@playwright/test";

const port = 8091;

// The smoke tests run against the real binary: frontend built, embedded, served by Go.
// Set ZOOP_E2E_BASE_URL to run them against an app that is already up instead
// (CI points it at the Docker image it is about to deploy).
const external = process.env.ZOOP_E2E_BASE_URL;

export default defineConfig({
  testDir: "e2e",
  use: { baseURL: external ?? `http://localhost:${port}` },
  webServer: external
    ? undefined
    : {
        command: "pnpm build:bin && go run ./cmd/migrate && ./bin/zoop",
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
