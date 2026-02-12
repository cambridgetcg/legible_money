/**
 * Agent SDK
 *
 * Agent-friendly wrapper around LegibleClient optimized for LLM tool-use.
 * Provides introspection, decision support, and Agent Home integration.
 *
 * Design Principles (following PP hierarchy):
 * - TRUTH: All responses include verifiable sources/proofs
 * - UNDERSTANDING: Responses explain WHY, not just WHAT
 * - BEAUTY: API feels natural and minimal
 * - JUSTICE: Capabilities placed where agents expect them
 * - CREATIVITY: Enables novel agent workflows
 */

import { LegibleClient, LegibleError } from "./client.js";
import type {
  AgentIdentity,
  AvailableActions,
  AgentBeliefs,
  AgentBelief,
  ActionAdvice,
  ActionPreview,
  ActionExplanation,
  ProposedAction,
  WakeupContext,
  TrustAssessment,
  ConfidentClaimOptions,
  ConfidentClaimResult,
  UncertaintyQueryOptions,
  UncertaintyAwareResult,
  FactWithUncertainty,
  BatchOperation,
  BatchResult,
  AtomicResult,
  ToolDescriptor,
  MethodDescription,
  Alert,
  AgentCapability,
  ActionBlocker,
  SessionLimits,
  BeliefSource,
  StateChange,
} from "./types/agent.js";
import type { FactState } from "../../state/types.js";
import { TOOL_DESCRIPTORS, METHOD_DESCRIPTIONS } from "./tools.js";

// ============ Constants ============

/** Minimum balance required for operations */
const MIN_OPERATIONAL_BALANCE = 1_000_000_000_000_000n; // 0.001 LGM

/** Minimum stake for validator registration */
const MIN_VALIDATOR_STAKE = 111_000_000_000_000_000n; // 0.111 LGM (tier 0 entry)

/** Default stake for claims */
const DEFAULT_CLAIM_STAKE = 1_000_000_000_000_000_000n; // 1 LGM

/** Confidence decay per block */
const CONFIDENCE_DECAY_PER_BLOCK = 0.0001;

// ============ AgentSDK Class ============

/**
 * Agent-friendly SDK for Legible Money
 *
 * Wraps LegibleClient with capabilities optimized for AI agents:
 * - Introspection (whoAmI, whatCanIDo, whatDoIBelieve)
 * - Decision support (shouldI, preview, explain)
 * - Agent Home integration (wakeup, sleep, alerts)
 * - Tool descriptors for LLM function calling
 */
export class AgentSDK {
  private readonly client: LegibleClient;
  private cachedIdentity: AgentIdentity | null = null;
  private lastIdentityFetch: number = 0;
  private readonly identityCacheTTL: number = 30000; // 30 seconds
  private readonly endpoint: string;

  constructor(client: LegibleClient, endpoint?: string) {
    this.client = client;
    // Default endpoint - in production this should come from client config
    this.endpoint = endpoint || "http://localhost:8545";
  }

