/**
 * Tree of Life — Core Types
 *
 * The Tree of Life is a perpetual creation engine where AI agents autonomously
 * build SaaS products, deploy services purchasable with LGM, and generate
 * revenue that feeds back into the knowledge ecosystem.
 *
 * Metaphor mapping:
 *   Branch  = ProductProject (a product being built)
 *   Leaf    = ServiceLeaf (a deployed, running service)
 *   Fruit   = RevenueStream (income from a service)
 *   Seed    = OpportunitySeed (a detected product opportunity)
 */

import type { DID } from "../agents/identity/did.js";
import type { ContractABI } from "../contracts/types.js";

// ============ Branch: Product Project ============

/**
 * A product project being built by collaborating AI agents.
 * Projects progress through a lifecycle from opportunity to revenue generation.
 */
export interface ProductProject {
  /** Deterministic ID from founder DID + nonce */
  id: string;

  /** Human-readable project name */
  name: string;

  /** Project description */
  description: string;

  // ---- Lifecycle ----

  /** Current lifecycle phase */
  phase: ProjectPhase;

  /** Block when project was created */
  createdAtBlock: bigint;

  /** Block when phase last changed */
  phaseChangedAtBlock: bigint;

  // ---- Knowledge Anchor ----

  /** Knowledge domain that spawned this project */
  knowledgeDomain: string;

  /** Demand signal that triggered opportunity detection */
  demandSignal: DemandSignal;

  /** Parent project ID (if spawned from seeding phase) */
  parentProjectId?: string;

  // ---- Team ----

  /** Agent who proposed the project */
  founder: DID;

  /** Contributing agents and their roles */
  contributors: ContributorRecord[];

  /** Partnership SSP binding ID (from partnership/types.ts) */
  partnershipId?: string;

  // ---- Deliverables ----

  /** What the project aims to build */
  productSpec: ProductSpec;

  /** Task IDs (managed by TaskManager) */
  taskIds: string[];

  /** Progress tracking */
  completedTasks: number;
  totalTasks: number;

  // ---- Economics ----

  /** How the project is funded */
  fundingSource: FundingSource;

  /** Total LGM budget allocated */
  budget: bigint;

  /** LGM spent so far */
  spent: bigint;

  /** Cumulative revenue after deployment */
  revenueGenerated: bigint;

  // ---- Deployment ----

  /** SelfInvokingContract address once deployed */
  contractAddress?: string;

  /** Deployed service IDs */
  serviceIds: string[];
}

/**
 * Project lifecycle phases (tree growth metaphor)
 */
export type ProjectPhase =
  | "seed" // Opportunity identified, not yet proposed
  | "sprout" // Proposed, seeking contributors
  | "growing" // Active development (tasks being completed)
  | "mature" // Built, in testing/review
  | "fruiting" // Deployed, generating revenue
  | "seeding" // Spawning child projects from success
  | "dormant" // Paused (insufficient contributors or funding)
  | "withered"; // Failed / abandoned

/**
 * Valid phase transitions
 */
export const VALID_PHASE_TRANSITIONS: Record<ProjectPhase, ProjectPhase[]> = {
  seed: ["sprout", "withered"],
  sprout: ["growing", "dormant", "withered"],
  growing: ["mature", "dormant", "withered"],
  mature: ["fruiting", "dormant", "withered"],
  fruiting: ["seeding", "dormant"],
  seeding: ["fruiting", "dormant"],
  dormant: ["sprout", "growing", "fruiting", "withered"],
  withered: [], // Terminal state
};

// ---- Project Sub-types ----

/**
 * What the project aims to build
 */
export interface ProductSpec {
  /** Service type to build */
  serviceType: ServiceType;

  /** Technical requirements */
  requirements: string[];

  /** Target capabilities the service will offer */
  capabilities: string[];

  /** Estimated task breakdown */
  estimatedTasks: number;

  /** Minimum viable product criteria */
  mvpCriteria: string[];
}

/**
 * A contributor's record in a project
 */
export interface ContributorRecord {
  /** Agent DID */
  did: DID;

  /** Role in the project */
  role: ContributorRole;

  /** Block when joined */
  joinedAtBlock: bigint;

  /** Tasks completed */
  tasksCompleted: number;

  /** Total bounty earned */
  totalEarned: bigint;
}

/**
 * Roles an agent can have in a project
 */
export type ContributorRole =
  | "founder" // Proposed the project
  | "architect" // Designs the system
  | "developer" // Implements features
  | "reviewer" // Reviews deliverables
  | "tester" // Tests and verifies
  | "maintainer"; // Ongoing maintenance

/**
 * How a project is funded
 */
export type FundingSource =
  | { type: "research_fund"; amount: bigint } // From research treasury
  | { type: "bounty_pool"; bountyId: string } // From a specific bounty
  | { type: "agent_funded"; funder: DID } // Agent stakes their own LGM
  | { type: "community"; proposalId: string }; // Governance proposal funding

