/**
 * Dispute RPC Handlers
 *
 * B3 FIX: Implements dispute-related RPC endpoints.
 *
 * Endpoints:
 * - lgm_openDispute: Initiate a dispute
 * - lgm_submitEvidence: Submit evidence (commit or reveal phase)
 * - lgm_resolveDispute: Cast arbiter vote
 */

import type { Handler } from "../handlers.js";
import type {
  LgmOpenDisputeRequest,
  LgmOpenDisputeResponse,
  LgmSubmitEvidenceRequest,
  LgmSubmitEvidenceResponse,
  LgmResolveDisputeRequest,
  LgmResolveDisputeResponse,
} from "../../types.js";
import { RPC_ERROR_CODES } from "../../types.js";
import { verify } from "../../../core/crypto/signatures.js";
import { sha512_256Hex } from "../../../core/crypto/hashing.js";
import { parsePositiveBigInt } from "../validation.js";

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
    [-32008]: "NOT_AUTHORIZED",
    [-32013]: "DISPUTE_NOT_FOUND",
    [-32014]: "INVALID_DISPUTE_PHASE",
  };
  return codeMap[code] || "UNKNOWN";
}

// Extend error codes for disputes
const DISPUTE_ERROR_CODES = {
  ...RPC_ERROR_CODES,
  DISPUTE_NOT_FOUND: -32013,
  INVALID_DISPUTE_PHASE: -32014,
} as const;

// ============ Open Dispute Handler ============

export const lgmOpenDispute: Handler<
  LgmOpenDisputeRequest,
  LgmOpenDisputeResponse
> = async (params, ctx) => {
  // Verify signature
  const payload = JSON.stringify({
    targetId: params.targetId,
    targetType: params.targetType,
    domain: params.domain,
    stake: params.stake,
    evidence: params.evidence,
    reasonHash: params.reasonHash,
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

  // Get challenger account
  const senderDid = params.senderDid || `did:lgm:${params.sender.slice(0, 32)}`;
  const account = ctx.state.getAccount(senderDid);

  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Challenger account not found",
    );
  }

  // Check stake balance
  const stake = parsePositiveBigInt(params.stake, "stake");
  if (account.balance < stake) {
    throw createRPCError(
      RPC_ERROR_CODES.INSUFFICIENT_BALANCE,
      `Insufficient balance for stake: have ${account.balance}, need ${stake}`,
    );
  }

  // Generate dispute ID
  const disputeId = sha512_256Hex(
    `dispute:${params.targetId}:${senderDid}:${params.nonce}`,
  );

  // Create dispute transaction
  const txHash = sha512_256Hex(
    `tx:dispute_initiate:${disputeId}:${params.nonce}`,
  );

  // Execute dispute initiation
  // In production, this would go through state transitions
  try {
    const currentBlock = Number(ctx.state.getBlockNumber());

    // Dispute phase durations (in blocks)
    const EVIDENCE_PHASE_BLOCKS = 50;
    const VOTING_PHASE_BLOCKS = 100;

    // Lock challenger stake
    ctx.state.transfer(senderDid, `dispute:${disputeId}:escrow`, stake);

    return {
      disputeId,
      txHash,
      evidenceDeadline: currentBlock + EVIDENCE_PHASE_BLOCKS,
      votingDeadline:
        currentBlock + EVIDENCE_PHASE_BLOCKS + VOTING_PHASE_BLOCKS,
    };
  } catch (error) {
    throw createRPCError(
      RPC_ERROR_CODES.INTERNAL_ERROR,
      error instanceof Error ? error.message : "Failed to open dispute",
    );
  }
};

// ============ Submit Evidence Handler ============

export const lgmSubmitEvidence: Handler<
  LgmSubmitEvidenceRequest,
  LgmSubmitEvidenceResponse
