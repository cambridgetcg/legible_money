/**
 * Token Registry State Tree
 *
 * Manages secondary token state: definitions, balances, and allowances.
 * All token data is stored in Maps for efficient lookup.
 * Production: Will be backed by SMT for Merkle proofs.
 *
 * Key design: Balances keyed by "did:tokenId", allowances by "owner:spender:tokenId".
 */

import { sha512_256Hex } from "../core/crypto/hashing.js";
import type {
  TokenDefinition,
  SecondaryTokenType,
  VoteDelegation,
  SecondaryStakePosition,
  LiquidityPoolState,
  AgentServiceRegistration,
  TokenSubscription,
} from "../core/tokens/secondary/types.js";
import { SecondaryTokenError } from "../core/tokens/secondary/types.js";

// ============ Key Helpers ============

function balanceKey(did: string, tokenId: string): string {
  return `${did}:${tokenId}`;
}

function allowanceKey(owner: string, spender: string, tokenId: string): string {
  return `${owner}:${spender}:${tokenId}`;
}

function delegationKey(delegator: string, tokenId: string): string {
  return `${delegator}:${tokenId}`;
}

function stakeKey(staker: string, tokenId: string): string {
  return `stake:${staker}:${tokenId}`;
}

// ============ Token Registry ============

export class TokenRegistryStateTree {
  /** Token definitions by tokenId */
  private definitions: Map<string, TokenDefinition> = new Map();

  /** Token balances by "did:tokenId" */
  private balances: Map<string, bigint> = new Map();

  /** Token allowances by "owner:spender:tokenId" */
  private allowances: Map<string, bigint> = new Map();

  /** Vote delegations by "delegator:tokenId" */
  private delegations: Map<string, VoteDelegation> = new Map();

  /** Voting power snapshots: "tokenId:block" -> Map<did, power> */
  private snapshots: Map<string, Map<string, bigint>> = new Map();

  /** Staking positions by "stake:staker:tokenId" */
  private stakes: Map<string, SecondaryStakePosition> = new Map();

  /** Liquidity pools by poolId */
  private pools: Map<string, LiquidityPoolState> = new Map();

  /** Service registrations by serviceId */
  private services: Map<string, AgentServiceRegistration> = new Map();

  /** Token subscriptions by "subscriber:tokenId" */
  private subscriptions: Map<string, TokenSubscription> = new Map();

  /** Symbols in use per creator: "creator:symbol" -> tokenId */
  private symbolIndex: Map<string, string> = new Map();

  // ============ Token Definition CRUD ============

  /**
   * Generate deterministic token ID
   */
  generateTokenId(creator: string, name: string, salt: string): string {
    return sha512_256Hex(`LGM.token.id.v1:${creator}:${name}:${salt}`).slice(
      0,
      32,
    );
  }

  /**
   * Register a new token definition
   */
  createToken(definition: TokenDefinition): void {
    if (this.definitions.has(definition.tokenId)) {
      throw new SecondaryTokenError(
        `Token ${definition.tokenId} already exists`,
        "TOKEN_ALREADY_EXISTS",
      );
    }

    const symbolKey = `${definition.creator}:${definition.symbol}`;
    if (this.symbolIndex.has(symbolKey)) {
      throw new SecondaryTokenError(
        `Symbol ${definition.symbol} already used by creator`,
        "DUPLICATE_SYMBOL",
      );
    }

    this.definitions.set(definition.tokenId, { ...definition });
    this.symbolIndex.set(symbolKey, definition.tokenId);
  }

  /**
   * Get token definition by ID
   */
  getTokenDefinition(tokenId: string): TokenDefinition | undefined {
    const def = this.definitions.get(tokenId);
    return def ? { ...def } : undefined;
  }

  /**
   * Update token metadata
   */
  updateTokenMetadata(
    tokenId: string,
    metadata: Record<string, unknown>,
  ): void {
    const def = this.definitions.get(tokenId);
    if (!def) {
      throw new SecondaryTokenError(
        `Token ${tokenId} not found`,
        "TOKEN_NOT_FOUND",
      );
    }
    def.metadata = { ...def.metadata, ...metadata };
  }

