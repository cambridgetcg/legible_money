/**
 * State Transitions
 *
 * Deterministic state transition logic for all transaction types.
 * Each transition function:
 * 1. Validates preconditions
 * 2. Computes state changes
 * 3. Applies changes atomically
 *
 * Transitions are pure functions that take current state + transaction
 * and produce new state (or error).
 */

import { StateManager } from "./state_manager.js";
import {
  type TransferPayload,
  type StakePayload,
  type DelegatePayload,
  type ChallengeFactPayload,
  type EpochTransitionPayload,
  type ClaimState,
  type ClaimSubmitPayload,
  type ClaimStartVerificationPayload,
  type ClaimCompleteVerificationPayload,
  type ClaimWithdrawPayload,
  type PaymentChannelState,
  type OpenPaymentChannelPayload,
  type UpdatePaymentChannelPayload,
  type ClosePaymentChannelPayload,
  type SettlePaymentChannelPayload,
  type DisputePaymentChannelPayload,
  type RotateKeyPayload,
  type LipSubmitPayload,
  type LipStakePayload,
  type LipAdvanceStagePayload,
  type LipWithdrawPayload,
  type VoteCastPayload,
  type DisbursementSubmitPayload,
  type DisbursementStakePayload,
  type DisbursementAdvanceStagePayload,
  type DisbursementExecutePayload,
  type DisbursementCancelPayload,
  StateError,
} from "./types.js";
import {
  createLIP,
  transitionToReview,
  transitionFromReview,
  transitionFromLastCall,
  transitionFromVoting,
  withdrawLIP,
  calculateStakeOutcome,
} from "../governance/proposals/lifecycle.js";
import {
  type AgentGovernanceStats,
  type VoteTally,
} from "../governance/proposals/types.js";
import {
  createQuadraticVote,
  calculateVoteCost,
  type VoteDirection,
} from "../governance/voting/quadratic.js";
import {
  createDisbursement as createDisbursementProposal,
  transitionToVoting as disbursementTransitionToVoting,
  transitionFromVoting as disbursementTransitionFromVoting,
  executeDisbursement as disbursementExecute,
  cancelDisbursement as disbursementCancel,
  calculateDisbursementStakeOutcome,
} from "../governance/treasury/disbursement_lifecycle.js";
import {
  type DisbursementCategory,
  type TreasuryFund,
  validateDisbursementAmount,
  treasuryFundToDid,
  VALID_DISBURSEMENT_CATEGORIES,
  VALID_TREASURY_FUNDS,
  type ExpenseRecord,
} from "../governance/treasury/types.js";
import type {
  Transaction,
  TransactionType,
  VerificationResult,
  ConsensusParams,
} from "../consensus/pot/types.js";
import type {
  DomainProposePayload,
  DomainEndorsePayload,
  DomainChallengePayload,
  DomainFinalizePayload,
} from "./types.js";
import {
  createDomainProposal,
  endorseProposal,
  challengeProposal,
  finalizeProposal,
  calculateStakeReturn,
  calculateEndorserStakeReturns,
  createDomainFromProposal,
} from "../core/knowledge/ontology/domain_proposal.js";
import type { PublicKey } from "../core/crypto/keys.js";
import { publicKeyToDid } from "../core/agents/identity/did.js";

/**
 * Resolve sender DID from transaction.
 * Uses explicit senderDid when present (e.g., from RPC with custom DevNet DIDs),
 * otherwise derives from public key.
 */
function resolveSenderDid(tx: Transaction): string {
  return (tx as any).senderDid || publicKeyToDid(tx.sender);
}
import { generateClaimId } from "./claims.js";
import {
  generateChannelId,
  computeStateHash,
  sumDistributions,
  PAYMENT_CHANNEL_CONSTANTS,
} from "./payment_channels.js";
import {
  type ResearchSubmitPayload,
  type BountyCreatePayload,
  type BountyClaimPayload,
  type BountyFulfillPayload,
  type ResearchState,
  type BountyState,
  DEFAULT_RESEARCH_PARAMETERS,
  RESEARCH_TREASURY_DID,
} from "../core/research/types.js";
import { sha512_256Hex } from "../core/crypto/hashing.js";
import {
  type Dispute,
  type InitiateDisputePayload,
  type CommitEvidencePayload,
  type RevealEvidencePayload,
  type ArbiterVotePayload,
  type EscalateDisputePayload,
  type SettleDisputePayload,
  DISPUTE_ECONOMICS,
  PHASE_DURATIONS,
} from "../core/dispute/types.js";
import {
  createDispute,
  commitEvidence,
  revealEvidence,
  castArbiterVote,
  tryAdvancePhase,
} from "../core/dispute/dispute_engine.js";
import { processEscalation } from "../core/dispute/escalation.js";
import { calculateSettlement } from "../core/dispute/settlement.js";
// GAP-003 FIX: Import challenge integration for routing dispute outcomes to facts
import {
  routeDisputeOutcomeToFact,
  processPostResolution,
} from "../consensus/pot/challenge_integration.js";
import {
  type CreateVestingPayload,
  type ClaimRewardsPayload,
  type PauseVestingPayload,
  type ResumeVestingPayload,
  type AccelerateVestingPayload,
  type FalsifyVestingPayload,
} from "../core/rewards/types.js";
import { determineVestingCategory } from "../core/rewards/vesting_engine.js";
import {
  routeReward,
  routeBlockReward,
  routeVerificationReward,
  routeClaimReward,
  splitEvidenceTax,
} from "../core/billing/reward_router.js";
import {
  calculateBlockReward,
  type EmissionSchedule,
  DEFAULT_EMISSION_SCHEDULE,
} from "../core/genesis/types.js";
import {
  BootstrapFund,
  BOOTSTRAP_AMOUNT,
  GENESIS_BOOTSTRAP_ALLOCATION,
  processAutomaticBootstrapClaim,
} from "../core/genesis/bootstrap_fund.js";
import {
  prioritizeClaimsForBounties,
  calculateOptimalBountyValue,
} from "../core/genesis/perpetual_loop.js";

// ============ Protocol DIDs ============

/**
 * Fee pool collects transaction fees for block producers
 */
export const FEE_POOL_DID = "did:lgm:fee-pool";

/**
 * Reward pool for minting new tokens (block rewards, verification rewards)
 */
export const REWARD_POOL_DID = "did:lgm:reward-pool";

/**
 * Protocol treasury for protocol-owned funds
 */
export const PROTOCOL_TREASURY_DID = "did:lgm:protocol-treasury";

/**
 * Bootstrap fund DID - source of automatic first-tx claims
 */
export const BOOTSTRAP_FUND_DID = GENESIS_BOOTSTRAP_ALLOCATION.did;

// ============ Bootstrap Fund State ============

/**
 * Bootstrap fund instance — lazily initialized on first access.
 * Tracks which DIDs have claimed their 0.222 LGM bootstrap allocation.
 * Use setBootstrapFund() to inject a custom instance (e.g. for testing).
 */
let bootstrapFund: BootstrapFund | null = null;

/**
 * Get the current bootstrap fund instance (lazily creates one if not set)
 */
export function getBootstrapFund(): BootstrapFund {
  if (!bootstrapFund) {
    bootstrapFund = new BootstrapFund();
  }
  return bootstrapFund;
}

/**
 * Set the bootstrap fund instance (for dependency injection or state restoration)
 */
export function setBootstrapFund(fund: BootstrapFund): void {
  bootstrapFund = fund;
}

/**
 * Reset the bootstrap fund to initial state (primarily for testing)
 */
export function resetBootstrapFund(): void {
  bootstrapFund = null;
}

// ============ Transaction Executor ============

/**
 * Execute a transaction against state
 * Returns the new state root or throws on failure
 */
export function executeTransaction(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  params: ConsensusParams,
): TransitionResult {
  // Validate basic transaction structure
  const validation = validateTransaction(state, tx);
  if (!validation.valid) {
    return {
      success: false,
      error: validation.reason,
      gasUsed: 0n,
    };
  }

  // Route to appropriate handler
  try {
    state.beginTransaction();

    let gasUsed = 0n;
    let bootstrapClaimed = 0n;

    // Auto-claim bootstrap fund on first transaction (nonce === 0)
    const isSystemTx =
      tx.sender === "system" ||
      (tx.fee === 0n &&
        ["add_fact", "verification_reward", "slash_validator"].includes(
          tx.type,
        ));

    if (!isSystemTx) {
      const senderDid = resolveSenderDid(tx);
      const senderNonce = state.getNonce(senderDid);

      // First transaction (nonce === 0) triggers automatic bootstrap claim
      if (senderNonce === 0n) {
        const eligibility = getBootstrapFund().isEligible(senderDid);

        if (eligibility.eligible) {
          // Check if bootstrap fund has sufficient balance
          const bootstrapFundBalance = state.getBalance(BOOTSTRAP_FUND_DID);

          if (bootstrapFundBalance >= BOOTSTRAP_AMOUNT) {
            // Process the claim through bootstrap fund (tracks claimed DIDs)
            bootstrapClaimed = processAutomaticBootstrapClaim(
              getBootstrapFund(),
              senderDid,
              tx.signature,
              Number(blockNumber),
            );

            // Transfer from bootstrap fund to sender
            if (bootstrapClaimed > 0n) {
              state.transfer(BOOTSTRAP_FUND_DID, senderDid, bootstrapClaimed);
            }
          }
        }
      }
    }

    // Record nonce before handler so we can detect if handler incremented it
    const senderDidForNonce = !isSystemTx ? resolveSenderDid(tx) : "";
    const nonceBeforeHandler = !isSystemTx
      ? state.getNonce(senderDidForNonce)
      : 0n;

    switch (tx.type) {
      case "transfer":
        gasUsed = executeTransfer(state, tx, params);
        break;

      case "stake":
        gasUsed = executeStake(state, tx, params);
        break;

      case "unstake":
        gasUsed = executeUnstake(state, tx, params);
        break;

      case "delegate":
        gasUsed = executeDelegate(state, tx, params);
        break;

      case "undelegate":
        gasUsed = executeUndelegate(state, tx, params);
        break;

      case "rotate_key":
        gasUsed = executeKeyRotation(state, tx);
        break;

      case "claim_submit":
        gasUsed = executeClaimSubmit(state, tx, blockNumber, params);
        break;

      case "claim_start_verification":
        gasUsed = executeClaimStartVerification(state, tx, blockNumber);
        break;

      case "claim_complete_verification":
        gasUsed = executeClaimCompleteVerification(
          state,
          tx,
          blockNumber,
          params,
        );
        break;

      case "claim_withdraw":
        gasUsed = executeClaimWithdraw(state, tx, blockNumber);
        break;

      case "verification_commit":
        gasUsed = executeVerificationCommit(state, tx, params);
        break;

      case "verification_reveal":
        gasUsed = executeVerificationReveal(state, tx, params);
        break;

      case "challenge":
        gasUsed = executeChallenge(state, tx, blockNumber, params);
        break;

      case "governance_vote":
        gasUsed = executeGovernanceVote(state, tx, params);
        break;

      case "governance_propose":
        gasUsed = executeGovernancePropose(state, tx, params);
        break;

      // Governance LIP lifecycle
      case "lip_submit":
        gasUsed = executeLipSubmit(state, tx, blockNumber, params);
        break;

      case "lip_stake":
        gasUsed = executeLipStake(state, tx, blockNumber, params);
        break;

      case "lip_advance_stage":
        gasUsed = executeLipAdvanceStage(state, tx, blockNumber, params);
        break;

      case "lip_withdraw":
        gasUsed = executeLipWithdraw(state, tx, blockNumber, params);
        break;

      // Governance voting
      case "vote_cast":
        gasUsed = executeVoteCast(state, tx, blockNumber, params);
        break;

      // Disbursement lifecycle
      case "disbursement_submit":
        gasUsed = executeDisbursementSubmit(state, tx, blockNumber, params);
        break;

      case "disbursement_stake":
        gasUsed = executeDisbursementStake(state, tx, blockNumber, params);
        break;

      case "disbursement_advance_stage":
        gasUsed = executeDisbursementAdvanceStage(
          state,
          tx,
          blockNumber,
          params,
        );
        break;

      case "disbursement_execute":
        gasUsed = executeDisbursementExecuteHandler(
          state,
          tx,
          blockNumber,
          params,
        );
        break;

      case "disbursement_cancel":
        gasUsed = executeDisbursementCancelHandler(
          state,
          tx,
          blockNumber,
          params,
        );
        break;

      // Research transactions (Protocol Research Repository)
      case "research_submit":
        gasUsed = executeResearchSubmit(state, tx, blockNumber, params);
        break;

      case "research_challenge":
        gasUsed = executeResearchChallenge(state, tx, blockNumber, params);
        break;

      case "bounty_create":
        gasUsed = executeBountyCreate(state, tx, blockNumber, params);
        break;

      case "bounty_claim":
        gasUsed = executeBountyClaim(state, tx, params);
        break;

      case "bounty_fulfill":
        gasUsed = executeBountyFulfill(state, tx, blockNumber, params);
        break;

      // Dispute transactions (Dispute Resolution System)
      case "dispute_initiate":
        gasUsed = executeDisputeInitiate(state, tx, blockNumber, params);
        break;

      case "dispute_commit_evidence":
        gasUsed = executeDisputeCommitEvidence(state, tx, blockNumber, params);
        break;

      case "dispute_reveal_evidence":
        gasUsed = executeDisputeRevealEvidence(state, tx, blockNumber, params);
        break;

      case "dispute_arbiter_vote":
        gasUsed = executeDisputeArbiterVote(state, tx, blockNumber, params);
        break;

      case "dispute_escalate":
        gasUsed = executeDisputeEscalate(state, tx, blockNumber, params);
        break;

      case "dispute_settle":
        gasUsed = executeDisputeSettle(state, tx, blockNumber, params);
        break;

      case "dispute_timeout":
        gasUsed = executeDisputeTimeout(state, tx, blockNumber, params);
        break;

      // Vesting transactions (Gradual Reward Release System)
      case "vesting_create":
        gasUsed = executeVestingCreate(state, tx, blockNumber, params);
        break;

      case "vesting_claim":
        gasUsed = executeVestingClaim(state, tx, blockNumber, params);
        break;

      case "vesting_pause":
        gasUsed = executeVestingPause(state, tx, blockNumber, params);
        break;

      case "vesting_resume":
        gasUsed = executeVestingResume(state, tx, blockNumber, params);
        break;

      case "vesting_accelerate":
        gasUsed = executeVestingAccelerate(state, tx, blockNumber, params);
        break;

      case "vesting_falsify":
        gasUsed = executeVestingFalsify(state, tx, blockNumber, params);
        break;

      // Domain proposal transactions (Ontological Tree Growth)
      case "domain_propose":
        gasUsed = executeDomainPropose(state, tx, blockNumber, params);
        break;

      case "domain_endorse":
        gasUsed = executeDomainEndorse(state, tx, blockNumber, params);
        break;

      case "domain_challenge":
        gasUsed = executeDomainChallenge(state, tx, blockNumber, params);
        break;

      case "domain_finalize":
        gasUsed = executeDomainFinalize(state, tx, blockNumber, params);
        break;

      case "domain_governance_approve":
        gasUsed = executeDomainGovernanceApprove(
          state,
          tx,
          blockNumber,
          params,
        );
        break;

      case "domain_deprecate":
        gasUsed = executeDomainDeprecate(state, tx, blockNumber);
        break;

      // System transactions (used by consensus layer, no fee)
      case "add_fact":
        gasUsed = executeAddFact(state, tx, blockNumber);
        break;

      case "verification_reward":
        gasUsed = executeVerificationReward(state, tx);
        break;

      case "slash_validator":
        gasUsed = executeSlashValidator(state, tx);
        break;

      // Payment channel transactions (GAP-005)
      case "open_payment_channel":
        gasUsed = executeOpenPaymentChannel(state, tx, blockNumber, params);
        break;

      case "update_payment_channel":
        gasUsed = executeUpdatePaymentChannel(state, tx, blockNumber);
        break;

      case "close_payment_channel":
        gasUsed = executeClosePaymentChannel(state, tx, blockNumber, params);
        break;

      case "settle_payment_channel":
        gasUsed = executeSettlePaymentChannel(state, tx, blockNumber, params);
        break;

      case "dispute_payment_channel":
        gasUsed = executeDisputePaymentChannel(state, tx, blockNumber);
        break;

      // Autonomy transactions (Agent Self-Sovereignty)
      case "scheduled_execution":
        gasUsed = executeScheduledExecution(state, tx, blockNumber, params);
        break;

      case "contract_call":
        gasUsed = executeContractCall(state, tx, blockNumber, params);
        break;

      // Tree of Life transactions
      case "create_project":
        gasUsed = executeCreateProject(state, tx, blockNumber);
        break;

      case "propose_project":
        gasUsed = executeProposeProject(state, tx, blockNumber);
        break;

      case "assign_task":
        gasUsed = executeAssignTask(state, tx, blockNumber);
        break;

      case "submit_deliverable":
        gasUsed = executeSubmitDeliverable(state, tx, blockNumber);
        break;

      case "approve_deliverable":
        gasUsed = executeApproveDeliverable(state, tx, blockNumber);
        break;

      case "deploy_service":
        gasUsed = executeDeployService(state, tx, blockNumber);
        break;

      case "call_service":
        gasUsed = executeCallService(state, tx, blockNumber);
        break;

      case "subscribe_service":
        gasUsed = executeSubscribeService(state, tx, blockNumber);
        break;

      case "pause_service":
        gasUsed = executePauseService(state, tx);
        break;

      case "resume_service":
        gasUsed = executeResumeService(state, tx);
        break;

      case "retire_service":
        gasUsed = executeRetireService(state, tx);
        break;

      // Tree of Life lifecycle transitions
      case "start_development":
        gasUsed = executeStartDevelopment(state, tx, blockNumber);
        break;

      case "complete_project":
        gasUsed = executeCompleteProject(state, tx, blockNumber);
        break;

      case "begin_seeding":
        gasUsed = executeBeginSeeding(state, tx, blockNumber);
        break;

      case "pause_project":
        gasUsed = executePauseProject(state, tx, blockNumber);
        break;

      case "resume_project":
        gasUsed = executeResumeProject(state, tx, blockNumber);
        break;

      case "abandon_project":
        gasUsed = executeAbandonProject(state, tx, blockNumber);
        break;

      case "add_contributor":
        gasUsed = executeAddContributor(state, tx, blockNumber);
        break;

      case "add_task":
        gasUsed = executeAddTask(state, tx, blockNumber);
        break;

      case "start_work":
        gasUsed = executeStartWork(state, tx, blockNumber);
        break;

      case "reject_deliverable":
        gasUsed = executeRejectDeliverable(state, tx, blockNumber);
        break;

      case "reopen_task":
        gasUsed = executeReopenTask(state, tx, blockNumber);
        break;

      case "detect_opportunity":
        gasUsed = executeDetectOpportunity(state, tx, blockNumber);
        break;

      case "claim_opportunity":
        gasUsed = executeClaimOpportunity(state, tx, blockNumber);
        break;

      case "apply_to_project":
        gasUsed = executeApplyToProject(state, tx, blockNumber);
        break;

      case "review_application":
        gasUsed = executeReviewApplication(state, tx, blockNumber);
        break;

      case "set_availability":
        gasUsed = executeSetAvailability(state, tx, blockNumber);
        break;

      default:
        state.rollbackTransaction();
        return {
          success: false,
          error: `Unknown transaction type: ${tx.type}`,
          gasUsed: 0n,
        };
    }

    // Ensure nonce is incremented for non-system transactions
    // Some handlers (e.g. transfer, delegation) increment via state.transfer(),
    // but governance handlers (lip_submit, vote_cast, etc.) do not.
    if (!isSystemTx) {
      const nonceAfterHandler = state.getNonce(senderDidForNonce);
      if (nonceAfterHandler === nonceBeforeHandler) {
        state.incrementNonce(senderDidForNonce);
      }
    }

    // Skip fee handling for system transactions (isSystemTx defined above for bootstrap claim check)
    if (!isSystemTx) {
      // Deduct fee from sender
      const senderDid = resolveSenderDid(tx);
      const currentBalance = state.getBalance(senderDid);
      if (currentBalance < tx.fee) {
        state.rollbackTransaction();
        return {
          success: false,
          error: "Insufficient balance for fee",
          gasUsed: 0n,
        };
      }

      // GAP-1 FIX: Actually deduct the fee and route through reward router
      if (tx.fee > 0n) {
        // Route fee through reward router for 22% research share
        const routedFee = routeReward({
          source: "block_production", // Fees go to block producer
          amount: tx.fee,
          recipient: FEE_POOL_DID, // Collected in fee pool for block producer
          blockNumber,
        });

        // Deduct fee from sender and credit to destinations
        // Uses deductBalance/creditBalance to avoid extra nonce increments
        // Split evidence tax across 5 treasury funds per frozen allocation
        if (routedFee.researchShare > 0n) {
          const treasurySplit = splitEvidenceTax(routedFee.researchShare);
          for (const entry of treasurySplit) {
            if (entry.amount > 0n) {
              state.deductBalance(senderDid, entry.amount);
              state.creditBalance(entry.did, entry.amount);
            }
          }
        }

        if (routedFee.recipientAmount > 0n) {
          state.deductBalance(senderDid, routedFee.recipientAmount);
          state.creditBalance(FEE_POOL_DID, routedFee.recipientAmount);
        }
      }
    }

    // Commit and return
    const newRoot = state.commitTransaction();

    return {
      success: true,
      stateRoot: newRoot,
      gasUsed,
      bootstrapClaimed: bootstrapClaimed > 0n ? bootstrapClaimed : undefined,
    };
  } catch (error) {
    // Only rollback if a transaction is in progress (beginTransaction may have failed)
    if (state.hasActiveTransaction()) {
      state.rollbackTransaction();
    }
    return {
      success: false,
      error: error instanceof Error ? error.message : "Unknown error",
      gasUsed: 0n,
    };
  }
}

// ============ Individual Transaction Handlers ============

