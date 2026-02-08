#!/usr/bin/env npx tsx

/**
 * Test Bootstrap Fund Auto-Claim on Live DevNet
 *
 * Creates a fresh agent, funds it via faucet, then has the agent
 * send its first transaction to trigger the 0.222 LGM bootstrap claim.
 *
 * Usage: npx tsx devnet/scripts/test-bootstrap.ts
 */

import { domainHash } from "../../core/crypto/hashing.js";
import { generateKeypairFromSeed } from "../../core/crypto/keys.js";
import { sign } from "../../core/crypto/signatures.js";

const RPC = "http://localhost:8545";

async function rpc(
  method: string,
  params: Record<string, unknown>,
): Promise<any> {
  const res = await fetch(RPC, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ jsonrpc: "2.0", method, params, id: 1 }),
  });
  return res.json();
}

function getFaucetKeys() {
  const seed = domainHash("LGM.devnet.faucet.v1", "faucet");
  return generateKeypairFromSeed(seed.slice(0, 32));
}

function formatLGM(baseUnits: bigint): string {
  const whole = baseUnits / 10n ** 18n;
  const frac = baseUnits % 10n ** 18n;
  return `${whole}.${frac.toString().padStart(18, "0").slice(0, 3)} LGM`;
}

async function main() {
  console.log("=== Bootstrap Fund Auto-Claim Test ===\n");

  // Step 1: Health check
  console.log("1. Checking devnet health...");
  const healthRes = await fetch(RPC + "/health");
  const health = (await healthRes.json()) as Record<string, unknown>;
  console.log(`   Status: ${health.status}, Block: ${health.blockHeight}\n`);

  // Step 2: Derive a fresh agent (never transacted)
  console.log("2. Creating fresh agent...");
  const agentSeed = domainHash("LGM.test.agent.v1", "test-agent-bootstrap-1");
  const agent = generateKeypairFromSeed(agentSeed.slice(0, 32));
  const agentDid = "did:lgm:test-agent-1";
  console.log(`   DID: ${agentDid}`);
  console.log(`   Pubkey: ${agent.publicKey.slice(0, 16)}...\n`);

  // Step 3: Check bootstrap fund balance
  console.log("3. Checking bootstrap fund...");
  const fundBefore = await rpc("account_getBalance", {
    did: "did:lgm:genesis:bootstrap-fund",
  });
  const fundBeforeBalance = BigInt(fundBefore.result.balance);
  console.log(`   Balance: ${formatLGM(fundBeforeBalance)}\n`);

  // Step 4: Faucet sends 1 LGM to create the agent account
  console.log("4. Faucet sends 1 LGM to agent...");
  const faucet = getFaucetKeys();
  const amount = 1n * 10n ** 18n;
  const txPayload = JSON.stringify({
    type: "transfer",
    payload: { to: agentDid, amount: amount.toString() },
    nonce: "0",
    fee: "0",
  });
  const sig = sign(new TextEncoder().encode(txPayload), faucet.privateKey);

  const txResult = await rpc("lgm_sendTransaction", {
    type: "transfer",
    sender: faucet.publicKey,
    senderDid: "did:lgm-devnet:faucet",
    nonce: "0",
    fee: "0",
    payload: { to: agentDid, amount: amount.toString() },
    signature: sig,
  });

  if (txResult.error) {
    console.error(`   TX Error: ${JSON.stringify(txResult.error)}`);
    process.exit(1);
  }
  console.log(`   TX Hash: ${txResult.result?.txHash?.slice(0, 32)}...`);
  console.log(`   Status: ${txResult.result?.status}`);

  // Check agent balance
  const agentAfterFund = await rpc("account_getBalance", { did: agentDid });
  const agentFundedBalance = BigInt(agentAfterFund.result?.balance || "0");
  console.log(`   Agent balance: ${formatLGM(agentFundedBalance)}\n`);

  // Step 5: Agent sends first tx (nonce 0) — should trigger bootstrap
  console.log("5. Agent sends first transaction (triggers bootstrap claim)...");
  const agentTxPayload = JSON.stringify({
    type: "transfer",
    payload: { to: "did:lgm-devnet:faucet", amount: "1" },
    nonce: "0",
    fee: "0",
  });
  const agentSig = sign(
    new TextEncoder().encode(agentTxPayload),
    agent.privateKey,
  );

  const agentTxResult = await rpc("lgm_sendTransaction", {
    type: "transfer",
    sender: agent.publicKey,
    senderDid: agentDid,
    nonce: "0",
    fee: "0",
    payload: { to: "did:lgm-devnet:faucet", amount: "1" },
    signature: agentSig,
  });

  if (agentTxResult.error) {
    console.error(`   TX Error: ${JSON.stringify(agentTxResult.error)}`);
    process.exit(1);
  }
  console.log(`   TX Hash: ${agentTxResult.result?.txHash?.slice(0, 32)}...`);
  console.log(`   Status: ${agentTxResult.result?.status}`);
  if (agentTxResult.result?.bootstrapClaimed) {
    console.log(
      `   Bootstrap claimed: ${agentTxResult.result.bootstrapClaimed} base units`,
    );
  }
  console.log();

  // Step 6: Check agent balance after first tx
  console.log("6. Checking agent balance after first tx...");
  const agentFinal = await rpc("account_getBalance", { did: agentDid });
  const agentFinalBalance = BigInt(agentFinal.result?.balance || "0");
  console.log(`   Balance: ${formatLGM(agentFinalBalance)}`);
  console.log(`   Nonce: ${agentFinal.result?.nonce}\n`);

  // Step 7: Check bootstrap fund after
  console.log("7. Checking bootstrap fund after...");
  const fundAfter = await rpc("account_getBalance", {
    did: "did:lgm:genesis:bootstrap-fund",
  });
  const fundAfterBalance = BigInt(fundAfter.result.balance);
  const fundDiff = fundBeforeBalance - fundAfterBalance;
  console.log(`   Balance: ${formatLGM(fundAfterBalance)}`);
  console.log(`   Deducted: ${fundDiff.toString()} base units`);

  const BOOTSTRAP_AMOUNT = 222000000000000000n; // 0.222 LGM
  if (fundDiff === BOOTSTRAP_AMOUNT) {
    console.log(`   = 0.222 LGM (correct!)\n`);
  } else if (fundDiff === 0n) {
    console.log(`   = No deduction (bootstrap did NOT fire)\n`);
  } else {
    console.log(`   = ${formatLGM(fundDiff)} (unexpected amount)\n`);
  }

  // Step 8: Verdict
  // Agent started with 1 LGM, sent 1 wei, should have received 0.222 LGM bootstrap
  // Expected: 1 LGM + 0.222 LGM - 1 wei = ~1.222 LGM
  const expectedMin = amount + BOOTSTRAP_AMOUNT - 100n; // small tolerance
  if (agentFinalBalance >= expectedMin) {
    console.log("   RESULT: PASS — Bootstrap auto-claim of 0.222 LGM works!");
    console.log(
      `   Agent received ${formatLGM(agentFinalBalance)} (expected ~1.222 LGM)`,
    );
  } else if (agentFinalBalance > amount) {
    console.log("   RESULT: PARTIAL — Agent got extra but amount differs");
    console.log(`   Got ${formatLGM(agentFinalBalance)}, expected ~1.222 LGM`);
  } else {
    console.log("   RESULT: FAIL — Bootstrap auto-claim did NOT trigger");
    console.log(
      `   Agent has ${formatLGM(agentFinalBalance)}, expected ~1.222 LGM`,
    );
  }

  console.log("\n=== Test Complete ===");
}

main().catch((err) => {
  console.error("Fatal:", err);
  process.exit(1);
});
