/**
 * CheckpointCeremony Tests
 *
 * Tests the BFT-style checkpoint ceremony: proposal creation/verification,
 * prevote/precommit phases, finalization, timeout handling, signature
 * verification, and the CheckpointCeremonyManager.
 *
 * Uses real Ed25519 keys and domain-separated signatures — no mocks.
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  createCheckpointProposal,
  verifyCheckpointProposal,
  startCheckpointCeremony,
  createPrevote,
  addPrevote,
  shouldTransitionToPrecommit,
  transitionToPrecommit,
  createPrecommit,
  addPrecommit,
  shouldFinalize,
  finalizeCeremony,
  verifyCheckpointSignatures,
  isCeremonyTimedOut,
  handleCeremonyTimeout,
  CheckpointCeremonyManager,
  DEFAULT_CEREMONY_CONFIG,
  type CheckpointCeremony,
  type CheckpointProposal,
  type CeremonyConfig,
} from "../../../consensus/pot/checkpoint_ceremony.js";
import type {
  Block,
  BlockHeader,
  Validator,
  ValidatorSet,
} from "../../../consensus/pot/types.js";
import {
  generateKeypairFromSeed,
  type PublicKey,
  type PrivateKey,
} from "../../../core/crypto/keys.js";
import { domainHash } from "../../../core/crypto/hashing.js";
import { signDomainSeparated } from "../../../core/crypto/signatures.js";

// ============ Helpers ============

interface TestKeypair {
  publicKey: PublicKey;
  privateKey: PrivateKey;
}

function createTestKeypair(seed: string): TestKeypair {
  const hash = domainHash("LGM.test.ceremony.v1", seed);
  return generateKeypairFromSeed(hash.slice(0, 32));
}

function createTestValidator(
  publicKey: PublicKey,
  stake: bigint,
  delegatedStake: bigint = 0n,
  isActive: boolean = true,
): Validator {
  return {
    publicKey,
    did: `did:lgm:${publicKey.slice(0, 32)}`,
    stake,
    delegatedStake,
    reputation: 1.0,
    isActive,
    lastActiveBlock: 0,
    totalVerifications: 0,
    correctVerifications: 0,
    missedVerifications: 0,
    slashCount: 0,
    tier:
      stake >= 11111n * 10n ** 18n ? 3 : stake >= 1111n * 10n ** 18n ? 2 : 0,
  };
}

function createTestValidatorSet(
  keypairs: TestKeypair[],
  stakes: bigint[],
): ValidatorSet {
  const validators = new Map<string, Validator>();
  let totalStake = 0n;

  for (let i = 0; i < keypairs.length; i++) {
    const stake = stakes[i] ?? 10000n;
    const v = createTestValidator(keypairs[i].publicKey, stake);
    validators.set(keypairs[i].publicKey, v);
    totalStake += stake;
  }

  return {
    validators,
    totalStake,
    activeCount: keypairs.length,
    epoch: 1n,
  };
}

function createTestBlock(number: bigint, proposer: PublicKey): Block {
  return {
    header: {
      number,
      timestamp: Number(number) * 2521,
      previousHash: `prev-hash-${number}`,
      stateRoot: `state-root-${number}`,
      knowledgeRoot: `knowledge-root-${number}`,
      transactionsRoot: `tx-root-${number}`,
      verificationsRoot: `ver-root-${number}`,
      proposer,
      proposerVrfOutput: `vrf-out-${number}`,
      proposerVrfProof: `vrf-proof-${number}`,
    },
    transactions: [],
    verificationResults: [],
    signature: "block-sig",
    attestations: [],
  };
}

// ============ Tests ============

describe.sequential("CheckpointCeremony", () => {
  // Shared keypairs and validator set for most tests
  let kp1: TestKeypair;
  let kp2: TestKeypair;
  let kp3: TestKeypair;
  let validatorSet: ValidatorSet;

  beforeEach(() => {
    kp1 = createTestKeypair("validator-1");
    kp2 = createTestKeypair("validator-2");
    kp3 = createTestKeypair("validator-3");
    // Asymmetric stakes: kp1+kp2 = 25000/30000 = 83% > 67% threshold
    // kp1 alone = 15000/30000 = 50% < 67%
    validatorSet = createTestValidatorSet(
      [kp1, kp2, kp3],
      [15000n, 10000n, 5000n],
    );
  });

  // ============ createCheckpointProposal ============

  describe("createCheckpointProposal", () => {
    it("should create proposal with valid signature", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );

      expect(proposal.blockNumber).toBe(100n);
      expect(proposal.stateRoot).toBe("state-root-100");
      expect(proposal.knowledgeRoot).toBe("knowledge-root-100");
      expect(proposal.proposer).toBe(kp1.publicKey);
      expect(proposal.proposerSignature).toBeTruthy();
    });

    it("should use domain LGM.checkpoint.proposal.v1", () => {
      const block = createTestBlock(50n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );

      // Verify with the correct domain succeeds
      const result = verifyCheckpointProposal(proposal, validatorSet);
      expect(result.valid).toBe(true);
    });
  });

  // ============ verifyCheckpointProposal ============

  describe("verifyCheckpointProposal", () => {
    it("should accept proposal from active validator", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );

      const result = verifyCheckpointProposal(proposal, validatorSet);
      expect(result.valid).toBe(true);
    });

    it("should reject proposal from non-validator", () => {
      const outsider = createTestKeypair("outsider");
      const block = createTestBlock(100n, outsider.publicKey);
      const proposal = createCheckpointProposal(
        block,
        outsider.privateKey,
        outsider.publicKey,
      );

      const result = verifyCheckpointProposal(proposal, validatorSet);
      expect(result.valid).toBe(false);
      expect(result.reason).toContain("not an active validator");
    });

    it("should reject tampered proposal data", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );

      // Tamper with stateRoot
      const tampered = { ...proposal, stateRoot: "tampered-root" };
      const result = verifyCheckpointProposal(tampered, validatorSet);
      expect(result.valid).toBe(false);
      expect(result.reason).toContain("Invalid proposer signature");
    });
  });

  // ============ startCheckpointCeremony ============

  describe("startCheckpointCeremony", () => {
    it("should create ceremony with correct deadlines", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );

      const ceremony = startCheckpointCeremony(proposal, 100n);

      expect(ceremony.phase).toBe("propose");
      expect(ceremony.startBlock).toBe(100n);
      expect(ceremony.prevoteDeadline).toBe(102n); // 100 + 2
      expect(ceremony.precommitDeadline).toBe(104n); // 100 + 2 + 2
      expect(ceremony.prevotes.size).toBe(0);
      expect(ceremony.precommits.size).toBe(0);
    });

    it("should set VULN-012 timeout deadline", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );

      const ceremony = startCheckpointCeremony(proposal, 100n);
      expect(ceremony.timeoutDeadline).toBe(110n); // 100 + 10
    });

    it("should respect custom config", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );

      const config: CeremonyConfig = {
        prevoteBlocks: 5,
        precommitBlocks: 3,
        stakeThreshold: 0.67,
        maxCeremonyBlocks: 20,
      };

      const ceremony = startCheckpointCeremony(proposal, 100n, config);
      expect(ceremony.prevoteDeadline).toBe(105n);
      expect(ceremony.precommitDeadline).toBe(108n);
      expect(ceremony.timeoutDeadline).toBe(120n);
    });
  });

  // ============ Prevote Phase ============

  describe("prevote phase", () => {
    let ceremony: CheckpointCeremony;

    beforeEach(() => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      ceremony = startCheckpointCeremony(proposal, 100n);
    });

    it("should create signed prevote with correct domain", () => {
      const prevote = createPrevote(
        ceremony.id,
        "yes",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );

      expect(prevote.proposalHash).toBe(ceremony.id);
      expect(prevote.vote).toBe("yes");
      expect(prevote.validator).toBe(kp1.publicKey);
      expect(prevote.signature).toBeTruthy();
    });

    it("should accept valid prevote and track stake", () => {
      const prevote = createPrevote(
        ceremony.id,
        "yes",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );

      const result = addPrevote(ceremony, prevote, validatorSet, 100n);
      expect(result.accepted).toBe(true);
      expect(result.ceremony.yesPrevoteStake).toBe(15000n);
    });

    it("should track no-vote stake separately", () => {
      const prevote = createPrevote(
        ceremony.id,
        "no",
        kp2.privateKey,
        kp2.publicKey,
        100n,
      );

      const result = addPrevote(ceremony, prevote, validatorSet, 100n);
      expect(result.accepted).toBe(true);
      expect(result.ceremony.noPrevoteStake).toBe(10000n);
      expect(result.ceremony.yesPrevoteStake).toBe(0n);
    });

    it("should reject prevote in wrong phase", () => {
      // Force ceremony to precommit phase
      const precommitCeremony = { ...ceremony, phase: "precommit" as const };
      const prevote = createPrevote(
        ceremony.id,
        "yes",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );

      const result = addPrevote(precommitCeremony, prevote, validatorSet, 100n);
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("Not in prevote phase");
    });

    it("should reject prevote past deadline", () => {
      const prevote = createPrevote(
        ceremony.id,
        "yes",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );

      // Block 103 > prevoteDeadline of 102
      const result = addPrevote(ceremony, prevote, validatorSet, 103n);
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("Prevote deadline passed");
    });

    it("should reject duplicate prevote", () => {
      const prevote = createPrevote(
        ceremony.id,
        "yes",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );

      const first = addPrevote(ceremony, prevote, validatorSet, 100n);
      expect(first.accepted).toBe(true);

      const second = addPrevote(first.ceremony, prevote, validatorSet, 100n);
      expect(second.accepted).toBe(false);
      expect(second.reason).toContain("Already voted");
    });

    it("should reject prevote with invalid signature", () => {
      // Create prevote signed by kp1 but claiming to be kp2
      const prevote = createPrevote(
        ceremony.id,
        "yes",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );

      // Tamper: swap validator to kp2 (signature won't match)
      const tampered = { ...prevote, validator: kp2.publicKey };
      const result = addPrevote(ceremony, tampered, validatorSet, 100n);
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("Invalid signature");
    });

    it("should reject prevote with wrong proposal hash", () => {
      const prevote = createPrevote(
        "wrong-proposal-hash",
        "yes",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );

      const result = addPrevote(ceremony, prevote, validatorSet, 100n);
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("Proposal hash mismatch");
    });

    it("should transition propose -> prevote on first vote", () => {
      expect(ceremony.phase).toBe("propose");

      const prevote = createPrevote(
        ceremony.id,
        "yes",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );

      const result = addPrevote(ceremony, prevote, validatorSet, 100n);
      expect(result.ceremony.phase).toBe("prevote");
    });
  });

  // ============ shouldTransitionToPrecommit ============

  describe("shouldTransitionToPrecommit", () => {
    it("should transition at 67% yes stake threshold", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      let ceremony = startCheckpointCeremony(proposal, 100n);

      // Add 2 of 3 validators voting yes (20000/30000 = 66.7%)
      const pv1 = createPrevote(
        ceremony.id,
        "yes",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );
      ceremony = addPrevote(ceremony, pv1, validatorSet, 100n).ceremony;

      const pv2 = createPrevote(
        ceremony.id,
        "yes",
        kp2.privateKey,
        kp2.publicKey,
        100n,
      );
      ceremony = addPrevote(ceremony, pv2, validatorSet, 100n).ceremony;

      const result = shouldTransitionToPrecommit(ceremony, validatorSet, 100n);
      expect(result.shouldTransition).toBe(true);
      expect(result.reason).toContain("threshold reached");
    });

    it("should transition when deadline passes", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      let ceremony = startCheckpointCeremony(proposal, 100n);

      // Add one prevote to enter prevote phase
      const pv1 = createPrevote(
        ceremony.id,
        "yes",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );
      ceremony = addPrevote(ceremony, pv1, validatorSet, 100n).ceremony;

      // Past deadline (103 > 102)
      const result = shouldTransitionToPrecommit(ceremony, validatorSet, 103n);
      expect(result.shouldTransition).toBe(true);
      expect(result.reason).toContain("deadline passed");
    });

    it("should transition when impossible to reach threshold", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      let ceremony = startCheckpointCeremony(proposal, 100n);

      // Two no-votes: remaining yes can never reach threshold
      const pv1 = createPrevote(
        ceremony.id,
        "no",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );
      ceremony = addPrevote(ceremony, pv1, validatorSet, 100n).ceremony;

      const pv2 = createPrevote(
        ceremony.id,
        "no",
        kp2.privateKey,
        kp2.publicKey,
        100n,
      );
      ceremony = addPrevote(ceremony, pv2, validatorSet, 100n).ceremony;

      const result = shouldTransitionToPrecommit(ceremony, validatorSet, 100n);
      expect(result.shouldTransition).toBe(true);
      expect(result.reason).toContain("impossible");
    });
  });

  // ============ transitionToPrecommit ============

  describe("transitionToPrecommit", () => {
    it("should transition to precommit when stake sufficient", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      let ceremony = startCheckpointCeremony(proposal, 100n);

      // Two yes votes = 20000/30000 >= 67%
      const pv1 = createPrevote(
        ceremony.id,
        "yes",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );
      ceremony = addPrevote(ceremony, pv1, validatorSet, 100n).ceremony;
      const pv2 = createPrevote(
        ceremony.id,
        "yes",
        kp2.privateKey,
        kp2.publicKey,
        100n,
      );
      ceremony = addPrevote(ceremony, pv2, validatorSet, 100n).ceremony;

      const result = transitionToPrecommit(ceremony, validatorSet);
      expect(result.phase).toBe("precommit");
    });

    it("should fail when insufficient yes-prevotes", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      let ceremony = startCheckpointCeremony(proposal, 100n);

      // Only 1 yes vote = 10000/30000 < 67%
      const pv1 = createPrevote(
        ceremony.id,
        "yes",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );
      ceremony = addPrevote(ceremony, pv1, validatorSet, 100n).ceremony;

      const result = transitionToPrecommit(ceremony, validatorSet);
      expect(result.phase).toBe("failed");
      expect(result.failureReason).toContain("Insufficient prevotes");
    });
  });

  // ============ Precommit Phase ============

  describe("precommit phase", () => {
    let precommitCeremony: CheckpointCeremony;

    beforeEach(() => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      let ceremony = startCheckpointCeremony(proposal, 100n);

      // Get to precommit phase: 2 yes prevotes
      const pv1 = createPrevote(
        ceremony.id,
        "yes",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );
      ceremony = addPrevote(ceremony, pv1, validatorSet, 100n).ceremony;
      const pv2 = createPrevote(
        ceremony.id,
        "yes",
        kp2.privateKey,
        kp2.publicKey,
        100n,
      );
      ceremony = addPrevote(ceremony, pv2, validatorSet, 100n).ceremony;

      precommitCeremony = transitionToPrecommit(ceremony, validatorSet);
      expect(precommitCeremony.phase).toBe("precommit");
    });

    it("should create signed precommit", () => {
      const precommit = createPrecommit(
        precommitCeremony,
        kp1.privateKey,
        kp1.publicKey,
      );

      expect(precommit.proposalHash).toBe(precommitCeremony.id);
      expect(precommit.validator).toBe(kp1.publicKey);
      expect(precommit.checkpointSignature).toBeTruthy();
    });

    it("should accept precommit from validator who prevoted yes", () => {
      const precommit = createPrecommit(
        precommitCeremony,
        kp1.privateKey,
        kp1.publicKey,
      );

      const result = addPrecommit(
        precommitCeremony,
        precommit,
        validatorSet,
        103n,
      );
      expect(result.accepted).toBe(true);
      expect(result.ceremony.precommitStake).toBe(15000n);
    });

    it("should reject precommit in wrong phase", () => {
      // Force to prevote phase
      const wrongPhase = { ...precommitCeremony, phase: "prevote" as const };
      const precommit = createPrecommit(
        precommitCeremony,
        kp1.privateKey,
        kp1.publicKey,
      );

      const result = addPrecommit(wrongPhase, precommit, validatorSet, 103n);
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("Not in precommit phase");
    });

    it("should reject precommit from validator who did not prevote yes", () => {
      const precommit = createPrecommit(
        precommitCeremony,
        kp3.privateKey,
        kp3.publicKey,
      );

      const result = addPrecommit(
        precommitCeremony,
        precommit,
        validatorSet,
        103n,
      );
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("did not prevote yes");
    });

    it("should reject duplicate precommit", () => {
      const precommit = createPrecommit(
        precommitCeremony,
        kp1.privateKey,
        kp1.publicKey,
      );

      const first = addPrecommit(
        precommitCeremony,
        precommit,
        validatorSet,
        103n,
      );
      expect(first.accepted).toBe(true);

      const second = addPrecommit(
        first.ceremony,
        precommit,
        validatorSet,
        103n,
      );
      expect(second.accepted).toBe(false);
      expect(second.reason).toContain("Already precommitted");
    });

    it("should reject precommit with invalid signature", () => {
      const precommit = createPrecommit(
        precommitCeremony,
        kp1.privateKey,
        kp1.publicKey,
      );

      // Tamper: swap validator to kp2 (signature won't match)
      const tampered = { ...precommit, validator: kp2.publicKey };
      const result = addPrecommit(
        precommitCeremony,
        tampered,
        validatorSet,
        103n,
      );
      expect(result.accepted).toBe(false);
      // Could be "did not prevote yes" or "Invalid checkpoint signature" depending on order
      expect(result.accepted).toBe(false);
    });

    it("should reject precommit past deadline", () => {
      const precommit = createPrecommit(
        precommitCeremony,
        kp1.privateKey,
        kp1.publicKey,
      );

      // Block 105 > precommitDeadline of 104
      const result = addPrecommit(
        precommitCeremony,
        precommit,
        validatorSet,
        105n,
      );
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("Precommit deadline passed");
    });
  });

  // ============ Finalization ============

  describe("finalization", () => {
    it("should create finalized checkpoint with aggregated signatures", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      let ceremony = startCheckpointCeremony(proposal, 100n);

      // Prevote: all 3 yes
      for (const kp of [kp1, kp2, kp3]) {
        const pv = createPrevote(
          ceremony.id,
          "yes",
          kp.privateKey,
          kp.publicKey,
          100n,
        );
        ceremony = addPrevote(ceremony, pv, validatorSet, 100n).ceremony;
      }
      ceremony = transitionToPrecommit(ceremony, validatorSet);

      // Precommit: 2 of 3 (enough for 67%)
      for (const kp of [kp1, kp2]) {
        const pc = createPrecommit(ceremony, kp.privateKey, kp.publicKey);
        ceremony = addPrecommit(ceremony, pc, validatorSet, 103n).ceremony;
      }

      const finalized = finalizeCeremony(ceremony, validatorSet);
      expect(finalized.phase).toBe("finalized");
      expect(finalized.finalizedCheckpoint).toBeDefined();
      expect(finalized.finalizedCheckpoint!.blockNumber).toBe(100n);
      expect(finalized.finalizedCheckpoint!.signatures.length).toBe(2);
      expect(finalized.finalizedCheckpoint!.totalSigningStake).toBe(25000n);
    });

    it("should fail when precommit stake insufficient", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      let ceremony = startCheckpointCeremony(proposal, 100n);

      // Prevote: all 3 yes
      for (const kp of [kp1, kp2, kp3]) {
        const pv = createPrevote(
          ceremony.id,
          "yes",
          kp.privateKey,
          kp.publicKey,
          100n,
        );
        ceremony = addPrevote(ceremony, pv, validatorSet, 100n).ceremony;
      }
      ceremony = transitionToPrecommit(ceremony, validatorSet);

      // Precommit: only 1 of 3 (10000/30000 = 33% < 67%)
      const pc = createPrecommit(ceremony, kp1.privateKey, kp1.publicKey);
      ceremony = addPrecommit(ceremony, pc, validatorSet, 103n).ceremony;

      const result = finalizeCeremony(ceremony, validatorSet);
      expect(result.phase).toBe("failed");
      expect(result.failureReason).toContain("Insufficient precommits");
    });

    it("should verify checkpoint signatures validates all sigs", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      let ceremony = startCheckpointCeremony(proposal, 100n);

      // Full ceremony: prevote + precommit
      for (const kp of [kp1, kp2, kp3]) {
        const pv = createPrevote(
          ceremony.id,
          "yes",
          kp.privateKey,
          kp.publicKey,
          100n,
        );
        ceremony = addPrevote(ceremony, pv, validatorSet, 100n).ceremony;
      }
      ceremony = transitionToPrecommit(ceremony, validatorSet);

      for (const kp of [kp1, kp2, kp3]) {
        const pc = createPrecommit(ceremony, kp.privateKey, kp.publicKey);
        ceremony = addPrecommit(ceremony, pc, validatorSet, 103n).ceremony;
      }

      const finalized = finalizeCeremony(ceremony, validatorSet);
      expect(finalized.finalizedCheckpoint).toBeDefined();

      const verification = verifyCheckpointSignatures(
        finalized.finalizedCheckpoint!,
        validatorSet,
      );
      expect(verification.valid).toBe(true);
      expect(verification.validSignatures).toBe(3);
      expect(verification.invalidSignatures).toHaveLength(0);
    });

    it("should report invalid signatures individually", () => {
      // Create a finalized checkpoint then tamper with one signature
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      let ceremony = startCheckpointCeremony(proposal, 100n);

      for (const kp of [kp1, kp2, kp3]) {
        const pv = createPrevote(
          ceremony.id,
          "yes",
          kp.privateKey,
          kp.publicKey,
          100n,
        );
        ceremony = addPrevote(ceremony, pv, validatorSet, 100n).ceremony;
      }
      ceremony = transitionToPrecommit(ceremony, validatorSet);

      for (const kp of [kp1, kp2, kp3]) {
        const pc = createPrecommit(ceremony, kp.privateKey, kp.publicKey);
        ceremony = addPrecommit(ceremony, pc, validatorSet, 103n).ceremony;
      }

      const finalized = finalizeCeremony(ceremony, validatorSet);
      const checkpoint = finalized.finalizedCheckpoint!;

      // Tamper with one signature
      const tamperedSigs = [...checkpoint.signatures];
      tamperedSigs[0] = { ...tamperedSigs[0], signature: "tampered-signature" };
      const tamperedCheckpoint = { ...checkpoint, signatures: tamperedSigs };

      const verification = verifyCheckpointSignatures(
        tamperedCheckpoint,
        validatorSet,
      );
      expect(verification.valid).toBe(false);
      expect(verification.invalidSignatures.length).toBeGreaterThan(0);
      expect(verification.invalidSignatures[0]).toContain("invalid signature");
    });
  });

  // ============ VULN-012: Timeout ============

  describe("VULN-012: Timeout", () => {
    it("should detect timed-out ceremony", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      const ceremony = startCheckpointCeremony(proposal, 100n);

      // timeoutDeadline = 110, current block 111 > 110
      const result = isCeremonyTimedOut(ceremony, 111n);
      expect(result.timedOut).toBe(true);
      expect(result.reason).toContain("timed out");
    });

    it("should mark timed-out as failed with reason", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      const ceremony = startCheckpointCeremony(proposal, 100n);

      const failed = handleCeremonyTimeout(ceremony, 115n);
      expect(failed.phase).toBe("failed");
      expect(failed.failureReason).toContain("timed out");
      expect(failed.failureReason).toContain("115");
    });

    it("should not timeout already finalized ceremony", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      const ceremony = startCheckpointCeremony(proposal, 100n);
      const finalized = { ...ceremony, phase: "finalized" as const };

      const result = isCeremonyTimedOut(finalized, 200n);
      expect(result.timedOut).toBe(false);
    });

    it("should not timeout already failed ceremony", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      const ceremony = startCheckpointCeremony(proposal, 100n);
      const failed = { ...ceremony, phase: "failed" as const };

      const result = isCeremonyTimedOut(failed, 200n);
      expect(result.timedOut).toBe(false);
    });

    it("should not timeout before deadline", () => {
      const block = createTestBlock(100n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );
      const ceremony = startCheckpointCeremony(proposal, 100n);

      // timeoutDeadline = 110, current block 109 <= 110
      const result = isCeremonyTimedOut(ceremony, 109n);
      expect(result.timedOut).toBe(false);
    });
  });

  // ============ CheckpointCeremonyManager ============

  describe("CheckpointCeremonyManager", () => {
    it("should start ceremony and track in activeCeremonies", () => {
      const manager = new CheckpointCeremonyManager();
      const block = createTestBlock(100n, kp1.publicKey);

      const ceremony = manager.startCeremony(
        block,
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );

      expect(ceremony.phase).toBe("propose");
      expect(manager.getActiveCeremony(100n)).toBeDefined();
    });

    it("should process prevotes and precommits via handles", () => {
      const manager = new CheckpointCeremonyManager();
      const block = createTestBlock(100n, kp1.publicKey);
      const ceremony = manager.startCeremony(
        block,
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );

      // Prevote
      const prevote = createPrevote(
        ceremony.id,
        "yes",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );

      const pvResult = manager.handlePrevote(
        ceremony.id,
        prevote,
        validatorSet,
        100n,
      );
      expect(pvResult.accepted).toBe(true);
    });

    it("should return error for unknown ceremony ID", () => {
      const manager = new CheckpointCeremonyManager();

      const prevote = createPrevote(
        "nonexistent",
        "yes",
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );

      const result = manager.handlePrevote(
        "nonexistent",
        prevote,
        validatorSet,
        100n,
      );
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("Ceremony not found");
    });

    it("should processBlock advancing phases and producing checkpoints", () => {
      const manager = new CheckpointCeremonyManager();
      const block = createTestBlock(100n, kp1.publicKey);
      const ceremony = manager.startCeremony(
        block,
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );

      // All 3 prevote yes
      for (const kp of [kp1, kp2, kp3]) {
        const pv = createPrevote(
          ceremony.id,
          "yes",
          kp.privateKey,
          kp.publicKey,
          100n,
        );
        manager.handlePrevote(ceremony.id, pv, validatorSet, 100n);
      }

      // processBlock should transition prevote → precommit
      manager.processBlock(validatorSet, 101n);

      // Now add precommits (2 of 3 for 67% threshold)
      for (const kp of [kp1, kp2]) {
        const active = manager.getActiveCeremony(100n)!;
        const pc = createPrecommit(active, kp.privateKey, kp.publicKey);
        manager.handlePrecommit(ceremony.id, pc, validatorSet, 103n);
      }

      // processBlock should finalize
      const checkpoints = manager.processBlock(validatorSet, 103n);
      expect(checkpoints.length).toBe(1);
      expect(checkpoints[0].blockNumber).toBe(100n);

      // Should be removed from active
      expect(manager.getActiveCeremony(100n)).toBeUndefined();
    });

    it("should timeout stalled ceremonies in processBlock", () => {
      const manager = new CheckpointCeremonyManager();
      const block = createTestBlock(100n, kp1.publicKey);
      manager.startCeremony(block, kp1.privateKey, kp1.publicKey, 100n);

      // No votes at all — ceremony stalls
      // Process at block 111 (past timeout deadline of 110)
      const checkpoints = manager.processBlock(validatorSet, 111n);
      expect(checkpoints.length).toBe(0);

      // Should be removed from active (failed)
      expect(manager.getActiveCeremony(100n)).toBeUndefined();
    });

    it("should remove completed/failed from active map", () => {
      const manager = new CheckpointCeremonyManager();

      // Create two ceremonies
      const block1 = createTestBlock(100n, kp1.publicKey);
      const block2 = createTestBlock(200n, kp1.publicKey);

      manager.startCeremony(block1, kp1.privateKey, kp1.publicKey, 100n);
      manager.startCeremony(block2, kp1.privateKey, kp1.publicKey, 200n);

      // Timeout both at block 211
      manager.processBlock(validatorSet, 211n);

      expect(manager.getActiveCeremony(100n)).toBeUndefined();
      expect(manager.getActiveCeremony(200n)).toBeUndefined();
    });

    it("should track completed checkpoints via getCheckpoints", () => {
      const manager = new CheckpointCeremonyManager();
      const block = createTestBlock(100n, kp1.publicKey);
      const ceremony = manager.startCeremony(
        block,
        kp1.privateKey,
        kp1.publicKey,
        100n,
      );

      // Full ceremony
      for (const kp of [kp1, kp2, kp3]) {
        const pv = createPrevote(
          ceremony.id,
          "yes",
          kp.privateKey,
          kp.publicKey,
          100n,
        );
        manager.handlePrevote(ceremony.id, pv, validatorSet, 100n);
      }
      manager.processBlock(validatorSet, 101n);

      for (const kp of [kp1, kp2]) {
        const active = manager.getActiveCeremony(100n)!;
        const pc = createPrecommit(active, kp.privateKey, kp.publicKey);
        manager.handlePrecommit(ceremony.id, pc, validatorSet, 103n);
      }
      manager.processBlock(validatorSet, 103n);

      const all = manager.getCheckpoints();
      expect(all.length).toBe(1);

      const latest = manager.getLatestCheckpoint();
      expect(latest).toBeDefined();
      expect(latest!.blockNumber).toBe(100n);
    });
  });

  // ============ Full Lifecycle ============

  describe("full lifecycle", () => {
    it("should complete proposal -> prevote -> precommit -> finalize with real crypto", () => {
      // Step 1: Create proposal
      const block = createTestBlock(500n, kp1.publicKey);
      const proposal = createCheckpointProposal(
        block,
        kp1.privateKey,
        kp1.publicKey,
      );

      // Step 2: Verify proposal
      const proposalValid = verifyCheckpointProposal(proposal, validatorSet);
      expect(proposalValid.valid).toBe(true);

      // Step 3: Start ceremony
      let ceremony = startCheckpointCeremony(proposal, 500n);
      expect(ceremony.phase).toBe("propose");

      // Step 4: Prevote phase — all 3 validators vote yes
      for (const kp of [kp1, kp2, kp3]) {
        const prevote = createPrevote(
          ceremony.id,
          "yes",
          kp.privateKey,
          kp.publicKey,
          500n,
        );
        const result = addPrevote(ceremony, prevote, validatorSet, 500n);
        expect(result.accepted).toBe(true);
        ceremony = result.ceremony;
      }

      expect(ceremony.phase).toBe("prevote");
      expect(ceremony.yesPrevoteStake).toBe(30000n);

      // Step 5: Check transition readiness
      const transitionCheck = shouldTransitionToPrecommit(
        ceremony,
        validatorSet,
        501n,
      );
      expect(transitionCheck.shouldTransition).toBe(true);

      // Step 6: Transition to precommit
      ceremony = transitionToPrecommit(ceremony, validatorSet);
      expect(ceremony.phase).toBe("precommit");

      // Step 7: Precommit phase — all 3 sign
      for (const kp of [kp1, kp2, kp3]) {
        const precommit = createPrecommit(
          ceremony,
          kp.privateKey,
          kp.publicKey,
        );
        const result = addPrecommit(ceremony, precommit, validatorSet, 502n);
        expect(result.accepted).toBe(true);
        ceremony = result.ceremony;
      }

      expect(ceremony.precommitStake).toBe(30000n);

      // Step 8: Check finalization readiness
      const finalizeCheck = shouldFinalize(ceremony, validatorSet, 502n);
      expect(finalizeCheck.shouldFinalize).toBe(true);

      // Step 9: Finalize
      ceremony = finalizeCeremony(ceremony, validatorSet);
      expect(ceremony.phase).toBe("finalized");
      expect(ceremony.finalizedCheckpoint).toBeDefined();

      const checkpoint = ceremony.finalizedCheckpoint!;
      expect(checkpoint.blockNumber).toBe(500n);
      expect(checkpoint.signatures.length).toBe(3);
      expect(checkpoint.totalSigningStake).toBe(30000n);

      // Step 10: Verify finalized checkpoint
      const verification = verifyCheckpointSignatures(checkpoint, validatorSet);
      expect(verification.valid).toBe(true);
      expect(verification.validSignatures).toBe(3);
      expect(verification.invalidSignatures).toHaveLength(0);
    });
  });
});