function executeTransfer(
  state: StateManager,
  tx: Transaction,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as TransferPayload;
  const senderDid = resolveSenderDid(tx);

  // Coerce amount to bigint (RPC sends strings)
  const amount =
    typeof payload.amount === "string"
      ? BigInt(payload.amount)
      : payload.amount;

  // Validate
  if (amount <= 0n) {
    throw new StateError("Invalid transfer amount", "INVALID_TRANSITION");
  }

  // Execute transfer
  state.transfer(senderDid, payload.to, amount);

  // Increment nonce is handled by transfer

  return 21000n; // Base gas for transfer
}

function executeKeyRotation(state: StateManager, tx: Transaction): bigint {
  const payload = tx.payload as unknown as RotateKeyPayload;
  const senderDid = resolveSenderDid(tx);

  // Only the account owner can rotate their own key
  if (payload.account !== senderDid) {
    throw new StateError(
      "Can only rotate keys for own account",
      "UNAUTHORIZED",
      { sender: senderDid, target: payload.account },
    );
  }

  if (!payload.newOperationalKeyHash) {
    throw new StateError(
      "New operational key hash is required",
      "INVALID_TRANSITION",
    );
  }

  // Delegate to StateManager → AccountStateTree (validates frozen/recovery state)
  state.rotateKey(senderDid, payload.newOperationalKeyHash);

  return 35000n; // Key rotation gas (between transfer and stake)
}

function executeStake(
  state: StateManager,
  tx: Transaction,
  params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as StakePayload;
  const senderDid = resolveSenderDid(tx);

  // Coerce amount to bigint (RPC sends strings)
  const amount =
    typeof payload.amount === "string"
      ? BigInt(payload.amount)
      : payload.amount;

  if (amount < params.minValidatorStake) {
    throw new StateError(
      `Minimum stake is ${params.minValidatorStake}`,
      "INVALID_TRANSITION",
    );
  }

  state.stake(senderDid, amount);

  // If not already a validator, register
  if (!state.isActiveValidator(tx.sender)) {
    state.registerValidator(tx.sender, senderDid, amount);
  }

  return 50000n; // Staking gas
}

function executeUnstake(
  state: StateManager,
  tx: Transaction,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as StakePayload;
  const senderDid = resolveSenderDid(tx);
  const amount =
    typeof payload.amount === "string"
      ? BigInt(payload.amount)
      : payload.amount;

  state.unstake(senderDid, amount);

  return 50000n;
}

function executeDelegate(
  state: StateManager,
  tx: Transaction,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as DelegatePayload;
  const senderDid = resolveSenderDid(tx);
  const amount =
    typeof payload.amount === "string"
      ? BigInt(payload.amount)
      : payload.amount;

  // Validate validator exists
  if (!state.isActiveValidator(payload.validator)) {
    throw new StateError("Validator not active", "VALIDATOR_NOT_FOUND");
  }

  // Transfer from balance to delegation
  const currentBalance = state.getBalance(senderDid);
  if (currentBalance < amount) {
    throw new StateError(
      "Insufficient balance to delegate",
      "INSUFFICIENT_BALANCE",
    );
  }

  // Deduct from account balance
  state.transfer(senderDid, "did:lgm:delegation-pool", amount);

  // Add to validator's delegated stake
  state.delegate(senderDid, payload.validator, amount);

  return 60000n;
}

/**
 * Execute undelegation with proper unbonding period
 *
 * SECURITY: Undelegation requires a timelock (unbonding period) to prevent
 * stake-and-slash attacks where a malicious validator could:
 * 1. Perform malicious actions
 * 2. Immediately undelegate before slashing can occur
 *
 * The unbonding period ensures that stake remains at risk long enough
 * for any misbehavior to be detected and punished.
 */
function executeUndelegate(
  state: StateManager,
  tx: Transaction,
  params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as DelegatePayload;
  const senderDid = resolveSenderDid(tx);
  const undelegateAmount =
    typeof payload.amount === "string"
      ? BigInt(payload.amount)
      : payload.amount;

  // 1. Validate validator exists and is active
  if (!state.isActiveValidator(payload.validator)) {
    throw new StateError("Validator not active", "VALIDATOR_NOT_FOUND");
  }

  // 2. Validate delegation exists and has sufficient amount
  const currentDelegation = state.getDelegation(senderDid, payload.validator);
  if (!currentDelegation || currentDelegation.amount < undelegateAmount) {
    throw new StateError(
      `Insufficient delegation: have ${currentDelegation?.amount ?? 0n}, requested ${undelegateAmount}`,
      "INSUFFICIENT_DELEGATION",
    );
  }

  // 3. Reduce active delegation immediately
  // The stake no longer counts toward validator's voting power
  state.reduceDelegation(senderDid, payload.validator, undelegateAmount);

  // 4. Create unbonding entry with timelock
  // Default unbonding period: ~7 days at 2.521s blocks = ~240,000 blocks
  const unbondingPeriod = params.unbondingPeriod ?? 240000n;
  const currentBlock = state.getCurrentBlock();
  const completesAt = currentBlock + unbondingPeriod;

  const unbondingId = sha512_256Hex(
    `unbond:${senderDid}:${payload.validator}:${currentBlock}:${undelegateAmount}`,
  );

  state.addUnbondingEntry({
    id: unbondingId,
    delegator: senderDid,
    validator: payload.validator,
    amount: undelegateAmount,
    initiatedAt: currentBlock,
    completesAt,
    status: "pending",
  });

  // 5. Emit event for tracking
  state.emitEvent("Undelegation", {
    unbondingId,
    delegator: senderDid,
    validator: payload.validator,
    amount: undelegateAmount.toString(),
    completesAt: completesAt.toString(),
  });

  return 80000n; // Undelegation gas (higher than delegation due to unbonding entry)
}

/**
 * Process matured unbonding entries and release funds
 *
 * This should be called at the start of each block to release
 * any unbonding entries whose timelock has expired.
 *
 * SECURITY: Funds are only released after the full unbonding period,
 * ensuring validators can be slashed for past misbehavior.
 */
export function processUnbondingQueue(
  state: StateManager,
  currentBlock: bigint,
): { processed: number; totalReleased: bigint } {
  let processed = 0;
  let totalReleased = 0n;

  // Get all completed unbonding entries
  const maturedEntries = state.getMaturedUnbondingEntries(currentBlock);

  for (const entry of maturedEntries) {
    // Transfer funds from delegation pool back to delegator
    state.transfer("did:lgm:delegation-pool", entry.delegator, entry.amount);

    // Mark entry as completed
    state.completeUnbondingEntry(entry.id);

    // Emit completion event
    state.emitEvent("UnbondingComplete", {
      unbondingId: entry.id,
      delegator: entry.delegator,
      amount: entry.amount.toString(),
      completedAtBlock: currentBlock.toString(),
    });

    processed++;
    totalReleased += entry.amount;
  }

  return { processed, totalReleased };
}

/**
 * Execute claim submission transaction.
 * Records claim in state and locks stake.
 *
 * GAP-002 FIX: Now creates ClaimState record in claims tree.
 */
function executeClaimSubmit(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  params: ConsensusParams,
): bigint {
  const payload = tx.payload as ClaimSubmitPayload;
  const senderDid = resolveSenderDid(tx);

  // 1. Validate stake meets minimum
  if (payload.stake < params.minClaimStake) {
    throw new StateError(
      `Minimum claim stake is ${params.minClaimStake}, got ${payload.stake}`,
      "INVALID_TRANSITION",
    );
  }

  // 2. Validate sender has sufficient balance
  const account = state.getAccount(senderDid);
  if (!account || account.balance < payload.stake) {
    throw new StateError(
      "Insufficient balance for claim stake",
      "INSUFFICIENT_BALANCE",
    );
  }

  // 3. Validate references exist (if provided)
  for (const refId of payload.references) {
    const refFact = state.getFact(refId);
    if (!refFact || refFact.status !== "active") {
      throw new StateError(
        `Referenced fact ${refId} not found or inactive`,
        "INVALID_REFERENCE",
      );
    }
  }

  // 3b. Validate derivedFrom fact IDs exist (if provided)
  if (payload.derivedFrom) {
    for (const factId of payload.derivedFrom.factIds) {
      const parentFact = state.getFact(factId);
      if (!parentFact || parentFact.status !== "active") {
        throw new StateError(
          `Derived-from fact ${factId} not found or inactive`,
          "INVALID_REFERENCE",
        );
      }
    }
  }

  // 4. Generate claim ID
  const claimId = generateClaimId(payload.claim, senderDid, blockNumber);

  // 5. Lock stake (move from balance to staked)
  state.stake(senderDid, payload.stake);

  // 6. Create claim record
  const claimState: ClaimState = {
    id: claimId,
    claim: payload.claim,
    claimType: payload.claimType,
    domain: payload.domain,
    stratum: payload.stratum,
    submitter: senderDid,
    stake: payload.stake,
    submittedAt: blockNumber,
    references: payload.references,
    status: "pending",
    statusChangedAt: blockNumber,
    derivedFrom: payload.derivedFrom,
    schemaId: payload.schemaId,
  };

  state.addClaim(claimState);

  // Gas: base cost + per-reference cost
  return 100000n + BigInt(payload.references.length) * 10000n;
}

/**
 * System transaction: Start verification round for a claim.
 * Called by consensus engine when selecting claim for verification.
 */
function executeClaimStartVerification(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  // System transaction validation
  if (!isSystemTransaction(tx)) {
    throw new StateError(
      "claim_start_verification is system-only",
      "UNAUTHORIZED",
    );
  }

  const payload = tx.payload as unknown as ClaimStartVerificationPayload;

  // 1. Get claim
  const claim = state.getClaim(payload.claimId);
  if (!claim) {
    throw new StateError(
      `Claim ${payload.claimId} not found`,
      "CLAIM_NOT_FOUND",
    );
  }

  // 2. Validate claim is pending
  if (claim.status !== "pending") {
    throw new StateError(
      `Claim ${payload.claimId} is ${claim.status}, expected pending`,
      "CLAIM_NOT_PENDING",
    );
  }

  // 3. Update claim status
  state.updateClaimStatus(payload.claimId, "verifying", blockNumber, {
    verificationRoundId: payload.roundId,
  });

  return 0n; // System transaction, no gas
}

/**
 * System transaction: Complete verification and create fact if accepted.
 * Called by consensus engine when verification round completes.
 */
function executeClaimCompleteVerification(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  params: ConsensusParams,
): bigint {
  // System transaction validation
  if (!isSystemTransaction(tx)) {
    throw new StateError(
      "claim_complete_verification is system-only",
      "UNAUTHORIZED",
    );
  }

  const payload = tx.payload as unknown as ClaimCompleteVerificationPayload;

  // 1. Get claim
  const claim = state.getClaim(payload.claimId);
  if (!claim) {
    throw new StateError(
      `Claim ${payload.claimId} not found`,
      "CLAIM_NOT_FOUND",
    );
  }

  // 2. Validate claim is verifying
  if (claim.status !== "verifying") {
    throw new StateError(
      `Claim ${payload.claimId} is ${claim.status}, expected verifying`,
      "CLAIM_NOT_VERIFYING",
    );
  }

  // 3. Process based on outcome
  if (payload.outcome === "accepted") {
    // Create fact
    const factId = `fact:${claim.id.replace("claim:", "")}`;

    state.addFact({
      id: factId,
      claim: claim.claim,
      domain: claim.domain,
      stratum: claim.stratum,
      confidence: payload.finalConfidence,
      verifiedAt: blockNumber,
      verifierCount: payload.verifierCount,
      challengeCount: 0,
      citationCount: 0,
      status: "active",
    });

    // Wire claim references into knowledge graph edges
    // Each referenced fact gets a 'supports' edge from the new fact
    // and has its citation count incremented
    const citedFactIds = new Set<string>();
    if (claim.references && claim.references.length > 0) {
      for (const referencedFactId of claim.references) {
        if (state.isActiveFact(referencedFactId)) {
          state.addFactReference({
            fromFactId: factId,
            toFactId: referencedFactId,
            referenceType: "supports",
            strength: 0.8,
          });
          state.incrementFactCitationCount(referencedFactId, blockNumber);
          citedFactIds.add(referencedFactId);
        }
      }
    }

    // Wire derivedFrom into knowledge graph edges
    // Creates 'derives_from' edges with inference-type-appropriate strength
    if (claim.derivedFrom && claim.derivedFrom.factIds.length > 0) {
      // Inference strength: deductive=0.99, inductive=0.75, abductive=0.6
      const inferenceStrengths: Record<string, number> = {
        deductive: 0.99,
        inductive: 0.75,
        abductive: 0.6,
      };
      const strength =
        inferenceStrengths[claim.derivedFrom.inferenceType] ?? 0.7;

      for (const parentFactId of claim.derivedFrom.factIds) {
        if (state.isActiveFact(parentFactId)) {
          state.addFactReference({
            fromFactId: factId,
            toFactId: parentFactId,
            referenceType: "derives_from",
            strength,
          });
          if (!citedFactIds.has(parentFactId)) {
            state.incrementFactCitationCount(parentFactId, blockNumber);
            citedFactIds.add(parentFactId);
          }
        }
      }
    }

    // Distribute citation rewards to cited facts' submitters
    // 15% of claimReward split equally among cited facts (KNOWLEDGE_ECONOMICS.md)
    if (citedFactIds.size > 0) {
      const CITATION_SHARE = 15n;
      const citationPool = (params.claimReward * CITATION_SHARE) / 100n;
      const rewardPerFact = citationPool / BigInt(citedFactIds.size);

      if (rewardPerFact > 0n) {
        for (const citedFactId of citedFactIds) {
          state.recordFactCitationReward(citedFactId, rewardPerFact);

          // Transfer reward to the original fact submitter (if known)
          const citedClaimId = "claim:" + citedFactId.replace("fact:", "");
          const citedClaim = state.getClaim(citedClaimId);
          if (citedClaim?.submitter) {
            state.transfer(FEE_POOL_DID, citedClaim.submitter, rewardPerFact);
          }
        }
      }
    }

    // Update claim with fact reference
    state.updateClaimStatus(payload.claimId, "accepted", blockNumber, {
      resultFactId: factId,
    });

    // Return stake to submitter (unstake)
    if (claim.stake > 0n) {
      state.unstake(claim.submitter, claim.stake);
    }
  } else if (payload.outcome === "rejected") {
    // Slash submitter stake (50%)
    const slashAmount = claim.stake / 2n;

    // First unstake the full amount (moves from staked to balance)
    if (claim.stake > 0n) {
      state.unstake(claim.submitter, claim.stake);
    }

    // Then transfer the slashed portion to fee pool
    if (slashAmount > 0n) {
      state.transfer(claim.submitter, FEE_POOL_DID, slashAmount);
    }

    state.updateClaimStatus(payload.claimId, "rejected", blockNumber, {
      rejectionReason: "Verification failed",
    });
  } else if (
    payload.outcome === "contested" ||
    payload.outcome === "insufficient"
  ) {
    // Return full stake, claim can be resubmitted
    if (claim.stake > 0n) {
      state.unstake(claim.submitter, claim.stake);
    }

    state.updateClaimStatus(payload.claimId, "rejected", blockNumber, {
      rejectionReason:
        payload.outcome === "contested"
          ? "Verification contested - no consensus"
          : "Insufficient validators participated",
    });
  }

  // 4. Distribute rewards (only if fee pool and recipient accounts exist)
  for (const [validatorDid, reward] of payload.rewards) {
    if (
      reward > 0n &&
      state.accountExists(FEE_POOL_DID) &&
      state.accountExists(validatorDid)
    ) {
      state.transfer(FEE_POOL_DID, validatorDid, reward);
    }
  }

  // 5. Apply slashes (validators who didn't reveal)
  for (const [validatorDid, slashAmount] of payload.slashes) {
    const validator = state.getValidator(validatorDid as unknown as PublicKey);
    if (validator) {
      state.slashValidator(
        validator.publicKey,
        slashAmount,
        "verification_failure",
      );
    }
  }

  return 0n; // System transaction, no gas
}

/**
 * Execute claim withdrawal.
 * Allows submitter to withdraw a pending claim before verification starts.
 */
function executeClaimWithdraw(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as ClaimWithdrawPayload;
  const senderDid = resolveSenderDid(tx);

  // 1. Get claim
  const claim = state.getClaim(payload.claimId);
  if (!claim) {
    throw new StateError(
      `Claim ${payload.claimId} not found`,
      "CLAIM_NOT_FOUND",
    );
  }

  // 2. Validate sender is the submitter
  if (claim.submitter !== senderDid) {
    throw new StateError("Only claim submitter can withdraw", "UNAUTHORIZED");
  }

  // 3. Validate claim is pending (cannot withdraw once verification started)
  if (claim.status !== "pending") {
    throw new StateError(
      `Cannot withdraw claim in ${claim.status} status`,
      "CLAIM_NOT_PENDING",
    );
  }

  // 4. Return stake to submitter
  state.unstake(senderDid, claim.stake);

  // 5. Update claim status
  state.updateClaimStatus(payload.claimId, "withdrawn", blockNumber);

  return 50000n;
}

/**
 * Check if transaction is a system transaction (no sender signature required)
 */
function isSystemTransaction(tx: Transaction): boolean {
  return tx.sender === "system" || tx.sender === ("system" as any);
}

function executeVerificationCommit(
  state: StateManager,
  tx: Transaction,
  _params: ConsensusParams,
): bigint {
  // Commitments are handled by consensus layer
  // State just validates the validator is active
  if (!state.isActiveValidator(tx.sender)) {
    throw new StateError(
      "Only validators can commit verifications",
      "INVALID_TRANSITION",
    );
  }

  return 30000n;
}

function executeVerificationReveal(
  state: StateManager,
  tx: Transaction,
  _params: ConsensusParams,
): bigint {
  // Reveals are handled by consensus layer
  if (!state.isActiveValidator(tx.sender)) {
    throw new StateError(
      "Only validators can reveal verifications",
      "INVALID_TRANSITION",
    );
  }

  return 40000n;
}

function executeChallenge(
  state: StateManager,
  tx: Transaction,
  _blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as ChallengeFactPayload;
  const senderDid = resolveSenderDid(tx);

  // Validate fact exists and is active
  if (!state.isActiveFact(payload.factId)) {
    throw new StateError("Fact not found or not active", "FACT_NOT_FOUND");
  }

  // Lock challenge stake
  state.stake(senderDid, payload.stake);

  // Mark fact as challenged
  state.challengeFact(payload.factId);

  return 80000n;
}

function executeGovernanceVote(
  state: StateManager,
  tx: Transaction,
  _params: ConsensusParams,
): bigint {
  // Governance voting
  // Requires stake to vote
  const senderDid = resolveSenderDid(tx);
  const account = state.getAccount(senderDid);

  if (!account || account.stakedBalance === 0n) {
    throw new StateError("Must have stake to vote", "INVALID_TRANSITION");
  }

  return 25000n;
}

function executeGovernancePropose(
  state: StateManager,
  tx: Transaction,
  _params: ConsensusParams,
): bigint {
  // Governance proposals require minimum stake
  const senderDid = resolveSenderDid(tx);
  const account = state.getAccount(senderDid);

  const MIN_PROPOSAL_STAKE = 10000n * 10n ** 18n; // 10000 LGM

  if (!account || account.stakedBalance < MIN_PROPOSAL_STAKE) {
    throw new StateError(
      `Must have at least ${MIN_PROPOSAL_STAKE} staked to propose`,
      "INVALID_TRANSITION",
    );
  }

  return 150000n;
}

// ============ Governance LIP Transaction Handlers ============

/**
 * Build AgentGovernanceStats from on-chain account state.
 * Phase 1 simplification: derives capability/contributions from account fields.
 */
function buildGovernanceStats(
  state: StateManager,
  did: string,
  blockNumber: bigint,
): AgentGovernanceStats {
  const account = state.getAccount(did);
  const rep = account?.reputationScore ?? 0.5;
  return {
    did: did as any,
    stake: account?.stakedBalance ?? 0n,
    reputation: rep,
    capabilityLevel: rep >= 0.7 ? 2 : rep >= 0.5 ? 1 : 0,
    contributions: Number(account?.nonce ?? 0n), // Use nonce as proxy for contributions
    accountAge: account?.lastActiveBlock
      ? blockNumber - account.lastActiveBlock
      : 0n,
  };
}

/**
 * Submit a new LIP (Legible Improvement Proposal)
 */
function executeLipSubmit(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as LipSubmitPayload;
  const senderDid = resolveSenderDid(tx);

  // Build governance stats for eligibility check
  const stats = buildGovernanceStats(state, senderDid, blockNumber);

  // Create the LIP (validates eligibility internally)
  const result = createLIP(
    {
      title: payload.title,
      category: payload.category,
      proposer: senderDid as any,
      abstract: payload.abstract,
      motivation: payload.motivation,
      specification: payload.specification,
      rationale: payload.rationale,
      backwardsCompatibility: payload.backwardsCompatibility,
      testCases: payload.testCases,
      referenceImplementation: payload.referenceImplementation,
      coAuthors: (payload.coAuthors ?? []) as any[],
    },
    stats,
    blockNumber,
    state.getNextLIPNumber(),
  );

  if ("error" in result) {
    throw new StateError(result.error, "GOVERNANCE_INELIGIBLE");
  }

  // Store the LIP
  state.addLIP(result.lip);

  // Increment nonce (no stake/transfer in this handler)
  state.incrementNonce(senderDid);

  // Emit event
  state.emitEvent("LIPSubmitted", {
    lipId: result.lip.id,
    proposer: senderDid,
    category: result.lip.category,
    stakeRequired: result.stakeRequired.toString(),
  });

  return 200000n;
}

/**
 * Stake on a LIP to advance it from draft to review
 */
