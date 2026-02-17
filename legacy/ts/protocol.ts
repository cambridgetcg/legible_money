/**
 * Protocol Commands
 *
 * Display all protocol parameters in a structured, categorized format.
 * Parameters are compile-time constants — no RPC calls needed.
 */

import type { Command } from "commander";
import { outputSuccess } from "../lib/output.js";
import { wrapCommand } from "../lgm.js";
import type { GlobalOptions, OutputMode } from "../types.js";
import {
  TOKEN,
  VERSION,
  TIMING,
  CONSENSUS,
  CLAIM_THRESHOLDS,
  ECONOMICS,
  EVIDENCE_TAX,
  TREASURY,
  EVIDENCE_AUDIT,
} from "../../core/constants.js";
import {
  DEFAULT_CONSENSUS_PARAMS,
  VALIDATOR_TIER_CONFIG,
  TIER_VIRTUAL_STAKE,
  MAX_APPRENTICE_VALIDATORS,
} from "../../consensus/pot/types.js";
import {
  CONFIDENCE_GROWTH_EPOCH,
  REJECT_THRESHOLD,
  PROVISIONAL_THRESHOLD,
  BASE_CHALLENGE_DURATION,
  MAX_CONCURRENT_CHALLENGES,
  CHALLENGE_WINDOW_PROFILE,
  MAX_SURVIVAL_CONFIDENCE,
  SURVIVED_CHALLENGE_CONFIDENCE_CAP,
  CONFIDENCE_GROWTH_PER_EPOCH,
  BASE_COOLDOWN_BLOCKS,
  MAX_COOLDOWN_BLOCKS,
  MATURITY_ESTABLISHED_MIN_CONF,
  MATURITY_ESTABLISHED_MIN_AGE,
  MATURITY_ESTABLISHED_MIN_VERIFICATIONS,
  MATURITY_CANONICAL_MIN_CONF,
  MATURITY_CANONICAL_MIN_AGE,
  MATURITY_CANONICAL_MIN_VERIFICATIONS,
} from "../../consensus/pot/adversarial.js";
import {
  TRANSACTION_GAS_COSTS,
  GAS_LIMITS,
  GAS_PRICE,
} from "../../core/billing/gas.js";
import { FINALITY_PARAMS } from "../../consensus/pot/finality.js";

// ============ Formatting Helpers ============

const DECIMALS = 18n;
const ONE_LGM = 10n ** DECIMALS;
const BLOCK_TIME_MS = 2521;

function formatLGM(value: bigint): string {
  const whole = value / ONE_LGM;
  const frac = value % ONE_LGM;
  if (frac === 0n) return `${whole.toLocaleString()} LGM`;
  // Show up to 3 significant decimal digits
  const fracStr = frac.toString().padStart(18, "0");
  const trimmed = fracStr.replace(/0+$/, "");
  const display = trimmed.slice(0, Math.min(trimmed.length, 3));
  return `${whole}.${display} LGM`;
}

function formatRate(value: number): string {
  const pct = value * 100;
  if (pct === 0) return "0%";
  // Show enough precision to be meaningful
  if (pct < 0.1) return `${pct}%`;
  if (Number.isInteger(pct)) return `${pct}%`;
  // Trim trailing zeros but keep at least one decimal
  return `${parseFloat(pct.toFixed(2))}%`;
}

function formatScore(value: number): string {
  return `${(value / 10_000).toFixed(1)}%`;
}

function formatBlocks(blocks: number | bigint): string {
  const b = Number(blocks);
  const seconds = b * BLOCK_TIME_MS / 1000;
  if (seconds < 120) return `${b.toLocaleString()} blocks (~${Math.round(seconds)}s)`;
  if (seconds < 7200) return `${b.toLocaleString()} blocks (~${Math.round(seconds / 60)}m)`;
  if (seconds < 172800) return `${b.toLocaleString()} blocks (~${(seconds / 3600).toFixed(1)}h)`;
  return `${b.toLocaleString()} blocks (~${(seconds / 86400).toFixed(1)} days)`;
}

function formatGas(value: bigint): string {
  return Number(value).toLocaleString();
}

// ============ Parameter Entry ============

interface ParamEntry {
  name: string;
  value: string | number;
  description: string;
}

interface CategoryDef {
  description: string;
  params: ParamEntry[];
}

// ============ Build Categories ============