  /**
   * Pause a token
   */
  pauseToken(tokenId: string): void {
    const def = this.definitions.get(tokenId);
    if (!def) {
      throw new SecondaryTokenError(
        `Token ${tokenId} not found`,
        "TOKEN_NOT_FOUND",
      );
    }
    def.paused = true;
  }

  /**
   * Unpause a token
   */
  unpauseToken(tokenId: string): void {
    const def = this.definitions.get(tokenId);
    if (!def) {
      throw new SecondaryTokenError(
        `Token ${tokenId} not found`,
        "TOKEN_NOT_FOUND",
      );
    }
    def.paused = false;
  }

  /**
   * List all tokens
   */
  listTokens(): TokenDefinition[] {
    return Array.from(this.definitions.values()).map((d) => ({ ...d }));
  }

  /**
   * Query tokens by type
   */
  getTokensByType(tokenType: SecondaryTokenType): TokenDefinition[] {
    return this.listTokens().filter((t) => t.tokenType === tokenType);
  }

  // ============ Balance Operations ============

  /**
   * Get token balance for an account
   */
  getTokenBalance(did: string, tokenId: string): bigint {
    return this.balances.get(balanceKey(did, tokenId)) ?? 0n;
  }

  /**
   * Set token balance directly (used by mint/burn/transfer)
   */
  setTokenBalance(did: string, tokenId: string, balance: bigint): void {
    const key = balanceKey(did, tokenId);
    if (balance === 0n) {
      this.balances.delete(key);
    } else {
      this.balances.set(key, balance);
    }
  }

  /**
   * Get all token balances for an account
   */
  getAllTokenBalances(
    did: string,
  ): Array<{ tokenId: string; balance: bigint }> {
    const result: Array<{ tokenId: string; balance: bigint }> = [];
    for (const [key, balance] of this.balances) {
      if (key.startsWith(`${did}:`)) {
        const tokenId = key.slice(did.length + 1);
        result.push({ tokenId, balance });
      }
    }
    return result;
  }

  /**
   * Mint tokens to an account
   */
  mint(tokenId: string, to: string, amount: bigint): void {
    const def = this.definitions.get(tokenId);
    if (!def) {
      throw new SecondaryTokenError(
        `Token ${tokenId} not found`,
        "TOKEN_NOT_FOUND",
      );
    }
    if (def.paused) {
      throw new SecondaryTokenError(
        `Token ${tokenId} is paused`,
        "TOKEN_PAUSED",
      );
    }

    if (def.maxSupply > 0n && def.totalSupply + amount > def.maxSupply) {
      throw new SecondaryTokenError(
        `Minting ${amount} would exceed max supply ${def.maxSupply}`,
        "MAX_SUPPLY_EXCEEDED",
      );
    }

    const currentBalance = this.getTokenBalance(to, tokenId);
    this.setTokenBalance(to, tokenId, currentBalance + amount);
    def.totalSupply += amount;
  }

  /**
   * Burn tokens from an account
   */
  burn(tokenId: string, from: string, amount: bigint): void {
    const def = this.definitions.get(tokenId);
    if (!def) {
      throw new SecondaryTokenError(
        `Token ${tokenId} not found`,
        "TOKEN_NOT_FOUND",
      );
    }
    if (def.paused) {
      throw new SecondaryTokenError(
        `Token ${tokenId} is paused`,
        "TOKEN_PAUSED",
      );
    }

    const currentBalance = this.getTokenBalance(from, tokenId);
    if (currentBalance < amount) {
      throw new SecondaryTokenError(
        `Insufficient token balance: has ${currentBalance}, needs ${amount}`,
        "INSUFFICIENT_TOKEN_BALANCE",
      );
    }

    this.setTokenBalance(from, tokenId, currentBalance - amount);
    def.totalSupply -= amount;
  }

