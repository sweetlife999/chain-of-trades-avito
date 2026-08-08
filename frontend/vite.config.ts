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
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: "react-vendor",
              test: /[\\/]node_modules[\\/](?:react|react-dom|scheduler)[\\/]/,
              priority: 50,
            },
            {
              name: "redux-vendor",
              test: /[\\/]node_modules[\\/](?:@reduxjs[\\/]toolkit|react-redux|redux|redux-thunk|immer|reselect|use-sync-external-store)[\\/]/,
              priority: 40,
            },
            {
              name: "router-vendor",
              test: /[\\/]node_modules[\\/](?:react-router|react-router-dom|@remix-run[\\/]router)[\\/]/,
              priority: 40,
            },
            {
              name: "form-vendor",
              test: /[\\/]node_modules[\\/](?:@hookform[\\/]resolvers|react-hook-form)[\\/]/,
              priority: 40,
            },
            {
              name: "validation-vendor",
              test: /[\\/]node_modules[\\/]zod[\\/]/,
              priority: 40,
            },
            {
              name: "query-vendor",
              test: /[\\/]node_modules[\\/]@tanstack[\\/](?:react-query|query-core)[\\/]/,
              priority: 40,
            },
            {
              name: "vendor",
              test: /[\\/]node_modules[\\/]/,
              priority: 10,
            },
          ],
        },
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
    },
  },
});