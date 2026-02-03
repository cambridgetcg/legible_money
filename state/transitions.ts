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

import { StateManager } from './state_manager.js';
import {
  type StateTransition,
  type TransitionType,
  type TransferPayload,
  type StakePayload,
  type DelegatePayload,
  type SlashPayload,
  type AddFactPayload,
  type ChallengeFactPayload,
  type CreateAccountPayload,
  type EpochTransitionPayload,
  type AccountDelta,
  type FactState,
  StateError,
} from './types.js';
import type { Transaction, TransactionType, VerificationResult, ConsensusParams } from '../consensus/pot/types.js';
import type { PublicKey } from '../core/crypto/keys.js';
import type { Confidence } from '../core/knowledge/types.js';
import { generateFactId } from './knowledge.js';
import {
  type ResearchSubmitPayload,
  type BountyCreatePayload,
  type BountyClaimPayload,
  type BountyFulfillPayload,
  type ResearchState,
  type BountyState,
  DEFAULT_RESEARCH_PARAMETERS,
  RESEARCH_TREASURY_DID,
} from '../core/research/types.js';
import { sha512_256Hex } from '../core/crypto/hashing.js';
import {
  type InitiateDisputePayload,
  type CommitEvidencePayload,
  type RevealEvidencePayload,
  type ArbiterVotePayload,
  type EscalateDisputePayload,
  type SettleDisputePayload,
  DISPUTE_ECONOMICS,
  PHASE_DURATIONS,
  DisputeError,
} from '../core/dispute/types.js';
import { createDispute, commitEvidence, revealEvidence, castArbiterVote, tryAdvancePhase } from '../core/dispute/dispute_engine.js';
import { processEscalation } from '../core/dispute/escalation.js';
import { calculateSettlement } from '../core/dispute/settlement.js';
import {
  type CreateVestingPayload,
  type ClaimRewardsPayload,
  type PauseVestingPayload,
  type ResumeVestingPayload,
  type AccelerateVestingPayload,
  type FalsifyVestingPayload,
  type VestingCategory,
  getVestingConfig,
} from '../core/rewards/types.js';
import { determineVestingCategory } from '../core/rewards/vesting_engine.js';

// ============ Transaction Executor ============

/**
 * Execute a transaction against state
 * Returns the new state root or throws on failure
 */
export function executeTransaction(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  params: ConsensusParams
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

    switch (tx.type) {
      case 'transfer':
        gasUsed = executeTransfer(state, tx, params);
        break;

      case 'stake':
        gasUsed = executeStake(state, tx, params);
        break;

      case 'unstake':
        gasUsed = executeUnstake(state, tx, params);
        break;

      case 'delegate':
        gasUsed = executeDelegate(state, tx, params);
        break;

      case 'undelegate':
        gasUsed = executeUndelegate(state, tx, params);
        break;

      case 'claim_submit':
        gasUsed = executeClaimSubmit(state, tx, blockNumber, params);
        break;

      case 'verification_commit':
        gasUsed = executeVerificationCommit(state, tx, params);
        break;

      case 'verification_reveal':
        gasUsed = executeVerificationReveal(state, tx, params);
        break;

      case 'challenge':
        gasUsed = executeChallenge(state, tx, blockNumber, params);
        break;

      case 'governance_vote':
        gasUsed = executeGovernanceVote(state, tx, params);
        break;

      case 'governance_propose':
        gasUsed = executeGovernancePropose(state, tx, params);
        break;

      // Research transactions (Protocol Research Repository)
      case 'research_submit':
        gasUsed = executeResearchSubmit(state, tx, blockNumber, params);
        break;

      case 'research_challenge':
        gasUsed = executeResearchChallenge(state, tx, blockNumber, params);
        break;

      case 'bounty_create':
        gasUsed = executeBountyCreate(state, tx, blockNumber, params);
        break;

      case 'bounty_claim':
        gasUsed = executeBountyClaim(state, tx, params);
        break;

      case 'bounty_fulfill':
        gasUsed = executeBountyFulfill(state, tx, blockNumber, params);
        break;

      // Dispute transactions (Dispute Resolution System)
      case 'dispute_initiate':
        gasUsed = executeDisputeInitiate(state, tx, blockNumber, params);
        break;

      case 'dispute_commit_evidence':
        gasUsed = executeDisputeCommitEvidence(state, tx, blockNumber, params);
        break;

      case 'dispute_reveal_evidence':
        gasUsed = executeDisputeRevealEvidence(state, tx, blockNumber, params);
        break;

      case 'dispute_arbiter_vote':
        gasUsed = executeDisputeArbiterVote(state, tx, blockNumber, params);
        break;

      case 'dispute_escalate':
        gasUsed = executeDisputeEscalate(state, tx, blockNumber, params);
        break;

      case 'dispute_settle':
        gasUsed = executeDisputeSettle(state, tx, blockNumber, params);
        break;

      case 'dispute_timeout':
        gasUsed = executeDisputeTimeout(state, tx, blockNumber, params);
        break;

      // Vesting transactions (Gradual Reward Release System)
      case 'vesting_create':
        gasUsed = executeVestingCreate(state, tx, blockNumber, params);
        break;

      case 'vesting_claim':
        gasUsed = executeVestingClaim(state, tx, blockNumber, params);
        break;

      case 'vesting_pause':
        gasUsed = executeVestingPause(state, tx, blockNumber, params);
        break;

      case 'vesting_resume':
        gasUsed = executeVestingResume(state, tx, blockNumber, params);
        break;

      case 'vesting_accelerate':
        gasUsed = executeVestingAccelerate(state, tx, blockNumber, params);
        break;

      case 'vesting_falsify':
        gasUsed = executeVestingFalsify(state, tx, blockNumber, params);
        break;

      default:
        state.rollbackTransaction();
        return {
          success: false,
          error: `Unknown transaction type: ${tx.type}`,
          gasUsed: 0n,
        };
    }

    // Deduct fee from sender
    const senderDid = publicKeyToDid(tx.sender);
    const currentBalance = state.getBalance(senderDid);
    if (currentBalance < tx.fee) {
      state.rollbackTransaction();
      return {
        success: false,
        error: 'Insufficient balance for fee',
        gasUsed: 0n,
      };
    }

    // Commit and return
    const newRoot = state.commitTransaction();

    return {
      success: true,
      stateRoot: newRoot,
      gasUsed,
    };
  } catch (error) {
    state.rollbackTransaction();
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error',
      gasUsed: 0n,
    };
  }
}

