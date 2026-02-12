/**
 * Evaluation Daemon Entry Point
 *
 * Starts the Claude-powered evaluation daemon for Legible Money validators.
 *
 * Usage:
 *   npx tsx daemon/src/index.ts --config ./daemon-config.json
 *
 * Environment variables:
 *   ANTHROPIC_API_KEY          - Required: Anthropic API key for Claude
 *   LGM_RPC_ENDPOINT           - RPC endpoint (default: http://localhost:8545)
 *   LGM_WS_ENDPOINT            - WebSocket endpoint (default: ws://localhost:8546)
 *   LGM_VALIDATOR_KEY_PATH     - Path to validator operational key JSON
 *   LGM_VALIDATOR_ADDRESS      - Validator bech32 address
 *   LGM_AGENT_DID              - Agent DID (did:lgm:...)
 *   LGM_DATA_DIR               - Data directory (default: ./data)
 *   LGM_LOG_LEVEL              - Log level: debug|info|warn|error (default: info)
 */

import { loadConfig } from "./config.js";
import { EvaluationDaemon } from "./daemon.js";

async function main(): Promise<void> {
  // Validate ANTHROPIC_API_KEY is set
  if (!process.env.ANTHROPIC_API_KEY) {
    console.error("Error: ANTHROPIC_API_KEY environment variable is required");
    console.error(
      "Set it to your Anthropic API key, or use an invalid value for deterministic-only mode",
    );
    process.exit(1);
  }

  const config = loadConfig(process.argv);
  const daemon = await EvaluationDaemon.create(config);

  // Graceful shutdown (pattern from devnet/cmd/node.ts)
  const shutdown = async () => {
    console.log("\nShutting down...");
    await daemon.stop();
    process.exit(0);
  };

  process.on("SIGINT", shutdown);
  process.on("SIGTERM", shutdown);

  process.on("uncaughtException", (err) => {
    console.error("Uncaught exception:", err);
    daemon.stop().finally(() => process.exit(1));
  });

  process.on("unhandledRejection", (reason) => {
    console.error("Unhandled rejection:", reason);
  });

  await daemon.start();
}

main().catch((err) => {
  console.error("Fatal error:", err);
  process.exit(1);
});
