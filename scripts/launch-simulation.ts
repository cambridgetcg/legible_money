#!/usr/bin/env npx tsx
/**
 * Launch Simulation
 *
 * Boots the Legible Money protocol from genesis in-process and simulates
 * participating as a new user: requesting tokens, submitting claims,
 * watching verification rounds, and observing economic flows.
 *
 * Run: npx tsx scripts/launch-simulation.ts
 */

import { TestNetwork } from "../tests/framework/test_harness.js";
import type { DID } from "../core/agents/identity/did.js";

// ============ Formatting Helpers ============

const ONE_LGM = 10n ** 18n;

function formatLGM(wei: bigint): string {
  const whole = wei / ONE_LGM;
  const frac = (wei % ONE_LGM).toString().padStart(18, "0").slice(0, 4);
  return `${whole}.${frac} LGM`;
}

function banner(title: string): void {
  const line = "═".repeat(62);
  console.log(`\n╔${line}╗`);
  console.log(`║  ${title.padEnd(60)}║`);
  console.log(`╚${line}╝`);
}

function step(num: number, title: string): void {
  console.log(`\n┌${"─".repeat(62)}┐`);
  console.log(`│  STEP ${num}: ${title.padEnd(52)}│`);
  console.log(`└${"─".repeat(62)}┘`);
}

function info(label: string, value: string | number | bigint): void {
  const valStr = typeof value === "bigint" ? formatLGM(value) : String(value);
  console.log(`  ${label.padEnd(30)} ${valStr}`);
}

// ============ Main Simulation ============

