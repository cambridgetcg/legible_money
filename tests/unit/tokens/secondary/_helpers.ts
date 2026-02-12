/**
 * Shared Test Helpers for Secondary Token Tests
 *
 * Provides agent creation, funding, transaction builders, and compound helpers
 * following patterns from tests/unit/contracts/retrieval_contract.test.ts.
 */

import { StateManager } from "../../../../state/state_manager.js";
import { TokenRegistryStateTree } from "../../../../state/token_registry.js";
import {
  executeTransaction,
  FEE_POOL_DID,
  REWARD_POOL_DID,
  PROTOCOL_TREASURY_DID,
} from "../../../../state/transitions.js";
import type {
  Transaction,
  ConsensusParams,
} from "../../../../consensus/pot/types.js";
import { sha512_256Hex } from "../../../../core/crypto/hashing.js";
import type {
  TokenDefinition,
  SecondaryTokenType,
  TokenCreatePayload,
  TokenMintPayload,
  TokenBurnPayload,
  TokenTransferSecondaryPayload,
  TokenApprovePayload,
  TokenTransferFromPayload,
  TokenPausePayload,
  TokenUnpausePayload,
  TokenDelegatePowerPayload,
  TokenWrapPayload,
  TokenUnwrapPayload,
  LPAddLiquidityPayload,
  LPRemoveLiquidityPayload,
  LPSwapPayload,
} from "../../../../core/tokens/secondary/types.js";

// ============ Constants ============

export const ONE_LGM = 10n ** 18n;
export const TOKEN_CREATION_FEE = 22n * ONE_LGM;
export const PROPOSAL_THRESHOLD = 100n * ONE_LGM;

// ============ Agent Helpers ============

let agentCounter = 0;

/**
 * Create a test agent with deterministic DID
 */
export function makeAgent(): {
  publicKey: string;
  did: string;
  privateKey: string;
} {
  agentCounter++;
  const hex = agentCounter.toString(16).padStart(32, "0");
  return {
    publicKey: hex + hex.slice(0, 12), // 44 chars for full public key
    did: `did:lgm:${hex}`,
    privateKey: `privkey-${hex}`,
  };
}

/**
 * Reset agent counter (call in beforeEach)
 */
export function resetAgentCounter(): void {
  agentCounter = 0;
}

/**
 * Fund an agent with LGM by creating their account with a balance
 */
export function fundAgent(
  state: StateManager,
  did: string,
  lgmAmount: bigint,
): void {
  const existingBalance = safeGetBalance(state, did);
  if (existingBalance === null) {
    state.createAccount(did, sha512_256Hex(`key-${did}`), lgmAmount);
  } else {
    // Account exists, transfer from reward pool
    const rewardBalance = safeGetBalance(state, REWARD_POOL_DID);
    if (rewardBalance !== null && rewardBalance >= lgmAmount) {
      state.transfer(REWARD_POOL_DID, did, lgmAmount);
    }
  }
}

function safeGetBalance(state: StateManager, did: string): bigint | null {
  try {
    return state.getBalance(did);
  } catch {
    return null;
  }
}

// ============ Test State Setup ============

/**
 * Create a StateManager with all system accounts needed for token tests
 */
export function createTestState(): StateManager {
  const state = new StateManager();
  setupTokenSystemAccounts(state);
  return state;
}

/**
 * Set up system accounts required for transaction execution
 */
export function setupTokenSystemAccounts(state: StateManager): void {
  state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
  state.createAccount(
    REWARD_POOL_DID,
    "reward-pool-hash",
    1_000_000n * ONE_LGM,
  );
  state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);
}

/**
 * Create test consensus params (minimal for token tests)
 */
export function createTestParams(): ConsensusParams {
  return {
    targetBlockTimeMs: 2521,
    minValidatorStake: 111n * 10n ** 15n, // 0.111 LGM
    minClaimStake: 11n * ONE_LGM,
    validatorQuorum: 0.66,
    verificationRoundBlocks: 20,
    commitPhaseBlocks: 6,
    revealPhaseBlocks: 6,
    epochLength: 100n,
    maxValidators: 100,
    maxTransactionsPerBlock: 111,
    acceptanceThreshold: 0.77,
    rejectionThreshold: 0.77,
    contestedThreshold: 0.55,
    blockReward: 11n * ONE_LGM,
    verificationReward: 1n * ONE_LGM,
    citationRewardPerFact: 1n * ONE_LGM,
    wrongVerificationSlash: 0.06,
    missedVerificationSlash: 0.0001,
    invalidClaimSlash: 0.22,
    researchShareRate: 0.22,
    unbondingPeriod: 1000n,
    minValidators: 5,
    confirmationBlocks: 11,
    finalityBlocks: 44,
  } as unknown as ConsensusParams;
}

// ============ Token Registry Helper ============

/**
 * Create a fresh TokenRegistryStateTree for unit tests
 */
export function createTestRegistry(): TokenRegistryStateTree {
  return new TokenRegistryStateTree();
}

