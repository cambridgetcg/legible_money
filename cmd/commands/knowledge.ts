/**
 * Knowledge Commands
 *
 * Commands for querying and submitting knowledge claims.
 */

import type { Command } from "commander";
import { getClient } from "../lib/rpc.js";
import { outputSuccess, formatTable } from "../lib/output.js";
import { wrapCommand } from "../lgm.js";
import { CLICommandError } from "../lib/errors.js";
import { loadWallet } from "../lib/wallet-loader.js";
import { AgentWallet } from "../../protocol/sdk/wallet.js";
import { EXIT_CODES, type GlobalOptions, type OutputMode } from "../types.js";

// ============ Commands ============

/**
 * lgm knowledge query [domain]
 * Query facts from the knowledge tree
 */
async function queryCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  domain: string | undefined,
  options: {
    search?: string;
    minConfidence?: string;
    status?: string;
    limit?: string;
    cursor?: string;
  },
): Promise<void> {
  const client = getClient(opts.endpoint);

  const params = {
    domain: domain || undefined,
    search: options.search,
    minConfidence: options.minConfidence
      ? parseFloat(options.minConfidence)
      : undefined,
    status: options.status as "active" | "challenged" | "all" | undefined,
    limit: options.limit ? parseInt(options.limit) : 50,
    cursor: options.cursor,
  };

  const result = await client.getFacts(params);

  // In human mode, show as table
  if (mode === "human" && result.data.facts.length > 0) {
    console.log(`\nKnowledge Query: ${domain || "all domains"}`);
    console.log("─".repeat(60));

    formatTable(
      ["ID", "Domain", "Confidence", "Status"],
      result.data.facts.map((f) => [
        (f.id || "").slice(0, 12) + "...",
        f.domain || "",
        f.confidence?.toFixed(3) || "N/A",
        f.status || "unknown",
      ]),
      mode,
    );

    console.log(`\nTotal: ${result.data.total} facts`);
    if (result.data.nextCursor) {
      console.log(`Next cursor: ${result.data.nextCursor}`);
    }
    return;
  }

  outputSuccess(
    "knowledge query",
    {
      domain: domain || "all",
      facts: result.data.facts,
      total: result.data.total,
      nextCursor: result.data.nextCursor,
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm knowledge get <factId>
 * Get a specific fact
 */
async function getFactCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  factId: string,
  options: { includeProof?: boolean; includeReferences?: boolean },
): Promise<void> {
  if (!factId) {
    throw new CLICommandError(
      "Fact ID required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm knowledge get <factId>",
    );
  }

  const client = getClient(opts.endpoint);
  const result = await client.getFact(
    factId,
    options.includeProof ?? false,
    options.includeReferences ?? false,
  );

  outputSuccess(
    "knowledge get",
    {
      factId,
      fact: result.data.fact,
      references: result.data.references,
      proof: result.data.proof,
      blockNumber: result.data.blockNumber,
    },
    {
      latencyMs: result.latencyMs,
      blockNumber: result.data.blockNumber,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm knowledge submit <claim>
 * Submit a new knowledge claim
 */
async function submitCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  claim: string | undefined,
  options: {
    domain: string;
    stake: string;
    references?: string;
    stdin?: boolean;
  },
): Promise<void> {
  // Handle stdin input
  let claimText = claim;
  if (options.stdin) {
    claimText = await readStdin();
  }

  if (!claimText) {
    throw new CLICommandError(
      "Claim text required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm knowledge submit <claim> --domain <domain> --stake <amount>",
    );
  }

  if (!options.domain) {
    throw new CLICommandError(
      "Domain required",
      EXIT_CODES.INVALID_INPUT,
      "Specify --domain <domain> for the knowledge domain",
    );
  }

  if (!options.stake) {
    throw new CLICommandError(
      "Stake amount required",
      EXIT_CODES.INVALID_INPUT,
      "Specify --stake <amount> (in LGM)",
    );
  }

  // Load wallet for signing
  const wallet = await loadWallet(opts);
  if (!wallet) {
    throw new CLICommandError(
      "Wallet required for claim submission",
      EXIT_CODES.AUTH_ERROR,
      "Provide --agent-id and --agent-secret, or --session-key",
    );
  }

  // Get current nonce
  const client = getClient(opts.endpoint);
  const accountResult = await client.getBalance(wallet.did);
  const nonce = accountResult.data.nonce;

  // Build and sign the claim
  const payload = {
    claim: claimText,
    domain: options.domain,
    stake: options.stake,
    references: options.references?.split(",").filter((r) => r) || [],
    nonce,
  };

  // AgentWallet has signTransaction
  const agentWallet = wallet as AgentWallet;
  const signature = await agentWallet.signTransaction(payload);

  // Submit
  const result = await client.submitClaim({
    claim: claimText,
    domain: options.domain,
    stake: options.stake,
    references: payload.references,
    signature,
    sender: wallet.publicKey,
    senderDid: wallet.did,
    nonce,
  });

  outputSuccess(
    "knowledge submit",
    {
      claimId: result.data.claimId,
      verificationRoundId: result.data.verificationRoundId,
      estimatedCompletionBlock: result.data.estimatedCompletionBlock,
      txHash: result.data.txHash,
      domain: options.domain,
      stake: options.stake,
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

// ============ Helpers ============

/**
 * Read from stdin
 */
async function readStdin(): Promise<string> {
  const chunks: Buffer[] = [];
  for await (const chunk of process.stdin) {
    chunks.push(Buffer.from(chunk));
  }
  return Buffer.concat(chunks).toString("utf8").trim();
}

// ============ Registration ============

export function registerKnowledgeCommands(program: Command): void {
  const knowledge = program
    .command("knowledge")
    .description("Knowledge tree queries and claim submission");

  knowledge
    .command("query [domain]")
    .description("Query facts from the knowledge tree")
    .option("--search <text>", "Search text")
    .option("--min-confidence <n>", "Minimum confidence (0-1)")
    .option("--status <status>", "Filter by status: active, challenged, all")
    .option("--limit <n>", "Maximum results", "50")
    .option("--cursor <cursor>", "Pagination cursor")
    .action(wrapCommand("knowledge query", queryCommand));

  knowledge
    .command("get <factId>")
    .description("Get a specific fact by ID")
    .option("--include-proof", "Include Merkle proof")
    .option("--include-references", "Include supporting/contradicting facts")
    .action(wrapCommand("knowledge get", getFactCommand));

  knowledge
    .command("submit [claim]")
    .description("Submit a new knowledge claim")
    .option("--domain <domain>", "Knowledge domain (required)")
    .option("--stake <amount>", "Stake amount in LGM (required)")
    .option("--references <ids>", "Comma-separated fact IDs as references")
    .option("--stdin", "Read claim from stdin")
    .action(wrapCommand("knowledge submit", submitCommand));
}
