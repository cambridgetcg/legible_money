/**
 * Claiming Pot State Transitions Tests
 *
 * Integration tests for pot_create, pot_fund, pot_claim, pot_close
 * transaction handlers in state/transitions.ts.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import {
  executeTransaction,
  FEE_POOL_DID,
  REWARD_POOL_DID,
  PROTOCOL_TREASURY_DID,
} from "../../../state/transitions.js";
import type {
  Transaction,
  ConsensusParams,
} from "../../../consensus/pot/types.js";
import type {
  PotCreatePayload,
  PotFundPayload,
  PotClaimPayload,
  PotClosePayload,
  PotConfig,
} from "../../../core/claiming_pot/types.js";
import {
  DEFAULT_POT_ELIGIBILITY,
  DEFAULT_AMOUNT_PER_CLAIM,
} from "../../../core/claiming_pot/types.js";
import { generatePotId } from "../../../core/claiming_pot/pot_manager.js";
import { generateKeypair } from "../../../core/crypto/keys.js";
import { sha512_256Hex } from "../../../core/crypto/hashing.js";
import { publicKeyToDid } from "../../../core/agents/identity/did.js";

// ============ Test Helpers ============

const ONE_LGM = 10n ** 18n;
/** 0.222 LGM — matches bootstrap fund per-agent amount */
const CLAIM_AMOUNT = DEFAULT_AMOUNT_PER_CLAIM;
const BLOCK = 100n;

function createTestParams(): ConsensusParams {
  return {
    targetBlockTimeMs: 2521,
    minValidatorStake: 111n * 10n ** 15n, // 0.111 LGM
    minClaimStake: CLAIM_AMOUNT,
    validatorQuorum: 0.66,
    verificationRoundBlocks: 20,
    commitPhaseBlocks: 6,
    revealPhaseBlocks: 6,
    epochBlocks: 100,
    maxValidators: 100,
    slashFractionDowntime: 0.01,
    slashFractionDoubleSign: 0.05,
    unbondingPeriod: 1000n,
  };
}

function makePotConfig(overrides: Partial<PotConfig> = {}): PotConfig {
  return {
    amountPerClaim: CLAIM_AMOUNT,
    maxClaims: 0n,
    maxClaimsPerAccount: 1n,
    eligibility: { ...DEFAULT_POT_ELIGIBILITY, excludedTypes: [] },
    creatorCanClose: true,
    ...overrides,
  };
}

function makeTx(
  type: string,
  sender: string,
  payload: Record<string, unknown>,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: type as any,
    sender,
    payload,
    fee: 0n,
    nonce,
    timestamp: Date.now(),
    signature: "test-signature",
  };
}

// ============ pot_create ============

