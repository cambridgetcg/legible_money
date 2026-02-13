/**
 * DIMENSION 2: Consensus & Validator Security Tests
 * File 1: Byzantine Fault Injection & Equivocation
 *
 * Tests the protocol's resilience against:
 *   - Block and commitment equivocation
 *   - Commit-reveal phase manipulation
 *   - Quorum and threshold boundary conditions
 */

import { describe, it, expect, beforeEach } from "vitest";
import { generateKeypair } from "../../../core/crypto/keys.js";
import { sign } from "../../../core/crypto/signatures.js";
import { sha512_256Hex } from "../../../core/crypto/hashing.js";
import {
  detectBlockEquivocation,
  detectCommitmentEquivocation,
  EquivocationDetector,
  calculateEquivocationSlash,
  verifyEquivocationEvidence,
  type EquivocationEvidence,
  type BlockEquivocationEvidence,
  type CommitmentEquivocationEvidence,
} from "../../../consensus/pot/equivocation.js";
import {
  createVerificationRound,
  startRound,
  addCommitmentWithVRF,
  addReveal,
  aggregateResults,
  generateCommitment,
  generateReveal,
  transitionToReveal,
  type ValidatorInfoMap,
} from "../../../consensus/pot/verification_round.js";
import {
  createValidatorSet,
  slashValidatorByAmount,
  calculateVotingPower,
  getEligibleValidators,
} from "../../../consensus/pot/validator_selection.js";
import {
  DEFAULT_CONSENSUS_PARAMS,
  type Block,
  type BlockHeader,
  type ValidatorCommitment,
  type ConsensusParams,
  type Validator,
  type ValidatorSet,
  type VerificationRound,
} from "../../../consensus/pot/types.js";
import type { ClaimSubmission } from "../../../core/knowledge/types.js";

// ============ Test Helpers ============

const LGM = 10n ** 18n;

function makeKeypair() {
  return generateKeypair();
}

function hashBlockHeader(header: BlockHeader): string {
  return sha512_256Hex(
    JSON.stringify({
      number: header.number.toString(),
      timestamp: header.timestamp,
      previousHash: header.previousHash,
      stateRoot: header.stateRoot,
      knowledgeRoot: header.knowledgeRoot,
      transactionsRoot: header.transactionsRoot,
      verificationsRoot: header.verificationsRoot,
      proposer: header.proposer,
    }),
  );
}

function createSignedBlock(
  privateKey: string,
  publicKey: string,
  height: bigint,
  opts: Partial<BlockHeader> = {},
): Block {
  const header: BlockHeader = {
    number: height,
    timestamp: Date.now(),
    previousHash: opts.previousHash ?? "prev-hash-" + height.toString(),
    stateRoot: opts.stateRoot ?? "state-root-" + height.toString(),
    knowledgeRoot: opts.knowledgeRoot ?? "knowledge-root-" + height.toString(),
    transactionsRoot: opts.transactionsRoot ?? "tx-root-" + height.toString(),
    verificationsRoot:
      opts.verificationsRoot ?? "ver-root-" + height.toString(),
    proposer: publicKey,
    proposerVrfOutput: "vrf-output",
    proposerVrfProof: "vrf-proof",
    ...opts,
  };

  const headerHash = hashBlockHeader(header);
  const signature = sign(new TextEncoder().encode(headerHash), privateKey);

  return {
    header,
    transactions: [],
    verificationResults: [],
    signature,
    attestations: [],
  };
}

function createTestClaim(): ClaimSubmission {
  return {
    claim: "Water boils at 100C at standard pressure",
    type: "empirical",
    domain: "science.chemistry",
    references: [
      {
        type: "supports",
        source: "textbook-123",
        contentHash: "abc",
        retrievedAt: Date.now(),
        reliabilityScore: 0.9,
      },
    ],
    proposedConfidence: 0.85,
    stake: 100n * 10n ** 15n,
  } as unknown as ClaimSubmission;
}

function createMockValidatorSet(
  validators: Array<{
    publicKey: string;
    did: string;
    stake: bigint;
    delegatedStake?: bigint;
    reputation?: number;
    isActive?: boolean;
  }>,
): ValidatorSet {
  return createValidatorSet(
    validators.map((v) => ({
      publicKey: v.publicKey,
      did: v.did,
      stake: v.stake,
    })),
    0n,
  );
}

