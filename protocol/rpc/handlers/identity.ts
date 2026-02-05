/**
 * Identity RPC Handlers
 *
 * Implements key rotation and key info query endpoints.
 *
 * Endpoints:
 * - lgm_rotateKey: Rotate the operational key for an account
 * - lgm_getKeyInfo: Query current key metadata for an account
 */

import type { Handler } from "../handlers.js";
import type {
  LgmRotateKeyRequest,
  LgmRotateKeyResponse,
  LgmGetKeyInfoRequest,
  LgmGetKeyInfoResponse,
} from "../../types.js";
import { RPC_ERROR_CODES } from "../../types.js";
import { verify } from "../../../core/crypto/signatures.js";
import { sha512_256Hex } from "../../../core/crypto/hashing.js";
import { validateDid } from "../validation.js";

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
    [-32003]: "INVALID_SIGNATURE",
    [-32004]: "INVALID_NONCE",
    [-32013]: "ACCOUNT_FROZEN",
    [-32014]: "ACCOUNT_IN_RECOVERY",
  };
  return codeMap[code] || "UNKNOWN";
}

// ============ Rotate Key Handler ============

export const lgmRotateKey: Handler<
  LgmRotateKeyRequest,
  LgmRotateKeyResponse
> = async (params, ctx) => {
  // Verify signature - signed by the CURRENT operational key
  const payload = JSON.stringify({
    did: params.did,
    newOperationalKeyHash: params.newOperationalKeyHash,
    action: "rotate_key",
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

  if (
    !params.newOperationalKeyHash ||
    !/^[a-fA-F0-9]{64}$/.test(params.newOperationalKeyHash)
  ) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_PARAMS,
      "newOperationalKeyHash must be a 64-character hex string",
    );
  }

  // Get account and verify it exists
  const account = ctx.state.getAccount(params.did);
  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Account not found",
    );
  }

  // Verify sender owns this account (sender pubkey hash must match current operational key)
  const senderKeyHash = sha512_256Hex(params.sender);
  if (senderKeyHash !== account.operationalKeyHash) {
    throw createRPCError(
      RPC_ERROR_CODES.NOT_AUTHORIZED,
      "Sender does not own this account",
    );
  }

  // Verify nonce matches
  if (BigInt(params.nonce) !== account.nonce) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_NONCE,
      `Nonce mismatch: expected ${account.nonce}, got ${params.nonce}`,
    );
  }

  // Check account status (frozen / in recovery)
  if (account.flags.frozen) {
    throw createRPCError(RPC_ERROR_CODES.ACCOUNT_FROZEN, "Account is frozen");
  }

  if (account.flags.inRecovery) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_IN_RECOVERY,
      "Account is in recovery; use recovery flow",
    );
  }

  // Execute key rotation via state manager
  ctx.state.rotateKey(params.did, params.newOperationalKeyHash);

  // Generate transaction hash
  const txHash = sha512_256Hex(
    `rotate_key:${params.did}:${params.newOperationalKeyHash}:${params.nonce}`,
  );

  // Get updated account for new version
  const updated = ctx.state.getAccount(params.did);

  return {
    txHash,
    success: true,
    newKeyVersion: updated?.operationalKeyVersion ?? 1,
  };
};

// ============ Get Key Info Handler ============

export const lgmGetKeyInfo: Handler<
  LgmGetKeyInfoRequest,
  LgmGetKeyInfoResponse
> = async (params, ctx) => {
  validateDid(params.did, "did");

  const account = ctx.state.getAccount(params.did);
  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Account not found",
    );
  }

  return {
    did: params.did,
    operationalKeyHash: account.operationalKeyHash,
    operationalKeyVersion: account.operationalKeyVersion,
    isFrozen: account.flags.frozen,
    isInRecovery: account.flags.inRecovery,
  };
};

// ============ Export Registry ============

export const identityHandlers: Record<string, Handler<any, any>> = {
  lgm_rotateKey: lgmRotateKey,
  lgm_getKeyInfo: lgmGetKeyInfo,
};
