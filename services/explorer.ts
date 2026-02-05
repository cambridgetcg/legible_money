#!/usr/bin/env node

/**
 * Legible Money Block Explorer Service
 *
 * Simple block explorer with real-time updates via WebSocket.
 */

import { createServer, IncomingMessage, ServerResponse } from "http";
import { LegibleClient } from "../protocol/sdk/client.js";

// ============ Configuration ============

const PORT = parseInt(process.env.PORT || "3000");
const RPC_ENDPOINT = process.env.RPC_ENDPOINT || "http://localhost:8545";
const WS_ENDPOINT = process.env.WS_ENDPOINT || "ws://localhost:8546";
const CHAIN_NAME = process.env.CHAIN_NAME || "Legible Money";

// ============ State ============

interface ExplorerStats {
  latestBlock: number;
  totalTransactions: number;
  totalFacts: number;
  activeValidators: number;
  totalStaked: bigint;
}

let client: LegibleClient;
const stats: ExplorerStats = {
  latestBlock: 0,
  totalTransactions: 0,
  totalFacts: 0,
  activeValidators: 0,
  totalStaked: 0n,
};

// ============ API Handlers ============

async function getOverview(): Promise<object> {
  const health = await client.getNetworkHealth();

  return {
    network: CHAIN_NAME,
    chainId: health.chainId || "lgm-devnet-1",
    latestBlock: stats.latestBlock,
    blockTime: "2.521s",
    totalTransactions: stats.totalTransactions,
    totalFacts: stats.totalFacts,
    activeValidators: stats.activeValidators,
    totalStaked: stats.totalStaked.toString(),
    status: health.status,
  };
}

async function getBlocks(
  limit: number = 10,
  offset: number = 0,
): Promise<object[]> {
  const blocks = [];
  const latest = stats.latestBlock;

  for (let i = latest - offset; i > Math.max(0, latest - offset - limit); i--) {
    try {
      const block = await client.getBlock(BigInt(i));
      if (block) {
        blocks.push({
          number: block.header.number,
          timestamp: block.header.timestamp,
          proposer: block.header.proposer.slice(0, 16) + "...",
          transactions: block.transactions.length,
          verifications: block.verificationResults.length,
          hash: block.hash.slice(0, 16) + "...",
        });
      }
    } catch {
      // Block not found, skip
    }
  }

  return blocks;
}

async function getBlock(query: string): Promise<object | null> {
  try {
    let block;
    if (query.startsWith("0x") || query.length === 64) {
      block = await client.getBlock(query);
    } else {
      block = await client.getBlock(BigInt(query));
    }

    if (!block) return null;

    return {
      number: block.header.number,
      hash: block.hash,
      timestamp: block.header.timestamp,
      proposer: block.header.proposer,
      stateRoot: block.header.stateRoot,
      knowledgeRoot: block.header.knowledgeRoot,
      previousHash: block.header.previousHash,
      transactions: block.transactions.map((tx) => ({
        hash: tx.hash,
        type: tx.type,
        from: tx.from,
        to: tx.to,
        amount: tx.amount?.toString(),
      })),
      verifications: block.verificationResults.map((v) => ({
        claimId: v.claimId,
        outcome: v.outcome,
        confidence: v.confidence,
      })),
    };
  } catch {
    return null;
  }
}

async function getAccount(did: string): Promise<object | null> {
  try {
    const account = await client.getAccount(did);
    if (!account.account) return null;

    return {
      did,
      balance: account.account.balance.toString(),
      stakedBalance: account.account.stakedBalance.toString(),
      nonce: account.account.nonce,
      reputation: account.account.reputation,
      isValidator: account.account.stakedBalance > 0n,
    };
  } catch {
    return null;
  }
}

async function getFacts(
  domain?: string,
  limit: number = 20,
): Promise<object[]> {
  try {
    const facts = await client.searchFacts({ domain, limit });
    return facts.map((f) => ({
      id: f.id,
      claim: f.claim,
      domain: f.domain,
      confidence: f.confidence,
      status: f.status,
      verifierCount: f.verifierCount,
    }));
  } catch {
    return [];
  }
}

