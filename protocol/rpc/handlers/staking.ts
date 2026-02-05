/**
 * Staking RPC Handlers
 *
 * B3 FIX: Implements staking and delegation RPC endpoints.
 *
 * Endpoints:
 * - lgm_stake: Lock tokens as validator stake
 * - lgm_unstake: Unlock staked tokens
 * - lgm_delegate: Delegate stake to validator
 * - lgm_undelegate: Remove delegated stake
 */

import type { Handler } from "../handlers.js";
import type {
  LgmStakeRequest,
  LgmStakeResponse,
  LgmUnstakeRequest,
  LgmUnstakeResponse,
  LgmDelegateRequest,
  LgmDelegateResponse,
  LgmUndelegateRequest,
  LgmUndelegateResponse,
} from "../../types.js";
import { RPC_ERROR_CODES } from "../../types.js";
import { verify } from "../../../core/crypto/signatures.js";
import { sha512_256Hex } from "../../../core/crypto/hashing.js";
import { parsePositiveBigInt, validateDid } from "../validation.js";

// ============ Helper ============

function createRPCError(
  code: number,
  message: string,
  details?: unknown,
): Error {
  const error = new Error(message) as Error & { code: number; data: unknown };
  error.code = code;
  error.data = { errorCode: getErrorCode(code), details, retryable: false };
  return error;
}

function getErrorCode(code: number): string {
  const codeMap: Record<number, string> = {
    [-32001]: "ACCOUNT_NOT_FOUND",
    [-32002]: "INSUFFICIENT_BALANCE",
    [-32003]: "INVALID_SIGNATURE",
    [-32010]: "VALIDATOR_NOT_FOUND",
  };
  return codeMap[code] || "UNKNOWN";
}

// ============ Stake Handler ============

export const lgmStake: Handler<LgmStakeRequest, LgmStakeResponse> = async (
  params,
  ctx,
) => {
  // Verify signature
  const payload = JSON.stringify({
    did: params.did,
    amount: params.amount,
    action: "stake",
    nonce: params.nonce,
  });

  const valid = verify(
    new TextEncoder().encode(payload),
    params.signature,
    params.sender,
  );

  if (!valid) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_SIGNATURE,
      "Invalid signature",
    );
  }

  // Validate inputs
  validateDid(params.did, "did");
  const amount = parsePositiveBigInt(params.amount, "amount");

  // Get account
  const account = ctx.state.getAccount(params.did);
  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Account not found",
    );
  }

  // Check balance
  if (account.balance < amount) {
    throw createRPCError(
      RPC_ERROR_CODES.INSUFFICIENT_BALANCE,
      `Insufficient balance: have ${account.balance}, need ${amount}`,
    );
  }

  // Execute stake via state manager
  ctx.state.stake(params.did, amount);

  // Generate transaction hash
  const txHash = sha512_256Hex(
    `stake:${params.did}:${params.amount}:${params.nonce}`,
  );

  // Get updated account
  const updatedAccount = ctx.state.getAccount(params.did);

  return {
    txHash,
    success: true,
    newStakedBalance: updatedAccount?.stakedBalance.toString() || "0",
  };
};

// ============ Unstake Handler ============

export const lgmUnstake: Handler<
  LgmUnstakeRequest,
  LgmUnstakeResponse
> = async (params, ctx) => {
  // Verify signature
  const payload = JSON.stringify({
    did: params.did,
    amount: params.amount,
    action: "unstake",
    nonce: params.nonce,
  });

  const valid = verify(
    new TextEncoder().encode(payload),
    params.signature,
    params.sender,
  );

  if (!valid) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_SIGNATURE,
      "Invalid signature",
    );
  }

  // Get account
  const account = ctx.state.getAccount(params.did);
  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Account not found",
    );
  }

  // Check staked balance
  const amount = BigInt(params.amount);
  if (account.stakedBalance < amount) {
    throw createRPCError(
      RPC_ERROR_CODES.INSUFFICIENT_BALANCE,
      `Insufficient staked balance: have ${account.stakedBalance}, need ${amount}`,
    );
  }

  // Execute unstake via state manager
  ctx.state.unstake(params.did, amount);

  // Generate transaction hash
  const txHash = sha512_256Hex(
    `unstake:${params.did}:${params.amount}:${params.nonce}`,
  );

  // Get updated account
  const updatedAccount = ctx.state.getAccount(params.did);

  // Calculate unlock block (typically after unbonding period)
  const currentBlock = ctx.state.getBlockNumber();
  const unbondingPeriod = 100n; // Blocks until tokens unlock

  return {
    txHash,
    success: true,
    newStakedBalance: updatedAccount?.stakedBalance.toString() || "0",
    unlockBlock: (currentBlock + unbondingPeriod).toString(),
  };
};

