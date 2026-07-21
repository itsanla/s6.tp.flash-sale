import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  server: {
    // Dipakai hanya saat pengembangan frontend terpisah; di produksi seluruh berkas
    // dilayani oleh binary Go yang sama.
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
});