function buildValidatorInfoMap(
  validators: Array<{
    publicKey: string;
    stake: bigint;
    reputation?: number;
  }>,
): ValidatorInfoMap {
  const map = new Map<string, { stake: bigint; reputation: number }>();
  for (const v of validators) {
    map.set(v.publicKey, {
      stake: v.stake,
      reputation: v.reputation ?? 0.5,
    });
  }
  return map;
}

// ============ 1. Equivocation Detection (8 tests) ============

describe("Equivocation Detection", () => {
  let kp1: ReturnType<typeof makeKeypair>;
  let kp2: ReturnType<typeof makeKeypair>;

  beforeEach(() => {
    kp1 = makeKeypair();
    kp2 = makeKeypair();
  });

  it("two blocks at same height from same proposer with valid signatures: MUST detect equivocation", () => {
    const block1 = createSignedBlock(kp1.privateKey, kp1.publicKey, 100n, {
      stateRoot: "state-A",
    });
    const block2 = createSignedBlock(kp1.privateKey, kp1.publicKey, 100n, {
      stateRoot: "state-B",
    });

    const result = detectBlockEquivocation(block1, block2);
    expect(result.detected).toBe(true);
    expect(result.evidence).toBeDefined();
    expect(result.evidence!.type).toBe("block_proposal");
    expect(result.evidence!.validator).toBe(kp1.publicKey);
  });

  it("two blocks at same height from DIFFERENT proposers: MUST NOT flag equivocation", () => {
    const block1 = createSignedBlock(kp1.privateKey, kp1.publicKey, 100n);
    const block2 = createSignedBlock(kp2.privateKey, kp2.publicKey, 100n);

    const result = detectBlockEquivocation(block1, block2);
    expect(result.detected).toBe(false);
    expect(result.reason).toBe("Different proposers");
  });

  it("two blocks at different heights from same proposer: MUST NOT flag equivocation", () => {
    const block1 = createSignedBlock(kp1.privateKey, kp1.publicKey, 100n);
    const block2 = createSignedBlock(kp1.privateKey, kp1.publicKey, 101n);

    const result = detectBlockEquivocation(block1, block2);
    expect(result.detected).toBe(false);
    expect(result.reason).toBe("Different heights");
  });

  it("commitment equivocation: same validator, same round, different commitments: MUST detect", () => {
    const roundId = "round-123";
    const commitment1: ValidatorCommitment = {
      validator: kp1.publicKey,
      commitment: "commitment-hash-A",
      submittedAtHeight: 10,
      signature: "sig-A-placeholder",
    };
    const commitment2: ValidatorCommitment = {
      validator: kp1.publicKey,
      commitment: "commitment-hash-B",
      submittedAtHeight: 10,
      signature: "sig-B-placeholder",
    };

    const result = detectCommitmentEquivocation(
      roundId,
      commitment1,
      commitment2,
    );
    expect(result.detected).toBe(true);
    expect(result.evidence).toBeDefined();
    expect(result.evidence!.type).toBe("verification_commitment");
    expect(result.evidence!.validator).toBe(kp1.publicKey);
  });

  it("commitment from different validators in same round: MUST NOT flag", () => {
    const roundId = "round-123";
    const commitment1: ValidatorCommitment = {
      validator: kp1.publicKey,
      commitment: "commitment-hash-A",
      submittedAtHeight: 10,
      signature: "sig-A",
    };
    const commitment2: ValidatorCommitment = {
      validator: kp2.publicKey,
      commitment: "commitment-hash-B",
      submittedAtHeight: 10,
      signature: "sig-B",
    };

    const result = detectCommitmentEquivocation(
      roundId,
      commitment1,
      commitment2,
    );
    expect(result.detected).toBe(false);
    expect(result.reason).toBe("Different validators");
  });

  it("calculateEquivocationSlash returns 100% of stake (full forfeiture)", () => {
    const stake = 1000n * LGM;
    const slash = calculateEquivocationSlash(stake);
    expect(slash).toBe(stake);
  });

  it("evidence verification: valid equivocation evidence passes verification", () => {
    // Create two blocks with valid signatures from same proposer at same height
    const block1 = createSignedBlock(kp1.privateKey, kp1.publicKey, 100n, {
      stateRoot: "state-A",
    });
    const block2 = createSignedBlock(kp1.privateKey, kp1.publicKey, 100n, {
      stateRoot: "state-B",
    });

    const result = detectBlockEquivocation(block1, block2);
    expect(result.detected).toBe(true);

    const verified = verifyEquivocationEvidence(result.evidence!);
    expect(verified).toBe(true);
  });

  it("evidence verification: fabricated evidence (invalid signatures) fails", () => {
    const block1Header: BlockHeader = {
      number: 100n,
      timestamp: Date.now(),
      previousHash: "prev",
      stateRoot: "state-A",
      knowledgeRoot: "kr",
      transactionsRoot: "tr",
      verificationsRoot: "vr",
      proposer: kp1.publicKey,
      proposerVrfOutput: "vrf",
      proposerVrfProof: "proof",
    };
    const block2Header: BlockHeader = {
      ...block1Header,
      stateRoot: "state-B",
    };

    const fabricatedEvidence: EquivocationEvidence = {
      type: "block_proposal",
      validator: kp1.publicKey,
      timestamp: Date.now(),
      evidence: {
        type: "block_proposal",
        blockHeight: 100n,
        block1Hash: hashBlockHeader(block1Header),
        block1Signature: "invalid-signature-aabbccdd",
        block2Hash: hashBlockHeader(block2Header),
        block2Signature: "invalid-signature-eeff0011",
        block1Header,
        block2Header,
      } as BlockEquivocationEvidence,
    };

    const verified = verifyEquivocationEvidence(fabricatedEvidence);
    expect(verified).toBe(false);
  });
});

