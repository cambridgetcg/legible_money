/**
 * Finality Tests
 *
 * Tests for block finality, checkpoint verification, fork choice,
 * fact finality, and signature aggregation.
 */

import { describe, it, expect } from "vitest";
import {
  isBlockFinalized,
  getConfirmations,
  calculateFinalityProbability,
  calculateByzantineFraction,
  shouldCreateCheckpoint,
  chooseFork,
  isReorgAllowed,
  isFactFinalized,
  getFactFinalityStatus,
  aggregateCheckpointSignatures,
  getSignersFromBitmap,
  FINALITY_PARAMS,
  type Checkpoint,
  type CheckpointSignature,
} from "../../../consensus/pot/finality.js";
import type {
  ValidatorSet,
  Block,
  BlockHeader,
} from "../../../consensus/pot/types.js";

// ============ Test Helpers ============

function createMockValidatorSet(
  validators: Array<{
    key: string;
    stake: bigint;
    delegatedStake?: bigint;
    isActive?: boolean;
  }>,
): ValidatorSet {
  const validatorMap = new Map();
  let totalStake = 0n;

  for (const v of validators) {
    const stake = v.stake;
    const delegatedStake = v.delegatedStake ?? 0n;
    validatorMap.set(v.key, {
      publicKey: v.key,
      stake,
      delegatedStake,
      isActive: v.isActive ?? true,
      reputation: 0.5,
      joinedAt: 0n,
    });
    totalStake += stake + delegatedStake;
  }

  return {
    validators: validatorMap,
    totalStake,
    epoch: 1n,
    epochStartBlock: 0n,
  };
}

function createMockBlock(
  number: bigint,
  proposer: string = "validator1",
): Block {
  return {
    header: {
      number,
      timestamp: Date.now(),
      previousHash: "prev",
      stateRoot: "state",
      knowledgeRoot: "knowledge",
      transactionsRoot: "tx",
      verificationsRoot: "ver",
      proposer,
      epoch: 1n,
      parentHash: "prev",
    },
    transactions: [],
    verificationResults: [],
    attestations: [],
    signature: "sig",
  } as any;
}

// ============ Block Finality Tests ============

describe("isBlockFinalized", () => {
  it("should finalize after CONFIRMATION_DEPTH blocks", () => {
    const depth = BigInt(FINALITY_PARAMS.CONFIRMATION_DEPTH);
    expect(isBlockFinalized(10n, 10n + depth, [])).toBe(true);
    expect(isBlockFinalized(10n, 10n + depth - 1n, [])).toBe(false);
  });

  it("should finalize blocks before latest checkpoint", () => {
    const checkpoints: Checkpoint[] = [
      {
        blockNumber: 100n,
        blockHash: "hash",
        stateRoot: "state",
        knowledgeRoot: "knowledge",
        signatures: [],
        totalSigningStake: 0n,
        createdAt: Date.now(),
      },
    ];

    expect(isBlockFinalized(50n, 60n, checkpoints)).toBe(true);
    expect(isBlockFinalized(100n, 105n, checkpoints)).toBe(true);
    // Block after checkpoint, not enough confirmations
    expect(isBlockFinalized(101n, 105n, checkpoints)).toBe(false);
  });
});

describe("getConfirmations", () => {
  it("should return difference between current and target block", () => {
    expect(getConfirmations(10n, 25n)).toBe(15n);
    expect(getConfirmations(0n, 19n)).toBe(19n);
  });

  it("should return 0 for future blocks", () => {
    expect(getConfirmations(100n, 50n)).toBe(0n);
  });
});

// ============ Finality Probability Tests ============

describe("calculateFinalityProbability", () => {
  it("should return 0 for 0 confirmations", () => {
    expect(calculateFinalityProbability(0n)).toBe(0);
  });

  it("should increase with more confirmations", () => {
    const p1 = calculateFinalityProbability(1n);
    const p5 = calculateFinalityProbability(5n);
    const p19 = calculateFinalityProbability(19n);

    expect(p1).toBeGreaterThan(0);
    expect(p5).toBeGreaterThan(p1);
    expect(p19).toBeGreaterThan(p5);
  });

  it("should approach 1.0 at 19 confirmations (default 0.33 byzantine)", () => {
    const p = calculateFinalityProbability(19n, 0.33);
    expect(p).toBeGreaterThan(0.999);
  });

  it("should be lower with higher byzantine fraction", () => {
    const low = calculateFinalityProbability(5n, 0.1);
    const high = calculateFinalityProbability(5n, 0.4);

    expect(low).toBeGreaterThan(high);
  });
});