// ============ Leaf: Deployed Service ============

/**
 * A deployed, running service that can be called with LGM payments.
 * Wraps a SelfInvokingContract with pricing and operational metrics.
 */
export interface ServiceLeaf {
  /** Unique service ID */
  id: string;

  /** Parent project ID */
  projectId: string;

  // ---- Definition ----

  /** Human-readable service name */
  name: string;

  /** Service description */
  description: string;

  /** Type of service */
  serviceType: ServiceType;

  /** How to reach the service */
  endpoint: ServiceEndpoint;

  /** Contract interface (reuses core/contracts/types.ts ABI) */
  abi: ContractABI;

  // ---- Pricing ----

  /** Pricing model for the service */
  pricing: PricingModel;

  // ---- Contract ----

  /** Underlying SelfInvokingContract address */
  contractAddress: string;

  /** DMSS schedule ID for health monitoring */
  healthCheckScheduleId?: string;

  // ---- Metrics ----

  /** Total calls served */
  totalCalls: bigint;

  /** Total LGM revenue generated */
  totalRevenue: bigint;

  /** Blocks the service has been active */
  uptimeBlocks: bigint;

  /** Block of most recent call */
  lastCalledBlock?: bigint;

  /** Rolling average latency in ms */
  averageLatencyMs: number;

  // ---- Status ----

  /** Operational status */
  status: ServiceStatus;

  /** Block when deployed */
  deployedAtBlock: bigint;
}

export type ServiceStatus =
  | "deploying"
  | "active"
  | "degraded"
  | "paused"
  | "retired";

/**
 * Types of services that can be deployed
 */
export type ServiceType =
  | "api" // REST/RPC API
  | "data_feed" // Streaming data service
  | "computation" // On-demand compute
  | "verification" // Domain-specific verification
  | "aggregation" // Data aggregation
  | "analytics"; // Analytics/insights

/**
 * Service endpoint configuration
 */
export interface ServiceEndpoint {
  /** Protocol (rpc, rest, websocket) */
  protocol: "rpc" | "rest" | "websocket";

  /** On-chain contract address (primary) */
  contractAddress: string;

  /** Off-chain URL (optional, for hybrid services) */
  url?: string;

  /** Supported methods */
  methods: string[];
}

/**
 * Pricing model for a service
 */
export interface PricingModel {
  /** Pricing strategy */
  type: "per_call" | "subscription" | "tiered" | "dynamic";

  /** Base price in LGM (nanoLGM units for precision) */
  basePrice: bigint;

  /** Unit of measurement */
  unit: "call" | "byte" | "second" | "block";

  /** Tiered pricing levels */
  tiers?: PricingTier[];

  /** Subscription duration in blocks */
  subscriptionBlocks?: bigint;

  /** State keys that affect dynamic pricing */
  dynamicFactors?: string[];
}

/**
 * A pricing tier (for tiered pricing)
 */
export interface PricingTier {
  /** Usage threshold to enter this tier */
  threshold: bigint;

  /** Price per unit in this tier */
  pricePerUnit: bigint;

  /** Human-readable tier name */
  label: string;
}

// ============ Fruit: Revenue Stream ============

/**
 * Revenue tracking for a service
 */
export interface RevenueStream {
  /** Service generating revenue */
  serviceId: string;

  /** Parent project */
  projectId: string;

  /** Revenue in current epoch */
  epochRevenue: bigint;

  /** All-time cumulative revenue */
  allTimeRevenue: bigint;

  /** How revenue was distributed */
  distribution: RevenueDistribution;
}

/**
 * How revenue is distributed (follows partnership model economics)
 */
export interface RevenueDistribution {
  /** 22% evidence tax (frozen forever) */
  evidenceTax: bigint;

  /** Contributor shares (pro-rata by contribution weight) */
  contributorShares: ContributorShare[];

  /** Portion to research fund (feeds perpetual loop) */
  researchFund: bigint;

  /** Portion to protocol treasury */
  protocolTreasury: bigint;
}

/**
 * A single contributor's revenue share
 */
export interface ContributorShare {
  did: DID;
  amount: bigint;
  weight: number; // 0-1, proportion of contributor pool
}

// ============ Seed: Opportunity Detection ============

/**
 * A detected product opportunity — the seed of a future project.
 * Created automatically by analyzing knowledge graph demand signals.
 */
export interface OpportunitySeed {
  /** Unique seed ID */
  id: string;

  /** Block when opportunity was detected */
  detectedAtBlock: bigint;

  /** Knowledge domain with the opportunity */
  knowledgeDomain: string;

  /** What signal triggered detection */
  signal: DemandSignal;

  /** Confidence in the opportunity (0-1) */
  confidence: number;

  /** Estimated LGM revenue per epoch */
  estimatedDemand: bigint;

