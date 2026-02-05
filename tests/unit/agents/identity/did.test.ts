/**
 * DID (Decentralized Identifier) Unit Tests
 *
 * Tests the W3C-compliant DID generation and management for AI agents.
 * DID format: did:lgm:[network]:[44-char-hex-address]
 *
 * PP HIERARCHY:
 *   TRUTH: Verify DID format and cryptographic correctness
 *   UNDERSTANDING: Test all DID operations
 *   BEAUTY: Clear test structure matching module organization
 *   JUSTICE: Tests in proper unit test location
 */

import { describe, it, expect } from "vitest";
import {
  DIDSchema,
  CapabilitySchema,
  createDID,
  didToAddress,
  didToNetwork,
  createDIDDocument,
  type DID,
  type AgentIdentity,
} from "../../../../core/agents/identity/did.js";
import { generateKeypair } from "../../../../core/crypto/keys.js";

// ============ DID Schema Validation ============

describe.sequential("DID Schema Validation", () => {
  it("should validate correct mainnet DID format", () => {
    const validDID = "did:lgm:mainnet:" + "a".repeat(44);
    const result = DIDSchema.safeParse(validDID);
    expect(result.success).toBe(true);
  });

  it("should validate correct testnet DID format", () => {
    const validDID = "did:lgm:testnet:" + "b".repeat(44);
    const result = DIDSchema.safeParse(validDID);
    expect(result.success).toBe(true);
  });

  it("should validate correct devnet DID format", () => {
    const validDID = "did:lgm:devnet:" + "c".repeat(44);
    const result = DIDSchema.safeParse(validDID);
    expect(result.success).toBe(true);
  });

  it("should reject DID with wrong method", () => {
    const invalidDID = "did:eth:mainnet:" + "a".repeat(44);
    const result = DIDSchema.safeParse(invalidDID);
    expect(result.success).toBe(false);
  });

  it("should reject DID with wrong network", () => {
    const invalidDID = "did:lgm:localnet:" + "a".repeat(44);
    const result = DIDSchema.safeParse(invalidDID);
    expect(result.success).toBe(false);
  });

  it("should reject DID with wrong 4-part address length", () => {
    const invalidDID = "did:lgm:mainnet:" + "a".repeat(32); // Wrong length for 4-part format
    const result = DIDSchema.safeParse(invalidDID);
    expect(result.success).toBe(false);
  });

  it("should accept 3-part compact DID format", () => {
    const compactDID = "did:lgm:" + "a".repeat(32);
    const result = DIDSchema.safeParse(compactDID);
    expect(result.success).toBe(true);
  });

  it("should reject DID with non-hex characters", () => {
    const invalidDID = "did:lgm:mainnet:" + "g".repeat(44); // 'g' is not hex
    const result = DIDSchema.safeParse(invalidDID);
    expect(result.success).toBe(false);
  });

  it("should reject empty string", () => {
    const result = DIDSchema.safeParse("");
    expect(result.success).toBe(false);
  });

  it("should reject malformed DID", () => {
    const result = DIDSchema.safeParse("not-a-did");
    expect(result.success).toBe(false);
  });
});

// ============ Capability Schema Validation ============

describe.sequential("Capability Schema Validation", () => {
  it("should validate all standard capabilities", () => {
    const capabilities = [
      "compute.inference",
      "compute.training",
      "data.storage",
      "data.retrieval",
      "service.api",
      "service.task",
      "coordination.validator",
      "coordination.oracle",
    ];

    for (const cap of capabilities) {
      const result = CapabilitySchema.safeParse(cap);
      expect(result.success).toBe(true);
    }
  });

  it("should reject unknown capability", () => {
    const result = CapabilitySchema.safeParse("unknown");
    expect(result.success).toBe(false);
  });

  it("should reject partial capability names", () => {
    const result = CapabilitySchema.safeParse("compute");
    expect(result.success).toBe(false);
  });
});

// ============ DID Creation ============

