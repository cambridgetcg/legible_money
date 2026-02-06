/**
 * Account State Tree
 *
 * Sparse Merkle Tree for agent account state.
 * Key: hash(DID) - 32 bytes
 * Value: serialized AccountState
 *
 * Provides:
 * - O(log n) balance lookups with proofs
 * - Atomic balance transfers
 * - Session key tracking
 * - Nonce management
 */

import {
  InMemorySMT,
  type MerkleRoot,
  type MerkleProofData,
} from "../core/crypto/merkle.js";
import { sha512_256 } from "../core/crypto/hashing.js";
import {
  type AccountState,
  type AccountDelta,
  type AccountFlags,
  type AccountKey,
  type BalanceProof,
  StateError,
} from "./types.js";

// ============ Serialization ============

/**
 * Serialize account state to bytes for SMT storage
 * Fixed-size encoding for efficiency
 */
function serializeAccountState(account: AccountState): Uint8Array {
  // Encode as JSON for now (TODO: binary encoding for production)
  const json = JSON.stringify({
    did: account.did,
    balance: account.balance.toString(),
    stakedBalance: account.stakedBalance.toString(),
    nonce: account.nonce.toString(),
    operationalKeyHash: account.operationalKeyHash,
    operationalKeyVersion: account.operationalKeyVersion,
    sessionKeyCount: account.sessionKeyCount,
    reputationScore: account.reputationScore,
    lastActiveBlock: account.lastActiveBlock.toString(),
    flags: packFlags(account.flags),
    availability: account.availability
      ? {
          available: account.availability.available,
          capabilities: account.availability.capabilities,
          preferredDomains: account.availability.preferredDomains,
          minimumBounty: account.availability.minimumBounty.toString(),
          updatedAtBlock: account.availability.updatedAtBlock.toString(),
        }
      : undefined,
  });
  return new TextEncoder().encode(json);
}

/**
 * Deserialize account state from bytes
 */
function deserializeAccountState(bytes: Uint8Array): AccountState {
  const json = new TextDecoder().decode(bytes);
  const obj = JSON.parse(json);
  return {
    did: obj.did,
    balance: BigInt(obj.balance),
    stakedBalance: BigInt(obj.stakedBalance),
    nonce: BigInt(obj.nonce),
    operationalKeyHash: obj.operationalKeyHash,
    operationalKeyVersion: obj.operationalKeyVersion,
    sessionKeyCount: obj.sessionKeyCount,
    reputationScore: obj.reputationScore,
    lastActiveBlock: BigInt(obj.lastActiveBlock),
    flags: unpackFlags(obj.flags),
    availability: obj.availability
      ? {
          available: obj.availability.available,
          capabilities: obj.availability.capabilities,
          preferredDomains: obj.availability.preferredDomains,
          minimumBounty: BigInt(obj.availability.minimumBounty),
          updatedAtBlock: BigInt(obj.availability.updatedAtBlock),
        }
      : undefined,
  };
}

/**
 * Pack flags into a number
 */
function packFlags(flags: AccountFlags): number {
  let packed = 0;
  if (flags.isValidator) packed |= 1;
  if (flags.canSubmitClaims) packed |= 2;
  if (flags.canChallenge) packed |= 4;
  if (flags.inRecovery) packed |= 8;
  if (flags.frozen) packed |= 16;
  return packed;
}

/**
 * Unpack flags from a number
 */
function unpackFlags(packed: number): AccountFlags {
  return {
    isValidator: (packed & 1) !== 0,
    canSubmitClaims: (packed & 2) !== 0,
    canChallenge: (packed & 4) !== 0,
    inRecovery: (packed & 8) !== 0,
    frozen: (packed & 16) !== 0,
  };
}

/**
 * Convert DID to SMT key
 */
function didToKey(did: string): AccountKey {
  return sha512_256(did);
}

// ============ Account State Tree ============

export class AccountStateTree {
  private smt: InMemorySMT;
  private accountCache: Map<string, AccountState> = new Map();
  private readonly cacheSize: number;

  constructor(cacheSize: number = 10000) {
    this.smt = new InMemorySMT();
    this.cacheSize = cacheSize;
  }

  /**
   * Get the current root hash
   */
  getRoot(): MerkleRoot {
    return this.smt.getRoot();
  }

  /**
   * Get account by DID
   */
  getAccount(did: string): AccountState | null {
    // Check cache first
    const cached = this.accountCache.get(did);
    if (cached) return cached;

    // Get from SMT
    const key = didToKey(did);
    const value = this.smt.get(key);
    if (!value) return null;

    const account = deserializeAccountState(value);
    this.cacheAccount(did, account);
    return account;
  }