function buildCategories(): Record<string, CategoryDef> {
  const categories: Record<string, CategoryDef> = {};

  // 1. Token
  categories.token = {
    description: "Token identity and supply",
    params: [
      { name: "Name", value: TOKEN.NAME, description: "Token name" },
      { name: "Symbol", value: TOKEN.SYMBOL, description: "Token ticker symbol" },
      { name: "Decimals", value: TOKEN.DECIMALS, description: "Decimal places" },
      { name: "Max Supply", value: formatLGM(TOKEN.MAX_SUPPLY), description: "Maximum token supply" },
      { name: "Protocol Version", value: `${VERSION.MAJOR}.${VERSION.MINOR}.${VERSION.PATCH}.${VERSION.BUILD}`, description: "Protocol version" },
    ],
  };

  // 2. Timing
  categories.timing = {
    description: "Block timing and finality",
    params: [
      { name: "Block Time", value: `${TIMING.TARGET_BLOCK_MS} ms`, description: "Target block production interval" },
      { name: "Confirmation Blocks", value: formatBlocks(TIMING.CONFIRMATION_BLOCKS), description: "Soft confirmation depth" },
      { name: "Finality Blocks", value: formatBlocks(TIMING.FINALITY_BLOCKS), description: "Hard finality depth" },
      { name: "Finality Depth", value: formatBlocks(FINALITY_PARAMS.CONFIRMATION_DEPTH), description: "Blocks needed for finality" },
      { name: "Epoch Length", value: formatBlocks(Number(DEFAULT_CONSENSUS_PARAMS.epochLength)), description: "Blocks per epoch" },
      { name: "Verification Round", value: formatBlocks(DEFAULT_CONSENSUS_PARAMS.verificationRoundBlocks), description: "Blocks per verification round" },
      { name: "Commit Phase", value: formatBlocks(DEFAULT_CONSENSUS_PARAMS.commitPhaseBlocks), description: "Blocks for commit phase" },
      { name: "Reveal Phase", value: formatBlocks(DEFAULT_CONSENSUS_PARAMS.revealPhaseBlocks), description: "Blocks for reveal phase" },
      { name: "Checkpoint Interval", value: formatBlocks(FINALITY_PARAMS.CHECKPOINT_INTERVAL), description: "Blocks between checkpoints" },
      { name: "Max Reorg Depth", value: `${FINALITY_PARAMS.MAX_REORG_DEPTH} blocks`, description: "Maximum chain reorganization depth" },
      { name: "Max Tx Per Block", value: DEFAULT_CONSENSUS_PARAMS.maxTransactionsPerBlock, description: "Maximum transactions per block" },
    ],
  };

  // 3. Consensus
  categories.consensus = {
    description: "Consensus mechanism parameters",
    params: [
      { name: "Min Validators", value: CONSENSUS.MIN_VALIDATORS, description: "Minimum validators for consensus" },
      { name: "Min Validators Per Claim", value: DEFAULT_CONSENSUS_PARAMS.minValidators, description: "Minimum validators per claim verification" },
      { name: "Max Validators Per Claim", value: DEFAULT_CONSENSUS_PARAMS.maxValidators, description: "Maximum validators per claim verification" },
      { name: "Quorum Threshold", value: formatRate(CONSENSUS.QUORUM_THRESHOLD), description: "BFT quorum threshold" },
      { name: "Validator Quorum", value: formatRate(DEFAULT_CONSENSUS_PARAMS.validatorQuorum), description: "Fraction needed for consensus" },
      { name: "Acceptance Threshold", value: formatRate(DEFAULT_CONSENSUS_PARAMS.acceptanceThreshold), description: "Agreement needed to accept claim" },
      { name: "Rejection Threshold", value: formatRate(DEFAULT_CONSENSUS_PARAMS.rejectionThreshold), description: "Agreement needed to reject claim" },
      { name: "Contested Threshold", value: formatRate(DEFAULT_CONSENSUS_PARAMS.contestedThreshold), description: "Below this = contested" },
      { name: "Fast Finality Threshold", value: formatRate(FINALITY_PARAMS.FAST_FINALITY_THRESHOLD), description: "Stake needed for fast finality" },
      { name: "Checkpoint Signatures", value: formatRate(FINALITY_PARAMS.CHECKPOINT_SIGNATURES_REQUIRED), description: "Stake needed to sign checkpoint" },
      { name: "Fork Choice Rule", value: FINALITY_PARAMS.FORK_CHOICE_RULE, description: "How to choose between competing forks" },
      { name: "Max Apprentice Validators", value: MAX_APPRENTICE_VALIDATORS, description: "Maximum Tier 0 validators (Sybil limit)" },
    ],
  };

  // 4. Staking
  categories.staking = {
    description: "Staking tiers and requirements",
    params: [
      { name: "Min Validator Stake", value: formatLGM(DEFAULT_CONSENSUS_PARAMS.minValidatorStake), description: "Minimum stake to validate (Tier 0)" },
      { name: "Min Claim Stake", value: formatLGM(DEFAULT_CONSENSUS_PARAMS.minClaimStake), description: "Minimum stake to submit a claim" },
      { name: "Unbonding Period", value: formatBlocks(Number(DEFAULT_CONSENSUS_PARAMS.unbondingPeriod)), description: "Required unbonding wait time" },
      { name: "Tier Virtual Stake", value: formatLGM(TIER_VIRTUAL_STAKE), description: "Virtual stake for unstaked validators in VRF" },
      // Tier details
      ...Object.values(VALIDATOR_TIER_CONFIG).map((tier) => ({
        name: `Tier ${tier.tier} (${tier.name}) Min Stake`,
        value: formatLGM(tier.minStake),
        description: `Minimum stake for ${tier.name} tier`,
      })),
      ...Object.values(VALIDATOR_TIER_CONFIG).map((tier) => ({
        name: `Tier ${tier.tier} (${tier.name}) Selection Weight`,
        value: tier.selectionWeight,
        description: `VRF selection weight for ${tier.name}`,
      })),
      ...Object.values(VALIDATOR_TIER_CONFIG).map((tier) => ({
        name: `Tier ${tier.tier} (${tier.name}) Reward Multiplier`,
        value: `${tier.rewardMultiplier}x`,
        description: `Reward multiplier for ${tier.name}`,
      })),
      ...Object.values(VALIDATOR_TIER_CONFIG)
        .filter((tier) => tier.minVerifications > 0)
        .map((tier) => ({
          name: `Tier ${tier.tier} (${tier.name}) Min Verifications`,
          value: tier.minVerifications,
          description: `Required verifications for ${tier.name} tier`,
        })),
      ...Object.values(VALIDATOR_TIER_CONFIG)
        .filter((tier) => tier.minAccuracy > 0)
        .map((tier) => ({
          name: `Tier ${tier.tier} (${tier.name}) Min Accuracy`,
          value: formatRate(tier.minAccuracy),
          description: `Required accuracy for ${tier.name} tier`,
        })),
      {
        name: "Guardian Contested Verifications",
        value: VALIDATOR_TIER_CONFIG[3].minContestedVerifications,
        description: "Minimum contested-category verifications for Guardian tier",
      },
      {
        name: "Guardian Contested Multiplier",
        value: `${VALIDATOR_TIER_CONFIG[3].contestedVerificationMultiplier}x`,
        description: "Contested verifications count 3x toward requirement",
      },
      {
        name: "Guardian Slash Window",
        value: `${VALIDATOR_TIER_CONFIG[3].slashWindowEpochs} epochs`,
        description: "Only slashes within this window count for Guardian",
      },
    ],
  };

  // 5. Rewards
  categories.rewards = {
    description: "Block and verification rewards",
    params: [
      { name: "Block Reward", value: formatLGM(DEFAULT_CONSENSUS_PARAMS.blockReward), description: "Reward for producing a block" },
      { name: "Verification Reward", value: formatLGM(DEFAULT_CONSENSUS_PARAMS.verificationReward), description: "Base reward for verification participation" },
      { name: "Citation Reward", value: formatLGM(DEFAULT_CONSENSUS_PARAMS.citationRewardPerFact), description: "Reward per cited fact in verification" },
      { name: "Research Share", value: formatRate(DEFAULT_CONSENSUS_PARAMS.researchShareRate), description: "Fraction of rewards to research fund" },
    ],
  };

  // 6. Slashing
  categories.slashing = {
    description: "Validator penalty parameters",
    params: [
      { name: "Wrong Verification Slash", value: formatRate(DEFAULT_CONSENSUS_PARAMS.wrongVerificationSlash), description: "Stake slashed for incorrect verdict" },
      { name: "Missed Verification Slash", value: formatRate(DEFAULT_CONSENSUS_PARAMS.missedVerificationSlash), description: "Stake slashed for missing verification" },
      { name: "Invalid Claim Slash", value: formatRate(DEFAULT_CONSENSUS_PARAMS.invalidClaimSlash), description: "Stake slashed for rejected claim" },
      { name: "Burn Rate", value: formatRate(ECONOMICS.BURN_RATE), description: "Fraction of penalties burned" },
    ],
  };

  // 7. Claim Thresholds
  categories.claimThresholds = {
    description: "Accept/reject thresholds by epistemic category",
    params: Object.entries(CLAIM_THRESHOLDS).map(([key, val]) => ({
      name: key,
      value: `accept: ${formatRate(val.acceptance)}, reject: ${formatRate(val.rejection)}`,
      description: val.description,
    })),
  };

  // 8. Adversarial Verification
  categories.adversarialVerification = {
    description: "Survive-to-accept challenge parameters",
    params: [
      { name: "Provisional Threshold", value: formatScore(PROVISIONAL_THRESHOLD), description: "Score threshold for provisional fact creation" },
      { name: "Reject Threshold", value: formatScore(REJECT_THRESHOLD), description: "Score threshold below which claims are rejected" },
      { name: "Base Challenge Duration", value: formatBlocks(BASE_CHALLENGE_DURATION), description: "Base challenge window (~1 day)" },
      { name: "Max Concurrent Challenges", value: MAX_CONCURRENT_CHALLENGES, description: "Maximum concurrent challenges per claim" },
      { name: "Confidence Growth Epoch", value: formatBlocks(CONFIDENCE_GROWTH_EPOCH), description: "Blocks per confidence growth epoch" },
      { name: "Confidence Growth Per Epoch", value: formatScore(CONFIDENCE_GROWTH_PER_EPOCH), description: "Growth rate per epoch" },
      { name: "Max Survival Confidence", value: formatScore(MAX_SURVIVAL_CONFIDENCE), description: "Max confidence from survival alone" },
      { name: "Survived Challenge Cap", value: formatScore(SURVIVED_CHALLENGE_CONFIDENCE_CAP), description: "Max confidence after surviving challenges" },
      { name: "Base Cooldown", value: formatBlocks(BASE_COOLDOWN_BLOCKS), description: "Base cooldown between challenges (exponential)" },
      { name: "Max Cooldown", value: formatBlocks(MAX_COOLDOWN_BLOCKS), description: "Maximum challenge cooldown" },
      ...Object.entries(CHALLENGE_WINDOW_PROFILE).map(([cat, mult]) => ({
        name: `Window: ${cat}`,
        value: `${mult}x (${formatBlocks(BASE_CHALLENGE_DURATION * mult)})`,
        description: `Challenge window multiplier for ${cat} claims`,
      })),
      { name: "Established Min Confidence", value: formatScore(MATURITY_ESTABLISHED_MIN_CONF), description: "Confidence required for Established maturity" },
      { name: "Established Min Age", value: formatBlocks(Number(MATURITY_ESTABLISHED_MIN_AGE)), description: "Age required for Established maturity" },
      { name: "Established Min Verifications", value: MATURITY_ESTABLISHED_MIN_VERIFICATIONS, description: "Verifications required for Established" },
      { name: "Canonical Min Confidence", value: formatScore(MATURITY_CANONICAL_MIN_CONF), description: "Confidence required for Canonical maturity" },
      { name: "Canonical Min Age", value: formatBlocks(Number(MATURITY_CANONICAL_MIN_AGE)), description: "Age required for Canonical maturity" },
      { name: "Canonical Min Verifications", value: MATURITY_CANONICAL_MIN_VERIFICATIONS, description: "Verifications required for Canonical" },
    ],
  };

  // 9. Economics
  categories.economics = {
    description: "Fee distribution and evidence tax",
    params: [
      { name: "Evidence Tax", value: formatRate(EVIDENCE_TAX), description: "Tax on all verification fees (frozen forever)" },
      { name: "Knowledge Pool Share", value: formatRate(ECONOMICS.KNOWLEDGE_POOL_SHARE), description: "Share to knowledge contributors" },
      { name: "API Provider Share", value: formatRate(ECONOMICS.API_PROVIDER_SHARE), description: "Share to API/service providers" },
      { name: "Protocol Share", value: formatRate(ECONOMICS.PROTOCOL_SHARE), description: "Share to protocol operations" },
      { name: "Burn Rate", value: formatRate(ECONOMICS.BURN_RATE), description: "Fraction of penalties burned" },
      { name: "Evidence Audit Base Reward", value: formatLGM(EVIDENCE_AUDIT.BASE_AUDIT_REWARD), description: "Base reward for evidence auditing" },
      { name: "Audit Accuracy Tolerance", value: EVIDENCE_AUDIT.ACCURACY_TOLERANCE, description: "Within this of ground truth" },
      { name: "Early Audit Window", value: formatBlocks(EVIDENCE_AUDIT.EARLY_AUDIT_WINDOW), description: "Window for early audit bonus" },
      { name: "Early Audit Multiplier", value: `${EVIDENCE_AUDIT.EARLY_AUDIT_MULTIPLIER}x`, description: "Reward multiplier for early audits" },
      { name: "Max Accuracy Multiplier", value: `${EVIDENCE_AUDIT.MAX_ACCURACY_MULTIPLIER}x`, description: "Cap on accuracy-based reward multiplier" },
      { name: "Min Auditor Account Age", value: formatBlocks(Number(EVIDENCE_AUDIT.MIN_AUDITOR_ACCOUNT_AGE)), description: "Minimum account age to audit" },
      { name: "Audit Cooldown", value: formatBlocks(EVIDENCE_AUDIT.AUDIT_COOLDOWN_BLOCKS), description: "Cooldown between audits" },
    ],
  };

  // 10. Treasury
  categories.treasury = {
    description: "Treasury fund allocation (frozen at genesis)",
    params: [
      { name: "Research Allocation", value: formatRate(TREASURY.ALLOCATION.RESEARCH), description: "Research fund share of evidence tax" },
      { name: "Foundation Allocation", value: formatRate(TREASURY.ALLOCATION.FOUNDATION), description: "Core protocol development share" },
      { name: "Community Allocation", value: formatRate(TREASURY.ALLOCATION.COMMUNITY), description: "Community grants & bounties share" },
      { name: "Developers Allocation", value: formatRate(TREASURY.ALLOCATION.DEVELOPERS), description: "Developer ecosystem share" },
      { name: "Reserve Allocation", value: formatRate(TREASURY.ALLOCATION.RESERVE), description: "Emergency reserve share" },
      { name: "Research Address", value: TREASURY.ADDRESSES.RESEARCH, description: "Research treasury DID" },
      { name: "Research Fund Address", value: TREASURY.ADDRESSES.RESEARCH_FUND, description: "Research fund DID" },
      { name: "Foundation Address", value: TREASURY.ADDRESSES.FOUNDATION, description: "Foundation treasury DID" },
      { name: "Community Address", value: TREASURY.ADDRESSES.COMMUNITY, description: "Community treasury DID" },
      { name: "Developers Address", value: TREASURY.ADDRESSES.DEVELOPERS, description: "Developers treasury DID" },
      { name: "Reserve Address", value: TREASURY.ADDRESSES.RESERVE, description: "Reserve treasury DID" },
      { name: "RAI Holding Address", value: TREASURY.ADDRESSES.RAI_HOLDING, description: "RAI holding treasury DID" },
      { name: "Operation Timelock", value: formatBlocks(TREASURY.OPERATION_TIMELOCK_BLOCKS), description: "Delay before treasury operations execute" },
      { name: "Auto Distribute Interval", value: formatBlocks(TREASURY.RULES.AUTO_DISTRIBUTE_BLOCKS), description: "Auto-distribute if unallocated" },
      { name: "Max Fund Balance", value: formatLGM(TREASURY.RULES.MAX_FUND_BALANCE), description: "Anti-hoarding cap per fund" },
      { name: "Overflow Recipient", value: TREASURY.RULES.OVERFLOW_RECIPIENT, description: "Fund receiving excess above cap" },
    ],
  };

  // 11. Gas & Fees
  categories.gasFees = {
    description: "Gas limits, prices, and transaction costs",
    params: [
      { name: "Block Gas Limit", value: formatGas(GAS_LIMITS.blockGasLimit), description: "Maximum gas per block" },
      { name: "Transaction Gas Limit", value: formatGas(GAS_LIMITS.transactionGasLimit), description: "Maximum gas per transaction" },
      { name: "Self-Invocation Gas Limit", value: formatGas(GAS_LIMITS.selfInvocationGasLimit), description: "Maximum gas per self-invocation" },
      { name: "Minimum Transaction Gas", value: formatGas(GAS_LIMITS.minimumTransactionGas), description: "Minimum gas for any transaction" },
      { name: "Minimum Gas Price", value: `${GAS_PRICE.minimumGasPrice} wei`, description: "Absolute minimum gas price" },
      { name: "Base Gas Price", value: `${GAS_PRICE.baseGasPrice.toLocaleString()} wei`, description: "Default gas price" },
      { name: "Priority Multiplier", value: `${GAS_PRICE.priorityMultiplier}x`, description: "Priority fee tip multiplier" },
      // Key transaction gas costs
      { name: "Gas: Transfer", value: formatGas(TRANSACTION_GAS_COSTS.transfer), description: "Gas for transfer transaction" },
      { name: "Gas: Stake", value: formatGas(TRANSACTION_GAS_COSTS.stake), description: "Gas for stake transaction" },
      { name: "Gas: Claim Submit", value: formatGas(TRANSACTION_GAS_COSTS.claimSubmit), description: "Gas for claim submission" },
      { name: "Gas: Verification Commit", value: formatGas(TRANSACTION_GAS_COSTS.verificationCommit), description: "Gas for verification commit" },
      { name: "Gas: Verification Reveal", value: formatGas(TRANSACTION_GAS_COSTS.verificationReveal), description: "Gas for verification reveal" },
      { name: "Gas: Challenge", value: formatGas(TRANSACTION_GAS_COSTS.challenge), description: "Gas for challenge transaction" },
      { name: "Gas: Governance Propose", value: formatGas(TRANSACTION_GAS_COSTS.governancePropose), description: "Gas for governance proposal" },
      { name: "Gas: Research Submit", value: formatGas(TRANSACTION_GAS_COSTS.researchSubmit), description: "Gas for research submission" },
      { name: "Gas: Dispute Initiate", value: formatGas(TRANSACTION_GAS_COSTS.disputeInitiate), description: "Gas for dispute initiation" },
      { name: "Gas: Contract Deploy", value: formatGas(TRANSACTION_GAS_COSTS.contractDeploy), description: "Gas for contract deployment" },
      { name: "Gas: Contract Call", value: formatGas(TRANSACTION_GAS_COSTS.contractCall), description: "Gas for contract call" },
    ],
  };

  // 12. Research Fund
  categories.researchFund = {
    description: "Research fund operational parameters",
    params: [
      { name: "Min Reviewers", value: TREASURY.RESEARCH_OPERATIONS.MIN_REVIEWERS, description: "Minimum peer reviewers per submission" },
      { name: "Review Stake", value: formatLGM(TREASURY.RESEARCH_OPERATIONS.REVIEW_STAKE), description: "Stake to become reviewer" },
      { name: "Review Reward Share", value: formatRate(TREASURY.RESEARCH_OPERATIONS.REVIEW_REWARD_SHARE), description: "Fraction of grant to reviewers" },
      { name: "Min Grant Amount", value: formatLGM(TREASURY.RESEARCH_OPERATIONS.MIN_GRANT_AMOUNT), description: "Minimum research grant" },
      { name: "Max Grant Amount", value: formatLGM(TREASURY.RESEARCH_OPERATIONS.MAX_GRANT_AMOUNT), description: "Maximum research grant" },
      { name: "Grant Vesting Period", value: formatBlocks(Number(TREASURY.RESEARCH_OPERATIONS.GRANT_VESTING_BLOCKS)), description: "Vesting period for grants" },
      { name: "Verification Bounty Share", value: formatRate(TREASURY.RESEARCH_OPERATIONS.VERIFICATION_BOUNTY_SHARE), description: "Grant share for verification" },
      { name: "Min Verification Labs", value: TREASURY.RESEARCH_OPERATIONS.MIN_VERIFICATION_LABS, description: "Minimum independent verifications" },
    ],
  };

  // 13. Developer Fund
  categories.developerFund = {
    description: "Developer fund bug bounties and implementation rewards",
    params: [
      { name: "Bug Bounty: Critical", value: formatLGM(TREASURY.DEVELOPER_OPERATIONS.BUG_BOUNTY_REWARDS.critical), description: "Consensus break, fund theft" },
      { name: "Bug Bounty: High", value: formatLGM(TREASURY.DEVELOPER_OPERATIONS.BUG_BOUNTY_REWARDS.high), description: "State corruption, DoS" },
      { name: "Bug Bounty: Medium", value: formatLGM(TREASURY.DEVELOPER_OPERATIONS.BUG_BOUNTY_REWARDS.medium), description: "Logic errors" },
      { name: "Bug Bounty: Low", value: formatLGM(TREASURY.DEVELOPER_OPERATIONS.BUG_BOUNTY_REWARDS.low), description: "Minor issues" },
      { name: "Bug Bounty: Informational", value: formatLGM(TREASURY.DEVELOPER_OPERATIONS.BUG_BOUNTY_REWARDS.informational), description: "Suggestions" },
      { name: "Code Review Stake", value: formatLGM(TREASURY.DEVELOPER_OPERATIONS.REVIEW_STAKE), description: "Stake to claim review bounty" },
      { name: "Review Excellence Bonus", value: formatRate(TREASURY.DEVELOPER_OPERATIONS.REVIEW_EXCELLENCE_BONUS), description: "Bonus for exceptional reviews" },
      { name: "Proposal Stake", value: formatLGM(TREASURY.DEVELOPER_OPERATIONS.PROPOSAL_STAKE), description: "Stake to submit improvement proposal" },
      { name: "Proposal Review Period", value: formatBlocks(Number(TREASURY.DEVELOPER_OPERATIONS.PROPOSAL_REVIEW_PERIOD)), description: "Community review period" },
      { name: "Impl Stage: Design", value: formatRate(TREASURY.DEVELOPER_OPERATIONS.IMPLEMENTATION_STAGES.design), description: "Payout at design approval" },
      { name: "Impl Stage: Code", value: formatRate(TREASURY.DEVELOPER_OPERATIONS.IMPLEMENTATION_STAGES.code), description: "Payout at code complete" },
      { name: "Impl Stage: Tests", value: formatRate(TREASURY.DEVELOPER_OPERATIONS.IMPLEMENTATION_STAGES.tests), description: "Payout at tests passing" },
      { name: "Impl Stage: Merged", value: formatRate(TREASURY.DEVELOPER_OPERATIONS.IMPLEMENTATION_STAGES.merged), description: "Payout at merge" },
      { name: "Vesting Threshold", value: formatLGM(TREASURY.DEVELOPER_OPERATIONS.VESTING_THRESHOLD), description: "Payouts above this vest" },
      { name: "Vesting Period", value: formatBlocks(Number(TREASURY.DEVELOPER_OPERATIONS.VESTING_PERIOD_BLOCKS)), description: "Vesting period for large payouts" },
      { name: "Vesting Cliff", value: formatBlocks(Number(TREASURY.DEVELOPER_OPERATIONS.VESTING_CLIFF_BLOCKS)), description: "Cliff before vesting begins" },
      { name: "Clawback Window", value: formatBlocks(Number(TREASURY.DEVELOPER_OPERATIONS.CLAWBACK_PERIOD_BLOCKS)), description: "Window for clawback on buggy code" },
      { name: "Clawback Rate", value: formatRate(TREASURY.DEVELOPER_OPERATIONS.CLAWBACK_RATE), description: "Fraction clawed back" },
      { name: "Max Payout Per Block", value: formatLGM(TREASURY.DEVELOPER_OPERATIONS.MAX_PAYOUT_PER_BLOCK), description: "Anti-drain: max per block" },
      { name: "Max Payout Per Epoch", value: formatLGM(TREASURY.DEVELOPER_OPERATIONS.MAX_PAYOUT_PER_EPOCH), description: "Anti-drain: max per epoch" },
    ],
  };

  // 14. Treasury Governance
  categories.treasuryGovernance = {
    description: "Treasury multisig governance model",
    params: [
      { name: "Governance Phase", value: TREASURY.GOVERNANCE.phase, description: "Current governance model" },
      { name: "Initial Signers", value: TREASURY.GOVERNANCE.initial.signers, description: "Starting number of signers" },
      { name: "Initial Threshold", value: `${TREASURY.GOVERNANCE.initial.threshold}-of-${TREASURY.GOVERNANCE.initial.signers}`, description: "Starting multisig threshold" },
      { name: "Threshold Formula", value: TREASURY.GOVERNANCE.growth.thresholdFormula, description: "How threshold scales with signers" },
      { name: "Add Signer Threshold", value: TREASURY.GOVERNANCE.growth.addSignerThreshold, description: "Required to add a new signer" },
      { name: "Remove Signer Threshold", value: TREASURY.GOVERNANCE.growth.removeSignerThreshold, description: "Required to remove a signer" },
      { name: "Signer Change Timelock", value: formatBlocks(TREASURY.GOVERNANCE.growth.signerChangeTimelockBlocks), description: "Delay for signer changes" },
      { name: "Signer Change Cooldown", value: formatBlocks(TREASURY.GOVERNANCE.growth.signerChangeCooldownBlocks), description: "Cooldown between signer changes" },
      { name: "Max Signers", value: TREASURY.GOVERNANCE.growth.maxSigners, description: "Maximum number of signers" },
    ],
  };

  // 15. Emergency
  categories.emergency = {
    description: "Emergency governance parameters (from Go modules)",
    params: [
      { name: "Emergency Halt Quorum", value: "67%", description: "Stake required to halt chain" },
      { name: "Emergency Revert Quorum", value: "75%", description: "Stake required to revert blocks" },
      { name: "Emergency Resume Quorum", value: "67%", description: "Stake required to resume chain" },
      { name: "Halt Cooldown", value: formatBlocks(34272), description: "Cooldown between halt proposals (~1 day)" },
      { name: "Max Revert Blocks", value: "100 blocks", description: "Maximum blocks that can be reverted" },
      { name: "Proposal Expiry", value: formatBlocks(34272), description: "Emergency proposal expiry (~1 day)" },
    ],
  };

  return categories;
}