async function getFact(factId: string): Promise<object | null> {
  try {
    const fact = await client.getFact(factId, true);
    if (!fact) return null;

    return {
      id: fact.id,
      claim: fact.claim,
      domain: fact.domain,
      confidence: fact.confidence,
      status: fact.status,
      verifierCount: fact.verifierCount,
      challengeCount: fact.challengeCount,
      verifiedAt: fact.verifiedAt,
      references: fact.references,
    };
  } catch {
    return null;
  }
}

async function getValidators(): Promise<object[]> {
  try {
    const validators = await client.getValidators();
    return validators.map((v) => ({
      publicKey: v.publicKey.slice(0, 16) + "...",
      did: v.did,
      stake: v.stake.toString(),
      delegatedStake: v.delegatedStake?.toString() || "0",
      commission: v.commission,
      blocksProduced: v.blocksProduced,
      verificationAccuracy: v.verificationAccuracy,
      isActive: v.isActive,
    }));
  } catch {
    return [];
  }
}

// ============ HTTP Server ============

async function handleRequest(
  req: IncomingMessage,
  res: ServerResponse,
): Promise<void> {
  // CORS headers
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Methods", "GET, OPTIONS");
  res.setHeader("Access-Control-Allow-Headers", "Content-Type");

  if (req.method === "OPTIONS") {
    res.writeHead(204);
    res.end();
    return;
  }

  const url = new URL(req.url || "/", `http://${req.headers.host}`);

  try {
    // API routes
    if (url.pathname === "/api/overview" || url.pathname === "/") {
      const data = await getOverview();
      sendJson(res, 200, data);
      return;
    }

    if (url.pathname === "/api/blocks") {
      const limit = parseInt(url.searchParams.get("limit") || "10");
      const offset = parseInt(url.searchParams.get("offset") || "0");
      const data = await getBlocks(limit, offset);
      sendJson(res, 200, { blocks: data });
      return;
    }

    if (url.pathname.startsWith("/api/block/")) {
      const query = url.pathname.replace("/api/block/", "");
      const data = await getBlock(query);
      if (data) {
        sendJson(res, 200, data);
      } else {
        sendJson(res, 404, { error: "Block not found" });
      }
      return;
    }

    if (url.pathname.startsWith("/api/account/")) {
      const did = decodeURIComponent(url.pathname.replace("/api/account/", ""));
      const data = await getAccount(did);
      if (data) {
        sendJson(res, 200, data);
      } else {
        sendJson(res, 404, { error: "Account not found" });
      }
      return;
    }

    if (url.pathname === "/api/facts") {
      const domain = url.searchParams.get("domain") || undefined;
      const limit = parseInt(url.searchParams.get("limit") || "20");
      const data = await getFacts(domain, limit);
      sendJson(res, 200, { facts: data });
      return;
    }

    if (url.pathname.startsWith("/api/fact/")) {
      const factId = url.pathname.replace("/api/fact/", "");
      const data = await getFact(factId);
      if (data) {
        sendJson(res, 200, data);
      } else {
        sendJson(res, 404, { error: "Fact not found" });
      }
      return;
    }

    if (url.pathname === "/api/validators") {
      const data = await getValidators();
      sendJson(res, 200, { validators: data });
      return;
    }

    if (url.pathname === "/health") {
      sendJson(res, 200, { status: "healthy", uptime: process.uptime() });
      return;
    }

    if (url.pathname === "/metrics") {
      res.writeHead(200, { "Content-Type": "text/plain" });
      res.end(
        `
# HELP explorer_latest_block Latest block number
# TYPE explorer_latest_block gauge
explorer_latest_block ${stats.latestBlock}

# HELP explorer_total_facts Total facts in knowledge base
# TYPE explorer_total_facts gauge
explorer_total_facts ${stats.totalFacts}

# HELP explorer_active_validators Active validators
# TYPE explorer_active_validators gauge
explorer_active_validators ${stats.activeValidators}
`.trim(),
      );
      return;
    }

    // Simple HTML interface
    if (url.pathname === "/ui") {
      res.writeHead(200, { "Content-Type": "text/html" });
      res.end(getHtmlInterface());
      return;
    }

    sendJson(res, 404, { error: "Not found" });
  } catch (error) {
    const message = error instanceof Error ? error.message : "Internal error";
    sendJson(res, 500, { error: message });
  }
}