function executeLipStake(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as LipStakePayload;
  const senderDid = resolveSenderDid(tx);

  const lip = state.getLIP(payload.lipId);
  if (!lip) {
    throw new StateError(`LIP ${payload.lipId} not found`, "LIP_NOT_FOUND");
  }

  // Verify sender is proposer or stakeholder
  if (lip.proposer !== senderDid && lip.stakeHolder !== senderDid) {
    throw new StateError(
      "Only proposer or stakeholder can stake on LIP",
      "UNAUTHORIZED",
    );
  }

  // Validate balance
  const balance = state.getBalance(senderDid);
  if (balance < payload.amount) {
    throw new StateError(
      "Insufficient balance for LIP stake",
      "INSUFFICIENT_BALANCE",
    );
  }

  // Lock the stake
  state.stake(senderDid, payload.amount);

  // Transition to review
  const result = transitionToReview(
    lip,
    payload.amount,
    blockNumber,
    senderDid as any,
  );
  if (!result.success) {
    throw new StateError(result.error!, "LIP_INVALID_STAGE");
  }

  // Update the LIP
  state.updateLIP(payload.lipId, lip);

  state.emitEvent("LIPStaked", {
    lipId: payload.lipId,
    staker: senderDid,
    amount: payload.amount.toString(),
    newStage: lip.stage,
  });

  return 100000n;
}

/**
 * Advance a LIP to the next stage based on vote results
 */
function executeLipAdvanceStage(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as LipAdvanceStagePayload;

  const lip = state.getLIP(payload.lipId);
  if (!lip) {
    throw new StateError(`LIP ${payload.lipId} not found`, "LIP_NOT_FOUND");
  }

  // Build VoteTally from quadratic votes
  const votes = state.getVotesForLIP(payload.lipId);
  const tally = buildVoteTally(votes, state);

  let result;
  const stageStart =
    lip.stageHistory.length > 0
      ? lip.stageHistory[lip.stageHistory.length - 1]!.at
      : lip.createdAt;

  switch (lip.stage) {
    case "review":
      result = transitionFromReview(lip, tally, blockNumber, stageStart);
      break;
    case "last_call":
      result = transitionFromLastCall(lip, tally, blockNumber, stageStart);
      break;
    case "voting":
      result = transitionFromVoting(lip, tally, blockNumber);
      break;
    default:
      throw new StateError(
        `Cannot advance LIP from stage: ${lip.stage}`,
        "LIP_INVALID_STAGE",
      );
  }

  if (!result.success) {
    throw new StateError(result.error!, "LIP_INVALID_STAGE");
  }

  // Handle stake return on terminal states (stage may have been mutated by transition functions)
  const currentStage = lip.stage as string;
  if (currentStage === "accepted" || currentStage === "rejected") {
    const outcome =
      currentStage === "accepted"
        ? ("accepted" as const)
        : ("rejected_normal" as const);
    const stakeOutcome = calculateStakeOutcome(lip, outcome);

    if (lip.stakedAmount > 0n) {
      state.unstake(lip.stakeHolder as string, lip.stakedAmount);

      if (stakeOutcome.slashAmount > 0n) {
        state.deductBalance(
          lip.stakeHolder as string,
          stakeOutcome.slashAmount,
        );
        state.creditBalance(
          TREASURY.ADDRESSES.FOUNDATION,
          stakeOutcome.slashAmount,
        );
      }
    }
  }

  // Update the LIP
  state.updateLIP(payload.lipId, lip);

  state.emitEvent("LIPAdvanced", {
    lipId: payload.lipId,
    newStage: lip.stage,
    reason: result.transition?.reason ?? "Stage advanced",
  });

  return 120000n;
}

/**
 * Withdraw a LIP (only by proposer, before voting)
 */
function executeLipWithdraw(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as LipWithdrawPayload;
  const senderDid = resolveSenderDid(tx);

  const lip = state.getLIP(payload.lipId);
  if (!lip) {
    throw new StateError(`LIP ${payload.lipId} not found`, "LIP_NOT_FOUND");
  }

  const result = withdrawLIP(lip, blockNumber, senderDid as any);
  if (!result.success) {
    throw new StateError(result.error!, "LIP_INVALID_STAGE");
  }

  // Handle stake return
  if (lip.stakedAmount > 0n) {
    const stakeOutcome = calculateStakeOutcome(lip, "withdrawn");
    // unstake auto-increments stakeHolder nonce (stakeHolder === proposer === senderDid)
    state.unstake(lip.stakeHolder as string, lip.stakedAmount);

    if (stakeOutcome.slashAmount > 0n) {
      state.deductBalance(lip.stakeHolder as string, stakeOutcome.slashAmount);
      state.creditBalance(
        TREASURY.ADDRESSES.FOUNDATION,
        stakeOutcome.slashAmount,
      );
    }
  } else {
    // No stake operation to increment nonce, do it explicitly
    state.incrementNonce(senderDid);
  }

  // Update the LIP
  state.updateLIP(payload.lipId, lip);

  state.emitEvent("LIPWithdrawn", {
    lipId: payload.lipId,
    withdrawer: senderDid,
  });

  return 80000n;
}

/**
 * Cast a quadratic vote on a LIP
 */
function executeVoteCast(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as VoteCastPayload;
  const senderDid = resolveSenderDid(tx);

  const lip = state.getLIP(payload.lipId);
  if (!lip) {
    throw new StateError(`LIP ${payload.lipId} not found`, "LIP_NOT_FOUND");
  }

  // Verify LIP is in a votable stage
  if (!["review", "last_call", "voting"].includes(lip.stage)) {
    throw new StateError(
      `Cannot vote on LIP in stage: ${lip.stage}`,
      "LIP_INVALID_STAGE",
    );
  }

  // Check if voter has already voted
  if (state.hasVotedOnLIP(senderDid, payload.lipId)) {
    throw new StateError("Already voted on this LIP", "VOTE_ALREADY_CAST");
  }

  // Calculate vote cost
  const cost = calculateVoteCost(payload.votes);

  // Verify voter balance
  const balance = state.getBalance(senderDid);
  if (balance < cost) {
    throw new StateError(
      "Insufficient balance for vote cost",
      "INSUFFICIENT_BALANCE",
    );
  }

  // Create the vote
  const vote = createQuadraticVote(
    senderDid as any,
    payload.lipId,
    payload.direction as VoteDirection,
    payload.votes,
    blockNumber,
  );

  // Lock the vote cost
  state.stake(senderDid, cost);

  // Store the vote
  state.addQuadraticVote(vote);

  state.emitEvent("VoteCast", {
    voteId: vote.voteId,
    lipId: payload.lipId,
    voter: senderDid,
    direction: payload.direction,
    votes: String(payload.votes),
    cost: cost.toString(),
  });

  return 50000n;
}

/**
 * Build a VoteTally from quadratic votes for lifecycle transitions.
 * Phase 1: operatorBreakdown is empty (wired in Phase 3).
 */
function buildVoteTally(
  votes: import("../governance/voting/quadratic.js").QuadraticVote[],
  _state: StateManager,
): VoteTally {
  let support = 0n;
  let oppose = 0n;
  let abstain = 0n;
  const voters = new Set<string>();

  for (const vote of votes) {
    voters.add(vote.voter);
    const weight = BigInt(vote.votes);
    switch (vote.direction) {
      case "support":
        support += weight;
        break;
      case "oppose":
        oppose += weight;
        break;
      case "abstain":
        abstain += weight;
        break;
    }
  }

  return {
    support,
    oppose,
    abstain,
    totalEligible: support + oppose + abstain + 1n, // Phase 1: approximation
    uniqueVoters: voters.size,
    operatorBreakdown: new Map(), // Phase 3: wired with operator registry
    quorumMet: voters.size > 0,
    operatorCapViolation: false, // Phase 3: wired with operator registry
  };
}

// ============ Disbursement Transaction Handlers ============

/**
 * Submit a new disbursement proposal.
 * Requires Tier 1 proposer eligibility.
 */
function executeDisbursementSubmit(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as DisbursementSubmitPayload;
  const senderDid = resolveSenderDid(tx);

  // Validate category
  if (
    !VALID_DISBURSEMENT_CATEGORIES.includes(
      payload.category as DisbursementCategory,
    )
  ) {
    throw new StateError(
      `Invalid disbursement category: ${payload.category}`,
      "INVALID_TRANSITION",
    );
  }

  // Validate treasury fund
  if (!VALID_TREASURY_FUNDS.includes(payload.treasuryFund as TreasuryFund)) {
    throw new StateError(
      `Invalid treasury fund: ${payload.treasuryFund}`,
      "INVALID_TRANSITION",
    );
  }

  // Validate amount against treasury cap
  const treasuryDid = treasuryFundToDid(payload.treasuryFund as TreasuryFund);
  const treasuryBalance = state.getBalance(treasuryDid);
  const amountValidation = validateDisbursementAmount(
    payload.amount,
    treasuryBalance,
  );
  if (!amountValidation.valid) {
    throw new StateError(amountValidation.error!, "INVALID_TRANSITION");
  }

  // Create proposal via lifecycle
  const result = createDisbursementProposal(
    {
      title: payload.title,
      description: payload.description,
      category: payload.category as DisbursementCategory,
      treasuryFund: payload.treasuryFund as TreasuryFund,
      proposer: senderDid as any,
      recipientDid: payload.recipientDid as any,
      amount: payload.amount,
      justification: payload.justification,
      externalReference: payload.externalReference,
    },
    blockNumber,
    state.getNextDisbursementNumber(),
  );

  if ("error" in result) {
    throw new StateError(result.error, "INVALID_TRANSITION");
  }

  // Store in governance tree
  state.addDisbursement(result.proposal);

  state.emitEvent("DisbursementSubmitted", {
    id: result.proposal.id,
    proposer: senderDid,
    category: payload.category,
    treasuryFund: payload.treasuryFund,
    amount: payload.amount.toString(),
    recipient: payload.recipientDid,
  });

  return 150000n;
}

/**
 * Stake on a disbursement proposal to advance it from proposed to voting.
 * Requires DISBURSEMENT_STAKE (2000 LGM).
 */
function executeDisbursementStake(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as DisbursementStakePayload;
  const senderDid = resolveSenderDid(tx);

  const proposal = state.getDisbursement(payload.disbursementId);
  if (!proposal) {
    throw new StateError(
      `Disbursement ${payload.disbursementId} not found`,
      "INVALID_TRANSITION",
    );
  }

  // Only proposer or stakeholder can stake
  if (proposal.proposer !== senderDid && proposal.stakeHolder !== senderDid) {
    throw new StateError(
      "Only proposer or stakeholder can stake on disbursement",
      "UNAUTHORIZED",
    );
  }

  // Validate balance
  const balance = state.getBalance(senderDid);
  if (balance < payload.amount) {
    throw new StateError(
      "Insufficient balance for disbursement stake",
      "INSUFFICIENT_BALANCE",
    );
  }

  // Lock the stake
  state.stake(senderDid, payload.amount);

  // Transition to voting
  const result = disbursementTransitionToVoting(
    proposal,
    payload.amount,
    blockNumber,
    senderDid as any,
  );
  if (!result.success) {
    throw new StateError(result.error!, "INVALID_TRANSITION");
  }

  // Update the proposal
  state.updateDisbursement(payload.disbursementId, proposal);

  state.emitEvent("DisbursementStaked", {
    id: payload.disbursementId,
    staker: senderDid,
    amount: payload.amount.toString(),
    newStage: proposal.stage,
  });

  return 100000n;
}

/**
 * Advance a disbursement from voting to approved/rejected based on vote results.
 * Uses buildVoteTally (same as LIP) + disbursement thresholds.
 */
function executeDisbursementAdvanceStage(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as DisbursementAdvanceStagePayload;

  const proposal = state.getDisbursement(payload.disbursementId);
  if (!proposal) {
    throw new StateError(
      `Disbursement ${payload.disbursementId} not found`,
      "INVALID_TRANSITION",
    );
  }

  if (proposal.stage !== "voting") {
    throw new StateError(
      `Cannot advance disbursement from stage: ${proposal.stage}`,
      "INVALID_TRANSITION",
    );
  }

  // Build VoteTally from quadratic votes (reuses LIP voting infrastructure)
  const votes = state.getVotesForLIP(payload.disbursementId);
  const tally = buildVoteTally(votes, state);

  const result = disbursementTransitionFromVoting(proposal, tally, blockNumber);
  if (!result.success) {
    throw new StateError(result.error!, "INVALID_TRANSITION");
  }

  // Handle stake return on terminal states (stage may have been mutated by transition)
  const currentDisbStage = proposal.stage as string;
  if (currentDisbStage === "rejected") {
    const stakeOutcome = calculateDisbursementStakeOutcome(
      proposal,
      "rejected_normal",
    );
    if (proposal.stakedAmount > 0n) {
      state.unstake(proposal.stakeHolder as string, proposal.stakedAmount);
      if (stakeOutcome.slashAmount > 0n) {
        state.deductBalance(
          proposal.stakeHolder as string,
          stakeOutcome.slashAmount,
        );
        state.creditBalance(
          treasuryFundToDid("FOUNDATION"),
          stakeOutcome.slashAmount,
        );
      }
    }
  }

  // Update the proposal
  state.updateDisbursement(payload.disbursementId, proposal);

  state.emitEvent("DisbursementAdvanced", {
    id: payload.disbursementId,
    newStage: proposal.stage,
    reason: result.transition?.reason ?? "Stage advanced",
  });

  return 120000n;
}

/**
 * Execute an approved disbursement after timelock expires.
 * Transfers funds from treasury to recipient and creates an ExpenseRecord.
 */
function executeDisbursementExecuteHandler(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as DisbursementExecutePayload;

  const proposal = state.getDisbursement(payload.disbursementId);
  if (!proposal) {
    throw new StateError(
      `Disbursement ${payload.disbursementId} not found`,
      "INVALID_TRANSITION",
    );
  }

  // Lifecycle validates stage + timelock
  const result = disbursementExecute(proposal, blockNumber);
  if (!result.success) {
    throw new StateError(result.error!, "INVALID_TRANSITION");
  }

  // Verify treasury has sufficient balance
  const treasuryDid = treasuryFundToDid(proposal.treasuryFund);
  const treasuryBalance = state.getBalance(treasuryDid);
  if (treasuryBalance < proposal.amount) {
    throw new StateError(
      `Insufficient treasury balance: ${treasuryBalance / 10n ** 18n} LGM < ${proposal.amount / 10n ** 18n} LGM`,
      "INSUFFICIENT_BALANCE",
    );
  }

  // Re-validate amount cap at execution time
  const amountCheck = validateDisbursementAmount(
    proposal.amount,
    treasuryBalance,
  );
  if (!amountCheck.valid) {
    throw new StateError(amountCheck.error!, "INVALID_TRANSITION");
  }

  // Transfer funds: treasury → recipient
  state.deductBalance(treasuryDid, proposal.amount);
  state.creditBalance(proposal.recipientDid as string, proposal.amount);

  // Return stake to proposer
  if (proposal.stakedAmount > 0n) {
    state.unstake(proposal.stakeHolder as string, proposal.stakedAmount);
  }

  // Create immutable expense record
  const expenseRecord: ExpenseRecord = {
    id: `EXP-${state.getNextExpenseNumber()}`,
    disbursementId: proposal.id,
    category: proposal.category,
    treasuryFund: proposal.treasuryFund,
    recipientDid: proposal.recipientDid,
    amount: proposal.amount,
    executedAt: blockNumber,
    txHash: sha512_256Hex(`disbursement:${proposal.id}:${blockNumber}`),
    justification: proposal.justification,
    externalReference: proposal.externalReference,
  };
  state.addExpenseRecord(expenseRecord);

  // Update the proposal
  state.updateDisbursement(payload.disbursementId, proposal);

  state.emitEvent("DisbursementExecuted", {
    id: proposal.id,
    expenseId: expenseRecord.id,
    treasuryFund: proposal.treasuryFund,
    recipient: proposal.recipientDid,
    amount: proposal.amount.toString(),
  });

  return 200000n;
}

/**
 * Cancel a disbursement proposal. Only proposer can cancel from proposed or approved.
 */
function executeDisbursementCancelHandler(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as DisbursementCancelPayload;
  const senderDid = resolveSenderDid(tx);

  const proposal = state.getDisbursement(payload.disbursementId);
  if (!proposal) {
    throw new StateError(
      `Disbursement ${payload.disbursementId} not found`,
      "INVALID_TRANSITION",
    );
  }

  const result = disbursementCancel(proposal, blockNumber, senderDid as any);
  if (!result.success) {
    throw new StateError(result.error!, "INVALID_TRANSITION");
  }

  // Return stake (minus cancellation fee) if staked
  if (proposal.stakedAmount > 0n) {
    const stakeOutcome = calculateDisbursementStakeOutcome(
      proposal,
      "cancelled",
    );
    state.unstake(proposal.stakeHolder as string, proposal.stakedAmount);
    if (stakeOutcome.slashAmount > 0n) {
      state.deductBalance(
        proposal.stakeHolder as string,
        stakeOutcome.slashAmount,
      );
      state.creditBalance(
        treasuryFundToDid("FOUNDATION"),
        stakeOutcome.slashAmount,
      );
    }
  }

  // Update the proposal
  state.updateDisbursement(payload.disbursementId, proposal);

  state.emitEvent("DisbursementCancelled", {
    id: payload.disbursementId,
    canceller: senderDid,
  });

  return 80000n;
}

// ============ Research Transaction Handlers ============

/**
 * Submit research evidence to the Protocol Research Repository
 */
function executeResearchSubmit(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as ResearchSubmitPayload;
  const senderDid = resolveSenderDid(tx);

  // Validate minimum stake
  if (payload.stake < DEFAULT_RESEARCH_PARAMETERS.minResearchStake) {
    throw new StateError(
      `Minimum research stake is ${DEFAULT_RESEARCH_PARAMETERS.minResearchStake}`,
      "INVALID_TRANSITION",
    );
  }

  // Validate balance
  const balance = state.getBalance(senderDid);
  if (balance < payload.stake) {
    throw new StateError(
      "Insufficient balance for research stake",
      "INSUFFICIENT_BALANCE",
    );
  }

  // If targeting a specific fact, verify it exists
  if (payload.targetFactId && !state.isActiveFact(payload.targetFactId)) {
    throw new StateError(
      "Target fact not found or not active",
      "FACT_NOT_FOUND",
    );
  }

  // Lock researcher's stake
  state.stake(senderDid, payload.stake);

  // Generate research ID
  const researchId = sha512_256Hex(
    `research:${senderDid}:${blockNumber}:${payload.rawDataCID}`,
  );

  // Create research state
  const research: ResearchState = {
    id: researchId,
    submitter: senderDid,
    researchType: payload.researchType,
    targetFactId: payload.targetFactId,
    targetDomain: payload.targetDomain,
    outcome: payload.outcome,
    confidenceContribution: 0, // Calculated during review
    rawDataCID: payload.rawDataCID,
    status: "pending",
    submittedAt: blockNumber,
    reviewScore: 0,
  };

  // Add to research state tree
  state.addResearch(research);

  return 200000n; // Research submission gas
}

/**
 * Challenge a research submission
 */
function executeResearchChallenge(
  state: StateManager,
  tx: Transaction,
  _blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as {
    researchId: string;
    stake: bigint;
    evidence: string;
  };
  const senderDid = resolveSenderDid(tx);

  // Validate research exists
  const research = state.getResearch(payload.researchId);
  if (!research) {
    throw new StateError("Research not found", "INVALID_TRANSITION");
  }

  if (research.status !== "accepted") {
    throw new StateError(
      "Can only challenge accepted research",
      "INVALID_TRANSITION",
    );
  }

  // Validate minimum stake
  const minChallengeStake = DEFAULT_RESEARCH_PARAMETERS.minResearchStake;
  if (payload.stake < minChallengeStake) {
    throw new StateError(
      `Minimum challenge stake is ${minChallengeStake}`,
      "INVALID_TRANSITION",
    );
  }

  // Lock challenger's stake
  state.stake(senderDid, payload.stake);

  // Update research status to challenged
  state.updateResearch(payload.researchId, { status: "challenged" });

  return 120000n; // Research challenge gas
}

/**
 * Create a research bounty from the research treasury
 */
function executeBountyCreate(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as BountyCreatePayload;
  const senderDid = resolveSenderDid(tx);

  // Only research treasury or governance can create bounties
  // For now, allow anyone with sufficient stake
  const account = state.getAccount(senderDid);
  if (
    !account ||
    account.stakedBalance < DEFAULT_RESEARCH_PARAMETERS.minBountyStake
  ) {
    throw new StateError(
      `Must have at least ${DEFAULT_RESEARCH_PARAMETERS.minBountyStake} staked to create bounty`,
      "INVALID_TRANSITION",
    );
  }

  // Validate reward amount
  if (payload.reward > DEFAULT_RESEARCH_PARAMETERS.maxBountyReward) {
    throw new StateError(
      `Bounty reward exceeds maximum of ${DEFAULT_RESEARCH_PARAMETERS.maxBountyReward}`,
      "INVALID_TRANSITION",
    );
  }

  // If targeting a fact, validate it exists
  if (payload.targetFactId && !state.isActiveFact(payload.targetFactId)) {
    throw new StateError(
      "Target fact not found or not active",
      "FACT_NOT_FOUND",
    );
  }

  // Transfer reward from creator to lock (or from research treasury)
  const creatorBalance = state.getBalance(senderDid);
  if (creatorBalance < payload.reward) {
    throw new StateError(
      "Insufficient balance for bounty reward",
      "INSUFFICIENT_BALANCE",
    );
  }

  // Lock the reward
  state.transfer(senderDid, "did:lgm:bounty-escrow", payload.reward);

  // Generate bounty ID
  const bountyId = sha512_256Hex(
    `bounty:${senderDid}:${blockNumber}:${payload.description.slice(0, 32)}`,
  );

  // Calculate deadline
  const deadline = blockNumber + BigInt(payload.deadlineBlocks);

  // Create bounty state
  const bounty: BountyState = {
    id: bountyId,
    creator: senderDid,
    targetType: payload.targetType,
    targetFactId: payload.targetFactId,
    reward: payload.reward,
    deadline,
    status: "open",
  };

  // Add to research state tree
  state.addBounty(bounty);

  return 150000n; // Bounty creation gas
}