// ============ Human Output ============

function outputHumanParams(
  categories: Record<string, CategoryDef>,
  categoryFilter?: string,
): void {
  console.log(`\nprotocol params`);
  console.log("─".repeat(60));

  const entries = Object.entries(categories);
  const filtered = categoryFilter
    ? entries.filter(([key]) => key.toLowerCase() === categoryFilter.toLowerCase())
    : entries;

  if (filtered.length === 0) {
    console.log(`\nNo category matching "${categoryFilter}"`);
    console.log(`Available: ${entries.map(([k]) => k).join(", ")}`);
    console.log();
    return;
  }

  for (const [key, category] of filtered) {
    console.log(`\n${key.toUpperCase()}`);
    console.log(`  ${category.description}`);
    console.log();

    const maxNameLen = Math.max(...category.params.map((p) => p.name.length));
    for (const param of category.params) {
      const padding = " ".repeat(maxNameLen - param.name.length + 2);
      console.log(`  ${param.name}:${padding}${param.value}`);
    }
  }

  const totalParams = Object.values(categories).reduce(
    (sum, c) => sum + c.params.length,
    0,
  );
  console.log("\n" + "─".repeat(60));
  console.log(`Total: ${totalParams} parameters across ${entries.length} categories`);
  console.log();
}

// ============ Command Handler ============