describe.sequential("DID Creation", () => {
  it("should create valid DID from public key", () => {
    const { publicKey } = generateKeypair();
    const did = createDID(publicKey, "mainnet");

    expect(did).toMatch(/^did:lgm:mainnet:[a-f0-9]{44}$/);
  });

  it("should create different DIDs for different keys", () => {
    const { publicKey: key1 } = generateKeypair();
    const { publicKey: key2 } = generateKeypair();

    const did1 = createDID(key1, "mainnet");
    const did2 = createDID(key2, "mainnet");

    expect(did1).not.toBe(did2);
  });

  it("should create same DID for same key", () => {
    const { publicKey } = generateKeypair();

    const did1 = createDID(publicKey, "mainnet");
    const did2 = createDID(publicKey, "mainnet");

    expect(did1).toBe(did2);
  });

  it("should create different DIDs for different networks", () => {
    const { publicKey } = generateKeypair();

    const mainnetDID = createDID(publicKey, "mainnet");
    const testnetDID = createDID(publicKey, "testnet");
    const devnetDID = createDID(publicKey, "devnet");

    // DIDs differ only by network component
    expect(mainnetDID).toContain(":mainnet:");
    expect(testnetDID).toContain(":testnet:");
    expect(devnetDID).toContain(":devnet:");

    // Address part should be the same
    const mainnetAddr = didToAddress(mainnetDID);
    const testnetAddr = didToAddress(testnetDID);
    expect(mainnetAddr).toBe(testnetAddr);
  });

  it("should use first 44 characters of public key hex", () => {
    const { publicKey } = generateKeypair();
    const did = createDID(publicKey, "mainnet");
    const address = didToAddress(did);

    // Address should be first 44 chars of public key
    expect(address).toBe(publicKey.slice(0, 44));
  });
});

// ============ DID Parsing ============

describe.sequential("DID Parsing", () => {
  it("should extract address from DID", () => {
    const address = "a".repeat(44);
    const did = `did:lgm:mainnet:${address}` as DID;

    expect(didToAddress(did)).toBe(address);
  });

  it("should extract network from DID", () => {
    const address = "a".repeat(44);

    expect(didToNetwork(`did:lgm:mainnet:${address}` as DID)).toBe("mainnet");
    expect(didToNetwork(`did:lgm:testnet:${address}` as DID)).toBe("testnet");
    expect(didToNetwork(`did:lgm:devnet:${address}` as DID)).toBe("devnet");
  });

  it("should handle DIDs with lowercase hex", () => {
    const address = "abcdef0123456789".repeat(3).slice(0, 44);
    const did = `did:lgm:mainnet:${address}` as DID;

    expect(didToAddress(did)).toBe(address);
  });
});

// ============ DID Document Creation ============