async function main() {
  banner("LEGIBLE MONEY — LAUNCH SIMULATION");
  console.log("\n  Booting protocol from genesis...\n");

  // ── Create the network ──────────────────────────────────────────────
  const network = new TestNetwork({
    validatorCount: 3,
    fullNodeCount: 0,
    testAccounts: 3,
    validatorStake: 10_000n * ONE_LGM,
    deterministicMode: true,
    autoBlocks: false,
    blockInterval: 100,
    consensusParams: {
      blockReward: 11n * ONE_LGM,
      verificationReward: 1n * ONE_LGM,
      minClaimStake: 111n * ONE_LGM,
      targetBlockTimeMs: 2521,
      verificationRoundBlocks: 6,
      commitPhaseBlocks: 2,
      revealPhaseBlocks: 2,
    },
  });

  await network.init();
  await network.start();

  const state = network.getState();
  const consensus = network.getConsensus();

  // ═══════════════════════════════════════════════════════════════════
  step(1, "GENESIS STATE");
  // ═══════════════════════════════════════════════════════════════════

  console.log("\n  Validators:");
  for (const v of network.validators) {
    const acct = state.getAccount(v.did);
    const validator = state.getValidator(v.publicKey);
    info(`  ${v.name}`, acct?.balance ?? 0n);
    console.log(
      `    Stake: ${formatLGM(validator?.stake ?? 0n)}, DID: ${v.did}`,
    );
  }

  console.log("\n  Test Accounts:");
  for (const a of network.accounts) {
    const acct = state.getAccount(a.did);
    info(`  ${a.name}`, acct?.balance ?? 0n);
  }

  // Check genesis axioms
  const axiomFact = state.getFact("test:axiom:1");
  console.log(`\n  Genesis axioms: ${axiomFact ? "1 loaded" : "MISSING"}`);
  if (axiomFact) {
    console.log(
      `    "${axiomFact.claim}" (confidence: ${axiomFact.confidence})`,
    );
  }

  // ═══════════════════════════════════════════════════════════════════
  step(2, "MINE FIRST 5 BLOCKS (empty — no claims yet)");
  // ═══════════════════════════════════════════════════════════════════

  const blocks = await network.mineBlocks(5);
  console.log(`  Mined ${blocks.length} blocks`);
  console.log(`  Current block: ${network.getCurrentBlock()}`);

  // Check validator-0 balance (should have block rewards)
  const v0 = network.validators[0];
  const v0acct = state.getAccount(v0.did);
  info("  validator-0 balance after 5 blocks", v0acct?.balance ?? 0n);

  // ═══════════════════════════════════════════════════════════════════
  step(3, "SUBMIT A CLAIM — 'Water boils at 100°C'");
  // ═══════════════════════════════════════════════════════════════════

  const claimant = network.accounts[0];
  const claimantAcctBefore = state.getAccount(claimant.did);
  info("  Claimant balance before", claimantAcctBefore?.balance ?? 0n);

  let claimId: string;
  try {
    claimId = await network.submitClaim(
      "Water boils at 100 degrees Celsius at standard atmospheric pressure",
      "science.physics",
      claimant,
      111n * ONE_LGM,
    );
    console.log(`  Claim submitted! ID: ${claimId}`);
  } catch (error) {
    console.log(`  CLAIM FAILED: ${(error as Error).message}`);
    console.log("\n  This tells us something about the protocol constraints.");

    // Try with a validator instead (they have more funds)
    console.log("\n  Trying with validator-0 as claimant...");
    try {
      claimId = await network.submitClaim(
        "Water boils at 100 degrees Celsius at standard atmospheric pressure",
        "science.physics",
        v0,
        111n * ONE_LGM,
      );
      console.log(`  Claim submitted! ID: ${claimId}`);
    } catch (error2) {
      console.log(`  STILL FAILED: ${(error2 as Error).message}`);
      // Continue anyway to explore more
      claimId = "";
    }
  }

  const claimantAcctAfter = state.getAccount(claimant.did);
  info("  Claimant balance after", claimantAcctAfter?.balance ?? 0n);

  // ═══════════════════════════════════════════════════════════════════
  step(4, "VERIFICATION ROUND — Validators evaluate the claim");
  // ═══════════════════════════════════════════════════════════════════

  if (claimId) {
    // Check active rounds
    const activeRounds = consensus.getActiveRounds();
    console.log(`  Active verification rounds: ${activeRounds.length}`);

    if (activeRounds.length > 0) {
      const round = activeRounds[0];
      console.log(`  Round ID: ${round.id}`);
      console.log(`  Phase: ${round.phase}`);
      console.log(`  Start block: ${round.startBlock}`);
      console.log(`  Commit deadline: ${round.commitDeadline}`);
      console.log(`  Reveal deadline: ${round.revealDeadline}`);

      // Validators submit commitments (all confirm with high confidence)
      console.log("\n  Validators committing...");
      try {
        await network.completeVerificationRoundDeterministic(
          round.id,
          network.validators.map((v) => ({
            validator: v,
            verdict: "confirmed" as const,
            confidence: 0.9,
          })),
        );
        console.log("  Verification round completed!");
      } catch (error) {
        console.log(`  VERIFICATION ERROR: ${(error as Error).message}`);
      }

      // Mine a few more blocks to finalize
      await network.mineBlocks(3);

      // Check if claim became a fact
      const rounds = consensus.getActiveRounds();
      console.log(`  Remaining active rounds: ${rounds.length}`);
    }
  } else {
    console.log("  Skipping (no claim to verify)");
  }

  // ═══════════════════════════════════════════════════════════════════
  step(5, "CHECK KNOWLEDGE STATE — Did the fact register?");
  // ═══════════════════════════════════════════════════════════════════

  if (claimId) {
    // Try different fact ID patterns
    const factPatterns = [claimId, `fact:${claimId}`];

    let factFound = false;
    for (const pattern of factPatterns) {
      const fact = state.getFact(pattern);
      if (fact) {
        console.log(`  FACT FOUND (id: ${pattern})`);
        console.log(`    Claim: "${fact.claim}"`);
        console.log(`    Domain: ${fact.domain}`);
        console.log(`    Confidence: ${fact.confidence}`);
        console.log(`    Status: ${fact.status}`);
        console.log(`    Verifier count: ${fact.verifierCount}`);
        factFound = true;
        break;
      }
    }

    if (!factFound) {
      console.log(
        "  Fact not found in state (may not have been finalized yet)",
      );
      console.log("  Checking all facts...");

      // Try to get facts from the knowledge tree
      const allFacts: Array<{ id: string; claim: string; confidence: number }> =
        [];
      // Only genesis axioms may be present
      const genesisCheck = state.getFact("test:axiom:1");
      if (genesisCheck) {
        allFacts.push({
          id: "test:axiom:1",
          claim: genesisCheck.claim,
          confidence: genesisCheck.confidence,
        });
      }
      console.log(`  Known facts: ${allFacts.length}`);
      for (const f of allFacts) {
        console.log(`    [${f.id}] "${f.claim}" (conf: ${f.confidence})`);
      }
    }
  }

  // ═══════════════════════════════════════════════════════════════════
  step(6, "ECONOMIC STATE — Balances, rewards, fees");
  // ═══════════════════════════════════════════════════════════════════

  console.log("\n  Final balances:");
  for (const v of network.validators) {
    const acct = state.getAccount(v.did);
    const val = state.getValidator(v.publicKey);
    info(`  ${v.name}`, acct?.balance ?? 0n);
    console.log(`    Stake: ${formatLGM(val?.stake ?? 0n)}`);
  }

  for (const a of network.accounts) {
    const acct = state.getAccount(a.did);
    info(`  ${a.name}`, acct?.balance ?? 0n);
  }

  // Check treasury accounts
  const treasuryDids: DID[] = [
    "did:lgm:protocol-treasury" as DID,
    "did:lgm:research" as DID,
  ];
  console.log("\n  Treasury accounts:");
  for (const did of treasuryDids) {
    const acct = state.getAccount(did);
    if (acct) {
      info(`  ${did}`, acct.balance);
    } else {
      console.log(`  ${did}: not found`);
    }
  }

  // ═══════════════════════════════════════════════════════════════════
  step(7, "MINE 20 MORE BLOCKS — Observe block reward accumulation");
  // ═══════════════════════════════════════════════════════════════════

  const beforeReward = state.getAccount(v0.did)?.balance ?? 0n;
  await network.mineBlocks(20);
  const afterReward = state.getAccount(v0.did)?.balance ?? 0n;

  info("  validator-0 before", beforeReward);
  info("  validator-0 after", afterReward);
  info("  Reward accumulated", afterReward - beforeReward);
  console.log(`  Current block: ${network.getCurrentBlock()}`);

  // ═══════════════════════════════════════════════════════════════════
  step(8, "SUBMIT SECOND CLAIM — Test rejection scenario");
  // ═══════════════════════════════════════════════════════════════════

  try {
    const claim2Id = await network.submitClaim(
      "The Earth is flat and rests on four elephants standing on a turtle",
      "science.geography",
      network.accounts[1],
      111n * ONE_LGM,
    );
    console.log(`  Claim submitted! ID: ${claim2Id}`);

    const rounds2 = consensus.getActiveRounds();
    if (rounds2.length > 0) {
      const round2 =
        rounds2.find((r) => r.id === claim2Id) || rounds2[rounds2.length - 1];
      console.log("  Validators rejecting...");
      try {
        await network.completeVerificationRoundDeterministic(
          round2.id,
          network.validators.map((v) => ({
            validator: v,
            verdict: "rejected" as const,
            confidence: 0.95,
          })),
        );
        console.log("  Verification round completed (expected: rejected)");
      } catch (error) {
        console.log(`  VERIFICATION ERROR: ${(error as Error).message}`);
      }

      await network.mineBlocks(3);
    }
  } catch (error) {
    console.log(`  CLAIM FAILED: ${(error as Error).message}`);
  }

  // ═══════════════════════════════════════════════════════════════════
  step(9, "CONSENSUS STATE SUMMARY");
  // ═══════════════════════════════════════════════════════════════════

  const consensusState = consensus.getState();
  console.log(`  Current block:         ${consensusState.currentBlock}`);
  console.log(
    `  Last block hash:       ${consensusState.lastBlockHash?.slice(0, 32)}...`,
  );
  console.log(`  Last finalized block:  ${consensusState.lastFinalizedBlock}`);
  console.log(
    `  Validators:            ${consensusState.validatorSet.validators.length}`,
  );
  console.log(`  Active rounds:         ${consensus.getActiveRounds().length}`);

  // ═══════════════════════════════════════════════════════════════════
  step(10, "PROTOCOL OBSERVATIONS & FINDINGS");
  // ═══════════════════════════════════════════════════════════════════

  const findings: string[] = [];

  // Check if block rewards are working
  if (afterReward > beforeReward) {
    findings.push(
      `[OK] Block rewards working: +${formatLGM(afterReward - beforeReward)} over 20 blocks`,
    );
  } else {
    findings.push("[ISSUE] No block rewards accumulated in 20 blocks");
  }

  // Check validator balances are reasonable
  for (const v of network.validators) {
    const acct = state.getAccount(v.did);
    if (acct && acct.balance < v.stake) {
      findings.push(
        `[WARN] ${v.name} balance (${formatLGM(acct.balance)}) < initial stake (${formatLGM(v.stake)})`,
      );
    }
  }

  // Check if research fund is receiving its 22% cut
  const researchAcct = state.getAccount("did:lgm:research" as DID);
  if (researchAcct) {
    findings.push(
      `[INFO] Research fund balance: ${formatLGM(researchAcct.balance)}`,
    );
  }

  // Check claim economics
  if (claimId) {
    const claimantFinal = state.getAccount(claimant.did);
    const claimantInitial = claimantAcctBefore?.balance ?? 0n;
    const claimantDelta = (claimantFinal?.balance ?? 0n) - claimantInitial;
    if (claimantDelta < 0n) {
      findings.push(
        `[INFO] Claimant lost ${formatLGM(-claimantDelta)} (stake locked or burned)`,
      );
    } else if (claimantDelta > 0n) {
      findings.push(
        `[INFO] Claimant gained ${formatLGM(claimantDelta)} (claim reward minus stake)`,
      );
    } else {
      findings.push("[INFO] Claimant balance unchanged");
    }
  }

  console.log("");
  for (const f of findings) {
    console.log(`  ${f}`);
  }

  // ═══════════════════════════════════════════════════════════════════
  banner("SIMULATION COMPLETE");
  // ═══════════════════════════════════════════════════════════════════

  console.log(`\n  Total blocks produced: ${network.getCurrentBlock()}`);
  console.log(
    `  Protocol is ${consensus.getActiveRounds().length > 0 ? "ACTIVE" : "IDLE"}`,
  );
  console.log("");

  await network.stop();
}

main().catch((error) => {
  console.error("\nFATAL:", error);
  process.exit(1);
});
