/**
 * Node Role Types & Capabilities
 *
 * Defines the role-based node architecture for LGM.
 * Roles are additive capabilities on the existing LegibleNode class —
 * each role specifies which topics, storage namespaces, sync strategies,
 * and protocols it needs.
 */

import { PUBSUB_TOPICS, PROTOCOL_IDS } from "../config.js";

// ============ Node Roles ============

/**
 * All supported node roles in LGM.
 *
 * Existing roles:
 * - full: Standard node with complete state
 * - validator: Block producer with staked LGM
 * - archive: Full historical state retention
 * - chatnode: Light client for queries
 *
 * New specialized roles:
 * - seed: Bootstrap infrastructure for peer discovery
 * - sentry: DDoS shield for validators
 * - knowledge_gateway: Optimized knowledge/evidence query serving
 * - relay: Network partition bridge
 * - evidence_oracle: External evidence fetch/archive pipeline
 */
export type NodeRole =
  | "full"
  | "validator"
  | "archive"
  | "chatnode"
  | "seed"
  | "sentry"
  | "knowledge_gateway"
  | "relay"
  | "evidence_oracle";

// ============ Role Capabilities ============

/**
 * Sync strategy determines how a node synchronizes with the network.
 */
export type SyncStrategy =
  | "full" // Download and verify all blocks
  | "state" // State sync then follow head
  | "light" // Headers + Merkle proofs only
  | "headers_only" // Block headers for relay/routing
  | "none"; // No sync (stateless, e.g. seed)

/**
 * Defines the capabilities and resource requirements for a node role.
 * Used by the orchestrator to configure topic subscriptions, storage,
 * sync strategy, and connection parameters.
 */
export interface RoleCapabilities {
  /** GossipSub topics this role subscribes to */
  topics: string[];

  /** Storage namespaces this role persists (0x01-0xf0 prefix bytes) */
  storageNamespaces: number[];

  /** How this role synchronizes with the network */
  syncStrategy: SyncStrategy;

  /** Custom libp2p protocols this role registers */
  protocols: string[];

  /** DHT mode: server (serve queries) or client (query only) */
  dhtMode: "server" | "client";

  /** Maximum peer connections */
  maxConnections: number;

  /** Minimum peer connections to maintain */
  minConnections: number;

  /** Human-readable description */
  description: string;
}

// ============ New Protocol IDs for Specialized Roles ============

/**
 * Protocol IDs for new role-specific protocols.
 * Extends PROTOCOL_IDS from network/config.ts.
 */
export const ROLE_PROTOCOL_IDS = {
  /** Seed: Peer exchange protocol */
  PEER_EXCHANGE: "/lgm/pex/1.0.0",

  /** Seed: Bootstrap service */
  BOOTSTRAP: "/lgm/bootstrap/1.0.0",

  /** Sentry: Private validator link */
  SENTRY_LINK: "/lgm/sentry/1.0.0",

  /** Knowledge Gateway: Fact/evidence queries */
  KNOWLEDGE_QUERY: "/lgm/knowledge/1.0.0",

  /** Knowledge Gateway: Evidence-specific queries */
  EVIDENCE_QUERY: "/lgm/evidence-query/1.0.0",

  /** Relay: Cross-region relay protocol */
  RELAY: "/lgm/relay/1.0.0",

  /** Evidence Oracle: External evidence fetch */
  EVIDENCE_FETCH: "/lgm/evidence-fetch/1.0.0",

  /** Evidence Oracle: Content archival */
  ARCHIVE: "/lgm/archive/1.0.0",
} as const;

// ============ All Topics (convenience array) ============

const ALL_TOPICS = Object.values(PUBSUB_TOPICS);

/** Core block propagation topics needed for consensus participation */
const CORE_TOPICS = [
  PUBSUB_TOPICS.BLOCKS,
  PUBSUB_TOPICS.TRANSACTIONS,
  PUBSUB_TOPICS.CONSENSUS,
  PUBSUB_TOPICS.VALIDATORS,
];

/** Knowledge/evidence topics for truth-focused roles */
const KNOWLEDGE_TOPICS = [
  PUBSUB_TOPICS.CLAIMS,
  PUBSUB_TOPICS.VERIFICATIONS,
  PUBSUB_TOPICS.EVIDENCE,
  PUBSUB_TOPICS.EVIDENCE_REQUESTS,
  PUBSUB_TOPICS.ATTESTATIONS,
];

/** Evidence lifecycle topics */
const EVIDENCE_TOPICS = [
  PUBSUB_TOPICS.EVIDENCE,
  PUBSUB_TOPICS.EVIDENCE_REQUESTS,
  PUBSUB_TOPICS.ATTESTATIONS,
];

// ============ Storage Namespace Byte Values ============

/**
 * Numeric values of STORAGE_NAMESPACES prefixes for role filtering.
 * References state/storage/types.ts STORAGE_NAMESPACES.
 */