function sendJson(res: ServerResponse, status: number, data: object): void {
  res.writeHead(status, { "Content-Type": "application/json" });
  res.end(JSON.stringify(data, null, 2));
}

function getHtmlInterface(): string {
  return `<!DOCTYPE html>
<html>
<head>
  <title>${CHAIN_NAME} Explorer</title>
  <style>
    body { font-family: monospace; background: #1a1a2e; color: #eee; padding: 20px; }
    h1 { color: #00d9ff; }
    .card { background: #16213e; padding: 15px; margin: 10px 0; border-radius: 8px; }
    .label { color: #888; }
    .value { color: #00d9ff; }
    a { color: #00d9ff; }
    table { width: 100%; border-collapse: collapse; }
    th, td { padding: 8px; text-align: left; border-bottom: 1px solid #333; }
    th { color: #888; }
  </style>
</head>
<body>
  <h1>${CHAIN_NAME} Explorer</h1>
  <div class="card">
    <div><span class="label">API Endpoints:</span></div>
    <ul>
      <li><a href="/api/overview">/api/overview</a> - Network overview</li>
      <li><a href="/api/blocks">/api/blocks</a> - Recent blocks</li>
      <li>/api/block/:number - Block details</li>
      <li>/api/account/:did - Account details</li>
      <li><a href="/api/facts">/api/facts</a> - Recent facts</li>
      <li>/api/fact/:id - Fact details</li>
      <li><a href="/api/validators">/api/validators</a> - Validator set</li>
    </ul>
  </div>
  <div id="stats" class="card">Loading...</div>
  <script>
    fetch('/api/overview')
      .then(r => r.json())
      .then(data => {
        document.getElementById('stats').innerHTML =
          '<div><span class="label">Latest Block:</span> <span class="value">' + data.latestBlock + '</span></div>' +
          '<div><span class="label">Status:</span> <span class="value">' + data.status + '</span></div>' +
          '<div><span class="label">Validators:</span> <span class="value">' + data.activeValidators + '</span></div>' +
          '<div><span class="label">Facts:</span> <span class="value">' + data.totalFacts + '</span></div>';
      });
  </script>
</body>
</html>`;
}

// ============ Background Tasks ============

async function updateStats(): Promise<void> {
  try {
    const blockNumber = await client.getBlockNumber();
    stats.latestBlock = Number(blockNumber);

    const validators = await client.getValidators();
    stats.activeValidators = validators.filter((v) => v.isActive).length;
    stats.totalStaked = validators.reduce(
      (sum, v) => sum + BigInt(v.stake),
      0n,
    );
  } catch {
    // Ignore errors in background task
  }
}

// ============ Main ============

async function main(): Promise<void> {
  console.log(`${CHAIN_NAME} Explorer`);
  console.log("=".repeat(CHAIN_NAME.length + 9) + "\n");

  // Initialize client
  client = new LegibleClient({
    endpoint: RPC_ENDPOINT,
  });

  console.log(`RPC endpoint: ${RPC_ENDPOINT}`);
  console.log(`WS endpoint: ${WS_ENDPOINT}\n`);

  // Start background stats update
  await updateStats();
  setInterval(updateStats, 5000);

  // Start server
  const server = createServer(handleRequest);

  server.listen(PORT, () => {
    console.log(`Explorer listening on port ${PORT}`);
    console.log(`UI available at http://localhost:${PORT}/ui`);
  });

  // Graceful shutdown
  process.on("SIGINT", () => {
    console.log("\nShutting down...");
    server.close();
    process.exit(0);
  });

  process.on("SIGTERM", () => {
    console.log("\nShutting down...");
    server.close();
    process.exit(0);
  });
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
