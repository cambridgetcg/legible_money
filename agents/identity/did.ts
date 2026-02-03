import { z } from 'zod';
import { PublicKey, bytesToHex } from '../../core/crypto/keys.js';

/**
 * Decentralized Identifier (DID) for AI Agents
 * Format: did:lgm:<network>:<address>
 */
export const DIDSchema = z.string().regex(/^did:lgm:(mainnet|testnet|devnet):[a-f0-9]{44}$/);

export type DID = z.infer<typeof DIDSchema>;

/**
 * Agent capability attestations
 */
export const CapabilitySchema = z.enum([
  'compute.inference',      // Can run inference
  'compute.training',       // Can train models
  'data.storage',           // Can store data
  'data.retrieval',         // Can retrieve/search data
  'service.api',            // Can provide API services
  'service.task',           // Can complete tasks
  'coordination.validator', // Can validate transactions
  'coordination.oracle',    // Can provide oracle data
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
    endpoint?: string;  // API endpoint for the agent
    version?: string;
  };
  createdAt: number;
  updatedAt: number;
}

/**
 * Create a DID from a public key
 */
export function createDID(publicKey: PublicKey, network: 'mainnet' | 'testnet' | 'devnet' = 'devnet'): DID {
  const keyHex = bytesToHex(publicKey);
  return `did:lgm:${network}:${keyHex.slice(0, 44)}`;
}

/**
 * Resolve DID to address component
 */
export function didToAddress(did: DID): string {
  const parts = did.split(':');
  return parts[3];
}

/**
 * Get network from DID
 */
export function didToNetwork(did: DID): 'mainnet' | 'testnet' | 'devnet' {
  const parts = did.split(':');
  return parts[2] as 'mainnet' | 'testnet' | 'devnet';
}

/**
 * Agent identity document for off-chain resolution
 */
export interface DIDDocument {
  '@context': 'https://www.w3.org/ns/did/v1';
  id: DID;
  verificationMethod: [{
    id: string;
    type: 'Ed25519VerificationKey2020';
    controller: DID;
    publicKeyMultibase: string;
  }];
  authentication: string[];
  service?: [{
    id: string;
    type: 'AgentEndpoint';
    serviceEndpoint: string;
  }];
}

/**
 * Create a DID document for an agent
 */
export function createDIDDocument(identity: AgentIdentity): DIDDocument {
  const keyId = `${identity.did}#key-1`;

  return {
    '@context': 'https://www.w3.org/ns/did/v1',
    id: identity.did,
    verificationMethod: [{
      id: keyId,
      type: 'Ed25519VerificationKey2020',
      controller: identity.did,
      publicKeyMultibase: 'z' + bytesToHex(identity.publicKey),
    }],
    authentication: [keyId],
    service: identity.metadata.endpoint ? [{
      id: `${identity.did}#agent`,
      type: 'AgentEndpoint',
      serviceEndpoint: identity.metadata.endpoint,
    }] : undefined,
  };
}
