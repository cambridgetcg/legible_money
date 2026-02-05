/**
 * Identity RPC Handler Tests
 *
 * Tests for key rotation and key info query endpoints.
 * Covers: signature verification, nonce checks, frozen/recovery guards,
 * ownership verification, successful rotation, and key info retrieval.
 */

import { describe, it, expect } from "vitest";
import {
  lgmRotateKey,
  lgmGetKeyInfo,
} from "../../../../protocol/rpc/handlers/identity.js";
import { StateManager } from "../../../../state/state_manager.js";
import { generateKeypair } from "../../../../core/crypto/keys.js";
import { sign } from "../../../../core/crypto/signatures.js";
import { sha512_256Hex } from "../../../../core/crypto/hashing.js";

// ============ Test Helpers ============

function createTestContext(
  accounts?: Array<{
    did: string;
    keyHash: string;
    balance?: bigint;
    frozen?: boolean;
    inRecovery?: boolean;
  }>,
) {
  const state = new StateManager();

  for (const a of accounts ?? []) {
    state.createAccount(a.did, a.keyHash, a.balance ?? 0n);
    if (a.frozen || a.inRecovery) {
      // Update account flags directly via the accounts tree
      const existing = state.getAccount(a.did);
      if (existing) {
        (state as any).accounts.updateAccount(a.did, {
          flags: {
            ...existing.flags,
            frozen: a.frozen ?? false,
            inRecovery: a.inRecovery ?? false,
          },
        });
      }
    }
  }

  return {
    state,
    consensus: {} as any,
    stateIntegration: {} as any,
    nodeVersion: "0.1.0",
    networkId: "test",
  };
}

function buildRotateKeyPayload(did: string, newKeyHash: string, nonce: string) {
  return JSON.stringify({
    did,
    newOperationalKeyHash: newKeyHash,
    action: "rotate_key",
    nonce,
  });
}

// ============ lgm_rotateKey Tests ============

