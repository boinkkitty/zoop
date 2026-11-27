import { sentryVitePlugin } from "@sentry/vite-plugin";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

// Source maps go to Sentry only when CI hands the build an auth token (pushes
// to main). They're "hidden" (no sourceMappingURL comment) and deleted after
// upload; the Dockerfile also deletes any left, so none are ever served.
const sentryAuthToken = process.env.SENTRY_AUTH_TOKEN;

export default defineConfig({
  root: "web",
  plugins: [
    react(),
    sentryVitePlugin({
      disable: !sentryAuthToken,
      authToken: sentryAuthToken,
      org: process.env.SENTRY_ORG,
      project: process.env.SENTRY_PROJECT,
      release: { name: process.env.VITE_SENTRY_RELEASE },
      sourcemaps: { filesToDeleteAfterUpload: ["web/dist/**/*.map"] },
      telemetry: false,
    }),
  ],
  build: {
    outDir: "dist",
    emptyOutDir: true,
    sourcemap: "hidden",
  },
  server: {
    proxy: { "/api": "http://localhost:8080" },
  },
  test: {
    root: ".",
    include: ["web/src/**/*.test.ts", "web/src/**/*.test.tsx"],
    passWithNoTests: true,
  },
});
