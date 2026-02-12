/**
 * Vitest Configuration - Optimized for Apple M4 Max
 *
 * Hardware: 14 CPU cores (10P + 4E), 32 GPU cores, 36GB RAM
 *
 * Optimizations:
 * - Parallel test execution across all performance cores
 * - Thread pool for maximum throughput
 * - Isolated workers to prevent memory leaks
 * - Native ESM for faster module loading
 */

import { defineConfig } from "vitest/config";

// M4 Max has 10 performance cores - use them all for testing
const PERFORMANCE_CORES = 10;

export default defineConfig({
  test: {
    // Use all performance cores for parallel testing
    pool: "threads",
    poolOptions: {
      threads: {
        // Use performance cores, leave efficiency cores for system
        minThreads: PERFORMANCE_CORES,
        maxThreads: PERFORMANCE_CORES,
        // Isolate tests to prevent memory leaks
        isolate: true,
        // Use native V8 thread pool
        useAtomics: true,
      },
    },

    // Test execution settings
    testTimeout: 30000,
    hookTimeout: 10000,

    // File patterns (includes .prop.ts for property-based tests)
    include: [
      "tests/**/*.test.ts",
      "tests/**/*.spec.ts",
      "tests/**/*.prop.ts",
      "genesis/tests/**/*.test.ts",
      "daemon/tests/**/*.test.ts",
    ],
    exclude: ["node_modules", "dist"],

    // Coverage configuration
    coverage: {
      provider: "v8",
      reporter: ["text", "json", "html"],
      reportsDirectory: "./coverage",
      // Parallel coverage collection
      all: true,
    },

    // Reporter for parallel output
    reporters: ["default"],
    outputFile: {
      json: "./test-results/results.json",
    },

    // Global test setup
    globals: true,
    environment: "node",

    // Cache for faster subsequent runs
    cache: {
      dir: "./node_modules/.vitest",
    },

    // Sequence for deterministic parallel runs
    sequence: {
      shuffle: false,
      concurrent: true,
    },

    // Memory optimization
    maxConcurrency: PERFORMANCE_CORES,
  },

  // Build optimization
  esbuild: {
    target: "esnext",
    platform: "node",
    // Use all cores for transformation
    treeShaking: true,
  },
});
