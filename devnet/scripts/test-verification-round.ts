#!/usr/bin/env npx tsx

/**
 * Test Full Verification Round
 *
 * Exercises the complete PoT verification flow:
 * 1. Submit a claim
 * 2. Check verification round created
 * 3. Validators submit commitments
 * 4. Validators submit reveals
 * 5. Check final consensus result
 */

import { sign, signDomainSeparated } from "../../core/crypto/signatures.js";
import {
  sha512_256Hex,
  domainHash,
  HASH_DOMAINS,
} from "../../core/crypto/hashing.js";
import { randomBytes } from "crypto";

const RPC_ENDPOINT = process.env.RPC_ENDPOINT || "http://localhost:8545";

// Faucet account (claim submitter)
const FAUCET = {
  did: "did:lgm-devnet:faucet",
  publicKey: "8a2bc7e44e4e3dd9e29f730cd8799ca55184a7686c7a35f210d68cb7f42ceebd",
  privateKey:
    "1c799e340883cd53a1cbc0c4eb45864b03a64d637bb7d35c5a1dcadaf3ba7325",
};

// Validator keys (from devnet/secrets/*.key)
const VALIDATORS = [
  {
    name: "validator-1",
    publicKey:
      "87b59695691cba3b493028ab00d5aee71ece5a2758b74186b1f158255ffbbd20",
    privateKey:
      "905668215e5c57d3a6c81c4c458e665cbd4e682645eacdcb4568e927f0b0f517",
  },
  {
    name: "validator-2",
    publicKey:
      "4835e4dbec8b45f45547b229fa02532a1a8f16734838d64ea89358a459c2de30",
    privateKey:
      "bc1ab2674796961b0b49eba653f06f380fef1b5c9c4701be25f26204e9b875e2",
  },
  {
    name: "validator-3",
    publicKey:
      "6daf8df73c7e806f485657cf8205ed6c561e50ca0166976998c819c6526958a9",
    privateKey:
      "4452cdff7b5b20a3a272cdee55103961d9940e2ec2833a0ca8d618ed38199340",
  },
];

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

function generateNonce(): string {
  return randomBytes(32).toString("hex");
}

function generateCommitment(
  roundId: string,
  verdict: string,
  confidence: number,
  nonce: string,
  referencesChecked: string[] = [],
): string {
  // Must match verification_round.ts addReveal verification:
  // const commitData = JSON.stringify({roundId, verdict, confidence, referencesChecked, nonce})
  // const expectedCommitment = sha512_256Hex(domainHash(HASH_DOMAINS.COMMITMENT, commitData).toString())
  const commitData = JSON.stringify({
    roundId,
    verdict,
    confidence,
    referencesChecked: [...referencesChecked].sort(),
    nonce,
  });
  const hash = domainHash(HASH_DOMAINS.COMMITMENT, commitData);
  return sha512_256Hex(hash.toString());
}

