/**
 * Genesis Core Tests: DID (Decentralized Identifiers)
 *
 * Comprehensive tests for agents/identity/did.ts
 *
 * PP ALIGNMENT:
 *   TRUTH: Verified against W3C DID spec patterns
 *   UNDERSTANDING: Tests demonstrate what each function does
 *   BEAUTY: Complete coverage with clear organization
 *   JUSTICE: Tests in genesis/tests/ where they belong
 *   CREATIVITY: Building verification path for Genesis Core
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
  type Capability,
  type AgentIdentity,
  type DIDDocument,
} from "../../../core/agents/identity/did.js";
import { generateKeypair } from "../../../core/crypto/keys.js";

// ============ Test Constants ============

const VALID_PUBLIC_KEY =
  "3b6a27bcceb6a42d62a3a8d02a6f0d73653215771de243a63ac048a18b59da29";
const VALID_ADDRESS = "3b6a27bcceb6a42d62a3a8d02a6f0d73653215771de2";

const VALID_DIDS = {
  mainnet: `did:lgm:mainnet:${VALID_ADDRESS}`,
  testnet: `did:lgm:testnet:${VALID_ADDRESS}`,
  devnet: `did:lgm:devnet:${VALID_ADDRESS}`,
};

// ============ DID Schema Tests ============

describe("DIDSchema", () => {
  describe("valid DIDs", () => {
    it("should accept mainnet DID", () => {
      const result = DIDSchema.safeParse(VALID_DIDS.mainnet);
      expect(result.success).toBe(true);
    });

    it("should accept testnet DID", () => {
      const result = DIDSchema.safeParse(VALID_DIDS.testnet);
      expect(result.success).toBe(true);
    });

    it("should accept devnet DID", () => {
      const result = DIDSchema.safeParse(VALID_DIDS.devnet);
      expect(result.success).toBe(true);
    });

    it("should accept all lowercase hex", () => {
      // DID address must be exactly 44 hex characters (22 bytes)
      const did =
        "did:lgm:mainnet:abcdef0123456789abcdef0123456789abcdef0123aa";
      const result = DIDSchema.safeParse(did);
      expect(result.success).toBe(true);
    });
  });

  describe("invalid DIDs", () => {
    it("should reject wrong prefix", () => {
      const did = "did:eth:mainnet:" + VALID_ADDRESS;
      const result = DIDSchema.safeParse(did);
      expect(result.success).toBe(false);
    });

    it("should reject unknown network", () => {
      const did = "did:lgm:localnet:" + VALID_ADDRESS;
      const result = DIDSchema.safeParse(did);
      expect(result.success).toBe(false);
    });

    it("should reject uppercase hex", () => {
      const did = "did:lgm:mainnet:" + VALID_ADDRESS.toUpperCase();
      const result = DIDSchema.safeParse(did);
      expect(result.success).toBe(false);
    });

    it("should reject short 4-part address", () => {
      const did = "did:lgm:mainnet:" + VALID_ADDRESS.slice(0, 40);
      const result = DIDSchema.safeParse(did);
      expect(result.success).toBe(false);
    });

    it("should reject long address", () => {
      const did = "did:lgm:mainnet:" + VALID_ADDRESS + "abcd";
      const result = DIDSchema.safeParse(did);
      expect(result.success).toBe(false);
    });

    it("should reject non-hex characters", () => {
      const did = "did:lgm:mainnet:ghijklmnopqrstuvwxyz01234567890123456789ab";
      const result = DIDSchema.safeParse(did);
      expect(result.success).toBe(false);
    });

    it("should reject missing parts", () => {
      // 'did:lgm:mainnet' has 3 parts but 'mainnet' is not 32-char hex
      expect(DIDSchema.safeParse("did:lgm:mainnet").success).toBe(false);
      expect(DIDSchema.safeParse("did:lgm").success).toBe(false);
      expect(DIDSchema.safeParse("did").success).toBe(false);
      expect(DIDSchema.safeParse("").success).toBe(false);
    });

    it("should reject non-string input", () => {
      expect(DIDSchema.safeParse(null).success).toBe(false);
      expect(DIDSchema.safeParse(undefined).success).toBe(false);
      expect(DIDSchema.safeParse(123).success).toBe(false);
      expect(DIDSchema.safeParse({}).success).toBe(false);
    });
  });

  describe("DID format specification", () => {
    it("should match 4-part format: did:lgm:<network>:<44-char-hex>", () => {
      const regex = /^did:lgm:(mainnet|testnet|devnet):[a-f0-9]{44}$/;

      expect(VALID_DIDS.mainnet).toMatch(regex);
      expect(VALID_DIDS.testnet).toMatch(regex);
      expect(VALID_DIDS.devnet).toMatch(regex);
    });

    it("should have exactly 4 colon-separated parts for network DIDs", () => {
      const parts = VALID_DIDS.mainnet.split(":");
      expect(parts.length).toBe(4);
      expect(parts[0]).toBe("did");
      expect(parts[1]).toBe("lgm");
      expect(parts[2]).toBe("mainnet");
      expect(parts[3]).toHaveLength(44);
    });

    it("should accept 3-part compact format: did:lgm:<32-char-hex>", () => {
      const compactDid = "did:lgm:" + "a".repeat(32);
      const result = DIDSchema.safeParse(compactDid);
      expect(result.success).toBe(true);
    });

    it("should reject 3-part format with wrong length", () => {
      const shortDid = "did:lgm:" + "a".repeat(16);
      expect(DIDSchema.safeParse(shortDid).success).toBe(false);

      const longDid = "did:lgm:" + "a".repeat(48);
      expect(DIDSchema.safeParse(longDid).success).toBe(false);
    });
  });
});

// ============ Capability Schema Tests ============

describe("CapabilitySchema", () => {
  const ALL_CAPABILITIES: Capability[] = [
    "compute.inference",
    "compute.training",
    "data.storage",
    "data.retrieval",
    "service.api",
    "service.task",
    "coordination.validator",
    "coordination.oracle",
  ];

  describe("valid capabilities", () => {
    it("should accept all defined capabilities", () => {
      for (const capability of ALL_CAPABILITIES) {
        const result = CapabilitySchema.safeParse(capability);
        expect(result.success).toBe(true);
      }
    });

    it("should have compute capabilities", () => {
      expect(CapabilitySchema.safeParse("compute.inference").success).toBe(
        true,
      );
      expect(CapabilitySchema.safeParse("compute.training").success).toBe(true);
    });

    it("should have data capabilities", () => {
      expect(CapabilitySchema.safeParse("data.storage").success).toBe(true);
      expect(CapabilitySchema.safeParse("data.retrieval").success).toBe(true);
    });

    it("should have service capabilities", () => {
      expect(CapabilitySchema.safeParse("service.api").success).toBe(true);
      expect(CapabilitySchema.safeParse("service.task").success).toBe(true);
    });

    it("should have coordination capabilities", () => {
      expect(CapabilitySchema.safeParse("coordination.validator").success).toBe(
        true,
      );
      expect(CapabilitySchema.safeParse("coordination.oracle").success).toBe(
        true,
      );
    });
  });

  describe("invalid capabilities", () => {
    it("should reject unknown capabilities", () => {
      expect(CapabilitySchema.safeParse("compute.unknown").success).toBe(false);
      expect(CapabilitySchema.safeParse("data.unknown").success).toBe(false);
      expect(CapabilitySchema.safeParse("unknown.category").success).toBe(
        false,
      );
    });

    it("should reject partial matches", () => {
      expect(CapabilitySchema.safeParse("compute").success).toBe(false);
      expect(CapabilitySchema.safeParse("inference").success).toBe(false);
    });

    it("should reject non-string input", () => {
      expect(CapabilitySchema.safeParse(null).success).toBe(false);
      expect(CapabilitySchema.safeParse(undefined).success).toBe(false);
      expect(CapabilitySchema.safeParse(123).success).toBe(false);
    });
  });

  describe("capability categories", () => {
    it("should have 8 total capabilities", () => {
      expect(ALL_CAPABILITIES.length).toBe(8);
    });

    it("should organize by domain (compute, data, service, coordination)", () => {
      const compute = ALL_CAPABILITIES.filter((c) => c.startsWith("compute."));
      const data = ALL_CAPABILITIES.filter((c) => c.startsWith("data."));
      const service = ALL_CAPABILITIES.filter((c) => c.startsWith("service."));
      const coordination = ALL_CAPABILITIES.filter((c) =>
        c.startsWith("coordination."),
      );

      expect(compute.length).toBe(2);
      expect(data.length).toBe(2);
      expect(service.length).toBe(2);
      expect(coordination.length).toBe(2);
    });
  });
});

// ============ createDID Tests ============

describe("createDID", () => {
  describe("basic functionality", () => {
    it("should create DID from public key", () => {
      const did = createDID(VALID_PUBLIC_KEY);

      expect(did.startsWith("did:lgm:")).toBe(true);
    });

    it("should default to devnet", () => {
      const did = createDID(VALID_PUBLIC_KEY);

      expect(did).toContain(":devnet:");
    });

    it("should use specified network", () => {
      const mainnetDid = createDID(VALID_PUBLIC_KEY, "mainnet");
      const testnetDid = createDID(VALID_PUBLIC_KEY, "testnet");
      const devnetDid = createDID(VALID_PUBLIC_KEY, "devnet");

      expect(mainnetDid).toContain(":mainnet:");
      expect(testnetDid).toContain(":testnet:");
      expect(devnetDid).toContain(":devnet:");
    });

    it("should use first 44 chars of public key", () => {
      const did = createDID(VALID_PUBLIC_KEY);
      const address = did.split(":")[3];

      expect(address).toBe(VALID_PUBLIC_KEY.slice(0, 44));
      expect(address).toHaveLength(44);
    });
  });

  describe("determinism", () => {
    it("should produce same DID for same inputs", () => {
      const did1 = createDID(VALID_PUBLIC_KEY, "mainnet");
      const did2 = createDID(VALID_PUBLIC_KEY, "mainnet");

      expect(did1).toBe(did2);
    });

    it("should produce different DIDs for different public keys", () => {
      const keypair1 = generateKeypair();
      const keypair2 = generateKeypair();

      const did1 = createDID(keypair1.publicKey);
      const did2 = createDID(keypair2.publicKey);

      expect(did1).not.toBe(did2);
    });

    it("should produce different DIDs for different networks", () => {
      const mainnetDid = createDID(VALID_PUBLIC_KEY, "mainnet");
      const testnetDid = createDID(VALID_PUBLIC_KEY, "testnet");

      expect(mainnetDid).not.toBe(testnetDid);
    });
  });

  describe("schema compliance", () => {
    it("should create schema-valid DIDs", () => {
      const keypair = generateKeypair();
      const did = createDID(keypair.publicKey, "mainnet");

      const result = DIDSchema.safeParse(did);
      expect(result.success).toBe(true);
    });

    it("should create valid DIDs for all networks", () => {
      const keypair = generateKeypair();

      for (const network of ["mainnet", "testnet", "devnet"] as const) {
        const did = createDID(keypair.publicKey, network);
        const result = DIDSchema.safeParse(did);
        expect(result.success).toBe(true);
      }
    });

    it("should create 1000 valid DIDs", () => {
      for (let i = 0; i < 1000; i++) {
        const keypair = generateKeypair();
        const did = createDID(keypair.publicKey);
        const result = DIDSchema.safeParse(did);
        expect(result.success).toBe(true);
      }
    });
  });
});

// ============ didToAddress Tests ============

describe("didToAddress", () => {
  it("should extract address from mainnet DID", () => {
    const address = didToAddress(VALID_DIDS.mainnet as DID);
    expect(address).toBe(VALID_ADDRESS);
  });

  it("should extract address from testnet DID", () => {
    const address = didToAddress(VALID_DIDS.testnet as DID);
    expect(address).toBe(VALID_ADDRESS);
  });

  it("should extract address from devnet DID", () => {
    const address = didToAddress(VALID_DIDS.devnet as DID);
    expect(address).toBe(VALID_ADDRESS);
  });

  it("should return 44-character hex string for 4-part DID", () => {
    const address = didToAddress(VALID_DIDS.mainnet as DID);

    expect(address).toHaveLength(44);
    expect(address).toMatch(/^[a-f0-9]{44}$/);
  });

  it("should extract address from 3-part compact DID", () => {
    const hash = "a".repeat(32);
    const did = `did:lgm:${hash}` as DID;
    expect(didToAddress(did)).toBe(hash);
  });

  it("should be inverse of createDID address component", () => {
    const keypair = generateKeypair();
    const did = createDID(keypair.publicKey, "mainnet");
    const extractedAddress = didToAddress(did);

    expect(extractedAddress).toBe(keypair.publicKey.slice(0, 44));
  });
});

// ============ didToNetwork Tests ============

describe("didToNetwork", () => {
  it("should extract mainnet", () => {
    const network = didToNetwork(VALID_DIDS.mainnet as DID);
    expect(network).toBe("mainnet");
  });

  it("should extract testnet", () => {
    const network = didToNetwork(VALID_DIDS.testnet as DID);
    expect(network).toBe("testnet");
  });

  it("should extract devnet", () => {
    const network = didToNetwork(VALID_DIDS.devnet as DID);
    expect(network).toBe("devnet");
  });

  it("should be inverse of createDID network parameter", () => {
    const keypair = generateKeypair();

    for (const network of ["mainnet", "testnet", "devnet"] as const) {
      const did = createDID(keypair.publicKey, network);
      const extractedNetwork = didToNetwork(did);
      expect(extractedNetwork).toBe(network);
    }
  });

  it("should default to mainnet for 3-part compact DID", () => {
    const hash = "a".repeat(32);
    const did = `did:lgm:${hash}` as DID;
    expect(didToNetwork(did)).toBe("mainnet");
  });
});

// ============ DID Roundtrip Tests ============

describe("DID Roundtrip", () => {
  it("should roundtrip public key -> DID -> address", () => {
    const keypair = generateKeypair();
    const did = createDID(keypair.publicKey, "mainnet");
    const address = didToAddress(did);

    // Address should be first 44 chars of public key
    expect(address).toBe(keypair.publicKey.slice(0, 44));
  });

  it("should roundtrip network through DID", () => {
    const keypair = generateKeypair();

    for (const network of ["mainnet", "testnet", "devnet"] as const) {
      const did = createDID(keypair.publicKey, network);
      expect(didToNetwork(did)).toBe(network);
    }
  });

  it("should preserve all information through roundtrip", () => {
    const keypair = generateKeypair();
    const network = "testnet";

    const did = createDID(keypair.publicKey, network);
    const extractedAddress = didToAddress(did);
    const extractedNetwork = didToNetwork(did);

    // Reconstruct DID
    const reconstructed = `did:lgm:${extractedNetwork}:${extractedAddress}`;
    expect(reconstructed).toBe(did);
  });
});

// ============ AgentIdentity Tests ============

describe("AgentIdentity", () => {
  function createTestIdentity(): AgentIdentity {
    const keypair = generateKeypair();
    return {
      did: createDID(keypair.publicKey, "devnet"),
      publicKey: keypair.publicKey,
      capabilities: ["compute.inference", "service.api"],
      metadata: {
        name: "Test Agent",
        description: "A test agent for unit tests",
        endpoint: "https://api.example.com/agent",
        version: "1.0.0",
      },
      createdAt: Date.now(),
      updatedAt: Date.now(),
    };
  }

  it("should have valid DID", () => {
    const identity = createTestIdentity();
    const result = DIDSchema.safeParse(identity.did);
    expect(result.success).toBe(true);
  });

  it("should have valid capabilities", () => {
    const identity = createTestIdentity();

    for (const capability of identity.capabilities) {
      const result = CapabilitySchema.safeParse(capability);
      expect(result.success).toBe(true);
    }
  });

  it("should have consistent DID and publicKey", () => {
    const identity = createTestIdentity();
    const addressFromDid = didToAddress(identity.did);

    expect(addressFromDid).toBe(identity.publicKey.slice(0, 44));
  });

  it("should support empty capabilities array", () => {
    const keypair = generateKeypair();
    const identity: AgentIdentity = {
      did: createDID(keypair.publicKey),
      publicKey: keypair.publicKey,
      capabilities: [],
      metadata: {},
      createdAt: Date.now(),
      updatedAt: Date.now(),
    };

    expect(identity.capabilities).toHaveLength(0);
  });

  it("should support minimal metadata", () => {
    const keypair = generateKeypair();
    const identity: AgentIdentity = {
      did: createDID(keypair.publicKey),
      publicKey: keypair.publicKey,
      capabilities: ["compute.inference"],
      metadata: {},
      createdAt: Date.now(),
      updatedAt: Date.now(),
    };

    expect(identity.metadata.name).toBeUndefined();
    expect(identity.metadata.endpoint).toBeUndefined();
  });
});

// ============ DIDDocument Tests ============

describe("createDIDDocument", () => {
  function createTestIdentity(withEndpoint: boolean = true): AgentIdentity {
    const keypair = generateKeypair();
    return {
      did: createDID(keypair.publicKey, "mainnet"),
      publicKey: keypair.publicKey,
      capabilities: ["compute.inference"],
      metadata: {
        name: "Test Agent",
        endpoint: withEndpoint ? "https://api.example.com/agent" : undefined,
      },
      createdAt: Date.now(),
      updatedAt: Date.now(),
    };
  }

  describe("W3C DID Document compliance", () => {
    it("should have correct @context", () => {
      const identity = createTestIdentity();
      const doc = createDIDDocument(identity);

      expect(doc["@context"]).toBe("https://www.w3.org/ns/did/v1");
    });

    it("should have id matching identity DID", () => {
      const identity = createTestIdentity();
      const doc = createDIDDocument(identity);

      expect(doc.id).toBe(identity.did);
    });

    it("should have verificationMethod array", () => {
      const identity = createTestIdentity();
      const doc = createDIDDocument(identity);

      expect(Array.isArray(doc.verificationMethod)).toBe(true);
      expect(doc.verificationMethod.length).toBe(1);
    });

    it("should have authentication array", () => {
      const identity = createTestIdentity();
      const doc = createDIDDocument(identity);

      expect(Array.isArray(doc.authentication)).toBe(true);
      expect(doc.authentication.length).toBe(1);
    });
  });

  describe("verificationMethod", () => {
    it("should have correct key ID format", () => {
      const identity = createTestIdentity();
      const doc = createDIDDocument(identity);
      const vm = doc.verificationMethod[0];

      expect(vm.id).toBe(`${identity.did}#key-1`);
    });

    it("should use Ed25519VerificationKey2020 type", () => {
      const identity = createTestIdentity();
      const doc = createDIDDocument(identity);
      const vm = doc.verificationMethod[0];

      expect(vm.type).toBe("Ed25519VerificationKey2020");
    });

    it("should have controller matching DID", () => {
      const identity = createTestIdentity();
      const doc = createDIDDocument(identity);
      const vm = doc.verificationMethod[0];

      expect(vm.controller).toBe(identity.did);
    });

    it("should encode public key with multibase z prefix", () => {
      const identity = createTestIdentity();
      const doc = createDIDDocument(identity);
      const vm = doc.verificationMethod[0];

      expect(vm.publicKeyMultibase.startsWith("z")).toBe(true);
      expect(vm.publicKeyMultibase).toBe("z" + identity.publicKey);
    });
  });

  describe("authentication", () => {
    it("should reference the verificationMethod", () => {
      const identity = createTestIdentity();
      const doc = createDIDDocument(identity);

      const keyId = doc.verificationMethod[0].id;
      expect(doc.authentication).toContain(keyId);
    });
  });

  describe("service endpoint", () => {
    it("should include service when endpoint is provided", () => {
      const identity = createTestIdentity(true);
      const doc = createDIDDocument(identity);

      expect(doc.service).toBeDefined();
      expect(doc.service?.length).toBe(1);
    });

    it("should not include service when endpoint is not provided", () => {
      const identity = createTestIdentity(false);
      const doc = createDIDDocument(identity);

      expect(doc.service).toBeUndefined();
    });

    it("should have correct service format", () => {
      const identity = createTestIdentity(true);
      const doc = createDIDDocument(identity);
      const service = doc.service![0];

      expect(service.id).toBe(`${identity.did}#agent`);
      expect(service.type).toBe("AgentEndpoint");
      expect(service.serviceEndpoint).toBe(identity.metadata.endpoint);
    });
  });

  describe("document consistency", () => {
    it("should create consistent documents for same identity", () => {
      const identity = createTestIdentity();
      const doc1 = createDIDDocument(identity);
      const doc2 = createDIDDocument(identity);

      expect(JSON.stringify(doc1)).toBe(JSON.stringify(doc2));
    });

    it("should create different documents for different identities", () => {
      const identity1 = createTestIdentity();
      const identity2 = createTestIdentity();

      const doc1 = createDIDDocument(identity1);
      const doc2 = createDIDDocument(identity2);

      expect(doc1.id).not.toBe(doc2.id);
    });
  });
});

// ============ Type Safety Tests ============

describe("Type Safety", () => {
  it("DID type should be string", () => {
    const did: DID = VALID_DIDS.mainnet as DID;
    expect(typeof did).toBe("string");
  });

  it("Capability type should be string", () => {
    const capability: Capability = "compute.inference";
    expect(typeof capability).toBe("string");
  });

  it("createDID should return DID type", () => {
    const keypair = generateKeypair();
    const did: DID = createDID(keypair.publicKey);

    // TypeScript compilation proves type is correct
    expect(typeof did).toBe("string");
  });

  it("DIDDocument should have required structure", () => {
    const keypair = generateKeypair();
    const identity: AgentIdentity = {
      did: createDID(keypair.publicKey),
      publicKey: keypair.publicKey,
      capabilities: [],
      metadata: {},
      createdAt: Date.now(),
      updatedAt: Date.now(),
    };

    const doc: DIDDocument = createDIDDocument(identity);

    // TypeScript compilation proves structure is correct
    expect(doc["@context"]).toBeDefined();
    expect(doc.id).toBeDefined();
    expect(doc.verificationMethod).toBeDefined();
    expect(doc.authentication).toBeDefined();
  });
});

// ============ Edge Cases ============

describe("Edge Cases", () => {
  it("should handle public key with leading zeros", () => {
    const publicKey =
      "0000000000000000000000000000000000000000000000000000000000000000";
    const did = createDID(publicKey, "mainnet");

    expect(DIDSchema.safeParse(did).success).toBe(true);
    expect(didToAddress(did)).toBe(publicKey.slice(0, 44));
  });

  it("should handle public key with all fs", () => {
    const publicKey =
      "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff";
    const did = createDID(publicKey, "mainnet");

    expect(DIDSchema.safeParse(did).success).toBe(true);
    expect(didToAddress(did)).toBe(publicKey.slice(0, 44));
  });

  it("should handle identity with all capabilities", () => {
    const keypair = generateKeypair();
    const identity: AgentIdentity = {
      did: createDID(keypair.publicKey),
      publicKey: keypair.publicKey,
      capabilities: [
        "compute.inference",
        "compute.training",
        "data.storage",
        "data.retrieval",
        "service.api",
        "service.task",
        "coordination.validator",
        "coordination.oracle",
      ],
      metadata: {},
      createdAt: Date.now(),
      updatedAt: Date.now(),
    };

    expect(identity.capabilities.length).toBe(8);
    for (const cap of identity.capabilities) {
      expect(CapabilitySchema.safeParse(cap).success).toBe(true);
    }
  });

  it("should handle timestamps at epoch", () => {
    const keypair = generateKeypair();
    const identity: AgentIdentity = {
      did: createDID(keypair.publicKey),
      publicKey: keypair.publicKey,
      capabilities: [],
      metadata: {},
      createdAt: 0,
      updatedAt: 0,
    };

    expect(identity.createdAt).toBe(0);
    expect(identity.updatedAt).toBe(0);
  });

  it("should handle very long metadata strings", () => {
    const keypair = generateKeypair();
    const longString = "a".repeat(10000);
    const identity: AgentIdentity = {
      did: createDID(keypair.publicKey),
      publicKey: keypair.publicKey,
      capabilities: [],
      metadata: {
        name: longString,
        description: longString,
      },
      createdAt: Date.now(),
      updatedAt: Date.now(),
    };

    const doc = createDIDDocument(identity);
    expect(doc.id).toBe(identity.did);
  });

  it("should handle special characters in metadata", () => {
    const keypair = generateKeypair();
    const identity: AgentIdentity = {
      did: createDID(keypair.publicKey),
      publicKey: keypair.publicKey,
      capabilities: [],
      metadata: {
        name: "Test Agent 🤖",
        description: "Special chars: <>&\"' newline\n tab\t",
        endpoint: "https://example.com/path?param=value&other=123",
      },
      createdAt: Date.now(),
      updatedAt: Date.now(),
    };

    const doc = createDIDDocument(identity);
    expect(doc.service?.[0].serviceEndpoint).toBe(identity.metadata.endpoint);
  });
});

// ============ Integration Tests ============

describe("Integration with Keys", () => {
  it("should create valid DID from generated keypair", () => {
    const keypair = generateKeypair();
    const did = createDID(keypair.publicKey, "mainnet");

    expect(DIDSchema.safeParse(did).success).toBe(true);
  });

  it("should create 100 unique DIDs from unique keypairs", () => {
    const dids = new Set<string>();

    for (let i = 0; i < 100; i++) {
      const keypair = generateKeypair();
      const did = createDID(keypair.publicKey);
      dids.add(did);
    }

    expect(dids.size).toBe(100);
  });

  it("should maintain key-DID relationship", () => {
    const keypair = generateKeypair();
    const did = createDID(keypair.publicKey);
    const address = didToAddress(did);

    // Address is derived from public key
    expect(keypair.publicKey.startsWith(address)).toBe(true);
  });
});
