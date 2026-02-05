/**
 * Network Configuration
 *
 * Configuration for the libp2p networking layer.
 */

// ============ Types ============

export interface NetworkConfig {
  // Node identity
  peerId?: string; // Existing peer ID (generated if not provided)
  privateKey?: Uint8Array; // Ed25519 private key

  // Listening
  listenAddresses: string[]; // Multiaddresses to listen on
  announceAddresses?: string[]; // Addresses to announce

  // Bootstrap
  bootstrapPeers: string[]; // Bootstrap node multiaddresses
  enableMdns: boolean; // Local peer discovery

  // Connection limits
  maxConnections: number;
  minConnections: number;

  // Timeouts
  connectionTimeout: number; // ms
  dialTimeout: number; // ms

  // PubSub
  pubsub: {
    enabled: boolean;
    scoreThresholds?: PubSubScoreThresholds;
  };

  // DHT
  dht: {
    enabled: boolean;
    clientMode: boolean; // true = don't serve DHT queries
  };

  // Protocol versions
  protocolVersions: {
    sync: string;
    consensus: string;
    claims: string;
    queries: string;
  };
}

export interface PubSubScoreThresholds {
  gossipThreshold: number;
  publishThreshold: number;
  graylistThreshold: number;
}

// ============ Protocol IDs (Single Source of Truth) ============

/**
 * Protocol identifiers for all libp2p protocols.
 * All protocol handlers MUST reference these constants.
 */
export const PROTOCOL_IDS = {
  SYNC: "/lgm/sync/1.0.0",
  STATE_SYNC: "/lgm/state-sync/1.0.0",
  LIGHT_SYNC: "/lgm/light-sync/1.0.0",
  CONSENSUS: "/lgm/consensus/1.0.0",
  CLAIMS: "/lgm/claims/1.0.0",
  QUERIES: "/lgm/queries/1.0.0",
} as const;

// ============ Default Configuration ============

export const DEFAULT_NETWORK_CONFIG: NetworkConfig = {
  // Listen on all interfaces
  listenAddresses: [
    "/ip4/0.0.0.0/tcp/9000",
    "/ip4/0.0.0.0/tcp/9001/ws",
    "/ip6/::/tcp/9000",
  ],

  // Bootstrap peers (mainnet - to be populated)
  bootstrapPeers: [],

  // Enable local discovery for devnet
  enableMdns: true,

  // Connection limits
  maxConnections: 100,
  minConnections: 10,

  // Timeouts
  connectionTimeout: 30000, // 30 seconds
  dialTimeout: 10000, // 10 seconds

  // PubSub
  pubsub: {
    enabled: true,
    scoreThresholds: {
      gossipThreshold: -100,
      publishThreshold: -1000,
      graylistThreshold: -10000,
    },
  },

  // DHT
  dht: {
    enabled: true,
    clientMode: false,
  },

  // Protocol versions (derived from PROTOCOL_IDS)
  protocolVersions: {
    sync: PROTOCOL_IDS.SYNC,
    consensus: PROTOCOL_IDS.CONSENSUS,
    claims: PROTOCOL_IDS.CLAIMS,
    queries: PROTOCOL_IDS.QUERIES,
  },
};

// ============ Devnet Configuration ============

export const DEVNET_CONFIG: NetworkConfig = {
  ...DEFAULT_NETWORK_CONFIG,
  listenAddresses: ["/ip4/127.0.0.1/tcp/9000", "/ip4/127.0.0.1/tcp/9001/ws"],
  bootstrapPeers: [], // Will discover via mDNS
  enableMdns: true,
  maxConnections: 50,
  minConnections: 1,
};

// ============ Testnet Configuration ============

export const TESTNET_BOOTSTRAP_PEERS = [
  // Testnet bootstrap nodes (to be populated)
  // '/ip4/testnet1.legible.money/tcp/9000/p2p/12D3Koo...',
];

export const TESTNET_CONFIG: NetworkConfig = {
  ...DEFAULT_NETWORK_CONFIG,
  bootstrapPeers: TESTNET_BOOTSTRAP_PEERS,
  enableMdns: false,
};

// ============ Protocol Topics ============

/**
 * PubSub topics for different message types
 */