async function main() {
  console.log(
    "╔════════════════════════════════════════════════════════════════╗",
  );
  console.log(
    "║         PROOF OF TRUTH - FULL VERIFICATION ROUND TEST          ║",
  );
  console.log(
    "╚════════════════════════════════════════════════════════════════╝\n",
  );

  console.log(`RPC Endpoint: ${RPC_ENDPOINT}\n`);

  // ═══════════════════════════════════════════════════════════════════════════
  // STEP 1: Check network health
  // ═══════════════════════════════════════════════════════════════════════════
  console.log(
    "┌─────────────────────────────────────────────────────────────────┐",
  );
  console.log(
    "│ STEP 1: Check Network Health                                   │",
  );
  console.log(
    "└─────────────────────────────────────────────────────────────────┘",
  );

  const health = await rpcCall("network_health", {});
  if (health.error) {
    console.log("❌ Network unhealthy:", health.error);
    return;
  }
  console.log("✓ Network status:", health.result.status);
  console.log("  Block:", health.result.latestBlock);
  console.log("");

  // ═══════════════════════════════════════════════════════════════════════════
  // STEP 2: Submit a claim
  // ═══════════════════════════════════════════════════════════════════════════
  console.log(
    "┌─────────────────────────────────────────────────────────────────┐",
  );
  console.log(
    "│ STEP 2: Submit Claim                                           │",
  );
  console.log(
    "└─────────────────────────────────────────────────────────────────┘",
  );

  const claim = "The sum of angles in a triangle is 180 degrees";
  const domain = "math.geometry";
  const stake = "100000000000000000000"; // 100 LGM
  const claimNonce = "2";
  const references = ["axiom:math:euclidean"];

  const payload = JSON.stringify({
    claim,
    domain,
    stake,
    references,
    nonce: claimNonce,
  });

  const payloadBytes = new TextEncoder().encode(payload);
  const claimSignature = sign(payloadBytes, FAUCET.privateKey);

  console.log("  Claim:", claim);
  console.log("  Domain:", domain);
  console.log("  Stake: 100 LGM");

  const submitResult = await rpcCall("knowledge_submitClaim", {
    claim,
    domain,
    stake,
    references,
    nonce: claimNonce,
    signature: claimSignature,
    sender: FAUCET.publicKey,
    senderDid: FAUCET.did,
  });

  if (submitResult.error) {
    console.log("❌ Submit error:", submitResult.error);
    return;
  }

  const roundId = submitResult.result.verificationRoundId;
  console.log("✓ Claim submitted!");
  console.log("  Round ID:", roundId);
  console.log("");

  // ═══════════════════════════════════════════════════════════════════════════
  // STEP 3: Check verification round
  // ═══════════════════════════════════════════════════════════════════════════
  console.log(
    "┌─────────────────────────────────────────────────────────────────┐",
  );
  console.log(
    "│ STEP 3: Check Verification Round                               │",
  );
  console.log(
    "└─────────────────────────────────────────────────────────────────┘",
  );

  const roundResult = await rpcCall("verification_getRound", { roundId });
  if (roundResult.error) {
    console.log("❌ Round error:", roundResult.error);
    return;
  }

  const round = roundResult.result.round;
  if (!round) {
    console.log("❌ Round not found");
    return;
  }

  console.log("✓ Round found!");
  console.log("  Phase:", round.phase);
  console.log("  Selected Validators:", round.selectedValidators?.length || 0);
  console.log("  Commit Deadline: block", round.commitDeadline);
  console.log("  Reveal Deadline: block", round.revealDeadline);
  console.log("");

  // ═══════════════════════════════════════════════════════════════════════════
  // STEP 4: Validators submit commitments
  // ═══════════════════════════════════════════════════════════════════════════
  console.log(
    "┌─────────────────────────────────────────────────────────────────┐",
  );
  console.log(
    "│ STEP 4: Validators Submit Commitments                          │",
  );
  console.log(
    "└─────────────────────────────────────────────────────────────────┘",
  );

  // Store nonces for reveal phase
  const validatorNonces: Map<string, string> = new Map();
  const validatorVerdicts: Map<
    string,
    { verdict: string; confidence: number }
  > = new Map();

  for (const validator of VALIDATORS) {
    // Simulate validator deciding on verdict (in real scenario, they'd evaluate the claim)
    const verdict = "confirmed"; // All validators agree
    const confidence = 0.85 + Math.random() * 0.1; // 0.85-0.95
    const nonce = generateNonce();

    validatorNonces.set(validator.publicKey, nonce);
    validatorVerdicts.set(validator.publicKey, { verdict, confidence });

    // Generate commitment
    const commitment = generateCommitment(roundId, verdict, confidence, nonce);

    // Sign commitment with domain separation (must match verification_round.ts)
    // Domain: 'LGM.verification.commit.v1', Message: `${roundId}:${commitment}`
    const commitSignature = signDomainSeparated(
      "LGM.verification.commit.v1",
      `${roundId}:${commitment}`,
      validator.privateKey,
    );

    console.log(`  ${validator.name}: submitting commitment...`);

    const commitResult = await rpcCall("verification_submitCommitment", {
      roundId,
      commitment,
      validatorKey: validator.publicKey,
      signature: commitSignature,
    });

    if (commitResult.error) {
      console.log(`    ❌ Error: ${commitResult.error.message}`);
    } else {
      console.log(`    ✓ Accepted`);
    }
  }
  console.log("");

  // ═══════════════════════════════════════════════════════════════════════════
  // STEP 5: Check round status after commits
  // ═══════════════════════════════════════════════════════════════════════════
  console.log(
    "┌─────────────────────────────────────────────────────────────────┐",
  );
  console.log(
    "│ STEP 5: Check Round After Commits                              │",
  );
  console.log(
    "└─────────────────────────────────────────────────────────────────┘",
  );

  const roundAfterCommit = await rpcCall("verification_getRound", { roundId });
  if (roundAfterCommit.result?.round) {
    const r = roundAfterCommit.result.round;
    console.log("  Phase:", r.phase);
    console.log(
      "  Commitments received:",
      Object.keys(r.commitments || {}).length,
    );
  }
  console.log("");

  // ═══════════════════════════════════════════════════════════════════════════
  // STEP 5.5: Wait for commit phase to end
  // ═══════════════════════════════════════════════════════════════════════════
  console.log(
    "┌─────────────────────────────────────────────────────────────────┐",
  );
  console.log(
    "│ STEP 5.5: Wait for Commit Phase to End                         │",
  );
  console.log(
    "└─────────────────────────────────────────────────────────────────┘",
  );
  console.log("  Waiting for blocks to advance past commit deadline...");
  console.log("  (Block time: 2521ms, commit phase: 4 blocks = ~10 seconds)");

  // Wait for blocks to advance (4 blocks * 2521ms = ~10 seconds)
  // Add extra buffer for processing
  const waitTime = 15000; // 15 seconds
  console.log(`  Waiting ${waitTime / 1000} seconds...`);
  await new Promise((resolve) => setTimeout(resolve, waitTime));

  // Check block number
  const afterWait = await rpcCall("network_health", {});
  console.log(`  Block now: ${afterWait.result?.latestBlock}`);
  console.log("");

  // ═══════════════════════════════════════════════════════════════════════════
  // STEP 6: Validators submit reveals
  // ═══════════════════════════════════════════════════════════════════════════
  console.log(
    "┌─────────────────────────────────────────────────────────────────┐",
  );
  console.log(
    "│ STEP 6: Validators Submit Reveals                              │",
  );
  console.log(
    "└─────────────────────────────────────────────────────────────────┘",
  );

  for (const validator of VALIDATORS) {
    const nonce = validatorNonces.get(validator.publicKey)!;
    const { verdict, confidence } = validatorVerdicts.get(validator.publicKey)!;
    const referencesChecked: string[] = [];
    const reasoning = undefined;

    // Sign reveal with domain separation (must match verification_round.ts)
    const revealData = JSON.stringify({
      roundId,
      verdict,
      confidence,
      nonce,
      referencesChecked: [...referencesChecked].sort(),
      reasoning,
    });
    const revealSignature = signDomainSeparated(
      "LGM.verification.reveal.v1",
      revealData,
      validator.privateKey,
    );

    console.log(
      `  ${validator.name}: revealing vote (${verdict}, ${confidence.toFixed(2)})...`,
    );

    const revealResult = await rpcCall("verification_submitReveal", {
      roundId,
      verdict,
      confidence,
      nonce,
      validatorKey: validator.publicKey,
      signature: revealSignature,
    });

    if (revealResult.error) {
      console.log(`    ❌ Error: ${revealResult.error.message}`);
    } else {
      console.log(
        `    ✓ Accepted (round status: ${revealResult.result.roundStatus})`,
      );
    }
  }
  console.log("");

  // ═══════════════════════════════════════════════════════════════════════════
  // STEP 7: Check final round status
  // ═══════════════════════════════════════════════════════════════════════════
  console.log(
    "┌─────────────────────────────────────────────────────────────────┐",
  );
  console.log(
    "│ STEP 7: Check Final Round Status                               │",
  );
  console.log(
    "└─────────────────────────────────────────────────────────────────┘",
  );

  const finalRound = await rpcCall("verification_getRound", { roundId });
  if (finalRound.result?.round) {
    const r = finalRound.result.round;
    console.log("  Phase:", r.phase);
    console.log("  Commitments:", Object.keys(r.commitments || {}).length);
    console.log("  Reveals:", Object.keys(r.reveals || {}).length);

    if (r.result) {
      console.log("");
      console.log(
        "  ═══════════════════════════════════════════════════════════",
      );
      console.log("  VERIFICATION RESULT:");
      console.log(
        "  ═══════════════════════════════════════════════════════════",
      );
      console.log("  Outcome:", r.result.outcome);
      console.log("  Final Confidence:", r.result.finalConfidence?.toFixed(4));
      console.log(
        "  Participating Validators:",
        r.result.participatingValidators,
      );
    }
  }
  console.log("");

  // ═══════════════════════════════════════════════════════════════════════════
  // STEP 8: Check if fact was created
  // ═══════════════════════════════════════════════════════════════════════════
  console.log(
    "┌─────────────────────────────────────────────────────────────────┐",
  );
  console.log(
    "│ STEP 8: Check Facts in Domain                                  │",
  );
  console.log(
    "└─────────────────────────────────────────────────────────────────┘",
  );

  const facts = await rpcCall("knowledge_getFacts", { domain, limit: 10 });
  if (facts.result) {
    console.log(`  Facts in ${domain}:`, facts.result.total);
    for (const fact of facts.result.facts) {
      console.log(`    - ${fact.claimText.substring(0, 50)}...`);
      console.log(
        `      confidence: ${fact.confidence}, status: ${fact.status}`,
      );
    }
  }
  console.log("");

  // ═══════════════════════════════════════════════════════════════════════════
  // SUMMARY
  // ═══════════════════════════════════════════════════════════════════════════
  console.log(
    "╔════════════════════════════════════════════════════════════════╗",
  );
  console.log(
    "║                     TEST COMPLETE                              ║",
  );
  console.log(
    "╚════════════════════════════════════════════════════════════════╝",
  );
}

main().catch(console.error);