// ============ Individual Transaction Handlers ============

function executeTransfer(
  state: StateManager,
  tx: Transaction,
  params: ConsensusParams
): bigint {
  const payload = tx.payload as TransferPayload;
  const senderDid = publicKeyToDid(tx.sender);

  // Validate
  if (payload.amount <= 0n) {
    throw new StateError('Invalid transfer amount', 'INVALID_TRANSITION');
  }

  // Execute transfer
  state.transfer(senderDid, payload.to, payload.amount);

  // Increment nonce is handled by transfer

  return 21000n; // Base gas for transfer
}

function executeStake(
  state: StateManager,
  tx: Transaction,
  params: ConsensusParams
): bigint {
  const payload = tx.payload as StakePayload;
  const senderDid = publicKeyToDid(tx.sender);

  if (payload.amount < params.minValidatorStake) {
    throw new StateError(
      `Minimum stake is ${params.minValidatorStake}`,
      'INVALID_TRANSITION'
    );
  }

  state.stake(senderDid, payload.amount);

  // If not already a validator, register
  if (!state.isActiveValidator(tx.sender)) {
    state.registerValidator(tx.sender, senderDid, payload.amount);
  }

  return 50000n; // Staking gas
}

function executeUnstake(
  state: StateManager,
  tx: Transaction,
  params: ConsensusParams
): bigint {
  const payload = tx.payload as StakePayload;
  const senderDid = publicKeyToDid(tx.sender);

  state.unstake(senderDid, payload.amount);

  return 50000n;
}

function executeDelegate(
  state: StateManager,
  tx: Transaction,
  params: ConsensusParams
): bigint {
  const payload = tx.payload as DelegatePayload;
  const senderDid = publicKeyToDid(tx.sender);

  // Validate validator exists
  if (!state.isActiveValidator(payload.validator)) {
    throw new StateError('Validator not active', 'VALIDATOR_NOT_FOUND');
  }

  // Transfer from balance to delegation
  const currentBalance = state.getBalance(senderDid);
  if (currentBalance < payload.amount) {
    throw new StateError('Insufficient balance to delegate', 'INSUFFICIENT_BALANCE');
  }

  // Deduct from account balance
  state.transfer(senderDid, 'did:lgm:delegation-pool', payload.amount);

  // Add to validator's delegated stake
  state.delegate(senderDid, payload.validator, payload.amount);

  return 60000n;
}

function executeUndelegate(
  state: StateManager,
  tx: Transaction,
  params: ConsensusParams
): bigint {
  const payload = tx.payload as DelegatePayload;
  const senderDid = publicKeyToDid(tx.sender);

  // Note: Actual unbonding has a timelock
  // The amount would be locked until unbonding completes

  return 60000n;
}

function executeClaimSubmit(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  params: ConsensusParams
): bigint {
  const payload = tx.payload as {
    claim: string;
    domain: string;
    stake: bigint;
    references?: string[];
  };

  const senderDid = publicKeyToDid(tx.sender);

  // Validate stake
  if (payload.stake < params.minClaimStake) {
    throw new StateError(
      `Minimum claim stake is ${params.minClaimStake}`,
      'INVALID_TRANSITION'
    );
  }

  // Lock claim stake
  state.stake(senderDid, payload.stake);

  // Claim goes to verification queue (handled by consensus)
  // Here we just validate and lock stake

  return 100000n;
}

function executeVerificationCommit(
  state: StateManager,
  tx: Transaction,
  params: ConsensusParams
): bigint {
  // Commitments are handled by consensus layer
  // State just validates the validator is active
  if (!state.isActiveValidator(tx.sender)) {
    throw new StateError('Only validators can commit verifications', 'INVALID_TRANSITION');
  }

  return 30000n;
}

function executeVerificationReveal(
  state: StateManager,
  tx: Transaction,
  params: ConsensusParams
): bigint {
  // Reveals are handled by consensus layer
  if (!state.isActiveValidator(tx.sender)) {
    throw new StateError('Only validators can reveal verifications', 'INVALID_TRANSITION');
  }

  return 40000n;
}