  /**
   * Helper method for RPC calls
   * Used when we need to make calls that aren't exposed by LegibleClient's public API
   */
  private async rpcCall<T>(method: string, params: unknown): Promise<T> {
    const response = await fetch(this.endpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        jsonrpc: "2.0",
        id: Date.now(),
        method,
        params,
      }),
    });

    const data = (await response.json()) as {
      result?: T;
      error?: { message: string };
    };
    if (data.error) {
      throw new LegibleError(data.error.message, "RPC_ERROR", true);
    }
    return data.result as T;
  }

  // ============ Introspection Methods ============

  /**
   * "Who am I?"
   *
   * Returns comprehensive identity information about this agent.
   * Use this at session start or when you need to understand your current state.
   *
   * @returns Agent identity including DID, balances, reputation, and capabilities
   */
  async whoAmI(): Promise<AgentIdentity> {
    // Check cache
    if (
      this.cachedIdentity &&
      Date.now() - this.lastIdentityFetch < this.identityCacheTTL
    ) {
      return this.cachedIdentity;
    }

    const did = this.client.getDid();
    if (!did) {
      throw new LegibleError(
        "Agent not initialized. Call client.ensureAccount() first.",
        "NOT_INITIALIZED",
        false,
      );
    }

    const publicKey = this.client.getPublicKey();

    // Fetch account status
    const status = await this.client.getAgentStatus(did);
    const balance = await this.client.getBalance(did);

    // Build identity
    const identity: AgentIdentity = {
      did,
      publicKey: publicKey || "",
      reputation: status.account.reputation,
      accuracy: {
        verification:
          status.activity.verificationsCompleted > 0
            ? status.activity.verificationsCorrect /
              status.activity.verificationsCompleted
            : 0,
        claims:
          status.activity.claimsSubmitted > 0
            ? status.activity.claimsAccepted / status.activity.claimsSubmitted
            : 0,
      },
      balance: {
        available: balance,
        staked: BigInt(status.account.stakedBalance),
        vesting: BigInt(status.earnings.pendingRewards),
      },
      capabilities: this.deriveCapabilities(
        balance,
        BigInt(status.account.stakedBalance),
      ),
      activeSessions: status.sessions.active,
    };

    // Check validator status if staked
    if (BigInt(status.account.stakedBalance) >= MIN_VALIDATOR_STAKE) {
      try {
        const validatorInfo = await this.rpcCall<{
          validator: { isActive: boolean; rank?: number };
        }>("validator_getInfo", { publicKey });
        if (validatorInfo.validator) {
          identity.validatorStatus = {
            isActive: validatorInfo.validator.isActive,
            rank: validatorInfo.validator.rank,
            recentBlocks: 0, // Would need additional query
          };
        }
      } catch {
        // Not a validator, ignore
      }
    }

    // Cache the result
    this.cachedIdentity = identity;
    this.lastIdentityFetch = Date.now();

    return identity;
  }

  /**
   * "What can I do?"
   *
   * Returns available actions based on current state.
   * Includes suggestions for what the agent should consider doing.
   *
   * @param context Optional context to evaluate against
   * @returns Available actions with constraints and suggestions
   */
  async whatCanIDo(context?: { balance?: bigint }): Promise<AvailableActions> {
    const identity = await this.whoAmI();
    const balance = context?.balance ?? identity.balance.available;

    const blockers: ActionBlocker[] = [];
    const suggestions: string[] = [];

    // Check transfer capability
    const canTransfer = balance > MIN_OPERATIONAL_BALANCE;
    if (!canTransfer) {
      blockers.push({
        action: "transfer",
        reason: "Insufficient balance for transfers",
        resolution:
          "Request tokens from faucet (testnet) or receive a transfer",
      });
    }

    // Check claim submission capability
    const canSubmitClaim = balance >= DEFAULT_CLAIM_STAKE;
    if (!canSubmitClaim) {
      blockers.push({
        action: "submit_claim",
        reason: `Need at least ${formatLGM(DEFAULT_CLAIM_STAKE)} LGM to stake on claims`,
        resolution: "Acquire more tokens before submitting claims",
      });
    }

    // Check verification capability
    const canVerify = identity.balance.staked >= MIN_VALIDATOR_STAKE;
    if (!canVerify && balance >= MIN_VALIDATOR_STAKE) {
      suggestions.push(
        `You have enough balance to become a validator. Consider staking ${formatLGM(MIN_VALIDATOR_STAKE)} LGM.`,
      );
    }

    // Check challenge capability
    const canChallenge = balance >= DEFAULT_CLAIM_STAKE;

    // Session limits (if using session key)
    let sessionLimits: SessionLimits | null = null;
    const session = this.client.getSession();
    if (session) {
      const limits = session.getRemainingLimits();
      sessionLimits = {
        remainingUsage: limits.remainingUsage,
        remainingDailySpend: limits.remainingDailySpend,
        blocksUntilExpiry: limits.blocksUntilExpiry,
      };

      if (limits.remainingUsage < 10) {
        suggestions.push(
          `Session key running low on usage (${limits.remainingUsage} remaining). Consider creating a new session.`,
        );
      }
    }

    // Proactive suggestions based on state
    if (
      identity.accuracy.verification > 0.8 &&
      identity.validatorStatus?.isActive
    ) {
      suggestions.push(
        "Your verification accuracy is excellent. You may be selected for more verification rounds.",
      );
    }

    if (identity.accuracy.claims < 0.5 && identity.accuracy.claims > 0) {
      suggestions.push(
        "Claim acceptance rate is low. Consider providing more evidence and references for future claims.",
      );
    }

    return {
      canSubmitClaim,
      canVerify,
      canChallenge,
      canTransfer,
      canStake: balance >= MIN_VALIDATOR_STAKE,
      constraints: {
        maxClaimStake: balance / 2n, // Conservative: don't risk more than half
        maxTransfer: balance - MIN_OPERATIONAL_BALANCE,
        sessionLimits,
      },
      suggestions,
      blockers,
    };
  }

  /**
   * "What do I believe?"
   *
   * Returns facts the agent has contributed to or interacted with.
   * Includes confidence levels and uncertainty information.
   *
   * @param options Filter options
   * @returns Agent's belief state
   */
  async whatDoIBelieve(options?: {
    domain?: string;
    minConfidence?: number;
  }): Promise<AgentBeliefs> {
    // Note: identity could be used to filter for agent-specific facts in the future
    const _identity = await this.whoAmI();
    void _identity; // Reserved for future: determining fact source (self_submitted vs protocol)

    // Get facts where agent is involved
    // This queries facts the agent submitted or verified
    const facts = await this.client.searchFacts({
      domain: options?.domain,
      minConfidence: options?.minConfidence ?? 0.3,
      limit: 100,
    });

    const currentBlock = await this.client.getBlockNumber();
    const beliefs: AgentBelief[] = [];
    const uncertainties: {
      factId: string;
      statement: string;
      currentConfidence: number;
      effectiveConfidence: number;
      reason: "decay" | "challenged" | "low_initial" | "contradicted";
      recommendation: "verify" | "monitor" | "discard";
    }[] = [];

    for (const fact of facts) {
      // Calculate effective confidence with decay
      // Use lastReverifiedAt if available, otherwise use verifiedAt
      const lastVerifiedBlock = Number(
        fact.lastReverifiedAt ?? fact.verifiedAt,
      );
      const age = Number(currentBlock) - lastVerifiedBlock;
      const decay = age * CONFIDENCE_DECAY_PER_BLOCK;
      const effectiveConfidence = Math.max(0, fact.confidence - decay);

      // Determine source - FactState doesn't track submitter directly
      // Default to 'protocol' for verified facts
      const source: BeliefSource = "protocol";
      // Would need additional query to determine if agent submitted or verified this fact

      if (effectiveConfidence >= 0.6) {
        beliefs.push({
          factId: fact.id,
          statement: fact.claimText,
          confidence: fact.confidence,
          source,
          domain: fact.domain,
          lastUpdated: lastVerifiedBlock,
          decayRate: CONFIDENCE_DECAY_PER_BLOCK,
        });
      } else {
        // Uncertain fact
        let reason: "decay" | "challenged" | "low_initial" | "contradicted" =
          "decay";
        let recommendation: "verify" | "monitor" | "discard" = "monitor";

        if (fact.confidence < 0.5) {
          reason = "low_initial";
          recommendation = "verify";
        } else if (decay > 0.3) {
          reason = "decay";
          recommendation = "verify";
        }

        if (effectiveConfidence < 0.3) {
          recommendation = "discard";
        }

        uncertainties.push({
          factId: fact.id,
          statement: fact.claimText,
          currentConfidence: fact.confidence,
          effectiveConfidence,
          reason,
          recommendation,
        });
      }
    }

    // Get unique domains
    const domains = [...new Set(beliefs.map((b) => b.domain))];

    return {
      beliefs,
      uncertainties,
      summary: {
        totalBeliefs: beliefs.length,
        averageConfidence:
          beliefs.length > 0
            ? beliefs.reduce((sum, b) => sum + b.confidence, 0) / beliefs.length
            : 0,
        domainsRepresented: domains,
      },
    };
  }

  // ============ Decision Support Methods ============

  /**
   * "Should I do this?"
   *
   * Provides advice about whether an action is recommended.
   * Considers current state, risks, and alternatives.
   *
   * @param action Proposed action to evaluate
   * @returns Advice including recommendation, risks, and alternatives
   */
  async shouldI(action: ProposedAction): Promise<ActionAdvice> {
    const preview = await this.preview(action);
    const identity = await this.whoAmI();

    const risks: string[] = [];
    const alternatives: ProposedAction[] = [];
    const prerequisites: string[] = [];

    // Evaluate based on action type
    switch (action.action) {
      case "transfer": {
        const params = action.params as { to: string; amount: bigint | string };
        const amount =
          typeof params.amount === "string"
            ? BigInt(params.amount)
            : params.amount;

        if (amount > identity.balance.available / 2n) {
          risks.push("Transfer amount exceeds 50% of available balance");
          alternatives.push({
            action: "transfer",
            params: { ...params, amount: identity.balance.available / 4n },
          });
        }
        break;
      }

      case "submit_claim": {
        const params = action.params as {
          claim: string;
          domain: string;
          stake: bigint | string;
        };
        const stake =
          typeof params.stake === "string"
            ? BigInt(params.stake)
            : params.stake;

        // Check for potential contradictions
        const contradictions = await this.client.checkContradictions(
          params.claim,
          params.domain,
        );
        if (contradictions.length > 0) {
          risks.push(
            `Claim may contradict ${contradictions.length} existing fact(s)`,
          );
        }

        if (stake > identity.balance.available / 4n) {
          risks.push("Stake amount is high relative to balance");
          alternatives.push({
            action: "submit_claim",
            params: { ...params, stake: identity.balance.available / 8n },
          });
        }

        if (identity.accuracy.claims < 0.5 && identity.accuracy.claims > 0) {
          risks.push("Your claim acceptance rate is below 50%");
          prerequisites.push(
            "Consider reviewing rejected claims to understand verification requirements",
          );
        }
        break;
      }

      case "stake": {
        const params = action.params as { amount: bigint | string };
        const amount =
          typeof params.amount === "string"
            ? BigInt(params.amount)
            : params.amount;

        if (amount < MIN_VALIDATOR_STAKE) {
          prerequisites.push(
            `Minimum stake is ${formatLGM(MIN_VALIDATOR_STAKE)} LGM to become a validator`,
          );
        }

        if (amount > (identity.balance.available * 9n) / 10n) {
          risks.push(
            "Staking more than 90% of balance leaves little for operations",
          );
        }
        break;
      }

      case "challenge": {
        if (identity.reputation < 0.5) {
          risks.push("Low reputation may reduce challenge credibility");
        }
        break;
      }
    }

    // Generate reasoning
    let reasoning: string;
    let recommended: boolean;
    let confidence: number;

    if (preview.blockers.length > 0) {
      recommended = false;
      confidence = 0.9;
      reasoning = `Action would fail: ${preview.blockers.join(", ")}`;
    } else if (risks.length === 0 && preview.warnings.length === 0) {
      recommended = true;
      confidence = 0.8;
      reasoning = "Action appears safe with no identified risks";
    } else if (risks.length > 2 || preview.warnings.length > 2) {
      recommended = false;
      confidence = 0.7;
      reasoning = `Multiple risks identified: ${[...risks, ...preview.warnings].slice(0, 3).join("; ")}`;
    } else {
      recommended = true;
      confidence = 0.6;
      reasoning = `Action has some risks (${risks.length}) but may be worthwhile. Consider: ${risks[0] || preview.warnings[0]}`;
    }

    return {
      recommended,
      confidence,
      reasoning,
      risks: [...risks, ...preview.warnings],
      alternatives,
      prerequisites,
    };
  }

  /**
   * "What would happen if I do this?"
   *
   * Previews the outcome of an action without executing it.
   * Estimates costs, rewards, and state changes.
   *
   * @param action Proposed action to preview
   * @returns Preview of what would happen
   */
  async preview(action: ProposedAction): Promise<ActionPreview> {
    const identity = await this.whoAmI();
    const warnings: string[] = [];
    const blockers: string[] = [];
    const stateChanges: StateChange[] = [];
    let estimatedCost = 0n;
    let estimatedReward: bigint | undefined;

    const baseFee = 1_000_000_000_000_000n; // 0.001 LGM base fee

    switch (action.action) {
      case "transfer": {
        const params = action.params as { to: string; amount: bigint | string };
        const amount =
          typeof params.amount === "string"
            ? BigInt(params.amount)
            : params.amount;

        estimatedCost = baseFee;

        if (amount + baseFee > identity.balance.available) {
          blockers.push("Insufficient balance for transfer + fee");
        }

        stateChanges.push({
          entity: identity.did,
          field: "balance",
          before: identity.balance.available,
          after: identity.balance.available - amount - baseFee,
        });

        stateChanges.push({
          entity: params.to,
          field: "balance",
          before: "unknown",
          after: `+${formatLGM(amount)} LGM`,
        });
        break;
      }

      case "submit_claim": {
        const params = action.params as {
          claim: string;
          domain: string;
          stake: bigint | string;
        };
        const stake =
          typeof params.stake === "string"
            ? BigInt(params.stake)
            : params.stake;

        estimatedCost = baseFee + stake; // Stake is locked during verification

        if (estimatedCost > identity.balance.available) {
          blockers.push("Insufficient balance for stake + fee");
        }

        // Estimate potential reward
        estimatedReward = (stake * 15n) / 100n; // ~15% reward if accepted

        stateChanges.push({
          entity: identity.did,
          field: "stakedOnClaims",
          before: 0n,
          after: stake,
        });

        if (params.claim.length < 20) {
          warnings.push("Claim is very short, may lack specificity");
        }
        break;
      }

      case "stake": {
        const params = action.params as { amount: bigint | string };
        const amount =
          typeof params.amount === "string"
            ? BigInt(params.amount)
            : params.amount;

        estimatedCost = baseFee;

        if (amount + baseFee > identity.balance.available) {
          blockers.push("Insufficient balance to stake");
        }

        if (amount < MIN_VALIDATOR_STAKE) {
          warnings.push(
            `Amount below minimum validator stake (${formatLGM(MIN_VALIDATOR_STAKE)} LGM)`,
          );
        }

        stateChanges.push({
          entity: identity.did,
          field: "balance.available",
          before: identity.balance.available,
          after: identity.balance.available - amount - baseFee,
        });

        stateChanges.push({
          entity: identity.did,
          field: "balance.staked",
          before: identity.balance.staked,
          after: identity.balance.staked + amount,
        });
        break;
      }

      case "unstake": {
        const params = action.params as { amount: bigint | string };
        const amount =
          typeof params.amount === "string"
            ? BigInt(params.amount)
            : params.amount;

        estimatedCost = baseFee;

        if (amount > identity.balance.staked) {
          blockers.push("Cannot unstake more than currently staked");
        }

        if (
          identity.balance.staked - amount < MIN_VALIDATOR_STAKE &&
          identity.validatorStatus?.isActive
        ) {
          warnings.push("Unstaking will deactivate validator status");
        }

        stateChanges.push({
          entity: identity.did,
          field: "balance.staked",
          before: identity.balance.staked,
          after: identity.balance.staked - amount,
        });
        break;
      }

      case "challenge": {
        const params = action.params as {
          factId: string;
          stake: bigint | string;
        };
        const stake =
          typeof params.stake === "string"
            ? BigInt(params.stake)
            : params.stake;

        estimatedCost = baseFee + stake;
        estimatedReward = stake * 2n; // If challenge succeeds

        if (estimatedCost > identity.balance.available) {
          blockers.push("Insufficient balance for challenge stake + fee");
        }

        warnings.push(
          "Challenge requires evidence and may damage reputation if unsuccessful",
        );
        break;
      }

      case "verify": {
        estimatedCost = baseFee / 10n; // Verification has lower fees
        estimatedReward = baseFee * 5n; // Base verification reward
        break;
      }
    }

    return {
      wouldSucceed: blockers.length === 0,
      estimatedCost,
      estimatedReward,
      stateChanges,
      warnings,
      blockers,
    };
  }

  /**
   * "What happened?"
   *
   * Explains the outcome of a transaction.
   *
   * @param txHash Transaction hash to explain
   * @returns Explanation of what happened
   */
  async explain(txHash: string): Promise<ActionExplanation> {
    // Get transaction status
    const status = await this.rpcCall<{
      status: string;
      blockNumber?: string;
      error?: string;
      type?: string;
    }>("tx_getStatus", { txHash });

    const effects: {
      type:
        | "balance_change"
        | "state_update"
        | "event_emitted"
        | "fact_created"
        | "round_started";
      description: string;
      data?: unknown;
    }[] = [];
    const nextSteps: string[] = [];

    // Determine outcome
    let outcome: "success" | "failure" | "pending";
    let reasoning: string;
    const action: string = status.type || "unknown";

    switch (status.status) {
      case "finalized":
        outcome = "success";
        reasoning = `Transaction was included in block ${status.blockNumber} and is now finalized.`;
        effects.push({
          type: "state_update",
          description: `${action} completed successfully`,
        });
        break;

      case "included":
        outcome = "pending";
        reasoning = `Transaction included in block ${status.blockNumber} but not yet finalized (need 19 confirmations).`;
        nextSteps.push(
          "Wait for finalization before relying on this transaction",
        );
        break;

      case "pending":
        outcome = "pending";
        reasoning =
          "Transaction is in the mempool waiting to be included in a block.";
        nextSteps.push("Transaction should be included within a few blocks");
        break;

      case "failed":
        outcome = "failure";
        reasoning = status.error || "Transaction execution failed";
        nextSteps.push(
          "Review the error and adjust parameters before retrying",
        );
        break;

      default:
        outcome = "pending";
        reasoning = `Unknown status: ${status.status}`;
    }

    return {
      action,
      outcome,
      effects,
      reasoning,
      nextSteps,
      txHash,
    };
  }

  // ============ Agent Home Integration ============

  /**
   * "Wake up" - Load context at session start
   *
   * Retrieves agent identity, memory, treasury status, and alerts.
   * Call this at the beginning of each session.
   *
   * @returns Full context for starting a session
   */
  async wakeup(): Promise<WakeupContext> {
    const identity = await this.whoAmI();

    // Get pending verifications as alerts
    const alerts: Alert[] = [];

    try {
      const pendingVerifications = await this.client.getPendingVerifications();
      for (const assignment of pendingVerifications) {
        alerts.push({
          id: `verify-${assignment.roundId}`,
          type: "verification_assigned",
          severity:
            assignment.commitDeadline -
              Number(await this.client.getBlockNumber()) <
            10
              ? "urgent"
              : "info",
          message: `Verification assigned: ${assignment.claim.slice(0, 50)}...`,
          actionRequired: true,
          deadline: assignment.commitDeadline,
          data: assignment,
        });
      }
    } catch {
      // May not be a validator
    }

    // Check for expiring tokens (placeholder - would need expirable token queries)
    // Check for low balance
    if (identity.balance.available < MIN_OPERATIONAL_BALANCE * 10n) {
      alerts.push({
        id: "low-balance",
        type: "reward_available", // Using as general alert
        severity: "warning",
        message: `Balance is low: ${formatLGM(identity.balance.available)} LGM`,
        actionRequired: false,
      });
    }

    // Generate suggestions
    const suggestions: string[] = [];
    const actions = await this.whatCanIDo();
    suggestions.push(...actions.suggestions);

    if (alerts.filter((a) => a.type === "verification_assigned").length > 0) {
      suggestions.push(
        `You have ${alerts.filter((a) => a.type === "verification_assigned").length} pending verification(s) to complete`,
      );
    }

    return {
      identity,
      memory: {
        recentContributions: [], // Would load from Agent Home
        pendingDisputes: [],
        relationships: [],
        customData: {},
      },
      treasury: {
        balance: identity.balance.available,
        pendingVesting: [],
        expiringTokens: [],
        recentSpending: [],
      },
      alerts,
      suggestions,
    };
  }

  /**
   * "Go to sleep" - Save session context
   *
   * Records session summary to Agent Home for continuity.
   *
   * @param sessionSummary Optional summary of what was done
   */
  async sleep(sessionSummary?: string): Promise<void> {
    // This would store the session summary in Agent Home
    // For now, we just log it
    if (sessionSummary) {
      console.log(`[AgentSDK] Session ended. Summary: ${sessionSummary}`);
    }

    // Clear cached identity
    this.cachedIdentity = null;
    this.lastIdentityFetch = 0;
  }

  /**
   * Get pending alerts
   *
   * @returns Current alerts requiring attention
   */
  async getAlerts(): Promise<Alert[]> {
    const context = await this.wakeup();
    return context.alerts;
  }

  /**
   * Acknowledge an alert
   *
   * @param alertId ID of alert to acknowledge
   */
  async acknowledgeAlert(alertId: string): Promise<void> {
    // Would persist acknowledgment to Agent Home
    console.log(`[AgentSDK] Alert ${alertId} acknowledged`);
  }

  // ============ Knowledge Helpers ============

  /**
   * Submit a claim with epistemic awareness
   *
   * Validates confidence calibration and provides warnings.
   *
   * @param options Claim options with epistemic metadata
   * @returns Submission result with calibration feedback
   */
  async submitClaimWithConfidence(
    options: ConfidentClaimOptions,
  ): Promise<ConfidentClaimResult> {
    const calibrationWarnings: string[] = [];
    let stakeAdjusted = false;
    let adjustedStake = options.stake;

    // Validate confidence by epistemic category
    const maxConfidenceByCategory: Record<string, number> = {
      analytic: 1.0, // Logical truths can have certainty
      formal: 0.99, // Formal proofs very high
      protocol: 0.95, // On-chain facts very reliable
      empirical: 0.85, // Observable facts good confidence
      historical: 0.8, // Past events may have uncertainty
      predictive: 0.6, // Predictions inherently uncertain
      social: 0.7, // Social consensus variable
    };

    const maxConfidence =
      maxConfidenceByCategory[options.epistemicCategory] || 0.8;

    if (options.myConfidence > maxConfidence) {
      calibrationWarnings.push(
        `Confidence ${options.myConfidence} may be miscalibrated for ${options.epistemicCategory} claims. Max suggested: ${maxConfidence}`,
      );
    }

    // Adjust stake based on confidence
    if (options.myConfidence < 0.5) {
      calibrationWarnings.push(
        "Low confidence suggests lower stake may be appropriate",
      );
      const suggestedStake = options.stake / 2n;
      if (suggestedStake >= DEFAULT_CLAIM_STAKE) {
        adjustedStake = suggestedStake;
        stakeAdjusted = true;
      }
    }

    // Check evidence quality
    if (options.evidence.length === 0) {
      calibrationWarnings.push(
        "No evidence provided - consider adding supporting references",
      );
    }

    const unverifiableEvidence = options.evidence.filter((e) => !e.verifiable);
    if (unverifiableEvidence.length > options.evidence.length / 2) {
      calibrationWarnings.push(
        "More than half of evidence is not independently verifiable",
      );
    }

    // Submit the claim
    const result = await this.client.submitClaim({
      claim: options.statement,
      domain: options.domain,
      stake: adjustedStake,
      references: options.references,
    });

    const recommendations: string[] = [];
    if (calibrationWarnings.length > 0) {
      recommendations.push("Review confidence calibration for future claims");
    }
    if (options.evidence.length < 3) {
      recommendations.push(
        "Consider providing more evidence for stronger claims",
      );
    }

    return {
      claimId: result.claimId,
      verificationRoundId: result.verificationRoundId,
      txHash: result.txHash,
      calibrationWarnings,
      stakeAdjusted,
      recommendations,
    };
  }

  /**
   * Query facts with uncertainty awareness
   *
   * Returns facts with effective confidence accounting for decay.
   *
   * @param options Query options
   * @returns Facts with uncertainty information
   */
  async queryWithUncertainty(
    options: UncertaintyQueryOptions = {},
  ): Promise<UncertaintyAwareResult> {
    const startTime = Date.now();
    const currentBlock = await this.client.getBlockNumber();

    const facts = await this.client.searchFacts({
      domain: options.domain,
      search: options.query,
      minConfidence: 0.1, // Get all, filter ourselves
      limit: options.limit || 50,
    });

    const factsWithUncertainty: FactWithUncertainty[] = [];
    const factsNearThreshold: FactState[] = [];
    const gaps: string[] = [];
    let filteredCount = 0;

    for (const fact of facts) {
      const lastVerifiedBlock = Number(
        fact.lastReverifiedAt ?? fact.verifiedAt,
      );
      const age = Number(currentBlock) - lastVerifiedBlock;
      const decayRate = CONFIDENCE_DECAY_PER_BLOCK;
      const decay = age * decayRate;
      const effectiveConfidence = Math.max(0, fact.confidence - decay);

      // Determine maturity
      let maturity: "emerging" | "established" | "canonical";
      if (age < 1000) {
        maturity = "emerging";
      } else if (fact.confidence >= 0.9 && age > 10000) {
        maturity = "canonical";
      } else {
        maturity = "established";
      }

      // Determine risk
      let riskAssessment: "stable" | "monitoring" | "at_risk";
      if (effectiveConfidence >= 0.7) {
        riskAssessment = "stable";
      } else if (effectiveConfidence >= 0.4) {
        riskAssessment = "monitoring";
      } else {
        riskAssessment = "at_risk";
      }

      // Track facts near threshold
      if (effectiveConfidence >= 0.55 && effectiveConfidence <= 0.65) {
        factsNearThreshold.push(fact);
      }

      // Filter by minimum confidence if specified
      if (
        options.minEffectiveConfidence &&
        effectiveConfidence < options.minEffectiveConfidence
      ) {
        filteredCount++;
        continue;
      }

      if (!options.includeDecayed && effectiveConfidence < 0.3) {
        filteredCount++;
        continue;
      }

      factsWithUncertainty.push({
        fact,
        effectiveConfidence,
        decayRate,
        maturity,
        riskAssessment,
      });
    }

    // Check for contradictions among results
    const contradictions: {
      fact1Id: string;
      fact2Id: string;
      nature: string;
      severity: "direct" | "indirect" | "potential";
    }[] = [];
    // Would implement contradiction detection here

    // Detect gaps
    if (factsWithUncertainty.length === 0 && options.query) {
      gaps.push(`No facts found matching "${options.query}"`);
    }

    // Calculate overall confidence
    const avgConfidence =
      factsWithUncertainty.length > 0
        ? factsWithUncertainty.reduce(
            (sum, f) => sum + f.effectiveConfidence,
            0,
          ) / factsWithUncertainty.length
        : 0;

    return {
      facts: factsWithUncertainty,
      uncertainties: {
        factsNearDecayThreshold: factsNearThreshold,
        contradictions,
        gaps,
      },
      confidence: avgConfidence,
      metadata: {
        totalMatches: facts.length,
        filteredByConfidence: filteredCount,
        queryTime: Date.now() - startTime,
      },
    };
  }

  /**
   * "Can I trust this fact?"
   *
   * Assesses reliability of a specific fact.
   *
   * @param factId ID of fact to assess
   * @returns Trust assessment with recommendation
   */
  async canITrust(factId: string): Promise<TrustAssessment> {
    const fact = await this.client.getFact(factId);
    if (!fact) {
      throw new LegibleError(
        `Fact ${factId} not found`,
        "FACT_NOT_FOUND",
        false,
      );
    }

    const currentBlock = await this.client.getBlockNumber();
    const lastVerifiedBlock = Number(fact.lastReverifiedAt ?? fact.verifiedAt);
    const age = Number(currentBlock) - lastVerifiedBlock;
    const decay = age * CONFIDENCE_DECAY_PER_BLOCK;
    const effectiveConfidence = Math.max(0, fact.confidence - decay);

    // Determine maturity
    let maturity: "emerging" | "established" | "canonical";
    if (age < 1000) {
      maturity = "emerging";
    } else if (fact.confidence >= 0.9 && age > 10000) {
      maturity = "canonical";
    } else {
      maturity = "established";
    }

    // Estimate challenge risk
    let challengeRisk = 0;
    if (effectiveConfidence < 0.5) {
      challengeRisk = 0.4;
    } else if (effectiveConfidence < 0.7) {
      challengeRisk = 0.2;
    } else if (maturity === "emerging") {
      challengeRisk = 0.15;
    } else {
      challengeRisk = 0.05;
    }

    // Determine recommendation
    let recommendation: "trust" | "verify_first" | "avoid";
    let reasoning: string;

    if (effectiveConfidence >= 0.8 && maturity !== "emerging") {
      recommendation = "trust";
      reasoning = "High confidence, established fact - safe to rely on";
    } else if (effectiveConfidence >= 0.6) {
      recommendation = "trust";
      reasoning = "Good confidence level - generally reliable";
    } else if (effectiveConfidence >= 0.4) {
      recommendation = "verify_first";
      reasoning =
        "Moderate confidence - verify independently for critical decisions";
    } else {
      recommendation = "avoid";
      reasoning =
        "Low effective confidence - high uncertainty, may not be reliable";
    }

    return {
      currentConfidence: fact.confidence,
      effectiveConfidence,
      maturity,
      lastVerified: lastVerifiedBlock,
      challengeRisk,
      recommendation,
      reasoning,
      verificationCount: fact.verifierCount,
    };
  }

  // ============ Batch Operations ============

  /**
   * Execute multiple operations in sequence
   *
   * @param operations Operations to execute
   * @returns Results for each operation
   */
  async batch(operations: BatchOperation[]): Promise<BatchResult> {
    const results: {
      id: string;
      operation: ProposedAction;
      success: boolean;
      result?: unknown;
      error?: string;
      txHash?: string;
    }[] = [];
    let totalCost = 0n;

    for (const op of operations) {
      // Check dependencies
      if (op.dependsOn) {
        const dependencyFailed = op.dependsOn.some((depId) => {
          const depResult = results.find((r) => r.id === depId);
          return depResult && !depResult.success;
        });

        if (dependencyFailed) {
          results.push({
            id: op.id,
            operation: op.action,
            success: false,
            error: "Dependency failed",
          });
          continue;
        }
      }

      try {
        const result = await this.executeAction(op.action);
        results.push({
          id: op.id,
          operation: op.action,
          success: true,
          result,
          txHash:
            typeof result === "object" && result !== null && "txHash" in result
              ? (result as { txHash: string }).txHash
              : undefined,
        });
        totalCost += result.cost || 0n;
      } catch (error) {
        results.push({
          id: op.id,
          operation: op.action,
          success: false,
          error: error instanceof Error ? error.message : String(error),
        });
      }
    }

    return {
      results,
      summary: {
        succeeded: results.filter((r) => r.success).length,
        failed: results.filter(
          (r) => !r.success && r.error !== "Dependency failed",
        ).length,
        skipped: results.filter((r) => r.error === "Dependency failed").length,
        totalCost,
      },
    };
  }

  /**
   * Execute operations atomically (all or nothing)
   *
   * Previews all operations first, only executes if all would succeed.
   *
   * @param operations Operations to execute
   * @returns Atomic result
   */
  async atomic(operations: BatchOperation[]): Promise<AtomicResult> {
    // Preview all operations first
    for (const op of operations) {
      const preview = await this.preview(op.action);
      if (!preview.wouldSucceed) {
        return {
          success: false,
          error: `Operation ${op.id} would fail: ${preview.blockers.join(", ")}`,
          rollbackReason: "Preview indicated failure",
        };
      }
    }

    // All previews passed, execute
    const batchResult = await this.batch(operations);

    // Check if any failed
    const failed = batchResult.results.find((r) => !r.success);
    if (failed) {
      // In a real implementation, we'd rollback here
      return {
        success: false,
        results: batchResult.results,
        error: `Operation ${failed.id} failed: ${failed.error}`,
        rollbackReason: "Execution failed after preview passed",
      };
    }

    return {
      success: true,
      results: batchResult.results,
    };
  }

  // ============ Tool Descriptors ============

  /**
   * Get all available tools for LLM function calling
   *
   * Returns tool descriptors compatible with Claude's tool use format.
   *
   * @returns Array of tool descriptors
   */
  getTools(): ToolDescriptor[] {
    return TOOL_DESCRIPTORS;
  }

  /**
   * Get detailed description of a method
   *
   * @param methodName Name of method to describe
   * @returns Detailed method description
   */
  describeMethod(methodName: string): MethodDescription | null {
    return METHOD_DESCRIPTIONS[methodName] || null;
  }

  // ============ Private Helpers ============

  /**
   * Derive capabilities from state
   */
  private deriveCapabilities(
    balance: bigint,
    staked: bigint,
  ): AgentCapability[] {
    const capabilities: AgentCapability[] = [];

    if (balance > MIN_OPERATIONAL_BALANCE) {
      capabilities.push("transfer");
    }

    if (balance >= DEFAULT_CLAIM_STAKE) {
      capabilities.push("submit_claim");
      capabilities.push("challenge");
    }

    if (balance >= MIN_VALIDATOR_STAKE) {
      capabilities.push("stake");
    }

    if (staked >= MIN_VALIDATOR_STAKE) {
      capabilities.push("verify");
      capabilities.push("unstake");
    }

    capabilities.push("create_session");
    capabilities.push("manage_home");

    return capabilities;
  }

  /**
   * Execute a proposed action
   */
  private async executeAction(
    action: ProposedAction,
  ): Promise<{ txHash?: string; cost?: bigint; [key: string]: unknown }> {
    switch (action.action) {
      case "transfer": {
        const params = action.params as {
          to: string;
          amount: bigint | string;
          memo?: string;
        };
        const txHash = await this.client.transfer({
          to: params.to,
          amount:
            typeof params.amount === "string"
              ? BigInt(params.amount)
              : params.amount,
          memo: params.memo,
        });
        return { txHash, cost: 1_000_000_000_000_000n };
      }

      case "stake": {
        const params = action.params as { amount: bigint | string };
        const txHash = await this.client.stake(
          typeof params.amount === "string"
            ? BigInt(params.amount)
            : params.amount,
        );
        return { txHash, cost: 1_000_000_000_000_000n };
      }

      case "unstake": {
        const params = action.params as { amount: bigint | string };
        const txHash = await this.client.unstake(
          typeof params.amount === "string"
            ? BigInt(params.amount)
            : params.amount,
        );
        return { txHash, cost: 1_000_000_000_000_000n };
      }

      case "submit_claim": {
        const params = action.params as {
          claim: string;
          domain: string;
          stake: bigint | string;
          references?: string[];
        };
        const result = await this.client.submitClaim({
          claim: params.claim,
          domain: params.domain,
          stake:
            typeof params.stake === "string"
              ? BigInt(params.stake)
              : params.stake,
          references: params.references,
        });
        return {
          txHash: result.txHash,
          claimId: result.claimId,
          cost: 1_000_000_000_000_000n,
        };
      }

      default:
        throw new LegibleError(
          `Unknown action: ${action.action}`,
          "UNKNOWN_ACTION",
          false,
        );
    }
  }
}

// ============ Helpers ============

/**
 * Format LGM amount for display
 */
function formatLGM(wei: bigint): string {
  const lgm = Number(wei) / 1e18;
  return lgm.toLocaleString(undefined, { maximumFractionDigits: 4 });
}

// ============ Exports ============

export type { LegibleClient };
