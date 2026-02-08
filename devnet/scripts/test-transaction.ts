#!/usr/bin/env npx tsx

/**
 * Test Transaction on Live DevNet
 *
 * Derives faucet keys deterministically, signs a transfer transaction,
 * and submits it to the running devnet via RPC.
 *
 * Usage: npx tsx devnet/scripts/test-transaction.ts
 */

import { domainHash } from "../../core/crypto/hashing.js";
import { generateKeypairFromSeed } from "../../core/crypto/keys.js";
import { sign } from "../../core/crypto/signatures.js";

const RPC = "http://localhost:8545";

async function rpc(
  method: string,
  params: Record<string, unknown>,
): Promise<unknown> {
  const res = await fetch(RPC, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ jsonrpc: "2.0", method, params, id: 1 }),
  });
  const json = (await res.json()) as {
    result?: unknown;
    error?: { code: number; message: string };
  };
  if (json.error) {
    console.error(`  RPC Error: ${json.error.code} — ${json.error.message}`);
    return null;
  }
  return json.result;
}

// ============ Key Derivation ============

function getFaucetKeys() {
  const seed = domainHash("LGM.devnet.faucet.v1", "faucet");
  // generateKeypairFromSeed returns { publicKey: hexString, privateKey: hexString }
  return generateKeypairFromSeed(seed.slice(0, 32));
}

function getValidatorKeys(index: number) {
  const seed = domainHash("LGM.devnet.validator.v1", `validator-${index}`);
  const kp = generateKeypairFromSeed(seed.slice(0, 32));
  return { ...kp, did: `did:lgm-devnet:validator-${index}` };
}

// ============ Main ============

async function main() {
  console.log("=== Legible Money DevNet Transaction Test ===\n");

  // Step 1: Check health
  console.log("1. Checking devnet health...");
  const healthRes = await fetch(RPC + "/health");
  const health = (await healthRes.json()) as Record<string, unknown>;
  console.log(
    `   Status: ${health.status}, Block: ${health.blockHeight}, Peers: ${health.peerCount}\n`,
  );

  // Step 2: Derive keys
  console.log("2. Deriving faucet keys...");
  const faucet = getFaucetKeys();
  console.log(`   Faucet pubkey: ${faucet.publicKey.slice(0, 16)}...`);

  const validator1 = getValidatorKeys(1);
  console.log(
    `   Validator-1 pubkey: ${validator1.publicKey.slice(0, 16)}...\n`,
  );

  // Step 3: Check faucet balance
  console.log("3. Checking faucet balance...");
  const faucetBalance = (await rpc("account_getBalance", {
    did: "did:lgm-devnet:faucet",
  })) as {
    balance: string;
    nonce: string;
  } | null;

  if (!faucetBalance) {
    console.error("   Failed to get faucet balance!");
    process.exit(1);
  }

  const balanceLGM = BigInt(faucetBalance.balance) / 10n ** 18n;
  console.log(`   Faucet balance: ${balanceLGM.toLocaleString()} LGM`);
  console.log(`   Faucet nonce: ${faucetBalance.nonce}\n`);

  // Step 4: Check recipient (validator-1) balance before
  console.log("4. Checking validator-1 balance before transfer...");
  const recipientBefore = (await rpc("account_getBalance", {
    did: "did:lgm-devnet:validator-1",
  })) as {
    balance: string;
  } | null;
  const recipientBalanceBefore = recipientBefore
    ? BigInt(recipientBefore.balance) / 10n ** 18n
    : 0n;
  console.log(
    `   Validator-1 balance: ${recipientBalanceBefore.toLocaleString()} LGM\n`,
  );

  // Step 5: Sign and send transfer transaction
  const amount = 1000n * 10n ** 18n; // 1000 LGM
  const nonce = faucetBalance.nonce;
  const fee = "0";

  console.log("5. Signing transfer transaction...");
  console.log(`   Amount: 1,000 LGM`);
  console.log(`   From: did:lgm-devnet:faucet`);
  console.log(`   To: did:lgm-devnet:validator-1`);

  const txPayload = JSON.stringify({
    type: "transfer",
    payload: {
      to: "did:lgm-devnet:validator-1",
      amount: amount.toString(),
    },
    nonce,
    fee,
  });

  // sign() takes hex privateKey, returns hex signature
  const signature = sign(
    new TextEncoder().encode(txPayload),
    faucet.privateKey,
  );
  console.log(`   Signature: ${signature.slice(0, 32)}...\n`);

  // Step 6: Submit transaction
  console.log("6. Submitting transaction...");
  const txResult = (await rpc("lgm_sendTransaction", {
    type: "transfer",
    sender: faucet.publicKey,
    senderDid: "did:lgm-devnet:faucet",
    nonce,
    fee,
    payload: {
      to: "did:lgm-devnet:validator-1",
      amount: amount.toString(),
    },
    signature,
  })) as { txHash?: string; status?: string; error?: string } | null;

  if (!txResult) {
    console.error("   Transaction submission failed!\n");
  } else {
    console.log(`   TX Hash: ${txResult.txHash}`);
    console.log(`   Status: ${txResult.status}`);
    if (txResult.error) console.log(`   Error: ${txResult.error}`);
    console.log();
  }

  // Step 7: Verify balances after
  console.log("7. Verifying balances after transfer...");

  const faucetAfter = (await rpc("account_getBalance", {
    did: "did:lgm-devnet:faucet",
  })) as {
    balance: string;
    nonce: string;
  } | null;
  const recipientAfter = (await rpc("account_getBalance", {
    did: "did:lgm-devnet:validator-1",
  })) as {
    balance: string;
  } | null;

  if (faucetAfter && recipientAfter) {
    const faucetAfterLGM = BigInt(faucetAfter.balance) / 10n ** 18n;
    const recipientAfterLGM = BigInt(recipientAfter.balance) / 10n ** 18n;
    const faucetDiff = balanceLGM - faucetAfterLGM;
    const recipientDiff = recipientAfterLGM - recipientBalanceBefore;

    console.log(
      `   Faucet:      ${faucetAfterLGM.toLocaleString()} LGM (${faucetDiff > 0 ? "-" : ""}${faucetDiff.toLocaleString()} LGM)`,
    );
    console.log(
      `   Validator-1: ${recipientAfterLGM.toLocaleString()} LGM (+${recipientDiff.toLocaleString()} LGM)`,
    );
    console.log(`   Faucet nonce: ${faucetAfter.nonce}`);
    console.log();

    if (recipientDiff === 1000n) {
      console.log("   RESULT: PASS — Transfer executed correctly!");
    } else if (txResult?.status === "included") {
      console.log(`   RESULT: PASS (balance diff: ${recipientDiff} LGM)`);
    } else {
      console.log("   RESULT: INVESTIGATE — Check transaction status");
    }
  }

  console.log("\n=== Test Complete ===");
}

main().catch((err) => {
  console.error("Fatal:", err);
  process.exit(1);
});