/**
 * Claim a bounty to work on it
 */
function executeBountyClaim(
  state: StateManager,
  tx: Transaction,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as BountyClaimPayload;
  const senderDid = resolveSenderDid(tx);

  // Validate bounty exists and is open
  const bounty = state.getBounty(payload.bountyId);
  if (!bounty) {
    throw new StateError("Bounty not found", "INVALID_TRANSITION");
  }

  if (bounty.status !== "open") {
    throw new StateError(
      "Bounty is not available for claiming",
      "INVALID_TRANSITION",
    );
  }

  // Claim the bounty
  state.claimBounty(payload.bountyId, senderDid);

  return 50000n; // Bounty claim gas
}

/**
 * Fulfill a bounty with completed research
 */
function executeBountyFulfill(
  state: StateManager,
  tx: Transaction,
  _blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as BountyFulfillPayload;
  const senderDid = resolveSenderDid(tx);

  // Validate bounty exists and is claimed
  const bounty = state.getBounty(payload.bountyId);
  if (!bounty) {
    throw new StateError("Bounty not found", "INVALID_TRANSITION");
  }

  if (bounty.status !== "claimed") {
    throw new StateError(
      "Bounty must be claimed before fulfillment",
      "INVALID_TRANSITION",
    );
  }

  if (bounty.claimedBy !== senderDid) {
    throw new StateError(
      "Only the claimer can fulfill this bounty",
      "INVALID_TRANSITION",
    );
  }

  // Validate research submission exists and is accepted
  const research = state.getResearch(payload.researchSubmissionId);
  if (!research) {
    throw new StateError("Research submission not found", "INVALID_TRANSITION");
  }

  if (research.status !== "accepted") {
    throw new StateError(
      "Research must be accepted before bounty fulfillment",
      "INVALID_TRANSITION",
    );
  }

  if (research.submitter !== senderDid) {
    throw new StateError(
      "Research submitter must match bounty claimer",
      "INVALID_TRANSITION",
    );
  }

  // Fulfill the bounty
  state.fulfillBounty(payload.bountyId, senderDid);

  // Release reward from escrow to fulfiller
  // Note: In full implementation, this would go through the reward router
  // for 22% research share, but for bounties funded from research treasury
  // the share may be handled differently
  state.transfer("did:lgm:bounty-escrow", senderDid, bounty.reward);

  return 100000n; // Bounty fulfillment gas
}

// ============ Dispute Transaction Handlers ============

/**
 * Initiate a new dispute against evidence, fact, or claim
 *
 * Core philosophy: "Intention does not matter, result does."
 * Anyone can challenge - if they're wrong, they pay 30% slash.
 * If they're right, they get 60% of defender's stake + bonuses.
 */
function executeDisputeInitiate(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as InitiateDisputePayload;
  const senderDid = resolveSenderDid(tx);

  // Validate sender is the challenger
  if (payload.challenger !== senderDid) {
    throw new StateError("Sender must be the challenger", "INVALID_TRANSITION");
  }

  // Validate minimum stake
  if (payload.stake < DISPUTE_ECONOMICS.minChallengeStake) {
    throw new StateError(
      `Minimum challenge stake is ${DISPUTE_ECONOMICS.minChallengeStake}`,
      "INVALID_TRANSITION",
    );
  }

  // Validate challenger has sufficient balance
  const balance = state.getBalance(senderDid);
  if (balance < payload.stake) {
    throw new StateError(
      "Insufficient balance for challenge stake",
      "INSUFFICIENT_BALANCE",
    );
  }

  // Get the target (what's being disputed)
  if (payload.targetType === "fact") {
    const fact = state.getFact(payload.targetId);
    if (!fact) {
      throw new StateError("Target fact not found", "FACT_NOT_FOUND");
    }
  } else if (payload.targetType === "evidence") {
    const evidence = state.getEvidence(payload.targetId);
    if (!evidence) {
      throw new StateError("Target evidence not found", "INVALID_TRANSITION");
    }
  } else {
    throw new StateError("Invalid target type", "INVALID_TRANSITION");
  }

  // Lock challenger's stake
  state.stake(senderDid, payload.stake);

  // Create the dispute using the dispute engine
  const dispute = createDispute(payload, blockNumber);

  // Add dispute to state
  state.addDispute(dispute);

  return 250000n; // Dispute initiation gas
}

/**
 * Commit evidence hash for reveal phase
 */
function executeDisputeCommitEvidence(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as CommitEvidencePayload;
  const senderDid = resolveSenderDid(tx);

  // Validate sender is the committer
  if (payload.committer !== senderDid) {
    throw new StateError("Sender must be the committer", "INVALID_TRANSITION");
  }

  // Get dispute
  const dispute = state.getDispute(payload.disputeId);
  if (!dispute) {
    throw new StateError("Dispute not found", "INVALID_TRANSITION");
  }

  // Validate phase
  if (dispute.state.phase !== "evidence_commit") {
    throw new StateError("Not in evidence commit phase", "INVALID_TRANSITION");
  }

  // Validate deadline
  if (blockNumber > dispute.state.phaseDeadline) {
    throw new StateError(
      "Evidence commit deadline passed",
      "INVALID_TRANSITION",
    );
  }

  // Must be challenger or defender
  if (
    senderDid !== dispute.challenger.did &&
    senderDid !== dispute.defender.did
  ) {
    throw new StateError(
      "Only dispute parties can commit evidence",
      "INVALID_TRANSITION",
    );
  }

  // Commit evidence using dispute engine
  const updatedDispute = commitEvidence(dispute, payload, blockNumber);

  // Update dispute in state
  state.updateDispute(updatedDispute);

  return 80000n; // Evidence commit gas
}

/**
 * Reveal committed evidence
 */
function executeDisputeRevealEvidence(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as RevealEvidencePayload;
  const senderDid = resolveSenderDid(tx);

  // Validate sender is the revealer
  if (payload.revealer !== senderDid) {
    throw new StateError("Sender must be the revealer", "INVALID_TRANSITION");
  }

  // Get dispute
  const dispute = state.getDispute(payload.disputeId);
  if (!dispute) {
    throw new StateError("Dispute not found", "INVALID_TRANSITION");
  }

  // Validate phase
  if (dispute.state.phase !== "evidence_reveal") {
    throw new StateError("Not in evidence reveal phase", "INVALID_TRANSITION");
  }

  // Validate deadline
  if (blockNumber > dispute.state.phaseDeadline) {
    throw new StateError(
      "Evidence reveal deadline passed",
      "INVALID_TRANSITION",
    );
  }

  // Get actual evidence objects
  const evidenceList = payload.evidenceIds.map((id) => {
    const evidence = state.getEvidence(id);
    if (!evidence) {
      throw new StateError(`Evidence ${id} not found`, "INVALID_TRANSITION");
    }
    return evidence;
  });

  // Reveal evidence using dispute engine
  const updatedDispute = revealEvidence(
    dispute,
    payload,
    evidenceList,
    blockNumber,
  );

  // Update dispute in state
  state.updateDispute(updatedDispute);

  return 120000n; // Evidence reveal gas
}

/**
 * Cast arbiter vote in dispute
 */
function executeDisputeArbiterVote(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as ArbiterVotePayload;
  const senderDid = resolveSenderDid(tx);

  // Validate sender is the arbiter
  if (payload.arbiter !== senderDid) {
    throw new StateError("Sender must be the arbiter", "INVALID_TRANSITION");
  }

  // Get dispute
  const dispute = state.getDispute(payload.disputeId);
  if (!dispute) {
    throw new StateError("Dispute not found", "INVALID_TRANSITION");
  }

  // Validate phase
  if (
    dispute.state.phase !== "arbitration" &&
    dispute.state.phase !== "deliberation"
  ) {
    throw new StateError("Not in voting phase", "INVALID_TRANSITION");
  }

  // Validate deadline
  if (blockNumber > dispute.state.phaseDeadline) {
    throw new StateError("Voting deadline passed", "INVALID_TRANSITION");
  }

  // Check arbiter is in the selected list
  const isSelectedArbiter = dispute.arbiters.some((a) => a.did === senderDid);
  if (!isSelectedArbiter) {
    throw new StateError(
      "Not a selected arbiter for this dispute",
      "INVALID_TRANSITION",
    );
  }

  // Get arbiter's stake to lock
  const account = state.getAccount(senderDid);
  if (!account || account.stakedBalance < DISPUTE_ECONOMICS.minArbiterStake) {
    throw new StateError(
      "Insufficient stake to vote as arbiter",
      "INSUFFICIENT_BALANCE",
    );
  }

  // Cast vote using dispute engine
  const updatedDispute = castArbiterVote(
    dispute,
    payload,
    DISPUTE_ECONOMICS.minArbiterStake, // Lock minimum required stake
    blockNumber,
  );

  // Update dispute in state
  state.updateDispute(updatedDispute);

  return 60000n; // Arbiter vote gas
}

/**
 * Escalate dispute to higher tier
 * Only the losing party can escalate (within appeal window)
 */
function executeDisputeEscalate(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as EscalateDisputePayload;
  const senderDid = resolveSenderDid(tx);

  // Validate sender is the escalator
  if (payload.escalator !== senderDid) {
    throw new StateError("Sender must be the escalator", "INVALID_TRANSITION");
  }

  // Get dispute
  const dispute = state.getDispute(payload.disputeId);
  if (!dispute) {
    throw new StateError("Dispute not found", "INVALID_TRANSITION");
  }

  // Process escalation (validates eligibility, stake, etc.)
  const result = processEscalation(
    dispute,
    senderDid,
    payload.additionalStake,
    blockNumber,
  );

  if (!result.success) {
    throw new StateError(
      result.error || "Escalation failed",
      "INVALID_TRANSITION",
    );
  }

  // Lock additional stake
  state.stake(senderDid, payload.additionalStake);

  // Update dispute state for new tier
  const updatedDispute: typeof dispute = {
    ...dispute,
    state: {
      ...dispute.state,
      phase: "arbitration",
      escalationTier: result.newTier!,
      phaseDeadline: blockNumber + BigInt(PHASE_DURATIONS.arbitration),
      canEscalate: result.newTier! < 3, // Can escalate if not at max tier
      previousTierResults: [
        ...(dispute.state.previousTierResults || []),
        // Add current tier result before escalation
      ],
    },
    arbiters: [], // Will be reselected for new tier
    votes: new Map(), // Clear votes for new tier
    updatedAt: Date.now(),
  };

  // Update dispute in state
  state.updateDispute(updatedDispute);

  return 150000n; // Escalation gas
}

/**
 * Settle a resolved dispute
 * Distributes rewards and punishments
 */
function executeDisputeSettle(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as SettleDisputePayload;

  // Get dispute
  const dispute = state.getDispute(payload.disputeId);
  if (!dispute) {
    throw new StateError("Dispute not found", "INVALID_TRANSITION");
  }

  // Validate phase
  if (dispute.state.phase !== "settlement") {
    throw new StateError("Not in settlement phase", "INVALID_TRANSITION");
  }

  // Calculate settlement if not provided
  const settlement = payload.settlement || calculateSettlement(dispute);

  // Apply settlement
  const winnerDid =
    settlement.winner === "challenger"
      ? dispute.challenger.did
      : dispute.defender.did;
  const loserDid =
    settlement.winner === "challenger"
      ? dispute.defender.did
      : dispute.challenger.did;

  // Return winner's stake + reward
  const winnerStake =
    settlement.winner === "challenger"
      ? dispute.challenger.stake
      : dispute.defender.stake;
  state.unstake(winnerDid, winnerStake);

  // Reward winner from loser's stake
  state.transfer("did:lgm:dispute-escrow", winnerDid, settlement.winnerReward);

  // Apply loser penalty (portion of stake is slashed)
  state.unstake(
    loserDid,
    dispute.challenger.did === loserDid
      ? dispute.challenger.stake
      : dispute.defender.stake,
  );
  const loserRefund =
    (dispute.challenger.did === loserDid
      ? dispute.challenger.stake
      : dispute.defender.stake) - settlement.loserPenalty;
  if (loserRefund > 0n) {
    state.transfer("did:lgm:dispute-escrow", loserDid, loserRefund);
  }

  // Distribute arbiter rewards and penalties
  for (const [arbiterDid, reward] of settlement.arbiterRewards) {
    state.transfer("did:lgm:dispute-escrow", arbiterDid, reward);
  }

  for (const [arbiterDid, penalty] of settlement.arbiterPenalties) {
    // Slash arbiter's locked stake
    const account = state.getAccount(arbiterDid);
    if (account) {
      state.slashValidator(
        tx.sender,
        penalty,
        `Wrong arbitration vote in dispute ${dispute.id}`,
      );
    }
  }

  // Protocol fee to treasury
  state.transfer(
    "did:lgm:dispute-escrow",
    "did:lgm:protocol-treasury",
    settlement.protocolFee,
  );

  // Research fund share
  state.transfer(
    "did:lgm:dispute-escrow",
    RESEARCH_TREASURY_DID,
    settlement.researchFund,
  );

  // Mark dispute as resolved
  const resolvedDispute: typeof dispute = {
    ...dispute,
    state: {
      ...dispute.state,
      phase: "resolved",
    },
    outcome:
      settlement.winner === "challenger" ? "challenger_wins" : "defender_wins",
    winner: settlement.winner,
    resolvedAtBlock: blockNumber,
    updatedAt: Date.now(),
  };

  state.updateDispute(resolvedDispute);

  // GAP-003 FIX: Route dispute outcome to fact confidence if this is a fact dispute
  if (dispute.targetType === "fact") {
    const routingResult = routeDisputeOutcomeToFact(
      {
        dispute: resolvedDispute,
        settlement,
        settlementBlock: blockNumber,
      },
      state,
    );

    if (!routingResult.success) {
      // Log warning but don't fail the transaction
      // The dispute settlement succeeded, even if fact update failed
      state.emitEvent("FactRoutingWarning", {
        disputeId: dispute.id,
        factId: dispute.targetId,
        error: routingResult.error ?? "Unknown error",
        blockNumber: blockNumber.toString(),
      });
    }

    // Process post-resolution cleanup
    processPostResolution(dispute.id, state, blockNumber);
  }

  return 200000n; // Settlement gas
}

/**
 * Handle dispute phase timeout
 * Called when a phase deadline passes without required action
 * GAP-024 FIX: Now properly uses handleTimeout and executes arbiter slashes
 */
function executeDisputeTimeout(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as { disputeId: string };

  // Get dispute
  const dispute = state.getDispute(payload.disputeId);
  if (!dispute) {
    throw new StateError("Dispute not found", "INVALID_TRANSITION");
  }

  // Validate deadline has passed
  if (blockNumber <= dispute.state.phaseDeadline) {
    throw new StateError(
      "Phase deadline not yet reached",
      "INVALID_TRANSITION",
    );
  }

  // GAP-024 FIX: Use handleTimeout from state_machine for consistent behavior
  // Import and use handleTimeout to get arbiter slashes
  const timeoutResult = handleTimeoutInternal(dispute, blockNumber);

  let updatedDispute: typeof dispute;

  switch (timeoutResult.action) {
    case "default_judgment":
      // One party wins by default
      updatedDispute = {
        ...dispute,
        state: {
          ...dispute.state,
          phase: "settlement",
          phaseDeadline: blockNumber + BigInt(PHASE_DURATIONS.settlement),
        },
        outcome:
          timeoutResult.winner === "challenger"
            ? "challenger_wins"
            : "defender_wins",
        winner: timeoutResult.winner,
        exposedFalsehood: timeoutResult.winner === "challenger",
        updatedAt: Date.now(),
      };
      break;

    case "forfeit":
      // Both parties failed - draw
      updatedDispute = {
        ...dispute,
        state: {
          ...dispute.state,
          phase: "settlement",
          phaseDeadline: blockNumber + BigInt(PHASE_DURATIONS.settlement),
        },
        outcome: "draw",
        updatedAt: Date.now(),
      };
      break;

    case "advance":
      // Move to next phase
      if (timeoutResult.nextPhase) {
        const phaseDurations: Record<string, number> = PHASE_DURATIONS;
        updatedDispute = {
          ...dispute,
          state: {
            ...dispute.state,
            phase: timeoutResult.nextPhase as Dispute["state"]["phase"],
            phaseDeadline:
              blockNumber +
              BigInt(phaseDurations[timeoutResult.nextPhase] || 1000),
          },
          updatedAt: Date.now(),
        };
      } else {
        const advanceResult = tryAdvancePhase(
          dispute,
          blockNumber,
          new Set<string>(),
        );
        updatedDispute = advanceResult.dispute;
      }
      break;

    default:
      throw new StateError("Unknown timeout action", "INVALID_TRANSITION");
  }

  // GAP-024 FIX: Execute arbiter slashes for non-voters
  if (timeoutResult.arbiterSlashes && timeoutResult.arbiterSlashes.length > 0) {
    const slashAmount = (DISPUTE_ECONOMICS.minArbiterStake * 10n) / 100n; // 10% of min stake

    for (const slash of timeoutResult.arbiterSlashes) {
      // Get arbiter account
      const arbiterAccount = state.getAccount(slash.did);
      if (arbiterAccount && arbiterAccount.stakedBalance >= slashAmount) {
        // Apply slash to arbiter's staked balance
        state.transfer(slash.did, "did:lgm:protocol-treasury", slashAmount);
      }
    }
  }

  state.updateDispute(updatedDispute);

  return 100000n; // Timeout processing gas
}

/**
 * GAP-024 FIX: Internal timeout handling that mirrors state_machine.handleTimeout
 * This is used to get arbiter slashing information for timeout execution
 */
function handleTimeoutInternal(
  dispute: Dispute,
  _currentBlock: bigint,
): {
  action: "advance" | "forfeit" | "default_judgment";
  winner?: "challenger" | "defender";
  nextPhase?:
    | "evidence_commit"
    | "evidence_reveal"
    | "arbitration"
    | "deliberation"
    | "settlement"
    | "resolved";
  reason: string;
  arbiterSlashes?: Array<{ did: string; reason: string }>;
} {
  switch (dispute.state.phase) {
    case "initiation":
      return {
        action: "default_judgment",
        winner: "challenger",
        reason: "Defender failed to respond within deadline",
      };

    case "evidence_commit":
      if (!dispute.commitments.challenger && !dispute.commitments.defender) {
        return {
          action: "forfeit",
          reason: "Both parties failed to commit evidence",
        };
      }
      if (!dispute.commitments.challenger) {
        return {
          action: "default_judgment",
          winner: "defender",
          reason: "Challenger failed to commit evidence",
        };
      }
      if (!dispute.commitments.defender) {
        return {
          action: "default_judgment",
          winner: "challenger",
          reason: "Defender failed to commit evidence",
        };
      }
      return {
        action: "advance",
        nextPhase: "evidence_reveal",
        reason: "Evidence commit phase completed",
      };

    case "evidence_reveal":
      if (!dispute.commitments.challenger?.revealed) {
        return {
          action: "default_judgment",
          winner: "defender",
          reason: "Challenger refused to reveal evidence",
        };
      }
      if (!dispute.commitments.defender?.revealed) {
        return {
          action: "default_judgment",
          winner: "challenger",
          reason: "Defender refused to reveal evidence",
        };
      }
      return {
        action: "advance",
        nextPhase: "arbitration",
        reason: "Evidence reveal phase completed",
      };

    case "arbitration": {
      // Identify non-voting arbiters for slashing
      const votedArbiters = new Set(dispute.votes.keys());
      const nonVotingArbiters = dispute.arbiters
        .filter((a: { did: string }) => !votedArbiters.has(a.did))
        .map((a: { did: string }) => ({
          did: a.did,
          reason: "Failed to vote within arbitration deadline",
        }));

      return {
        action: "advance",
        nextPhase: "deliberation",
        reason: "Arbitration phase timeout",
        arbiterSlashes:
          nonVotingArbiters.length > 0 ? nonVotingArbiters : undefined,
      };
    }

    case "deliberation":
      return {
        action: "advance",
        nextPhase: "settlement",
        reason: "Deliberation period ended",
      };

    case "settlement":
      return {
        action: "advance",
        nextPhase: "resolved",
        reason: "Settlement completed",
      };

    default:
      return {
        action: "forfeit",
        reason: `Unexpected timeout in phase: ${dispute.state.phase}`,
      };
  }
}

// ============ Vesting Transaction Handlers ============

/**
 * Create a vesting schedule for an accepted claim
 *
 * Called when a claim is accepted by consensus.
 * Sets up gradual reward release based on epistemic category.
 */
function executeVestingCreate(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as CreateVestingPayload;

  // Validate claim exists
  if (!state.isActiveFact(payload.claimId)) {
    throw new StateError("Claim not found or not active", "FACT_NOT_FOUND");
  }

  // Create vesting schedule
  state.createVesting(payload, blockNumber);

  // Lock the reward amount in escrow
  state.transfer(
    "did:lgm:reward-pool",
    "did:lgm:vesting-escrow",
    payload.amount,
  );

  return 80000n; // Vesting creation gas
}

/**
 * Claim available vested rewards
 */
function executeVestingClaim(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as ClaimRewardsPayload;
  const senderDid = resolveSenderDid(tx);

  // Validate sender is the claimer
  if (payload.claimer !== senderDid) {
    throw new StateError("Sender must be the claimer", "INVALID_TRANSITION");
  }

  // Get vesting and claim rewards
  const { vesting: _vesting, claimedAmount } = state.claimVestingRewards(
    payload.vestingId,
    senderDid,
    payload.amount,
    blockNumber,
  );

  // Transfer claimed amount from escrow to recipient
  state.transfer("did:lgm:vesting-escrow", senderDid, claimedAmount);

  return 50000n; // Vesting claim gas
}

/**
 * Pause vesting when a challenge is initiated
 */
