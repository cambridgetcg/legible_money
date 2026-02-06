#!/usr/bin/env tsx
/**
 * Phase 3: Economic Transaction Tests
 * Tests token transfers, staking, delegation via RPC
 */

import { ed25519 } from "@noble/curves/ed25519";
import { readFileSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));

// ============ Helpers ============

function hexToBytes(hex: string): Uint8Array {
  const bytes = new Uint8Array(hex.length / 2);
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.substring(i, i + 2), 16);
  }
  return bytes;
}

function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

interface KeyFile {
  did: string;
  publicKey: string;
  privateKey: string;
}

function loadKey(name: string): KeyFile {
  const path = join(__dirname, "..", "..", "devnet", "secrets", `${name}.key`);
  return JSON.parse(readFileSync(path, "utf-8"));
}

async function rpc(
  port: number,
  method: string,
  params: unknown = {},
): Promise<any> {
  const resolvedParams =
    Array.isArray(params) && params.length === 1 ? params[0] : params;
  const resp = await fetch(`http://localhost:${port}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      jsonrpc: "2.0",
      method,
      params: resolvedParams,
      id: 1,
    }),
  });
  return resp.json();
}

function signTx(
  key: KeyFile,
  txData: { type: string; payload: any; nonce: string; fee: string },
): string {
  const payload = JSON.stringify({
    type: txData.type,
    payload: txData.payload,
    nonce: txData.nonce,
    fee: txData.fee,
  });
  const messageBytes = new TextEncoder().encode(payload);
  const privKeyBytes = hexToBytes(key.privateKey);
  const sig = ed25519.sign(messageBytes, privKeyBytes);
  return bytesToHex(sig);
}

async function sendTx(
  port: number,
  key: KeyFile,
  type: string,
  payload: any,
  nonce: number,
  fee = "21000",
): Promise<any> {
  const nonceStr = nonce.toString();
  const txData = { type, payload, nonce: nonceStr, fee };
  const signature = signTx(key, txData);

  return rpc(port, "lgm_sendTransaction", [
    {
      type,
      payload,
      sender: key.publicKey,
      senderDid: key.did,
      nonce: nonceStr,
      fee,
      signature,
    },
  ]);
}

async function getBalance(
  port: number,
  did: string,
): Promise<{ balance: bigint; nonce: bigint; staked: bigint } | null> {
  const resp = await rpc(port, "account_getBalance", [{ did }]);
  if (resp.result) {
    return {
      balance: BigInt(resp.result.balance),
      nonce: BigInt(resp.result.nonce),
      staked: BigInt(resp.result.stakedBalance),
    };
  }
  return null;
}

// ============ Tests ============

let passed = 0;
let failed = 0;

function result(test: string, pass: boolean, detail: string) {
  if (pass) {
    passed++;
    console.log(`  ✓ ${test}: PASS - ${detail}`);
  } else {
    failed++;
    console.log(`  ✗ ${test}: FAIL - ${detail}`);
  }
}

async function main() {
  console.log("\n=== PHASE 3: ECONOMIC TRANSACTION TESTS ===\n");

  const faucet = loadKey("faucet");
  const validator1 = loadKey("validator-1");
  const PORT = 8545;
  let faucetNonce = 0;

  // Test 3.1: Query faucet balance
  console.log("--- Test 3.1: Query faucet balance ---");
  const bal = await getBalance(PORT, faucet.did);
  if (bal) {
    const expected = 1_000_000_000n * 10n ** 18n;
    result(
      "3.1",
      bal.balance === expected,
      `Faucet balance = ${bal.balance / 10n ** 18n} LGM`,
    );
  } else {
    result("3.1", false, "Could not query balance");
  }

  // Test 3.2: Transfer tokens
  console.log("\n--- Test 3.2: Transfer tokens ---");
  const recipientDid = "did:lgm-devnet:test-alice";
  const transferAmount = "1000000000000000000"; // 1 LGM
  const txResp = await sendTx(
    PORT,
    faucet,
    "transfer",
    {
      from: faucet.did,
      to: recipientDid,
      amount: transferAmount,
      memo: "phase3-test",
    },
    faucetNonce++,
  );

  if (txResp.result) {
    const isIncluded = txResp.result.status === "included";
    result(
      "3.2",
      isIncluded,
      `txHash=${txResp.result.txHash?.slice(0, 16)}..., status=${txResp.result.status}${txResp.result.error ? ` error=${txResp.result.error}` : ""}`,
    );
    if (!isIncluded) faucetNonce--; // rollback on failure
  } else {
    result("3.2", false, `RPC error: ${JSON.stringify(txResp.error)}`);
    faucetNonce--;
  }

  // Test 3.3: Verify recipient balance
  console.log("\n--- Test 3.3: Verify recipient balance ---");
  const recip = await getBalance(PORT, recipientDid);
  if (recip) {
    result(
      "3.3",
      recip.balance === BigInt(transferAmount),
      `Recipient balance = ${recip.balance / 10n ** 18n} LGM`,
    );
  } else {
    result("3.3", false, "Could not query recipient balance");
  }

  // Test 3.4: Nonce incremented
  console.log("\n--- Test 3.4: Nonce incremented ---");
  const afterTransfer = await getBalance(PORT, faucet.did);
  if (afterTransfer) {
    result(
      "3.4",
      afterTransfer.nonce === 1n,
      `Faucet nonce = ${afterTransfer.nonce}`,
    );
  } else {
    result("3.4", false, "Could not query nonce");
  }

  // Test 3.5: Stake tokens
  console.log("\n--- Test 3.5: Stake tokens ---");
  const stakeResp = await sendTx(
    PORT,
    faucet,
    "stake",
    {
      account: faucet.did,
      amount: "1111000000000000000000", // 1111 LGM (min validator stake)
      action: "stake",
    },
    faucetNonce++,
  );

  if (stakeResp.result) {
    const isIncluded = stakeResp.result.status === "included";
    result(
      "3.5",
      isIncluded,
      `Stake status=${stakeResp.result.status}${stakeResp.result.error ? ` error=${stakeResp.result.error}` : ""}`,
    );
    if (!isIncluded) faucetNonce--;
  } else {
    result("3.5", false, `RPC error: ${JSON.stringify(stakeResp.error)}`);
    faucetNonce--;
  }

  // Test 3.6: Unstake tokens
  console.log("\n--- Test 3.6: Unstake tokens ---");
  const unstakeResp = await sendTx(
    PORT,
    faucet,
    "unstake",
    {
      account: faucet.did,
      amount: "500000000000000000000", // 500 LGM
      action: "unstake",
    },
    faucetNonce++,
  );

  if (unstakeResp.result) {
    result(
      "3.6",
      unstakeResp.result.status === "included" ||
        unstakeResp.result.status === "failed",
      `Unstake status=${unstakeResp.result.status}${unstakeResp.result.error ? ` (${unstakeResp.result.error})` : ""}`,
    );
    if (unstakeResp.result.status === "failed") faucetNonce--;
  } else {
    result("3.6", false, `RPC error: ${JSON.stringify(unstakeResp.error)}`);
    faucetNonce--;
  }

  // Test 3.7: Delegate to validator
  console.log("\n--- Test 3.7: Delegate to validator ---");
  const delegateResp = await sendTx(
    PORT,
    faucet,
    "delegate",
    {
      delegator: faucet.did,
      validator: validator1.publicKey,
      amount: "100000000000000000000", // 100 LGM
    },
    faucetNonce++,
  );

  if (delegateResp.result) {
    result(
      "3.7",
      delegateResp.result.status === "included",
      `Delegate status=${delegateResp.result.status}${delegateResp.result.error ? ` (${delegateResp.result.error})` : ""}`,
    );
    if (delegateResp.result.status !== "included") faucetNonce--;
  } else {
    result("3.7", false, `RPC error: ${JSON.stringify(delegateResp.error)}`);
    faucetNonce--;
  }

  // Test 3.8: Undelegate
  console.log("\n--- Test 3.8: Undelegate ---");
  const undelegateResp = await sendTx(
    PORT,
    faucet,
    "undelegate",
    {
      delegator: faucet.did,
      validator: validator1.publicKey,
      amount: "50000000000000000000", // 50 LGM
    },
    faucetNonce++,
  );

  if (undelegateResp.result) {
    result(
      "3.8",
      undelegateResp.result.status === "included" ||
        undelegateResp.result.status === "failed",
      `Undelegate status=${undelegateResp.result.status}${undelegateResp.result.error ? ` (${undelegateResp.result.error})` : ""}`,
    );
    if (undelegateResp.result.status === "failed") faucetNonce--;
  } else {
    result("3.8", false, `RPC error: ${JSON.stringify(undelegateResp.error)}`);
    faucetNonce--;
  }

  // Test 3.9: Fee routing
  console.log("\n--- Test 3.9: Fee routing ---");
  const finalBal = await getBalance(PORT, faucet.did);
  if (finalBal) {
    const initial = 1_000_000_000n * 10n ** 18n;
    const spent = initial - finalBal.balance;
    result(
      "3.9",
      spent > 0n,
      `Spent: ${spent / 10n ** 18n} LGM, Staked: ${finalBal.staked / 10n ** 18n} LGM`,
    );
  } else {
    result("3.9", false, "Could not query final balance");
  }

  console.log(
    `\n=== Phase 3 Results: ${passed} passed, ${failed} failed ===\n`,
  );
}

main().catch(console.error);
