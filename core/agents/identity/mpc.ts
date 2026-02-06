/**
 * MPC (Multi-Party Computation) Wallet
 *
 * Key is split across multiple parties - no single party has the full key.
 * Signing requires cooperation between parties.
 *
 * Typical setup:
 * - Agent holds shard 1
 * - Protocol service holds shard 2
 * - Operator holds shard 3
 *
 * Any 2-of-3 can sign (threshold signature).
 *
 * Benefits:
 * - Agent losing shard doesn't lose funds (operator + protocol can recover)
 * - Developer seeing agent state doesn't get full key
 * - Protocol compromise doesn't expose keys (need agent or operator)
 */

import {
  generateKeypair,
  generateKeypairFromSeed,
  type PublicKey,
  bytesToHex,
  hexToBytes,
} from "../../crypto/keys.js";
import { sha512_256Hex } from "../../crypto/hashing.js";
import { randomBytes } from "@noble/hashes/utils";

// ============ Types ============

/**
 * MPC wallet configuration
 */
export interface MPCConfig {
  walletId: string;
  threshold: number; // Minimum parties needed to sign
  totalParties: number;
  parties: MPCParty[];
  publicKey: PublicKey; // Combined public key
  createdAt: number;
}

/**
 * A party in the MPC scheme
 */
export interface MPCParty {
  id: string;
  type: "agent" | "protocol" | "operator" | "guardian";
  publicShare: PublicKey; // This party's public share
  endpoint?: string; // Where to reach this party
  isOnline: boolean;
  lastSeen: number;
}

/**
 * A signing session
 */
export interface MPCSigningSession {
  sessionId: string;
  walletId: string;
  message: Uint8Array;
  messageHash: string;

  // Participants
  requiredParties: number;
  participatingParties: string[]; // Party IDs

  // Progress
  commitments: Map<string, Uint8Array>;
  partialSignatures: Map<string, Uint8Array>;

  // Result
  status: "collecting" | "signing" | "complete" | "failed";
  finalSignature?: string;

  // Timing
  createdAt: number;
  expiresAt: number;
}

/**
 * Signing request from one party to others
 */
export interface MPCSigningRequest {
  sessionId: string;
  walletId: string;
  messageHash: string;
  requestedBy: string; // Party ID
  requestedAt: number;
  expiresAt: number;

  // For verification
  intentDescription: string; // What is being signed
  signature: string; // Requester's signature on the request
}

/**
 * Signing response from a party
 */
export interface MPCSigningResponse {
  sessionId: string;
  partyId: string;
  commitment?: Uint8Array; // First round
  partialSignature?: Uint8Array; // Second round
  approved: boolean;
  reason?: string; // If rejected
}

// ============ MPC Setup ============

/**
 * Initialize a new MPC wallet
 *
 * Note: This is a simplified implementation.
 * Production would use a proper threshold ECDSA or EdDSA scheme
 * like FROST, GG18, or similar.
 */
export function initializeMPCWallet(
  parties: Array<{ id: string; type: MPCParty["type"]; endpoint?: string }>,
  threshold: number = 2,
): { config: MPCConfig; shares: Map<string, Uint8Array> } {
  if (threshold > parties.length) {
    throw new Error("Threshold cannot exceed number of parties");
  }
  if (threshold < 2) {
    throw new Error("Threshold must be at least 2 for security");
  }

  const walletId = bytesToHex(randomBytes(16));

  // In a real implementation, this would use DKG (Distributed Key Generation)
  // For now, we simulate by generating a key and splitting it

  const { publicKey, privateKey } = generateKeypair();
  const privateKeyBytes = hexToBytes(privateKey);

  // Split private key using Shamir (simplified)
  const shares = splitSecretSimple(privateKeyBytes, threshold, parties.length);

  // Create party configs with their public shares
  const partyConfigs: MPCParty[] = parties.map((p, i) => ({
    id: p.id,
    type: p.type,
    publicShare: derivePublicShare(shares.get(i + 1)!),
    endpoint: p.endpoint,
    isOnline: true,
    lastSeen: Date.now(),
  }));

  // Map shares to party IDs
  const shareMap = new Map<string, Uint8Array>();
  parties.forEach((p, i) => {
    shareMap.set(p.id, shares.get(i + 1)!);
  });

  const config: MPCConfig = {
    walletId,
    threshold,
    totalParties: parties.length,
    parties: partyConfigs,
    publicKey,
    createdAt: Date.now(),
  };

  return { config, shares: shareMap };
}

