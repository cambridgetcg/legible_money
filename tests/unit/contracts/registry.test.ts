/**
 * Contract Registry Tests
 *
 * Tests contract registration, discovery, indexing, querying,
 * address generation, statistics, and state export/import.
 *
 * Each test creates its own ContractRegistry to ensure isolation
 * under concurrent test execution.
 */

import { describe, it, expect } from "vitest";
import { ContractRegistry } from "../../../core/contracts/registry.js";
import {
  ContractError,
  type SelfInvokingContract,
  type ContractMetadata,
  type ContractABI,
} from "../../../core/contracts/types.js";

// ============ Test Helpers ============

const CREATOR_DID = "did:lgm:00000000000000000000000000000001";
const OTHER_DID = "did:lgm:00000000000000000000000000000002";

function createMinimalABI(
  functions: ContractABI["functions"] = [],
): ContractABI {
  return {
    functions,
    events: [],
    stateVariables: [],
  };
}

function createMetadata(
  overrides: Partial<ContractMetadata> = {},
): ContractMetadata {
  return {
    name: "TestContract",
    description: "A test self-invoking contract",
    version: "1.0.0",
    author: CREATOR_DID,
    abi: createMinimalABI(),
    ...overrides,
  };
}

function createContract(
  overrides: Partial<SelfInvokingContract> = {},
): SelfInvokingContract {
  return {
    address: "contract_" + Math.random().toString(36).slice(2, 10),
    creator: CREATOR_DID,
    name: "TestContract",
    version: "1.0.0",
    codeHash: "abc123",
    isActive: true,
    invocations: [],
    totalInvocations: 0,
    stateHash: "000",
    deployedAtBlock: 100n,
    gasLimit: 1_000_000n,
    prepaidGas: 500_000n,
    ...overrides,
  };
}

function freshRegistry(): ContractRegistry {
  return new ContractRegistry();
}

// ============ Tests ============