// ============ Byzantine Fraction Tests ============

describe("calculateByzantineFraction", () => {
  it("should return minimum 10% with no issues", () => {
    const vs = createMockValidatorSet([
      { key: "v1", stake: 100n },
      { key: "v2", stake: 100n },
      { key: "v3", stake: 100n },
      { key: "v4", stake: 100n },
      { key: "v5", stake: 100n },
    ]);

    const result = calculateByzantineFraction(vs);
    expect(result.byzantineFraction).toBeGreaterThanOrEqual(0.1);
  });

  it("should account for offline validators", () => {
    const vs = createMockValidatorSet([
      { key: "v1", stake: 100n },
      { key: "v2", stake: 100n },
      { key: "v3", stake: 100n },
    ]);

    const resultClean = calculateByzantineFraction(vs);
    const resultOffline = calculateByzantineFraction(vs, new Set(["v1"]));

    expect(resultOffline.byzantineFraction).toBeGreaterThanOrEqual(
      resultClean.byzantineFraction,
    );
    expect(resultOffline.breakdown.offlineStake).toBe(100n);
  });

  it("should account for flagged validators", () => {
    const vs = createMockValidatorSet([
      { key: "v1", stake: 100n },
      { key: "v2", stake: 100n },
      { key: "v3", stake: 100n },
    ]);

    const result = calculateByzantineFraction(vs, new Set(), new Set(["v1"]));
    expect(result.breakdown.flaggedStake).toBe(100n);
  });

  it("should detect concentration risk", () => {
    // One validator has 70% of stake
    const vs = createMockValidatorSet([
      { key: "v1", stake: 700n },
      { key: "v2", stake: 100n },
      { key: "v3", stake: 100n },
      { key: "v4", stake: 100n },
    ]);

    const result = calculateByzantineFraction(vs);
    expect(result.breakdown.concentrationRisk).toBeGreaterThan(0.5);
  });

  it("should cap at 0.49", () => {
    // All stake offline - should still cap at 49%
    const vs = createMockValidatorSet([
      { key: "v1", stake: 500n },
      { key: "v2", stake: 500n },
    ]);

    const result = calculateByzantineFraction(vs, new Set(["v1", "v2"]));
    expect(result.byzantineFraction).toBeLessThanOrEqual(0.49);
  });
});

// ============ Checkpoint Tests ============

describe("shouldCreateCheckpoint", () => {
  it("should create checkpoint at interval boundary", () => {
    const interval = BigInt(FINALITY_PARAMS.CHECKPOINT_INTERVAL);
    expect(shouldCreateCheckpoint(interval, 0n)).toBe(true);
    expect(shouldCreateCheckpoint(interval * 2n, interval)).toBe(true);
  });

  it("should not create checkpoint before interval", () => {
    expect(shouldCreateCheckpoint(100n, 0n)).toBe(false);
    expect(shouldCreateCheckpoint(1000n, 0n)).toBe(false);
  });
});

// ============ Fork Choice Tests ============

describe("chooseFork", () => {
  it("should return null for empty candidates", () => {
    expect(chooseFork([])).toBeNull();
  });

  it("should prefer higher attesting stake", () => {
    const result = chooseFork([
      {
        block: createMockBlock(10n, "v1"),
        attestingStake: 100n,
        chainLength: 10n,
      },
      {
        block: createMockBlock(10n, "v2"),
        attestingStake: 200n,
        chainLength: 10n,
      },
    ]);

    expect(result?.totalStakeWeight).toBe(200n);
  });

  it("should use chain length as tiebreaker", () => {
    const result = chooseFork([
      {
        block: createMockBlock(10n, "v1"),
        attestingStake: 100n,
        chainLength: 10n,
      },
      {
        block: createMockBlock(11n, "v2"),
        attestingStake: 100n,
        chainLength: 11n,
      },
    ]);

    expect(result?.chainLength).toBe(11n);
  });
});

// ============ Reorg Tests ============

describe("isReorgAllowed", () => {
  it("should allow reorg within bounds", () => {
    const result = isReorgAllowed(100n, 90n, 0n);
    expect(result.allowed).toBe(true);
  });

  it("should reject reorg past checkpoint", () => {
    const result = isReorgAllowed(100n, 40n, 50n);
    expect(result.allowed).toBe(false);
    expect(result.reason).toContain("checkpoint");
  });

  it("should reject reorg exceeding MAX_REORG_DEPTH", () => {
    const maxDepth = BigInt(FINALITY_PARAMS.MAX_REORG_DEPTH);
    const result = isReorgAllowed(200n, 200n - maxDepth - 1n, 0n);
    expect(result.allowed).toBe(false);
    expect(result.reason).toContain("exceeds maximum");
  });
});

