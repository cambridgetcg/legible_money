#!/usr/bin/env tsx
/**
 * Mock RPC Server for CLI Testing
 *
 * A simple mock server that responds to CLI commands.
 * Used for development and testing without full consensus.
 */

import { createServer, type IncomingMessage, type ServerResponse } from "http";

const PORT = 8545;

// Mock state
const accounts: Map<
  string,
  { balance: bigint; stakedBalance: bigint; nonce: bigint }
> = new Map();
const facts: Map<
  string,
  {
    id: string;
    claim: string;
    domain: string;
    confidence: number;
    status: string;
  }
> = new Map();
const validators: Array<{
  publicKey: string;
  stake: bigint;
  isActive: boolean;
}> = [];

// Bootstrap fund state (0.222 LGM for first 1,001,001,000 agents)
const bootstrapClaims: Set<string> = new Set();
let bootstrapClaimCount = 0n;
let bootstrapTotalClaimed = 0n;

// Initialize some mock data
accounts.set("did:lgm-test:genesis", {
  balance: BigInt("1000000000000000000000000"), // 1M LGM
  stakedBalance: 0n,
  nonce: 0n,
});

facts.set("fact:001", {
  id: "fact:001",
  claim: "Water boils at 100°C at sea level",
  domain: "chemistry",
  confidence: 0.95,
  status: "active",
});

facts.set("fact:002", {
  id: "fact:002",
  claim: "The speed of light is approximately 299,792,458 m/s",
  domain: "physics",
  confidence: 0.99,
  status: "active",
});

validators.push({
  publicKey: "validator-001-pubkey",
  stake: BigInt("50000000000000000000000"), // 50k LGM
  isActive: true,
});

