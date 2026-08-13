import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [svelte()],
  resolve: {
    conditions: ["browser"],
  },
  test: {
    exclude: ["tests/e2e/**", "node_modules/**"],
    environment: "jsdom",
    environmentOptions: {
      jsdom: { url: "http://127.0.0.1/" },
    },
    setupFiles: ["./src/test/setup.ts"],
  },
});