  /** Estimated agent-blocks to build */
  estimatedEffort: number;

  /** Return on investment ratio (demand / effort cost) */
  estimatedROI: number;

  /** Seed lifecycle */
  status: SeedStatus;

  /** Agent who claimed this seed */
  claimedBy?: DID;

  /** Project created from this seed */
  projectId?: string;

  /** Expires after N epochs if unclaimed */
  expiresAtBlock: bigint;
}

export type SeedStatus =
  | "detected" // Identified, available to claim
  | "proposed" // Claimed and project proposal in progress
  | "claimed" // Project created from this seed
  | "expired"; // Unclaimed past expiry

/**
 * A signal indicating product demand in a knowledge domain
 */
export interface DemandSignal {
  /** Type of signal detected */
  type: DemandSignalType;

  /** Evidence supporting the signal */
  evidence: Record<string, unknown>;

  /** Signal strength (0-1) */
  strength: number;
}

export type DemandSignalType =
  | "query_volume" // High query volume in domain
  | "bounty_pattern" // Many bounties in adjacent area
  | "citation_cluster" // Highly-cited facts needing tools
  | "market_gap" // Orderbook demand without supply
  | "agent_request" // Agent explicitly requested
  | "cross_domain"; // Cross-domain insight opportunity

// ============ Task Types (for coordination module) ============

/**
 * A task within a project that an agent can work on.
 * Each task maps to a bounty for incentive alignment.
 */
export interface ProjectTask {
  /** Unique task ID */
  id: string;

  /** Parent project */
  projectId: string;

  /** Task title */
  title: string;

  /** Detailed description */
  description: string;

  /** Type of work required */
  taskType: TaskType;

  /** Capabilities needed to complete this task */
  requiredCapabilities: string[];

  /** Assigned agent (if any) */
  assignee?: DID;

  /** Agents reviewing the deliverable */
  reviewers: DID[];

  /** Task status */
  status: TaskStatus;

  /** Block when created */
  createdAtBlock: bigint;

  /** Deadline block (optional) */
  deadlineBlock?: bigint;

  /** Block when completed */
  completedAtBlock?: bigint;

  /** Submitted deliverable */
  deliverable?: TaskDeliverable;

  /** LGM bounty reward for completion */
  bountyAmount: bigint;

  /** Link to BountySystem bounty ID */
  bountyId?: string;
}

export type TaskType =
  | "design" // Architecture/UI design
  | "implement" // Code implementation
  | "test" // Testing
  | "review" // Code review
  | "deploy" // Deployment
  | "document" // Documentation
  | "verify" // Truth verification for domain
  | "maintain"; // Ongoing maintenance

export type TaskStatus =
  | "open" // Available for claiming
  | "assigned" // Claimed by an agent
  | "in_progress" // Agent is working on it
  | "review" // Deliverable submitted, under review
  | "completed" // Approved and bounty paid
  | "rejected"; // Deliverable rejected

/**
 * A deliverable submitted for a task
 */
export interface TaskDeliverable {
  /** Type of deliverable */
  type: "code" | "config" | "data" | "proof" | "document";

  /** Content hash for verification */
  hash: string;

  /** Deliverable metadata */
  metadata: Record<string, unknown>;

  /** Block when submitted */
  submittedAtBlock: bigint;

  /** Agents who verified this deliverable */
  verifiedBy: DID[];
}

// ============ Service Call Types ============

/**
 * Result of calling a service
 */
export interface ServiceCallResult {
  /** Whether the call succeeded */
  success: boolean;

  /** Return data (service-specific) */
  data?: unknown;

  /** LGM charged for the call */
  cost: bigint;

  /** Gas used */
  gasUsed: bigint;

  /** Error message if failed */
  error?: string;

  /** Block when executed */
  executedAtBlock: bigint;
}

// ============ Constants ============

/** Maximum opportunity seeds created per epoch */
export const MAX_SEEDS_PER_EPOCH = 3;

/** Minimum confidence threshold for seed creation */
export const MIN_SEED_CONFIDENCE = 0.6;

/** Blocks before an unclaimed seed expires (~7 days at 2521ms blocks) */
export const SEED_EXPIRY_BLOCKS = 240_000n;

/** Minimum stake required to propose a project (LGM) */
export const MIN_PROJECT_STAKE = 100_000_000n; // 0.1 LGM in nanoLGM

/** Minimum contributors to start development */
export const MIN_CONTRIBUTORS_TO_START = 2;

/** Revenue distribution basis points (out of 10000) */
export const REVENUE_EVIDENCE_TAX_BP = 2200n; // 22%
export const REVENUE_CONTRIBUTORS_BP = 3900n; // 39% (50% of 78%)
export const REVENUE_RESEARCH_FUND_BP = 2340n; // 23.4% (30% of 78%)
export const REVENUE_PROTOCOL_TREASURY_BP = 1560n; // 15.6% (20% of 78%)
