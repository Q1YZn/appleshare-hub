import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vite";

export default defineConfig({
  root: "frontend",
  plugins: [vue()],
  build: {
    outDir: "../web",
    emptyOutDir: true
  }
});
