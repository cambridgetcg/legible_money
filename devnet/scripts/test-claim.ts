#!/usr/bin/env npx ts-node

/**
 * Test Claim Submission
 *
 * Submits a test claim to the devnet to verify the PoT system.
 */

import { sign } from "../../core/crypto/signatures.js";

const RPC_ENDPOINT = process.env.RPC_ENDPOINT || "http://localhost:8545";

// Faucet account keys (from devnet/secrets/faucet.key)
const FAUCET = {
  did: "did:lgm-devnet:faucet",
  publicKey: "8a2bc7e44e4e3dd9e29f730cd8799ca55184a7686c7a35f210d68cb7f42ceebd",
  privateKey:
    "1c799e340883cd53a1cbc0c4eb45864b03a64d637bb7d35c5a1dcadaf3ba7325",
};

interface RpcResponse {
  jsonrpc: string;
  id: number;
  result?: any;
  error?: any;
}

async function rpcCall(method: string, params: unknown): Promise<RpcResponse> {
  const response = await fetch(RPC_ENDPOINT, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      jsonrpc: "2.0",
      method,
      params,
      id: Date.now(),
    }),
  });
  return response.json() as Promise<RpcResponse>;
}

async function main() {
  console.log("🧪 Testing Claim Submission to Devnet\n");
  console.log(`RPC Endpoint: ${RPC_ENDPOINT}\n`);

  // 1. Check network health
  console.log("1. Checking network health...");
  const health = await rpcCall("network_health", {});
  console.log("   Health:", health.result || health.error);

  // 2. Check faucet balance
  console.log("\n2. Checking faucet balance...");
  const balance = await rpcCall("account_getBalance", { did: FAUCET.did });
  console.log("   Balance:", balance.result || balance.error);

  // 3. Create and sign a test claim
  console.log("\n3. Creating test claim...");

  const claim = "2 + 2 = 4";
  const domain = "math.arithmetic";
  const stake = "100000000000000000000"; // 100 LGM
  const nonce = "1";
  // Reference the zero axiom from genesis
  const references: string[] = ["axiom:math:zero"];

  const payload = JSON.stringify({
    claim,
    domain,
    stake,
    references,
    nonce,
  });

  console.log("   Claim:", claim);
  console.log("   Domain:", domain);
  console.log("   Stake:", stake, "(100 LGM)");

  // Sign the payload
  const payloadBytes = new TextEncoder().encode(payload);
  const signature = sign(payloadBytes, FAUCET.privateKey);

  console.log("   Signature:", signature.slice(0, 32) + "...");

  // 4. Submit the claim
  console.log("\n4. Submitting claim...");
  const submitResult = await rpcCall("knowledge_submitClaim", {
    claim,
    domain,
    stake,
    references,
    nonce,
    signature,
    sender: FAUCET.publicKey,
    senderDid: FAUCET.did, // Use faucet's named DID
  });

  if (submitResult.error) {
    console.log("   ❌ Error:", submitResult.error);
  } else {
    console.log("   ✅ Claim submitted successfully!");
    console.log("   Claim ID:", submitResult.result?.claimId);
    console.log(
      "   Verification Round:",
      submitResult.result?.verificationRoundId,
    );
    console.log("   TX Hash:", submitResult.result?.txHash);
  }

  // 5. Check facts in the domain
  console.log("\n5. Checking facts in math.arithmetic domain...");
  const facts = await rpcCall("knowledge_getFacts", {
    domain: "math.arithmetic",
    limit: 10,
  });
  console.log("   Facts:", facts.result || facts.error);

  console.log("\n✅ Test complete!");
}

main().catch(console.error);