const NS = {
  ACCOUNTS: 0x01,
  VALIDATORS: 0x02,
  KNOWLEDGE: 0x03,
  EVIDENCE: 0x04,
  DISPUTES: 0x05,
  VESTING: 0x06,
  RESEARCH: 0x07,
  REFERENCES: 0x08,
  CLAIMS: 0x09,
  PAYMENT_CHANNELS: 0x0a,
  PROJECTS: 0x0b,
  SERVICES: 0x0c,
  GOVERNANCE: 0x0d,
  TASKS: 0x0e,
  SUBSCRIPTIONS: 0x0f,
  BLOCK_HEADERS: 0x10,
  BLOCK_BODIES: 0x11,
  BLOCK_RECEIPTS: 0x12,
  TX_INDEX: 0x13,
  SEEDS: 0x14,
  SCHEDULES: 0x15,
  SNAPSHOTS: 0x20,
  SNAPSHOT_META: 0x21,
  ARCHIVE: 0x30,
  PROOFS: 0x40,
  CONFIG: 0xf0,
} as const;

/** All state namespaces (accounts through governance) */
const ALL_STATE_NS = [
  NS.ACCOUNTS,
  NS.VALIDATORS,
  NS.KNOWLEDGE,
  NS.EVIDENCE,
  NS.DISPUTES,
  NS.VESTING,
  NS.RESEARCH,
  NS.REFERENCES,
  NS.CLAIMS,
  NS.PAYMENT_CHANNELS,
  NS.PROJECTS,
  NS.SERVICES,
  NS.GOVERNANCE,
  NS.TASKS,
  NS.SUBSCRIPTIONS,
  NS.SEEDS,
  NS.SCHEDULES,
];

/** Block storage namespaces */
const BLOCK_NS = [
  NS.BLOCK_HEADERS,
  NS.BLOCK_BODIES,
  NS.BLOCK_RECEIPTS,
  NS.TX_INDEX,
];

/** Infra namespaces */
const INFRA_NS = [NS.SNAPSHOTS, NS.SNAPSHOT_META, NS.CONFIG];

/** Full node storage: all state + blocks + infra */
const FULL_STORAGE = [...ALL_STATE_NS, ...BLOCK_NS, ...INFRA_NS];

/** Knowledge-focused storage */
const KNOWLEDGE_STORAGE = [
  NS.KNOWLEDGE,
  NS.EVIDENCE,
  NS.REFERENCES,
  NS.CLAIMS,
  NS.RESEARCH,
  NS.CONFIG,
];

/** Evidence-focused storage */
const EVIDENCE_STORAGE = [NS.EVIDENCE, NS.REFERENCES, NS.CONFIG];

// ============ All Core Protocol IDs ============

const CORE_PROTOCOLS = [
  PROTOCOL_IDS.SYNC,
  PROTOCOL_IDS.STATE_SYNC,
  PROTOCOL_IDS.CONSENSUS,
  PROTOCOL_IDS.CLAIMS,
  PROTOCOL_IDS.QUERIES,
];

// ============ Role Capabilities Map ============

/**
 * Defines capabilities for every node role.
 * Used by the node orchestrator to configure LegibleNode instances.
 */