describe("lgm_rotateKey", () => {
  it("should rotate key successfully", async () => {
    const keypair = generateKeypair();
    const keyHash = sha512_256Hex(keypair.publicKey);
    const did = "did:lgm:test-rotate";

    const newKeypair = generateKeypair();
    const newKeyHash = sha512_256Hex(newKeypair.publicKey);

    const ctx = createTestContext([{ did, keyHash, balance: 1000n }]);

    const payload = buildRotateKeyPayload(did, newKeyHash, "0");
    const signature = sign(
      new TextEncoder().encode(payload),
      keypair.privateKey,
    );

    const result = await lgmRotateKey(
      {
        did,
        newOperationalKeyHash: newKeyHash,
        nonce: "0",
        signature,
        sender: keypair.publicKey,
      },
      ctx,
    );

    expect(result.success).toBe(true);
    expect(result.txHash).toBeTruthy();
    expect(result.newKeyVersion).toBe(2); // 1 (initial) + 1 rotation

    // Verify state was updated
    const account = ctx.state.getAccount(did);
    expect(account?.operationalKeyHash).toBe(newKeyHash);
    expect(account?.operationalKeyVersion).toBe(2);
  });

  it("should reject invalid signature", async () => {
    const keypair = generateKeypair();
    const keyHash = sha512_256Hex(keypair.publicKey);
    const did = "did:lgm:test-badsig";
    const newKeyHash = "a".repeat(64);

    const ctx = createTestContext([{ did, keyHash }]);

    await expect(
      lgmRotateKey(
        {
          did,
          newOperationalKeyHash: newKeyHash,
          nonce: "0",
          signature: "invalid-signature",
          sender: keypair.publicKey,
        },
        ctx,
      ),
    ).rejects.toThrow("Invalid signature");
  });

  it("should reject when account not found", async () => {
    const keypair = generateKeypair();
    const newKeyHash = "b".repeat(64);
    const did = "did:lgm:nonexistent";

    const ctx = createTestContext([]);

    const payload = buildRotateKeyPayload(did, newKeyHash, "0");
    const signature = sign(
      new TextEncoder().encode(payload),
      keypair.privateKey,
    );

    await expect(
      lgmRotateKey(
        {
          did,
          newOperationalKeyHash: newKeyHash,
          nonce: "0",
          signature,
          sender: keypair.publicKey,
        },
        ctx,
      ),
    ).rejects.toThrow("Account not found");
  });

  it("should reject when sender does not own account", async () => {
    const ownerKeypair = generateKeypair();
    const attackerKeypair = generateKeypair();
    const ownerKeyHash = sha512_256Hex(ownerKeypair.publicKey);
    const did = "did:lgm:test-notowner";
    const newKeyHash = "c".repeat(64);

    const ctx = createTestContext([{ did, keyHash: ownerKeyHash }]);

    // Attacker signs with their own key
    const payload = buildRotateKeyPayload(did, newKeyHash, "0");
    const signature = sign(
      new TextEncoder().encode(payload),
      attackerKeypair.privateKey,
    );

    await expect(
      lgmRotateKey(
        {
          did,
          newOperationalKeyHash: newKeyHash,
          nonce: "0",
          signature,
          sender: attackerKeypair.publicKey,
        },
        ctx,
      ),
    ).rejects.toThrow("does not own");
  });

  it("should reject when nonce does not match", async () => {
    const keypair = generateKeypair();
    const keyHash = sha512_256Hex(keypair.publicKey);
    const did = "did:lgm:test-badnonce";
    const newKeyHash = "d".repeat(64);

    const ctx = createTestContext([{ did, keyHash }]);

    const payload = buildRotateKeyPayload(did, newKeyHash, "999");
    const signature = sign(
      new TextEncoder().encode(payload),
      keypair.privateKey,
    );

    await expect(
      lgmRotateKey(
        {
          did,
          newOperationalKeyHash: newKeyHash,
          nonce: "999",
          signature,
          sender: keypair.publicKey,
        },
        ctx,
      ),
    ).rejects.toThrow("Nonce mismatch");
  });

  it("should reject when account is frozen", async () => {
    const keypair = generateKeypair();
    const keyHash = sha512_256Hex(keypair.publicKey);
    const did = "did:lgm:test-frozen";
    const newKeyHash = "e".repeat(64);

    const ctx = createTestContext([{ did, keyHash, frozen: true }]);

    const payload = buildRotateKeyPayload(did, newKeyHash, "0");
    const signature = sign(
      new TextEncoder().encode(payload),
      keypair.privateKey,
    );

    await expect(
      lgmRotateKey(
        {
          did,
          newOperationalKeyHash: newKeyHash,
          nonce: "0",
          signature,
          sender: keypair.publicKey,
        },
        ctx,
      ),
    ).rejects.toThrow("frozen");
  });

  it("should reject when account is in recovery", async () => {
    const keypair = generateKeypair();
    const keyHash = sha512_256Hex(keypair.publicKey);
    const did = "did:lgm:test-recovery";
    const newKeyHash = "f".repeat(64);

    const ctx = createTestContext([{ did, keyHash, inRecovery: true }]);

    const payload = buildRotateKeyPayload(did, newKeyHash, "0");
    const signature = sign(
      new TextEncoder().encode(payload),
      keypair.privateKey,
    );

    await expect(
      lgmRotateKey(
        {
          did,
          newOperationalKeyHash: newKeyHash,
          nonce: "0",
          signature,
          sender: keypair.publicKey,
        },
        ctx,
      ),
    ).rejects.toThrow("recovery");
  });

  it("should reject invalid newOperationalKeyHash format", async () => {
    const keypair = generateKeypair();
    const keyHash = sha512_256Hex(keypair.publicKey);
    const did = "did:lgm:test-badhash";

    const ctx = createTestContext([{ did, keyHash }]);

    const payload = buildRotateKeyPayload(did, "not-a-hex-hash", "0");
    const signature = sign(
      new TextEncoder().encode(payload),
      keypair.privateKey,
    );

    await expect(
      lgmRotateKey(
        {
          did,
          newOperationalKeyHash: "not-a-hex-hash",
          nonce: "0",
          signature,
          sender: keypair.publicKey,
        },
        ctx,
      ),
    ).rejects.toThrow("64-character hex");
  });
});

// ============ lgm_getKeyInfo Tests ============

describe("lgm_getKeyInfo", () => {
  it("should return key info for existing account", async () => {
    const keyHash = "a".repeat(64);
    const did = "did:lgm:test-keyinfo";

    const ctx = createTestContext([{ did, keyHash }]);

    const result = await lgmGetKeyInfo({ did }, ctx);

    expect(result.did).toBe(did);
    expect(result.operationalKeyHash).toBe(keyHash);
    expect(result.operationalKeyVersion).toBe(1); // Initial version from createAccount
    expect(result.isFrozen).toBe(false);
    expect(result.isInRecovery).toBe(false);
  });

  it("should reflect frozen status", async () => {
    const keyHash = "b".repeat(64);
    const did = "did:lgm:test-frozen-info";

    const ctx = createTestContext([{ did, keyHash, frozen: true }]);

    const result = await lgmGetKeyInfo({ did }, ctx);
    expect(result.isFrozen).toBe(true);
  });

  it("should reject for nonexistent account", async () => {
    const ctx = createTestContext([]);

    await expect(
      lgmGetKeyInfo(
        {
          did: "did:lgm:nobody",
        },
        ctx,
      ),
    ).rejects.toThrow("Account not found");
  });

  it("should reject invalid DID format", async () => {
    const ctx = createTestContext([]);

    await expect(
      lgmGetKeyInfo(
        {
          did: "not-a-did",
        },
        ctx,
      ),
    ).rejects.toThrow("valid DID");
  });
});