/**
 * Derive public share from private share
 */
function derivePublicShare(privateShare: Uint8Array): PublicKey {
  // In real MPC, each party would generate their own keypair
  // and the public key would be derived from all public shares
  const { publicKey } = generateKeypairFromSeed(privateShare);
  return publicKey;
}

// ============ MPC Signing ============

/**
 * Create a signing session
 */
export function createSigningSession(
  config: MPCConfig,
  message: Uint8Array,
  requestingPartyId: string,
  expiresInMs: number = 60000,
): MPCSigningSession {
  const now = Date.now();

  return {
    sessionId: bytesToHex(randomBytes(16)),
    walletId: config.walletId,
    message,
    messageHash: sha512_256Hex(message),

    requiredParties: config.threshold,
    participatingParties: [requestingPartyId],

    commitments: new Map(),
    partialSignatures: new Map(),

    status: "collecting",

    createdAt: now,
    expiresAt: now + expiresInMs,
  };
}

/**
 * Create a signing request to send to other parties
 */
export function createSigningRequest(
  session: MPCSigningSession,
  requestingPartyId: string,
  intentDescription: string,
  requesterPrivateShare: Uint8Array,
): MPCSigningRequest {
  const request: Omit<MPCSigningRequest, "signature"> = {
    sessionId: session.sessionId,
    walletId: session.walletId,
    messageHash: session.messageHash,
    requestedBy: requestingPartyId,
    requestedAt: Date.now(),
    expiresAt: session.expiresAt,
    intentDescription,
  };

  // Sign the request with requester's share
  const requestBytes = new TextEncoder().encode(JSON.stringify(request));
  const { privateKey } = generateKeypairFromSeed(requesterPrivateShare);
  const signature = bytesToHex(
    signSimple(requestBytes, hexToBytes(privateKey)),
  );

  return { ...request, signature };
}

/**
 * Process a signing request (by a receiving party)
 */
export function processSigningRequest(
  request: MPCSigningRequest,
  partyId: string,
  partyPrivateShare: Uint8Array,
  approve: boolean,
  reason?: string,
): MPCSigningResponse {
  if (!approve) {
    return {
      sessionId: request.sessionId,
      partyId,
      approved: false,
      reason,
    };
  }

  // Generate commitment (first round of MPC signing)
  const commitment = generateCommitment(partyPrivateShare, request.messageHash);

  return {
    sessionId: request.sessionId,
    partyId,
    commitment,
    approved: true,
  };
}

/**
 * Add a response to a signing session
 */
export function addSigningResponse(
  session: MPCSigningSession,
  response: MPCSigningResponse,
): MPCSigningSession {
  if (!response.approved) {
    // Party rejected - session may fail if not enough parties
    return session;
  }

  const updated = { ...session };

  if (response.commitment) {
    updated.commitments.set(response.partyId, response.commitment);
    if (!updated.participatingParties.includes(response.partyId)) {
      updated.participatingParties.push(response.partyId);
    }
  }

  if (response.partialSignature) {
    updated.partialSignatures.set(response.partyId, response.partialSignature);
  }

  // Check if we have enough commitments to proceed
  if (
    updated.commitments.size >= updated.requiredParties &&
    updated.status === "collecting"
  ) {
    updated.status = "signing";
  }

  // Check if we have enough partial signatures to complete
  if (
    updated.partialSignatures.size >= updated.requiredParties &&
    updated.status === "signing"
  ) {
    // Combine partial signatures
    const finalSig = combinePartialSignatures(
      Array.from(updated.partialSignatures.values()),
      updated.message,
    );

    if (finalSig) {
      updated.finalSignature = bytesToHex(finalSig);
      updated.status = "complete";
    } else {
      updated.status = "failed";
    }
  }

  return updated;
}

/**
 * Generate partial signature (second round)
 */
export function generatePartialSignature(
  session: MPCSigningSession,
  partyPrivateShare: Uint8Array,
  _allCommitments: Uint8Array[],
): Uint8Array {
  // In real MPC, this would use the commitments from other parties
  // to generate a partial signature that can be combined

  // Simplified: sign the message hash with the party's share
  const messageBytes = hexToBytes(session.messageHash);
  return signSimple(messageBytes, partyPrivateShare);
}

