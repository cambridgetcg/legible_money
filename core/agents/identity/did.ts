import { z } from "zod";
import type { PublicKey } from "../../crypto/keys.js";

/**
 * Decentralized Identifier (DID) for AI Agents
 *
 * Accepts two formats:
 *   - 4-part: did:lgm:<network>:<44-char-hex>  (full W3C-style with network)
 *   - 3-part: did:lgm:<32-char-hex>            (compact hash-based, most common)
 */
export const DIDSchema = z
  .string()
  .regex(/^did:lgm:(?:(?:mainnet|testnet|devnet):[a-f0-9]{44}|[a-f0-9]{32})$/);

export type DID = z.infer<typeof DIDSchema>;

/**
 * Agent capability attestations
 */
export const CapabilitySchema = z.enum([
  "compute.inference", // Can run inference
  "compute.training", // Can train models
  "data.storage", // Can store data
  "data.retrieval", // Can retrieve/search data
  "service.api", // Can provide API services
  "service.task", // Can complete tasks
  "coordination.validator", // Can validate transactions
  "coordination.oracle", // Can provide oracle data
]);

export type Capability = z.infer<typeof CapabilitySchema>;

/**
 * Agent metadata stored on-chain
 */
export interface AgentIdentity {
  did: DID;
  publicKey: PublicKey;
  capabilities: Capability[];
  metadata: {
    name?: string;
    description?: string;
    endpoint?: string; // API endpoint for the agent
    version?: string;
  };
  createdAt: number;
  updatedAt: number;
}

/**
 * Canonical on-chain DID derivation from a public key.
 * This is the single source of truth used by the state layer,
 * wallet, SDK, and all transaction processing.
 *
 * Format: did:lgm:{first 32 hex chars of publicKey}
 */
export function publicKeyToDid(publicKey: PublicKey): string {
  return `did:lgm:${publicKey.slice(0, 32)}`;
}

/**
 * Create a full W3C-style DID with network component.
 * Used for DID documents and off-chain identity resolution.
 *
 * Format: did:lgm:{network}:{first 44 hex chars of publicKey}
 */
export function createDID(
  publicKey: PublicKey,
  network: "mainnet" | "testnet" | "devnet" = "devnet",
): DID {
  // publicKey is already hex-encoded, use first 44 chars
  return `did:lgm:${network}:${publicKey.slice(0, 44)}`;
}

/**
 * Resolve DID to address component
 * Handles both 3-part (did:lgm:{hash}) and 4-part (did:lgm:{network}:{address}) formats
 */
export function didToAddress(did: DID): string {
  const parts = did.split(":");
  // 4-part: did:lgm:{network}:{address}
  if (parts.length === 4) return parts[3];
  // 3-part: did:lgm:{hash}
  return parts[2];
}

/**
 * Get network from DID
 * Returns 'mainnet' as default for 3-part DIDs (no network component)
 */
export function didToNetwork(did: DID): "mainnet" | "testnet" | "devnet" {
  const parts = did.split(":");
  // 4-part: did:lgm:{network}:{address}
  if (parts.length === 4) return parts[2] as "mainnet" | "testnet" | "devnet";
  // 3-part DIDs default to mainnet
  return "mainnet";
}

/**
 * Agent identity document for off-chain resolution
 */
export interface DIDDocument {
  "@context": "https://www.w3.org/ns/did/v1";
  id: DID;
  verificationMethod: [
    {
      id: string;
      type: "Ed25519VerificationKey2020";
      controller: DID;
      publicKeyMultibase: string;
    },
  ];
  authentication: string[];
  service?: [
    {
      id: string;
      type: "AgentEndpoint";
      serviceEndpoint: string;
    },
  ];
}

/**
 * Create a DID document for an agent
 */
export function createDIDDocument(identity: AgentIdentity): DIDDocument {
  const keyId = `${identity.did}#key-1`;

  return {
    "@context": "https://www.w3.org/ns/did/v1",
    id: identity.did,
    verificationMethod: [
      {
        id: keyId,
        type: "Ed25519VerificationKey2020",
        controller: identity.did,
        // publicKey is already hex-encoded
        publicKeyMultibase: "z" + identity.publicKey,
      },
    ],
    authentication: [keyId],
    service: identity.metadata.endpoint
      ? [
          {
            id: `${identity.did}#agent`,
            type: "AgentEndpoint",
            serviceEndpoint: identity.metadata.endpoint,
          },
        ]
      : undefined,
  };
}