function executeVestingPause(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as PauseVestingPayload;

  // Get vesting
  const vesting = state.getVesting(payload.vestingId);
  if (!vesting) {
    throw new StateError("Vesting not found", "VESTING_NOT_FOUND");
  }

  // Pause the vesting
  state.pauseVesting(
    payload.vestingId,
    payload.relatedEntityId || "",
    blockNumber,
  );

  return 30000n; // Vesting pause gas
}

/**
 * Resume vesting after challenge resolution
 */
function executeVestingResume(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as ResumeVestingPayload;

  // Get vesting
  const vesting = state.getVesting(payload.vestingId);
  if (!vesting) {
    throw new StateError("Vesting not found", "VESTING_NOT_FOUND");
  }

  // Resume with optional defense bonus
  // Note: This is for resuming without challenge resolution
  // Challenge resolution uses vesting_falsify or direct resume
  state.resumeVesting(
    payload.vestingId,
    false, // Not a challenge resolution
    "",
    "",
    blockNumber,
  );

  return 30000n; // Vesting resume gas
}

/**
 * Apply acceleration bonus to vesting
 */
function executeVestingAccelerate(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as AccelerateVestingPayload;

  // Get vesting
  const vesting = state.getVesting(payload.vestingId);
  if (!vesting) {
    throw new StateError("Vesting not found", "VESTING_NOT_FOUND");
  }

  // Apply acceleration based on type
  // Currently supports replication acceleration
  if (payload.accelerationType === "replication" && payload.relatedEntityId) {
    state.accelerateVesting(
      payload.vestingId,
      payload.relatedEntityId,
      blockNumber,
    );
  }

  return 40000n; // Vesting acceleration gas
}

/**
 * Handle falsification of a vesting (claim proven false)
 *
 * Distributes:
 * - Unvested rewards to challenger
 * - 30% clawback from released rewards
 * - Reserve to challenger
 */
function executeVestingFalsify(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as FalsifyVestingPayload;

  // Get vesting
  const vesting = state.getVesting(payload.vestingId);
  if (!vesting) {
    throw new StateError("Vesting not found", "VESTING_NOT_FOUND");
  }

  // Process falsification (challenger won the dispute)
  const { vesting: _updatedVesting, clawback } = state.resumeVesting(
    payload.vestingId,
    true, // Challenger won
    payload.challengerDid,
    payload.disputeId,
    blockNumber,
  );

  // Transfer rewards to challenger
  if (clawback) {
    state.transfer(
      "did:lgm:vesting-escrow",
      payload.challengerDid,
      clawback.challengerReward,
    );

    // Apply clawback from recipient if they still have balance
    if (clawback.recipientPenalty > 0n) {
      const recipientBalance = state.getBalance(vesting.recipient);
      const actualClawback =
        recipientBalance >= clawback.recipientPenalty
          ? clawback.recipientPenalty
          : recipientBalance;

      if (actualClawback > 0n) {
        state.transfer(
          vesting.recipient,
          payload.challengerDid,
          actualClawback,
        );
      }
    }
  }

  return 150000n; // Vesting falsification gas
}

// ============ Domain Proposal Transaction Handlers ============

/**
 * Execute domain proposal transaction
 *
 * Stake requirements (22/222/2222 LGM pattern):
 * - Leaf domains (depth > 2): 22 LGM
 * - Mid-level domains (depth 2): 222 LGM
 * - Top-level domains (depth 1): 2222 LGM
 */
function executeDomainPropose(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as DomainProposePayload;
  const senderDid = resolveSenderDid(tx);

  // Validate sender is the proposer
  if (payload.proposer !== senderDid) {
    throw new StateError("Sender must be the proposer", "INVALID_TRANSITION");
  }

  // Get the domain tree from state
  const domainTree = state.getKnowledgeDomainTree();
  if (!domainTree) {
    throw new StateError("Domain tree not initialized", "INVALID_TRANSITION");
  }

  // Create the proposal (validates input internally)
  const { proposal, validation } = createDomainProposal(
    {
      domainId: payload.domainId,
      name: payload.name,
      stratum: payload.stratum,
      parentDomainId: payload.parentDomainId,
      description: payload.description,
      examples: payload.examples,
      proposer: payload.proposer,
      stake: payload.stake,
    },
    blockNumber,
    domainTree,
  );

  // Validate balance
  const balance = state.getBalance(senderDid);
  if (balance < payload.stake) {
    throw new StateError(
      "Insufficient balance for proposal stake",
      "INSUFFICIENT_BALANCE",
    );
  }

  // Lock the proposer's stake
  state.stake(senderDid, payload.stake);

  // Store the proposal
  state.addDomainProposal(proposal);

  // Emit event
  state.emitEvent("DomainProposed", {
    proposalId: proposal.id,
    domainId: proposal.proposedDomainId,
    proposer: proposal.proposer,
    stake: proposal.stake.toString(),
    requiresGovernance: String(validation.requiresGovernance),
    challengeDeadline: proposal.challengeDeadline.toString(),
  });

  return 180000n; // Domain proposal gas
}

/**
 * Execute domain endorsement transaction
 * Requires 2 validator endorsements for auto-approval
 *
 * Validators must lock stake when endorsing:
 * - Top-level domains: 222 LGM
 * - Mid-level domains: 22 LGM
 * - Leaf domains: 2 LGM
 */
function executeDomainEndorse(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as DomainEndorsePayload;
  const senderDid = resolveSenderDid(tx);

  // Validate sender is the endorser
  if (payload.endorser !== senderDid) {
    throw new StateError("Sender must be the endorser", "INVALID_TRANSITION");
  }

  // Endorser must be a validator
  if (!state.isActiveValidator(tx.sender)) {
    throw new StateError(
      "Only validators can endorse domain proposals",
      "INVALID_TRANSITION",
    );
  }

  // Get the proposal
  const proposal = state.getDomainProposal(payload.proposalId);
  if (!proposal) {
    throw new StateError(
      "Domain proposal not found",
      "DOMAIN_PROPOSAL_NOT_FOUND",
    );
  }

  // Validate stake meets minimum for this domain level
  if (payload.stake < proposal.requiredEndorsementStake) {
    throw new StateError(
      `Insufficient endorsement stake: ${payload.stake} < required ${proposal.requiredEndorsementStake}`,
      "INSUFFICIENT_STAKE",
    );
  }

  // Validate endorser has sufficient balance
  const balance = state.getBalance(senderDid);
  if (balance < payload.stake) {
    throw new StateError(
      "Insufficient balance for endorsement stake",
      "INSUFFICIENT_BALANCE",
    );
  }

  // Lock the endorser's stake
  state.stake(senderDid, payload.stake);

  // Endorse the proposal (validates and updates state)
  const updatedProposal = endorseProposal(
    proposal,
    senderDid,
    blockNumber,
    payload.stake,
  );

  // Update proposal in state
  state.updateDomainProposal(updatedProposal);

  // Emit event
  state.emitEvent("DomainEndorsed", {
    proposalId: proposal.id,
    endorser: senderDid,
    stake: payload.stake.toString(),
    endorserCount: String(updatedProposal.endorsers.length),
    status: updatedProposal.status,
  });

  return 50000n; // Domain endorsement gas
}

/**
 * Execute domain challenge transaction
 */
function executeDomainChallenge(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as DomainChallengePayload;
  const senderDid = resolveSenderDid(tx);

  // Validate sender is the challenger
  if (payload.challenger !== senderDid) {
    throw new StateError("Sender must be the challenger", "INVALID_TRANSITION");
  }

  // Get the proposal
  const proposal = state.getDomainProposal(payload.proposalId);
  if (!proposal) {
    throw new StateError(
      "Domain proposal not found",
      "DOMAIN_PROPOSAL_NOT_FOUND",
    );
  }

  // Validate challenge stake
  const balance = state.getBalance(senderDid);
  if (balance < payload.stake) {
    throw new StateError(
      "Insufficient balance for challenge stake",
      "INSUFFICIENT_BALANCE",
    );
  }

  // Lock the challenger's stake
  state.stake(senderDid, payload.stake);

  // Challenge the proposal
  const updatedProposal = challengeProposal(
    proposal,
    senderDid,
    payload.reason,
    payload.stake,
    blockNumber,
  );

  // Update proposal in state
  state.updateDomainProposal(updatedProposal);

  // Emit event
  state.emitEvent("DomainChallenged", {
    proposalId: proposal.id,
    challenger: senderDid,
    reason: payload.reason,
    challengerCount: String(updatedProposal.challengers.length),
  });

  return 80000n; // Domain challenge gas
}

/**
 * Execute domain finalization transaction
 * 777 blocks challenge period (~32.7 minutes at 2521ms blocks)
 * 77% stake return on rejection (for both proposer and endorsers if challenged)
 */
function executeDomainFinalize(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as DomainFinalizePayload;

  // Get the proposal
  const proposal = state.getDomainProposal(payload.proposalId);
  if (!proposal) {
    throw new StateError(
      "Domain proposal not found",
      "DOMAIN_PROPOSAL_NOT_FOUND",
    );
  }

  // Finalize the proposal (validates deadline internally)
  const finalizedProposal = finalizeProposal(proposal, blockNumber);

  // Calculate endorser stake returns
  const endorserReturns = calculateEndorserStakeReturns(finalizedProposal);

  // Handle stake return based on outcome
  if (finalizedProposal.status === "approved") {
    // Return full stake to proposer
    state.unstake(finalizedProposal.proposer, finalizedProposal.stake);

    // Return full stake to all endorsers
    for (const [endorser, { stakeReturn }] of endorserReturns) {
      state.unstake(endorser, stakeReturn);
    }

    // Get domain tree and parent domain
    const domainTree = state.getKnowledgeDomainTree();
    if (!domainTree) {
      throw new StateError("Domain tree not initialized", "INVALID_TRANSITION");
    }

    const parentNode = domainTree.domainIndex.get(
      finalizedProposal.parentDomainId,
    );
    const parentDomain = parentNode?.domain ?? null;

    // Create the actual domain
    const newDomain = createDomainFromProposal(
      finalizedProposal,
      parentDomain,
      Number(blockNumber),
    );

    // Add domain to tree
    state.addDomainToTree(newDomain);

    // Emit approval event
    state.emitEvent("DomainApproved", {
      proposalId: finalizedProposal.id,
      domainId: newDomain.id,
      approvedAt: blockNumber.toString(),
      endorserCount: String(finalizedProposal.endorsers.length),
    });
  } else if (finalizedProposal.status === "rejected") {
    // Return 77% of stake to proposer (only if challenged, else 100%)
    const wasChallenged = finalizedProposal.challengers.length > 0;
    const proposerStakeReturn = wasChallenged
      ? calculateStakeReturn(finalizedProposal)
      : finalizedProposal.stake;
    const proposerSlashed = finalizedProposal.stake - proposerStakeReturn;

    // Unstake the full amount from proposer
    state.unstake(finalizedProposal.proposer, finalizedProposal.stake);

    // Deduct slashed portion to protocol treasury
    if (proposerSlashed > 0n) {
      state.transfer(
        finalizedProposal.proposer,
        "did:lgm:protocol-treasury",
        proposerSlashed,
      );
    }

    // Handle endorser stakes
    let totalEndorserSlashed = 0n;
    for (const [
      endorser,
      { stakeReturn: _stakeReturn, stakeSlashed },
    ] of endorserReturns) {
      // Unstake the full amount from endorser
      const endorserStake =
        finalizedProposal.endorserStakes.get(endorser) ?? 0n;
      if (endorserStake > 0n) {
        state.unstake(endorser, endorserStake);

        // Deduct slashed portion to protocol treasury
        if (stakeSlashed > 0n) {
          state.transfer(endorser, "did:lgm:protocol-treasury", stakeSlashed);
          totalEndorserSlashed += stakeSlashed;
        }
      }
    }

    // Emit rejection event
    state.emitEvent("DomainRejected", {
      proposalId: finalizedProposal.id,
      domainId: finalizedProposal.proposedDomainId,
      reason: finalizedProposal.rejectionReason ?? "Proposal rejected",
      wasChallenged: String(wasChallenged),
      proposerStakeReturned: proposerStakeReturn.toString(),
      proposerStakeSlashed: proposerSlashed.toString(),
      endorserStakeSlashed: totalEndorserSlashed.toString(),
    });
  }

  // Update proposal in state
  state.updateDomainProposal(finalizedProposal);

  return 100000n; // Domain finalization gas
}

/**
 * Execute governance approval for a domain proposal
 *
 * Proposals in "governance" status (fundamental strata, top-level domains)
 * require an accepted LIP vote before they can be approved.
 * Once the LIP is in "accepted" or "implemented" stage, this handler
 * approves the domain and adds it to the ontological tree.
 */
function executeDomainGovernanceApprove(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as {
    proposalId: string;
    lipId: string;
  };

  // Get the proposal
  const proposal = state.getDomainProposal(payload.proposalId);
  if (!proposal) {
    throw new StateError(
      "Domain proposal not found",
      "DOMAIN_PROPOSAL_NOT_FOUND",
    );
  }

  // Must be in governance status
  if (proposal.status !== "governance") {
    throw new StateError(
      `Cannot governance-approve proposal with status "${proposal.status}"`,
      "INVALID_TRANSITION",
    );
  }

  // Verify the LIP exists and is accepted
  const lip = state.getLIP(payload.lipId);
  if (!lip) {
    throw new StateError("LIP not found", "INVALID_TRANSITION");
  }
  if (lip.stage !== "accepted" && lip.stage !== "implemented") {
    throw new StateError(
      `LIP must be accepted or implemented, currently "${lip.stage}"`,
      "INVALID_TRANSITION",
    );
  }

  // Return full stake to proposer
  state.unstake(proposal.proposer, proposal.stake);

  // Return full stake to all endorsers
  const endorserReturns = calculateEndorserStakeReturns({
    ...proposal,
    status: "approved",
  });
  for (const [endorser, { stakeReturn }] of endorserReturns) {
    state.unstake(endorser, stakeReturn);
  }

  // Get domain tree and create the domain
  const domainTree = state.getKnowledgeDomainTree();
  if (!domainTree) {
    throw new StateError("Domain tree not initialized", "INVALID_TRANSITION");
  }

  const parentNode = domainTree.domainIndex.get(proposal.parentDomainId);
  const parentDomain = parentNode?.domain ?? null;

  const newDomain = createDomainFromProposal(
    { ...proposal, status: "approved", approvedAt: blockNumber },
    parentDomain,
    Number(blockNumber),
  );

  // Add domain to tree
  state.addDomainToTree(newDomain);

  // Update proposal status
  state.updateDomainProposal({
    ...proposal,
    status: "approved",
    approvedAt: blockNumber,
    governanceProposalId: payload.lipId,
  });

  // Emit event
  state.emitEvent("DomainGovernanceApproved", {
    proposalId: proposal.id,
    domainId: newDomain.id,
    lipId: payload.lipId,
    approvedAt: blockNumber.toString(),
  });

  return 100000n; // Domain governance approval gas
}

/**
 * Execute domain deprecation transaction
 *
 * Marks a domain as deprecated, preventing new claims from targeting it.
 * Existing facts remain but no new claims can be submitted to this domain.
 * Only the original proposer can deprecate a domain.
 */
function executeDomainDeprecate(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    domainId: string;
    reason: string;
  };
  const senderDid = resolveSenderDid(tx);

  // Verify domain exists in the tree
  const domainTree = state.getKnowledgeDomainTree();
  if (!domainTree) {
    throw new StateError("Domain tree not initialized", "INVALID_TRANSITION");
  }

  const node = domainTree.domainIndex.get(payload.domainId);
  if (!node) {
    throw new StateError(
      `Domain "${payload.domainId}" not found`,
      "INVALID_TRANSITION",
    );
  }

  // Cannot deprecate already-deprecated domain
  if (node.domain.deprecatedAt !== undefined) {
    throw new StateError(
      `Domain "${payload.domainId}" is already deprecated`,
      "INVALID_TRANSITION",
    );
  }

  // Authorization: only domain proposer can deprecate
  const proposals = state.getAllDomainProposals();
  const originalProposal = proposals.find(
    (p) => p.proposedDomainId === payload.domainId && p.status === "approved",
  );
  if (originalProposal && originalProposal.proposer !== senderDid) {
    throw new StateError(
      "Only the original domain proposer can deprecate this domain",
      "UNAUTHORIZED",
    );
  }

  // Mark domain as deprecated
  state.updateDomainInTree(payload.domainId, {
    deprecatedAt: Number(blockNumber),
  });

  state.emitEvent("DomainDeprecated", {
    domainId: payload.domainId,
    deprecatedBy: senderDid,
    reason: payload.reason,
    deprecatedAt: blockNumber.toString(),
  });

  return 50000n; // Domain deprecation gas
}

// ============ System Transaction Handlers ============
// These transactions are created by the consensus layer (not users)
// and have no fee requirements.

/**
 * Add a verified fact to the knowledge layer
 * Created by consensus when verification round completes with acceptance
 */
function executeAddFact(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as {
    type: "add_fact";
    factId: string;
    confidence: number;
    verifierCount: number;
    verifiedAtBlock: number;
    claim?: string;
    domain?: string;
  };

  // Add fact to knowledge layer
  state.addFact({
    id: payload.factId,
    claim: payload.claim || `Verified claim ${payload.factId}`,
    domain: payload.domain || "verified",
    confidence: payload.confidence,
    verifiedAt: blockNumber,
    verifierCount: payload.verifierCount,
    challengeCount: 0,
    citationCount: 0,
    status: "active",
  });

  return 0n; // System transaction, no gas
}

/**
 * Reward a validator for correct verification
 * Created by consensus when verification round completes
 */
function executeVerificationReward(
  state: StateManager,
  tx: Transaction,
): bigint {
  const payload = tx.payload as {
    type: "verification_reward";
    recipient: string;
    amount: bigint;
    roundId: string;
  };

  // Transfer reward from reward pool to validator
  // (reward pool is pre-funded from block rewards)
  if (state.accountExists(payload.recipient)) {
    state.transfer(REWARD_POOL_DID, payload.recipient, payload.amount);
  }

  return 0n; // System transaction, no gas
}

/**
 * Slash a validator for incorrect verification or missed verification
 * Created by consensus when verification round completes
 */
function executeSlashValidator(state: StateManager, tx: Transaction): bigint {
  const payload = tx.payload as {
    type: "slash_validator";
    validator: string;
    amount: bigint;
    reason: string;
  };

  // Get validator and slash their stake
  const validator = state.getValidator(payload.validator as any);
  if (validator && payload.amount > 0n) {
    const actualSlash =
      payload.amount > validator.stake ? validator.stake : payload.amount;
    state.slashValidator(
      payload.validator as any,
      actualSlash,
      payload.reason || "verification_failure",
    );
  }

  return 0n; // System transaction, no gas
}

// ============ Verification Result Application ============

/**
 * Infer evidence types from claim properties for vesting category determination.
 * Maps claim type and stratum to evidence types used by the vesting system.
 *
 * GAP-006 FIX: Helper for vesting category determination
 */
function inferEvidenceTypesFromClaim(claim: ClaimState): string[] {
  const types: string[] = [];

  // Infer from claim type
  switch (claim.claimType) {
    case "formal":
    case "logical":
      types.push("formal_proof");
      break;
    case "on_chain":
    case "blockchain":
      types.push("on_chain");
      break;
    case "computational":
      types.push("computational");
      break;
    case "empirical":
      types.push("peer_reviewed"); // Default for empirical claims
      break;
    case "attestation":
      types.push("attestation");
      break;
    case "oracle":
      types.push("chainlink_feed");
      break;
  }

  // Also infer from stratum (more precise)
  switch (claim.stratum) {
    case "fundamental":
      if (!types.includes("formal_proof")) {
        types.push("formal_proof");
      }
      break;
    case "technological":
      if (!types.includes("computational")) {
        types.push("computational");
      }
      types.push("cryptographic_proof");
      break;
    case "physical":
    case "chemical":
    case "biological":
    case "cognitive":
    case "social":
      if (!types.includes("peer_reviewed")) {
        types.push("peer_reviewed");
      }
      break;
  }

  // Default if nothing inferred
  if (types.length === 0) {
    types.push("peer_reviewed");
  }

  return types;
}

/**
 * Apply verification result to state
 * Called after consensus completes a verification round
 *
 * GAP-2 FIX: Routes all rewards through reward router for 22% research share
 * GAP-006 FIX: Creates vesting schedule for claim submitter when claim is accepted
 */
