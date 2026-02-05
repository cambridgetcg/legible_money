/**
 * Contract Registry
 *
 * Manages registration, discovery, and metadata for self-invoking contracts.
 * Provides a decentralized registry for contract discovery and verification.
 *
 * From Patent: Smart contracts are deployed and managed through the BVM,
 * with their self-invocation schedules registered with the SPU.
 *
 * PP HIERARCHY:
 *   TRUTH: Registry state is verifiable via Merkle proofs
 *   UNDERSTANDING: Clear interface for contract discovery
 *   BEAUTY: Single registry manages all contract metadata
 *   JUSTICE: All contracts are equally discoverable
 */

import { sha512_256Hex, domainHash } from "../crypto/hashing.js";
import type { DID } from "../agents/identity/did.js";
import {
  type SelfInvokingContract,
  type RegistryEntry,
  type ContractMetadata,
  type ABIFunction,
  ContractError,
} from "./types.js";

/**
 * Contract Registry
 * Manages contract registration and discovery
 */
export class ContractRegistry {
  private contracts: Map<string, SelfInvokingContract> = new Map();
  private entries: Map<string, RegistryEntry> = new Map();
  private byCreator: Map<string, Set<string>> = new Map();
  private byTag: Map<string, Set<string>> = new Map();
  private byDomain: Map<string, Set<string>> = new Map();
  private byFunction: Map<string, Set<string>> = new Map();

  constructor() {}

  // ============ Registration ============

  /**
   * Register a new contract in the registry
   */
  register(
    contract: SelfInvokingContract,
    metadata: ContractMetadata,
    tags: string[],
    currentBlock: bigint,
  ): RegistryEntry {
    // Check if already registered
    if (this.contracts.has(contract.address)) {
      throw new ContractError(
        `Contract ${contract.address} already registered`,
        "DEPLOYMENT_FAILED",
      );
    }

    // Create registry entry
    const entry: RegistryEntry = {
      address: contract.address,
      metadata,
      verified: false,
      registeredAtBlock: currentBlock,
      tags,
    };

    // Store contract and entry
    this.contracts.set(contract.address, contract);
    this.entries.set(contract.address, entry);

    // Index by creator
    const creatorContracts = this.byCreator.get(contract.creator) ?? new Set();
    creatorContracts.add(contract.address);
    this.byCreator.set(contract.creator, creatorContracts);

    // Index by tags
    for (const tag of tags) {
      const tagContracts = this.byTag.get(tag) ?? new Set();
      tagContracts.add(contract.address);
      this.byTag.set(tag, tagContracts);
    }

    // Index by domain
    if (contract.knowledgeDomain) {
      const domainContracts =
        this.byDomain.get(contract.knowledgeDomain) ?? new Set();
      domainContracts.add(contract.address);
      this.byDomain.set(contract.knowledgeDomain, domainContracts);
    }

    // Index by function selectors
    for (const fn of metadata.abi.functions) {
      const fnContracts = this.byFunction.get(fn.selector) ?? new Set();
      fnContracts.add(contract.address);
      this.byFunction.set(fn.selector, fnContracts);
    }

    return entry;
  }

  /**
   * Update contract state
   */
  updateContract(
    address: string,
    updates: Partial<SelfInvokingContract>,
  ): SelfInvokingContract {
    const contract = this.contracts.get(address);
    if (!contract) {
      throw new ContractError(
        `Contract ${address} not found`,
        "CONTRACT_NOT_FOUND",
      );
    }

    const updated = { ...contract, ...updates };
    this.contracts.set(address, updated);
    return updated;
  }

  /**
   * Mark contract as verified (source code verified)
   */
  verify(address: string, sourceHash: string): RegistryEntry {
    const entry = this.entries.get(address);
    if (!entry) {
      throw new ContractError(
        `Contract ${address} not found`,
        "CONTRACT_NOT_FOUND",
      );
    }

    // Verify source hash matches
    if (entry.metadata.sourceHash !== sourceHash) {
      throw new ContractError("Source hash mismatch", "DEPLOYMENT_FAILED", {
        expected: entry.metadata.sourceHash,
        provided: sourceHash,
      });
    }

    entry.verified = true;
    this.entries.set(address, entry);
    return entry;
  }

