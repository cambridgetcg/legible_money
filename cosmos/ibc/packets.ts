/**
 * IBC Packet Definitions
 *
 * Defines packets for cross-chain knowledge sharing via IBC.
 * Enables other chains to access Legible Money's verified knowledge base.
 *
 * PP HIERARCHY:
 *   TRUTH: Packets carry verified facts with proofs
 *   UNDERSTANDING: Clear packet semantics
 *   BEAUTY: Elegant cross-chain protocol
 *   JUSTICE: Fair knowledge access across chains
 *   CREATIVITY: Novel knowledge oracle via IBC
 */

// ============ Channel Configuration ============

/**
 * IBC channel configuration for knowledge sharing
 */
export const KNOWLEDGE_CHANNEL_CONFIG = {
  /** Port ID for knowledge module */
  portId: 'knowledge',

  /** Channel version */
  version: 'lgm-knowledge-1',

  /** Channel ordering */
  ordering: 'UNORDERED' as const,
} as const;

// ============ Packet Types ============

/**
 * All packet types for knowledge IBC
 */
export type KnowledgePacket =
  | FactQueryPacket
  | FactResponsePacket
  | DomainQueryPacket
  | DomainResponsePacket
  | SubscriptionPacket
  | UpdateNotificationPacket;

/**
 * Query a specific fact
 */
export interface FactQueryPacket {
  type: 'fact_query';

  /** Unique query ID for correlation */
  queryId: string;

  /** Fact ID to query */
  factId: string;

  /** Include Merkle proof */
  includeProof: boolean;

  /** Minimum confidence required */
  minConfidence?: number;
}

/**
 * Response with fact data
 */
export interface FactResponsePacket {
  type: 'fact_response';

  /** Query ID this responds to */
  queryId: string;

  /** Whether the fact was found */
  found: boolean;

  /** Fact data (if found) */
  fact?: PacketFact;

  /** Merkle proof (if requested) */
  proof?: MerkleProof;

  /** Block height at query time */
  height: bigint;
}

/**
 * Query facts in a domain
 */
export interface DomainQueryPacket {
  type: 'domain_query';

  /** Unique query ID */
  queryId: string;

  /** Domain to query */
  domain: string;

  /** Minimum confidence filter */
  minConfidence?: number;

  /** Maximum results */
  limit?: number;

  /** Pagination cursor */
  cursor?: string;
}

/**
 * Response with domain facts
 */
export interface DomainResponsePacket {
  type: 'domain_response';

  /** Query ID this responds to */
  queryId: string;

  /** Facts in the domain */
  facts: PacketFact[];

  /** Total count in domain */
  totalCount: number;

  /** Next cursor for pagination */
  nextCursor?: string;

  /** Block height at query time */
  height: bigint;
}

/**
 * Subscribe to updates for a domain or fact
 */
export interface SubscriptionPacket {
  type: 'subscription';

  /** Subscription ID */
  subscriptionId: string;

  /** What to subscribe to */
  target: SubscriptionTarget;

  /** Subscription action */
  action: 'subscribe' | 'unsubscribe';
}

export type SubscriptionTarget =
  | { type: 'fact'; factId: string }
  | { type: 'domain'; domain: string }
  | { type: 'all' };

/**
 * Notification of fact update
 */
export interface UpdateNotificationPacket {
  type: 'update_notification';

  /** Subscription ID this is for */
  subscriptionId: string;

  /** Update type */
  updateType: 'created' | 'updated' | 'challenged' | 'falsified';

  /** Updated fact */
  fact: PacketFact;

  /** Block height of update */
  height: bigint;
}

// ============ Packet Data Types ============

/**
 * Fact data for packets (simplified for IBC)
 */
export interface PacketFact {
  /** Fact ID */
  id: string;

  /** The claim */
  claim: string;

  /** Domain */
  domain: string;

  /** Confidence score (0-1) */
  confidence: number;

  /** Verification status */
  status: 'pending' | 'active' | 'challenged' | 'falsified';

  /** Number of verifiers */
  verifierCount: number;

  /** Block when verified */
  verifiedAt: bigint;

  /** References to other facts */
  references: string[];

  /** PP metadata */
  ppMetadata?: PacketPPMetadata;
}

/**
 * Purpose Prompter metadata for cross-chain
 */
export interface PacketPPMetadata {
  /** PP scores */
  scores: {
    truth: number;
    understanding: number;
    beauty: number;
    justice: number;
    creativity: number;
    composite: number;
  };

  /** Confidence reasoning */
  confidenceReasoning: string;
}

/**
 * Merkle proof for verification
 */
export interface MerkleProof {
  /** Proof nodes */
  nodes: ProofNode[];