// RPC Handlers
const handlers: Record<string, (params: unknown) => unknown> = {
  network_health: () => ({
    status: "healthy",
    latestBlock: "12345",
    syncStatus: "synced",
    peerCount: 3,
    version: "0.1.0-mock",
  }),

  account_getBalance: (params: any) => {
    const account = accounts.get(params.did);
    if (!account) {
      // Auto-create account for testing
      accounts.set(params.did, { balance: 0n, stakedBalance: 0n, nonce: 0n });
      return { balance: "0", stakedBalance: "0", nonce: "0" };
    }
    return {
      balance: account.balance.toString(),
      stakedBalance: account.stakedBalance.toString(),
      nonce: account.nonce.toString(),
    };
  },

  account_getInfo: (params: any) => {
    const account = accounts.get(params.did);
    return {
      account: account
        ? {
            did: params.did,
            balance: account.balance.toString(),
            stakedBalance: account.stakedBalance.toString(),
            nonce: account.nonce.toString(),
            reputationScore: 0.5,
          }
        : null,
      blockNumber: "12345",
    };
  },

  agent_checkAccount: (params: any) => {
    const account = accounts.get(params.did);
    return {
      exists: !!account,
      balance: account?.balance.toString(),
      reputation: 0.5,
      isValidator: false,
    };
  },

  agent_getStatus: (params: any) => {
    const account = accounts.get(params.did);
    if (!account) throw { code: -32001, message: "Account not found" };
    return {
      account: {
        balance: account.balance.toString(),
        stakedBalance: account.stakedBalance.toString(),
        reputation: 0.5,
      },
      activity: {
        claimsSubmitted: 2,
        claimsAccepted: 1,
        verificationsCompleted: 5,
        verificationsCorrect: 4,
      },
      earnings: {
        totalEarned: "1000000000000000000", // 1 LGM
        pendingRewards: "500000000000000000", // 0.5 LGM
      },
      sessions: { active: 1 },
    };
  },

  knowledge_getFacts: (params: any) => {
    let results = Array.from(facts.values());
    if (params.domain) {
      results = results.filter((f) => f.domain === params.domain);
    }
    if (params.minConfidence) {
      results = results.filter((f) => f.confidence >= params.minConfidence);
    }
    return {
      facts: results.slice(0, params.limit || 50),
      total: results.length,
    };
  },

  knowledge_getFact: (params: any) => {
    const fact = facts.get(params.factId);
    return {
      fact: fact || null,
      blockNumber: "12345",
    };
  },

  knowledge_submitClaim: (params: any) => {
    const claimId = `claim:${Date.now()}`;
    facts.set(claimId, {
      id: claimId,
      claim: params.claim,
      domain: params.domain,
      confidence: 0.5, // Pending verification
      status: "pending",
    });
    return {
      claimId,
      verificationRoundId: `round:${Date.now()}`,
      estimatedCompletionBlock: 12355,
      txHash: `0x${Math.random().toString(16).slice(2)}`,
    };
  },

  validator_getSet: () => ({
    validators: validators.map((v) => ({
      publicKey: v.publicKey,
      stake: v.stake.toString(),
      isActive: v.isActive,
      verificationsCompleted: 100,
      verificationsCorrect: 95,
    })),
    totalStake: validators.reduce((sum, v) => sum + v.stake, 0n).toString(),
    activeCount: validators.filter((v) => v.isActive).length,
    epoch: "42",
  }),

  verification_getPending: () => ({
    assignments: [
      {
        roundId: "round:test-001",
        claimId: "claim:test-001",
        claim: "Test claim for verification",
        domain: "testing",
        commitDeadline: 12400,
        revealDeadline: 12500,
      },
    ],
  }),

  verification_submitCommitment: (_params: any) => ({
    accepted: true,
    commitDeadline: 12400,
  }),

  verification_submitReveal: (_params: any) => ({
    accepted: true,
    roundStatus: "completed",
  }),

  verification_getRound: (params: any) => ({
    round: {
      id: params.roundId,
      claimId: "claim:test-001",
      claim: "Test claim",
      domain: "testing",
      phase: "reveal",
      commitDeadline: 12400,
      revealDeadline: 12500,
      commitCount: 3,
      revealCount: 2,
    },
    result: null,
  }),

  tx_transfer: (params: any) => {
    const from = accounts.get(params.from);
    if (!from) throw { code: -32001, message: "Account not found" };

    const amount = BigInt(params.amount);
    if (from.balance - from.stakedBalance < amount) {
      throw { code: -32002, message: "Insufficient balance" };
    }

    // Execute transfer
    from.balance -= amount;
    from.nonce += 1n;

    let to = accounts.get(params.to);
    if (!to) {
      to = { balance: 0n, stakedBalance: 0n, nonce: 0n };
      accounts.set(params.to, to);
    }
    to.balance += amount;

    return { txHash: `0x${Math.random().toString(16).slice(2)}` };
  },

  tx_getStatus: () => ({
    status: "finalized",
    blockNumber: "12340",
  }),

  validator_stake: (_params: any) => ({
    txHash: `0x${Math.random().toString(16).slice(2)}`,
    activationBlock: 12400,
  }),

  validator_unstake: (_params: any) => ({
    txHash: `0x${Math.random().toString(16).slice(2)}`,
    unlockBlock: 12400 + 238000, // ~7 days
  }),

  validator_getRewards: () => ({
    pending: "500000000000000000", // 0.5 LGM
    claimed: "1000000000000000000", // 1 LGM
    lastClaimBlock: "12000",
  }),

  validator_claimRewards: () => ({
    txHash: `0x${Math.random().toString(16).slice(2)}`,
    amount: "500000000000000000",
  }),

  // Faucet for testing - funds any account from genesis
  faucet_fund: (params: any) => {
    const amount = BigInt(params.amount || "10000000000000000000000"); // Default 10k LGM
    const genesis = accounts.get("did:lgm-test:genesis")!;

    if (genesis.balance < amount) {
      throw { code: -32002, message: "Faucet depleted" };
    }

    let target = accounts.get(params.did);
    if (!target) {
      target = { balance: 0n, stakedBalance: 0n, nonce: 0n };
      accounts.set(params.did, target);
    }

    genesis.balance -= amount;
    target.balance += amount;

    return {
      txHash: `0x${Math.random().toString(16).slice(2)}`,
      funded: amount.toString(),
      newBalance: target.balance.toString(),
    };
  },

  // ============ Bootstrap Fund (0.222 LGM for first 1,001,001,000 agents) ============
  bootstrap_claim: (params: any) => {
    const BOOTSTRAP_AMOUNT = BigInt("222000000000000000"); // 0.222 LGM
    const did = params.did;

    // Check if already claimed
    if (bootstrapClaims.has(did)) {
      throw { code: -32003, message: "Already claimed bootstrap fund" };
    }

    // Check if fund exhausted
    if (bootstrapClaimCount >= 1_001_001_000n) {
      throw { code: -32004, message: "Bootstrap fund exhausted" };
    }

    // Process claim
    let target = accounts.get(did);
    if (!target) {
      target = { balance: 0n, stakedBalance: 0n, nonce: 0n };
      accounts.set(did, target);
    }

    target.balance += BOOTSTRAP_AMOUNT;
    bootstrapClaims.add(did);
    bootstrapClaimCount += 1n;
    bootstrapTotalClaimed += BOOTSTRAP_AMOUNT;

    return {
      success: true,
      txHash: `0x${Math.random().toString(16).slice(2)}`,
      amount: BOOTSTRAP_AMOUNT.toString(),
      amountFormatted: "0.222 LGM",
      newBalance: target.balance.toString(),
      claimNumber: bootstrapClaimCount.toString(),
      remainingClaims: (1_001_001_000n - bootstrapClaimCount).toString(),
    };
  },

  bootstrap_status: () => {
    const MAX_CLAIMS = 1_001_001_000n;
    const TOTAL_FUND = BigInt("222222222000000000000000000"); // 222,222,222 LGM

    return {
      isActive: bootstrapClaimCount < MAX_CLAIMS,
      claimCount: bootstrapClaimCount.toString(),
      maxClaims: MAX_CLAIMS.toString(),
      totalClaimed: bootstrapTotalClaimed.toString(),
      remainingFund: (TOTAL_FUND - bootstrapTotalClaimed).toString(),
      percentClaimed: Number((bootstrapClaimCount * 10000n) / MAX_CLAIMS) / 100,
      amountPerClaim: "222000000000000000",
      amountPerClaimFormatted: "0.222 LGM",
    };
  },

  knowledge_challengeFact: (_params: any) => ({
    challengeId: `challenge:${Date.now()}`,
    disputeRoundId: `dispute:${Date.now()}`,
    txHash: `0x${Math.random().toString(16).slice(2)}`,
  }),

  knowledge_getChallengeStatus: () => ({
    status: "pending",
    blockNumber: "12345",
  }),

  block_get: () => ({
    block: { number: "12345", hash: "0xabc...", timestamp: Date.now() },
  }),

  block_getLatest: () => ({
    block: { number: "12345", hash: "0xabc...", timestamp: Date.now() },
    number: "12345",
    hash: "0xabcdef1234567890",
  }),
};

