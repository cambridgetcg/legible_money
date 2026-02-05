#!/usr/bin/env tsx
/**
 * Performance Benchmark - Optimized for Apple M4 Max
 *
 * Tests CPU throughput, parallel processing, and cryptographic performance.
 * Uses all available cores to measure maximum capability.
 *
 * Usage:
 *   npx tsx tools/benchmark.ts
 *   npx tsx tools/benchmark.ts --crypto
 *   npx tsx tools/benchmark.ts --parallel
 */

import {
  Worker,
  isMainThread,
  parentPort,
  workerData,
} from "node:worker_threads";
import { cpus, totalmem, freemem } from "node:os";
import { performance } from "node:perf_hooks";

// ============ Configuration ============

const PERFORMANCE_CORES = 10;
const EFFICIENCY_CORES = 4;
const TOTAL_CORES = PERFORMANCE_CORES + EFFICIENCY_CORES;
const ITERATIONS = 100_000;
const PARALLEL_ITERATIONS = 1_000_000;

// ============ Types ============

interface BenchmarkResult {
  name: string;
  iterations: number;
  totalMs: number;
  opsPerSecond: number;
  perIterationUs: number;
}

// ============ Worker Code ============

if (!isMainThread) {
  const { task, iterations, workerId } = workerData;

  let result = 0;
  const start = performance.now();

  switch (task) {
    case "hash":
      // Simulate hashing workload
      for (let i = 0; i < iterations; i++) {
        let hash = workerId * 31 + i;
        for (let j = 0; j < 100; j++) {
          hash = ((hash << 5) - hash + j) | 0;
        }
        result += hash;
      }
      break;

    case "math":
      // CPU-intensive math
      for (let i = 0; i < iterations; i++) {
        result += Math.sin(i) * Math.cos(i) * Math.tan(i % 1000);
      }
      break;

    case "memory":
      // Memory allocation/deallocation
      for (let i = 0; i < iterations; i++) {
        const arr = new Uint8Array(1024);
        arr.fill(i % 256);
        result += arr[0];
      }
      break;
  }

  const elapsed = performance.now() - start;
  parentPort?.postMessage({ workerId, elapsed, result });
}

// ============ Main Benchmark Code ============

