import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
  css: {
    preprocessorOptions: {
      scss: {
        additionalData: `@use "@/Styles/variables" as *; @use "@/Styles/mixins" as *;`,
      },
    },
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
        secure: false,

        rewrite: (path) => path.replace(/^\/api/, ""),
      },

      // Загруженные фотографии. Без rewrite: в базе лежит путь /uploads/<файл>, и
      // ровно его же ждёт Go — в проде этот путь проксирует Caddy тем же образом.
      "/uploads": {
        target: "http://localhost:8080",
        changeOrigin: true,
        secure: false,
      },
    },
  },
});