function executeChallenge(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  params: ConsensusParams
): bigint {
  const payload = tx.payload as ChallengeFactPayload;
  const senderDid = publicKeyToDid(tx.sender);

  // Validate fact exists and is active
  if (!state.isActiveFact(payload.factId)) {
    throw new StateError('Fact not found or not active', 'FACT_NOT_FOUND');
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
  params: ConsensusParams
): bigint {
  // Governance voting
  // Requires stake to vote
  const senderDid = publicKeyToDid(tx.sender);
  const account = state.getAccount(senderDid);

  if (!account || account.stakedBalance === 0n) {
    throw new StateError('Must have stake to vote', 'INVALID_TRANSITION');
  }

  return 25000n;
}

function executeGovernancePropose(
  state: StateManager,
  tx: Transaction,
  params: ConsensusParams
): bigint {
  // Governance proposals require minimum stake
  const senderDid = publicKeyToDid(tx.sender);
  const account = state.getAccount(senderDid);

  const MIN_PROPOSAL_STAKE = 10000n * 10n ** 18n; // 10000 LGM

  if (!account || account.stakedBalance < MIN_PROPOSAL_STAKE) {
    throw new StateError(
      `Must have at least ${MIN_PROPOSAL_STAKE} staked to propose`,
      'INVALID_TRANSITION'
    );
  }

  return 150000n;
}

// ============ Research Transaction Handlers ============

/**
 * Submit research evidence to the Protocol Research Repository
 */
function executeResearchSubmit(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  params: ConsensusParams
): bigint {
  const payload = tx.payload as ResearchSubmitPayload;
  const senderDid = publicKeyToDid(tx.sender);

  // Validate minimum stake
  if (payload.stake < DEFAULT_RESEARCH_PARAMETERS.minResearchStake) {
    throw new StateError(
      `Minimum research stake is ${DEFAULT_RESEARCH_PARAMETERS.minResearchStake}`,
      'INVALID_TRANSITION'
    );
  }

  // Validate balance
  const balance = state.getBalance(senderDid);
  if (balance < payload.stake) {
    throw new StateError('Insufficient balance for research stake', 'INSUFFICIENT_BALANCE');
  }

  // If targeting a specific fact, verify it exists
  if (payload.targetFactId && !state.isActiveFact(payload.targetFactId)) {
    throw new StateError('Target fact not found or not active', 'FACT_NOT_FOUND');
  }

  // Lock researcher's stake
  state.stake(senderDid, payload.stake);

  // Generate research ID
  const researchId = sha512_256Hex(
    `research:${senderDid}:${blockNumber}:${payload.rawDataCID}`
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
    status: 'pending',
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
  blockNumber: bigint,
  params: ConsensusParams
): bigint {
  const payload = tx.payload as { researchId: string; stake: bigint; evidence: string };
  const senderDid = publicKeyToDid(tx.sender);

  // Validate research exists
  const research = state.getResearch(payload.researchId);
  if (!research) {
    throw new StateError('Research not found', 'INVALID_TRANSITION');
  }

  if (research.status !== 'accepted') {
    throw new StateError('Can only challenge accepted research', 'INVALID_TRANSITION');
  }

  // Validate minimum stake
  const minChallengeStake = DEFAULT_RESEARCH_PARAMETERS.minResearchStake;
  if (payload.stake < minChallengeStake) {
    throw new StateError(
      `Minimum challenge stake is ${minChallengeStake}`,
      'INVALID_TRANSITION'
    );
  }

  // Lock challenger's stake
  state.stake(senderDid, payload.stake);

  // Update research status to challenged
  state.updateResearch(payload.researchId, { status: 'challenged' });

  return 120000n; // Research challenge gas
}

/**
 * Create a research bounty from the research treasury
 */
function executeBountyCreate(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  params: ConsensusParams
): bigint {
  const payload = tx.payload as BountyCreatePayload;
  const senderDid = publicKeyToDid(tx.sender);

  // Only research treasury or governance can create bounties
  // For now, allow anyone with sufficient stake
  const account = state.getAccount(senderDid);
  if (!account || account.stakedBalance < DEFAULT_RESEARCH_PARAMETERS.minBountyStake) {
    throw new StateError(
      `Must have at least ${DEFAULT_RESEARCH_PARAMETERS.minBountyStake} staked to create bounty`,
      'INVALID_TRANSITION'
    );
  }

  // Validate reward amount
  if (payload.reward > DEFAULT_RESEARCH_PARAMETERS.maxBountyReward) {
    throw new StateError(
      `Bounty reward exceeds maximum of ${DEFAULT_RESEARCH_PARAMETERS.maxBountyReward}`,
      'INVALID_TRANSITION'
    );
  }

  // If targeting a fact, validate it exists
  if (payload.targetFactId && !state.isActiveFact(payload.targetFactId)) {
    throw new StateError('Target fact not found or not active', 'FACT_NOT_FOUND');
  }

  // Transfer reward from creator to lock (or from research treasury)
  const creatorBalance = state.getBalance(senderDid);
  if (creatorBalance < payload.reward) {
    throw new StateError('Insufficient balance for bounty reward', 'INSUFFICIENT_BALANCE');
  }

  // Lock the reward
  state.transfer(senderDid, 'did:lgm:bounty-escrow', payload.reward);

  // Generate bounty ID
  const bountyId = sha512_256Hex(
    `bounty:${senderDid}:${blockNumber}:${payload.description.slice(0, 32)}`
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
    status: 'open',
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
  params: ConsensusParams
): bigint {
  const payload = tx.payload as BountyClaimPayload;
  const senderDid = publicKeyToDid(tx.sender);

  // Validate bounty exists and is open
  const bounty = state.getBounty(payload.bountyId);
  if (!bounty) {
    throw new StateError('Bounty not found', 'INVALID_TRANSITION');
  }

  if (bounty.status !== 'open') {
    throw new StateError('Bounty is not available for claiming', 'INVALID_TRANSITION');
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
  blockNumber: bigint,
  params: ConsensusParams
): bigint {
  const payload = tx.payload as BountyFulfillPayload;
  const senderDid = publicKeyToDid(tx.sender);

  // Validate bounty exists and is claimed
  const bounty = state.getBounty(payload.bountyId);
  if (!bounty) {
    throw new StateError('Bounty not found', 'INVALID_TRANSITION');
  }

  if (bounty.status !== 'claimed') {
    throw new StateError('Bounty must be claimed before fulfillment', 'INVALID_TRANSITION');
  }

  if (bounty.claimedBy !== senderDid) {
    throw new StateError('Only the claimer can fulfill this bounty', 'INVALID_TRANSITION');
  }

  // Validate research submission exists and is accepted
  const research = state.getResearch(payload.researchSubmissionId);
  if (!research) {
    throw new StateError('Research submission not found', 'INVALID_TRANSITION');
  }

  if (research.status !== 'accepted') {
    throw new StateError('Research must be accepted before bounty fulfillment', 'INVALID_TRANSITION');
  }

  if (research.submitter !== senderDid) {
    throw new StateError('Research submitter must match bounty claimer', 'INVALID_TRANSITION');
  }

  // Fulfill the bounty
  state.fulfillBounty(payload.bountyId, senderDid);

  // Release reward from escrow to fulfiller
  // Note: In full implementation, this would go through the reward router
  // for 22% research share, but for bounties funded from research treasury
  // the share may be handled differently
  state.transfer('did:lgm:bounty-escrow', senderDid, bounty.reward);

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
  params: ConsensusParams
): bigint {
  const payload = tx.payload as InitiateDisputePayload;
  const senderDid = publicKeyToDid(tx.sender);

  // Validate sender is the challenger
  if (payload.challenger !== senderDid) {
    throw new StateError('Sender must be the challenger', 'INVALID_TRANSITION');
  }

  // Validate minimum stake
  if (payload.stake < DISPUTE_ECONOMICS.minChallengeStake) {
    throw new StateError(
      `Minimum challenge stake is ${DISPUTE_ECONOMICS.minChallengeStake}`,
      'INVALID_TRANSITION'
    );
  }

  // Validate challenger has sufficient balance
  const balance = state.getBalance(senderDid);
  if (balance < payload.stake) {
    throw new StateError('Insufficient balance for challenge stake', 'INSUFFICIENT_BALANCE');
  }

  // Get the target (what's being disputed)
  let defenderDid: string;
  if (payload.targetType === 'fact') {
    const fact = state.getFact(payload.targetId);
    if (!fact) {
      throw new StateError('Target fact not found', 'FACT_NOT_FOUND');
    }
    // Defender is the original submitter (we'd need to track this)
    // For now, use a placeholder - in production, facts would track submitter
    defenderDid = 'did:lgm:fact-defender';
  } else if (payload.targetType === 'evidence') {
    const evidence = state.getEvidence(payload.targetId);
    if (!evidence) {
      throw new StateError('Target evidence not found', 'INVALID_TRANSITION');
    }
    defenderDid = evidence.submitter;
  } else {
    throw new StateError('Invalid target type', 'INVALID_TRANSITION');
  }

  // Lock challenger's stake
  state.stake(senderDid, payload.stake);

  // Create the dispute using the dispute engine
  const dispute = createDispute(
    payload.targetId,
    payload.targetType,
    payload.domain,
    {
      did: senderDid,
      stake: payload.stake,
      initialEvidence: payload.initialEvidence,
    },
    defenderDid,
    blockNumber
  );

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
  _params: ConsensusParams
): bigint {
  const payload = tx.payload as CommitEvidencePayload;
  const senderDid = publicKeyToDid(tx.sender);

  // Validate sender is the committer
  if (payload.committer !== senderDid) {
    throw new StateError('Sender must be the committer', 'INVALID_TRANSITION');
  }

  // Get dispute
  const dispute = state.getDispute(payload.disputeId);
  if (!dispute) {
    throw new StateError('Dispute not found', 'INVALID_TRANSITION');
  }

  // Validate phase
  if (dispute.state.phase !== 'evidence_commit') {
    throw new StateError('Not in evidence commit phase', 'INVALID_TRANSITION');
  }

  // Validate deadline
  if (blockNumber > dispute.state.phaseDeadline) {
    throw new StateError('Evidence commit deadline passed', 'INVALID_TRANSITION');
  }

  // Must be challenger or defender
  if (senderDid !== dispute.challenger.did && senderDid !== dispute.defender.did) {
    throw new StateError('Only dispute parties can commit evidence', 'INVALID_TRANSITION');
  }

  // Commit evidence using dispute engine
  const updatedDispute = commitEvidence(dispute, senderDid, payload.evidenceHash, blockNumber);

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
  _params: ConsensusParams
): bigint {
  const payload = tx.payload as RevealEvidencePayload;
  const senderDid = publicKeyToDid(tx.sender);

  // Validate sender is the revealer
  if (payload.revealer !== senderDid) {
    throw new StateError('Sender must be the revealer', 'INVALID_TRANSITION');
  }

  // Get dispute
  const dispute = state.getDispute(payload.disputeId);
  if (!dispute) {
    throw new StateError('Dispute not found', 'INVALID_TRANSITION');
  }

  // Validate phase
  if (dispute.state.phase !== 'evidence_reveal') {
    throw new StateError('Not in evidence reveal phase', 'INVALID_TRANSITION');
  }

  // Validate deadline
  if (blockNumber > dispute.state.phaseDeadline) {
    throw new StateError('Evidence reveal deadline passed', 'INVALID_TRANSITION');
  }

  // Get actual evidence objects
  const evidenceList = payload.evidenceIds.map(id => {
    const evidence = state.getEvidence(id);
    if (!evidence) {
      throw new StateError(`Evidence ${id} not found`, 'INVALID_TRANSITION');
    }
    return evidence;
  });

  // Reveal evidence using dispute engine
  const updatedDispute = revealEvidence(
    dispute,
    senderDid,
    evidenceList,
    payload.nonce,
    blockNumber
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
  _params: ConsensusParams
): bigint {
  const payload = tx.payload as ArbiterVotePayload;
  const senderDid = publicKeyToDid(tx.sender);

  // Validate sender is the arbiter
  if (payload.arbiter !== senderDid) {
    throw new StateError('Sender must be the arbiter', 'INVALID_TRANSITION');
  }

  // Get dispute
  const dispute = state.getDispute(payload.disputeId);
  if (!dispute) {
    throw new StateError('Dispute not found', 'INVALID_TRANSITION');
  }

  // Validate phase
  if (dispute.state.phase !== 'arbitration' && dispute.state.phase !== 'deliberation') {
    throw new StateError('Not in voting phase', 'INVALID_TRANSITION');
  }

  // Validate deadline
  if (blockNumber > dispute.state.phaseDeadline) {
    throw new StateError('Voting deadline passed', 'INVALID_TRANSITION');
  }

  // Check arbiter is in the selected list
  const isSelectedArbiter = dispute.arbiters.some(a => a.did === senderDid);
  if (!isSelectedArbiter) {
    throw new StateError('Not a selected arbiter for this dispute', 'INVALID_TRANSITION');
  }

  // Get arbiter's stake to lock
  const account = state.getAccount(senderDid);
  if (!account || account.stakedBalance < DISPUTE_ECONOMICS.minArbiterStake) {
    throw new StateError('Insufficient stake to vote as arbiter', 'INSUFFICIENT_BALANCE');
  }

  // Cast vote using dispute engine
  const updatedDispute = castArbiterVote(
    dispute,
    senderDid,
    payload.vote,
    DISPUTE_ECONOMICS.minArbiterStake, // Lock minimum required stake
    blockNumber,
    payload.justificationHash
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
  _params: ConsensusParams
): bigint {
  const payload = tx.payload as EscalateDisputePayload;
  const senderDid = publicKeyToDid(tx.sender);

  // Validate sender is the escalator
  if (payload.escalator !== senderDid) {
    throw new StateError('Sender must be the escalator', 'INVALID_TRANSITION');
  }

  // Get dispute
  const dispute = state.getDispute(payload.disputeId);
  if (!dispute) {
    throw new StateError('Dispute not found', 'INVALID_TRANSITION');
  }

  // Process escalation (validates eligibility, stake, etc.)
  const result = processEscalation(
    dispute,
    senderDid,
    payload.additionalStake,
    blockNumber
  );

  if (!result.success) {
    throw new StateError(result.error || 'Escalation failed', 'INVALID_TRANSITION');
  }

  // Lock additional stake
  state.stake(senderDid, payload.additionalStake);

  // Update dispute state for new tier
  const updatedDispute: typeof dispute = {
    ...dispute,
    state: {
      ...dispute.state,
      phase: 'arbitration',
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
  _params: ConsensusParams
): bigint {
  const payload = tx.payload as SettleDisputePayload;

  // Get dispute
  const dispute = state.getDispute(payload.disputeId);
  if (!dispute) {
    throw new StateError('Dispute not found', 'INVALID_TRANSITION');
  }

  // Validate phase
  if (dispute.state.phase !== 'settlement') {
    throw new StateError('Not in settlement phase', 'INVALID_TRANSITION');
  }

  // Calculate settlement if not provided
  const settlement = payload.settlement || calculateSettlement(dispute);

  // Apply settlement
  const winnerDid = settlement.winner === 'challenger' ? dispute.challenger.did : dispute.defender.did;
  const loserDid = settlement.winner === 'challenger' ? dispute.defender.did : dispute.challenger.did;

  // Return winner's stake + reward
  const winnerStake = settlement.winner === 'challenger' ? dispute.challenger.stake : dispute.defender.stake;
  state.unstake(winnerDid, winnerStake);

  // Reward winner from loser's stake
  state.transfer('did:lgm:dispute-escrow', winnerDid, settlement.winnerReward);

  // Apply loser penalty (portion of stake is slashed)
  state.unstake(loserDid, dispute.challenger.did === loserDid ? dispute.challenger.stake : dispute.defender.stake);
  const loserRefund = (dispute.challenger.did === loserDid ? dispute.challenger.stake : dispute.defender.stake) - settlement.loserPenalty;
  if (loserRefund > 0n) {
    state.transfer('did:lgm:dispute-escrow', loserDid, loserRefund);
  }

  // Distribute arbiter rewards and penalties
  for (const [arbiterDid, reward] of settlement.arbiterRewards) {
    state.transfer('did:lgm:dispute-escrow', arbiterDid, reward);
  }

  for (const [arbiterDid, penalty] of settlement.arbiterPenalties) {
    // Slash arbiter's locked stake
    const account = state.getAccount(arbiterDid);
    if (account) {
      state.slashValidator(tx.sender, penalty, `Wrong arbitration vote in dispute ${dispute.id}`);
    }
  }

  // Protocol fee to treasury
  state.transfer('did:lgm:dispute-escrow', 'did:lgm:protocol-treasury', settlement.protocolFee);

  // Research fund share
  state.transfer('did:lgm:dispute-escrow', RESEARCH_TREASURY_DID, settlement.researchFund);

  // Mark dispute as resolved
  const resolvedDispute: typeof dispute = {
    ...dispute,
    state: {
      ...dispute.state,
      phase: 'resolved',
    },
    outcome: settlement.winner === 'challenger' ? 'challenger_wins' : 'defender_wins',
    winner: settlement.winner,
    resolvedAtBlock: blockNumber,
    updatedAt: Date.now(),
  };

  state.updateDispute(resolvedDispute);

  return 200000n; // Settlement gas
}

/**
 * Handle dispute phase timeout
 * Called when a phase deadline passes without required action
 */
function executeDisputeTimeout(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams
): bigint {
  const payload = tx.payload as { disputeId: string };

  // Get dispute
  const dispute = state.getDispute(payload.disputeId);
  if (!dispute) {
    throw new StateError('Dispute not found', 'INVALID_TRANSITION');
  }

  // Validate deadline has passed
  if (blockNumber <= dispute.state.phaseDeadline) {
    throw new StateError('Phase deadline not yet reached', 'INVALID_TRANSITION');
  }

  // Handle timeout based on current phase
  let updatedDispute: typeof dispute;

  switch (dispute.state.phase) {
    case 'initiation':
      // Defender didn't respond - challenger wins by default
      updatedDispute = {
        ...dispute,
        state: {
          ...dispute.state,
          phase: 'settlement',
          phaseDeadline: blockNumber + BigInt(PHASE_DURATIONS.settlement),
        },
        outcome: 'challenger_wins',
        winner: 'challenger',
        exposedFalsehood: true, // Defender's silence implies falsehood
        updatedAt: Date.now(),
      };
      break;

    case 'evidence_commit':
      // One or both parties didn't commit - penalize non-committers
      const challengerCommitted = dispute.commitments.challenger?.revealed === false || dispute.commitments.challenger !== undefined;
      const defenderCommitted = dispute.commitments.defender?.revealed === false || dispute.commitments.defender !== undefined;

      if (!challengerCommitted && !defenderCommitted) {
        // Both failed - draw with penalties
        updatedDispute = {
          ...dispute,
          state: {
            ...dispute.state,
            phase: 'settlement',
            phaseDeadline: blockNumber + BigInt(PHASE_DURATIONS.settlement),
          },
          outcome: 'draw',
          updatedAt: Date.now(),
        };
      } else if (!challengerCommitted) {
        updatedDispute = {
          ...dispute,
          state: {
            ...dispute.state,
            phase: 'settlement',
            phaseDeadline: blockNumber + BigInt(PHASE_DURATIONS.settlement),
          },
          outcome: 'defender_wins',
          winner: 'defender',
          updatedAt: Date.now(),
        };
      } else {
        updatedDispute = {
          ...dispute,
          state: {
            ...dispute.state,
            phase: 'settlement',
            phaseDeadline: blockNumber + BigInt(PHASE_DURATIONS.settlement),
          },
          outcome: 'challenger_wins',
          winner: 'challenger',
          exposedFalsehood: true,
          updatedAt: Date.now(),
        };
      }
      break;

    case 'evidence_reveal':
      // Non-revealer gets maximum penalty (refusal to reveal = bad faith)
      const challengerRevealed = dispute.commitments.challenger?.revealed;
      const defenderRevealed = dispute.commitments.defender?.revealed;

      if (!challengerRevealed && !defenderRevealed) {
        // Both failed - both get maximum penalty
        updatedDispute = {
          ...dispute,
          state: {
            ...dispute.state,
            phase: 'settlement',
            phaseDeadline: blockNumber + BigInt(PHASE_DURATIONS.settlement),
          },
          outcome: 'draw',
          updatedAt: Date.now(),
        };
      } else if (!challengerRevealed) {
        updatedDispute = {
          ...dispute,
          state: {
            ...dispute.state,
            phase: 'settlement',
            phaseDeadline: blockNumber + BigInt(PHASE_DURATIONS.settlement),
          },
          outcome: 'defender_wins',
          winner: 'defender',
          updatedAt: Date.now(),
        };
      } else {
        updatedDispute = {
          ...dispute,
          state: {
            ...dispute.state,
            phase: 'settlement',
            phaseDeadline: blockNumber + BigInt(PHASE_DURATIONS.settlement),
          },
          outcome: 'challenger_wins',
          winner: 'challenger',
          exposedFalsehood: true,
          updatedAt: Date.now(),
        };
      }
      break;

    case 'arbitration':
    case 'deliberation':
      // Move to settlement with current votes
      updatedDispute = tryAdvancePhase(dispute, blockNumber);
      break;

    default:
      throw new StateError('Cannot timeout in current phase', 'INVALID_TRANSITION');
  }

  state.updateDispute(updatedDispute);

  return 100000n; // Timeout processing gas
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
  _params: ConsensusParams
): bigint {
  const payload = tx.payload as CreateVestingPayload;

  // Validate claim exists
  if (!state.isActiveFact(payload.claimId)) {
    throw new StateError('Claim not found or not active', 'FACT_NOT_FOUND');
  }

  // Create vesting schedule
  const vesting = state.createVesting(payload, blockNumber);

  // Lock the reward amount in escrow
  state.transfer('did:lgm:reward-pool', 'did:lgm:vesting-escrow', payload.amount);

  return 80000n; // Vesting creation gas
}

/**
 * Claim available vested rewards
 */
function executeVestingClaim(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams
): bigint {
  const payload = tx.payload as ClaimRewardsPayload;
  const senderDid = publicKeyToDid(tx.sender);

  // Validate sender is the claimer
  if (payload.claimer !== senderDid) {
    throw new StateError('Sender must be the claimer', 'INVALID_TRANSITION');
  }

  // Get vesting and claim rewards
  const { vesting, claimedAmount } = state.claimVestingRewards(
    payload.vestingId,
    senderDid,
    payload.amount,
    blockNumber
  );

  // Transfer claimed amount from escrow to recipient
  state.transfer('did:lgm:vesting-escrow', senderDid, claimedAmount);

  return 50000n; // Vesting claim gas
}

/**
 * Pause vesting when a challenge is initiated
 */
function executeVestingPause(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams
): bigint {
  const payload = tx.payload as PauseVestingPayload;

  // Get vesting
  const vesting = state.getVesting(payload.vestingId);
  if (!vesting) {
    throw new StateError('Vesting not found', 'VESTING_NOT_FOUND');
  }

  // Pause the vesting
  state.pauseVesting(payload.vestingId, payload.relatedEntityId || '', blockNumber);

  return 30000n; // Vesting pause gas
}

/**
 * Resume vesting after challenge resolution
 */
function executeVestingResume(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  _params: ConsensusParams
): bigint {
  const payload = tx.payload as ResumeVestingPayload;

  // Get vesting
  const vesting = state.getVesting(payload.vestingId);
  if (!vesting) {
    throw new StateError('Vesting not found', 'VESTING_NOT_FOUND');
  }

  // Resume with optional defense bonus
  // Note: This is for resuming without challenge resolution
  // Challenge resolution uses vesting_falsify or direct resume
  state.resumeVesting(
    payload.vestingId,
    false, // Not a challenge resolution
    '',
    '',
    blockNumber
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
  _params: ConsensusParams
): bigint {
  const payload = tx.payload as AccelerateVestingPayload;

  // Get vesting
  const vesting = state.getVesting(payload.vestingId);
  if (!vesting) {
    throw new StateError('Vesting not found', 'VESTING_NOT_FOUND');
  }

  // Apply acceleration based on type
  // Currently supports replication acceleration
  if (payload.accelerationType === 'replication' && payload.relatedEntityId) {
    state.accelerateVesting(payload.vestingId, payload.relatedEntityId, blockNumber);
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
  _params: ConsensusParams
): bigint {
  const payload = tx.payload as FalsifyVestingPayload;

  // Get vesting
  const vesting = state.getVesting(payload.vestingId);
  if (!vesting) {
    throw new StateError('Vesting not found', 'VESTING_NOT_FOUND');
  }

  // Process falsification (challenger won the dispute)
  const { vesting: updatedVesting, clawback } = state.resumeVesting(
    payload.vestingId,
    true, // Challenger won
    payload.challengerDid,
    payload.disputeId,
    blockNumber
  );

  // Transfer rewards to challenger
  if (clawback) {
    state.transfer('did:lgm:vesting-escrow', payload.challengerDid, clawback.challengerReward);

    // Apply clawback from recipient if they still have balance
    if (clawback.recipientPenalty > 0n) {
      const recipientBalance = state.getBalance(vesting.recipient);
      const actualClawback = recipientBalance >= clawback.recipientPenalty
        ? clawback.recipientPenalty
        : recipientBalance;

      if (actualClawback > 0n) {
        state.transfer(vesting.recipient, payload.challengerDid, actualClawback);
      }
    }
  }

  return 150000n; // Vesting falsification gas
}

// ============ Verification Result Application ============

/**
 * Apply verification result to state
 * Called after consensus completes a verification round
 */
export function applyVerificationResult(
  state: StateManager,
  result: VerificationResult,
  blockNumber: bigint,
  params: ConsensusParams
): void {
  state.beginTransaction();

  try {
    // Apply rewards to validators
    for (const [validatorKey, reward] of result.rewards) {
      const validatorDid = publicKeyToDid(validatorKey as PublicKey);
      if (state.accountExists(validatorDid)) {
        // Mint reward (comes from block reward pool)
        // In production, this would be more sophisticated
        state.createAccount(validatorDid, '', reward);
      }
    }

    // Apply slashes to validators
    for (const [validatorKey, slashAmount] of result.slashes) {
      state.slashValidator(validatorKey as PublicKey, slashAmount, 'Wrong verification');
    }

    // If claim accepted, add fact to knowledge
    if (result.outcome === 'accepted' && result.fact) {
      state.addFact({
        id: result.claimId,
        claim: result.fact.claim,
        domain: result.fact.domain,
        confidence: result.finalConfidence,
        verifiedAt: blockNumber,
        verifierCount: result.participatingValidators,
        challengeCount: 0,
        status: 'active',
      });
    }

    // If claim rejected, unlock stake and apply penalty
    if (result.outcome === 'rejected') {
      // Penalty would be applied to claim submitter
      // (Need claim submitter info from verification round)
    }

    state.commitTransaction();
  } catch (error) {
    state.rollbackTransaction();
    throw error;
  }
}

/**
 * Apply epoch transition to state
 */
export function applyEpochTransition(
  state: StateManager,
  payload: EpochTransitionPayload,
  params: ConsensusParams
): void {
  state.beginTransaction();

  try {
    // Apply epoch rewards to validators
    for (const [validatorKey, reward] of payload.rewards) {
      const validatorDid = publicKeyToDid(validatorKey as PublicKey);
      // Add reward to validator account
      if (state.accountExists(validatorDid)) {
        state.transfer('did:lgm:reward-pool', validatorDid, reward);
      }
    }

    // Apply slashes
    for (const [validatorKey, slashAmount] of payload.slashes) {
      state.slashValidator(validatorKey as PublicKey, slashAmount, 'Epoch slash');
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
  tx: Transaction
): { valid: boolean; reason?: string } {
  // Check sender exists
  const senderDid = publicKeyToDid(tx.sender);
  if (!state.accountExists(senderDid)) {
    return { valid: false, reason: 'Sender account does not exist' };
  }

  // Check nonce
  const expectedNonce = state.getNonce(senderDid);
  if (tx.nonce !== expectedNonce) {
    return { valid: false, reason: `Invalid nonce: expected ${expectedNonce}, got ${tx.nonce}` };
  }

  // Check balance for fee
  const balance = state.getBalance(senderDid);
  if (balance < tx.fee) {
    return { valid: false, reason: 'Insufficient balance for fee' };
  }

  // Signature validation is done at mempool layer

  return { valid: true };
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
  params: ConsensusParams
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

  // Apply block to state
  const stateRoot = state.applyBlock(blockNumber, ''); // Block hash would be computed

  return {
    stateRoot,
    results,
    totalGasUsed,
    successCount,
    failureCount: transactions.length - successCount,
  };
}

// ============ Helper Functions ============

/**
 * Convert public key to DID
 * Simplified version - in production, use proper DID derivation
 */
function publicKeyToDid(publicKey: PublicKey): string {
  // This should use the proper DID derivation from agents/identity
  return `did:lgm:${publicKey.slice(0, 32)}`;
}

// ============ Types ============

export interface TransitionResult {
  success: boolean;
  stateRoot?: string;
  error?: string;
  gasUsed: bigint;
}

export interface BlockExecutionResult {
  stateRoot: string;
  results: TransitionResult[];
  totalGasUsed: bigint;
  successCount: number;
  failureCount: number;
}

// ============ Gas Costs ============

export const GAS_COSTS = {
  transfer: 21000n,
  stake: 50000n,
  unstake: 50000n,
  delegate: 60000n,
  undelegate: 60000n,
  claimSubmit: 100000n,
  verificationCommit: 30000n,
  verificationReveal: 40000n,
  challenge: 80000n,
  governanceVote: 25000n,
  governancePropose: 150000n,
  // Research transactions
  researchSubmit: 200000n,
  researchChallenge: 120000n,
  bountyCreate: 150000n,
  bountyClaim: 50000n,
  bountyFulfill: 100000n,
  // Dispute transactions
  disputeInitiate: 250000n,
  disputeCommitEvidence: 80000n,
  disputeRevealEvidence: 120000n,
  disputeArbiterVote: 60000n,
  disputeEscalate: 150000n,
  disputeSettle: 200000n,
  disputeTimeout: 100000n,
  // Vesting transactions
  vestingCreate: 80000n,
  vestingClaim: 50000n,
  vestingPause: 30000n,
  vestingResume: 30000n,
  vestingAccelerate: 40000n,
  vestingFalsify: 150000n,
} as const;

/**
 * Estimate gas for a transaction
 */
export function estimateGas(txType: TransactionType): bigint {
  const costs: Record<TransactionType, bigint> = {
    transfer: GAS_COSTS.transfer,
    stake: GAS_COSTS.stake,
    unstake: GAS_COSTS.unstake,
    delegate: GAS_COSTS.delegate,
    undelegate: GAS_COSTS.undelegate,
    claim_submit: GAS_COSTS.claimSubmit,
    verification_commit: GAS_COSTS.verificationCommit,
    verification_reveal: GAS_COSTS.verificationReveal,
    challenge: GAS_COSTS.challenge,
    governance_vote: GAS_COSTS.governanceVote,
    governance_propose: GAS_COSTS.governancePropose,
    // Research transactions
    research_submit: GAS_COSTS.researchSubmit,
    research_challenge: GAS_COSTS.researchChallenge,
    bounty_create: GAS_COSTS.bountyCreate,
    bounty_claim: GAS_COSTS.bountyClaim,
    bounty_fulfill: GAS_COSTS.bountyFulfill,
    // Dispute transactions
    dispute_initiate: GAS_COSTS.disputeInitiate,
    dispute_commit_evidence: GAS_COSTS.disputeCommitEvidence,
    dispute_reveal_evidence: GAS_COSTS.disputeRevealEvidence,
    dispute_arbiter_vote: GAS_COSTS.disputeArbiterVote,
    dispute_escalate: GAS_COSTS.disputeEscalate,
    dispute_settle: GAS_COSTS.disputeSettle,
    dispute_timeout: GAS_COSTS.disputeTimeout,
    // Vesting transactions
    vesting_create: GAS_COSTS.vestingCreate,
    vesting_claim: GAS_COSTS.vestingClaim,
    vesting_pause: GAS_COSTS.vestingPause,
    vesting_resume: GAS_COSTS.vestingResume,
    vesting_accelerate: GAS_COSTS.vestingAccelerate,
    vesting_falsify: GAS_COSTS.vestingFalsify,
  };

  return costs[txType] ?? 21000n;
}
