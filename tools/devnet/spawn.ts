#!/usr/bin/env tsx
/**
 * Devnet Multi-Node Spawner - Optimized for Apple M4 Max
 *
 * Spawns multiple Legible Money nodes in parallel using worker threads.
 * Leverages all 10 performance cores for maximum throughput.
 *
 * Usage:
 *   npm run devnet:multi              # Default: 10 nodes
 *   npm run devnet:multi -- --nodes 5 # Custom count
 */

import {
  Worker,
  isMainThread,
  parentPort,
  workerData,
} from "node:worker_threads";
import { cpus } from "node:os";
import { fileURLToPath } from "node:url";

// Configuration
const PERFORMANCE_CORES = 10; // M4 Max performance cores
const DEFAULT_NODE_COUNT = PERFORMANCE_CORES;
const BASE_PORT = 9000;
const BASE_RPC_PORT = 8545;

interface NodeConfig {
  nodeId: number;
  p2pPort: number;
  rpcPort: number;
  dataDir: string;
  isValidator: boolean;
  bootstrapPeers: string[];
}

interface NodeStatus {
  nodeId: number;
  status: "starting" | "running" | "stopped" | "error";
  pid?: number;
  error?: string;
}

// ============ Worker Thread Code ============

if (!isMainThread) {
  // Worker thread - runs a single node
  const config = workerData as NodeConfig;

  // infrastructure/node module removed — use devnet/cmd/node.ts directly
  parentPort?.postMessage({
    type: "status",
    nodeId: config.nodeId,
    status: "error",
    error:
      "Multi-node spawner requires infrastructure/node module which has been removed. Use `npm run devnet` instead.",
  });
}

// ============ Main Thread Code ============

