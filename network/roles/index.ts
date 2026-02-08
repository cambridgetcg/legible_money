/**
 * Node Roles Module
 *
 * Role-based node architecture for LGM.
 * Re-exports types, capabilities, and role-specific implementations.
 */

// Core types and capabilities
export {
  type NodeRole,
  type SyncStrategy,
  type RoleCapabilities,
  ROLE_CAPABILITIES,
  ROLE_PROTOCOL_IDS,
  mergeRoleCapabilities,
  roleProcessesBlocks,
  roleParticipatesInConsensus,
  getDHTServerRoles,
} from "./types.js";

// Seed node
export {
  SeedNodeService,
  type SeedNodeConfig,
  type SeedNodeStats,
} from "./seed.js";

// Sentry node
export {
  SentryNodeService,
  type SentryConfig,
  type SentryStats,
} from "./sentry.js";

// Knowledge Gateway
export {
  KnowledgeGatewayService,
  type KnowledgeGatewayConfig,
  type KnowledgeGatewayStats,
  type KnowledgeQuery,
  type KnowledgeQueryResult,
  type KnowledgeBatchQuery,
  type KnowledgeBatchResult,
} from "./knowledge_gateway.js";

// Relay node
export {
  RelayNodeService,
  type RelayConfig,
  type RelayStats,
} from "./relay.js";

// Evidence Oracle
export {
  EvidenceOracleService,
  type EvidenceOracleConfig,
  type EvidenceOracleStats,
  type EvidenceFetchRequest,
  type EvidenceFetchResult,
  type OracleAttestation,
} from "./evidence_oracle.js";