export const ROLE_CAPABILITIES: Record<NodeRole, RoleCapabilities> = {
  // --- Existing Roles ---

  full: {
    topics: ALL_TOPICS,
    storageNamespaces: FULL_STORAGE,
    syncStrategy: "full",
    protocols: CORE_PROTOCOLS,
    dhtMode: "server",
    maxConnections: 100,
    minConnections: 10,
    description: "Standard full node with complete state and block history",
  },

  validator: {
    topics: ALL_TOPICS,
    storageNamespaces: FULL_STORAGE,
    syncStrategy: "full",
    protocols: CORE_PROTOCOLS,
    dhtMode: "server",
    maxConnections: 100,
    minConnections: 10,
    description: "Block-producing validator with staked LGM and full state",
  },

  archive: {
    topics: ALL_TOPICS,
    storageNamespaces: [...FULL_STORAGE, NS.ARCHIVE],
    syncStrategy: "full",
    protocols: [...CORE_PROTOCOLS, PROTOCOL_IDS.STATE_SYNC],
    dhtMode: "server",
    maxConnections: 150,
    minConnections: 10,
    description: "Archive node retaining complete historical state for queries",
  },

  chatnode: {
    topics: [
      PUBSUB_TOPICS.BLOCKS,
      PUBSUB_TOPICS.CLAIMS,
      PUBSUB_TOPICS.VERIFICATIONS,
    ],
    storageNamespaces: [NS.KNOWLEDGE, NS.PROOFS, NS.CONFIG],
    syncStrategy: "light",
    protocols: [PROTOCOL_IDS.LIGHT_SYNC, PROTOCOL_IDS.QUERIES],
    dhtMode: "client",
    maxConnections: 25,
    minConnections: 3,
    description: "Light client for knowledge queries with Merkle proofs",
  },

  // --- New Specialized Roles ---

  seed: {
    topics: [], // Stateless — no block/tx processing
    storageNamespaces: [NS.CONFIG], // Only peer address book
    syncStrategy: "none",
    protocols: [ROLE_PROTOCOL_IDS.PEER_EXCHANGE, ROLE_PROTOCOL_IDS.BOOTSTRAP],
    dhtMode: "server",
    maxConnections: 500,
    minConnections: 20,
    description:
      "Bootstrap seed node for peer discovery — stateless, high fan-out",
  },

  sentry: {
    topics: ALL_TOPICS, // Proxies everything to/from validator
    storageNamespaces: FULL_STORAGE, // Mirrors validator state
    syncStrategy: "full",
    protocols: [...CORE_PROTOCOLS, ROLE_PROTOCOL_IDS.SENTRY_LINK],
    dhtMode: "server",
    maxConnections: 200,
    minConnections: 10,
    description:
      "Validator DDoS shield — public-facing proxy hiding validator identity",
  },

  knowledge_gateway: {
    topics: KNOWLEDGE_TOPICS,
    storageNamespaces: KNOWLEDGE_STORAGE,
    syncStrategy: "state",
    protocols: [
      PROTOCOL_IDS.STATE_SYNC,
      PROTOCOL_IDS.QUERIES,
      ROLE_PROTOCOL_IDS.KNOWLEDGE_QUERY,
      ROLE_PROTOCOL_IDS.EVIDENCE_QUERY,
    ],
    dhtMode: "server",
    maxConnections: 300,
    minConnections: 10,
    description:
      "Optimized knowledge/evidence query serving for AI agent access",
  },

  relay: {
    topics: CORE_TOPICS, // Core propagation only
    storageNamespaces: [NS.BLOCK_HEADERS, NS.CONFIG], // Headers for routing
    syncStrategy: "headers_only",
    protocols: [PROTOCOL_IDS.SYNC, ROLE_PROTOCOL_IDS.RELAY],
    dhtMode: "server",
    maxConnections: 400,
    minConnections: 20,
    description:
      "Network partition bridge — high fan-out, low storage for message relay",
  },

  evidence_oracle: {
    topics: EVIDENCE_TOPICS,
    storageNamespaces: EVIDENCE_STORAGE,
    syncStrategy: "state",
    protocols: [
      PROTOCOL_IDS.STATE_SYNC,
      ROLE_PROTOCOL_IDS.EVIDENCE_FETCH,
      ROLE_PROTOCOL_IDS.ARCHIVE,
    ],
    dhtMode: "client",
    maxConnections: 100,
    minConnections: 5,
    description:
      "External evidence pipeline — fetch, archive, and verify evidence sources",
  },
} as const;

// ============ Utilities ============

/**
 * Get the merged capabilities for a node running multiple roles.
 * Topics and protocols are unioned; storage namespaces are unioned;
 * maxConnections takes the highest value; syncStrategy takes the most complete.
 */
export function mergeRoleCapabilities(roles: NodeRole[]): RoleCapabilities {
  if (roles.length === 0) {
    return ROLE_CAPABILITIES.full;
  }
  if (roles.length === 1) {
    return ROLE_CAPABILITIES[roles[0]];
  }

  const syncPriority: Record<SyncStrategy, number> = {
    full: 4,
    state: 3,
    light: 2,
    headers_only: 1,
    none: 0,
  };

  const topicSet = new Set<string>();
  const namespaceSet = new Set<number>();
  const protocolSet = new Set<string>();
  let maxConn = 0;
  let minConn = Infinity;
  let bestSync: SyncStrategy = "none";
  let bestDht: "server" | "client" = "client";

  for (const role of roles) {
    const caps = ROLE_CAPABILITIES[role];
    caps.topics.forEach((t) => topicSet.add(t));
    caps.storageNamespaces.forEach((n) => namespaceSet.add(n));
    caps.protocols.forEach((p) => protocolSet.add(p));
    if (caps.maxConnections > maxConn) maxConn = caps.maxConnections;
    if (caps.minConnections < minConn) minConn = caps.minConnections;
    if (syncPriority[caps.syncStrategy] > syncPriority[bestSync]) {
      bestSync = caps.syncStrategy;
    }
    if (caps.dhtMode === "server") bestDht = "server";
  }

  return {
    topics: Array.from(topicSet),
    storageNamespaces: Array.from(namespaceSet).sort((a, b) => a - b),
    syncStrategy: bestSync,
    protocols: Array.from(protocolSet),
    dhtMode: bestDht,
    maxConnections: maxConn,
    minConnections: minConn === Infinity ? 10 : minConn,
    description: `Multi-role node: ${roles.join(", ")}`,
  };
}

/**
 * Check if a role requires block processing (subscribes to BLOCKS topic).
 */
export function roleProcessesBlocks(role: NodeRole): boolean {
  return ROLE_CAPABILITIES[role].topics.includes(PUBSUB_TOPICS.BLOCKS);
}

/**
 * Check if a role participates in consensus.
 */
export function roleParticipatesInConsensus(role: NodeRole): boolean {
  return ROLE_CAPABILITIES[role].topics.includes(PUBSUB_TOPICS.CONSENSUS);
}

/**
 * Get all roles that serve DHT queries (server mode).
 */
export function getDHTServerRoles(): NodeRole[] {
  return (Object.keys(ROLE_CAPABILITIES) as NodeRole[]).filter(
    (role) => ROLE_CAPABILITIES[role].dhtMode === "server",
  );
}