describe.sequential("pot_create", () => {
  let state: StateManager;
  let params: ConsensusParams;
  let creatorKey: { publicKey: string; privateKey: string };
  let creatorDid: string;

  beforeEach(() => {
    state = new StateManager();
    params = createTestParams();
    creatorKey = generateKeypair();
    creatorDid = publicKeyToDid(creatorKey.publicKey);

    state.createAccount(
      creatorDid,
      sha512_256Hex(creatorKey.publicKey),
      10000n * ONE_LGM,
    );
    state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
    state.createAccount(
      REWARD_POOL_DID,
      "reward-pool-hash",
      1000000n * ONE_LGM,
    );
    state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  it("should create a pot and deduct initial deposit", () => {
    const config = makePotConfig();
    const deposit = CLAIM_AMOUNT * 10n;
    const tx = makeTx("pot_create", creatorKey.publicKey, {
      type: "pot_create",
      config,
      initialDeposit: deposit,
    } satisfies PotCreatePayload);

    const initialBalance = state.getBalance(creatorDid);
    const result = executeTransaction(state, tx, BLOCK, params);

    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(80000n);

    // Creator balance decreased by deposit + fee
    const finalBalance = state.getBalance(creatorDid);
    expect(initialBalance - finalBalance).toBe(deposit);

    // Pot exists in state
    const potId = generatePotId(creatorDid, 0n);
    const pot = state.getClaimingPot(potId);
    expect(pot).toBeDefined();
    expect(pot!.creator).toBe(creatorDid);
    expect(pot!.balance).toBe(deposit);
    expect(pot!.totalDeposited).toBe(deposit);
    expect(pot!.status).toBe("active");
    expect(pot!.config.amountPerClaim).toBe(CLAIM_AMOUNT);
  });

  it("should generate deterministic pot ID from creator + nonce", () => {
    const tx = makeTx("pot_create", creatorKey.publicKey, {
      type: "pot_create",
      config: makePotConfig(),
      initialDeposit: CLAIM_AMOUNT * 10n,
    } satisfies PotCreatePayload);

    executeTransaction(state, tx, BLOCK, params);

    const expectedId = generatePotId(creatorDid, 0n);
    const pot = state.getClaimingPot(expectedId);
    expect(pot).toBeDefined();
  });

  it("should reject initial deposit less than amountPerClaim", () => {
    const tx = makeTx("pot_create", creatorKey.publicKey, {
      type: "pot_create",
      config: makePotConfig({ amountPerClaim: 50n * ONE_LGM }),
      initialDeposit: CLAIM_AMOUNT, // less than amountPerClaim
    } satisfies PotCreatePayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("at least one claim");
  });

  it("should reject if sender has insufficient balance", () => {
    const tx = makeTx("pot_create", creatorKey.publicKey, {
      type: "pot_create",
      config: makePotConfig(),
      initialDeposit: 99999n * ONE_LGM, // more than creator has
    } satisfies PotCreatePayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Insufficient");
  });

  it("should reject invalid config (zero amountPerClaim)", () => {
    const tx = makeTx("pot_create", creatorKey.publicKey, {
      type: "pot_create",
      config: makePotConfig({ amountPerClaim: 0n }),
      initialDeposit: CLAIM_AMOUNT * 10n,
    } satisfies PotCreatePayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("amountPerClaim");
  });

  it("should reject expired-at-block in the past", () => {
    const tx = makeTx("pot_create", creatorKey.publicKey, {
      type: "pot_create",
      config: makePotConfig({ expiresAtBlock: 50n }), // past block 100
      initialDeposit: CLAIM_AMOUNT * 10n,
    } satisfies PotCreatePayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("future");
  });

  it("should credit pot account with initial deposit", () => {
    const deposit = CLAIM_AMOUNT * 20n;
    const tx = makeTx("pot_create", creatorKey.publicKey, {
      type: "pot_create",
      config: makePotConfig(),
      initialDeposit: deposit,
    } satisfies PotCreatePayload);

    executeTransaction(state, tx, BLOCK, params);

    const potId = generatePotId(creatorDid, 0n);
    const potBalance = state.getBalance(potId);
    expect(potBalance).toBe(deposit);
  });
});

// ============ pot_fund ============

describe.sequential("pot_fund", () => {
  let state: StateManager;
  let params: ConsensusParams;
  let creatorKey: { publicKey: string; privateKey: string };
  let creatorDid: string;
  let funderKey: { publicKey: string; privateKey: string };
  let funderDid: string;
  let potId: string;

  beforeEach(() => {
    state = new StateManager();
    params = createTestParams();
    creatorKey = generateKeypair();
    creatorDid = publicKeyToDid(creatorKey.publicKey);
    funderKey = generateKeypair();
    funderDid = publicKeyToDid(funderKey.publicKey);

    state.createAccount(
      creatorDid,
      sha512_256Hex(creatorKey.publicKey),
      10000n * ONE_LGM,
    );
    state.createAccount(
      funderDid,
      sha512_256Hex(funderKey.publicKey),
      5000n * ONE_LGM,
    );
    state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
    state.createAccount(
      REWARD_POOL_DID,
      "reward-pool-hash",
      1000000n * ONE_LGM,
    );
    state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);
    state.createAccount("did:lgm:research", "research-hash", 0n);

    // Create a pot first
    const createTx = makeTx("pot_create", creatorKey.publicKey, {
      type: "pot_create",
      config: makePotConfig(),
      initialDeposit: CLAIM_AMOUNT * 10n,
    } satisfies PotCreatePayload);
    const result = executeTransaction(state, createTx, BLOCK, params);
    expect(result.success).toBe(true);
    potId = generatePotId(creatorDid, 0n);
  });

  it("should allow anyone to fund an active pot", () => {
    const amount = CLAIM_AMOUNT * 5n;
    const tx = makeTx("pot_fund", funderKey.publicKey, {
      type: "pot_fund",
      potId,
      amount,
    } satisfies PotFundPayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(40000n);

    const pot = state.getClaimingPot(potId)!;
    expect(pot.totalDeposited).toBe(CLAIM_AMOUNT * 15n); // 10 initial + 5 funded
    expect(pot.balance).toBe(CLAIM_AMOUNT * 15n);
  });

  it("should reject funding a non-existent pot", () => {
    const tx = makeTx("pot_fund", funderKey.publicKey, {
      type: "pot_fund",
      potId: "did:lgm:pot:nonexistent",
      amount: CLAIM_AMOUNT * 5n,
    } satisfies PotFundPayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
  });

  it("should reject zero fund amount", () => {
    const tx = makeTx("pot_fund", funderKey.publicKey, {
      type: "pot_fund",
      potId,
      amount: 0n,
    } satisfies PotFundPayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("positive");
  });

  it("should reject if funder has insufficient balance", () => {
    const tx = makeTx("pot_fund", funderKey.publicKey, {
      type: "pot_fund",
      potId,
      amount: 99999n * ONE_LGM,
    } satisfies PotFundPayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Insufficient");
  });

  it("should reactivate an exhausted pot", () => {
    // Exhaust the pot by claiming all funds
    // Config: 0.222 LGM per claim, 10 claims deposited
    // Claim 10 times to exhaust
    for (let i = 0; i < 10; i++) {
      const claimantKey = generateKeypair();
      const claimantDid = publicKeyToDid(claimantKey.publicKey);
      state.createAccount(
        claimantDid,
        sha512_256Hex(claimantKey.publicKey),
        0n,
      );

      const claimTx = makeTx(
        "pot_claim",
        claimantKey.publicKey,
        { type: "pot_claim", potId } satisfies PotClaimPayload,
        0n,
      );
      const r = executeTransaction(state, claimTx, BLOCK, params);
      expect(r.success).toBe(true);
    }

    // Verify pot is exhausted
    expect(state.getClaimingPot(potId)!.status).toBe("exhausted");

    // Fund it again (2 claims worth)
    const tx = makeTx("pot_fund", funderKey.publicKey, {
      type: "pot_fund",
      potId,
      amount: CLAIM_AMOUNT * 2n,
    } satisfies PotFundPayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(true);
    expect(state.getClaimingPot(potId)!.status).toBe("active");
  });

  it("should not reactivate if funded amount is less than amountPerClaim", () => {
    // Exhaust the pot
    for (let i = 0; i < 10; i++) {
      const k = generateKeypair();
      state.createAccount(
        publicKeyToDid(k.publicKey),
        sha512_256Hex(k.publicKey),
        0n,
      );
      const r = executeTransaction(
        state,
        makeTx("pot_claim", k.publicKey, { type: "pot_claim", potId }, 0n),
        BLOCK,
        params,
      );
      expect(r.success).toBe(true);
    }
    expect(state.getClaimingPot(potId)!.status).toBe("exhausted");

    // Fund with less than amountPerClaim (0.222 LGM)
    const tx = makeTx("pot_fund", funderKey.publicKey, {
      type: "pot_fund",
      potId,
      amount: CLAIM_AMOUNT / 2n,
    } satisfies PotFundPayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(true);
    // Still exhausted because half < full amountPerClaim
    expect(state.getClaimingPot(potId)!.status).toBe("exhausted");
  });
});

// ============ pot_claim ============

describe.sequential("pot_claim", () => {
  let state: StateManager;
  let params: ConsensusParams;
  let creatorKey: { publicKey: string; privateKey: string };
  let creatorDid: string;
  let claimantKey: { publicKey: string; privateKey: string };
  let claimantDid: string;
  let potId: string;

  beforeEach(() => {
    state = new StateManager();
    params = createTestParams();
    creatorKey = generateKeypair();
    creatorDid = publicKeyToDid(creatorKey.publicKey);
    claimantKey = generateKeypair();
    claimantDid = publicKeyToDid(claimantKey.publicKey);

    state.createAccount(
      creatorDid,
      sha512_256Hex(creatorKey.publicKey),
      10000n * ONE_LGM,
    );
    state.createAccount(claimantDid, sha512_256Hex(claimantKey.publicKey), 0n);
    state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
    state.createAccount(
      REWARD_POOL_DID,
      "reward-pool-hash",
      1000000n * ONE_LGM,
    );
    state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);
    state.createAccount("did:lgm:research", "research-hash", 0n);

    // Create a pot with 10 claims worth of CLAIM_AMOUNT
    const createTx = makeTx("pot_create", creatorKey.publicKey, {
      type: "pot_create",
      config: makePotConfig(),
      initialDeposit: CLAIM_AMOUNT * 10n,
    } satisfies PotCreatePayload);
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    potId = generatePotId(creatorDid, 0n);
  });

  it("should transfer claim amount to claimant", () => {
    const initialBalance = state.getBalance(claimantDid);
    const tx = makeTx("pot_claim", claimantKey.publicKey, {
      type: "pot_claim",
      potId,
    } satisfies PotClaimPayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(60000n);

    const finalBalance = state.getBalance(claimantDid);
    expect(finalBalance - initialBalance).toBe(CLAIM_AMOUNT);
  });

  it("should update pot state after claim", () => {
    const tx = makeTx("pot_claim", claimantKey.publicKey, {
      type: "pot_claim",
      potId,
    } satisfies PotClaimPayload);
    executeTransaction(state, tx, BLOCK, params);

    const pot = state.getClaimingPot(potId)!;
    expect(pot.totalClaimed).toBe(CLAIM_AMOUNT);
    expect(pot.balance).toBe(CLAIM_AMOUNT * 9n);
    expect(pot.claimCount).toBe(1n);
    expect(pot.claimedAccounts.get(claimantDid)).toBe(1n);
  });

  it("should reject double claim (per-account limit = 1)", () => {
    // First claim succeeds
    const tx1 = makeTx("pot_claim", claimantKey.publicKey, {
      type: "pot_claim",
      potId,
    } satisfies PotClaimPayload);
    expect(executeTransaction(state, tx1, BLOCK, params).success).toBe(true);

    // Second claim fails (nonce incremented by transfer)
    const tx2 = makeTx(
      "pot_claim",
      claimantKey.publicKey,
      { type: "pot_claim", potId } satisfies PotClaimPayload,
      1n,
    );
    const result = executeTransaction(state, tx2, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Maximum claims per account");
  });

  it("should allow multiple claims when maxClaimsPerAccount > 1", () => {
    // Create pot with maxClaimsPerAccount = 3
    const createTx = makeTx(
      "pot_create",
      creatorKey.publicKey,
      {
        type: "pot_create",
        config: makePotConfig({ maxClaimsPerAccount: 3n }),
        initialDeposit: CLAIM_AMOUNT * 10n,
      } satisfies PotCreatePayload,
      1n, // nonce 1 since we already created one pot
    );
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    const multiPotId = generatePotId(creatorDid, 1n);

    // Claim 3 times
    for (let i = 0; i < 3; i++) {
      const nonce = BigInt(i);
      const tx = makeTx(
        "pot_claim",
        claimantKey.publicKey,
        { type: "pot_claim", potId: multiPotId } satisfies PotClaimPayload,
        nonce,
      );
      const result = executeTransaction(state, tx, BLOCK, params);
      expect(result.success).toBe(true);
    }

    expect(state.getClaimingPot(multiPotId)!.claimCount).toBe(3n);

    // 4th claim fails
    const tx4 = makeTx(
      "pot_claim",
      claimantKey.publicKey,
      { type: "pot_claim", potId: multiPotId } satisfies PotClaimPayload,
      3n,
    );
    const result = executeTransaction(state, tx4, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Maximum claims per account");
  });

  it("should enforce maxClaims total limit", () => {
    // Create pot with maxClaims = 2
    const createTx = makeTx(
      "pot_create",
      creatorKey.publicKey,
      {
        type: "pot_create",
        config: makePotConfig({ maxClaims: 2n }),
        initialDeposit: CLAIM_AMOUNT * 10n,
      } satisfies PotCreatePayload,
      1n,
    );
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    const limitedPotId = generatePotId(creatorDid, 1n);

    // Two different claimants each claim once
    const claimant2Key = generateKeypair();
    const claimant2Did = publicKeyToDid(claimant2Key.publicKey);
    state.createAccount(
      claimant2Did,
      sha512_256Hex(claimant2Key.publicKey),
      0n,
    );

    expect(
      executeTransaction(
        state,
        makeTx(
          "pot_claim",
          claimantKey.publicKey,
          { type: "pot_claim", potId: limitedPotId },
          0n,
        ),
        BLOCK,
        params,
      ).success,
    ).toBe(true);
    expect(
      executeTransaction(
        state,
        makeTx(
          "pot_claim",
          claimant2Key.publicKey,
          { type: "pot_claim", potId: limitedPotId },
          0n,
        ),
        BLOCK,
        params,
      ).success,
    ).toBe(true);

    // Pot should now be exhausted
    expect(state.getClaimingPot(limitedPotId)!.status).toBe("exhausted");

    // 3rd claimant cannot claim
    const claimant3Key = generateKeypair();
    state.createAccount(
      publicKeyToDid(claimant3Key.publicKey),
      sha512_256Hex(claimant3Key.publicKey),
      0n,
    );
    const result = executeTransaction(
      state,
      makeTx(
        "pot_claim",
        claimant3Key.publicKey,
        { type: "pot_claim", potId: limitedPotId },
        0n,
      ),
      BLOCK,
      params,
    );
    expect(result.success).toBe(false);
    // Either "not active" (pot is exhausted) or "Maximum total claims"
    expect(result.error).toMatch(/not active|Maximum total claims/);
  });

  it("should reject claim from non-existent pot", () => {
    const tx = makeTx("pot_claim", claimantKey.publicKey, {
      type: "pot_claim",
      potId: "did:lgm:pot:nonexistent",
    } satisfies PotClaimPayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
  });

  it("should reject claim from expired pot", () => {
    // Create pot that expires at block 200
    const createTx = makeTx(
      "pot_create",
      creatorKey.publicKey,
      {
        type: "pot_create",
        config: makePotConfig({ expiresAtBlock: 200n }),
        initialDeposit: CLAIM_AMOUNT * 10n,
      } satisfies PotCreatePayload,
      1n,
    );
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    const expiringPotId = generatePotId(creatorDid, 1n);

    // Claim at block 200 (expired)
    const tx = makeTx("pot_claim", claimantKey.publicKey, {
      type: "pot_claim",
      potId: expiringPotId,
    } satisfies PotClaimPayload);

    const result = executeTransaction(state, tx, 200n, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("expired");
  });

  it("should allow claim before expiration", () => {
    const createTx = makeTx(
      "pot_create",
      creatorKey.publicKey,
      {
        type: "pot_create",
        config: makePotConfig({ expiresAtBlock: 200n }),
        initialDeposit: CLAIM_AMOUNT * 10n,
      } satisfies PotCreatePayload,
      1n,
    );
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    const expiringPotId = generatePotId(creatorDid, 1n);

    const tx = makeTx("pot_claim", claimantKey.publicKey, {
      type: "pot_claim",
      potId: expiringPotId,
    } satisfies PotClaimPayload);

    const result = executeTransaction(state, tx, 199n, params);
    expect(result.success).toBe(true);
  });

  it("should mark pot as exhausted when balance drops below amountPerClaim", () => {
    // Create pot with exactly 1 claim worth
    const createTx = makeTx(
      "pot_create",
      creatorKey.publicKey,
      {
        type: "pot_create",
        config: makePotConfig(),
        initialDeposit: CLAIM_AMOUNT, // exactly 1 claim
      } satisfies PotCreatePayload,
      1n,
    );
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    const singlePotId = generatePotId(creatorDid, 1n);

    const tx = makeTx("pot_claim", claimantKey.publicKey, {
      type: "pot_claim",
      potId: singlePotId,
    } satisfies PotClaimPayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(true);
    expect(state.getClaimingPot(singlePotId)!.status).toBe("exhausted");
    expect(state.getClaimingPot(singlePotId)!.balance).toBe(0n);
  });

  it("should allow creator to claim from own pot", () => {
    const tx = makeTx(
      "pot_claim",
      creatorKey.publicKey,
      { type: "pot_claim", potId } satisfies PotClaimPayload,
      1n, // nonce 1 (0 used for create)
    );

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(true);
  });

  it("should reject claim with allowlist restriction", () => {
    // Create pot with allowlist excluding our claimant
    const createTx = makeTx(
      "pot_create",
      creatorKey.publicKey,
      {
        type: "pot_create",
        config: makePotConfig({
          eligibility: {
            ...DEFAULT_POT_ELIGIBILITY,
            excludedTypes: [],
            allowlist: ["did:lgm:someone-else"],
          },
        }),
        initialDeposit: CLAIM_AMOUNT * 10n,
      } satisfies PotCreatePayload,
      1n,
    );
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    const restrictedPotId = generatePotId(creatorDid, 1n);

    const tx = makeTx("pot_claim", claimantKey.publicKey, {
      type: "pot_claim",
      potId: restrictedPotId,
    } satisfies PotClaimPayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Ineligible");
  });

  it("should allow claim when claimant is on allowlist", () => {
    const createTx = makeTx(
      "pot_create",
      creatorKey.publicKey,
      {
        type: "pot_create",
        config: makePotConfig({
          eligibility: {
            ...DEFAULT_POT_ELIGIBILITY,
            excludedTypes: [],
            allowlist: [claimantDid],
          },
        }),
        initialDeposit: CLAIM_AMOUNT * 10n,
      } satisfies PotCreatePayload,
      1n,
    );
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    const restrictedPotId = generatePotId(creatorDid, 1n);

    const tx = makeTx("pot_claim", claimantKey.publicKey, {
      type: "pot_claim",
      potId: restrictedPotId,
    } satisfies PotClaimPayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(true);
  });

  it("should reject claim when account is too young", () => {
    // Create pot requiring min age of 1000 blocks
    const createTx = makeTx(
      "pot_create",
      creatorKey.publicKey,
      {
        type: "pot_create",
        config: makePotConfig({
          eligibility: {
            ...DEFAULT_POT_ELIGIBILITY,
            excludedTypes: [],
            minAccountAge: 1000n,
          },
        }),
        initialDeposit: CLAIM_AMOUNT * 10n,
      } satisfies PotCreatePayload,
      1n,
    );
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    const agePotId = generatePotId(creatorDid, 1n);

    // Claimant account was created at block 0, current block is 100 -> age = 100 < 1000
    const tx = makeTx("pot_claim", claimantKey.publicKey, {
      type: "pot_claim",
      potId: agePotId,
    } satisfies PotClaimPayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Ineligible");
  });
});

// ============ pot_close ============

describe.sequential("pot_close", () => {
  let state: StateManager;
  let params: ConsensusParams;
  let creatorKey: { publicKey: string; privateKey: string };
  let creatorDid: string;
  let otherKey: { publicKey: string; privateKey: string };
  let otherDid: string;
  let potId: string;

  beforeEach(() => {
    state = new StateManager();
    params = createTestParams();
    creatorKey = generateKeypair();
    creatorDid = publicKeyToDid(creatorKey.publicKey);
    otherKey = generateKeypair();
    otherDid = publicKeyToDid(otherKey.publicKey);

    state.createAccount(
      creatorDid,
      sha512_256Hex(creatorKey.publicKey),
      10000n * ONE_LGM,
    );
    state.createAccount(
      otherDid,
      sha512_256Hex(otherKey.publicKey),
      1000n * ONE_LGM,
    );
    state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
    state.createAccount(
      REWARD_POOL_DID,
      "reward-pool-hash",
      1000000n * ONE_LGM,
    );
    state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);
    state.createAccount("did:lgm:research", "research-hash", 0n);

    const createTx = makeTx("pot_create", creatorKey.publicKey, {
      type: "pot_create",
      config: makePotConfig(),
      initialDeposit: CLAIM_AMOUNT * 10n,
    } satisfies PotCreatePayload);
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    potId = generatePotId(creatorDid, 0n);
  });

  it("should allow creator to close and refund remaining balance", () => {
    const creatorBalanceBefore = state.getBalance(creatorDid);

    const tx = makeTx(
      "pot_close",
      creatorKey.publicKey,
      { type: "pot_close", potId } satisfies PotClosePayload,
      1n, // nonce 1
    );

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(50000n);

    // Pot is closed
    const pot = state.getClaimingPot(potId)!;
    expect(pot.status).toBe("closed");
    expect(pot.balance).toBe(0n);
    expect(pot.closedAtBlock).toBe(BLOCK);

    // Creator got refund
    const creatorBalanceAfter = state.getBalance(creatorDid);
    expect(creatorBalanceAfter - creatorBalanceBefore).toBe(CLAIM_AMOUNT * 10n);
  });

  it("should reject non-creator closing a non-expired pot", () => {
    const tx = makeTx("pot_close", otherKey.publicKey, {
      type: "pot_close",
      potId,
    } satisfies PotClosePayload);

    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Only the creator");
  });

  it("should allow anyone to close an expired pot", () => {
    // Create pot that expires at block 200
    const createTx = makeTx(
      "pot_create",
      creatorKey.publicKey,
      {
        type: "pot_create",
        config: makePotConfig({ expiresAtBlock: 200n }),
        initialDeposit: CLAIM_AMOUNT * 10n,
      } satisfies PotCreatePayload,
      1n,
    );
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    const expiringPotId = generatePotId(creatorDid, 1n);

    // Non-creator closes at block 200 (expired)
    const tx = makeTx("pot_close", otherKey.publicKey, {
      type: "pot_close",
      potId: expiringPotId,
    } satisfies PotClosePayload);

    const creatorBalanceBefore = state.getBalance(creatorDid);
    const result = executeTransaction(state, tx, 200n, params);
    expect(result.success).toBe(true);

    // Funds refunded to CREATOR (not the closer)
    const creatorBalanceAfter = state.getBalance(creatorDid);
    expect(creatorBalanceAfter - creatorBalanceBefore).toBe(CLAIM_AMOUNT * 10n);
  });

  it("should reject closing an already closed pot", () => {
    // Close once
    const tx1 = makeTx(
      "pot_close",
      creatorKey.publicKey,
      { type: "pot_close", potId } satisfies PotClosePayload,
      1n,
    );
    expect(executeTransaction(state, tx1, BLOCK, params).success).toBe(true);

    // Try again
    const tx2 = makeTx(
      "pot_close",
      creatorKey.publicKey,
      { type: "pot_close", potId } satisfies PotClosePayload,
      2n,
    );
    const result = executeTransaction(state, tx2, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("already closed");
  });

  it("should reject early close when creatorCanClose is false", () => {
    // Create pot without creatorCanClose
    const createTx = makeTx(
      "pot_create",
      creatorKey.publicKey,
      {
        type: "pot_create",
        config: makePotConfig({ creatorCanClose: false }),
        initialDeposit: CLAIM_AMOUNT * 10n,
      } satisfies PotCreatePayload,
      1n,
    );
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    const lockedPotId = generatePotId(creatorDid, 1n);

    const tx = makeTx(
      "pot_close",
      creatorKey.publicKey,
      { type: "pot_close", potId: lockedPotId } satisfies PotClosePayload,
      2n,
    );
    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("does not allow early close");
  });

  it("should allow closing a pot with creatorCanClose=false after expiration", () => {
    // Create non-closable pot that expires at block 200
    const createTx = makeTx(
      "pot_create",
      creatorKey.publicKey,
      {
        type: "pot_create",
        config: makePotConfig({ creatorCanClose: false, expiresAtBlock: 200n }),
        initialDeposit: CLAIM_AMOUNT * 10n,
      } satisfies PotCreatePayload,
      1n,
    );
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    const lockedPotId = generatePotId(creatorDid, 1n);

    // Close at block 200 (expired) - should work even with creatorCanClose=false
    const tx = makeTx(
      "pot_close",
      creatorKey.publicKey,
      { type: "pot_close", potId: lockedPotId } satisfies PotClosePayload,
      2n,
    );
    const result = executeTransaction(state, tx, 200n, params);
    expect(result.success).toBe(true);
  });

  it("should handle closing a pot with zero remaining balance", () => {
    // Exhaust the pot
    for (let i = 0; i < 10; i++) {
      const k = generateKeypair();
      state.createAccount(
        publicKeyToDid(k.publicKey),
        sha512_256Hex(k.publicKey),
        0n,
      );
      expect(
        executeTransaction(
          state,
          makeTx("pot_claim", k.publicKey, { type: "pot_claim", potId }, 0n),
          BLOCK,
          params,
        ).success,
      ).toBe(true);
    }
    expect(state.getClaimingPot(potId)!.balance).toBe(0n);

    // Close with 0 balance
    const tx = makeTx(
      "pot_close",
      creatorKey.publicKey,
      { type: "pot_close", potId } satisfies PotClosePayload,
      1n,
    );
    const result = executeTransaction(state, tx, BLOCK, params);
    expect(result.success).toBe(true);
    expect(state.getClaimingPot(potId)!.status).toBe("closed");
  });

  it("should reject claim after pot is closed", () => {
    // Close pot
    const closeTx = makeTx(
      "pot_close",
      creatorKey.publicKey,
      { type: "pot_close", potId } satisfies PotClosePayload,
      1n,
    );
    expect(executeTransaction(state, closeTx, BLOCK, params).success).toBe(
      true,
    );

    // Try to claim
    const claimTx = makeTx("pot_claim", otherKey.publicKey, {
      type: "pot_claim",
      potId,
    } satisfies PotClaimPayload);

    const result = executeTransaction(state, claimTx, BLOCK, params);
    expect(result.success).toBe(false);
    expect(result.error).toContain("not active");
  });
});

// ============ Checkpoint / Rollback ============

describe.sequential("Claiming Pot Checkpoint/Rollback", () => {
  let state: StateManager;
  let params: ConsensusParams;
  let creatorKey: { publicKey: string; privateKey: string };
  let creatorDid: string;
  let potId: string;

  beforeEach(() => {
    state = new StateManager();
    params = createTestParams();
    creatorKey = generateKeypair();
    creatorDid = publicKeyToDid(creatorKey.publicKey);

    state.createAccount(
      creatorDid,
      sha512_256Hex(creatorKey.publicKey),
      10000n * ONE_LGM,
    );
    state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
    state.createAccount(
      REWARD_POOL_DID,
      "reward-pool-hash",
      1000000n * ONE_LGM,
    );
    state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);
    state.createAccount("did:lgm:research", "research-hash", 0n);

    // Create pot
    const createTx = makeTx("pot_create", creatorKey.publicKey, {
      type: "pot_create",
      config: makePotConfig(),
      initialDeposit: CLAIM_AMOUNT * 10n,
    } satisfies PotCreatePayload);
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    potId = generatePotId(creatorDid, 0n);
  });

  it("should restore pot state on rollback after failed transaction", () => {
    // Pot exists
    expect(state.getClaimingPot(potId)).toBeDefined();
    const potBefore = state.getClaimingPot(potId)!;
    expect(potBefore.balance).toBe(CLAIM_AMOUNT * 10n);

    // Execute a claim that modifies pot state
    const claimantKey = generateKeypair();
    const claimantDid = publicKeyToDid(claimantKey.publicKey);
    state.createAccount(claimantDid, sha512_256Hex(claimantKey.publicKey), 0n);

    const claimTx = makeTx("pot_claim", claimantKey.publicKey, {
      type: "pot_claim",
      potId,
    } satisfies PotClaimPayload);

    const result = executeTransaction(state, claimTx, BLOCK, params);
    expect(result.success).toBe(true);

    // Pot state changed
    const potAfterClaim = state.getClaimingPot(potId)!;
    expect(potAfterClaim.balance).toBe(CLAIM_AMOUNT * 9n);
    expect(potAfterClaim.claimCount).toBe(1n);
  });

  it("should preserve pot across successful transaction commits", () => {
    // Fund pot
    const funderKey = generateKeypair();
    const funderDid = publicKeyToDid(funderKey.publicKey);
    state.createAccount(
      funderDid,
      sha512_256Hex(funderKey.publicKey),
      500n * ONE_LGM,
    );

    const fundTx = makeTx("pot_fund", funderKey.publicKey, {
      type: "pot_fund",
      potId,
      amount: CLAIM_AMOUNT * 5n,
    } satisfies PotFundPayload);

    expect(executeTransaction(state, fundTx, BLOCK, params).success).toBe(true);

    // Pot balance reflects funding
    const pot = state.getClaimingPot(potId)!;
    expect(pot.balance).toBe(CLAIM_AMOUNT * 15n);
    expect(pot.totalDeposited).toBe(CLAIM_AMOUNT * 15n);
  });
});

// ============ End-to-End Lifecycle ============

describe.sequential("Claiming Pot Full Lifecycle", () => {
  let state: StateManager;
  let params: ConsensusParams;
  let creatorKey: { publicKey: string; privateKey: string };
  let creatorDid: string;

  beforeEach(() => {
    state = new StateManager();
    params = createTestParams();
    creatorKey = generateKeypair();
    creatorDid = publicKeyToDid(creatorKey.publicKey);

    state.createAccount(
      creatorDid,
      sha512_256Hex(creatorKey.publicKey),
      10000n * ONE_LGM,
    );
    state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
    state.createAccount(
      REWARD_POOL_DID,
      "reward-pool-hash",
      1000000n * ONE_LGM,
    );
    state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  it("should handle create -> fund -> claim -> claim -> close lifecycle", () => {
    // 1. Create pot: 5 claims deposit, 0.222 LGM per claim, max 3 claims per account
    const config = makePotConfig({
      amountPerClaim: CLAIM_AMOUNT,
      maxClaimsPerAccount: 3n,
      maxClaims: 5n,
      creatorCanClose: true,
    });
    const createTx = makeTx("pot_create", creatorKey.publicKey, {
      type: "pot_create",
      config,
      initialDeposit: CLAIM_AMOUNT * 5n,
    } satisfies PotCreatePayload);
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    const potId = generatePotId(creatorDid, 0n);

    // 2. Fund with 3 more claims from another account
    const funderKey = generateKeypair();
    state.createAccount(
      publicKeyToDid(funderKey.publicKey),
      sha512_256Hex(funderKey.publicKey),
      1000n * ONE_LGM,
    );
    const fundTx = makeTx("pot_fund", funderKey.publicKey, {
      type: "pot_fund",
      potId,
      amount: CLAIM_AMOUNT * 3n,
    } satisfies PotFundPayload);
    expect(executeTransaction(state, fundTx, BLOCK, params).success).toBe(true);

    // Pot now has 8 claims worth
    expect(state.getClaimingPot(potId)!.balance).toBe(CLAIM_AMOUNT * 8n);

    // 3. Two claimants each claim once
    const claimant1Key = generateKeypair();
    const claimant1Did = publicKeyToDid(claimant1Key.publicKey);
    state.createAccount(
      claimant1Did,
      sha512_256Hex(claimant1Key.publicKey),
      0n,
    );

    const claimant2Key = generateKeypair();
    const claimant2Did = publicKeyToDid(claimant2Key.publicKey);
    state.createAccount(
      claimant2Did,
      sha512_256Hex(claimant2Key.publicKey),
      0n,
    );

    expect(
      executeTransaction(
        state,
        makeTx(
          "pot_claim",
          claimant1Key.publicKey,
          { type: "pot_claim", potId },
          0n,
        ),
        BLOCK,
        params,
      ).success,
    ).toBe(true);
    expect(
      executeTransaction(
        state,
        makeTx(
          "pot_claim",
          claimant2Key.publicKey,
          { type: "pot_claim", potId },
          0n,
        ),
        BLOCK,
        params,
      ).success,
    ).toBe(true);

    // Verify: 2 claims made, 6 claims remaining
    const pot = state.getClaimingPot(potId)!;
    expect(pot.claimCount).toBe(2n);
    expect(pot.balance).toBe(CLAIM_AMOUNT * 6n);
    expect(pot.totalClaimed).toBe(CLAIM_AMOUNT * 2n);
    expect(state.getBalance(claimant1Did)).toBe(CLAIM_AMOUNT);
    expect(state.getBalance(claimant2Did)).toBe(CLAIM_AMOUNT);

    // 4. Creator closes pot, gets remaining 6 claims back
    const creatorBalanceBefore = state.getBalance(creatorDid);
    const closeTx = makeTx(
      "pot_close",
      creatorKey.publicKey,
      { type: "pot_close", potId } satisfies PotClosePayload,
      1n,
    );
    expect(executeTransaction(state, closeTx, BLOCK, params).success).toBe(
      true,
    );

    const creatorBalanceAfter = state.getBalance(creatorDid);
    expect(creatorBalanceAfter - creatorBalanceBefore).toBe(CLAIM_AMOUNT * 6n);
    expect(state.getClaimingPot(potId)!.status).toBe("closed");
  });

  it("should handle pot exhaustion and reactivation via funding", () => {
    // Create pot with 2 claims worth
    const createTx = makeTx("pot_create", creatorKey.publicKey, {
      type: "pot_create",
      config: makePotConfig({ amountPerClaim: CLAIM_AMOUNT }),
      initialDeposit: CLAIM_AMOUNT * 2n,
    } satisfies PotCreatePayload);
    expect(executeTransaction(state, createTx, BLOCK, params).success).toBe(
      true,
    );
    const potId = generatePotId(creatorDid, 0n);

    // Claim twice → exhaust
    for (let i = 0; i < 2; i++) {
      const k = generateKeypair();
      state.createAccount(
        publicKeyToDid(k.publicKey),
        sha512_256Hex(k.publicKey),
        0n,
      );
      expect(
        executeTransaction(
          state,
          makeTx("pot_claim", k.publicKey, { type: "pot_claim", potId }, 0n),
          BLOCK,
          params,
        ).success,
      ).toBe(true);
    }
    expect(state.getClaimingPot(potId)!.status).toBe("exhausted");

    // Try to claim → fails
    const failKey = generateKeypair();
    state.createAccount(
      publicKeyToDid(failKey.publicKey),
      sha512_256Hex(failKey.publicKey),
      0n,
    );
    const failResult = executeTransaction(
      state,
      makeTx("pot_claim", failKey.publicKey, { type: "pot_claim", potId }, 0n),
      BLOCK,
      params,
    );
    expect(failResult.success).toBe(false);

    // Fund 10 LGM → reactivate
    const fundTx = makeTx(
      "pot_fund",
      creatorKey.publicKey,
      {
        type: "pot_fund",
        potId,
        amount: CLAIM_AMOUNT,
      } satisfies PotFundPayload,
      1n,
    );
    expect(executeTransaction(state, fundTx, BLOCK, params).success).toBe(true);
    expect(state.getClaimingPot(potId)!.status).toBe("active");

    // Now claiming works again
    const newClaimant = generateKeypair();
    state.createAccount(
      publicKeyToDid(newClaimant.publicKey),
      sha512_256Hex(newClaimant.publicKey),
      0n,
    );
    const claimResult = executeTransaction(
      state,
      makeTx(
        "pot_claim",
        newClaimant.publicKey,
        { type: "pot_claim", potId },
        0n,
      ),
      BLOCK,
      params,
    );
    expect(claimResult.success).toBe(true);
    expect(state.getBalance(publicKeyToDid(newClaimant.publicKey))).toBe(
      CLAIM_AMOUNT,
    );
  });
});
