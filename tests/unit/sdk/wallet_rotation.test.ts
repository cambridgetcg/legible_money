/**
 * Wallet Key Rotation SDK Tests
 *
 * Tests for AgentWallet.buildRotateKeyRequest().
 * Verifies that the method produces valid signed rotation payloads.
 */

import { describe, it, expect } from "vitest";
import { AgentWallet } from "../../../protocol/sdk/wallet.js";
import { verify } from "../../../core/crypto/signatures.js";
import { sha512_256Hex } from "../../../core/crypto/hashing.js";

describe("AgentWallet.buildRotateKeyRequest", () => {
  it("should produce a valid signed rotation request", async () => {
    const wallet = AgentWallet.fromAgentId(
      "test-agent",
      "test-secret",
      "testnet",
    );
    const nonce = 0n;

    const bundle = await wallet.buildRotateKeyRequest(nonce);

    // Request should have all required fields
    expect(bundle.request.did).toBe(wallet.did);
    expect(bundle.request.nonce).toBe("0");
    expect(bundle.request.sender).toBe(wallet.publicKey);
    expect(bundle.request.newOperationalKeyHash).toMatch(/^[a-f0-9]{64}$/);
    expect(bundle.request.signature).toBeTruthy();

    // New key hash should be hash of new public key
    expect(bundle.request.newOperationalKeyHash).toBe(
      sha512_256Hex(bundle.newPublicKey),
    );

    // Signature should be verifiable with the current wallet's public key
    const payload = JSON.stringify({
      did: wallet.did,
      newOperationalKeyHash: bundle.request.newOperationalKeyHash,
      action: "rotate_key",
      nonce: "0",
    });
    const valid = verify(
      new TextEncoder().encode(payload),
      bundle.request.signature,
      wallet.publicKey,
    );
    expect(valid).toBe(true);
  });

  it("should return a new wallet with different keys", async () => {
    const wallet = AgentWallet.fromAgentId(
      "test-agent-2",
      "test-secret-2",
      "testnet",
    );

    const bundle = await wallet.buildRotateKeyRequest(0n);

    // New wallet should have different keys
    expect(bundle.newWallet.publicKey).not.toBe(wallet.publicKey);
    // But same DID (DID is immutable, only operational key rotates)
    expect(bundle.newWallet.did).toBe(wallet.did);
  });

  it("should produce different keys on each call", async () => {
    const wallet = AgentWallet.fromAgentId(
      "test-agent-3",
      "test-secret-3",
      "testnet",
    );

    const bundle1 = await wallet.buildRotateKeyRequest(0n);
    const bundle2 = await wallet.buildRotateKeyRequest(1n);

    // Each rotation should generate a unique key
    expect(bundle1.newPublicKey).not.toBe(bundle2.newPublicKey);
    expect(bundle1.request.newOperationalKeyHash).not.toBe(
      bundle2.request.newOperationalKeyHash,
    );
  });

  it("should use the correct nonce in payload", async () => {
    const wallet = AgentWallet.fromAgentId(
      "test-agent-4",
      "test-secret-4",
      "testnet",
    );

    const bundle = await wallet.buildRotateKeyRequest(42n);
    expect(bundle.request.nonce).toBe("42");
  });

  it("new wallet should be able to sign messages", async () => {
    const wallet = AgentWallet.fromAgentId(
      "test-agent-5",
      "test-secret-5",
      "testnet",
    );
    const bundle = await wallet.buildRotateKeyRequest(0n);

    const message = new TextEncoder().encode("test message");
    const sig = await bundle.newWallet.sign(message);

    // Verify with the new wallet's public key
    const valid = AgentWallet.verify(message, sig, bundle.newWallet.publicKey);
    expect(valid).toBe(true);
  });
});