if (isMainThread) {
  const __filename = fileURLToPath(import.meta.url);

  class DevnetSpawner {
    private workers: Map<number, Worker> = new Map();
    private nodeStatuses: Map<number, NodeStatus> = new Map();
    private nodeListenAddresses: Map<number, string[]> = new Map();
    private nodeCount: number;

    constructor(nodeCount: number = DEFAULT_NODE_COUNT) {
      // Cap at available cores
      this.nodeCount = Math.min(nodeCount, cpus().length);
      console.log(`\n🚀 Legible Money Devnet - M4 Max Optimized`);
      console.log(`   Cores available: ${cpus().length}`);
      console.log(`   Nodes to spawn: ${this.nodeCount}\n`);
    }

    async spawn(): Promise<void> {
      const startTime = Date.now();

      // Generate node configurations
      const configs = this.generateConfigs();

      // Spawn all nodes in parallel
      const spawnPromises = configs.map((config) => this.spawnNode(config));

      await Promise.all(spawnPromises);

      const elapsed = Date.now() - startTime;
      console.log(`\n✅ All ${this.nodeCount} nodes started in ${elapsed}ms`);
      this.printStatus();

      // Connect peers to each other
      await this.connectPeers();

      // Handle shutdown
      process.on("SIGINT", () => this.shutdown());
      process.on("SIGTERM", () => this.shutdown());
    }

    private generateConfigs(): NodeConfig[] {
      const configs: NodeConfig[] = [];
      const bootstrapPeers: string[] = [];

      for (let i = 0; i < this.nodeCount; i++) {
        const config: NodeConfig = {
          nodeId: i,
          p2pPort: BASE_PORT + i,
          rpcPort: BASE_RPC_PORT + i,
          dataDir: `/tmp/legible-devnet/node-${i}`,
          isValidator: i < 5, // First 5 nodes are validators
          bootstrapPeers: [...bootstrapPeers],
        };

        // Add this node to bootstrap peers for subsequent nodes
        bootstrapPeers.push(`/ip4/127.0.0.1/tcp/${config.p2pPort}`);

        configs.push(config);
      }

      return configs;
    }

    private spawnNode(config: NodeConfig): Promise<void> {
      return new Promise((resolve, reject) => {
        const worker = new Worker(__filename, {
          workerData: config,
          // Register tsx loader in worker threads so .js → .ts resolution works
          execArgv: ["--import", "tsx"],
        });

        worker.on("message", (msg) => {
          if (msg.type === "status") {
            this.nodeStatuses.set(msg.nodeId, {
              nodeId: msg.nodeId,
              status: msg.status,
              pid: msg.pid,
              error: msg.error,
            });

            if (msg.status === "running") {
              if (msg.listenAddresses) {
                this.nodeListenAddresses.set(msg.nodeId, msg.listenAddresses);
              }
              console.log(`   ✓ Node ${msg.nodeId} running (PID: ${msg.pid})`);
              resolve();
            } else if (msg.status === "error") {
              console.error(`   ✗ Node ${msg.nodeId} failed: ${msg.error}`);
              reject(new Error(msg.error));
            }
          }
        });

        worker.on("error", (error) => {
          console.error(`   ✗ Node ${config.nodeId} worker error:`, error);
          reject(error);
        });

        worker.on("exit", (code) => {
          if (code !== 0) {
            console.log(`   Node ${config.nodeId} exited with code ${code}`);
          }
        });

        this.workers.set(config.nodeId, worker);
      });
    }

    private async connectPeers(): Promise<void> {
      // Collect all listen addresses (use only localhost TCP)
      const allAddresses: string[] = [];
      for (const [, addrs] of this.nodeListenAddresses) {
        const localhostAddr = addrs.find((a) => a.includes("127.0.0.1/tcp/"));
        if (localhostAddr) allAddresses.push(localhostAddr);
      }

      if (allAddresses.length < 2) {
        console.log("\n⚠️  Not enough peers to connect");
        return;
      }

      console.log(`\n🔗 Connecting ${allAddresses.length} peers...`);

      // Send all addresses to each worker
      for (const [nodeId, worker] of this.workers) {
        // Send addresses of OTHER nodes
        const otherAddresses = allAddresses.filter((addr) => {
          const nodeAddrs = this.nodeListenAddresses.get(nodeId) ?? [];
          return !nodeAddrs.includes(addr);
        });
        worker.postMessage({ type: "connect", addresses: otherAddresses });
      }

      // Wait for connections to establish
      await new Promise((resolve) => setTimeout(resolve, 2000));
      console.log("   Peer connections initiated");
    }

    private printStatus(): void {
      console.log("\n📊 Node Status:");
      console.log("─".repeat(50));

      for (const [nodeId, status] of this.nodeStatuses) {
        const icon = status.status === "running" ? "🟢" : "🔴";
        const role = nodeId < 5 ? "validator" : "full node";
        console.log(`   ${icon} Node ${nodeId}: ${status.status} (${role})`);
      }

      console.log("\n📡 RPC Endpoints:");
      for (let i = 0; i < this.nodeCount; i++) {
        console.log(`   http://localhost:${BASE_RPC_PORT + i}`);
      }
    }

    private async shutdown(): Promise<void> {
      console.log("\n\n🛑 Shutting down devnet...");

      const shutdownPromises = Array.from(this.workers.values()).map(
        (worker) =>
          new Promise<void>((resolve) => {
            worker.terminate().then(() => resolve());
          }),
      );

      await Promise.all(shutdownPromises);
      console.log("   All nodes terminated.\n");
      process.exit(0);
    }
  }

  // Parse command line arguments
  const args = process.argv.slice(2);
  let nodeCount = DEFAULT_NODE_COUNT;

  const nodesIndex = args.indexOf("--nodes");
  if (nodesIndex !== -1 && args[nodesIndex + 1]) {
    nodeCount = parseInt(args[nodesIndex + 1], 10);
  }

  // Run the spawner
  const spawner = new DevnetSpawner(nodeCount);
  spawner.spawn().catch(console.error);
}
