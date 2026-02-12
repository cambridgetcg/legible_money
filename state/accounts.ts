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
 * - Per-period balance tracking for token expiration (Dynamic Array Method)
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
    createdAtBlock: account.createdAtBlock.toString(),
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
    periodBalances: account.periodBalances
      ? Array.from(account.periodBalances.entries()).map(([k, v]) => [
          k,
          v.toString(),
        ])
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
    createdAtBlock: obj.createdAtBlock ? BigInt(obj.createdAtBlock) : 0n,
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
    periodBalances: obj.periodBalances
      ? new Map(
          (obj.periodBalances as Array<[number, string]>).map(([k, v]) => [
            k,
            BigInt(v),
          ]),
        )
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

  // Period tracking state
  private currentPeriod: number | undefined = undefined;
  private allDids: Set<string> = new Set();
  // Backing store for all accounts (survives cache eviction, used by getAllAccounts/sweep)
  private accountStore: Map<string, AccountState> = new Map();

  constructor(cacheSize: number = 10000) {
    this.smt = new InMemorySMT();
    this.cacheSize = cacheSize;
  }

  // ============ Period Tracking ============

  /**
   * Activate/deactivate period tracking.
   * When set, credits go to currentPeriod, debits use FIFO from oldest period.
   */
  setCurrentPeriod(period: number | undefined): void {
    this.currentPeriod = period;
  }

  /**
   * Get the current period (undefined if tracking is inactive).
   */
  getCurrentPeriod(): number | undefined {
    return this.currentPeriod;
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

    // Check backing store (survives cache eviction)
    const stored = this.accountStore.get(did);
    if (stored) {
      this.cacheAccount(did, stored);
      return stored;
    }

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
    blockNumber: bigint = 0n,
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
      createdAtBlock: blockNumber,
      lastActiveBlock: blockNumber,
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
    this.allDids.add(did);
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
   * Apply a delta to an account.
   * When period tracking is active, also updates periodBalances via lazy migration + FIFO.
   */
  applyDelta(delta: AccountDelta): MerkleRoot {
    const existing = this.getAccount(delta.did);
    if (!existing) {
      throw new StateError("Account not found", "ACCOUNT_NOT_FOUND", {
        did: delta.did,
      });
    }

    let newBalance = existing.balance;
    let periodBalances = existing.periodBalances;

    if (delta.balanceChange !== undefined) {
      newBalance = existing.balance + delta.balanceChange;
      if (newBalance < 0n) {
        throw new StateError("Insufficient balance", "INSUFFICIENT_BALANCE", {
          did: delta.did,
          balance: existing.balance.toString(),
          requested: (-delta.balanceChange).toString(),
        });
      }

      // Period tracking logic
      if (this.currentPeriod !== undefined) {
        // Lazy migration: if no periodBalances yet, migrate existing balance
        if (!periodBalances) {
          periodBalances = new Map<number, bigint>();
          if (existing.balance > 0n) {
            periodBalances.set(this.currentPeriod, existing.balance);
          }
        }

        if (delta.balanceChange > 0n) {
          // Credit: add to currentPeriod
          const current = periodBalances.get(this.currentPeriod) ?? 0n;
          periodBalances.set(this.currentPeriod, current + delta.balanceChange);
        } else if (delta.balanceChange < 0n) {
          // Debit: FIFO from oldest period
          let remaining = -delta.balanceChange;
          const sortedPeriods = Array.from(periodBalances.keys()).sort(
            (a, b) => a - b,
          );
          for (const period of sortedPeriods) {
            if (remaining <= 0n) break;
            const periodBal = periodBalances.get(period)!;
            if (periodBal <= remaining) {
              remaining -= periodBal;
              periodBalances.delete(period);
            } else {
              periodBalances.set(period, periodBal - remaining);
              remaining = 0n;
            }
          }
        }
      }
    }

    // Handle staking: when tokens are staked, debit from periodBalances (FIFO)
    // When tokens are unstaked, credit to currentPeriod (fresh expiration)
    if (
      delta.stakedChange !== undefined &&
      this.currentPeriod !== undefined &&
      delta.balanceChange !== undefined
    ) {
      // Staking and unstaking are handled through the balanceChange path above
      // (stake: balanceChange < 0, unstake: balanceChange > 0)
      // No additional period logic needed — applyDelta already handled it.
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
      periodBalances,
    };

    const key = didToKey(delta.did);
    const value = serializeAccountState(updated);
    const newRoot = this.smt.set(key, value);

    this.cacheAccount(delta.did, updated);
    return newRoot;
  }

  /**
   * Transfer balance between accounts.
   * Sender uses FIFO debit; recipient gets tokens at currentPeriod (fresh expiration).
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

    // H4: Self-transfer detection — only increment nonce, no balance change
    if (from === to) {
      return this.applyDelta({
        did: from,
        balanceChange: 0n,
        nonceIncrement: true,
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

    // Apply deltas — sender FIFO debit, recipient credit at currentPeriod
    this.applyDelta({
      did: from,
      balanceChange: -amount,
      nonceIncrement: true,
    });
    return this.applyDelta({ did: to, balanceChange: amount });
  }

  /**
   * Transfer preserving the sender's period-of-origin in the recipient.
   * Prevents ping-pong refresh attacks: tokens keep their original expiration.
   *
   * @param periodsUntilExpiration If provided, rejects transfer if FIFO includes expired periods
   */
  transferPreservingPeriods(
    from: string,
    to: string,
    amount: bigint,
    periodsUntilExpiration?: number,
  ): MerkleRoot {
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

    if (fromAccount.balance < amount) {
      throw new StateError("Insufficient balance", "INSUFFICIENT_BALANCE", {
        did: from,
        balance: fromAccount.balance.toString(),
        requested: amount.toString(),
      });
    }

    // If period tracking is inactive, fall back to regular transfer
    if (this.currentPeriod === undefined || !fromAccount.periodBalances) {
      return this.transfer(from, to, amount);
    }

    // Create recipient if doesn't exist
    if (!this.exists(to)) {
      this.createAccount(to, "", 0n);
    }

    // Compute FIFO debit plan from sender
    const debitPlan: Array<[number, bigint]> = [];
    let remaining = amount;
    const sortedPeriods = Array.from(fromAccount.periodBalances.keys()).sort(
      (a, b) => a - b,
    );

    // Expiration threshold: periods <= threshold are expired
    const expirationThreshold =
      periodsUntilExpiration !== undefined
        ? this.currentPeriod - periodsUntilExpiration
        : undefined;

    for (const period of sortedPeriods) {
      if (remaining <= 0n) break;

      // Check if this period is expired
      if (expirationThreshold !== undefined && period <= expirationThreshold) {
        throw new StateError(
          `Cannot transfer tokens from expired period ${period}`,
          "TOKENS_EXPIRED",
          {
            period: period.toString(),
            threshold: expirationThreshold.toString(),
          },
        );
      }

      const periodBal = fromAccount.periodBalances.get(period)!;
      if (periodBal <= remaining) {
        debitPlan.push([period, periodBal]);
        remaining -= periodBal;
      } else {
        debitPlan.push([period, remaining]);
        remaining = 0n;
      }
    }

    // Apply sender debit (FIFO from periodBalances)
    const senderPB = new Map(fromAccount.periodBalances);
    for (const [period, amt] of debitPlan) {
      const current = senderPB.get(period)!;
      if (current === amt) {
        senderPB.delete(period);
      } else {
        senderPB.set(period, current - amt);
      }
    }

    // Update sender
    this.updateAccount(from, {
      balance: fromAccount.balance - amount,
      nonce: fromAccount.nonce + 1n,
      periodBalances: senderPB,
    });

    // Apply recipient credit (preserving sender's period-of-origin)
    const toAccount = this.getAccount(to)!;
    let recipientPB = toAccount.periodBalances;

    // Lazy migration for recipient
    if (!recipientPB && this.currentPeriod !== undefined) {
      recipientPB = new Map<number, bigint>();
      if (toAccount.balance > 0n) {
        recipientPB.set(this.currentPeriod, toAccount.balance);
      }
    }

    if (recipientPB) {
      for (const [period, amt] of debitPlan) {
        const existing = recipientPB.get(period) ?? 0n;
        recipientPB.set(period, existing + amt);
      }
    }

    this.updateAccount(to, {
      balance: toAccount.balance + amount,
      periodBalances: recipientPB,
    });

    return this.getRoot();
  }

  /**
   * Credit balance to a specific period (not currentPeriod).
   * Used for unbonding release to preserve original expiration.
   */
  creditBalanceToSpecificPeriod(
    did: string,
    amount: bigint,
    period: number,
  ): MerkleRoot {
    if (amount <= 0n) {
      throw new StateError("Invalid credit amount", "INVALID_TRANSITION", {
        amount: amount.toString(),
      });
    }

    // Create recipient if doesn't exist
    if (!this.exists(did)) {
      this.createAccount(did, "", 0n);
    }

    const account = this.getAccount(did)!;

    // If period tracking is inactive, just credit normally
    if (this.currentPeriod === undefined) {
      return this.updateAccount(did, {
        balance: account.balance + amount,
      });
    }

    // Lazy migration
    let periodBalances = account.periodBalances;
    if (!periodBalances) {
      periodBalances = new Map<number, bigint>();
      if (account.balance > 0n) {
        periodBalances.set(this.currentPeriod, account.balance);
      }
    }

    // Credit to the specific period
    const existing = periodBalances.get(period) ?? 0n;
    periodBalances.set(period, existing + amount);

    return this.updateAccount(did, {
      balance: account.balance + amount,
      periodBalances,
    });
  }

  /**
   * Sweep expired periods from an account.
   * Removes all period entries <= threshold, deducts from balance, returns total swept.
   */
  sweepExpiredPeriods(did: string, threshold: number): bigint {
    const account = this.getAccount(did);
    if (!account || !account.periodBalances) return 0n;

    let swept = 0n;
    const periodBalances = new Map(account.periodBalances);

    for (const [period, balance] of account.periodBalances) {
      if (period <= threshold) {
        swept += balance;
        periodBalances.delete(period);
      }
    }

    if (swept > 0n) {
      this.updateAccount(did, {
        balance: account.balance - swept,
        periodBalances,
      });
    }

    return swept;
  }

  /**
   * Find all accounts with expired period balances.
   * Scans ALL known DIDs (not just cache).
   */
  getAccountsWithExpiredPeriods(
    threshold: number,
  ): Array<{ did: string; totalExpired: bigint }> {
    const results: Array<{ did: string; totalExpired: bigint }> = [];

    for (const did of this.allDids) {
      const account = this.getAccount(did);
      if (!account || !account.periodBalances) continue;

      let totalExpired = 0n;
      for (const [period, balance] of account.periodBalances) {
        if (period <= threshold) {
          totalExpired += balance;
        }
      }

      if (totalExpired > 0n) {
        results.push({ did, totalExpired });
      }
    }

    return results;
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
   * Get all accounts (uses allDids index for completeness beyond cache)
   */
  getAllAccounts(): AccountState[] {
    const results: AccountState[] = [];
    for (const did of this.allDids) {
      const account = this.getAccount(did);
      if (account) results.push(account);
    }
    return results;
  }

  /**
   * Get total supply (sum of all balances + staked)
   */
  getTotalSupply(): bigint {
    let total = 0n;
    for (const did of this.allDids) {
      const account = this.getAccount(did);
      if (account) {
        total += account.balance + account.stakedBalance;
      }
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
    cloned.currentPeriod = this.currentPeriod;
    cloned.allDids = new Set(this.allDids);
    // Deep clone backing store
    for (const [did, account] of this.accountStore) {
      const clonedAccount: AccountState = {
        ...account,
        flags: { ...account.flags },
        periodBalances: account.periodBalances
          ? new Map(account.periodBalances)
          : undefined,
        availability: account.availability
          ? { ...account.availability }
          : undefined,
      };
      cloned.accountStore.set(did, clonedAccount);
    }
    // Deep clone cache
    for (const [did, account] of this.accountCache) {
      const clonedAccount: AccountState = {
        ...account,
        flags: { ...account.flags },
        periodBalances: account.periodBalances
          ? new Map(account.periodBalances)
          : undefined,
        availability: account.availability
          ? { ...account.availability }
          : undefined,
      };
      cloned.accountCache.set(did, clonedAccount);
      // Re-create in SMT
      const key = didToKey(did);
      cloned.smt.set(key, serializeAccountState(clonedAccount));
    }
    // Also reconstruct any accounts in allDids that were evicted from cache
    for (const did of this.allDids) {
      if (!cloned.accountCache.has(did)) {
        const account = this.getAccount(did);
        if (account) {
          const clonedAccount: AccountState = {
            ...account,
            flags: { ...account.flags },
            periodBalances: account.periodBalances
              ? new Map(account.periodBalances)
              : undefined,
            availability: account.availability
              ? { ...account.availability }
              : undefined,
          };
          cloned.accountCache.set(did, clonedAccount);
          const key = didToKey(did);
          cloned.smt.set(key, serializeAccountState(clonedAccount));
        }
      }
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
    this.accountStore.set(did, account);
    this.allDids.add(did);
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
    tree["accountStore"].set(account.did, account);
    tree["allDids"].add(account.did);
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
    createdAtBlock: 0n,
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