  /**
   * Get account balance
   */
  getBalance(did: string): bigint {
    const account = this.getAccount(did);
    return account?.balance ?? 0n;
  }

  /**
   * Get account nonce
   */
  getNonce(did: string): bigint {
    const account = this.getAccount(did);
    return account?.nonce ?? 0n;
  }

  /**
   * Check if account exists
   */
  exists(did: string): boolean {
    return this.getAccount(did) !== null;
  }

  /**
   * Create a new account
   */
  createAccount(
    did: string,
    operationalKeyHash: string,
    initialBalance: bigint = 0n,
  ): MerkleRoot {
    if (this.exists(did)) {
      throw new StateError("Account already exists", "INVALID_TRANSITION", {
        did,
      });
    }

    const account: AccountState = {
      did,
      balance: initialBalance,
      stakedBalance: 0n,
      nonce: 0n,
      operationalKeyHash,
      operationalKeyVersion: 1,
      sessionKeyCount: 0,
      reputationScore: 0.5, // Neutral starting reputation
      lastActiveBlock: 0n,
      flags: {
        isValidator: false,
        canSubmitClaims: true,
        canChallenge: true,
        inRecovery: false,
        frozen: false,
      },
    };

    const key = didToKey(did);
    const value = serializeAccountState(account);
    const newRoot = this.smt.set(key, value);

    this.cacheAccount(did, account);
    return newRoot;
  }

  /**
   * Update account state
   */
  updateAccount(did: string, update: Partial<AccountState>): MerkleRoot {
    const existing = this.getAccount(did);
    if (!existing) {
      throw new StateError("Account not found", "ACCOUNT_NOT_FOUND", { did });
    }

    const updated: AccountState = {
      ...existing,
      ...update,
    };

    const key = didToKey(did);
    const value = serializeAccountState(updated);
    const newRoot = this.smt.set(key, value);

    this.cacheAccount(did, updated);
    return newRoot;
  }

  /**
   * Rotate the operational key for an account.
   * Updates the key hash and increments the version.
   */
  rotateKey(did: string, newOperationalKeyHash: string): MerkleRoot {
    const existing = this.getAccount(did);
    if (!existing) {
      throw new StateError("Account not found", "ACCOUNT_NOT_FOUND", { did });
    }

    if (existing.flags.frozen) {
      throw new StateError("Account is frozen", "ACCOUNT_FROZEN", { did });
    }

    if (existing.flags.inRecovery) {
      throw new StateError(
        "Account is in recovery; use recovery flow instead",
        "ACCOUNT_IN_RECOVERY",
        { did },
      );
    }

    if (newOperationalKeyHash === existing.operationalKeyHash) {
      throw new StateError(
        "New key must differ from current key",
        "INVALID_TRANSITION",
        { did },
      );
    }

    return this.updateAccount(did, {
      operationalKeyHash: newOperationalKeyHash,
      operationalKeyVersion: existing.operationalKeyVersion + 1,
    });
  }

  /**
   * Apply a delta to an account
   */
  applyDelta(delta: AccountDelta): MerkleRoot {
    const existing = this.getAccount(delta.did);
    if (!existing) {
      throw new StateError("Account not found", "ACCOUNT_NOT_FOUND", {
        did: delta.did,
      });
    }

    let newBalance = existing.balance;
    if (delta.balanceChange !== undefined) {
      newBalance = existing.balance + delta.balanceChange;
      if (newBalance < 0n) {
        throw new StateError("Insufficient balance", "INSUFFICIENT_BALANCE", {
          did: delta.did,
          balance: existing.balance.toString(),
          requested: (-delta.balanceChange).toString(),
        });
      }
    }

    let newStaked = existing.stakedBalance;
    if (delta.stakedChange !== undefined) {
      newStaked = existing.stakedBalance + delta.stakedChange;
      if (newStaked < 0n) {
        throw new StateError(
          "Insufficient staked balance",
          "INSUFFICIENT_BALANCE",
          {
            did: delta.did,
            staked: existing.stakedBalance.toString(),
            requested: (-delta.stakedChange).toString(),
          },
        );
      }
    }

    const updated: AccountState = {
      ...existing,
      balance: newBalance,
      stakedBalance: newStaked,
      nonce: delta.nonceIncrement ? existing.nonce + 1n : existing.nonce,
      reputationScore:
        delta.reputationChange !== undefined
          ? Math.max(
              0,
              Math.min(1, existing.reputationScore + delta.reputationChange),
            )
          : existing.reputationScore,
      flags: delta.setFlags
        ? { ...existing.flags, ...delta.setFlags }
        : existing.flags,
    };

    const key = didToKey(delta.did);
    const value = serializeAccountState(updated);
    const newRoot = this.smt.set(key, value);

    this.cacheAccount(delta.did, updated);
    return newRoot;
  }