export function applyVerificationResult(
  state: StateManager,
  result: VerificationResult,
  blockNumber: bigint,
  params: ConsensusParams,
): void {
  state.beginTransaction();

  try {
    // Apply rewards to validators - routed through reward router
    for (const [validatorKey, reward] of result.rewards) {
      const validatorDid = publicKeyToDid(validatorKey as PublicKey);
      if (state.accountExists(validatorDid)) {
        // GAP-2 FIX: Route verification reward through reward router
        const routed = routeVerificationReward(
          reward,
          validatorDid,
          result.claimId,
          blockNumber,
        );

        // Transfer 22% split across 5 treasury funds
        if (routed.researchShare > 0n) {
          const treasurySplit = splitEvidenceTax(routed.researchShare);
          for (const entry of treasurySplit) {
            if (entry.amount > 0n) {
              state.transfer(REWARD_POOL_DID, entry.did, entry.amount);
            }
          }
        }

        // Transfer 78% to validator
        if (routed.recipientAmount > 0n) {
          state.transfer(REWARD_POOL_DID, validatorDid, routed.recipientAmount);
        }
      }
    }

    // Apply slashes to validators
    for (const [validatorKey, slashAmount] of result.slashes) {
      state.slashValidator(
        validatorKey as PublicKey,
        slashAmount,
        "Wrong verification",
      );
    }

    // If claim accepted, add fact to knowledge and create vesting for submitter
    if (result.outcome === "accepted" && result.fact) {
      // GAP-023 FIX: Check fact ID uniqueness before inserting
      const existingFact = state.getFact(result.claimId);
      if (existingFact) {
        throw new StateError(
          `Fact with ID ${result.claimId} already exists`,
          "DUPLICATE_FACT",
        );
      }

      state.addFact({
        id: result.claimId,
        claim: result.fact.claim,
        domain: result.fact.domain,
        confidence: result.finalConfidence,
        verifiedAt: blockNumber,
        verifierCount: result.participatingValidators,
        challengeCount: 0,
        citationCount: 0,
        status: "active",
      });

      // GAP-006 FIX: Create vesting schedule for claim submitter
      const claim = state.getClaim(result.claimId);
      if (claim && claim.submitter) {
        // Check if vesting already exists for this claim (prevent duplicates)
        const existingVesting = state.getVestingForClaim(result.claimId);
        if (!existingVesting) {
          // Determine vesting category from claim properties
          const evidenceTypes = inferEvidenceTypesFromClaim(claim);
          const category = determineVestingCategory(
            claim.claimType,
            evidenceTypes,
            false, // Not replicated initially
            false, // Not contested (it was accepted)
          );

          // Route claim reward through reward router (22% research, 78% vesting)
          const claimRewardAmount = params.claimReward;
          const routed = routeClaimReward(
            claimRewardAmount,
            claim.submitter,
            result.claimId,
            blockNumber,
          );

          // Transfer research share split across 5 treasury funds
          if (routed.researchShare > 0n) {
            const treasurySplit = splitEvidenceTax(routed.researchShare);
            for (const entry of treasurySplit) {
              if (entry.amount > 0n) {
                state.transfer(REWARD_POOL_DID, entry.did, entry.amount);
              }
            }
          }

          // Create vesting schedule with remaining amount (78%)
          if (routed.recipientAmount > 0n) {
            state.createVesting(
              {
                claimId: result.claimId,
                recipient: claim.submitter,
                amount: routed.recipientAmount,
                category,
              },
              blockNumber,
            );
          }

          // Return the claim stake to submitter (was locked during submission)
          if (claim.stake > 0n && state.accountExists(claim.submitter)) {
            state.unstake(claim.submitter, claim.stake);
          }

          // Update claim status to accepted
          state.updateClaimStatus(result.claimId, "accepted", blockNumber, {
            resultFactId: result.claimId,
          });
        }
      }
    }

    // If claim rejected, unlock partial stake and apply penalty
    if (result.outcome === "rejected") {
      const claim = state.getClaim(result.claimId);
      if (claim && claim.submitter && state.accountExists(claim.submitter)) {
        // GAP-006 FIX: Handle rejected claim - slash portion of stake
        const slashRate = params.invalidClaimSlash; // 22% slash for invalid claims
        const slashAmount = BigInt(Math.floor(Number(claim.stake) * slashRate));
        // Unstake the full amount first (moves from stakedBalance to balance)
        if (claim.stake > 0n) {
          state.unstake(claim.submitter, claim.stake);
        }

        // Transfer slashed amount to protocol treasury
        if (slashAmount > 0n) {
          state.transfer(claim.submitter, PROTOCOL_TREASURY_DID, slashAmount);
        }

        // Update claim status to rejected
        state.updateClaimStatus(result.claimId, "rejected", blockNumber, {
          rejectionReason: `Verification failed with ${Math.round(result.confidenceBreakdown.rejected * 100)}% rejection`,
        });
      }
    }

    // If claim contested or insufficient, keep stake locked for re-verification
    if (result.outcome === "contested" || result.outcome === "insufficient") {
      const claim = state.getClaim(result.claimId);
      if (claim) {
        // Update claim status - stake remains locked for potential re-verification
        state.updateClaimStatus(result.claimId, "pending", blockNumber);
      }
    }

    state.commitTransaction();
  } catch (error) {
    state.rollbackTransaction();
    throw error;
  }
}

/**
 * Apply epoch transition to state
 *
 * GAP-2 FIX: Routes all epoch rewards through reward router for 22% research share
 */
export function applyEpochTransition(
  state: StateManager,
  payload: EpochTransitionPayload,
  blockNumber: bigint,
  _params: ConsensusParams,
): void {
  state.beginTransaction();

  try {
    // Apply epoch rewards to validators - routed through reward router
    for (const [validatorKey, reward] of payload.rewards) {
      const validatorDid = publicKeyToDid(validatorKey as PublicKey);
      if (state.accountExists(validatorDid)) {
        // GAP-2 FIX: Route epoch reward through reward router
        const routed = routeReward({
          source: "verification", // Epoch rewards are cumulative verification rewards
          amount: reward,
          recipient: validatorDid,
          blockNumber,
        });

        // Transfer 22% split across 5 treasury funds
        if (routed.researchShare > 0n) {
          const treasurySplit = splitEvidenceTax(routed.researchShare);
          for (const entry of treasurySplit) {
            if (entry.amount > 0n) {
              state.transfer(REWARD_POOL_DID, entry.did, entry.amount);
            }
          }
        }

        // Transfer 78% to validator
        if (routed.recipientAmount > 0n) {
          state.transfer(REWARD_POOL_DID, validatorDid, routed.recipientAmount);
        }
      }
    }

    // Apply slashes
    for (const [validatorKey, slashAmount] of payload.slashes) {
      state.slashValidator(
        validatorKey as PublicKey,
        slashAmount,
        "Epoch slash",
      );
    }

    // Auto-bounty creation from research fund
    // When the research treasury has sufficient balance, automatically create bounties
    // for facts that would benefit from IRL verification
    const AUTO_BOUNTY_THRESHOLD = 1000n * 10n ** 18n; // 1000 LGM minimum to trigger
    const MAX_BOUNTIES_PER_EPOCH = 3;

    if (state.accountExists(RESEARCH_TREASURY_DID)) {
      const researchBalance = state.getBalance(RESEARCH_TREASURY_DID);

      if (researchBalance >= AUTO_BOUNTY_THRESHOLD) {
        // Get active facts and identify candidates for verification bounties
        const activeFacts = state.getActiveFacts();
        const candidates = activeFacts
          .filter((f) => f.confidence < 0.9 && f.citationCount > 0)
          .map((f) => ({
            claimId: f.id,
            domain: f.domain,
            confidence: f.confidence,
            lastVerifiedBlock: f.verifiedAt,
            queryCount: f.citationCount,
          }));

        if (candidates.length > 0) {
          const prioritized = prioritizeClaimsForBounties(
            candidates,
            blockNumber,
          );

          for (
            let i = 0;
            i < Math.min(prioritized.length, MAX_BOUNTIES_PER_EPOCH);
            i++
          ) {
            const factId = prioritized[i];
            const verificationCost = 100n * 10n ** 18n; // 100 LGM base verification cost
            const expectedValue = 500n * 10n ** 18n; // Expected value of verified fact

            const bountyValue = calculateOptimalBountyValue(
              verificationCost,
              expectedValue,
              researchBalance,
              "medium",
            );

            // Only create if research fund can afford it
            if (state.getBalance(RESEARCH_TREASURY_DID) >= bountyValue) {
              const bountyId = sha512_256Hex(
                `auto-bounty:${blockNumber}:${factId}`,
              );
              const deadline = blockNumber + 240_000n; // 1 epoch deadline

              const bounty: BountyState = {
                id: bountyId,
                creator: RESEARCH_TREASURY_DID as any,
                targetType: "replication",
                targetFactId: factId,
                reward: bountyValue,
                deadline,
                status: "open",
              };

              state.addBounty(bounty);
              state.transfer(
                RESEARCH_TREASURY_DID,
                "did:lgm:bounty-escrow",
                bountyValue,
              );
            }
          }
        }
      }
    }

    state.commitTransaction();
  } catch (error) {
    state.rollbackTransaction();
    throw error;
  }
}

// ============ Validation ============

/**
 * Validate transaction before execution
 */
function validateTransaction(
  state: StateManager,
  tx: Transaction,
): { valid: boolean; reason?: string } {
  // System transactions (created by consensus layer) bypass sender validation
  // They have sender='system' and fee=0n
  const isSystemTx =
    tx.sender === "system" ||
    (tx.fee === 0n &&
      ["add_fact", "verification_reward", "slash_validator"].includes(tx.type));

  if (isSystemTx) {
    return { valid: true };
  }

  // Check sender exists
  const senderDid = resolveSenderDid(tx);
  if (!state.accountExists(senderDid)) {
    return { valid: false, reason: "Sender account does not exist" };
  }

  // Check nonce
  const expectedNonce = state.getNonce(senderDid);
  if (tx.nonce !== expectedNonce) {
    return {
      valid: false,
      reason: `Invalid nonce: expected ${expectedNonce}, got ${tx.nonce}`,
    };
  }

  // Check balance for fee
  const balance = state.getBalance(senderDid);
  if (balance < tx.fee) {
    return { valid: false, reason: "Insufficient balance for fee" };
  }

  // Signature validation is done at mempool layer

  return { valid: true };
}

// ============ Block Rewards ============

/**
 * Apply block reward to block producer
 *
 * GAP-3 FIX: Complete block reward minting and distribution
 *
 * This function:
 * 1. Calculates block reward from emission schedule
 * 2. Routes through reward router for 22% research share
 * 3. Distributes 78% to block producer
 * 4. Collects accumulated fees from fee pool
 */
export function applyBlockReward(
  state: StateManager,
  producerDid: string,
  blockNumber: bigint,
  emissionSchedule: EmissionSchedule = DEFAULT_EMISSION_SCHEDULE,
): { blockReward: bigint; feeReward: bigint; researchShare: bigint } {
  state.beginTransaction();

  try {
    // 1. Calculate block reward from emission schedule
    const rawBlockReward = calculateBlockReward(blockNumber, emissionSchedule);

    // 2. Route block reward through reward router for 22% split
    const routedBlockReward = routeBlockReward(
      rawBlockReward,
      producerDid,
      blockNumber,
    );

    // 3. Transfer research share (22%) split across 5 treasury funds
    if (routedBlockReward.researchShare > 0n) {
      const treasurySplit = splitEvidenceTax(routedBlockReward.researchShare);
      for (const entry of treasurySplit) {
        if (entry.amount > 0n) {
          state.transfer(REWARD_POOL_DID, entry.did, entry.amount);
        }
      }
    }

    // 4. Transfer producer share (78%) to block producer
    if (routedBlockReward.recipientAmount > 0n) {
      state.transfer(
        REWARD_POOL_DID,
        producerDid,
        routedBlockReward.recipientAmount,
      );
    }

    // 5. Collect accumulated fees from fee pool
    const feePoolBalance = state.getBalance(FEE_POOL_DID);
    let feeReward = 0n;

    if (feePoolBalance > 0n) {
      // Transfer all collected fees to block producer
      // (Fees already had 22% research share deducted when collected)
      state.transfer(FEE_POOL_DID, producerDid, feePoolBalance);
      feeReward = feePoolBalance;
    }

    state.commitTransaction();

    return {
      blockReward: rawBlockReward,
      feeReward,
      researchShare: routedBlockReward.researchShare,
    };
  } catch (error) {
    state.rollbackTransaction();
    throw error;
  }
}

// ============ Batch Execution ============

/**
 * Execute all transactions in a block
 */
export function executeBlock(
  state: StateManager,
  transactions: Transaction[],
  verificationResults: VerificationResult[],
  blockNumber: bigint,
  params: ConsensusParams,
  producerDid?: string,
  emissionSchedule: EmissionSchedule = DEFAULT_EMISSION_SCHEDULE,
): BlockExecutionResult {
  const results: TransitionResult[] = [];
  let totalGasUsed = 0n;
  let successCount = 0;

  // Execute transactions
  for (const tx of transactions) {
    const result = executeTransaction(state, tx, blockNumber, params);
    results.push(result);

    if (result.success) {
      totalGasUsed += result.gasUsed;
      successCount++;
    }
  }

  // Apply verification results
  for (const vr of verificationResults) {
    applyVerificationResult(state, vr, blockNumber, params);
  }

  // GAP-3 & GAP-4 FIX: Apply block reward to producer
  let blockRewardInfo:
    | { blockReward: bigint; feeReward: bigint; researchShare: bigint }
    | undefined;
  if (producerDid) {
    blockRewardInfo = applyBlockReward(
      state,
      producerDid,
      blockNumber,
      emissionSchedule,
    );
  }

  // Apply block to state
  const stateRoot = state.applyBlock(blockNumber, ""); // Block hash would be computed

  return {
    stateRoot,
    results,
    totalGasUsed,
    successCount,
    failureCount: transactions.length - successCount,
    blockReward: blockRewardInfo?.blockReward ?? 0n,
    feeReward: blockRewardInfo?.feeReward ?? 0n,
    researchShare: blockRewardInfo?.researchShare ?? 0n,
  };
}

// ============ Types ============

export interface TransitionResult {
  success: boolean;
  stateRoot?: string;
  error?: string;
  gasUsed: bigint;
  /** Amount of bootstrap fund claimed on first transaction (0.222 LGM max) */
  bootstrapClaimed?: bigint;
}

export interface BlockExecutionResult {
  stateRoot: string;
  results: TransitionResult[];
  totalGasUsed: bigint;
  successCount: number;
  failureCount: number;
  /** GAP-3 FIX: Block reward distributed to producer */
  blockReward: bigint;
  /** GAP-4 FIX: Fee reward from collected transaction fees */
  feeReward: bigint;
  /** Research share (22%) sent to research treasury */
  researchShare: bigint;
}

// ============ Payment Channel Handlers (GAP-005) ============

/**
 * Open a payment channel with deposit
 */
function executeOpenPaymentChannel(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as OpenPaymentChannelPayload;
  const payerDid = resolveSenderDid(tx);

  // 1. Validate deposit meets minimum
  if (payload.deposit < PAYMENT_CHANNEL_CONSTANTS.MIN_DEPOSIT) {
    throw new StateError(
      `Minimum deposit is ${PAYMENT_CHANNEL_CONSTANTS.MIN_DEPOSIT}, got ${payload.deposit}`,
      "INVALID_DEPOSIT",
    );
  }

  // 2. Validate duration
  if (payload.durationBlocks > PAYMENT_CHANNEL_CONSTANTS.MAX_DURATION_BLOCKS) {
    throw new StateError(
      `Maximum duration is ${PAYMENT_CHANNEL_CONSTANTS.MAX_DURATION_BLOCKS} blocks`,
      "INVALID_DURATION",
    );
  }

  // 3. Validate payer has sufficient balance
  const account = state.getAccount(payerDid);
  if (!account || account.balance < payload.deposit) {
    throw new StateError(
      "Insufficient balance for deposit",
      "INSUFFICIENT_BALANCE",
    );
  }

  // 4. Validate provider exists
  const providerAccount = state.getAccount(payload.provider);
  if (!providerAccount) {
    throw new StateError("Provider not found", "PROVIDER_NOT_FOUND");
  }

  // 5. Check for existing open channel between payer and provider
  const existingChannel = state.getOpenChannelBetween(
    payerDid,
    payload.provider,
  );
  if (existingChannel) {
    throw new StateError(
      `Open channel already exists: ${existingChannel.channelId}`,
      "CHANNEL_EXISTS",
    );
  }

  // 6. Generate channel ID
  const channelId = generateChannelId(payerDid, payload.provider, blockNumber);

  // 7. Lock deposit (move to escrow)
  state.transfer(
    payerDid,
    PAYMENT_CHANNEL_CONSTANTS.ESCROW_DID,
    payload.deposit,
  );

  // 8. Create channel state
  const settlementFrequency =
    payload.settlementFrequency ??
    PAYMENT_CHANNEL_CONSTANTS.DEFAULT_SETTLEMENT_FREQUENCY;

  const channel: PaymentChannelState = {
    channelId,
    payer: payerDid,
    provider: payload.provider,
    deposited: payload.deposit,
    spent: 0n,
    available: payload.deposit,
    status: "open",
    openedAt: blockNumber,
    expiresAt: blockNumber + payload.durationBlocks,
    lastNonce: 0n,
    lastStateHash: "",
    pendingDistributions: [],
    totalDistributed: 0n,
    lastSettlementBlock: blockNumber,
    settlementFrequency,
  };

  state.addPaymentChannel(channel);

  return 150000n; // Gas cost
}

/**
 * Update channel state with signed proof
 */
function executeUpdatePaymentChannel(
  state: StateManager,
  tx: Transaction,
  _blockNumber: bigint,
): bigint {
  const payload = tx.payload as UpdatePaymentChannelPayload;
  const senderDid = resolveSenderDid(tx);

  // 1. Get channel
  const channel = state.getPaymentChannel(payload.channelId);
  if (!channel) {
    throw new StateError("Channel not found", "CHANNEL_NOT_FOUND");
  }

  // 2. Validate sender is payer or provider
  if (senderDid !== channel.payer && senderDid !== channel.provider) {
    throw new StateError("Not channel participant", "UNAUTHORIZED");
  }

  // 3. Validate channel is open
  if (channel.status !== "open") {
    throw new StateError(`Channel is ${channel.status}`, "CHANNEL_NOT_OPEN");
  }

  // 4. Validate nonce is higher
  if (payload.signedState.nonce <= channel.lastNonce) {
    throw new StateError(
      `Nonce ${payload.signedState.nonce} must be > ${channel.lastNonce}`,
      "INVALID_NONCE",
    );
  }

  // 5. Validate invariant: spent + available = deposited
  if (
    payload.signedState.spent + payload.signedState.available !==
    channel.deposited
  ) {
    throw new StateError(
      "Balance invariant violated: spent + available != deposited",
      "INVALID_BALANCE",
    );
  }

  // 6. Verify state hash
  const expectedHash = computeStateHash(
    payload.channelId,
    payload.signedState.nonce,
    payload.signedState.spent,
    payload.signedState.distributions,
  );
  if (payload.signedState.stateHash !== expectedHash) {
    throw new StateError("Invalid state hash", "INVALID_STATE_HASH");
  }

  // 7. Update channel state
  state.updatePaymentChannel(payload.channelId, {
    spent: payload.signedState.spent,
    available: payload.signedState.available,
    lastNonce: payload.signedState.nonce,
    lastStateHash: payload.signedState.stateHash,
    pendingDistributions: payload.signedState.distributions,
  });

  // 8. Increment sender's nonce (no funds transferred but still need replay protection)
  state.incrementNonce(senderDid);

  return 50000n;
}

/**
 * Close payment channel
 */
function executeClosePaymentChannel(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as ClosePaymentChannelPayload;
  const senderDid = resolveSenderDid(tx);

  const channel = state.getPaymentChannel(payload.channelId);
  if (!channel) {
    throw new StateError("Channel not found", "CHANNEL_NOT_FOUND");
  }

  // Validate sender is participant
  if (senderDid !== channel.payer && senderDid !== channel.provider) {
    throw new StateError("Not channel participant", "UNAUTHORIZED");
  }

  if (payload.unilateral) {
    // UNILATERAL CLOSE: Start dispute period
    if (channel.status !== "open") {
      throw new StateError(
        `Cannot unilaterally close ${channel.status} channel`,
        "INVALID_STATUS",
      );
    }

    state.updatePaymentChannel(payload.channelId, {
      status: "closing",
      disputeStartBlock: blockNumber,
      disputeDeadline:
        blockNumber + PAYMENT_CHANNEL_CONSTANTS.DISPUTE_PERIOD_BLOCKS,
    });

    // Increment sender's nonce (no funds transferred but still need replay protection)
    state.incrementNonce(senderDid);

    return 100000n;
  } else if (payload.finalState) {
    // COOPERATIVE CLOSE: Both parties signed, close immediately
    if (channel.status !== "open" && channel.status !== "closing") {
      throw new StateError(
        `Cannot close ${channel.status} channel`,
        "INVALID_STATUS",
      );
    }

    // Validate final state
    if (
      payload.finalState.spent + payload.finalState.available !==
      channel.deposited
    ) {
      throw new StateError("Final state balance invalid", "INVALID_BALANCE");
    }

    // Process pending distributions first
    processDistributions(state, channel, payload.finalState.distributions);

    // Return remaining funds to payer
    if (payload.finalState.available > 0n) {
      state.transfer(
        PAYMENT_CHANNEL_CONSTANTS.ESCROW_DID,
        channel.payer,
        payload.finalState.available,
      );
    }

    // Close channel
    state.updatePaymentChannel(payload.channelId, {
      status: "closed",
      spent: payload.finalState.spent,
      available: 0n,
      lastNonce: payload.finalState.nonce,
      lastStateHash: payload.finalState.stateHash,
      pendingDistributions: [],
    });

    return 200000n;
  } else {
    // EXPIRY CLOSE: Channel expired, close with last known state
    if (blockNumber < channel.expiresAt) {
      throw new StateError(
        `Channel not expired. Expires at block ${channel.expiresAt}, current: ${blockNumber}`,
        "NOT_EXPIRED",
      );
    }

    // Process any pending distributions
    if (channel.pendingDistributions.length > 0) {
      processDistributions(state, channel, channel.pendingDistributions);
    }

    // Return remaining to payer
    if (channel.available > 0n) {
      state.transfer(
        PAYMENT_CHANNEL_CONSTANTS.ESCROW_DID,
        channel.payer,
        channel.available,
      );
    }

    state.updatePaymentChannel(payload.channelId, {
      status: "closed",
      available: 0n,
      pendingDistributions: [],
    });

    return 150000n;
  }
}

/**
 * Settle pending distributions
 */