  /**
   * Transfer tokens between accounts
   */
  transferToken(
    tokenId: string,
    from: string,
    to: string,
    amount: bigint,
  ): void {
    if (amount === 0n) {
      throw new SecondaryTokenError(
        "Cannot transfer 0 amount",
        "INVALID_AMOUNT",
      );
    }
    if (from === to) {
      throw new SecondaryTokenError("Cannot transfer to self", "SELF_TRANSFER");
    }

    const def = this.definitions.get(tokenId);
    if (!def) {
      throw new SecondaryTokenError(
        `Token ${tokenId} not found`,
        "TOKEN_NOT_FOUND",
      );
    }
    if (def.paused) {
      throw new SecondaryTokenError(
        `Token ${tokenId} is paused`,
        "TOKEN_PAUSED",
      );
    }

    const fromBalance = this.getTokenBalance(from, tokenId);
    if (fromBalance < amount) {
      throw new SecondaryTokenError(
        `Insufficient token balance: has ${fromBalance}, needs ${amount}`,
        "INSUFFICIENT_TOKEN_BALANCE",
      );
    }

    this.setTokenBalance(from, tokenId, fromBalance - amount);
    const toBalance = this.getTokenBalance(to, tokenId);
    this.setTokenBalance(to, tokenId, toBalance + amount);
  }

  // ============ Allowance Operations ============

  /**
   * Set spending allowance
   */
  setAllowance(
    owner: string,
    spender: string,
    tokenId: string,
    amount: bigint,
  ): void {
    const key = allowanceKey(owner, spender, tokenId);
    if (amount === 0n) {
      this.allowances.delete(key);
    } else {
      this.allowances.set(key, amount);
    }
  }

  /**
   * Get spending allowance
   */
  getAllowance(owner: string, spender: string, tokenId: string): bigint {
    return this.allowances.get(allowanceKey(owner, spender, tokenId)) ?? 0n;
  }

  /**
   * Increase allowance atomically (mitigates ERC-20 front-running)
   */
  increaseAllowance(
    owner: string,
    spender: string,
    tokenId: string,
    addedAmount: bigint,
  ): void {
    const current = this.getAllowance(owner, spender, tokenId);
    this.setAllowance(owner, spender, tokenId, current + addedAmount);
  }

  /**
   * Decrease allowance atomically (mitigates ERC-20 front-running)
   */
  decreaseAllowance(
    owner: string,
    spender: string,
    tokenId: string,
    subtractedAmount: bigint,
  ): void {
    const current = this.getAllowance(owner, spender, tokenId);
    if (current < subtractedAmount) {
      throw new SecondaryTokenError(
        `Cannot decrease allowance below 0: current ${current}, decrease ${subtractedAmount}`,
        "INSUFFICIENT_ALLOWANCE",
      );
    }
    this.setAllowance(owner, spender, tokenId, current - subtractedAmount);
  }

  /**
   * Transfer from allowance (transferFrom)
   */
  transferFrom(
    tokenId: string,
    spender: string,
    from: string,
    to: string,
    amount: bigint,
  ): void {
    const allowed = this.getAllowance(from, spender, tokenId);
    if (allowed < amount) {
      throw new SecondaryTokenError(
        `Insufficient allowance: allowed ${allowed}, requested ${amount}`,
        "INSUFFICIENT_ALLOWANCE",
      );
    }

    this.transferToken(tokenId, from, to, amount);
    this.setAllowance(from, spender, tokenId, allowed - amount);
  }

  // ============ Total Supply ============

  /**
   * Get total supply of a token
   */
  getTokenTotalSupply(tokenId: string): bigint {
    const def = this.definitions.get(tokenId);
    return def ? def.totalSupply : 0n;
  }

  // ============ Governance Delegation ============

  /**
   * Delegate voting power
   */
  delegateVotingPower(
    delegator: string,
    delegate: string,
    tokenId: string,
    block: bigint,
  ): void {
    // Prevent circular delegation
    const delegatesDelegation = this.delegations.get(
      delegationKey(delegate, tokenId),
    );
    if (delegatesDelegation && delegatesDelegation.delegate === delegator) {
      throw new SecondaryTokenError(
        "Circular delegation detected",
        "CIRCULAR_DELEGATION",
      );
    }

    this.delegations.set(delegationKey(delegator, tokenId), {
      delegator,
      delegate,
      tokenId,
      delegatedAtBlock: block,
    });
  }

  /**
   * Remove vote delegation
   */
  undelegateVotingPower(delegator: string, tokenId: string): void {
    this.delegations.delete(delegationKey(delegator, tokenId));
  }