  /**
   * Transfer balance between accounts
   */
  transfer(from: string, to: string, amount: bigint): MerkleRoot {
    if (amount <= 0n) {
      throw new StateError("Invalid transfer amount", "INVALID_TRANSITION", {
        amount: amount.toString(),
      });
    }

    const fromAccount = this.getAccount(from);
    if (!fromAccount) {
      throw new StateError("Sender not found", "ACCOUNT_NOT_FOUND", {
        did: from,
      });
    }

    if (fromAccount.flags.frozen) {
      throw new StateError("Sender account is frozen", "INVALID_TRANSITION", {
        did: from,
      });
    }

    if (fromAccount.balance < amount) {
      throw new StateError("Insufficient balance", "INSUFFICIENT_BALANCE", {
        did: from,
        balance: fromAccount.balance.toString(),
        requested: amount.toString(),
      });
    }

    // Create recipient if doesn't exist
    let toAccount = this.getAccount(to);
    if (!toAccount) {
      // For transfers to new accounts, create with minimal state
      this.createAccount(to, "", 0n);
      toAccount = this.getAccount(to)!;
    }

    if (toAccount.flags.frozen) {
      throw new StateError(
        "Recipient account is frozen",
        "INVALID_TRANSITION",
        { did: to },
      );
    }

    // Apply deltas
    this.applyDelta({
      did: from,
      balanceChange: -amount,
      nonceIncrement: true,
    });
    return this.applyDelta({ did: to, balanceChange: amount });
  }

  /**
   * Deduct balance without incrementing nonce
   * Used for fee routing where nonce should not change
   */
  deductBalance(did: string, amount: bigint): MerkleRoot {
    if (amount <= 0n) {
      throw new StateError("Invalid deduct amount", "INVALID_TRANSITION", {
        amount: amount.toString(),
      });
    }

    const account = this.getAccount(did);
    if (!account) {
      throw new StateError("Account not found", "ACCOUNT_NOT_FOUND", { did });
    }

    if (account.balance < amount) {
      throw new StateError("Insufficient balance", "INSUFFICIENT_BALANCE", {
        did,
        balance: account.balance.toString(),
        requested: amount.toString(),
      });
    }

    return this.applyDelta({ did, balanceChange: -amount });
  }

  /**
   * Credit balance without incrementing nonce
   * Used for fee routing where nonce should not change
   * Auto-creates recipient if it doesn't exist
   */
  creditBalance(did: string, amount: bigint): MerkleRoot {
    if (amount <= 0n) {
      throw new StateError("Invalid credit amount", "INVALID_TRANSITION", {
        amount: amount.toString(),
      });
    }

    // Create recipient if doesn't exist
    if (!this.exists(did)) {
      this.createAccount(did, "", 0n);
    }

    return this.applyDelta({ did, balanceChange: amount });
  }

  /**
   * Stake balance (move from balance to stakedBalance)
   */
  stake(did: string, amount: bigint): MerkleRoot {
    if (amount <= 0n) {
      throw new StateError("Invalid stake amount", "INVALID_TRANSITION", {
        amount: amount.toString(),
      });
    }

    const account = this.getAccount(did);
    if (!account) {
      throw new StateError("Account not found", "ACCOUNT_NOT_FOUND", { did });
    }

    if (account.balance < amount) {
      throw new StateError(
        "Insufficient balance to stake",
        "INSUFFICIENT_BALANCE",
        {
          did,
          balance: account.balance.toString(),
          requested: amount.toString(),
        },
      );
    }

    return this.applyDelta({
      did,
      balanceChange: -amount,
      stakedChange: amount,
      nonceIncrement: true,
    });
  }

