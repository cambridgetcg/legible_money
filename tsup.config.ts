/**
 * tsup Configuration - Optimized for Apple M4 Max
 *
 * Uses esbuild under the hood for native-speed bundling.
 * Leverages all CPU cores for parallel file processing.
 *
 * Hardware: 14 CPU cores (10P + 4E), 32 GPU cores, 36GB RAM
 */

import { defineConfig } from "tsup";

export default defineConfig({
  // Entry points - process in parallel
  entry: [
    "core/index.ts",
    "protocol/index.ts",
    "agents/index.ts",
    "consensus/index.ts",
    "state/index.ts",
    "network/index.ts",
    "messaging/index.ts",
    "endpoints/index.ts",
    "schedule/index.ts",
  ],

  // Output format
  format: ["esm"],
  target: "esnext",
  platform: "node",

  // Output directory
  outDir: "dist",

  // Generate declaration files
  dts: true,

  // Source maps for debugging
  sourcemap: true,

  // Clean output directory
  clean: true,

  // Split chunks for parallel loading
  splitting: true,

  // Tree shaking for smaller bundles
  treeshake: true,

  // Minification (disabled for dev, enable for production)
  minify: false,

  // esbuild options for maximum performance
  esbuildOptions(options) {
    // Use all available cores
    options.logLevel = "info";
    options.charset = "utf8";

    // Optimize for M4 Max
    options.legalComments = "none";
    options.treeShaking = true;
  },

  // Skip external dependencies (don't bundle node_modules)
  external: [
    "@noble/curves",
    "@noble/hashes",
    "libp2p",
    "@libp2p/*",
    "@chainsafe/*",
    "@multiformats/*",
    "level",
    "zod",
    "commander",
  ],

  // Watch mode settings
  watch: false,

  // Ignore patterns
  ignoreWatch: ["**/node_modules/**", "**/dist/**", "**/.git/**"],
});