  /**
   * Add tags to a contract
   */
  addTags(address: string, tags: string[]): RegistryEntry {
    const entry = this.entries.get(address);
    if (!entry) {
      throw new ContractError(
        `Contract ${address} not found`,
        "CONTRACT_NOT_FOUND",
      );
    }

    for (const tag of tags) {
      if (!entry.tags.includes(tag)) {
        entry.tags.push(tag);

        const tagContracts = this.byTag.get(tag) ?? new Set();
        tagContracts.add(address);
        this.byTag.set(tag, tagContracts);
      }
    }

    this.entries.set(address, entry);
    return entry;
  }

  // ============ Query Methods ============

  /**
   * Get contract by address
   */
  getContract(address: string): SelfInvokingContract | undefined {
    return this.contracts.get(address);
  }

  /**
   * Get registry entry by address
   */
  getEntry(address: string): RegistryEntry | undefined {
    return this.entries.get(address);
  }

  /**
   * Get contracts by creator
   */
  getByCreator(creator: DID): SelfInvokingContract[] {
    const addresses = this.byCreator.get(creator);
    if (!addresses) {
      return [];
    }

    return Array.from(addresses)
      .map((addr) => this.contracts.get(addr))
      .filter((c): c is SelfInvokingContract => c !== undefined);
  }

  /**
   * Get contracts by tag
   */
  getByTag(tag: string): SelfInvokingContract[] {
    const addresses = this.byTag.get(tag);
    if (!addresses) {
      return [];
    }

    return Array.from(addresses)
      .map((addr) => this.contracts.get(addr))
      .filter((c): c is SelfInvokingContract => c !== undefined);
  }

  /**
   * Get contracts by knowledge domain
   */
  getByDomain(domain: string): SelfInvokingContract[] {
    const addresses = this.byDomain.get(domain);
    if (!addresses) {
      return [];
    }

    return Array.from(addresses)
      .map((addr) => this.contracts.get(addr))
      .filter((c): c is SelfInvokingContract => c !== undefined);
  }

  /**
   * Get contracts that implement a specific function
   */
  getByFunction(selector: string): SelfInvokingContract[] {
    const addresses = this.byFunction.get(selector);
    if (!addresses) {
      return [];
    }

    return Array.from(addresses)
      .map((addr) => this.contracts.get(addr))
      .filter((c): c is SelfInvokingContract => c !== undefined);
  }

  /**
   * Get all verified contracts
   */
  getVerified(): SelfInvokingContract[] {
    const verified: SelfInvokingContract[] = [];
    for (const [address, entry] of this.entries) {
      if (entry.verified) {
        const contract = this.contracts.get(address);
        if (contract) {
          verified.push(contract);
        }
      }
    }
    return verified;
  }

  /**
   * Get all active contracts
   */
  getActive(): SelfInvokingContract[] {
    return Array.from(this.contracts.values()).filter((c) => c.isActive);
  }

  /**
   * Search contracts by name or description
   */
  search(query: string): RegistryEntry[] {
    const lowercaseQuery = query.toLowerCase();
    const results: RegistryEntry[] = [];

    for (const entry of this.entries.values()) {
      if (
        entry.metadata.name.toLowerCase().includes(lowercaseQuery) ||
        entry.metadata.description.toLowerCase().includes(lowercaseQuery)
      ) {
        results.push(entry);
      }
    }

    return results;
  }

  // ============ Self-Invokable Functions ============

  /**
   * Get all self-invokable functions from a contract
   */
  getSelfInvokableFunctions(address: string): ABIFunction[] {
    const entry = this.entries.get(address);
    if (!entry) {
      return [];
    }

    return entry.metadata.abi.functions.filter((fn) => fn.selfInvokable);
  }

  /**
   * Get all contracts with self-invokable functions
   */
  getContractsWithSelfInvokableFunctions(): SelfInvokingContract[] {
    const results: SelfInvokingContract[] = [];

    for (const [address, entry] of this.entries) {
      const hasSelfInvokable = entry.metadata.abi.functions.some(
        (fn) => fn.selfInvokable,
      );
      if (hasSelfInvokable) {
        const contract = this.contracts.get(address);
        if (contract) {
          results.push(contract);
        }
      }
    }

    return results;
  }

  // ============ Address Generation ============

