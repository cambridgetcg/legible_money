/**
 * Secondary Token Types (LGM-20)
 *
 * Defines the type system for secondary tokens on the LGM base layer.
 * All gas fees are paid in LGM. Secondary tokens are application-layer
 * assets for governance, staking, utility, and DeFi use cases.
 *
 * Key invariant: LGM total supply is unaffected by secondary token operations.
 */

import type { DID } from "../../agents/identity/did.js";

// ============ Protocol Constants ============

/**
 * Well-known token ID for wrapped LGM (wLGM).
 * Deterministic so wrap/unwrap can always find it without
 * relying on type-based lookup (which could match user-created wrapped tokens).
 */
export const WLGM_TOKEN_ID = "wlgm-protocol-v1";

// ============ Token Definition ============

/**
 * Supported secondary token types
 */
export type SecondaryTokenType =
  | "fungible" // Standard LGM-20 token
  | "governance" // Voting power + delegation
  | "utility" // Service access gating
  | "wrapped" // 1:1 LGM wrapper (wLGM)
  | "lp"; // Liquidity pool share

/**
 * On-chain token definition stored in the token registry
 */
export interface TokenDefinition {
  /** Deterministic ID: hash(creator + name + salt) */
  tokenId: string;

  /** Human-readable token name */
  name: string;

  /** Short symbol (e.g., "GOV", "wLGM") */
  symbol: string;

  /** Decimal places (0-36) */
  decimals: number;

  /** Creator DID (has mint/pause authority) */
  creator: DID;

  /** Maximum supply cap (0n = unlimited) */
  maxSupply: bigint;

  /** Current circulating supply */
  totalSupply: bigint;

  /** Token type determining behavior */
  tokenType: SecondaryTokenType;

  /** Arbitrary metadata (description, icon, etc.) */
  metadata: Record<string, unknown>;

  /** Block when token was created */
  createdAtBlock: bigint;

  /** Whether transfers are paused */
  paused: boolean;
}

// ============ Token Balances ============

/**
 * Per-account token balance entry
 */
export interface TokenBalanceEntry {
  /** Account DID */
  did: DID;

  /** Token ID */
  tokenId: string;

  /** Balance amount */
  balance: bigint;
}

/**
 * Allowance for delegated spending (approve/transferFrom)
 */
export interface TokenAllowance {
  /** Token owner */
  owner: DID;

  /** Approved spender */
  spender: DID;

  /** Token ID */
  tokenId: string;

  /** Approved amount */
  amount: bigint;
}

// ============ Governance Token Extensions ============

/**
 * Governance voting power delegation
 */
export interface VoteDelegation {
  /** Delegator DID */
  delegator: DID;

  /** Delegate DID (receives voting power) */
  delegate: DID;

  /** Token ID (governance token) */
  tokenId: string;

  /** Block when delegated */
  delegatedAtBlock: bigint;
}

/**
 * Snapshot of voting power at a specific block
 */
export interface VotingPowerSnapshot {
  /** Token ID */
  tokenId: string;

  /** Block number of snapshot */
  snapshotBlock: bigint;

  /** DID -> voting power at snapshot */
  powers: Map<DID, bigint>;
}

// ============ Staking Extensions ============

/**
 * Secondary token staking position
 */
export interface SecondaryStakePosition {
  /** Staker DID */
  staker: DID;

  /** Token being staked */
  tokenId: string;

  /** Amount staked */
  stakedAmount: bigint;

  /** Block when staked */
  stakedAtBlock: bigint;

  /** Accumulated LGM rewards (claimable) */
  accumulatedRewards: bigint;

  /** Block of last reward calculation */
  lastRewardBlock: bigint;
}

// ============ Wrapped LGM Extensions ============

/**
 * Wrapped LGM (wLGM) backing state
 * Invariant: wLGM totalSupply === LGM held in wLGM contract
 */
export interface WrappedLGMState {
  /** wLGM token ID */
  tokenId: string;

  /** LGM locked as backing */
  lgmBacking: bigint;

  /** Contract DID holding the LGM */
  contractDid: DID;
}

// ============ Liquidity Pool Extensions ============

/**
 * Liquidity pool state (constant-product AMM)
 */
export interface LiquidityPoolState {
  /** Pool identifier */
  poolId: string;

  /** LP token ID */
  lpTokenId: string;

  /** Token A (LGM or secondary) */
  tokenA: string; // "LGM" or tokenId

  /** Token B (secondary) */
  tokenB: string; // tokenId

  /** Reserve of token A */
  reserveA: bigint;

  /** Reserve of token B */
  reserveB: bigint;

  /** Swap fee in basis points (e.g., 30 = 0.3%) */
  swapFeeBps: number;

  /** Cumulative price for TWAP oracle */
  cumulativePriceA: bigint;
  cumulativePriceB: bigint;
  lastUpdateBlock: bigint;
}

// ============ Utility Token Extensions ============

/**
 * Token-gated access tier
 */
export interface AccessTier {
  /** Tier name */
  name: string;

  /** Minimum token balance required */
  minimumBalance: bigint;

  /** Access level (higher = more features) */
  level: number;
}

/**
 * Token subscription lock
 */
