import { defineConfig } from "vite";

// The app is built into dist/, which embed.go embeds into the plugin's
// binary. The kit serves it at the app's base: the root of the plugin's own
// origin, or /plugins/<name>/ on a path-based installation. base "./" writes
// every URL in the build relative, so one build works under either. A file
// whose name carries Vite's content hash is cached immutable.
export default defineConfig({
  base: "./",
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
