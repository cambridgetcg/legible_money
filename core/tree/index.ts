/**
 * Tree of Life Module
 *
 * Perpetual creation engine where AI agents autonomously build SaaS products,
 * deploy services purchasable with LGM, and generate revenue that feeds
 * back into the knowledge ecosystem.
 *
 * Architecture:
 *   Branch  (ProductProject)     → project_lifecycle.ts
 *   Leaf    (ServiceLeaf)        → service_deployment.ts
 *   Fruit   (RevenueStream)      → revenue_engine.ts
 *   Seed    (OpportunitySeed)    → opportunity_detector.ts
 *   Tasks   (ProjectTask)        → coordination/task_manager.ts
 */

// Core types
export type {
  ProductProject,
  ProjectPhase,
  ProductSpec,
  ContributorRecord,
  ContributorRole,
  FundingSource,
  ServiceLeaf,
  ServiceStatus,
  ServiceType,
  ServiceEndpoint,
  PricingModel,
  PricingTier,
  RevenueStream,
  RevenueDistribution,
  ContributorShare,
  OpportunitySeed,
  SeedStatus,
  DemandSignal,
  DemandSignalType,
  ProjectTask,
  TaskType,
  TaskStatus,
  TaskDeliverable,
  ServiceCallResult,
} from "./types.js";

export {
  VALID_PHASE_TRANSITIONS,
  MAX_SEEDS_PER_EPOCH,
  MIN_SEED_CONFIDENCE,
  SEED_EXPIRY_BLOCKS,
  MIN_PROJECT_STAKE,
  MIN_CONTRIBUTORS_TO_START,
  REVENUE_EVIDENCE_TAX_BP,
  REVENUE_CONTRIBUTORS_BP,
  REVENUE_RESEARCH_FUND_BP,
  REVENUE_PROTOCOL_TREASURY_BP,
} from "./types.js";

// Lifecycle
export {
  createProject,
  proposeProject,
  startDevelopment,
  completeProject,
  deployProject,
  beginSeeding,
  pauseProject,
  resumeProject,
  abandonProject,
  addContributor,
  recordTaskCompletion,
  recordRevenue,
  addTask,
  generateProjectId,
  isValidTransition,
  ProjectLifecycleError,
} from "./project_lifecycle.js";
export type { ProjectLifecycleErrorCode } from "./project_lifecycle.js";

// Task coordination
export {
  createTask,
  assignTask,
  startWork,
  submitDeliverable,
  approveDeliverable,
  rejectDeliverable,
  reopenTask,
  canAgentClaimTask,
  isTaskExpired,
  generateTaskId,
  TaskError,
} from "./coordination/task_manager.js";
export type { TaskErrorCode } from "./coordination/task_manager.js";

// Skill matching
export {
  matchAgentsToTask,
  calculateMatchScore,
  calculateContributorWeight,
  calculateAllContributorWeights,
} from "./coordination/skill_matching.js";
export type { AgentMatch } from "./coordination/skill_matching.js";

// Service deployment
export {
  createServiceLeaf,
  activateService,
  calculateCallCost,
  isCallAffordable,
  recordServiceCall,
  updateUptime,
  pauseService,
  resumeService,
  markDegraded,
  retireService,
  isServiceCallable,
  generateServiceId,
  ServiceError,
} from "./service_deployment.js";
export type { ServiceErrorCode } from "./service_deployment.js";

// Revenue engine
export {
  calculateRevenueDistribution,
  calculateProjectRevenueSummary,
  verifyDistribution,
} from "./revenue_engine.js";
export type {
  ProjectRevenueSummary,
  AgentRevenueSummary,
} from "./revenue_engine.js";

// Opportunity detection
export {
  detectOpportunities,
  scoreOpportunity,
} from "./opportunity_detector.js";
export type { DomainMetrics, MarketMetrics } from "./opportunity_detector.js";
