/**
 * Transaction RPC Handlers
 *
 * B3 FIX: Implements generic transaction submission endpoint.
 *
 * Endpoints:
 * - lgm_sendTransaction: Submit any transaction type
 */

import type { Handler } from "../handlers.js";
import type {
  LgmSendTransactionRequest,
  LgmSendTransactionResponse,
} from "../../types.js";
import { RPC_ERROR_CODES } from "../../types.js";
import { verify } from "../../../core/crypto/signatures.js";
import { sha512_256Hex } from "../../../core/crypto/hashing.js";
import type { TransitionType } from "../../../state/types.js";
import { DEFAULT_CONSENSUS_PARAMS } from "../../../consensus/pot/types.js";
import {
  parseNonNegativeBigInt,
  validatePublicKey,
  validateDid,
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
    [-32004]: "INVALID_NONCE",
    [-32602]: "INVALID_PARAMS",
  };
  return codeMap[code] || "UNKNOWN";
}

// Valid transaction types
const VALID_TX_TYPES: Set<string> = new Set([
  "transfer",
  "stake",
  "unstake",
  "delegate",
  "undelegate",
  "claim",
  "verify",
  "challenge",
  "slash",
  "add_fact",
  "challenge_fact",
  "epoch_transition",
  "dispute_initiate",
  "dispute_commit_evidence",
  "dispute_reveal_evidence",
  "dispute_arbiter_vote",
  "dispute_escalate",
  "dispute_settle",
  "dispute_timeout",
  "governance_propose",
  "governance_vote",
  "governance_execute",
  "research_bounty_create",
  "research_bounty_submit",
  "research_bounty_evaluate",
  "research_bounty_claim",
  "vesting_claim",
  "vesting_accelerate",
  "vesting_clawback",
  "domain_propose",
  "domain_endorse",
  "domain_challenge",
  "domain_finalize",
  "domain_governance_approve",
  "domain_deprecate",
  "open_payment_channel",
  "update_payment_channel",
  "close_payment_channel",
  "settle_payment_channel",
  "dispute_payment_channel",
  "autonomy_schedule_execution",
  "autonomy_cancel_schedule",
  "autonomy_execute_scheduled",
  "autonomy_invoke_contract",
  // Tree of Life transactions
  "create_project",
  "propose_project",
  "assign_task",
  "submit_deliverable",
  "approve_deliverable",
  "deploy_service",
  "call_service",
  "subscribe_service",
  "pause_service",
  "resume_service",
  "retire_service",
  // Tree of Life recruitment
  "apply_to_project",
  "review_application",
  "set_availability",
  // Tree of Life seeding
  "spawn_child_project",
  // Undecidable resolution
  "resolve_undecidable",
]);

// ============ Send Transaction Handler ============

export const lgmSendTransaction: Handler<
  LgmSendTransactionRequest,
  LgmSendTransactionResponse
> = async (params, ctx) => {
  // Validate transaction type
  if (!VALID_TX_TYPES.has(params.type)) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_PARAMS,
      `Invalid transaction type: ${params.type}`,
    );
  }

  // Verify signature
  const payload = JSON.stringify({
    type: params.type,
    payload: params.payload,
    nonce: params.nonce,
    fee: params.fee,
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

  // Validate sender public key and derive DID
  validatePublicKey(params.sender, "sender");
  const senderDid = params.senderDid
    ? validateDid(params.senderDid, "senderDid")
    : `did:lgm:${params.sender}`;
  const account = ctx.state.getAccount(senderDid);

  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Sender account not found",
    );
  }

  // Verify nonce (validate as non-negative BigInt)
  const nonce = parseNonNegativeBigInt(params.nonce, "nonce");
  const expectedNonce = account.nonce;
  if (nonce !== expectedNonce) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_NONCE,
      `Invalid nonce: expected ${expectedNonce}, got ${params.nonce}`,
    );
  }

  // Check fee balance (validate as non-negative BigInt)
  const fee = parseNonNegativeBigInt(params.fee, "fee");
  if (account.balance < fee) {
    throw createRPCError(
      RPC_ERROR_CODES.INSUFFICIENT_BALANCE,
      `Insufficient balance for fee: have ${account.balance}, need ${fee}`,
    );
  }

  // Generate transaction hash
  const txHash = sha512_256Hex(
    `tx:${params.type}:${senderDid}:${params.nonce}:${Date.now()}`,
  );

  // Create transaction object
  const blockNumber = ctx.state.getBlockNumber();
  const transaction = {
    id: txHash,
    type: params.type as TransitionType,
    sender: params.sender,
    senderDid,
    nonce: BigInt(params.nonce),
    fee,
    payload: params.payload,
    signature: params.signature,
    timestamp: Date.now(),
  };

  // Submit to mempool via consensus engine
  // Note: The consensus engine will validate and include in block
  try {
    // For now, execute directly via state transitions
    // In production, this would go through mempool → consensus → block
    const { executeTransaction } =
      await import("../../../state/transitions.js");

    const result = executeTransaction(
      ctx.state,
      transaction as any,
      blockNumber,
      DEFAULT_CONSENSUS_PARAMS,
    );

    if (result.success) {
      return {
        txHash,
        status: "included",
        blockNumber: blockNumber.toString(),
      };
    } else {
      return {
        txHash,
        status: "failed",
        error: result.error || "Transaction execution failed",
      };
    }
  } catch (error) {
    return {
      txHash,
      status: "failed",
      error:
        error instanceof Error ? error.message : "Transaction execution failed",
    };
  }
};

// ============ Export Registry ============

export const transactionHandlers: Record<string, Handler<any, any>> = {
  lgm_sendTransaction: lgmSendTransaction,
};