  /** State root */
  stateRoot: string;

  /** Leaf hash */
  leafHash: string;
}

export interface ProofNode {
  hash: string;
  position: 'left' | 'right';
}

// ============ Acknowledgements ============

/**
 * Packet acknowledgement
 */
export type PacketAcknowledgement =
  | SuccessAck
  | ErrorAck;

export interface SuccessAck {
  result: 'success';
  data?: unknown;
}

export interface ErrorAck {
  result: 'error';
  error: string;
  code: number;
}

/**
 * Error codes for acknowledgements
 */
export const ACK_ERROR_CODES = {
  FACT_NOT_FOUND: 1,
  DOMAIN_NOT_FOUND: 2,
  INVALID_QUERY: 3,
  CONFIDENCE_TOO_LOW: 4,
  SUBSCRIPTION_FAILED: 5,
  RATE_LIMITED: 6,
  INTERNAL_ERROR: 99,
} as const;

// ============ Packet Handlers ============

/**
 * Interface for handling incoming packets
 */
export interface PacketHandler {
  /** Handle fact query */
  onFactQuery(packet: FactQueryPacket): Promise<FactResponsePacket>;

  /** Handle domain query */
  onDomainQuery(packet: DomainQueryPacket): Promise<DomainResponsePacket>;

  /** Handle subscription */
  onSubscription(packet: SubscriptionPacket): Promise<SuccessAck | ErrorAck>;
}

/**
 * Interface for sending packets
 */
export interface PacketSender {
  /** Send a fact query */
  queryFact(
    channelId: string,
    factId: string,
    includeProof?: boolean
  ): Promise<FactResponsePacket>;

  /** Send a domain query */
  queryDomain(
    channelId: string,
    domain: string,
    options?: { minConfidence?: number; limit?: number }
  ): Promise<DomainResponsePacket>;

  /** Subscribe to updates */
  subscribe(
    channelId: string,
    target: SubscriptionTarget
  ): Promise<string>; // Returns subscription ID

  /** Unsubscribe */
  unsubscribe(channelId: string, subscriptionId: string): Promise<void>;
}

// ============ Packet Serialization ============

/**
 * Serialize packet for IBC
 */
export function serializePacket(packet: KnowledgePacket): Uint8Array {
  const json = JSON.stringify(packet, bigIntReplacer);
  return new TextEncoder().encode(json);
}

/**
 * Deserialize packet from IBC
 */
export function deserializePacket(data: Uint8Array): KnowledgePacket {
  const json = new TextDecoder().decode(data);
  return JSON.parse(json, bigIntReviver);
}

/**
 * Serialize acknowledgement
 */
export function serializeAck(ack: PacketAcknowledgement): Uint8Array {
  return new TextEncoder().encode(JSON.stringify(ack));
}

/**
 * Deserialize acknowledgement
 */
export function deserializeAck(data: Uint8Array): PacketAcknowledgement {
  return JSON.parse(new TextDecoder().decode(data));
}

// BigInt serialization helpers
function bigIntReplacer(_key: string, value: unknown): unknown {
  return typeof value === 'bigint' ? value.toString() + 'n' : value;
}

function bigIntReviver(_key: string, value: unknown): unknown {
  if (typeof value === 'string' && value.endsWith('n')) {
    return BigInt(value.slice(0, -1));
  }
  return value;
}

// ============ Channel Handshake ============

/**
 * Validate channel version during handshake
 */
export function validateVersion(version: string): boolean {
  return version === KNOWLEDGE_CHANNEL_CONFIG.version;
}

/**
 * Generate channel metadata for handshake
 */
export function generateChannelMetadata(): ChannelMetadata {
  return {
    version: KNOWLEDGE_CHANNEL_CONFIG.version,
    features: ['fact_query', 'domain_query', 'subscriptions'],
    chainId: '', // Filled by module
    knowledgeRoot: '', // Current state root
  };
}

export interface ChannelMetadata {
  version: string;
  features: string[];
  chainId: string;
  knowledgeRoot: string;
}

// ============ Rate Limiting ============

/**
 * Rate limit configuration for IBC queries
 */
export interface RateLimitConfig {
  /** Max queries per block */
  maxQueriesPerBlock: number;

  /** Max subscriptions per channel */
  maxSubscriptionsPerChannel: number;

  /** Query timeout (blocks) */
  queryTimeoutBlocks: bigint;
}

export const DEFAULT_RATE_LIMIT: RateLimitConfig = {
  maxQueriesPerBlock: 100,
  maxSubscriptionsPerChannel: 10,
  queryTimeoutBlocks: 100n,
};
