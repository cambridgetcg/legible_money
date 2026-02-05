/**
 * Expirable Tokens Module
 *
 * Implements token expiration using the Dynamic Array method from the patent:
 * "System And Method For Schedule Message And Self-Invoking Smart Contracts"
 *
 * This module enables:
 * - Tokens with limited lifespan (default ~6 months)
 * - Efficient O(1) balance tracking via child accounts
 * - Automatic retrieval of expired tokens
 * - RAI (Regenerative Agent Income) distribution system
 *
 * Key Components:
 * - types.ts: Core types for expirable tokens
 * - child_account.ts: Child Account Manager (per-period tracking)
 * - expiration_tracker.ts: Dynamic Array implementation
 * - retrieval_engine.ts: Expired token retrieval and RAI
 *
 * The circular economy model:
 * ┌────────────────────────────────────────────────────────┐
 * │                                                         │
 * │  ┌─────────┐     ┌──────────┐     ┌──────────────┐     │
 * │  │  Mint   │────▶│  Active  │────▶│   Expired    │     │
 * │  │ Tokens  │     │  Balance │     │   Balance    │     │
 * │  └─────────┘     └──────────┘     └──────────────┘     │
 * │       ▲                                  │             │
 * │       │                                  ▼             │
 * │  ┌─────────┐                       ┌──────────┐       │
 * │  │   RAI   │◀──────────────────────│ Retrieval│       │
 * │  │  Dist.  │                       │  Engine  │       │
 * │  └─────────┘                       └──────────┘       │
 * │                                                         │
 * └────────────────────────────────────────────────────────┘
 */

// ============ Types ============
export type {
  // Core token types
  ExpirableBalance,
  ExpirationPeriod,
  ChildAccount,
  ExpirableMint,
  MintSource,

  // Dynamic Array types
  DynamicArrayState,
  PeriodTransition,

  // Configuration
  ExpirationConfig,
  AccountType,
  AccountTypeConfig,

  // Retrieval types
  RetrievalRequest,
  RetrievalStatus,
  AccountRetrievalResult,
  BatchRetrievalResult,

  // RAI types
  RAIHoldingAccount,
  RAIDistribution,
  RAIEligibility,

  // Transfer types
  ExpirableTransfer,
  TransferPolicy,
  TransferWithFeeResult,
  UnlockResult,

  // Participation types
  ParticipationMetrics,
  ParticipationScore,
  ParticipationConfig,
  ParticipationResolver,

  // Events
  TokenEvent,
} from "./types.js";

export {
  DEFAULT_EXPIRATION_CONFIG,
  DEFAULT_TRANSFER_POLICY,
  DEFAULT_PARTICIPATION_CONFIG,
  TokenError,
  type TokenErrorCode,
} from "./types.js";

// ============ Child Account Manager ============
export { ChildAccountManager } from "./child_account.js";

// ============ Expiration Tracker (Dynamic Array) ============
export { ExpirationTracker } from "./expiration_tracker.js";

// ============ Retrieval Engine (RAI) ============
export { RetrievalEngine } from "./retrieval_engine.js";

// ============ Participation Score Calculator ============
export {
  componentScore,
  calculateDecayFactor,
  mapScoreToPeriods,
  calculateParticipationScore,
  createParticipationResolver,
} from "./participation.js";

// ============ Add New Features Below ============
// (Safe zone for parallel agent work - add new exports here alphabetically)