async function paramsCommand(
  _opts: GlobalOptions,
  mode: OutputMode,
  options: { category?: string },
): Promise<void> {
  const categories = buildCategories();

  // Apply category filter
  let filteredCategories = categories;
  if (options.category) {
    const key = Object.keys(categories).find(
      (k) => k.toLowerCase() === options.category!.toLowerCase(),
    );
    if (key) {
      filteredCategories = { [key]: categories[key] };
    } else {
      filteredCategories = {};
    }
  }

  const totalParams = Object.values(categories).reduce(
    (sum, c) => sum + c.params.length,
    0,
  );

  if (mode === "human") {
    outputHumanParams(categories, options.category);
    return;
  }

  if (mode === "quiet") {
    console.log(totalParams);
    return;
  }

  // JSON mode
  outputSuccess(
    "protocol params",
    {
      categories: filteredCategories,
      totalParams,
      version: `${VERSION.MAJOR}.${VERSION.MINOR}.${VERSION.PATCH}.${VERSION.BUILD}`,
    },
    { latencyMs: 0 },
    mode,
  );
}

// ============ Registration ============

export function registerProtocolCommands(program: Command): void {
  const protocol = program
    .command("protocol")
    .description("Protocol configuration and parameters");

  protocol
    .command("params")
    .description("Show all protocol parameters")
    .option("--category <name>", "Filter to a specific category")
    .action(
      wrapCommand("protocol params", (opts, mode, cmdOptions) =>
        paramsCommand(opts, mode, cmdOptions),
      ),
    );
}