export interface TokenSubscription {
  /** Subscriber DID */
  subscriber: DID;

  /** Token ID */
  tokenId: string;

  /** Amount locked */
  lockedAmount: bigint;

  /** Lock start block */
  startBlock: bigint;

  /** Lock end block */
  endBlock: bigint;
}

// ============ Agent Service Token Extensions ============

/**
 * AI agent service registration
 */
export interface AgentServiceRegistration {
  /** Service provider DID */
  provider: DID;

  /** Service identifier */
  serviceId: string;

  /** Token used for payment */
  paymentTokenId: string;

  /** Price per invocation */
  pricePerCall: bigint;

  /** Quality guarantee stake */
  qualityStake: bigint;

  /** Whether service is active */
  active: boolean;
}

// ============ Transaction Payloads ============

export interface TokenCreatePayload {
  type: "token_create";
  name: string;
  symbol: string;
  decimals: number;
  maxSupply: bigint;
  tokenType: SecondaryTokenType;
  metadata: Record<string, unknown>;
  salt: string;
}

export interface TokenMintPayload {
  type: "token_mint";
  tokenId: string;
  to: string; // DID
  amount: bigint;
}

export interface TokenBurnPayload {
  type: "token_burn";
  tokenId: string;
  amount: bigint;
}

export interface TokenTransferSecondaryPayload {
  type: "token_transfer_secondary";
  tokenId: string;
  to: string; // DID
  amount: bigint;
}

export interface TokenApprovePayload {
  type: "token_approve";
  tokenId: string;
  spender: string; // DID
  amount: bigint;
}

export interface TokenTransferFromPayload {
  type: "token_transfer_from";
  tokenId: string;
  from: string; // DID
  to: string; // DID
  amount: bigint;
}

export interface TokenPausePayload {
  type: "token_pause";
  tokenId: string;
}

export interface TokenUnpausePayload {
  type: "token_unpause";
  tokenId: string;
}

export interface TokenDelegatePowerPayload {
  type: "token_delegate_power";
  tokenId: string;
  delegate: string; // DID
}

export interface TokenWrapPayload {
  type: "token_wrap";
  amount: bigint;
}

export interface TokenUnwrapPayload {
  type: "token_unwrap";
  amount: bigint;
}

export interface LPAddLiquidityPayload {
  type: "lp_add_liquidity";
  poolId: string;
  amountA: bigint;
  amountB: bigint;
}

export interface LPRemoveLiquidityPayload {
  type: "lp_remove_liquidity";
  poolId: string;
  lpAmount: bigint;
}

export interface LPSwapPayload {
  type: "lp_swap";
  poolId: string;
  tokenIn: string; // "LGM" or tokenId
  amountIn: bigint;
  minAmountOut: bigint;
}

// ============ Gas Costs ============

export const SECONDARY_TOKEN_GAS_COSTS = {
  token_create: 150000n,
  token_mint: 50000n,
  token_burn: 30000n,
  token_transfer_secondary: 25000n,
  token_approve: 25000n,
  token_transfer_from: 30000n,
  token_pause: 25000n,
  token_unpause: 25000n,
  token_delegate_power: 30000n,
  token_wrap: 30000n,
  token_unwrap: 30000n,
  lp_add_liquidity: 60000n,
  lp_remove_liquidity: 50000n,
  lp_swap: 40000n,
} as const;

// ============ Constants ============

/** LGM fee for creating a new token (0.022 LGM) */
export const TOKEN_CREATION_FEE = 22n * 10n ** 15n;

/** Minimum swap fee in basis points */
export const MIN_SWAP_FEE_BPS = 10; // 0.1%

/** Default swap fee in basis points */
export const DEFAULT_SWAP_FEE_BPS = 30; // 0.3%

/** Minimum pool reserve to prevent draining */
export const MIN_POOL_RESERVE = 1000n;

// ============ Errors ============

export class SecondaryTokenError extends Error {
  constructor(
    message: string,
    public code: SecondaryTokenErrorCode,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "SecondaryTokenError";
  }
}

export type SecondaryTokenErrorCode =
  | "TOKEN_NOT_FOUND"
  | "TOKEN_ALREADY_EXISTS"
  | "TOKEN_PAUSED"
  | "INSUFFICIENT_TOKEN_BALANCE"
  | "INSUFFICIENT_LGM_FOR_FEE"
  | "INSUFFICIENT_ALLOWANCE"
  | "UNAUTHORIZED_MINT"
  | "UNAUTHORIZED_PAUSE"
  | "MAX_SUPPLY_EXCEEDED"
  | "INVALID_AMOUNT"
  | "SELF_TRANSFER"
  | "DUPLICATE_SYMBOL"
  | "INVALID_DECIMALS"
  | "POOL_NOT_FOUND"
  | "POOL_INSUFFICIENT_RESERVE"
  | "POOL_SLIPPAGE_EXCEEDED"
  | "WRAP_INSUFFICIENT_LGM"
  | "UNWRAP_INSUFFICIENT_WLGM"
  | "CIRCULAR_DELEGATION"
  | "SERVICE_NOT_FOUND"
  | "SERVICE_INACTIVE"
  | "INSUFFICIENT_STAKE";
