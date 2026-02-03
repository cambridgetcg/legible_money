/**
 * Truth Module
 *
 * Exports for the truth contribution and distribution system.
 */

export {
  // Constants
  BASE_CONTRIBUTION_WEIGHTS,
  COUNTERFACTUAL_ADJUSTMENTS,
  ANTI_GAMING,
  RESEARCH_SHARE_RATE,
  RESEARCH_TREASURY_DID,

  // Types
  type ContributionType,
  type TruthContributionRecord,
  type ContributionContext,
  type ContributionMetrics,
  type ContributionWeight,
  type RewardDistribution,
  type ContributionLedger,

  // Functions
  calculateContributionWeight,
  distributeRewards,
  distributeFactRewards,

  // Implementation
  InMemoryContributionLedger,
} from './contribution.js';