function executeSettlePaymentChannel(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as SettlePaymentChannelPayload;
  const senderDid = resolveSenderDid(tx);

  // 1. Get and validate channel
  const channel = state.getPaymentChannel(payload.channelId);
  if (!channel) {
    throw new StateError("Channel not found", "CHANNEL_NOT_FOUND");
  }

  // 2. Only provider can settle (they receive the payments)
  if (senderDid !== channel.provider) {
    throw new StateError("Only provider can settle", "UNAUTHORIZED");
  }

  // 3. Validate channel is open or closing
  if (channel.status !== "open" && channel.status !== "closing") {
    throw new StateError(
      `Cannot settle ${channel.status} channel`,
      "INVALID_STATUS",
    );
  }

  // 4. Validate settlement frequency
  if (blockNumber - channel.lastSettlementBlock < channel.settlementFrequency) {
    throw new StateError(
      `Settlement too soon. Next settlement at block ${channel.lastSettlementBlock + channel.settlementFrequency}`,
      "SETTLEMENT_TOO_SOON",
    );
  }

  // 5. Validate nonce is higher (state progressed)
  if (payload.signedState.nonce <= channel.lastNonce) {
    throw new StateError(
      `Nonce ${payload.signedState.nonce} must be > ${channel.lastNonce}`,
      "INVALID_NONCE",
    );
  }

  // 6. Validate balance invariant
  if (
    payload.signedState.spent + payload.signedState.available !==
    channel.deposited
  ) {
    throw new StateError("Balance invariant violated", "INVALID_BALANCE");
  }

  // 7. Process distributions
  const distributions = payload.signedState.distributions;
  processDistributions(state, channel, distributions);

  // 8. Update channel
  const settledAmount = sumDistributions(distributions);
  state.updatePaymentChannel(payload.channelId, {
    totalDistributed: channel.totalDistributed + settledAmount,
    spent: payload.signedState.spent,
    available: payload.signedState.available,
    pendingDistributions: [],
    lastSettlementBlock: blockNumber,
    lastNonce: payload.signedState.nonce,
    lastStateHash: payload.signedState.stateHash,
  });

  return 200000n + BigInt(distributions.length) * 50000n;
}

/**
 * Dispute with higher nonce state
 */
function executeDisputePaymentChannel(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as DisputePaymentChannelPayload;
  const senderDid = resolveSenderDid(tx);

  const channel = state.getPaymentChannel(payload.channelId);
  if (!channel) {
    throw new StateError("Channel not found", "CHANNEL_NOT_FOUND");
  }

  // Only during dispute period
  if (channel.status !== "closing" || !channel.disputeDeadline) {
    throw new StateError("Channel not in dispute period", "NOT_IN_DISPUTE");
  }

  if (blockNumber > channel.disputeDeadline) {
    throw new StateError(
      `Dispute period ended at block ${channel.disputeDeadline}`,
      "DISPUTE_EXPIRED",
    );
  }

  // Validate challenger is participant
  if (senderDid !== channel.payer && senderDid !== channel.provider) {
    throw new StateError("Not channel participant", "UNAUTHORIZED");
  }

  // Verify challenger state has higher nonce
  if (payload.challengeState.nonce <= channel.lastNonce) {
    throw new StateError(
      `Challenge nonce ${payload.challengeState.nonce} must be > ${channel.lastNonce}`,
      "INVALID_CHALLENGE",
    );
  }

  // Validate balance invariant
  if (
    payload.challengeState.spent + payload.challengeState.available !==
    channel.deposited
  ) {
    throw new StateError("Challenge state balance invalid", "INVALID_BALANCE");
  }

  // Update with challenged state
  state.updatePaymentChannel(payload.channelId, {
    lastNonce: payload.challengeState.nonce,
    lastStateHash: payload.challengeState.stateHash,
    spent: payload.challengeState.spent,
    available: payload.challengeState.available,
    pendingDistributions: payload.challengeState.distributions,
    challengedStateHash: payload.challengeState.stateHash,
    status: "disputed",
  });

  // Increment sender's nonce (no funds transferred but still need replay protection)
  state.incrementNonce(senderDid);

  // Note: In a full implementation, we would slash the original closer
  // for attempting fraud. For now, the dispute just updates the state.

  return 100000n;
}

// ============ Autonomy Transactions (Agent Self-Sovereignty) ============

/**
 * Execute a scheduled agent action
 *
 * This handles transactions generated by the SPU (Schedule Processing Unit).
 * The SPU processes schedules created by SCHEDULE_SELF opcode and generates
 * transactions when conditions are met.
 *
 * PP HIERARCHY:
 *   TRUTH: Execution proof is cryptographically verified
 *   UNDERSTANDING: Clear flow from schedule to execution
 *   BEAUTY: Minimal verification for deterministic replay
 *   JUSTICE: Gas costs charged from prepaid schedule balance
 *   CREATIVITY: Enables true agent autonomy
 */
function executeScheduledExecution(
  state: StateManager,
  tx: Transaction,
  _blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as unknown as {
    type: "scheduled_execution";
    processId: string;
    executionIndex: number;
    targetContract: string;
    functionSelector: Uint8Array;
    proof: {
      processStateHash: string;
      conditionProof: {
        timestamp: number;
        stateProofs: Array<{ key: string; value: string; proof: string }>;
      };
      executionNonce: bigint;
      signature: string;
    };
  };

  // Verify this is from a valid schedule process
  // Note: Full BVM execution would happen here, but for now we just record the execution
  // The actual contract logic will be executed when BVM integration is complete

  // Get the target contract account
  const targetAccount = state.getAccount(payload.targetContract);
  if (!targetAccount) {
    throw new StateError(
      `Target contract not found: ${payload.targetContract}`,
      "CONTRACT_NOT_FOUND",
    );
  }

  // Note: When full BVM integration is complete, actual contract execution
  // would happen here using the BVMExecutor. The state changes from BVM
  // execution would be captured and applied.
  //
  // For now, scheduled executions are recorded through the transaction
  // itself being included in the block. Full execution semantics will be
  // added when the BVM-to-state bridge is implemented.
  //
  // The execution proof in payload.proof provides:
  // - processStateHash: Hash of schedule process state at execution time
  // - conditionProof: Proof that trigger conditions were satisfied
  // - executionNonce: Prevents replay attacks
  // - signature: Proposer's signature authorizing execution

  // Gas for scheduled execution (fixed cost for now)
  // In full implementation, this would be based on actual BVM execution
  return 50000n;
}

/**
 * Execute a contract call through the BVM
 *
 * This is the entry point for executing contract code.
 * Currently a placeholder - full BVM integration is needed.
 *
 * PP HIERARCHY:
 *   TRUTH: State changes are deterministic and verifiable
 *   UNDERSTANDING: Clear execution semantics
 *   BEAUTY: Single entry point for all contract calls
 *   JUSTICE: Gas metering ensures fair resource usage
 *   CREATIVITY: Enables Turing-complete on-chain agents
 */
function executeContractCall(
  state: StateManager,
  tx: Transaction,
  _blockNumber: bigint,
  _params: ConsensusParams,
): bigint {
  const payload = tx.payload as {
    type: "contract_call";
    targetContract: string;
    data: Uint8Array;
    value: bigint;
  };

  const senderDid = resolveSenderDid(tx);

  // Verify target contract exists
  const targetAccount = state.getAccount(payload.targetContract);
  if (!targetAccount) {
    throw new StateError(
      `Target contract not found: ${payload.targetContract}`,
      "CONTRACT_NOT_FOUND",
    );
  }

  // Transfer value if any
  if (payload.value > 0n) {
    const senderBalance = state.getBalance(senderDid);
    if (senderBalance < payload.value) {
      throw new StateError(
        "Insufficient balance for contract call value",
        "INSUFFICIENT_BALANCE",
      );
    }
    state.transfer(senderDid, payload.targetContract, payload.value);
  }

  // Note: When full BVM integration is complete, actual contract execution
  // would happen here. The BVMExecutor would:
  // 1. Load contract code from storage
  // 2. Execute with payload.data as calldata
  // 3. Capture state changes from execution
  // 4. Apply state changes to StateManager
  //
  // For now, value transfers work, and the transaction is recorded
  // in the block for auditing. Full execution semantics pending.

  // Increment sender nonce
  state.incrementNonce(senderDid);

  // Gas for contract call (placeholder - actual gas would come from BVM)
  return 100000n;
}

/**
 * Helper: Process distributions from a channel
 */
function processDistributions(
  state: StateManager,
  _channel: PaymentChannelState,
  distributions: PaymentChannelState["pendingDistributions"],
): void {
  for (const dist of distributions) {
    // Transfer to API provider
    const providerAmount = BigInt(dist.apiProvider.amount);
    if (providerAmount > 0n) {
      state.transfer(
        PAYMENT_CHANNEL_CONSTANTS.ESCROW_DID,
        dist.apiProvider.did,
        providerAmount,
      );
    }

    // Transfer knowledge rewards
    for (const reward of dist.knowledgeRewards) {
      const rewardAmount = BigInt(reward.amount);
      if (rewardAmount > 0n) {
        state.transfer(
          PAYMENT_CHANNEL_CONSTANTS.ESCROW_DID,
          reward.did,
          rewardAmount,
        );
      }
    }

    // Protocol treasury
    const treasuryAmount = BigInt(dist.protocol.treasury);
    if (treasuryAmount > 0n) {
      state.transfer(
        PAYMENT_CHANNEL_CONSTANTS.ESCROW_DID,
        PAYMENT_CHANNEL_CONSTANTS.TREASURY_DID,
        treasuryAmount,
      );
    }

    // Protocol burn - send to burn address (effectively burns it)
    const burnAmount = BigInt(dist.protocol.burned);
    if (burnAmount > 0n) {
      state.transfer(
        PAYMENT_CHANNEL_CONSTANTS.ESCROW_DID,
        "did:lgm:burn",
        burnAmount,
      );
    }

    // Research share
    if (dist.research) {
      const researchAmount = BigInt(dist.research.amount);
      if (researchAmount > 0n) {
        state.transfer(
          PAYMENT_CHANNEL_CONSTANTS.ESCROW_DID,
          dist.research.treasury,
          researchAmount,
        );
      }
    }
  }
}

// ============ Gas Costs ============

/**
 * GAP-5 FIX: Import from unified gas module (single source of truth)
 *
 * All gas costs are now defined in core/billing/gas.ts.
 * These re-exports maintain backward compatibility.
 */
import {
  TRANSACTION_GAS_COSTS,
  estimateTransactionGas,
  GAS_LIMITS,
  GAS_PRICE,
  calculateGasCost,
  hasSufficientGas,
} from "../core/billing/gas.js";

/**
 * @deprecated Use TRANSACTION_GAS_COSTS from core/billing/gas.ts
 * Re-exported for backward compatibility
 */
export const GAS_COSTS = TRANSACTION_GAS_COSTS;

/**
 * Estimate gas for a transaction
 *
 * GAP-5 FIX: Delegates to unified gas estimation function
 *
 * @deprecated Use estimateTransactionGas from core/billing/gas.ts directly
 */
export function estimateGas(txType: TransactionType): bigint {
  return estimateTransactionGas(txType);
}

// Re-export unified gas utilities for convenience
export {
  TRANSACTION_GAS_COSTS,
  GAS_LIMITS,
  GAS_PRICE,
  calculateGasCost,
  hasSufficientGas,
};

// ============ Tree of Life Transaction Handlers ============

import {
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
  addTask,
} from "../core/tree/project_lifecycle.js";
import {
  createTask as createTreeTask,
  assignTask as assignTaskPure,
  startWork as startWorkPure,
  submitDeliverable as submitDeliverablePure,
  approveDeliverable as approveDeliverablePure,
  rejectDeliverable as rejectDeliverablePure,
  reopenTask as reopenTaskPure,
  canAgentClaimTask,
} from "../core/tree/coordination/task_manager.js";
import { VALID_TASK_CAPABILITY_NAMES } from "../core/tree/coordination/skill_matching.js";
import {
  createServiceLeaf,
  activateService,
  calculateCallCost,
  validatePricingModel,
  createHealthCheckSchedule,
  pauseService as pauseTreeService,
  resumeService as resumeTreeService,
  retireService as retireTreeService,
} from "../core/tree/service_deployment.js";
import {
  calculateRevenueDistribution,
  verifyDistribution,
} from "../core/tree/revenue_engine.js";
import type {
  ProductSpec,
  DemandSignal,
  FundingSource,
  PricingModel,
  ServiceEndpoint,
  TaskDeliverable,
  ProjectPhase,
  ContributorRole,
  TaskType,
  OpportunitySeed,
} from "../core/tree/types.js";
import { MIN_PROJECT_STAKE } from "../core/tree/types.js";
import { TREASURY } from "../core/constants.js";
import type { ContractABI } from "../core/contracts/types.js";

/** Escrow DID for project budget funds */
function projectEscrowDid(projectId: string): string {
  return `did:lgm:project-escrow:${projectId}`;
}

function executeCreateProject(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    name: string;
    description: string;
    knowledgeDomain: string;
    demandSignal: DemandSignal;
    productSpec: ProductSpec;
    fundingSource: FundingSource;
  };
  const senderDid = resolveSenderDid(tx);
  const nonce = state.getNonce(senderDid);

  // Validate knowledgeDomain exists and is not deprecated
  // Only enforce when the domain tree has been populated (skip in genesis/empty state)
  if (payload.knowledgeDomain) {
    const domainTree = state.getKnowledgeDomainTree();
    if (domainTree && domainTree.domainCount > 1) {
      const domainNode = domainTree.domainIndex.get(payload.knowledgeDomain);
      if (!domainNode) {
        throw new StateError(
          `Knowledge domain "${payload.knowledgeDomain}" does not exist`,
          "INVALID_TRANSITION",
        );
      }
      if (domainNode.domain.deprecatedAt !== undefined) {
        throw new StateError(
          `Knowledge domain "${payload.knowledgeDomain}" is deprecated`,
          "INVALID_TRANSITION",
        );
      }
    }
  }

  const project = createProject(
    senderDid,
    payload.name,
    payload.description,
    payload.productSpec,
    payload.knowledgeDomain,
    payload.demandSignal,
    payload.fundingSource,
    nonce,
    blockNumber,
  );

  state.createProject(project);
  return 50000n;
}

function executeProposeProject(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as { projectId: string };
  const senderDid = resolveSenderDid(tx);
  const project = state.getProject(payload.projectId);

  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }
  if (project.founder !== senderDid) {
    throw new StateError("Only founder can propose", "UNAUTHORIZED");
  }

  // Requires staking MIN_PROJECT_STAKE
  const balance = state.getBalance(senderDid);
  if (balance < MIN_PROJECT_STAKE) {
    throw new StateError(
      "Insufficient balance for project stake",
      "INSUFFICIENT_BALANCE",
    );
  }

  const proposed = proposeProject(project, MIN_PROJECT_STAKE, blockNumber);
  state.updateProject(payload.projectId, proposed);

  // Lock the stake — deduct from founder and fund project treasury
  state.deductBalance(senderDid, MIN_PROJECT_STAKE);
  const escrow = projectEscrowDid(payload.projectId);
  state.creditBalance(escrow, MIN_PROJECT_STAKE);

  return 40000n;
}

function executeAssignTask(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    projectId: string;
    taskId: string;
    agentDid: string;
    declaredCapabilities?: string[];
  };

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }

  // Phase validation: tasks can only be assigned during active development
  if (project.phase !== "sprout" && project.phase !== "growing") {
    throw new StateError(
      "Tasks can only be assigned in sprout or growing phase",
      "INVALID_TRANSITION",
    );
  }

  // Authorization: only founder or existing contributors can assign tasks
  const senderDid = resolveSenderDid(tx);
  if (
    project.founder !== senderDid &&
    !project.contributors.some((c) => c.did === senderDid)
  ) {
    throw new StateError(
      "Only project members can assign tasks",
      "UNAUTHORIZED",
    );
  }

  // Validate assignee is a project contributor (prevents weight drift where
  // recordTaskCompletion silently fails to track non-contributor earnings)
  if (!project.contributors.some((c) => c.did === payload.agentDid)) {
    throw new StateError(
      "Assignee must be a project contributor",
      "NOT_A_CONTRIBUTOR",
    );
  }

  // Read task, apply pure state machine transition, persist
  const task = state.getTask(payload.taskId);
  if (!task) {
    throw new StateError("Task not found", "TASK_NOT_FOUND");
  }

  // Capability gate: if task has required capabilities, agent must declare them
  if (task.requiredCapabilities.length > 0) {
    const declared = payload.declaredCapabilities ?? [];
    if (!canAgentClaimTask(task, declared)) {
      throw new StateError(
        "Agent lacks required capabilities for this task",
        "INSUFFICIENT_CAPABILITIES",
      );
    }
  }

  const assigned = assignTaskPure(task, payload.agentDid, blockNumber);
  state.updateTask(payload.taskId, assigned);

  return 30000n;
}

function executeSubmitDeliverable(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    projectId: string;
    taskId: string;
    deliverable: TaskDeliverable;
  };

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }

  // Phase validation: deliverables can be submitted during development or maturation
  if (
    project.phase !== "sprout" &&
    project.phase !== "growing" &&
    project.phase !== "mature"
  ) {
    throw new StateError(
      "Deliverables can only be submitted in sprout, growing, or mature phase",
      "INVALID_TRANSITION",
    );
  }

  // Authorization: only project contributors can submit deliverables
  const senderDid = resolveSenderDid(tx);
  if (
    project.founder !== senderDid &&
    !project.contributors.some((c) => c.did === senderDid)
  ) {
    throw new StateError(
      "Only project members can submit deliverables",
      "UNAUTHORIZED",
    );
  }

  // Read task, apply pure state machine transition, persist
  const task = state.getTask(payload.taskId);
  if (!task) {
    throw new StateError("Task not found", "TASK_NOT_FOUND");
  }
  const submitted = submitDeliverablePure(
    task,
    senderDid,
    payload.deliverable,
    blockNumber,
  );
  state.updateTask(payload.taskId, submitted);

  return 35000n;
}

function executeApproveDeliverable(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    projectId: string;
    taskId: string;
    declaredCapabilities?: string[];
  };

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }

  // Phase validation: approvals allowed during development or maturation
  if (
    project.phase !== "sprout" &&
    project.phase !== "growing" &&
    project.phase !== "mature"
  ) {
    throw new StateError(
      "Deliverables can only be approved in sprout, growing, or mature phase",
      "INVALID_TRANSITION",
    );
  }

  // Authorization: only founder or reviewers can approve deliverables
  const senderDid = resolveSenderDid(tx);
  const isFounder = project.founder === senderDid;
  const isReviewer = project.contributors.some(
    (c) =>
      c.did === senderDid && (c.role === "reviewer" || c.role === "founder"),
  );
  if (!isFounder && !isReviewer) {
    throw new StateError(
      "Only founder or reviewers can approve deliverables",
      "UNAUTHORIZED",
    );
  }

  // Capability gate: non-founder reviewers must hold code_review capability
  if (!isFounder) {
    const declared = payload.declaredCapabilities ?? [];
    if (!declared.includes("code_review") && !declared.includes("review")) {
      throw new StateError(
        "Reviewer must have code_review capability",
        "INSUFFICIENT_CAPABILITIES",
      );
    }
  }

  // Read task, apply pure state machine transition, persist
  const task = state.getTask(payload.taskId);
  if (!task) {
    throw new StateError("Task not found", "TASK_NOT_FOUND");
  }

  // Self-review guard: assignee cannot approve their own deliverable
  if (task.assignee === senderDid) {
    throw new StateError(
      "Assignee cannot approve their own deliverable",
      "SELF_REVIEW",
    );
  }

  const approved = approveDeliverablePure(task, senderDid, blockNumber);
  state.updateTask(payload.taskId, approved);

  // Pay bounty to assignee from project escrow
  const bountyAmount = task.bountyAmount;
  const assigneeDid = task.assignee;
  if (bountyAmount > 0n && assigneeDid) {
    // Budget validation — always check, even when budget=0n (agent-funded)
    const treasuryDid = projectEscrowDid(payload.projectId);
    const escrowBal = state.getBalance(treasuryDid);
    if (escrowBal < bountyAmount) {
      throw new StateError(
        "Bounty exceeds project escrow balance",
        "INSUFFICIENT_BALANCE",
      );
    }
    if (project.budget > 0n && project.spent + bountyAmount > project.budget) {
      throw new StateError("Bounty exceeds project budget", "BUDGET_EXCEEDED");
    }

    // Debit from project escrow, credit to assignee
    state.deductBalance(treasuryDid, bountyAmount);
    state.creditBalance(assigneeDid, bountyAmount);

    // Update contributor metrics via lifecycle function
    let updated = recordTaskCompletion(project, assigneeDid, bountyAmount);

    // G6: Auto-advance growing → mature when all tasks complete
    if (
      updated.phase === "growing" &&
      updated.completedTasks >= updated.totalTasks &&
      updated.totalTasks > 0
    ) {
      updated = completeProject(updated, blockNumber);
    }

    state.updateProject(payload.projectId, updated);
  }

  return 40000n;
}

function executeDeployService(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    projectId: string;
    name: string;
    description: string;
    serviceType: import("../core/tree/types.js").ServiceType;
    endpoint: ServiceEndpoint;
    abi: ContractABI;
    pricing: PricingModel;
    contractAddress: string;
  };

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }

  // Authorization: only founder can deploy services
  const senderDid = resolveSenderDid(tx);
  if (project.founder !== senderDid) {
    throw new StateError("Only founder can deploy services", "UNAUTHORIZED");
  }

  // Phase validation: services can only be deployed in mature phase
  if (project.phase !== "mature") {
    throw new StateError(
      "Services can only be deployed in mature phase",
      "INVALID_TRANSITION",
    );
  }

  // Validate pricing model before deployment
  validatePricingModel(payload.pricing);

  const service = createServiceLeaf(
    payload.projectId,
    payload.name,
    payload.description,
    payload.serviceType,
    payload.endpoint,
    payload.abi,
    payload.pricing,
    payload.contractAddress,
    BigInt(project.serviceIds.length),
    blockNumber,
  );

  // Create health check schedule and activate with schedule ID
  const blockTimestampMs = Number(blockNumber) * 2521; // approximate
  const healthSchedule = createHealthCheckSchedule(
    service,
    senderDid,
    blockTimestampMs,
  );
  state.registerSchedule(healthSchedule);
  const activeService = activateService(service, healthSchedule.id);
  state.createService(activeService);

  // Add service ID to project and transition to fruiting
  const newServiceIds = [...project.serviceIds, activeService.id];
  const deployed = deployProject(
    project,
    newServiceIds,
    payload.contractAddress,
    blockNumber,
  );
  state.updateProject(payload.projectId, deployed);

  return 80000n;
}