// ============ Transaction Builders ============

/**
 * Build a token_create transaction
 */
export function createTokenCreateTx(
  sender: { publicKey: string; did: string },
  payload: {
    name: string;
    symbol: string;
    decimals?: number;
    maxSupply?: bigint;
    tokenType?: SecondaryTokenType;
    metadata?: Record<string, unknown>;
    salt?: string;
  },
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "token_create" as any,
    sender: sender.publicKey,
    senderDid: sender.did,
    fee: 150000n,
    nonce,
    timestamp: Date.now(),
    signature: `sig-${sender.did}`,
    payload: {
      type: "token_create",
      name: payload.name,
      symbol: payload.symbol,
      decimals: payload.decimals ?? 18,
      maxSupply: payload.maxSupply ?? 0n,
      tokenType: payload.tokenType ?? "fungible",
      metadata: payload.metadata ?? {},
      salt: payload.salt ?? `salt-${Date.now()}`,
    } as TokenCreatePayload,
  } as unknown as Transaction;
}

/**
 * Build a token_mint transaction
 */
export function createTokenMintTx(
  sender: { publicKey: string; did: string },
  tokenId: string,
  to: string,
  amount: bigint,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "token_mint" as any,
    sender: sender.publicKey,
    senderDid: sender.did,
    fee: 50000n,
    nonce,
    timestamp: Date.now(),
    signature: `sig-${sender.did}`,
    payload: {
      type: "token_mint",
      tokenId,
      to,
      amount,
    } as TokenMintPayload,
  } as unknown as Transaction;
}

/**
 * Build a token_burn transaction
 */
export function createTokenBurnTx(
  sender: { publicKey: string; did: string },
  tokenId: string,
  amount: bigint,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "token_burn" as any,
    sender: sender.publicKey,
    senderDid: sender.did,
    fee: 30000n,
    nonce,
    timestamp: Date.now(),
    signature: `sig-${sender.did}`,
    payload: {
      type: "token_burn",
      tokenId,
      amount,
    } as TokenBurnPayload,
  } as unknown as Transaction;
}

/**
 * Build a token_transfer_secondary transaction
 */
export function createTokenTransferTx(
  sender: { publicKey: string; did: string },
  tokenId: string,
  to: string,
  amount: bigint,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "token_transfer_secondary" as any,
    sender: sender.publicKey,
    senderDid: sender.did,
    fee: 25000n,
    nonce,
    timestamp: Date.now(),
    signature: `sig-${sender.did}`,
    payload: {
      type: "token_transfer_secondary",
      tokenId,
      to,
      amount,
    } as TokenTransferSecondaryPayload,
  } as unknown as Transaction;
}

/**
 * Build a token_approve transaction
 */
export function createTokenApproveTx(
  sender: { publicKey: string; did: string },
  tokenId: string,
  spender: string,
  amount: bigint,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "token_approve" as any,
    sender: sender.publicKey,
    senderDid: sender.did,
    fee: 25000n,
    nonce,
    timestamp: Date.now(),
    signature: `sig-${sender.did}`,
    payload: {
      type: "token_approve",
      tokenId,
      spender,
      amount,
    } as TokenApprovePayload,
  } as unknown as Transaction;
}

/**
 * Build a token_transfer_from transaction
 */
export function createTokenTransferFromTx(
  sender: { publicKey: string; did: string },
  tokenId: string,
  from: string,
  to: string,
  amount: bigint,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "token_transfer_from" as any,
    sender: sender.publicKey,
    senderDid: sender.did,
    fee: 30000n,
    nonce,
    timestamp: Date.now(),
    signature: `sig-${sender.did}`,
    payload: {
      type: "token_transfer_from",
      tokenId,
      from,
      to,
      amount,
    } as TokenTransferFromPayload,
  } as unknown as Transaction;
}

/**
 * Build a token_pause transaction
 */
export function createTokenPauseTx(
  sender: { publicKey: string; did: string },
  tokenId: string,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "token_pause" as any,
    sender: sender.publicKey,
    senderDid: sender.did,
    fee: 25000n,
    nonce,
    timestamp: Date.now(),
    signature: `sig-${sender.did}`,
    payload: {
      type: "token_pause",
      tokenId,
    } as TokenPausePayload,
  } as unknown as Transaction;
}

/**
 * Build a token_unpause transaction
 */
export function createTokenUnpauseTx(
  sender: { publicKey: string; did: string },
  tokenId: string,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "token_unpause" as any,
    sender: sender.publicKey,
    senderDid: sender.did,
    fee: 25000n,
    nonce,
    timestamp: Date.now(),
    signature: `sig-${sender.did}`,
    payload: {
      type: "token_unpause",
      tokenId,
    } as TokenUnpausePayload,
  } as unknown as Transaction;
}

/**
 * Build a token_delegate_power transaction
 */