// ============ 2. Commit-Reveal Security (8 tests) ============

describe("Commit-Reveal Security", () => {
  const params: ConsensusParams = {
    ...DEFAULT_CONSENSUS_PARAMS,
    commitPhaseBlocks: 10,
    revealPhaseBlocks: 10,
    verificationRoundBlocks: 25,
    minValidators: 1,
    maxValidators: 22,
    validatorQuorum: 0.66,
    acceptanceThreshold: 0.77,
    rejectionThreshold: 0.77,
  };

  let validators: Array<ReturnType<typeof makeKeypair> & { stake: bigint }>;
  let validatorSet: ValidatorSet;
  let round: VerificationRound;

  beforeEach(() => {
    validators = Array.from({ length: 5 }, () => ({
      ...makeKeypair(),
      stake: 100n * LGM,
    }));

    validatorSet = createValidatorSet(
      validators.map((v, i) => ({
        publicKey: v.publicKey,
        did: `did:lgm:validator-${i}`,
        stake: v.stake,
      })),
    );

    const claim = createTestClaim();
    const selectedValidators = validators.map((v, i) => ({
      validator: v.publicKey,
      vrfOutput: "",
      vrfProof: "",
      stake: v.stake,
      priority: BigInt(i),
      selected: true,
    }));

    round = startRound(
      createVerificationRound(claim, selectedValidators, 5, params),
    );
  });

  it("commitment without VRF proof MUST be rejected", () => {
    const v = validators[0];
    const { commitment } = generateCommitment(
      v.privateKey,
      v.publicKey,
      round.id,
      "confirmed",
      0.9,
      ["ref-1"],
    );

    // No vrfOutput/vrfProof on the commitment
    const result = addCommitmentWithVRF(
      round,
      commitment,
      6, // current block within commit phase
      "prev-block-hash",
      v.stake,
      validatorSet.totalStake,
      params,
    );

    expect(result.accepted).toBe(false);
    expect(result.reason).toContain("VRF");
  });

  it("commitment after deadline MUST be rejected", () => {
    const v = validators[0];
    const { commitment } = generateCommitment(
      v.privateKey,
      v.publicKey,
      round.id,
      "confirmed",
      0.9,
      ["ref-1"],
    );
    commitment.vrfOutput = "fake-vrf-output";
    commitment.vrfProof = "fake-vrf-proof";

    // Submit after commit deadline (round starts at block 5, commit phase = 10 blocks, deadline = 15)
    const result = addCommitmentWithVRF(
      round,
      commitment,
      round.commitDeadline + 1, // past deadline
      "prev-block-hash",
      v.stake,
      validatorSet.totalStake,
      params,
    );

    expect(result.accepted).toBe(false);
    expect(result.reason).toContain("deadline");
  });

  it("double commitment from same validator MUST be rejected", () => {
    const v = validators[0];

    // Manually add a commitment to the round (simulating prior accepted commitment)
    const firstCommitment: ValidatorCommitment = {
      validator: v.publicKey,
      commitment: "first-commitment-hash",
      submittedAtHeight: 6,
      signature: "sig",
    };
    const roundWithCommitment: VerificationRound = {
      ...round,
      commitments: new Map([[v.publicKey, firstCommitment]]),
    };

    // Try to add a second commitment from the same validator
    const { commitment: secondCommitment } = generateCommitment(
      v.privateKey,
      v.publicKey,
      round.id,
      "rejected",
      0.3,
      ["ref-2"],
    );
    secondCommitment.vrfOutput = "vrf-output-2";
    secondCommitment.vrfProof = "vrf-proof-2";

    const result = addCommitmentWithVRF(
      roundWithCommitment,
      secondCommitment,
      7,
      "prev-block-hash",
      v.stake,
      validatorSet.totalStake,
      params,
    );

    expect(result.accepted).toBe(false);
    expect(result.reason).toContain("Already committed");
  });

  it("reveal that does not match stored commitment hash MUST be rejected", () => {
    const v = validators[0];

    // Generate a genuine commitment
    const { commitment, salt } = generateCommitment(
      v.privateKey,
      v.publicKey,
      round.id,
      "confirmed",
      0.9,
      ["ref-1"],
    );

    // Add the commitment to the round manually
    const roundWithCommitment: VerificationRound = {
      ...round,
      phase: "reveal",
      commitments: new Map([[v.publicKey, commitment]]),
    };

    // Generate reveal with DIFFERENT verdict (won't match commitment hash)
    const mismatchedReveal = generateReveal(
      v.privateKey,
      v.publicKey,
      round.id,
      "rejected", // Different from "confirmed" used in commitment
      0.2,
      salt,
      ["ref-1"],
    );

    const result = addReveal(
      roundWithCommitment,
      mismatchedReveal,
      round.commitDeadline + 1, // in reveal phase
    );

    expect(result.accepted).toBe(false);
    expect(result.reason).toContain("does not match commitment");
  });

  it("reveal after reveal deadline MUST be rejected", () => {
    const v = validators[0];
    const { commitment, salt } = generateCommitment(
      v.privateKey,
      v.publicKey,
      round.id,
      "confirmed",
      0.9,
      ["ref-1"],
    );

    const roundInReveal: VerificationRound = {
      ...round,
      phase: "reveal",
      commitments: new Map([[v.publicKey, commitment]]),
    };

    const reveal = generateReveal(
      v.privateKey,
      v.publicKey,
      round.id,
      "confirmed",
      0.9,
      salt,
      ["ref-1"],
    );

    const result = addReveal(
      roundInReveal,
      reveal,
      round.revealDeadline + 1, // past reveal deadline
    );

    expect(result.accepted).toBe(false);
    expect(result.reason).toContain("deadline");
  });

  it("reveal without prior commitment MUST be rejected", () => {
    const v = validators[0];

    const roundInReveal: VerificationRound = {
      ...round,
      phase: "reveal",
      commitments: new Map(), // Empty - no prior commitments
    };

    const reveal = generateReveal(
      v.privateKey,
      v.publicKey,
      round.id,
      "confirmed",
      0.9,
      "some-salt",
      ["ref-1"],
    );

    const result = addReveal(roundInReveal, reveal, round.commitDeadline + 1);

    expect(result.accepted).toBe(false);
    expect(result.reason).toContain("No commitment");
  });

  it("correct reveal matches commitment: hash(nonce || verdict) = stored commitment", () => {
    const v = validators[0];
    const { commitment, salt } = generateCommitment(
      v.privateKey,
      v.publicKey,
      round.id,
      "confirmed",
      0.9,
      ["ref-1"],
    );

    const roundInReveal: VerificationRound = {
      ...round,
      phase: "reveal",
      commitments: new Map([[v.publicKey, commitment]]),
    };

    const reveal = generateReveal(
      v.privateKey,
      v.publicKey,
      round.id,
      "confirmed",
      0.9,
      salt,
      ["ref-1"],
    );

    const result = addReveal(roundInReveal, reveal, round.commitDeadline + 1);

    expect(result.accepted).toBe(true);
    expect(result.round.reveals.has(v.publicKey)).toBe(true);
  });

  it("missing reveals during aggregation: validators who committed but did not reveal get slashed", () => {
    // Build a round where 5 validators committed but only 3 revealed
    const claim = createTestClaim();
    const selectedValidators = validators.map((v, i) => ({
      validator: v.publicKey,
      vrfOutput: "",
      vrfProof: "",
      stake: v.stake,
      priority: BigInt(i),
      selected: true,
    }));

    let aggRound = startRound(
      createVerificationRound(claim, selectedValidators, 5, params),
    );

    // All 5 validators commit
    const commitData: Array<{
      v: (typeof validators)[0];
      salt: string;
      commitment: ValidatorCommitment;
    }> = [];
    const commitmentsMap = new Map<string, ValidatorCommitment>();

    for (const v of validators) {
      const { commitment, salt } = generateCommitment(
        v.privateKey,
        v.publicKey,
        aggRound.id,
        "confirmed",
        0.9,
        ["ref-1"],
      );
      commitmentsMap.set(v.publicKey, commitment);
      commitData.push({ v, salt, commitment });
    }

    aggRound = { ...aggRound, commitments: commitmentsMap, phase: "reveal" };

    // Only first 3 validators reveal
    const revealsMap = new Map<string, ReturnType<typeof generateReveal>>();
    for (let i = 0; i < 3; i++) {
      const { v, salt } = commitData[i];
      const reveal = generateReveal(
        v.privateKey,
        v.publicKey,
        aggRound.id,
        "confirmed",
        0.9,
        salt,
        ["ref-1"],
      );
      revealsMap.set(v.publicKey, reveal);
    }

    aggRound = { ...aggRound, reveals: revealsMap };

    // Build validator stakes map
    const validatorStakes = buildValidatorInfoMap(
      validators.map((v) => ({ publicKey: v.publicKey, stake: v.stake })),
    );

    // Aggregate at a block after reveal deadline
    const aggregated = aggregateResults(
      aggRound,
      validatorStakes,
      aggRound.revealDeadline + 1,
      params,
    );

    // The two validators who did not reveal should be slashed
    expect(aggregated.result).toBeDefined();
    const slashes = aggregated.result!.slashes;
    expect(slashes.has(validators[3].publicKey)).toBe(true);
    expect(slashes.has(validators[4].publicKey)).toBe(true);

    // Slash amount should be missedVerificationSlash * stake
    const expectedSlashPerValidator =
      (validators[3].stake *
        BigInt(Math.floor(params.missedVerificationSlash * 10000))) /
      10000n;
    expect(slashes.get(validators[3].publicKey)).toBe(
      expectedSlashPerValidator,
    );
    expect(slashes.get(validators[4].publicKey)).toBe(
      expectedSlashPerValidator,
    );
  });
});