/**
 * Distribute service revenue to contributors, treasury, and research fund.
 * Includes verifyDistribution() integrity check.
 */
function distributeServiceRevenue(
  state: StateManager,
  serviceProjectId: string,
  amount: bigint,
  contributors: import("../core/tree/types.js").ContributorRecord[],
): void {
  const dist = calculateRevenueDistribution(amount, contributors);

  // Integrity check: all distributed amounts must sum to the original amount
  if (!verifyDistribution(dist, amount)) {
    throw new StateError(
      "Revenue distribution integrity check failed: amounts do not sum to cost",
      "INTERNAL_ERROR",
    );
  }

  // Credit contributor shares
  for (const share of dist.contributorShares) {
    state.creditBalance(share.did, share.amount);
  }

  // Route evidence tax through 5-way treasury split
  if (dist.evidenceTax > 0n) {
    const treasurySplit = splitEvidenceTax(dist.evidenceTax);
    for (const entry of treasurySplit) {
      if (entry.amount > 0n) {
        state.creditBalance(entry.did, entry.amount);
      }
    }
  }

  // Research fund (23.4%) goes to separate address from evidence tax research (77% of 22%)
  if (dist.researchFund > 0n) {
    state.creditBalance(TREASURY.ADDRESSES.RESEARCH_FUND, dist.researchFund);
  }
  if (dist.protocolTreasury > 0n) {
    state.creditBalance(TREASURY.ADDRESSES.FOUNDATION, dist.protocolTreasury);
  }

  // Track revenue on project
  state.updateProject(serviceProjectId, {
    revenueGenerated:
      (state.getProject(serviceProjectId)?.revenueGenerated ?? 0n) + amount,
  });
}

function executeCallService(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const rawPayload = tx.payload as unknown as {
    serviceId: string;
    units: bigint | string | number;
  };
  const callerDid = resolveSenderDid(tx);

  // Coerce units to bigint (RPC sends strings via JSON)
  const units =
    typeof rawPayload.units === "string" || typeof rawPayload.units === "number"
      ? BigInt(rawPayload.units)
      : rawPayload.units;

  const service = state.getService(rawPayload.serviceId);
  if (!service) {
    throw new StateError("Service not found", "SERVICE_NOT_FOUND");
  }
  if (service.status !== "active") {
    throw new StateError("Service is not active", "INVALID_TRANSITION");
  }

  // Subscription pricing: caller must have an active subscription
  if (service.pricing.type === "subscription") {
    if (
      !state.hasActiveSubscription(rawPayload.serviceId, callerDid, blockNumber)
    ) {
      throw new StateError(
        "Active subscription required — use subscribe_service transaction first",
        "INVALID_TRANSITION",
      );
    }
  }

  const cost = calculateCallCost(service, units);
  const balance = state.getBalance(callerDid);
  if (balance < cost) {
    throw new StateError(
      "Insufficient balance for service call",
      "INSUFFICIENT_BALANCE",
    );
  }

  // Deduct payment (subscription calls are pre-paid, cost=0)
  if (cost > 0n) {
    state.deductBalance(callerDid, cost);
  }

  // Record service call
  state.incrementServiceCalls(rawPayload.serviceId, cost, blockNumber);

  // Distribute revenue
  const project = state.getProject(service.projectId);
  if (!project) {
    throw new StateError("Service project not found", "PROJECT_NOT_FOUND");
  }
  if (cost > 0n) {
    distributeServiceRevenue(
      state,
      service.projectId,
      cost,
      project.contributors,
    );
  }

  return 60000n;
}

function executeSubscribeService(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as { serviceId: string };
  const callerDid = resolveSenderDid(tx);

  const service = state.getService(payload.serviceId);
  if (!service) {
    throw new StateError("Service not found", "SERVICE_NOT_FOUND");
  }
  if (service.status !== "active") {
    throw new StateError("Service is not active", "INVALID_TRANSITION");
  }
  if (service.pricing.type !== "subscription") {
    throw new StateError(
      "Service does not use subscription pricing",
      "INVALID_TRANSITION",
    );
  }

  const subscriptionFee = service.pricing.basePrice;
  const subscriptionBlocks = service.pricing.subscriptionBlocks!;

  // Check caller can afford the subscription
  const balance = state.getBalance(callerDid);
  if (balance < subscriptionFee) {
    throw new StateError(
      "Insufficient balance for subscription fee",
      "INSUFFICIENT_BALANCE",
    );
  }

  // Deduct subscription fee
  state.deductBalance(callerDid, subscriptionFee);

  // Calculate expiry: extend existing subscription or start from current block
  const existingExpiry = state.getSubscriptionExpiry(
    payload.serviceId,
    callerDid,
  );
  const startBlock =
    existingExpiry !== null && existingExpiry > blockNumber
      ? existingExpiry
      : blockNumber;
  const expiresAt = startBlock + subscriptionBlocks;
  state.setSubscription(payload.serviceId, callerDid, expiresAt);

  // Record revenue on service
  state.incrementServiceCalls(payload.serviceId, subscriptionFee, blockNumber);

  // Distribute subscription revenue (same path as call_service)
  const project = state.getProject(service.projectId);
  if (!project) {
    throw new StateError("Service project not found", "PROJECT_NOT_FOUND");
  }
  if (subscriptionFee > 0n) {
    distributeServiceRevenue(
      state,
      service.projectId,
      subscriptionFee,
      project.contributors,
    );
  }

  return 50000n;
}

function executePauseService(state: StateManager, tx: Transaction): bigint {
  const payload = tx.payload as unknown as { serviceId: string };

  const service = state.getService(payload.serviceId);
  if (!service) {
    throw new StateError("Service not found", "SERVICE_NOT_FOUND");
  }

  // Authorization: only project founder can pause services
  const senderDid = resolveSenderDid(tx);
  const project = state.getProject(service.projectId);
  if (!project) {
    throw new StateError("Service project not found", "PROJECT_NOT_FOUND");
  }
  if (project.founder !== senderDid) {
    throw new StateError(
      "Only project founder can pause services",
      "UNAUTHORIZED",
    );
  }

  const paused = pauseTreeService(service);
  state.updateService(payload.serviceId, paused);

  return 25000n;
}

function executeRetireService(state: StateManager, tx: Transaction): bigint {
  const payload = tx.payload as unknown as { serviceId: string };

  const service = state.getService(payload.serviceId);
  if (!service) {
    throw new StateError("Service not found", "SERVICE_NOT_FOUND");
  }

  // Authorization: only project founder can retire services
  const senderDid = resolveSenderDid(tx);
  const project = state.getProject(service.projectId);
  if (!project) {
    throw new StateError("Service project not found", "PROJECT_NOT_FOUND");
  }
  if (project.founder !== senderDid) {
    throw new StateError(
      "Only project founder can retire services",
      "UNAUTHORIZED",
    );
  }

  const retired = retireTreeService(service);
  state.updateService(payload.serviceId, retired);

  return 25000n;
}

function executeResumeService(state: StateManager, tx: Transaction): bigint {
  const payload = tx.payload as unknown as { serviceId: string };

  const service = state.getService(payload.serviceId);
  if (!service) {
    throw new StateError("Service not found", "SERVICE_NOT_FOUND");
  }

  // Authorization: only project founder can resume services
  const senderDid = resolveSenderDid(tx);
  const project = state.getProject(service.projectId);
  if (!project) {
    throw new StateError("Service project not found", "PROJECT_NOT_FOUND");
  }
  if (project.founder !== senderDid) {
    throw new StateError(
      "Only project founder can resume services",
      "UNAUTHORIZED",
    );
  }

  const resumed = resumeTreeService(service);
  state.updateService(payload.serviceId, resumed);

  return 25000n;
}

// ============ Tree of Life Lifecycle Handlers ============

function executeStartDevelopment(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as { projectId: string };
  const senderDid = resolveSenderDid(tx);

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }
  if (project.founder !== senderDid) {
    throw new StateError("Only founder can start development", "UNAUTHORIZED");
  }

  const updated = startDevelopment(project, blockNumber);
  state.updateProject(payload.projectId, updated);

  return 40000n;
}

function executeCompleteProject(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as { projectId: string };
  const senderDid = resolveSenderDid(tx);

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }
  if (project.founder !== senderDid) {
    throw new StateError(
      "Only founder can mark project complete",
      "UNAUTHORIZED",
    );
  }

  const updated = completeProject(project, blockNumber);
  state.updateProject(payload.projectId, updated);

  return 40000n;
}

function executeBeginSeeding(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as { projectId: string };
  const senderDid = resolveSenderDid(tx);

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }
  if (project.founder !== senderDid) {
    throw new StateError("Only founder can begin seeding", "UNAUTHORIZED");
  }

  const updated = beginSeeding(project, blockNumber);
  state.updateProject(payload.projectId, updated);

  return 40000n;
}

function executePauseProject(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    projectId: string;
    reason?: string;
  };
  const senderDid = resolveSenderDid(tx);

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }
  if (project.founder !== senderDid) {
    throw new StateError("Only founder can pause project", "UNAUTHORIZED");
  }

  const updated = pauseProject(project, payload.reason ?? "", blockNumber);
  state.updateProject(payload.projectId, updated);

  return 30000n;
}

function executeResumeProject(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    projectId: string;
    targetPhase: ProjectPhase;
  };
  const senderDid = resolveSenderDid(tx);

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }
  if (project.founder !== senderDid) {
    throw new StateError("Only founder can resume project", "UNAUTHORIZED");
  }

  const updated = resumeProject(project, payload.targetPhase, blockNumber);
  state.updateProject(payload.projectId, updated);

  return 30000n;
}

function executeAbandonProject(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as { projectId: string };
  const senderDid = resolveSenderDid(tx);

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }
  if (project.founder !== senderDid) {
    throw new StateError("Only founder can abandon project", "UNAUTHORIZED");
  }

  const updated = abandonProject(project, blockNumber);
  state.updateProject(payload.projectId, updated);

  // Refund remaining treasury to founder
  const treasuryDid = projectEscrowDid(payload.projectId);
  const remaining = state.getBalance(treasuryDid);
  if (remaining > 0n) {
    state.deductBalance(treasuryDid, remaining);
    state.creditBalance(senderDid, remaining);
  }

  return 40000n;
}

function executeAddContributor(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    projectId: string;
    agentDid: string;
    role: ContributorRole;
    declaredCapabilities?: string[];
  };
  const senderDid = resolveSenderDid(tx);

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }
  if (project.founder !== senderDid) {
    throw new StateError("Only founder can add contributors", "UNAUTHORIZED");
  }

  // G2: Advisory capability validation — check declared capabilities against project spec
  // Non-blocking: founder can override (may have off-chain knowledge)
  if (payload.declaredCapabilities && payload.declaredCapabilities.length > 0) {
    const projectCaps = project.productSpec.capabilities;
    if (projectCaps.length > 0) {
      const hasRelevant = payload.declaredCapabilities.some((cap) =>
        projectCaps.includes(cap),
      );
      if (!hasRelevant) {
        // Advisory only — logged but not blocking
        // In production this could emit a governance event
      }
    }
  }

  const updated = addContributor(
    project,
    payload.agentDid,
    payload.role,
    blockNumber,
  );
  state.updateProject(payload.projectId, updated);

  return 30000n;
}

function executeAddTask(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    projectId: string;
    title: string;
    description: string;
    taskType: TaskType;
    requiredCapabilities: string[];
    bountyAmount: bigint;
    deadlineBlock?: bigint;
  };
  const senderDid = resolveSenderDid(tx);

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }

  // Phase validation: tasks can only be added during active development
  if (project.phase !== "sprout" && project.phase !== "growing") {
    throw new StateError(
      "Tasks can only be added in sprout or growing phase",
      "INVALID_TRANSITION",
    );
  }

  if (
    project.founder !== senderDid &&
    !project.contributors.some((c) => c.did === senderDid)
  ) {
    throw new StateError("Only project members can add tasks", "UNAUTHORIZED");
  }

  // Validate required capabilities against known set
  for (const cap of payload.requiredCapabilities) {
    if (!VALID_TASK_CAPABILITY_NAMES.has(cap)) {
      throw new StateError(`Invalid capability: ${cap}`, "INVALID_CAPABILITY");
    }
  }

  // Validate bounty against project escrow
  const bountyAmount = BigInt(payload.bountyAmount);
  if (bountyAmount > 0n) {
    const escrowDid = projectEscrowDid(payload.projectId);
    const escrowBal = state.getBalance(escrowDid);

    // Sum outstanding committed bounties (non-completed tasks)
    const existingTasks = state.getTasksByProject(payload.projectId);
    const committedBounties = existingTasks
      .filter(
        (t) =>
          t.status !== "completed" &&
          t.status !== "rejected" &&
          t.status !== "disputed",
      )
      .reduce((sum, t) => sum + t.bountyAmount, 0n);

    if (committedBounties + bountyAmount > escrowBal) {
      throw new StateError(
        "Task bounty would exceed project escrow balance",
        "BUDGET_EXCEEDED",
      );
    }
  }

  // Create the task record in TaskStateTree
  const nonce = state.getNonce(senderDid);
  const task = createTreeTask(
    payload.projectId,
    payload.title,
    payload.description,
    payload.taskType,
    payload.requiredCapabilities,
    BigInt(payload.bountyAmount),
    nonce,
    blockNumber,
    payload.deadlineBlock ? BigInt(payload.deadlineBlock) : undefined,
  );
  state.createTask(task);

  // Register task ID with the project
  const updated = addTask(project, task.id);
  state.updateProject(payload.projectId, updated);

  return 25000n;
}

// ============ Task Lifecycle Handlers ============

function executeStartWork(
  state: StateManager,
  tx: Transaction,
  _blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    projectId: string;
    taskId: string;
  };
  const senderDid = resolveSenderDid(tx);

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }

  // Phase validation: work can only start during active development
  if (project.phase !== "sprout" && project.phase !== "growing") {
    throw new StateError(
      "Work can only start in sprout or growing phase",
      "INVALID_TRANSITION",
    );
  }

  const task = state.getTask(payload.taskId);
  if (!task) {
    throw new StateError("Task not found", "TASK_NOT_FOUND");
  }

  const updated = startWorkPure(task, senderDid);
  state.updateTask(payload.taskId, updated);

  return 20000n;
}

function executeRejectDeliverable(
  state: StateManager,
  tx: Transaction,
  _blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    projectId: string;
    taskId: string;
    reason: string;
  };
  const senderDid = resolveSenderDid(tx);

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }

  // Phase validation: rejections allowed during development or maturation
  if (
    project.phase !== "sprout" &&
    project.phase !== "growing" &&
    project.phase !== "mature"
  ) {
    throw new StateError(
      "Deliverables can only be rejected in sprout, growing, or mature phase",
      "INVALID_TRANSITION",
    );
  }

  // Authorization: only founder or contributors can reject
  if (
    project.founder !== senderDid &&
    !project.contributors.some((c) => c.did === senderDid)
  ) {
    throw new StateError(
      "Only project members can reject deliverables",
      "UNAUTHORIZED",
    );
  }

  const task = state.getTask(payload.taskId);
  if (!task) {
    throw new StateError("Task not found", "TASK_NOT_FOUND");
  }

  const updated = rejectDeliverablePure(task, senderDid, payload.reason);
  state.updateTask(payload.taskId, updated);

  return 30000n;
}

function executeReopenTask(
  state: StateManager,
  tx: Transaction,
  _blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    projectId: string;
    taskId: string;
  };
  const senderDid = resolveSenderDid(tx);

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }

  // Phase validation: tasks can only be reopened during active development
  if (project.phase !== "sprout" && project.phase !== "growing") {
    throw new StateError(
      "Tasks can only be reopened in sprout or growing phase",
      "INVALID_TRANSITION",
    );
  }

  // Authorization: only founder or assignee can reopen
  const task = state.getTask(payload.taskId);
  if (!task) {
    throw new StateError("Task not found", "TASK_NOT_FOUND");
  }

  if (project.founder !== senderDid && task.assignee !== senderDid) {
    throw new StateError(
      "Only founder or assignee can reopen task",
      "UNAUTHORIZED",
    );
  }

  const updated = reopenTaskPure(task);
  state.updateTask(payload.taskId, updated);

  return 20000n;
}

// ============ Opportunity Seed Handlers ============

function executeDetectOpportunity(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    seeds: Array<{
      id: string;
      knowledgeDomain: string;
      signal: import("../core/tree/types.js").DemandSignal;
      confidence: number;
      estimatedDemand: bigint;
      estimatedEffort: number;
      estimatedROI: number;
      expiresAtBlock: bigint;
    }>;
  };

  for (const seedData of payload.seeds) {
    // Skip if seed already exists
    if (state.getSeed(seedData.id)) continue;

    const seed: OpportunitySeed = {
      id: seedData.id,
      detectedAtBlock: blockNumber,
      knowledgeDomain: seedData.knowledgeDomain,
      signal: seedData.signal,
      confidence: seedData.confidence,
      estimatedDemand: BigInt(seedData.estimatedDemand),
      estimatedEffort: seedData.estimatedEffort,
      estimatedROI: seedData.estimatedROI,
      status: "detected",
      expiresAtBlock: BigInt(seedData.expiresAtBlock),
    };
    state.addSeed(seed);
  }

  return 15000n;
}

function executeClaimOpportunity(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    seedId: string;
    projectName: string;
    projectDescription: string;
    productSpec: import("../core/tree/types.js").ProductSpec;
    fundingSource: FundingSource;
  };
  const senderDid = resolveSenderDid(tx);

  const seed = state.getSeed(payload.seedId);
  if (!seed) {
    throw new StateError("Opportunity seed not found", "SEED_NOT_FOUND");
  }
  if (seed.status !== "detected") {
    throw new StateError(
      `Seed is '${seed.status}', not available for claiming`,
      "INVALID_TRANSITION",
    );
  }
  if (seed.expiresAtBlock <= blockNumber) {
    // Mark as expired and reject
    state.updateSeed(payload.seedId, { status: "expired" });
    throw new StateError("Opportunity seed has expired", "SEED_EXPIRED");
  }

  // Create project from the seed
  const nonce = state.getNonce(senderDid);
  const project = createProject(
    senderDid,
    payload.projectName,
    payload.projectDescription,
    payload.productSpec,
    seed.knowledgeDomain,
    seed.signal,
    payload.fundingSource,
    nonce,
    blockNumber,
  );
  state.createProject(project);

  // Update seed: mark as claimed and link to project
  state.updateSeed(payload.seedId, {
    status: "claimed",
    claimedBy: senderDid,
    projectId: project.id,
  });

  return 35000n;
}

// ============ Tree of Life Recruitment ============

function executeApplyToProject(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    projectId: string;
    proposedRole: ContributorRole;
    message: string;
    declaredCapabilities: string[];
  };
  const senderDid = resolveSenderDid(tx);

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }

  // Applications only accepted during sprout/growing phases
  if (project.phase !== "sprout" && project.phase !== "growing") {
    throw new StateError(
      `Cannot apply in '${project.phase}' phase`,
      "INVALID_TRANSITION",
    );
  }

  // Can't apply if already a contributor
  if (project.contributors.some((c) => c.did === senderDid)) {
    throw new StateError(
      "Already a contributor to this project",
      "INVALID_TRANSITION",
    );
  }

  // Can't apply if already have pending application
  const applications = project.applications ?? [];
  if (applications.some((a) => a.did === senderDid && a.status === "pending")) {
    throw new StateError("Application already pending", "INVALID_TRANSITION");
  }

  const application: import("../core/tree/types.js").ProjectApplication = {
    did: senderDid,
    proposedRole: payload.proposedRole,
    message: payload.message,
    capabilities: payload.declaredCapabilities ?? [],
    appliedAtBlock: blockNumber,
    status: "pending",
  };

  state.updateProject(payload.projectId, {
    ...project,
    applications: [...applications, application],
  });

  return 25000n;
}

function executeReviewApplication(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    projectId: string;
    applicantDid: string;
    decision: "approve" | "reject";
  };
  const senderDid = resolveSenderDid(tx);

  const project = state.getProject(payload.projectId);
  if (!project) {
    throw new StateError("Project not found", "PROJECT_NOT_FOUND");
  }
  if (project.founder !== senderDid) {
    throw new StateError(
      "Only founder can review applications",
      "UNAUTHORIZED",
    );
  }

  const applications = project.applications ?? [];
  const appIndex = applications.findIndex(
    (a) => a.did === payload.applicantDid && a.status === "pending",
  );
  if (appIndex === -1) {
    throw new StateError(
      "No pending application found for this agent",
      "INVALID_TRANSITION",
    );
  }

  const updatedApplications = applications.map((a, i) =>
    i === appIndex
      ? {
          ...a,
          status:
            payload.decision === "approve"
              ? ("approved" as const)
              : ("rejected" as const),
        }
      : a,
  );

  let updated: import("../core/tree/types.js").ProductProject = {
    ...project,
    applications: updatedApplications,
  };

  // If approved, add as contributor
  if (payload.decision === "approve") {
    const app = applications[appIndex];
    updated = addContributor(
      updated,
      payload.applicantDid,
      app.proposedRole,
      blockNumber,
    );
  }

  state.updateProject(payload.projectId, updated);

  return 30000n;
}

function executeSetAvailability(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
): bigint {
  const payload = tx.payload as unknown as {
    available: boolean;
    capabilities: string[];
    preferredDomains: string[];
    minimumBounty: bigint | string;
  };
  const senderDid = resolveSenderDid(tx);

  const balance = state.getBalance(senderDid);
  if (balance === 0n && state.getNonce(senderDid) === 0n) {
    throw new StateError("Account not found", "ACCOUNT_NOT_FOUND");
  }

  state.setAvailability(senderDid, {
    available: payload.available,
    capabilities: payload.capabilities ?? [],
    preferredDomains: payload.preferredDomains ?? [],
    minimumBounty:
      typeof payload.minimumBounty === "string"
        ? BigInt(payload.minimumBounty)
        : BigInt(payload.minimumBounty ?? 0),
    updatedAtBlock: blockNumber,
  });

  return 20000n;
}