export const PUBSUB_TOPICS = {
  // Block propagation
  BLOCKS: "/lgm/blocks/1.0.0",

  // Transaction propagation
  TRANSACTIONS: "/lgm/transactions/1.0.0",

  // Claim submissions
  CLAIMS: "/lgm/claims/1.0.0",

  // Verification votes
  VERIFICATIONS: "/lgm/verifications/1.0.0",

  // Consensus messages
  CONSENSUS: "/lgm/consensus/1.0.0",

  // Validator coordination
  VALIDATORS: "/lgm/validators/1.0.0",

  // === Truth Message System Topics ===

  // Evidence submissions (supporting/contradicting claims)
  EVIDENCE: "/lgm/evidence/1.0.0",

  // Evidence requests (with optional bounties)
  EVIDENCE_REQUESTS: "/lgm/evidence-requests/1.0.0",

  // Third-party attestations
  ATTESTATIONS: "/lgm/attestations/1.0.0",

  // Challenge announcements
  CHALLENGES: "/lgm/challenges/1.0.0",

  // Final resolution outcomes
  RESOLUTIONS: "/lgm/resolutions/1.0.0",

  // Acknowledgment messages (delivery confirmation)
  ACKNOWLEDGMENTS: "/lgm/acknowledgments/1.0.0",
} as const;

// ============ Message Types ============

/**
 * Network message envelope
 */
export interface NetworkMessage {
  type: string;
  version: string;
  timestamp: number;
  sender: string; // Peer ID
  payload: Uint8Array;
  signature?: string;
}

/**
 * Message types
 */
export const MESSAGE_TYPES = {
  // Block messages
  NEW_BLOCK: "new_block",
  BLOCK_REQUEST: "block_request",
  BLOCK_RESPONSE: "block_response",

  // Transaction messages
  NEW_TRANSACTION: "new_transaction",

  // Sync messages
  STATUS: "status",
  SYNC_REQUEST: "sync_request",
  SYNC_RESPONSE: "sync_response",

  // Claim messages
  NEW_CLAIM: "new_claim",
  VERIFICATION_COMMIT: "verification_commit",
  VERIFICATION_REVEAL: "verification_reveal",

  // Consensus messages
  PROPOSE: "propose",
  PREVOTE: "prevote",
  PRECOMMIT: "precommit",

  // Query messages
  QUERY: "query",
  QUERY_RESPONSE: "query_response",

  // === Truth Message System Types ===
  EVIDENCE: "evidence",
  EVIDENCE_REQUEST: "evidence_request",
  EVIDENCE_RESPONSE: "evidence_response",
  ATTESTATION: "attestation",
  CHALLENGE: "challenge",
  RESOLUTION: "resolution",
  ACKNOWLEDGMENT: "acknowledgment",
} as const;

// ============ BigInt JSON Serialization ============

/**
 * JSON replacer that handles BigInt and Uint8Array values.
 * Use with JSON.stringify for network messages containing bigint fields.
 */
export function networkReplacer(_key: string, value: unknown): unknown {
  if (typeof value === "bigint") {
    return { __bigint__: value.toString() };
  }
  if (value instanceof Uint8Array) {
    return { __uint8array__: Buffer.from(value).toString("base64") };
  }
  return value;
}

/**
 * JSON reviver that restores BigInt and Uint8Array values.
 * Use with JSON.parse for network messages containing bigint fields.
 */
export function networkReviver(_key: string, value: unknown): unknown {
  if (typeof value === "object" && value !== null) {
    if ("__bigint__" in value) {
      return BigInt((value as { __bigint__: string }).__bigint__);
    }
    if ("__uint8array__" in value) {
      return new Uint8Array(
        Buffer.from(
          (value as { __uint8array__: string }).__uint8array__,
          "base64",
        ),
      );
    }
  }
  return value;
}

// ============ Peer Scoring ============

/**
 * Peer scoring configuration
 */
export interface PeerScoreConfig {
  // Decay rates
  decayInterval: number; // ms
  decayToZero: number; // Score below which peer is disconnected

  // Weights for behaviors
  weights: {
    validBlock: number;
    invalidBlock: number;
    validTransaction: number;
    invalidTransaction: number;
    validClaim: number;
    invalidClaim: number;
    correctVerification: number;
    incorrectVerification: number;
    goodLatency: number;
    badLatency: number;
  };
}

export const DEFAULT_PEER_SCORE_CONFIG: PeerScoreConfig = {
  decayInterval: 60000, // 1 minute
  decayToZero: -1000,

  weights: {
    validBlock: 10,
    invalidBlock: -100,
    validTransaction: 1,
    invalidTransaction: -10,
    validClaim: 5,
    invalidClaim: -50,
    correctVerification: 20,
    incorrectVerification: -200,
    goodLatency: 1,
    badLatency: -5,
  },
};