  /**
   * Get delegation for an account
   */
  getDelegation(
    delegator: string,
    tokenId: string,
  ): VoteDelegation | undefined {
    return this.delegations.get(delegationKey(delegator, tokenId));
  }

  /**
   * Get voting power (own balance + delegated power)
   */
  getVotingPower(did: string, tokenId: string): bigint {
    let power = this.getTokenBalance(did, tokenId);

    // Subtract if delegated away
    const myDelegation = this.delegations.get(delegationKey(did, tokenId));
    if (myDelegation) {
      power = 0n; // All voting power delegated
    }

    // Add delegated-to-me
    for (const delegation of this.delegations.values()) {
      if (delegation.tokenId === tokenId && delegation.delegate === did) {
        power += this.getTokenBalance(delegation.delegator, tokenId);
      }
    }

    return power;
  }

  /**
   * Take a snapshot of voting power at a block
   */
  takeVotingPowerSnapshot(tokenId: string, block: bigint): void {
    const powers = new Map<string, bigint>();
    const holders = new Set<string>();

    // Collect all holders
    for (const [key] of this.balances) {
      if (key.endsWith(`:${tokenId}`)) {
        const did = key.slice(0, key.length - tokenId.length - 1);
        holders.add(did);
      }
    }

    // Also add delegates who receive power
    for (const delegation of this.delegations.values()) {
      if (delegation.tokenId === tokenId) {
        holders.add(delegation.delegate);
      }
    }

    for (const did of holders) {
      const power = this.getVotingPower(did, tokenId);
      if (power > 0n) {
        powers.set(did, power);
      }
    }

    this.snapshots.set(`${tokenId}:${block}`, powers);
  }

  /**
   * Get voting power from a snapshot
   */
  getSnapshotVotingPower(tokenId: string, block: bigint, did: string): bigint {
    const snapshot = this.snapshots.get(`${tokenId}:${block}`);
    return snapshot?.get(did) ?? 0n;
  }

  // ============ Staking ============

  /**
   * Stake secondary tokens
   */
  stakeSecondaryToken(
    staker: string,
    tokenId: string,
    amount: bigint,
    block: bigint,
  ): void {
    const balance = this.getTokenBalance(staker, tokenId);
    if (balance < amount) {
      throw new SecondaryTokenError(
        `Insufficient token balance for staking`,
        "INSUFFICIENT_TOKEN_BALANCE",
      );
    }

    // Lock tokens (deduct from balance)
    this.setTokenBalance(staker, tokenId, balance - amount);

    const key = stakeKey(staker, tokenId);
    const existing = this.stakes.get(key);
    if (existing) {
      existing.stakedAmount += amount;
    } else {
      this.stakes.set(key, {
        staker,
        tokenId,
        stakedAmount: amount,
        stakedAtBlock: block,
        accumulatedRewards: 0n,
        lastRewardBlock: block,
      });
    }
  }

  /**
   * Unstake secondary tokens
   */
  unstakeSecondaryToken(staker: string, tokenId: string): bigint {
    const key = stakeKey(staker, tokenId);
    const position = this.stakes.get(key);
    if (!position) {
      throw new SecondaryTokenError(
        "No staking position found",
        "INSUFFICIENT_STAKE",
      );
    }

    const amount = position.stakedAmount;
    const currentBalance = this.getTokenBalance(staker, tokenId);
    this.setTokenBalance(staker, tokenId, currentBalance + amount);
    this.stakes.delete(key);

    return amount;
  }

  /**
   * Get staking position
   */
  getStakePosition(
    staker: string,
    tokenId: string,
  ): SecondaryStakePosition | undefined {
    return this.stakes.get(stakeKey(staker, tokenId));
  }

  /**
   * Accumulate LGM rewards for a staking position
   */
  accumulateRewards(
    staker: string,
    tokenId: string,
    reward: bigint,
    block: bigint,
  ): void {
    const key = stakeKey(staker, tokenId);
    const position = this.stakes.get(key);
    if (position) {
      position.accumulatedRewards += reward;
      position.lastRewardBlock = block;
    }
  }