if (isMainThread) {
  console.log(`
╔══════════════════════════════════════════════════════════════╗
║       Legible Money Performance Benchmark                    ║
║       Optimized for Apple M4 Max                             ║
╚══════════════════════════════════════════════════════════════╝

System Information:
  CPU:     ${cpus()[0].model}
  Cores:   ${TOTAL_CORES} (${PERFORMANCE_CORES}P + ${EFFICIENCY_CORES}E)
  Memory:  ${(totalmem() / 1024 / 1024 / 1024).toFixed(1)} GB total
  Free:    ${(freemem() / 1024 / 1024 / 1024).toFixed(1)} GB available
`);

  const runBenchmark = async (
    name: string,
    fn: () => Promise<number> | number,
    iterations: number,
  ): Promise<BenchmarkResult> => {
    // Warmup
    for (let i = 0; i < 1000; i++) {
      await fn();
    }

    // Actual benchmark
    const start = performance.now();
    for (let i = 0; i < iterations; i++) {
      await fn();
    }
    const totalMs = performance.now() - start;

    return {
      name,
      iterations,
      totalMs,
      opsPerSecond: (iterations / totalMs) * 1000,
      perIterationUs: (totalMs / iterations) * 1000,
    };
  };

  const runParallelBenchmark = async (
    task: string,
    workerCount: number,
    iterationsPerWorker: number,
  ): Promise<BenchmarkResult> => {
    const start = performance.now();

    const workers = Array.from({ length: workerCount }, (_, i) => {
      return new Promise<{ elapsed: number }>((resolve, reject) => {
        const worker = new Worker(new URL(import.meta.url), {
          workerData: { task, iterations: iterationsPerWorker, workerId: i },
        });
        worker.on("message", (msg) => {
          worker.terminate();
          resolve(msg);
        });
        worker.on("error", reject);
      });
    });

    await Promise.all(workers);
    const totalMs = performance.now() - start;
    const totalIterations = workerCount * iterationsPerWorker;

    return {
      name: `${task} (${workerCount} workers)`,
      iterations: totalIterations,
      totalMs,
      opsPerSecond: (totalIterations / totalMs) * 1000,
      perIterationUs: (totalMs / totalIterations) * 1000,
    };
  };

  const printResult = (result: BenchmarkResult): void => {
    console.log(
      `  ${result.name.padEnd(35)} ${result.opsPerSecond.toFixed(0).padStart(12)} ops/sec  ${result.perIterationUs.toFixed(3).padStart(10)} µs/op`,
    );
  };

  const main = async (): Promise<void> => {
    const results: BenchmarkResult[] = [];

    // ============ Single-threaded benchmarks ============
    console.log("Single-threaded Benchmarks:");
    console.log("─".repeat(70));

    // JSON serialization
    const obj = { a: 1, b: "hello", c: [1, 2, 3], d: { nested: true } };
    results.push(
      await runBenchmark(
        "JSON serialize/parse",
        () => {
          JSON.parse(JSON.stringify(obj));
          return 1;
        },
        ITERATIONS,
      ),
    );
    printResult(results[results.length - 1]);

    // Array operations
    results.push(
      await runBenchmark(
        "Array sort (1000 items)",
        () => {
          const arr = Array.from({ length: 1000 }, () => Math.random());
          arr.sort((a, b) => a - b);
          return arr[0];
        },
        ITERATIONS / 10,
      ),
    );
    printResult(results[results.length - 1]);

    // Map operations
    results.push(
      await runBenchmark(
        "Map set/get (1000 ops)",
        () => {
          const map = new Map<number, number>();
          for (let i = 0; i < 1000; i++) {
            map.set(i, i * 2);
            map.get(i);
          }
          return map.size;
        },
        ITERATIONS / 10,
      ),
    );
    printResult(results[results.length - 1]);

    // BigInt operations
    results.push(
      await runBenchmark(
        "BigInt arithmetic",
        () => {
          let n = 1000000000000000000n;
          n = n * 2n + 1n;
          n = n / 3n;
          return Number(n % 1000n);
        },
        ITERATIONS,
      ),
    );
    printResult(results[results.length - 1]);

    // ============ Multi-threaded benchmarks ============
    console.log("\nMulti-threaded Benchmarks:");
    console.log("─".repeat(70));

    // Hash workload - scaling test
    for (const workers of [1, 4, 10, 14]) {
      results.push(
        await runParallelBenchmark(
          "hash",
          workers,
          PARALLEL_ITERATIONS / workers,
        ),
      );
      printResult(results[results.length - 1]);
    }

    // Math workload
    results.push(
      await runParallelBenchmark(
        "math",
        PERFORMANCE_CORES,
        PARALLEL_ITERATIONS / PERFORMANCE_CORES,
      ),
    );
    printResult(results[results.length - 1]);

    // Memory workload
    results.push(
      await runParallelBenchmark(
        "memory",
        PERFORMANCE_CORES,
        ITERATIONS / PERFORMANCE_CORES,
      ),
    );
    printResult(results[results.length - 1]);

    // ============ Summary ============
    console.log("\n");
    console.log("═".repeat(70));
    console.log("Summary:");
    console.log("─".repeat(70));

    const hashSingle = results.find((r) => r.name === "hash (1 workers)");
    const hashFull = results.find(
      (r) => r.name === `hash (${PERFORMANCE_CORES} workers)`,
    );

    if (hashSingle && hashFull) {
      const speedup = hashFull.opsPerSecond / hashSingle.opsPerSecond;
      console.log(`  Parallel speedup (10 cores): ${speedup.toFixed(2)}x`);
      console.log(
        `  Efficiency:                  ${((speedup / PERFORMANCE_CORES) * 100).toFixed(1)}%`,
      );
    }

    const maxOps = Math.max(...results.map((r) => r.opsPerSecond));
    console.log(
      `  Peak throughput:             ${(maxOps / 1_000_000).toFixed(2)}M ops/sec`,
    );
    console.log("");
  };

  main().catch(console.error);
}