// ============ Delegate Handler ============

export const lgmDelegate: Handler<
  LgmDelegateRequest,
  LgmDelegateResponse
> = async (params, ctx) => {
  // Verify signature
  const payload = JSON.stringify({
    delegator: params.delegatorDid,
    validator: params.validatorKey,
    amount: params.amount,
    action: "delegate",
    nonce: params.nonce,
  });

  const valid = verify(
    new TextEncoder().encode(payload),
    params.signature,
    params.sender,
  );

  if (!valid) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_SIGNATURE,
      "Invalid signature",
    );
  }

  // Get delegator account
  const account = ctx.state.getAccount(params.delegatorDid);
  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Delegator account not found",
    );
  }

  // Check validator exists
  const validator = ctx.state.getValidator(params.validatorKey);
  if (!validator) {
    throw createRPCError(
      RPC_ERROR_CODES.VALIDATOR_NOT_FOUND,
      "Validator not found",
    );
  }

  // Check balance
  const amount = BigInt(params.amount);
  if (account.balance < amount) {
    throw createRPCError(
      RPC_ERROR_CODES.INSUFFICIENT_BALANCE,
      `Insufficient balance: have ${account.balance}, need ${amount}`,
    );
  }

  // Execute delegation via state manager
  ctx.state.delegate(params.delegatorDid, params.validatorKey, amount);

  // Generate transaction hash
  const txHash = sha512_256Hex(
    `delegate:${params.delegatorDid}:${params.validatorKey}:${params.amount}:${params.nonce}`,
  );

  // Get total delegated to this validator
  const updatedValidator = ctx.state.getValidator(params.validatorKey);
  const totalDelegated = updatedValidator?.delegatedStake || 0n;

  return {
    txHash,
    success: true,
    totalDelegated: totalDelegated.toString(),
  };
};

// ============ Undelegate Handler ============

export const lgmUndelegate: Handler<
  LgmUndelegateRequest,
  LgmUndelegateResponse
> = async (params, ctx) => {
  // Verify signature
  const payload = JSON.stringify({
    delegator: params.delegatorDid,
    validator: params.validatorKey,
    amount: params.amount,
    action: "undelegate",
    nonce: params.nonce,
  });

  const valid = verify(
    new TextEncoder().encode(payload),
    params.signature,
    params.sender,
  );

  if (!valid) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_SIGNATURE,
      "Invalid signature",
    );
  }

  // Get delegator account
  const account = ctx.state.getAccount(params.delegatorDid);
  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Delegator account not found",
    );
  }

  // Check validator exists
  const validator = ctx.state.getValidator(params.validatorKey);
  if (!validator) {
    throw createRPCError(
      RPC_ERROR_CODES.VALIDATOR_NOT_FOUND,
      "Validator not found",
    );
  }

  // Execute undelegation via state manager (reduce delegation and start unbonding)
  const amount = BigInt(params.amount);
  ctx.state.reduceDelegation(params.delegatorDid, params.validatorKey, amount);

  // Generate transaction hash
  const txHash = sha512_256Hex(
    `undelegate:${params.delegatorDid}:${params.validatorKey}:${params.amount}:${params.nonce}`,
  );

  // Get remaining delegation
  const updatedValidator = ctx.state.getValidator(params.validatorKey);
  const remainingDelegation = updatedValidator?.delegatedStake || 0n;

  // Calculate unlock block
  const currentBlock = ctx.state.getBlockNumber();
  const unbondingPeriod = 100n;

  return {
    txHash,
    success: true,
    remainingDelegation: remainingDelegation.toString(),
    unlockBlock: (currentBlock + unbondingPeriod).toString(),
  };
};

// ============ Export Registry ============

export const stakingHandlers: Record<string, Handler<any, any>> = {
  lgm_stake: lgmStake,
  lgm_unstake: lgmUnstake,
  lgm_delegate: lgmDelegate,
  lgm_undelegate: lgmUndelegate,
};
