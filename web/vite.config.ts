import { mkdirSync, writeFileSync } from "node:fs";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    {
      // emptyOutDir wipes dist/; keep .gitkeep so //go:embed all:dist still compiles.
      name: "gitkeep-dist",
      closeBundle() {
        mkdirSync("dist", { recursive: true });
        writeFileSync("dist/.gitkeep", "");
      },
    },
  ],
  server: {
    port: 5173,
    proxy: {
      // Backend GraphQL (and WS subscriptions later)
      "/query": {
        target: "http://localhost:8080",
        changeOrigin: true,
        ws: true,
      },
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