// ============ 3. Quorum & Threshold (6 tests) ============

describe("Quorum & Threshold", () => {
  // Params: quorum = 0.66, acceptance = 0.77, rejection = 0.77
  const params: ConsensusParams = {
    ...DEFAULT_CONSENSUS_PARAMS,
    commitPhaseBlocks: 10,
    revealPhaseBlocks: 10,
    verificationRoundBlocks: 25,
    minValidators: 1,
    maxValidators: 100,
    validatorQuorum: 0.66,
    acceptanceThreshold: 0.77,
    rejectionThreshold: 0.77,
  };

  /**
   * Helper: build a round with N selected validators (each equal stake),
   * and have `revealCount` of them reveal with the given verdict.
   * Returns the aggregated round.
   */
  function buildAndAggregateRound(opts: {
    totalValidators: number;
    revealingValidators: number;
    verdicts: Array<"confirmed" | "rejected" | "uncertain">;
    stake?: bigint;
  }) {
    const stake = opts.stake ?? 100n * LGM;
    const allValidators = Array.from({ length: opts.totalValidators }, () => ({
      ...makeKeypair(),
      stake,
    }));

    const selectedValidators = allValidators.map((v, i) => ({
      validator: v.publicKey,
      vrfOutput: "",
      vrfProof: "",
      stake: v.stake,
      priority: BigInt(i),
      selected: true,
    }));

    const claim = createTestClaim();
    let round = startRound(
      createVerificationRound(claim, selectedValidators, 5, params),
    );

    // All validators commit
    const commitData: Array<{
      v: (typeof allValidators)[0];
      salt: string;
    }> = [];
    const commitmentsMap = new Map<string, ValidatorCommitment>();

    for (let i = 0; i < opts.revealingValidators; i++) {
      const v = allValidators[i];
      const verdict = opts.verdicts[i] ?? "confirmed";
      const { commitment, salt } = generateCommitment(
        v.privateKey,
        v.publicKey,
        round.id,
        verdict,
        0.9,
        ["ref-1"],
      );
      commitmentsMap.set(v.publicKey, commitment);
      commitData.push({ v, salt });
    }

    round = { ...round, commitments: commitmentsMap, phase: "reveal" };

    // Reveals
    const revealsMap = new Map<string, ReturnType<typeof generateReveal>>();
    for (let i = 0; i < opts.revealingValidators; i++) {
      const { v, salt } = commitData[i];
      const verdict = opts.verdicts[i] ?? "confirmed";
      const reveal = generateReveal(
        v.privateKey,
        v.publicKey,
        round.id,
        verdict,
        0.9,
        salt,
        ["ref-1"],
      );
      revealsMap.set(v.publicKey, reveal);
    }

    round = { ...round, reveals: revealsMap };

    const validatorStakes = buildValidatorInfoMap(
      allValidators.map((v) => ({
        publicKey: v.publicKey,
        stake: v.stake,
      })),
    );

    return aggregateResults(
      round,
      validatorStakes,
      round.revealDeadline + 1,
      params,
    );
  }

  it("participation at exactly 66% (quorum) MUST allow finalization", () => {
    // 100 validators, 66 participate => 66% participation (at quorum threshold)
    const verdicts: Array<"confirmed" | "rejected" | "uncertain"> =
      Array(66).fill("confirmed");

    const aggregated = buildAndAggregateRound({
      totalValidators: 100,
      revealingValidators: 66,
      verdicts,
    });

    expect(aggregated.result).toBeDefined();
    // With 66/100 participation at 0.66 quorum, and 100% confirmed,
    // outcome should be accepted (66/66 = 100% > 77% threshold)
    expect(aggregated.result!.outcome).toBe("accepted");
  });

  it("participation at 65% MUST NOT finalize (insufficient quorum)", () => {
    // 100 validators, 65 participate => 65% < 66% quorum
    const verdicts: Array<"confirmed" | "rejected" | "uncertain"> =
      Array(65).fill("confirmed");

    const aggregated = buildAndAggregateRound({
      totalValidators: 100,
      revealingValidators: 65,
      verdicts,
    });

    expect(aggregated.result).toBeDefined();
    expect(aggregated.result!.outcome).toBe("insufficient");
  });

  it("acceptance at exactly 77% threshold MUST accept the claim", () => {
    // 100 validators all participate; 77 confirm, 23 reject
    // Participation = 100% (>66%), confirmed fraction = 77/100 = 0.77
    const verdicts: Array<"confirmed" | "rejected" | "uncertain"> = [
      ...Array(77).fill("confirmed"),
      ...Array(23).fill("rejected"),
    ];

    const aggregated = buildAndAggregateRound({
      totalValidators: 100,
      revealingValidators: 100,
      verdicts,
    });

    expect(aggregated.result).toBeDefined();
    expect(aggregated.result!.outcome).toBe("accepted");
  });

  it("acceptance at 76% MUST NOT accept (below threshold)", () => {
    // 100 validators all participate; 76 confirm, 24 reject
    // confirmed fraction = 76/100 = 0.76 < 0.77
    const verdicts: Array<"confirmed" | "rejected" | "uncertain"> = [
      ...Array(76).fill("confirmed"),
      ...Array(24).fill("rejected"),
    ];

    const aggregated = buildAndAggregateRound({
      totalValidators: 100,
      revealingValidators: 100,
      verdicts,
    });

    expect(aggregated.result).toBeDefined();
    // Not enough for acceptance, not enough for rejection -> contested
    expect(aggregated.result!.outcome).toBe("contested");
  });

  it("rejection at 77% MUST reject", () => {
    // 100 validators all participate; 77 reject, 23 confirm
    const verdicts: Array<"confirmed" | "rejected" | "uncertain"> = [
      ...Array(23).fill("confirmed"),
      ...Array(77).fill("rejected"),
    ];

    const aggregated = buildAndAggregateRound({
      totalValidators: 100,
      revealingValidators: 100,
      verdicts,
    });

    expect(aggregated.result).toBeDefined();
    expect(aggregated.result!.outcome).toBe("rejected");
  });

  it("split vote (50/50) results in neither acceptance nor rejection", () => {
    // 100 validators all participate; 50 confirm, 50 reject
    const verdicts: Array<"confirmed" | "rejected" | "uncertain"> = [
      ...Array(50).fill("confirmed"),
      ...Array(50).fill("rejected"),
    ];

    const aggregated = buildAndAggregateRound({
      totalValidators: 100,
      revealingValidators: 100,
      verdicts,
    });

    expect(aggregated.result).toBeDefined();
    expect(aggregated.result!.outcome).toBe("contested");
    // Neither side reaches 77%
    expect(aggregated.result!.confidenceBreakdown.confirmed).toBeCloseTo(
      0.5,
      1,
    );
    expect(aggregated.result!.confidenceBreakdown.rejected).toBeCloseTo(0.5, 1);
  });
});
