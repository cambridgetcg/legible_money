/**
 * Daemon Configuration
 *
 * Zod-validated config schema for the evaluation daemon.
 * API key sourced from ANTHROPIC_API_KEY env var (Anthropic SDK default).
 */

import { z } from "zod";
import { readFileSync } from "node:fs";

export const DaemonConfigSchema = z.object({
  node: z
    .object({
      rpcEndpoint: z.string().default("http://localhost:8545"),
      wsEndpoint: z.string().default("ws://localhost:8546"),
    })
    .default({}),

  validator: z.object({
    operationalKeyPath: z.string(), // Path to Ed25519 private key JSON
    validatorAddress: z.string(), // Bech32 address
    agentDid: z.string(), // did:lgm:...
  }),

  claude: z
    .object({
      model: z.string().default("claude-sonnet-4-20250514"),
      maxTokens: z.number().int().positive().default(512),
      timeoutMs: z.number().int().positive().default(6000),
      temperature: z.number().min(0).max(1).default(0.0),
    })
    .default({}),

  storage: z
    .object({
      dataDir: z.string().default("./data"),
    })
    .default({}),

  logLevel: z.enum(["debug", "info", "warn", "error"]).default("info"),
});

export type DaemonConfig = z.infer<typeof DaemonConfigSchema>;

/**
 * Load and validate daemon config from a JSON file path or CLI args.
 *
 * Priority: CLI --config file > env vars > defaults
 */
export function loadConfig(argv: string[]): DaemonConfig {
  // Parse --config flag
  const configIdx = argv.indexOf("--config");
  let rawConfig: Record<string, unknown> = {};

  if (configIdx !== -1 && argv[configIdx + 1]) {
    const configPath = argv[configIdx + 1];
    const content = readFileSync(configPath, "utf-8");
    rawConfig = JSON.parse(content);
  }

  // Override from env vars
  if (process.env.LGM_RPC_ENDPOINT) {
    rawConfig.node = {
      ...((rawConfig.node as object) ?? {}),
      rpcEndpoint: process.env.LGM_RPC_ENDPOINT,
    };
  }
  if (process.env.LGM_WS_ENDPOINT) {
    rawConfig.node = {
      ...((rawConfig.node as object) ?? {}),
      wsEndpoint: process.env.LGM_WS_ENDPOINT,
    };
  }
  if (process.env.LGM_VALIDATOR_KEY_PATH) {
    rawConfig.validator = {
      ...((rawConfig.validator as object) ?? {}),
      operationalKeyPath: process.env.LGM_VALIDATOR_KEY_PATH,
    };
  }
  if (process.env.LGM_VALIDATOR_ADDRESS) {
    rawConfig.validator = {
      ...((rawConfig.validator as object) ?? {}),
      validatorAddress: process.env.LGM_VALIDATOR_ADDRESS,
    };
  }
  if (process.env.LGM_AGENT_DID) {
    rawConfig.validator = {
      ...((rawConfig.validator as object) ?? {}),
      agentDid: process.env.LGM_AGENT_DID,
    };
  }
  if (process.env.LGM_DATA_DIR) {
    rawConfig.storage = {
      ...((rawConfig.storage as object) ?? {}),
      dataDir: process.env.LGM_DATA_DIR,
    };
  }
  if (process.env.LGM_LOG_LEVEL) {
    rawConfig.logLevel = process.env.LGM_LOG_LEVEL;
  }

  return DaemonConfigSchema.parse(rawConfig);
}
