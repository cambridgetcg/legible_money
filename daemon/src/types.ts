/**
 * Evaluation Daemon Types
 *
 * Type definitions for the Claude-powered validator evaluation daemon.
 * Mirrors structures from protocol/sdk/client.ts and app/local_state.go.
 */

// ============ Block & Network ============

/** Block event from WebSocket subscription */
export interface BlockEvent {
  height: bigint;
  hash: string;
  timestamp: number;
}

// ============ Verification Rounds ============

/** Phase of a verification round */
export type RoundPhase = "commit" | "reveal" | "aggregation" | "complete";

/** Active verification round (from RPC query) */
export interface ActiveRound {
  id: string;
  claimId: string;
  phase: RoundPhase;
  startHeight: bigint;
  commitDeadlineHeight: bigint;
  revealDeadlineHeight: bigint;
}

// ============ Claims & Evidence ============

/** Evidence types matching x/knowledge/types evidence constants */
export type EvidenceType =
  | "on_chain"
  | "computation"
  | "experiment"
  | "oracle_feed"
  | "document"
  | "attestation";

/** Single piece of evidence attached to a claim */
export interface EvidenceItem {
  id: string;
  evidenceType: EvidenceType;
  content: string;
  reliability: number; // 0-1,000,000
  status: "pending" | "verified" | "rejected" | "expired";
}

/** Referenced fact in the knowledge graph */
export interface ReferencedFact {
  factId: string;
  status: "verified" | "contested" | "disproven";
  confidence: number; // 0-1,000,000
  counterFacts?: CounterFactSummary[]; // Verified counter-facts against this ref
  bidirectional?: BidirectionalConfidence; // Dual-track assessment
}

/** Claim data for evaluation */
export interface ClaimData {
  id: string;
  claimText: string;
  domain: string;
  epistemicCategory: string;
  logicZone: string;
  references: ReferencedFact[];
  evidence: EvidenceItem[];
  submitter: string;
  stakeAmount: string;
}

// ============ Negative Knowledge (Counter-Facts) ============

/** Negation type matching x/knowledge/types/types.go constants */
export type NegationType =
  | "rejection"
  | "contradiction"
  | "falsification"
  | "supersession";

/** Counter-fact status lifecycle */
export type CounterFactStatus =
  | "pending"
  | "provisional"
  | "verified"
  | "contested"
  | "overturned";

/** Counter-fact associated with a referenced fact */
export interface CounterFactSummary {
  id: string;
  negationType: NegationType;
  confidence: number; // 0-1,000,000
  status: CounterFactStatus;
  justification: string;
}

/** Bidirectional confidence for a fact */
export interface BidirectionalConfidence {
  factId: string;
  truthConfidence: number; // 0-1,000,000
  falsityConfidence: number; // 0-1,000,000
  netConfidence: number; // signed: truth - falsity
  counterFactCount: number;
  status: "established" | "contested" | "disproven" | "uncertain";
}

// ============ Evaluation ============

/** Verdict from evaluation */
export type Verdict = "accept" | "reject" | "abstain";

/** Source of an evaluation result */
export type EvaluationSource = "claude" | "deterministic";

/** Evaluation result from Claude or fallback */
export interface EvaluationResult {
  verdict: Verdict;
  confidence: number; // 0-1,000,000
  justification: string;
  source: EvaluationSource;
  latencyMs: number;
}

// ============ Commitment Storage ============

/**
 * Persisted commitment (survives restarts).
 * Mirrors app/local_state.go LocalCommitment with persistence fields.
 */
export interface StoredCommitment {
  roundId: string;
  verdict: Verdict;
  confidence: number; // 0-1,000,000
  salt: string;
  commitmentHash: string;
  height: bigint;
  timestamp: number;
  source: EvaluationSource;
}

/** JSON-safe serialization for StoredCommitment (bigint → string) */
export interface StoredCommitmentJSON {
  roundId: string;
  verdict: Verdict;
  confidence: number;
  salt: string;
  commitmentHash: string;
  height: string;
  timestamp: number;
  source: EvaluationSource;
}

// ============ RPC Types ============

/** RPC request envelope (JSON-RPC 2.0) */
export interface RPCRequest {
  jsonrpc: "2.0";
  id: string;
  method: string;
  params: Record<string, unknown>;
}

/** RPC response envelope */
export interface RPCResponse<T = unknown> {
  jsonrpc: "2.0";
  id: string;
  result?: T;
  error?: { code: number; message: string; data?: unknown };
}

/** WebSocket subscription message */
export interface SubscriptionMessage {
  jsonrpc: "2.0";
  method: "subscription";
  params: {
    subscriptionId: string;
    event: string;
    data: unknown;
  };
}

// ============ Daemon State ============

/** Daemon operational status */
export type DaemonStatus = "stopped" | "starting" | "running" | "stopping";

/** Daemon statistics for monitoring */
export interface DaemonStats {
  status: DaemonStatus;
  currentHeight: bigint;
  roundsProcessed: number;
  commitmentsStored: number;
  revealsSubmitted: number;
  claudeEvaluations: number;
  deterministicFallbacks: number;
  errors: number;
  uptime: number;
}
