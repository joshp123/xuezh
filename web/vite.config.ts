import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    emptyOutDir: false
  },
  server: {
    proxy: {
      "/api": "http://127.0.0.1:8765",
      "/artifacts": "http://127.0.0.1:8765"
    }
  }
});