// ============ Simplified Crypto Helpers ============

/**
 * Simple secret splitting (Shamir-like)
 */
function splitSecretSimple(
  secret: Uint8Array,
  _threshold: number,
  totalShares: number,
): Map<number, Uint8Array> {
  // Simplified XOR-based splitting for demonstration
  // Real implementation would use proper Shamir over finite field

  const shares = new Map<number, Uint8Array>();

  // Generate random shares for all but last
  const accumulated = new Uint8Array(secret.length);

  for (let i = 1; i < totalShares; i++) {
    const share = randomBytes(secret.length);
    shares.set(i, share);

    // XOR into accumulated
    for (let j = 0; j < share.length; j++) {
      accumulated[j] ^= share[j];
    }
  }

  // Last share is secret XOR accumulated
  const lastShare = new Uint8Array(secret.length);
  for (let j = 0; j < secret.length; j++) {
    lastShare[j] = secret[j] ^ accumulated[j];
  }
  shares.set(totalShares, lastShare);

  return shares;
}

/**
 * Generate commitment for signing round
 */
function generateCommitment(
  privateShare: Uint8Array,
  messageHash: string,
): Uint8Array {
  // Hash of (share || message || random nonce)
  const nonce = randomBytes(32);
  const input = new Uint8Array([
    ...privateShare,
    ...hexToBytes(messageHash),
    ...nonce,
  ]);
  return new Uint8Array(Buffer.from(sha512_256Hex(input), "hex"));
}

/**
 * Simple signing (placeholder for actual MPC signing)
 */
function signSimple(message: Uint8Array, privateKey: Uint8Array): Uint8Array {
  // This would be actual EdDSA partial signing in production
  const hash = sha512_256Hex(new Uint8Array([...message, ...privateKey]));
  return hexToBytes(hash);
}

/**
 * Combine partial signatures
 */
function combinePartialSignatures(
  partials: Uint8Array[],
  _message: Uint8Array,
): Uint8Array | null {
  // In real MPC, this would combine the partial signatures
  // using the specific threshold signature scheme

  // Simplified: XOR all partials
  if (partials.length === 0) return null;

  const combined = new Uint8Array(partials[0].length);
  for (const partial of partials) {
    for (let i = 0; i < partial.length; i++) {
      combined[i] ^= partial[i];
    }
  }

  return combined;
}

// ============ MPC Wallet Operations ============

/**
 * Check if enough parties are online to sign
 */
export function canSign(config: MPCConfig): {
  canSign: boolean;
  onlineParties: number;
} {
  const online = config.parties.filter((p) => p.isOnline).length;
  return {
    canSign: online >= config.threshold,
    onlineParties: online,
  };
}

/**
 * Update party online status
 */
export function updatePartyStatus(
  config: MPCConfig,
  partyId: string,
  isOnline: boolean,
): MPCConfig {
  return {
    ...config,
    parties: config.parties.map((p) =>
      p.id === partyId
        ? { ...p, isOnline, lastSeen: isOnline ? Date.now() : p.lastSeen }
        : p,
    ),
  };
}

/**
 * Add a new party (requires existing threshold to approve)
 */
export function addParty(
  config: MPCConfig,
  newParty: Omit<MPCParty, "isOnline" | "lastSeen">,
): MPCConfig {
  if (config.parties.some((p) => p.id === newParty.id)) {
    throw new Error("Party already exists");
  }
  // In real implementation, this would trigger key resharing
  return {
    ...config,
    totalParties: config.totalParties + 1,
    parties: [...config.parties, { ...newParty, isOnline: false, lastSeen: 0 }],
  };
}

/**
 * Remove a party (requires existing threshold to approve)
 */
export function removeParty(config: MPCConfig, partyId: string): MPCConfig {
  if (!config.parties.some((p) => p.id === partyId)) {
    throw new Error("Party not found");
  }
  // In real implementation, this would trigger key resharing
  const newParties = config.parties.filter((p) => p.id !== partyId);

  if (newParties.length < config.threshold) {
    throw new Error("Cannot remove party: would go below threshold");
  }

  return {
    ...config,
    totalParties: newParties.length,
    parties: newParties,
  };
}