describe("ContractRegistry", () => {
  // ============ Registration ============

  describe("register", () => {
    it("should register a new contract and return a RegistryEntry", () => {
      const registry = freshRegistry();
      const contract = createContract({ address: "addr_001" });
      const metadata = createMetadata();
      const tags = ["defi", "vesting"];

      const entry = registry.register(contract, metadata, tags, 50n);

      expect(entry.address).toBe("addr_001");
      expect(entry.verified).toBe(false);
      expect(entry.registeredAtBlock).toBe(50n);
      expect(entry.tags).toEqual(["defi", "vesting"]);
      expect(entry.metadata.name).toBe("TestContract");
    });

    it("should reject duplicate registration for same address", () => {
      const registry = freshRegistry();
      const contract = createContract({ address: "addr_dup" });
      const metadata = createMetadata();

      registry.register(contract, metadata, [], 1n);

      expect(() => registry.register(contract, metadata, [], 2n)).toThrowError(
        ContractError,
      );
    });

    it("should index contracts by creator", () => {
      const registry = freshRegistry();
      const c1 = createContract({ address: "cr_c1", creator: CREATOR_DID });
      const c2 = createContract({ address: "cr_c2", creator: CREATOR_DID });
      const c3 = createContract({ address: "cr_c3", creator: OTHER_DID });

      registry.register(c1, createMetadata(), [], 1n);
      registry.register(c2, createMetadata(), [], 2n);
      registry.register(c3, createMetadata({ author: OTHER_DID }), [], 3n);

      const creatorContracts = registry.getByCreator(CREATOR_DID);
      expect(creatorContracts).toHaveLength(2);
      expect(creatorContracts.map((c) => c.address).sort()).toEqual([
        "cr_c1",
        "cr_c2",
      ]);
    });

    it("should index contracts by tags", () => {
      const registry = freshRegistry();
      const c1 = createContract({ address: "tg_c1" });
      const c2 = createContract({ address: "tg_c2" });

      registry.register(c1, createMetadata(), ["defi", "staking"], 1n);
      registry.register(c2, createMetadata(), ["defi", "oracle"], 1n);

      expect(registry.getByTag("defi")).toHaveLength(2);
      expect(registry.getByTag("staking")).toHaveLength(1);
      expect(registry.getByTag("oracle")).toHaveLength(1);
      expect(registry.getByTag("nonexistent")).toHaveLength(0);
    });

    it("should index contracts by knowledge domain", () => {
      const registry = freshRegistry();
      const c1 = createContract({
        address: "dom_c1",
        knowledgeDomain: "mathematics",
      });
      const c2 = createContract({
        address: "dom_c2",
        knowledgeDomain: "physics",
      });
      const c3 = createContract({ address: "dom_c3" }); // no domain

      registry.register(c1, createMetadata(), [], 1n);
      registry.register(c2, createMetadata(), [], 1n);
      registry.register(c3, createMetadata(), [], 1n);

      expect(registry.getByDomain("mathematics")).toHaveLength(1);
      expect(registry.getByDomain("physics")).toHaveLength(1);
      expect(registry.getByDomain("chemistry")).toHaveLength(0);
    });

    it("should index contracts by function selector", () => {
      const registry = freshRegistry();
      const abi = createMinimalABI([
        {
          name: "transfer",
          selector: "a9059cbb",
          inputs: [],
          outputs: [],
          stateMutability: "nonpayable",
          selfInvokable: true,
        },
      ]);
      const contract = createContract({ address: "fn_contract" });
      const metadata = createMetadata({ abi });

      registry.register(contract, metadata, [], 1n);

      const result = registry.getByFunction("a9059cbb");
      expect(result).toHaveLength(1);
      expect(result[0].address).toBe("fn_contract");
    });
  });

  // ============ Update & Verify ============

  describe("updateContract", () => {
    it("should update mutable fields of a registered contract", () => {
      const registry = freshRegistry();
      const contract = createContract({ address: "upd_001", isActive: true });
      registry.register(contract, createMetadata(), [], 1n);

      const updated = registry.updateContract("upd_001", {
        isActive: false,
        totalInvocations: 5,
      });

      expect(updated.isActive).toBe(false);
      expect(updated.totalInvocations).toBe(5);
    });

    it("should throw for non-existent contract", () => {
      const registry = freshRegistry();
      expect(() =>
        registry.updateContract("nonexistent", { isActive: false }),
      ).toThrowError(ContractError);
    });

    it("should reject changing immutable address field", () => {
      const registry = freshRegistry();
      const contract = createContract({ address: "immut_addr" });
      registry.register(contract, createMetadata(), [], 1n);

      expect(() =>
        registry.updateContract("immut_addr", { address: "new_addr" }),
      ).toThrowError(ContractError);
    });

    it("should reject changing immutable creator field", () => {
      const registry = freshRegistry();
      const contract = createContract({
        address: "immut_creator",
        creator: CREATOR_DID,
      });
      registry.register(contract, createMetadata(), [], 1n);

      expect(() =>
        registry.updateContract("immut_creator", { creator: OTHER_DID }),
      ).toThrowError(ContractError);
    });

    it("should reject changing immutable deployedAtBlock field", () => {
      const registry = freshRegistry();
      const contract = createContract({
        address: "immut_block",
        deployedAtBlock: 100n,
      });
      registry.register(contract, createMetadata(), [], 1n);

      expect(() =>
        registry.updateContract("immut_block", { deployedAtBlock: 200n }),
      ).toThrowError(ContractError);
    });

    it("should allow setting same value for immutable fields (no-op)", () => {
      const registry = freshRegistry();
      const contract = createContract({
        address: "same_val",
        creator: CREATOR_DID,
        deployedAtBlock: 100n,
      });
      registry.register(contract, createMetadata(), [], 1n);

      const updated = registry.updateContract("same_val", {
        address: "same_val",
        creator: CREATOR_DID,
        deployedAtBlock: 100n,
      });
      expect(updated.address).toBe("same_val");
    });
  });

  describe("verify", () => {
    it("should mark contract as verified when source hash matches", () => {
      const registry = freshRegistry();
      const contract = createContract({ address: "verify_001" });
      const metadata = createMetadata({ sourceHash: "src_hash_abc" });
      registry.register(contract, metadata, [], 1n);

      const entry = registry.verify("verify_001", "src_hash_abc");
      expect(entry.verified).toBe(true);
    });

    it("should throw for non-existent contract", () => {
      const registry = freshRegistry();
      expect(() => registry.verify("nonexistent", "hash")).toThrowError(
        ContractError,
      );
    });

    it("should throw for source hash mismatch", () => {
      const registry = freshRegistry();
      const contract = createContract({ address: "verify_bad" });
      const metadata = createMetadata({ sourceHash: "correct_hash" });
      registry.register(contract, metadata, [], 1n);

      expect(() => registry.verify("verify_bad", "wrong_hash")).toThrowError(
        ContractError,
      );
    });
  });

  describe("addTags", () => {
    it("should add new tags to a contract entry", () => {
      const registry = freshRegistry();
      const contract = createContract({ address: "tag_001" });
      registry.register(contract, createMetadata(), ["initial"], 1n);

      const entry = registry.addTags("tag_001", ["new_tag", "another"]);
      expect(entry.tags).toContain("initial");
      expect(entry.tags).toContain("new_tag");
      expect(entry.tags).toContain("another");
    });

    it("should not duplicate existing tags", () => {
      const registry = freshRegistry();
      const contract = createContract({ address: "tag_dup" });
      registry.register(contract, createMetadata(), ["defi"], 1n);

      const entry = registry.addTags("tag_dup", ["defi", "new"]);
      const defiCount = entry.tags.filter((t) => t === "defi").length;
      expect(defiCount).toBe(1);
      expect(entry.tags).toContain("new");
    });

    it("should throw for non-existent contract", () => {
      const registry = freshRegistry();
      expect(() => registry.addTags("nonexistent", ["tag"])).toThrowError(
        ContractError,
      );
    });
  });

  // ============ Query Methods ============

  describe("getContract / getEntry", () => {
    it("should return contract by address", () => {
      const registry = freshRegistry();
      const contract = createContract({ address: "get_001" });
      registry.register(contract, createMetadata(), [], 1n);

      const retrieved = registry.getContract("get_001");
      expect(retrieved).toBeDefined();
      expect(retrieved!.address).toBe("get_001");
    });

    it("should return undefined for missing contract", () => {
      const registry = freshRegistry();
      expect(registry.getContract("nonexistent")).toBeUndefined();
    });

    it("should return entry by address", () => {
      const registry = freshRegistry();
      const contract = createContract({ address: "entry_001" });
      registry.register(contract, createMetadata(), ["test"], 5n);

      const entry = registry.getEntry("entry_001");
      expect(entry).toBeDefined();
      expect(entry!.registeredAtBlock).toBe(5n);
      expect(entry!.tags).toContain("test");
    });

    it("should return undefined entry for missing address", () => {
      const registry = freshRegistry();
      expect(registry.getEntry("nonexistent")).toBeUndefined();
    });
  });

  describe("getVerified", () => {
    it("should return only verified contracts", () => {
      const registry = freshRegistry();
      const c1 = createContract({ address: "v1" });
      const c2 = createContract({ address: "v2" });
      registry.register(c1, createMetadata({ sourceHash: "h1" }), [], 1n);
      registry.register(c2, createMetadata({ sourceHash: "h2" }), [], 1n);

      registry.verify("v1", "h1");

      const verified = registry.getVerified();
      expect(verified).toHaveLength(1);
      expect(verified[0].address).toBe("v1");
    });
  });

  describe("getActive", () => {
    it("should return only active contracts", () => {
      const registry = freshRegistry();
      const c1 = createContract({ address: "a1", isActive: true });
      const c2 = createContract({ address: "a2", isActive: false });
      const c3 = createContract({ address: "a3", isActive: true });

      registry.register(c1, createMetadata(), [], 1n);
      registry.register(c2, createMetadata(), [], 1n);
      registry.register(c3, createMetadata(), [], 1n);

      const active = registry.getActive();
      expect(active).toHaveLength(2);
      expect(active.map((c) => c.address).sort()).toEqual(["a1", "a3"]);
    });
  });

  describe("search", () => {
    it("should find contracts by name (case-insensitive)", () => {
      const registry = freshRegistry();
      const c1 = createContract({ address: "srch_1" });
      const c2 = createContract({ address: "srch_2" });
      registry.register(c1, createMetadata({ name: "Token Vesting" }), [], 1n);
      registry.register(
        c2,
        createMetadata({ name: "Oracle Price Feed" }),
        [],
        1n,
      );

      const results = registry.search("vesting");
      expect(results).toHaveLength(1);
      expect(results[0].address).toBe("srch_1");
    });

    it("should find contracts by description", () => {
      const registry = freshRegistry();
      const contract = createContract({ address: "desc_001" });
      registry.register(
        contract,
        createMetadata({
          description: "Manages knowledge verification deadlines",
        }),
        [],
        1n,
      );

      const results = registry.search("knowledge verification");
      expect(results).toHaveLength(1);
    });

    it("should return empty array for no matches", () => {
      const registry = freshRegistry();
      const contract = createContract({ address: "no_match" });
      registry.register(contract, createMetadata(), [], 1n);

      const results = registry.search("zzzznonexistent");
      expect(results).toHaveLength(0);
    });
  });

  // ============ Self-Invokable Functions ============

  describe("getSelfInvokableFunctions", () => {
    it("should return only self-invokable functions for a contract", () => {
      const registry = freshRegistry();
      const abi = createMinimalABI([
        {
          name: "transfer",
          selector: "a9059cbb",
          inputs: [],
          outputs: [],
          stateMutability: "nonpayable",
          selfInvokable: false,
        },
        {
          name: "distribute",
          selector: "deadbeef",
          inputs: [],
          outputs: [],
          stateMutability: "nonpayable",
          selfInvokable: true,
        },
      ]);
      const contract = createContract({ address: "si_001" });
      registry.register(contract, createMetadata({ abi }), [], 1n);

      const fns = registry.getSelfInvokableFunctions("si_001");
      expect(fns).toHaveLength(1);
      expect(fns[0].name).toBe("distribute");
    });

    it("should return empty array for non-existent contract", () => {
      const registry = freshRegistry();
      const fns = registry.getSelfInvokableFunctions("nonexistent");
      expect(fns).toEqual([]);
    });
  });

  describe("getContractsWithSelfInvokableFunctions", () => {
    it("should return contracts that have at least one self-invokable function", () => {
      const registry = freshRegistry();
      const abiWithSI = createMinimalABI([
        {
          name: "auto",
          selector: "11111111",
          inputs: [],
          outputs: [],
          stateMutability: "nonpayable",
          selfInvokable: true,
        },
      ]);
      const abiWithout = createMinimalABI([
        {
          name: "manual",
          selector: "22222222",
          inputs: [],
          outputs: [],
          stateMutability: "nonpayable",
          selfInvokable: false,
        },
      ]);

      const c1 = createContract({ address: "si_c1" });
      const c2 = createContract({ address: "si_c2" });

      registry.register(c1, createMetadata({ abi: abiWithSI }), [], 1n);
      registry.register(c2, createMetadata({ abi: abiWithout }), [], 1n);

      const results = registry.getContractsWithSelfInvokableFunctions();
      expect(results).toHaveLength(1);
      expect(results[0].address).toBe("si_c1");
    });
  });

  // ============ Address & Selector Generation ============

  describe("generateAddress (static)", () => {
    it("should generate deterministic address from creator + nonce", () => {
      const addr1 = ContractRegistry.generateAddress(CREATOR_DID, 0n);
      const addr2 = ContractRegistry.generateAddress(CREATOR_DID, 0n);
      expect(addr1).toBe(addr2);
    });

    it("should generate different addresses for different nonces", () => {
      const addr1 = ContractRegistry.generateAddress(CREATOR_DID, 0n);
      const addr2 = ContractRegistry.generateAddress(CREATOR_DID, 1n);
      expect(addr1).not.toBe(addr2);
    });

    it("should generate different addresses for different creators", () => {
      const addr1 = ContractRegistry.generateAddress(CREATOR_DID, 0n);
      const addr2 = ContractRegistry.generateAddress(OTHER_DID, 0n);
      expect(addr1).not.toBe(addr2);
    });

    it("should incorporate salt when provided", () => {
      const salt = new Uint8Array([1, 2, 3, 4]);
      const addrNoSalt = ContractRegistry.generateAddress(CREATOR_DID, 0n);
      const addrWithSalt = ContractRegistry.generateAddress(
        CREATOR_DID,
        0n,
        salt,
      );
      expect(addrNoSalt).not.toBe(addrWithSalt);
    });

    it("should return a 42-character hex string", () => {
      const addr = ContractRegistry.generateAddress(CREATOR_DID, 0n);
      expect(addr).toHaveLength(42);
      expect(addr).toMatch(/^[a-f0-9]{42}$/);
    });
  });

  describe("generateSelector (static)", () => {
    it("should generate deterministic selector from function signature", () => {
      const sel1 = ContractRegistry.generateSelector(
        "transfer(address,uint256)",
      );
      const sel2 = ContractRegistry.generateSelector(
        "transfer(address,uint256)",
      );
      expect(sel1).toBe(sel2);
    });

    it("should generate different selectors for different signatures", () => {
      const sel1 = ContractRegistry.generateSelector(
        "transfer(address,uint256)",
      );
      const sel2 = ContractRegistry.generateSelector(
        "approve(address,uint256)",
      );
      expect(sel1).not.toBe(sel2);
    });

    it("should return 8-character hex string (4 bytes)", () => {
      const sel = ContractRegistry.generateSelector("myFunction()");
      expect(sel).toHaveLength(8);
      expect(sel).toMatch(/^[a-f0-9]{8}$/);
    });
  });

  // ============ Statistics ============

  describe("getStatistics", () => {
    it("should return correct counts for empty registry", () => {
      const registry = freshRegistry();
      const stats = registry.getStatistics();
      expect(stats.totalContracts).toBe(0);
      expect(stats.activeContracts).toBe(0);
      expect(stats.verifiedContracts).toBe(0);
      expect(stats.totalInvocations).toBe(0);
    });

    it("should count active and verified contracts correctly", () => {
      const registry = freshRegistry();
      const c1 = createContract({
        address: "stat_s1",
        isActive: true,
        totalInvocations: 10,
      });
      const c2 = createContract({
        address: "stat_s2",
        isActive: false,
        totalInvocations: 5,
      });
      registry.register(c1, createMetadata({ sourceHash: "h1" }), ["tag"], 1n);
      registry.register(c2, createMetadata({ sourceHash: "h2" }), [], 1n);
      registry.verify("stat_s1", "h1");

      const stats = registry.getStatistics();
      expect(stats.totalContracts).toBe(2);
      expect(stats.activeContracts).toBe(1);
      expect(stats.verifiedContracts).toBe(1);
      expect(stats.totalInvocations).toBe(15);
    });

    it("should track contracts by domain and tag", () => {
      const registry = freshRegistry();
      const c1 = createContract({
        address: "stat_d1",
        knowledgeDomain: "math",
      });
      const c2 = createContract({
        address: "stat_d2",
        knowledgeDomain: "math",
      });
      registry.register(c1, createMetadata(), ["defi"], 1n);
      registry.register(c2, createMetadata(), ["defi", "oracle"], 1n);

      const stats = registry.getStatistics();
      expect(stats.contractsByDomain.get("math")).toBe(2);
      expect(stats.contractsByTag.get("defi")).toBe(2);
      expect(stats.contractsByTag.get("oracle")).toBe(1);
    });

    it("should track top creators", () => {
      const registry = freshRegistry();
      for (let i = 0; i < 3; i++) {
        registry.register(
          createContract({ address: `tc_${i}`, creator: CREATOR_DID }),
          createMetadata(),
          [],
          1n,
        );
      }
      registry.register(
        createContract({ address: "tc_other", creator: OTHER_DID }),
        createMetadata({ author: OTHER_DID }),
        [],
        1n,
      );

      const stats = registry.getStatistics();
      expect(stats.topCreators[0].creator).toBe(CREATOR_DID);
      expect(stats.topCreators[0].count).toBe(3);
    });
  });

  // ============ State Export / Import ============

  describe("exportState / importState", () => {
    it("should export and reimport state preserving all contracts", () => {
      const registry = freshRegistry();
      const c1 = createContract({
        address: "exp1",
        knowledgeDomain: "physics",
      });
      const c2 = createContract({ address: "exp2" });
      const abi = createMinimalABI([
        {
          name: "fn1",
          selector: "aabbccdd",
          inputs: [],
          outputs: [],
          stateMutability: "view",
          selfInvokable: true,
        },
      ]);

      registry.register(c1, createMetadata({ abi }), ["tag1"], 1n);
      registry.register(c2, createMetadata(), ["tag2"], 2n);

      const exported = registry.exportState();

      // Create a new registry and import
      const newRegistry = new ContractRegistry();
      newRegistry.importState(exported);

      // Verify contracts were imported
      expect(newRegistry.getContract("exp1")).toBeDefined();
      expect(newRegistry.getContract("exp2")).toBeDefined();

      // Verify indexes were rebuilt
      expect(newRegistry.getByCreator(CREATOR_DID)).toHaveLength(2);
      expect(newRegistry.getByDomain("physics")).toHaveLength(1);
      expect(newRegistry.getByTag("tag1")).toHaveLength(1);
      expect(newRegistry.getByTag("tag2")).toHaveLength(1);
      expect(newRegistry.getByFunction("aabbccdd")).toHaveLength(1);
    });

    it("should clear existing state before import", () => {
      const registry = freshRegistry();
      const c1 = createContract({ address: "old_001" });
      registry.register(c1, createMetadata(), [], 1n);

      // Import empty state
      const emptyRegistry = new ContractRegistry();
      const emptyState = emptyRegistry.exportState();
      registry.importState(emptyState);

      expect(registry.getContract("old_001")).toBeUndefined();
      expect(registry.getAllContracts()).toHaveLength(0);
    });
  });

  // ============ Iteration ============

  describe("getAllContracts / getAllEntries", () => {
    it("should return all contracts as array", () => {
      const registry = freshRegistry();
      registry.register(
        createContract({ address: "all1" }),
        createMetadata(),
        [],
        1n,
      );
      registry.register(
        createContract({ address: "all2" }),
        createMetadata(),
        [],
        1n,
      );

      const all = registry.getAllContracts();
      expect(all).toHaveLength(2);
    });

    it("should return all entries as array", () => {
      const registry = freshRegistry();
      registry.register(
        createContract({ address: "ent1" }),
        createMetadata(),
        ["x"],
        1n,
      );

      const entries = registry.getAllEntries();
      expect(entries).toHaveLength(1);
      expect(entries[0].address).toBe("ent1");
    });
  });
});