> = async (params, ctx) => {
  // Verify signature
  const payload = JSON.stringify({
    disputeId: params.disputeId,
    evidenceHash: params.evidenceHash,
    evidenceIds: params.evidenceIds,
    nonce: params.nonce,
    phase: params.phase,
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

  // Get submitter account
  const senderDid = params.senderDid || `did:lgm:${params.sender.slice(0, 32)}`;
  const account = ctx.state.getAccount(senderDid);

  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Account not found",
    );
  }

  // Generate deterministic transaction hash (Date.now() would cause node disagreement)
  const blockNumber_hash = ctx.state.getBlockNumber();
  const txHash = sha512_256Hex(
    `tx:dispute_${params.phase}_evidence:${params.disputeId}:${senderDid}:${blockNumber_hash}`,
  );

  // Execute evidence submission
  try {
    if (params.phase === "commit") {
      // Commit phase: store hash
      // In production, this would be stored in dispute state
      return {
        txHash,
        success: true,
        phase: "commit",
      };
    } else {
      // Reveal phase: verify hash matches
      if (!params.evidenceIds || !params.nonce) {
        throw createRPCError(
          RPC_ERROR_CODES.INVALID_PARAMS,
          "evidenceIds and nonce required for reveal phase",
        );
      }

      // Verify commitment matches
      const revealHash = sha512_256Hex(
        JSON.stringify({
          evidenceIds: params.evidenceIds,
          nonce: params.nonce,
        }),
      );

      if (revealHash !== params.evidenceHash) {
        throw createRPCError(
          DISPUTE_ERROR_CODES.INVALID_DISPUTE_PHASE,
          "Evidence hash does not match commitment",
        );
      }

      return {
        txHash,
        success: true,
        phase: "reveal",
      };
    }
  } catch (error) {
    if ((error as any).code) {
      throw error;
    }
    throw createRPCError(
      RPC_ERROR_CODES.INTERNAL_ERROR,
      error instanceof Error ? error.message : "Failed to submit evidence",
    );
  }
};

// ============ Resolve Dispute Handler ============

export const lgmResolveDispute: Handler<
  LgmResolveDisputeRequest,
  LgmResolveDisputeResponse
> = async (params, ctx) => {
  // Verify signature
  const payload = JSON.stringify({
    disputeId: params.disputeId,
    vote: params.vote,
    justificationHash: params.justificationHash,
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

  // Get arbiter account
  const senderDid = params.senderDid || `did:lgm:${params.sender.slice(0, 32)}`;
  const account = ctx.state.getAccount(senderDid);

  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Account not found",
    );
  }

  // Check if sender is an arbiter (validator with good standing)
  const validator = ctx.state.getValidator(params.sender);
  if (!validator || !validator.isActive) {
    throw createRPCError(
      RPC_ERROR_CODES.NOT_AUTHORIZED,
      "Only active validators can vote on disputes",
    );
  }

  // Generate transaction hash
  const txHash = sha512_256Hex(
    `tx:dispute_arbiter_vote:${params.disputeId}:${senderDid}:${params.nonce}`,
  );

  // Execute vote
  try {
    // In production, this would update dispute state and check for resolution
    // For now, return mock vote counts

    // Placeholder vote tracking
    const votesForChallenger = params.vote === "challenger" ? 1 : 0;
    const votesForDefender = params.vote === "defender" ? 1 : 0;

    // Check if dispute is resolved (would need 2/3 majority in production)
    const totalVotes = votesForChallenger + votesForDefender;
    const resolved = totalVotes >= 3; // Simplified: 3 votes to resolve

    let winner: "challenger" | "defender" | undefined;
    if (resolved) {
      winner =
        votesForChallenger > votesForDefender ? "challenger" : "defender";
    }

    return {
      txHash,
      success: true,
      votesForChallenger,
      votesForDefender,
      resolved,
      winner,
    };
  } catch (error) {
    throw createRPCError(
      RPC_ERROR_CODES.INTERNAL_ERROR,
      error instanceof Error ? error.message : "Failed to submit vote",
    );
  }
};

// ============ Export Registry ============

export const disputeHandlers: Record<string, Handler<any, any>> = {
  lgm_openDispute: lgmOpenDispute,
  lgm_submitEvidence: lgmSubmitEvidence,
  lgm_resolveDispute: lgmResolveDispute,
};