// ============ Fact Finality Tests ============

describe("isFactFinalized", () => {
  it("should not finalize with pending challenge", () => {
    const result = isFactFinalized(10n, 100n, true, []);
    expect(result.finalized).toBe(false);
    expect(result.reason).toContain("pending challenge");
  });

  it("should not finalize without enough confirmations", () => {
    const result = isFactFinalized(10n, 15n, false, []);
    expect(result.finalized).toBe(false);
    expect(result.reason).toContain("confirmations");
  });

  it("should finalize with sufficient confirmations and no challenges", () => {
    const depth = BigInt(FINALITY_PARAMS.CONFIRMATION_DEPTH);
    const result = isFactFinalized(10n, 10n + depth, false, []);
    expect(result.finalized).toBe(true);
  });
});

describe("getFactFinalityStatus", () => {
  it("should return correct status for unfinalized fact", () => {
    const status = getFactFinalityStatus(100n, 105n, 0);
    expect(status.isFinalized).toBe(false);
    expect(status.confirmations).toBe(5n);
    expect(status.confirmationsRequired).toBe(
      FINALITY_PARAMS.CONFIRMATION_DEPTH,
    );
    expect(status.estimatedFinalityBlock).toBe(
      100n + BigInt(FINALITY_PARAMS.CONFIRMATION_DEPTH),
    );
  });

  it("should return finalized for sufficient confirmations with no challenges", () => {
    const depth = BigInt(FINALITY_PARAMS.CONFIRMATION_DEPTH);
    const status = getFactFinalityStatus(100n, 100n + depth, 0);
    expect(status.isFinalized).toBe(true);
    expect(status.finalityProbability).toBeGreaterThan(0.99);
  });

  it("should not finalize even with confirmations if challenges exist", () => {
    const depth = BigInt(FINALITY_PARAMS.CONFIRMATION_DEPTH);
    const status = getFactFinalityStatus(100n, 100n + depth, 1);
    expect(status.isFinalized).toBe(false);
    expect(status.pendingChallenges).toBe(1);
  });
});

// ============ Signature Aggregation Tests ============

describe("aggregateCheckpointSignatures", () => {
  it("should create bitmap and combined hash", () => {
    const vs = createMockValidatorSet([
      { key: "v1", stake: 100n },
      { key: "v2", stake: 100n },
      { key: "v3", stake: 100n },
    ]);

    const sigs: CheckpointSignature[] = [
      { validator: "v1", signature: "sig1", stake: 100n },
      { validator: "v3", signature: "sig3", stake: 100n },
    ];

    const agg = aggregateCheckpointSignatures(sigs, vs);
    expect(agg.signerCount).toBe(2);
    expect(agg.combinedHash).toBeTruthy();
    expect(agg.signerBitmap).toBeTruthy();
  });

  it("should produce deterministic results", () => {
    const vs = createMockValidatorSet([
      { key: "v1", stake: 100n },
      { key: "v2", stake: 100n },
    ]);

    const sigs: CheckpointSignature[] = [
      { validator: "v1", signature: "sig1", stake: 100n },
      { validator: "v2", signature: "sig2", stake: 100n },
    ];

    const agg1 = aggregateCheckpointSignatures(sigs, vs);
    const agg2 = aggregateCheckpointSignatures(sigs, vs);

    expect(agg1.combinedHash).toBe(agg2.combinedHash);
    expect(agg1.signerBitmap).toBe(agg2.signerBitmap);
  });
});

describe("getSignersFromBitmap", () => {
  it("should reconstruct signers from bitmap", () => {
    const vs = createMockValidatorSet([
      { key: "v1", stake: 100n },
      { key: "v2", stake: 100n },
      { key: "v3", stake: 100n },
    ]);

    const sigs: CheckpointSignature[] = [
      { validator: "v1", signature: "sig1", stake: 100n },
      { validator: "v3", signature: "sig3", stake: 100n },
    ];

    const agg = aggregateCheckpointSignatures(sigs, vs);
    const recovered = getSignersFromBitmap(agg, vs);

    expect(recovered).toContain("v1");
    expect(recovered).toContain("v3");
    expect(recovered).not.toContain("v2");
    expect(recovered).toHaveLength(2);
  });
});