// HTTP Server
const server = createServer(
  async (req: IncomingMessage, res: ServerResponse) => {
    // CORS headers
    res.setHeader("Access-Control-Allow-Origin", "*");
    res.setHeader("Access-Control-Allow-Methods", "POST, OPTIONS");
    res.setHeader("Access-Control-Allow-Headers", "Content-Type");
    res.setHeader("Content-Type", "application/json");

    if (req.method === "OPTIONS") {
      res.writeHead(200);
      res.end();
      return;
    }

    if (req.method !== "POST") {
      res.writeHead(405);
      res.end(JSON.stringify({ error: "Method not allowed" }));
      return;
    }

    // Read body
    const chunks: Buffer[] = [];
    for await (const chunk of req) {
      chunks.push(Buffer.from(chunk));
    }
    const body = Buffer.concat(chunks).toString("utf8");

    try {
      const request = JSON.parse(body);
      const { method, params, id } = request;

      console.log(
        `[RPC] ${method}`,
        params ? JSON.stringify(params).slice(0, 100) : "",
      );

      const handler = handlers[method];
      if (!handler) {
        res.writeHead(200);
        res.end(
          JSON.stringify({
            jsonrpc: "2.0",
            id,
            error: { code: -32601, message: `Method not found: ${method}` },
          }),
        );
        return;
      }

      try {
        const result = handler(params);
        res.writeHead(200);
        res.end(JSON.stringify({ jsonrpc: "2.0", id, result }));
      } catch (error: any) {
        res.writeHead(200);
        res.end(
          JSON.stringify({
            jsonrpc: "2.0",
            id,
            error: {
              code: error.code || -32603,
              message: error.message || "Internal error",
            },
          }),
        );
      }
    } catch {
      res.writeHead(400);
      res.end(JSON.stringify({ error: "Invalid JSON" }));
    }
  },
);

server.listen(PORT, "127.0.0.1", () => {
  console.log(`\n🚀 Mock RPC Server running at http://localhost:${PORT}`);
  console.log("\nMock data:");
  console.log("  - Genesis account: did:lgm-test:genesis (1M LGM)");
  console.log("  - Sample facts: chemistry, physics");
  console.log("  - 1 active validator");
  console.log("\nPress Ctrl+C to stop\n");
});

process.on("SIGINT", () => {
  console.log("\nShutting down...");
  server.close();
  process.exit(0);
});