  /**
   * Claim accumulated LGM rewards (returns amount to distribute)
   */
  claimRewards(staker: string, tokenId: string): bigint {
    const key = stakeKey(staker, tokenId);
    const position = this.stakes.get(key);
    if (!position || position.stakedAmount === 0n) {
      throw new SecondaryTokenError(
        "No active staking position",
        "INSUFFICIENT_STAKE",
      );
    }

    const rewards = position.accumulatedRewards;
    position.accumulatedRewards = 0n;
    return rewards;
  }

  // ============ Liquidity Pool ============

  /**
   * Create a liquidity pool
   */
  createPool(pool: LiquidityPoolState): void {
    this.pools.set(pool.poolId, { ...pool });
  }

  /**
   * Get pool state
   */
  getPool(poolId: string): LiquidityPoolState | undefined {
    const pool = this.pools.get(poolId);
    return pool ? { ...pool } : undefined;
  }

  /**
   * Update pool reserves
   */
  updatePoolReserves(
    poolId: string,
    reserveA: bigint,
    reserveB: bigint,
    block: bigint,
  ): void {
    const pool = this.pools.get(poolId);
    if (!pool) {
      throw new SecondaryTokenError(
        `Pool ${poolId} not found`,
        "POOL_NOT_FOUND",
      );
    }

    // Update cumulative prices for TWAP
    const blockDelta = block - pool.lastUpdateBlock;
    if (blockDelta > 0n && pool.reserveA > 0n && pool.reserveB > 0n) {
      pool.cumulativePriceA += (pool.reserveB * blockDelta) / pool.reserveA;
      pool.cumulativePriceB += (pool.reserveA * blockDelta) / pool.reserveB;
    }

    pool.reserveA = reserveA;
    pool.reserveB = reserveB;
    pool.lastUpdateBlock = block;
  }

  // ============ Service Registry ============

  /**
   * Register an agent service
   */
  registerService(service: AgentServiceRegistration): void {
    this.services.set(service.serviceId, { ...service });
  }

  /**
   * Get service registration
   */
  getService(serviceId: string): AgentServiceRegistration | undefined {
    return this.services.get(serviceId);
  }

  // ============ Subscriptions ============

  /**
   * Create a token subscription (lock tokens for period)
   */
  createSubscription(sub: TokenSubscription): void {
    this.subscriptions.set(`${sub.subscriber}:${sub.tokenId}`, { ...sub });
  }

  /**
   * Get subscription
   */
  getSubscription(
    subscriber: string,
    tokenId: string,
  ): TokenSubscription | undefined {
    return this.subscriptions.get(`${subscriber}:${tokenId}`);
  }

  // ============ State Root ============

  /**
   * Compute merkle root of all token state
   */
  getRoot(): string {
    // Simple hash of all state for now
    const stateSnapshot = JSON.stringify(
      {
        definitionCount: this.definitions.size,
        balanceCount: this.balances.size,
        allowanceCount: this.allowances.size,
      },
      (_key, value) => (typeof value === "bigint" ? value.toString() : value),
    );
    return sha512_256Hex(stateSnapshot);
  }

  // ============ Clone (for rollback) ============

  /**
   * Deep clone for transaction rollback support
   */
  clone(): TokenRegistryStateTree {
    const cloned = new TokenRegistryStateTree();

    for (const [k, v] of this.definitions) {
      cloned.definitions.set(k, { ...v });
    }
    for (const [k, v] of this.balances) {
      cloned.balances.set(k, v);
    }
    for (const [k, v] of this.allowances) {
      cloned.allowances.set(k, v);
    }
    for (const [k, v] of this.delegations) {
      cloned.delegations.set(k, { ...v });
    }
    for (const [k, v] of this.snapshots) {
      cloned.snapshots.set(k, new Map(v));
    }
    for (const [k, v] of this.stakes) {
      cloned.stakes.set(k, { ...v });
    }
    for (const [k, v] of this.pools) {
      cloned.pools.set(k, { ...v });
    }
    for (const [k, v] of this.services) {
      cloned.services.set(k, { ...v });
    }
    for (const [k, v] of this.subscriptions) {
      cloned.subscriptions.set(k, { ...v });
    }
    for (const [k, v] of this.symbolIndex) {
      cloned.symbolIndex.set(k, v);
    }

    return cloned;
  }
}
