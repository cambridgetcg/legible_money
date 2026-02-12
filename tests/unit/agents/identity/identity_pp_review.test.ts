/**
 * Agent Identity PP Review Tests
 *
 * Tests for fixes identified during the Agent Identity PP review:
 * - P1-I1: account.ts validateSessionKeyAction recipient checking
 * - P1-I2: account.ts createSessionKeyAuth lastResetBlock init
 * - P1-I3: mpc.ts addParty duplicate ID check
 * - P1-I4: mpc.ts removeParty existence check
 * - P1-I5: session_keys.ts shouldDiscardSession block-height expiry
 * - P2: mpc.ts initializeMPCWallet no longer async
 * - Coverage: addShardToRecovery with config (holder identity verification)
 * - Coverage: addGuardian duplicate rejection
 * - Coverage: block-based daily reset in session_keys.ts
 * - Coverage: recipient restrictions in session_keys.ts
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  createSessionKeyAuth,
  validateSessionKeyAction,
  type SessionKeyAuth,
} from "../../../../core/agents/identity/account.js";
import {
  generateSessionKey,
  validateSessionAction,
  updateSessionUsage,
  shouldDiscardSession,
  isSessionExpired,
  BLOCKS_PER_DAY,
  type SessionKey,
} from "../../../../core/agents/identity/session_keys.js";
import {
  initializeMPCWallet,
  addParty,
  removeParty,
  canSign,
  type MPCConfig,
} from "../../../../core/agents/identity/mpc.js";
import {
  createRecoveryConfig,
  createRecoveryRequest,
  addShardToRecovery,
  addGuardian,
  type Guardian,
  type RecoveryConfig,
} from "../../../../core/agents/identity/recovery.js";
import { generateKeypair } from "../../../../core/crypto/keys.js";
import { sign } from "../../../../core/crypto/signatures.js";
import { publicKeyToDid } from "../../../../core/agents/identity/did.js";

// ============ P1-I1: validateSessionKeyAction recipient checking ============

describe.sequential("validateSessionKeyAction recipient checking", () => {
  let auth: SessionKeyAuth;

  beforeEach(() => {
    const { publicKey: sessionKey } = generateKeypair();
    const { publicKey: operationalKey } = generateKeypair();
    auth = createSessionKeyAuth(
      sessionKey,
      operationalKey,
      {
        canTransfer: true,
        allowedRecipients: ["did:lgm:allowed1", "did:lgm:allowed2"],
      },
      1000,
      100,
    );
    auth.expiresAt = 1000000;
  });

  it("should reject transfer to disallowed recipient", () => {
    const result = validateSessionKeyAction(
      auth,
      "canTransfer",
      50n * 10n ** 15n,
      200,
      "did:lgm:unauthorized",
    );

    expect(result.valid).toBe(false);
    expect(result.reason).toBe("Recipient not in allowed list");
  });

  it("should allow transfer to allowed recipient", () => {
    const result = validateSessionKeyAction(
      auth,
      "canTransfer",
      50n * 10n ** 15n,
      200,
      "did:lgm:allowed1",
    );

    expect(result.valid).toBe(true);
  });

  it("should allow transfer when no recipient specified", () => {
    const result = validateSessionKeyAction(
      auth,
      "canTransfer",
      50n * 10n ** 15n,
      200,
    );

    expect(result.valid).toBe(true);
  });

  it("should allow transfer when allowedRecipients is empty", () => {
    auth.capabilities.allowedRecipients = [];

    const result = validateSessionKeyAction(
      auth,
      "canTransfer",
      50n * 10n ** 15n,
      200,
      "did:lgm:anyone",
    );

    expect(result.valid).toBe(true);
  });

  it("should allow transfer when allowedRecipients is undefined", () => {
    auth.capabilities.allowedRecipients = undefined;

    const result = validateSessionKeyAction(
      auth,
      "canTransfer",
      50n * 10n ** 15n,
      200,
      "did:lgm:anyone",
    );

    expect(result.valid).toBe(true);
  });
});

// ============ P1-I2: createSessionKeyAuth lastResetBlock init ============

describe.sequential("createSessionKeyAuth currentBlock init", () => {
  it("should set authorizedAt and expiresAt from currentBlock", () => {
    const { publicKey: sessionKey } = generateKeypair();
    const { publicKey: operationalKey } = generateKeypair();

    const auth = createSessionKeyAuth(sessionKey, operationalKey, {}, 500, 100);

    expect(auth.authorizedAt).toBe(100);
    expect(auth.expiresAt).toBe(600); // 100 + 500
  });

  it("should set lastResetBlock to currentBlock", () => {
    const { publicKey: sessionKey } = generateKeypair();
    const { publicKey: operationalKey } = generateKeypair();

    const auth = createSessionKeyAuth(sessionKey, operationalKey, {}, 500, 42);

    expect(auth.capabilities.lastResetBlock).toBe(42);
  });

  it("should default currentBlock to 0", () => {
    const { publicKey: sessionKey } = generateKeypair();
    const { publicKey: operationalKey } = generateKeypair();

    const auth = createSessionKeyAuth(sessionKey, operationalKey, {});

    expect(auth.authorizedAt).toBe(0);
    expect(auth.expiresAt).toBe(1000); // 0 + 1000 default
    expect(auth.capabilities.lastResetBlock).toBe(0);
  });
});

// ============ P1-I3: MPC addParty duplicate ID check ============

describe.sequential("MPC addParty duplicate check", () => {
  let config: MPCConfig;

  beforeEach(() => {
    const result = initializeMPCWallet([
      { id: "agent", type: "agent" },
      { id: "protocol", type: "protocol" },
      { id: "operator", type: "operator" },
    ]);
    config = result.config;
  });

  it("should reject duplicate party ID", () => {
    expect(() =>
      addParty(config, {
        id: "agent",
        type: "agent",
        publicShare: generateKeypair().publicKey,
      }),
    ).toThrow("Party already exists");
  });

  it("should allow adding new party", () => {
    const updated = addParty(config, {
      id: "guardian",
      type: "guardian",
      publicShare: generateKeypair().publicKey,
    });

    expect(updated.totalParties).toBe(4);
    expect(updated.parties).toHaveLength(4);
  });
});

// ============ P1-I4: MPC removeParty existence check ============

describe.sequential("MPC removeParty existence check", () => {
  let config: MPCConfig;

  beforeEach(() => {
    const result = initializeMPCWallet([
      { id: "agent", type: "agent" },
      { id: "protocol", type: "protocol" },
      { id: "operator", type: "operator" },
    ]);
    config = result.config;
  });

  it("should reject removing non-existent party", () => {
    expect(() => removeParty(config, "nonexistent")).toThrow("Party not found");
  });

  it("should remove existing party", () => {
    const updated = removeParty(config, "operator");

    expect(updated.totalParties).toBe(2);
    expect(updated.parties.find((p) => p.id === "operator")).toBeUndefined();
  });

  it("should reject removing party below threshold", () => {
    const updated = removeParty(config, "operator"); // 3→2, threshold=2, OK

    expect(() => removeParty(updated, "protocol")).toThrow(
      "Cannot remove party: would go below threshold",
    );
  });
});

// ============ P1-I5: shouldDiscardSession block-height expiry ============

describe.sequential("shouldDiscardSession block-height expiry", () => {
  it("should discard session expired by block height", () => {
    const session = generateSessionKey({}, 3600000);
    session.expiresAtBlock = 1000n;

    expect(shouldDiscardSession(session, 1000n)).toBe(true);
    expect(shouldDiscardSession(session, 1001n)).toBe(true);
  });

  it("should keep session not yet expired by block height", () => {
    const session = generateSessionKey({}, 3600000);
    session.expiresAtBlock = 1000n;

    expect(shouldDiscardSession(session, 999n)).toBe(false);
  });

  it("should fall back to wall-clock when no block height provided", () => {
    const session = generateSessionKey({}, 3600000);
    session.expiresAtBlock = 1000n;

    // No currentBlock provided, session not wall-clock expired
    expect(shouldDiscardSession(session)).toBe(false);
  });

  it("should fall back to wall-clock when session has no block expiry", () => {
    const session = generateSessionKey({}, -1); // Already expired by wall-clock
    session.expiresAt = Date.now() - 1000;

    expect(shouldDiscardSession(session, 0n)).toBe(true);
  });
});

// ============ MPC initializeMPCWallet sync ============

describe.sequential("MPC initializeMPCWallet is synchronous", () => {
  it("should return config directly (not a Promise)", () => {
    const result = initializeMPCWallet([
      { id: "agent", type: "agent" },
      { id: "protocol", type: "protocol" },
    ]);

    // If it were async, result would be a Promise
    expect(result.config).toBeDefined();
    expect(result.shares).toBeDefined();
    expect(result.config.walletId).toBeTruthy();
  });
});

// ============ session_keys.ts block-based daily reset ============

describe.sequential("Block-based daily reset", () => {
  it("should reset daily spending based on block height", () => {
    const session = generateSessionKey(
      {
        allowedActions: ["transfer"],
        maxPerTransaction: 100n * 10n ** 15n,
        maxPerDay: 200n * 10n ** 15n,
        maxTotal: 1000n * 10n ** 15n,
      },
      3600000,
    );
    session.usage.dailySpent = 150n * 10n ** 15n;
    session.usage.lastDailyResetBlock = 0n;

    // Block-based: enough blocks have passed for reset
    const result = validateSessionAction(
      session,
      "transfer",
      60n * 10n ** 15n,
      BLOCKS_PER_DAY + 1n, // More than a day in blocks
    );

    expect(result.allowed).toBe(true);
  });

  it("should NOT reset daily spending when insufficient blocks", () => {
    const session = generateSessionKey(
      {
        allowedActions: ["transfer"],
        maxPerTransaction: 100n * 10n ** 15n,
        maxPerDay: 200n * 10n ** 15n,
        maxTotal: 1000n * 10n ** 15n,
      },
      3600000,
    );
    session.usage.dailySpent = 150n * 10n ** 15n;
    session.usage.lastDailyResetBlock = 0n;

    // Not enough blocks for reset
    const result = validateSessionAction(
      session,
      "transfer",
      60n * 10n ** 15n,
      100n, // Only 100 blocks, not a full day
    );

    expect(result.allowed).toBe(false);
    expect(result.reason).toBe("Exceeds daily limit");
  });

  it("should update lastDailyResetBlock on usage after reset", () => {
    const session = generateSessionKey(
      {
        allowedActions: ["transfer"],
        maxPerTransaction: 100n * 10n ** 15n,
        maxPerDay: 500n * 10n ** 15n,
        maxTotal: 1000n * 10n ** 15n,
      },
      3600000,
    );
    session.usage.dailySpent = 400n * 10n ** 15n;
    session.usage.lastDailyResetBlock = 0n;

    const currentBlock = BLOCKS_PER_DAY + 1n;
    const updated = updateSessionUsage(
      session,
      "transfer",
      50n * 10n ** 15n,
      currentBlock,
    );

    expect(updated.usage.lastDailyResetBlock).toBe(currentBlock);
    expect(updated.usage.dailySpent).toBe(50n * 10n ** 15n);
  });
});

// ============ session_keys.ts recipient restrictions ============

describe.sequential("Session key recipient restrictions", () => {
  it("should reject transfer to disallowed recipient", () => {
    const session = generateSessionKey({
      allowedActions: ["transfer"],
      allowedRecipients: ["did:lgm:allowed1", "did:lgm:allowed2"],
      maxPerTransaction: 100n * 10n ** 15n,
    });

    const result = validateSessionAction(
      session,
      "transfer",
      50n * 10n ** 15n,
      undefined,
      "did:lgm:unauthorized",
    );

    expect(result.allowed).toBe(false);
    expect(result.reason).toBe("Recipient not in allowed list");
  });

  it("should allow transfer to allowed recipient", () => {
    const session = generateSessionKey({
      allowedActions: ["transfer"],
      allowedRecipients: ["did:lgm:allowed1"],
      maxPerTransaction: 100n * 10n ** 15n,
    });

    const result = validateSessionAction(
      session,
      "transfer",
      50n * 10n ** 15n,
      undefined,
      "did:lgm:allowed1",
    );

    expect(result.allowed).toBe(true);
  });

  it("should allow any recipient when allowedRecipients is empty", () => {
    const session = generateSessionKey({
      allowedActions: ["transfer"],
      allowedRecipients: [],
      maxPerTransaction: 100n * 10n ** 15n,
    });

    const result = validateSessionAction(
      session,
      "transfer",
      50n * 10n ** 15n,
      undefined,
      "did:lgm:anyone",
    );

    expect(result.allowed).toBe(true);
  });
});

// ============ isSessionExpired with block height ============

describe.sequential("isSessionExpired block height", () => {
  it("should use block height over wall-clock", () => {
    const session = generateSessionKey({}, 3600000);
    session.expiresAtBlock = 500n;

    // Not expired by wall-clock, but expired by block height
    const result = isSessionExpired(session, 500n);

    expect(result.expired).toBe(true);
    expect(result.reason).toContain("block height");
  });

  it("should warn when near block expiry", () => {
    const session = generateSessionKey({}, 3600000);
    session.expiresAtBlock = 500n;

    // 50 blocks remaining (< 100 threshold)
    const result = isSessionExpired(session, 450n);

    expect(result.expired).toBe(false);
    expect(result.nearExpiry).toBe(true);
  });

  it("should not warn when far from block expiry", () => {
    const session = generateSessionKey({}, 3600000);
    session.expiresAtBlock = 500n;

    const result = isSessionExpired(session, 100n);

    expect(result.expired).toBe(false);
    expect(result.nearExpiry).toBeUndefined();
  });
});

// ============ BLOCKS_PER_DAY constant ============

describe.sequential("BLOCKS_PER_DAY", () => {
  it("should be approximately 86400s / 2.521s", () => {
    // 86400000ms / 2521ms = ~34272
    expect(BLOCKS_PER_DAY).toBe(34272n);
  });
});

// ============ addShardToRecovery with config (holder identity verification) ============

describe.sequential("addShardToRecovery holder identity verification", () => {
  it("should accept shard from authorized holder", () => {
    const holderKey = generateKeypair();
    const holderDid = publicKeyToDid(holderKey.publicKey);
    const recoveryKey = generateKeypair();

    const holders = [
      {
        type: "agent" as const,
        identifier: holderDid,
        canInitiate: true,
      },
      {
        type: "operator" as const,
        identifier: "did:lgm:operator",
        canInitiate: true,
      },
    ];
    const { config, shards } = createRecoveryConfig(
      recoveryKey.publicKey,
      holders,
    );

    const newOpKey = generateKeypair();
    const request = createRecoveryRequest(
      "did:lgm:target",
      holderDid,
      newOpKey.publicKey,
      config,
      1000,
    );

    const shard = shards.get(1)!;
    const message = `recovery:${request.id}:shard:${shard.index}`;
    const sig = sign(new TextEncoder().encode(message), holderKey.privateKey);

    const updated = addShardToRecovery(
      request,
      shard,
      sig,
      holderKey.publicKey,
      config,
    );

    expect(updated.shardsProvided).toContain(1);
  });

  it("should reject shard from unauthorized holder", () => {
    const holderKey = generateKeypair();
    const unauthorizedKey = generateKeypair();
    const recoveryKey = generateKeypair();

    const holders = [
      {
        type: "agent" as const,
        identifier: publicKeyToDid(holderKey.publicKey),
        canInitiate: true,
      },
      {
        type: "operator" as const,
        identifier: "did:lgm:operator",
        canInitiate: true,
      },
    ];
    const { config, shards } = createRecoveryConfig(
      recoveryKey.publicKey,
      holders,
    );

    const newOpKey = generateKeypair();
    const request = createRecoveryRequest(
      "did:lgm:target",
      "did:lgm:initiator",
      newOpKey.publicKey,
      config,
      1000,
    );

    const shard = shards.get(1)!;
    const message = `recovery:${request.id}:shard:${shard.index}`;
    // Sign with unauthorized key — signature would be valid for unauthorizedKey,
    // but identity check happens first
    const sig = sign(
      new TextEncoder().encode(message),
      unauthorizedKey.privateKey,
    );

    expect(() =>
      addShardToRecovery(
        request,
        shard,
        sig,
        unauthorizedKey.publicKey,
        config,
      ),
    ).toThrow("Shard holder identity mismatch");
  });

  it("should reject shard with unassigned index", () => {
    const holderKey = generateKeypair();
    const recoveryKey = generateKeypair();

    const holders = [
      {
        type: "agent" as const,
        identifier: publicKeyToDid(holderKey.publicKey),
        canInitiate: true,
      },
      {
        type: "operator" as const,
        identifier: "did:lgm:operator",
        canInitiate: true,
      },
    ];
    const { config, shards } = createRecoveryConfig(
      recoveryKey.publicKey,
      holders,
    );

    const newOpKey = generateKeypair();
    const request = createRecoveryRequest(
      "did:lgm:target",
      "did:lgm:initiator",
      newOpKey.publicKey,
      config,
      1000,
    );

    // Create a fake shard with index 99 (not assigned to any holder)
    const fakeShard = {
      index: 99,
      value: new Uint8Array(32),
      holderType: "agent" as const,
    };
    const message = `recovery:${request.id}:shard:${fakeShard.index}`;
    const sig = sign(new TextEncoder().encode(message), holderKey.privateKey);

    expect(() =>
      addShardToRecovery(request, fakeShard, sig, holderKey.publicKey, config),
    ).toThrow("Shard index not assigned to any holder");
  });

  it("should still work without config (backwards compatible)", () => {
    const holderKey = generateKeypair();
    const recoveryKey = generateKeypair();

    const holders = [
      {
        type: "agent" as const,
        identifier: "did:lgm:agent",
        canInitiate: true,
      },
      {
        type: "operator" as const,
        identifier: "did:lgm:operator",
        canInitiate: true,
      },
    ];
    const { config, shards } = createRecoveryConfig(
      recoveryKey.publicKey,
      holders,
    );

    const newOpKey = generateKeypair();
    const request = createRecoveryRequest(
      "did:lgm:target",
      "did:lgm:initiator",
      newOpKey.publicKey,
      config,
      1000,
    );

    const shard = shards.get(1)!;
    const message = `recovery:${request.id}:shard:${shard.index}`;
    const sig = sign(new TextEncoder().encode(message), holderKey.privateKey);

    // No config parameter — skips identity verification
    const updated = addShardToRecovery(
      request,
      shard,
      sig,
      holderKey.publicKey,
    );

    expect(updated.shardsProvided).toContain(1);
  });
});

// ============ addGuardian duplicate rejection ============

describe.sequential("addGuardian duplicate rejection", () => {
  let config: RecoveryConfig;

  beforeEach(() => {
    const recoveryKey = generateKeypair();
    const holders = [
      {
        type: "agent" as const,
        identifier: "did:lgm:agent",
        canInitiate: true,
      },
      {
        type: "operator" as const,
        identifier: "did:lgm:operator",
        canInitiate: true,
      },
    ];
    ({ config } = createRecoveryConfig(recoveryKey.publicKey, holders));
  });

  it("should reject duplicate guardian DID", () => {
    const guardian: Guardian = {
      did: "did:lgm:guardian1",
      publicKey: generateKeypair().publicKey,
      addedAt: 1000,
      lastActive: 1000,
      trustScore: 0.8,
    };

    config = addGuardian(config, guardian);

    const duplicate: Guardian = {
      did: "did:lgm:guardian1", // Same DID
      publicKey: generateKeypair().publicKey, // Different key
      addedAt: 2000,
      lastActive: 2000,
      trustScore: 0.9,
    };

    expect(() => addGuardian(config, duplicate)).toThrow(
      "Guardian already exists",
    );
  });

  it("should allow different guardian DIDs", () => {
    const guardian1: Guardian = {
      did: "did:lgm:guardian1",
      publicKey: generateKeypair().publicKey,
      addedAt: 1000,
      lastActive: 1000,
      trustScore: 0.8,
    };
    const guardian2: Guardian = {
      did: "did:lgm:guardian2",
      publicKey: generateKeypair().publicKey,
      addedAt: 1000,
      lastActive: 1000,
      trustScore: 0.8,
    };

    config = addGuardian(config, guardian1);
    config = addGuardian(config, guardian2);

    expect(config.socialRecovery!.guardians).toHaveLength(2);
  });

  it("should not mutate original config", () => {
    const guardian: Guardian = {
      did: "did:lgm:guardian1",
      publicKey: generateKeypair().publicKey,
      addedAt: 1000,
      lastActive: 1000,
      trustScore: 0.8,
    };

    const original = { ...config };
    const updated = addGuardian(config, guardian);

    // Original should not have socialRecovery
    expect(config.socialRecovery).toEqual(original.socialRecovery);
    expect(updated.socialRecovery).toBeDefined();
  });
});
