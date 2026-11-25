import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig({
  root: "web",
  plugins: [react()],
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  server: {
    proxy: { "/api": "http://localhost:8080" },
  },
  test: {
    root: ".",
    include: ["web/src/**/*.test.ts", "web/src/**/*.test.tsx"],
    // The first Vitest suites (autosave queue, Notice rules) arrive with #21 and #24.
    passWithNoTests: true,
  },
});
