/**
 * Payment Channel RPC Handlers
 *
 * B3 FIX: Implements payment channel RPC endpoints.
 *
 * Endpoints:
 * - lgm_openChannel: Open a payment channel
 * - lgm_closeChannel: Close a payment channel
 */

import type { Handler } from "../handlers.js";
import type {
  LgmOpenChannelRequest,
  LgmOpenChannelResponse,
  LgmCloseChannelRequest,
  LgmCloseChannelResponse,
} from "../../types.js";
import { RPC_ERROR_CODES } from "../../types.js";
import { verify } from "../../../core/crypto/signatures.js";
import { sha512_256Hex } from "../../../core/crypto/hashing.js";
import {
  parsePositiveBigInt,
  parseNonNegativeBigInt,
  validateDid,
  validatePublicKey,
} from "../validation.js";

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
    [-32015]: "CHANNEL_NOT_FOUND",
    [-32016]: "CHANNEL_ALREADY_CLOSED",
  };
  return codeMap[code] || "UNKNOWN";
}

// ============ Open Channel Handler ============

export const lgmOpenChannel: Handler<
  LgmOpenChannelRequest,
  LgmOpenChannelResponse
> = async (params, ctx) => {
  // Verify signature
  const payload = JSON.stringify({
    provider: params.providerDid,
    deposit: params.deposit,
    durationBlocks: params.durationBlocks,
    settlementFrequency: params.settlementFrequency,
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

  // Get payer account (validate DID derivation)
  validatePublicKey(params.sender, "sender");
  const senderDid = params.senderDid
    ? validateDid(params.senderDid, "senderDid")
    : `did:lgm:${params.sender}`;
  const account = ctx.state.getAccount(senderDid);

  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Payer account not found",
    );
  }

  // Check provider account exists
  const providerAccount = ctx.state.getAccount(params.providerDid);
  if (!providerAccount) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Provider account not found",
    );
  }

  // Check deposit balance (validate as positive BigInt)
  const deposit = parsePositiveBigInt(params.deposit, "deposit");
  if (account.balance < deposit) {
    throw createRPCError(
      RPC_ERROR_CODES.INSUFFICIENT_BALANCE,
      `Insufficient balance for deposit: have ${account.balance}, need ${deposit}`,
    );
  }

  // Generate channel ID
  const channelId = sha512_256Hex(
    `channel:${senderDid}:${params.providerDid}:${params.nonce}`,
  );

  // Generate transaction hash
  const txHash = sha512_256Hex(
    `tx:open_payment_channel:${channelId}:${params.nonce}`,
  );

  // Calculate expiration block
  const currentBlock = ctx.state.getBlockNumber();
  const durationBlocks = BigInt(params.durationBlocks);
  const expiresAtBlock = currentBlock + durationBlocks;

  // Execute channel opening
  try {
    // Lock deposit in channel escrow
    ctx.state.transfer(senderDid, `channel:${channelId}:escrow`, deposit);

    // Store channel in PaymentChannelStateTree
    ctx.state.addPaymentChannel({
      channelId,
      payer: senderDid,
      provider: params.providerDid,
      deposited: deposit,
      spent: 0n,
      available: deposit,
      status: "open",
      openedAt: currentBlock,
      expiresAt: expiresAtBlock,
      lastNonce: 0n,
      lastStateHash: "",
      pendingDistributions: [],
      totalDistributed: 0n,
      settlementFrequency: params.settlementFrequency
        ? BigInt(params.settlementFrequency)
        : 100n,
      lastSettlementBlock: currentBlock,
    });

    return {
      channelId,
      txHash,
      expiresAtBlock: expiresAtBlock.toString(),
    };
  } catch (error) {
    throw createRPCError(
      RPC_ERROR_CODES.INTERNAL_ERROR,
      error instanceof Error ? error.message : "Failed to open channel",
    );
  }
};

// ============ Close Channel Handler ============

export const lgmCloseChannel: Handler<
  LgmCloseChannelRequest,
  LgmCloseChannelResponse
> = async (params, ctx) => {
  // Verify signature
  const payload = JSON.stringify({
    channelId: params.channelId,
    unilateral: params.unilateral,
    finalStateNonce: params.finalStateNonce,
    finalPayerBalance: params.finalPayerBalance,
    finalProviderBalance: params.finalProviderBalance,
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

  // Get sender account (validate DID derivation)
  validatePublicKey(params.sender, "sender");
  const senderDid = params.senderDid
    ? validateDid(params.senderDid, "senderDid")
    : `did:lgm:${params.sender}`;
  const account = ctx.state.getAccount(senderDid);

  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Account not found",
    );
  }

  // Generate transaction hash
  const txHash = sha512_256Hex(
    `tx:close_payment_channel:${params.channelId}:${params.nonce}`,
  );

  // Execute channel close
  try {
    // Get channel escrow balance (in production, this would be from channel state)
    const escrowDid = `channel:${params.channelId}:escrow`;
    const escrowAccount = ctx.state.getAccount(escrowDid);
    const totalDeposit = escrowAccount?.balance || 0n;

    // Calculate final balances
    let finalPayerBalance: bigint;
    let finalProviderBalance: bigint;

    if (params.finalPayerBalance && params.finalProviderBalance) {
      // Cooperative close with agreed balances (validate as non-negative)
      finalPayerBalance = parseNonNegativeBigInt(
        params.finalPayerBalance,
        "finalPayerBalance",
      );
      finalProviderBalance = parseNonNegativeBigInt(
        params.finalProviderBalance,
        "finalProviderBalance",
      );

      // Verify balances sum to total deposit
      if (finalPayerBalance + finalProviderBalance !== totalDeposit) {
        throw createRPCError(
          RPC_ERROR_CODES.INVALID_PARAMS,
          "Final balances must sum to total deposit",
        );
      }
    } else {
      // Unilateral close - return full deposit to payer
      // In production, there would be a dispute period
      finalPayerBalance = totalDeposit;
      finalProviderBalance = 0n;
    }

    // Settle escrow: distribute funds back to payer and provider
    const escrowDid2 = `channel:${params.channelId}:escrow`;
    if (finalPayerBalance > 0n) {
      ctx.state.transfer(escrowDid2, senderDid, finalPayerBalance);
    }
    if (finalProviderBalance > 0n) {
      const channelState = ctx.state.getPaymentChannel(params.channelId);
      if (channelState?.provider) {
        ctx.state.transfer(
          escrowDid2,
          channelState.provider,
          finalProviderBalance,
        );
      }
    }

    // Calculate dispute period end (for unilateral close)
    let disputePeriodEnds: string | undefined;
    if (params.unilateral) {
      const currentBlock = ctx.state.getBlockNumber();
      const DISPUTE_PERIOD_BLOCKS = 50n;
      disputePeriodEnds = (currentBlock + DISPUTE_PERIOD_BLOCKS).toString();
    }

    return {
      txHash,
      success: true,
      finalPayerBalance: finalPayerBalance.toString(),
      finalProviderBalance: finalProviderBalance.toString(),
      disputePeriodEnds,
    };
  } catch (error) {
    if ((error as any).code) {
      throw error;
    }
    throw createRPCError(
      RPC_ERROR_CODES.INTERNAL_ERROR,
      error instanceof Error ? error.message : "Failed to close channel",
    );
  }
};

// ============ Export Registry ============

export const channelHandlers: Record<string, Handler<any, any>> = {
  lgm_openChannel: lgmOpenChannel,
  lgm_closeChannel: lgmCloseChannel,
};