describe.sequential("DID Document Creation", () => {
  it("should create valid W3C DID Document", () => {
    const { publicKey } = generateKeypair();
    const did = createDID(publicKey, "mainnet");

    const identity: AgentIdentity = {
      did,
      publicKey,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      capabilities: ["compute.inference", "coordination.validator"],
      metadata: {
        name: "Test Agent",
      },
    };

    const document = createDIDDocument(identity);

    expect(document["@context"]).toBe("https://www.w3.org/ns/did/v1");
    expect(document.id).toBe(did);
  });

  it("should include verification method with Ed25519", () => {
    const { publicKey } = generateKeypair();
    const did = createDID(publicKey, "mainnet");

    const identity: AgentIdentity = {
      did,
      publicKey,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      capabilities: [],
      metadata: {},
    };

    const document = createDIDDocument(identity);

    expect(document.verificationMethod).toHaveLength(1);
    expect(document.verificationMethod[0].type).toBe(
      "Ed25519VerificationKey2020",
    );
    expect(document.verificationMethod[0].controller).toBe(did);
  });

  it("should include multibase-encoded public key", () => {
    const { publicKey } = generateKeypair();
    const did = createDID(publicKey, "mainnet");

    const identity: AgentIdentity = {
      did,
      publicKey,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      capabilities: [],
      metadata: {},
    };

    const document = createDIDDocument(identity);

    // Multibase z-prefix for base58btc (actually just 'z' + hex in this implementation)
    expect(document.verificationMethod[0].publicKeyMultibase).toMatch(/^z/);
    expect(document.verificationMethod[0].publicKeyMultibase).toBe(
      "z" + publicKey,
    );
  });

  it("should include service endpoint when provided", () => {
    const { publicKey } = generateKeypair();
    const did = createDID(publicKey, "mainnet");

    const identity: AgentIdentity = {
      did,
      publicKey,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      capabilities: ["compute.inference"],
      metadata: {
        endpoint: "https://agent.example.com/api",
      },
    };

    const document = createDIDDocument(identity);

    expect(document.service).toBeDefined();
    expect(document.service).toHaveLength(1);
    expect(document.service![0].type).toBe("AgentEndpoint");
    expect(document.service![0].serviceEndpoint).toBe(
      "https://agent.example.com/api",
    );
  });

  it("should not include service when no endpoint", () => {
    const { publicKey } = generateKeypair();
    const did = createDID(publicKey, "mainnet");

    const identity: AgentIdentity = {
      did,
      publicKey,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      capabilities: ["compute.inference"],
      metadata: {},
    };

    const document = createDIDDocument(identity);

    expect(document.service).toBeUndefined();
  });

  it("should set authentication to verification method", () => {
    const { publicKey } = generateKeypair();
    const did = createDID(publicKey, "mainnet");

    const identity: AgentIdentity = {
      did,
      publicKey,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      capabilities: [],
      metadata: {},
    };

    const document = createDIDDocument(identity);

    expect(document.authentication).toContain(`${did}#key-1`);
  });
});

// ============ Edge Cases ============

describe.sequential("DID Edge Cases", () => {
  it("should handle public key with leading zeros", () => {
    // Create a key that might have leading zeros in hex
    const { publicKey } = generateKeypair();
    const did = createDID(publicKey, "mainnet");

    // Should still be valid
    const result = DIDSchema.safeParse(did);
    expect(result.success).toBe(true);
  });

  it("should create document with empty capabilities", () => {
    const { publicKey } = generateKeypair();
    const did = createDID(publicKey, "mainnet");

    const identity: AgentIdentity = {
      did,
      publicKey,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      capabilities: [],
      metadata: {},
    };

    const document = createDIDDocument(identity);

    // No service when no endpoint provided
    expect(document.service).toBeUndefined();
  });

  it("should create document with all capabilities and endpoint", () => {
    const { publicKey } = generateKeypair();
    const did = createDID(publicKey, "mainnet");

    const allCapabilities: Array<
      | "compute.inference"
      | "compute.training"
      | "data.storage"
      | "data.retrieval"
      | "service.api"
      | "service.task"
      | "coordination.validator"
      | "coordination.oracle"
    > = [
      "compute.inference",
      "compute.training",
      "data.storage",
      "data.retrieval",
      "service.api",
      "service.task",
      "coordination.validator",
      "coordination.oracle",
    ];

    const identity: AgentIdentity = {
      did,
      publicKey,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      capabilities: allCapabilities,
      metadata: {
        name: "Full Capability Agent",
        endpoint: "https://agent.example.com",
      },
    };

    const document = createDIDDocument(identity);

    // Should have one service endpoint
    expect(document.service).toHaveLength(1);
    expect(document.service![0].serviceEndpoint).toBe(
      "https://agent.example.com",
    );
  });

  it("should handle metadata with all optional fields", () => {
    const { publicKey } = generateKeypair();
    const did = createDID(publicKey, "mainnet");

    const identity: AgentIdentity = {
      did,
      publicKey,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      capabilities: ["compute.inference"],
      metadata: {
        name: "Test Agent",
        description: "A test agent for unit testing",
        endpoint: "https://test.example.com",
        version: "1.0.0",
      },
    };

    const document = createDIDDocument(identity);

    expect(document.id).toBe(did);
    expect(document.service).toBeDefined();
  });
});