  /**
   * Generate deterministic contract address
   * From Patent: Contract addresses are derived from creator + nonce
   */
  static generateAddress(
    creator: DID,
    nonce: bigint,
    salt?: Uint8Array,
  ): string {
    const input = salt
      ? `contract:${creator}:${nonce}:${Buffer.from(salt).toString("hex")}`
      : `contract:${creator}:${nonce}`;

    return Buffer.from(
      domainHash("LGM.contract.address.v1", input).slice(0, 21),
    )
      .toString("hex")
      .slice(0, 42);
  }

  /**
   * Generate function selector from signature
   */
  static generateSelector(functionSignature: string): string {
    const hash = sha512_256Hex(functionSignature);
    return hash.slice(0, 8); // First 4 bytes
  }

  // ============ Statistics ============

  /**
   * Get registry statistics
   */
  getStatistics(): {
    totalContracts: number;
    activeContracts: number;
    verifiedContracts: number;
    totalInvocations: number;
    contractsByDomain: Map<string, number>;
    contractsByTag: Map<string, number>;
    topCreators: Array<{ creator: DID; count: number }>;
  } {
    let activeContracts = 0;
    let verifiedContracts = 0;
    let totalInvocations = 0;
    const contractsByDomain = new Map<string, number>();
    const contractsByTag = new Map<string, number>();
    const creatorCounts = new Map<string, number>();

    for (const contract of this.contracts.values()) {
      if (contract.isActive) activeContracts++;
      totalInvocations += contract.totalInvocations;

      if (contract.knowledgeDomain) {
        const count = contractsByDomain.get(contract.knowledgeDomain) ?? 0;
        contractsByDomain.set(contract.knowledgeDomain, count + 1);
      }

      const creatorCount = creatorCounts.get(contract.creator) ?? 0;
      creatorCounts.set(contract.creator, creatorCount + 1);
    }

    for (const entry of this.entries.values()) {
      if (entry.verified) verifiedContracts++;

      for (const tag of entry.tags) {
        const count = contractsByTag.get(tag) ?? 0;
        contractsByTag.set(tag, count + 1);
      }
    }

    // Top creators
    const topCreators = Array.from(creatorCounts.entries())
      .map(([creator, count]) => ({ creator, count }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 10);

    return {
      totalContracts: this.contracts.size,
      activeContracts,
      verifiedContracts,
      totalInvocations,
      contractsByDomain,
      contractsByTag,
      topCreators,
    };
  }

  // ============ State Export/Import ============

  /**
   * Export registry state
   */
  exportState(): {
    contracts: Map<string, SelfInvokingContract>;
    entries: Map<string, RegistryEntry>;
  } {
    return {
      contracts: new Map(this.contracts),
      entries: new Map(this.entries),
    };
  }

  /**
   * Import registry state
   */
  importState(state: {
    contracts: Map<string, SelfInvokingContract>;
    entries: Map<string, RegistryEntry>;
  }): void {
    // Clear existing state
    this.contracts.clear();
    this.entries.clear();
    this.byCreator.clear();
    this.byTag.clear();
    this.byDomain.clear();
    this.byFunction.clear();

    // Import and rebuild indexes
    for (const [address, contract] of state.contracts) {
      this.contracts.set(address, contract);

      // Rebuild creator index
      const creatorContracts =
        this.byCreator.get(contract.creator) ?? new Set();
      creatorContracts.add(address);
      this.byCreator.set(contract.creator, creatorContracts);

      // Rebuild domain index
      if (contract.knowledgeDomain) {
        const domainContracts =
          this.byDomain.get(contract.knowledgeDomain) ?? new Set();
        domainContracts.add(address);
        this.byDomain.set(contract.knowledgeDomain, domainContracts);
      }
    }

    for (const [address, entry] of state.entries) {
      this.entries.set(address, entry);

      // Rebuild tag index
      for (const tag of entry.tags) {
        const tagContracts = this.byTag.get(tag) ?? new Set();
        tagContracts.add(address);
        this.byTag.set(tag, tagContracts);
      }

      // Rebuild function index
      for (const fn of entry.metadata.abi.functions) {
        const fnContracts = this.byFunction.get(fn.selector) ?? new Set();
        fnContracts.add(address);
        this.byFunction.set(fn.selector, fnContracts);
      }
    }
  }

  /**
   * Get all contracts (for iteration)
   */
  getAllContracts(): SelfInvokingContract[] {
    return Array.from(this.contracts.values());
  }

  /**
   * Get all entries (for iteration)
   */
  getAllEntries(): RegistryEntry[] {
    return Array.from(this.entries.values());
  }
}