  /**
   * Unstake balance (move from stakedBalance to balance)
   * Note: In practice, unstaking has a timelock
   */
  unstake(did: string, amount: bigint): MerkleRoot {
    if (amount <= 0n) {
      throw new StateError("Invalid unstake amount", "INVALID_TRANSITION", {
        amount: amount.toString(),
      });
    }

    const account = this.getAccount(did);
    if (!account) {
      throw new StateError("Account not found", "ACCOUNT_NOT_FOUND", { did });
    }

    if (account.stakedBalance < amount) {
      throw new StateError(
        "Insufficient staked balance",
        "INSUFFICIENT_BALANCE",
        {
          did,
          staked: account.stakedBalance.toString(),
          requested: amount.toString(),
        },
      );
    }

    return this.applyDelta({
      did,
      balanceChange: amount,
      stakedChange: -amount,
      nonceIncrement: true,
    });
  }

  /**
   * Increment account nonce without balance change
   * Used for transactions that don't transfer/stake but still need nonce protection
   */
  incrementNonce(did: string): MerkleRoot {
    const account = this.getAccount(did);
    if (!account) {
      throw new StateError("Account not found", "ACCOUNT_NOT_FOUND", { did });
    }

    return this.applyDelta({
      did,
      nonceIncrement: true,
    });
  }

  /**
   * Generate proof of account state
   */
  proveAccount(did: string): MerkleProofData {
    const key = didToKey(did);
    return this.smt.prove(key);
  }

  /**
   * Generate proof of balance
   */
  proveBalance(did: string, blockNumber: bigint): BalanceProof {
    const account = this.getAccount(did);
    const proof = this.proveAccount(did);

    return {
      account: did,
      balance: account?.balance ?? 0n,
      stateRoot: this.getRoot(),
      blockNumber,
      proof,
    };
  }

  /**
   * Verify a balance proof
   */
  static verifyBalanceProof(proof: BalanceProof): boolean {
    // Verify the Merkle proof
    const validProof = InMemorySMT.verify(proof.proof);
    if (!validProof) return false;

    // If account exists, verify balance matches
    if (proof.proof.value) {
      const account = deserializeAccountState(proof.proof.value);
      return account.balance === proof.balance;
    }

    // Account doesn't exist, balance should be 0
    return proof.balance === 0n;
  }

  /**
   * Get all accounts (for snapshot/debugging)
   * Note: In production, this would iterate the SMT
   */
  getAllAccounts(): AccountState[] {
    return Array.from(this.accountCache.values());
  }

  /**
   * Get total supply (sum of all balances + staked)
   */
  getTotalSupply(): bigint {
    let total = 0n;
    for (const account of this.accountCache.values()) {
      total += account.balance + account.stakedBalance;
    }
    return total;
  }

  /**
   * Batch update multiple accounts atomically
   */
  batchUpdate(deltas: AccountDelta[]): MerkleRoot {
    let root: MerkleRoot = this.getRoot();
    for (const delta of deltas) {
      root = this.applyDelta(delta);
    }
    return root;
  }

  /**
   * Clone the tree (for speculative execution)
   */
  clone(): AccountStateTree {
    const cloned = new AccountStateTree(this.cacheSize);
    // Deep clone cache
    for (const [did, account] of this.accountCache) {
      cloned.accountCache.set(did, { ...account, flags: { ...account.flags } });
      // Re-create in SMT
      const key = didToKey(did);
      cloned.smt.set(key, serializeAccountState(account));
    }
    return cloned;
  }

  // ============ Private Methods ============

  private cacheAccount(did: string, account: AccountState): void {
    // Simple LRU eviction
    if (this.accountCache.size >= this.cacheSize) {
      const firstKey = this.accountCache.keys().next().value;
      if (firstKey) this.accountCache.delete(firstKey);
    }
    this.accountCache.set(did, account);
  }
}

// ============ Factory Functions ============

/**
 * Create account tree from snapshot
 */
export function createAccountTreeFromSnapshot(
  accounts: AccountState[],
): AccountStateTree {
  const tree = new AccountStateTree();

  for (const account of accounts) {
    const key = didToKey(account.did);
    const value = serializeAccountState(account);
    tree["smt"].set(key, value);
    tree["accountCache"].set(account.did, account);
  }

  return tree;
}

/**
 * Default initial account state
 */
export function defaultAccountState(did: string): AccountState {
  return {
    did,
    balance: 0n,
    stakedBalance: 0n,
    nonce: 0n,
    operationalKeyHash: "",
    operationalKeyVersion: 0,
    sessionKeyCount: 0,
    reputationScore: 0.5,
    lastActiveBlock: 0n,
    flags: {
      isValidator: false,
      canSubmitClaims: true,
      canChallenge: true,
      inRecovery: false,
      frozen: false,
    },
  };
}