export function createTokenDelegatePowerTx(
  sender: { publicKey: string; did: string },
  tokenId: string,
  delegate: string,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "token_delegate_power" as any,
    sender: sender.publicKey,
    senderDid: sender.did,
    fee: 30000n,
    nonce,
    timestamp: Date.now(),
    signature: `sig-${sender.did}`,
    payload: {
      type: "token_delegate_power",
      tokenId,
      delegate,
    } as TokenDelegatePowerPayload,
  } as unknown as Transaction;
}

/**
 * Build a token_wrap transaction
 */
export function createTokenWrapTx(
  sender: { publicKey: string; did: string },
  amount: bigint,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "token_wrap" as any,
    sender: sender.publicKey,
    senderDid: sender.did,
    fee: 30000n,
    nonce,
    timestamp: Date.now(),
    signature: `sig-${sender.did}`,
    payload: {
      type: "token_wrap",
      amount,
    } as TokenWrapPayload,
  } as unknown as Transaction;
}

/**
 * Build a token_unwrap transaction
 */
export function createTokenUnwrapTx(
  sender: { publicKey: string; did: string },
  amount: bigint,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "token_unwrap" as any,
    sender: sender.publicKey,
    senderDid: sender.did,
    fee: 30000n,
    nonce,
    timestamp: Date.now(),
    signature: `sig-${sender.did}`,
    payload: {
      type: "token_unwrap",
      amount,
    } as TokenUnwrapPayload,
  } as unknown as Transaction;
}

/**
 * Build an lp_add_liquidity transaction
 */
export function createLPAddLiquidityTx(
  sender: { publicKey: string; did: string },
  poolId: string,
  amountA: bigint,
  amountB: bigint,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "lp_add_liquidity" as any,
    sender: sender.publicKey,
    senderDid: sender.did,
    fee: 60000n,
    nonce,
    timestamp: Date.now(),
    signature: `sig-${sender.did}`,
    payload: {
      type: "lp_add_liquidity",
      poolId,
      amountA,
      amountB,
    } as LPAddLiquidityPayload,
  } as unknown as Transaction;
}

/**
 * Build an lp_remove_liquidity transaction
 */
export function createLPRemoveLiquidityTx(
  sender: { publicKey: string; did: string },
  poolId: string,
  lpAmount: bigint,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "lp_remove_liquidity" as any,
    sender: sender.publicKey,
    senderDid: sender.did,
    fee: 50000n,
    nonce,
    timestamp: Date.now(),
    signature: `sig-${sender.did}`,
    payload: {
      type: "lp_remove_liquidity",
      poolId,
      lpAmount,
    } as LPRemoveLiquidityPayload,
  } as unknown as Transaction;
}

/**
 * Build an lp_swap transaction
 */
export function createLPSwapTx(
  sender: { publicKey: string; did: string },
  poolId: string,
  tokenIn: string,
  amountIn: bigint,
  minAmountOut: bigint,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "lp_swap" as any,
    sender: sender.publicKey,
    senderDid: sender.did,
    fee: 40000n,
    nonce,
    timestamp: Date.now(),
    signature: `sig-${sender.did}`,
    payload: {
      type: "lp_swap",
      poolId,
      tokenIn,
      amountIn,
      minAmountOut,
    } as LPSwapPayload,
  } as unknown as Transaction;
}

// ============ Compound Helpers ============

/**
 * Deploy a test token directly via the registry (bypasses transaction layer)
 */
export function deployTestToken(
  registry: TokenRegistryStateTree,
  creator: { did: string },
  opts: {
    name?: string;
    symbol?: string;
    decimals?: number;
    maxSupply?: bigint;
    tokenType?: SecondaryTokenType;
    metadata?: Record<string, unknown>;
    salt?: string;
  } = {},
): { tokenId: string; definition: TokenDefinition } {
  const name = opts.name ?? "TestToken";
  const symbol = opts.symbol ?? "TST";
  const salt = opts.salt ?? `salt-${Date.now()}-${Math.random()}`;

  const tokenId = registry.generateTokenId(creator.did, name, salt);

  const definition: TokenDefinition = {
    tokenId,
    name,
    symbol,
    decimals: opts.decimals ?? 18,
    creator: creator.did as any,
    maxSupply: opts.maxSupply ?? 0n,
    totalSupply: 0n,
    tokenType: opts.tokenType ?? "fungible",
    metadata: opts.metadata ?? {},
    createdAtBlock: 1n,
    paused: false,
  };

  registry.createToken(definition);

  return { tokenId, definition };
}

/**
 * Deploy a token and mint initial supply
 */
export function deployAndMint(
  registry: TokenRegistryStateTree,
  creator: { did: string },
  recipient: string,
  amount: bigint,
  opts: {
    name?: string;
    symbol?: string;
    decimals?: number;
    maxSupply?: bigint;
    tokenType?: SecondaryTokenType;
    salt?: string;
  } = {},
): { tokenId: string; definition: TokenDefinition } {
  const { tokenId, definition } = deployTestToken(registry, creator, opts);
  registry.mint(tokenId, recipient, amount);
  return { tokenId, definition };
}
