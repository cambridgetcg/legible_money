/**
 * Evaluation Daemon
 *
 * Orchestrates the Claude-powered validator evaluation lifecycle:
 * block → fetch rounds → evaluate claims → commit → reveal.
 *
 * Follows patterns from devnet/cmd/node.ts (EventEmitter, create/start/stop)
 * and protocol/sdk/client.ts:467-534 (startValidator loop) but with
 * persistent commitments and Claude AI evaluation.
 */

import { EventEmitter } from "node:events";
import type { DaemonConfig } from "./config.js";
import type {
  BlockEvent,
  DaemonStats,
  DaemonStatus,
  ActiveRound,
} from "./types.js";
import { RPCClient } from "./rpc_client.js";
import { RoundManager } from "./round_manager.js";
import {
  CommitmentStore,
  computeCommitmentHash,
  generateSalt,
} from "./commitment_store.js";
import { ClaudeEvaluator } from "./evaluator.js";
import { SessionManager } from "./session_manager.js";

/** Max age for commitment cleanup (in blocks) */
const COMMITMENT_MAX_AGE = 100n;

type LogLevel = "debug" | "info" | "warn" | "error";
const LOG_LEVELS: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
};

export class EvaluationDaemon extends EventEmitter {
  private readonly config: DaemonConfig;
  private readonly rpc: RPCClient;
  private readonly rounds: RoundManager;
  private readonly commitments: CommitmentStore;
  private readonly evaluator: ClaudeEvaluator;
  private readonly session: SessionManager;

  private status: DaemonStatus = "stopped";
  private currentHeight = 0n;
  private startTime = 0;
  private processing = false;

  // Stats
  private roundsProcessed = 0;
  private revealsSubmitted = 0;
  private claudeEvaluations = 0;
  private deterministicFallbacks = 0;
  private errorCount = 0;

  private constructor(
    config: DaemonConfig,
    rpc: RPCClient,
    rounds: RoundManager,
    commitments: CommitmentStore,
    evaluator: ClaudeEvaluator,
    session: SessionManager,
  ) {
    super();
    this.config = config;
    this.rpc = rpc;
    this.rounds = rounds;
    this.commitments = commitments;
    this.evaluator = evaluator;
    this.session = session;
  }

  /**
   * Create a new daemon instance (factory pattern from devnet/cmd/node.ts).
   */
  static async create(config: DaemonConfig): Promise<EvaluationDaemon> {
    const rpc = new RPCClient(config.node.rpcEndpoint, config.node.wsEndpoint);
    const rounds = new RoundManager(rpc, config.validator.validatorAddress);
    const commitments = new CommitmentStore(config.storage.dataDir);
    const evaluator = new ClaudeEvaluator({
      model: config.claude.model,
      maxTokens: config.claude.maxTokens,
      timeoutMs: config.claude.timeoutMs,
      temperature: config.claude.temperature,
    });
    const session = new SessionManager(
      rpc,
      config.validator.agentDid,
      config.validator.operationalKeyPath,
    );

    return new EvaluationDaemon(
      config,
      rpc,
      rounds,
      commitments,
      evaluator,
      session,
    );
  }

  /**
   * Start the daemon: connect WebSocket, subscribe to blocks, begin loop.
   */
  async start(): Promise<void> {
    if (this.status !== "stopped") {
      throw new Error(`Cannot start daemon in ${this.status} state`);
    }

    this.status = "starting";
    this.startTime = Date.now();
    this.log("info", "Starting evaluation daemon...");
    this.log("info", `Validator: ${this.config.validator.validatorAddress}`);
    this.log("info", `Agent DID: ${this.config.validator.agentDid}`);
    this.log("info", `Claude model: ${this.config.claude.model}`);
    this.log("info", `Data dir: ${this.config.storage.dataDir}`);

    // Check for pending commitments from previous session
    const pending = this.commitments.count();
    if (pending > 0) {
      this.log(
        "info",
        `Found ${pending} pending commitments from previous session`,
      );
    }

    // Connect WebSocket and subscribe to blocks
    this.rpc.on("block", (block: BlockEvent) => this.onBlock(block));
    this.rpc.on("error", (err: Error) => {
      this.errorCount++;
      this.log("error", `RPC error: ${err.message}`);
    });

    await this.rpc.connect();
    this.status = "running";
    this.log("info", "Daemon running — listening for blocks");
    this.emit("started");
  }

  /**
   * Stop the daemon gracefully. Flushes commitment store.
   */
  async stop(): Promise<void> {
    if (this.status !== "running") return;

    this.status = "stopping";
    this.log("info", "Stopping evaluation daemon...");

    // Wait for current processing to finish
    let waitCount = 0;
    while (this.processing && waitCount < 20) {
      await sleep(250);
      waitCount++;
    }

    this.rpc.disconnect();
    this.status = "stopped";
    this.log(
      "info",
      `Daemon stopped. Commitments on disk: ${this.commitments.count()}`,
    );
    this.emit("stopped");
  }

  /** Get current daemon statistics */
  getStats(): DaemonStats {
    return {
      status: this.status,
      currentHeight: this.currentHeight,
      roundsProcessed: this.roundsProcessed,
      commitmentsStored: this.commitments.count(),
      revealsSubmitted: this.revealsSubmitted,
      claudeEvaluations: this.claudeEvaluations,
      deterministicFallbacks: this.deterministicFallbacks,
      errors: this.errorCount,
      uptime: this.startTime > 0 ? Date.now() - this.startTime : 0,
    };
  }

  // ============ Core Loop ============

  /**
   * Handle a new block event.
   * Core loop mirrors client.ts:467-534 but with persistence + Claude.
   */
  private async onBlock(block: BlockEvent): Promise<void> {
    if (this.status !== "running" || this.processing) return;

    this.processing = true;
    this.currentHeight = block.height;
    this.session.updateHeight(block.height);

    try {
      // 1. Process commit phase rounds
      await this.processCommitPhase(block.height);

      // 2. Process reveal phase rounds
      await this.processRevealPhase(block.height);

      // 3. Cleanup expired data
      this.commitments.cleanupExpired(block.height, COMMITMENT_MAX_AGE);
      this.rounds.cleanupOld(block.height);

      // 4. Session key rotation check (handled automatically by SessionManager)
      await this.session.getSessionKey(block.height);
    } catch (err) {
      this.errorCount++;
      this.log(
        "error",
        `Block ${block.height} processing error: ${(err as Error).message}`,
      );
    } finally {
      this.processing = false;
    }
  }

  /**
   * Process all commit-phase rounds: evaluate → commit.
   */
  private async processCommitPhase(height: bigint): Promise<void> {
    const commitRounds = await this.rounds.getCommitRounds(height);

    for (const round of commitRounds) {
      // Idempotency guard: never re-process a committed round
      if (this.rounds.isProcessed(round.id)) {
        this.log("debug", `Skipping already-processed round ${round.id}`);
        continue;
      }

      // Also skip if we already have a stored commitment (from previous session)
      if (this.commitments.get(round.id)) {
        this.log(
          "debug",
          `Skipping round ${round.id} — commitment already stored`,
        );
        continue;
      }

      await this.evaluateAndCommit(round, height);
    }
  }

  /**
   * Evaluate a claim and submit commitment for a single round.
   */
  private async evaluateAndCommit(
    round: ActiveRound,
    height: bigint,
  ): Promise<void> {
    this.log("info", `Processing round ${round.id} (claim: ${round.claimId})`);

    try {
      // Fetch claim data
      const claim = await this.rounds.fetchClaimData(round.claimId);

      // Evaluate with Claude (6s timeout, deterministic fallback)
      const result = await this.evaluator.evaluate(claim);

      if (result.source === "claude") {
        this.claudeEvaluations++;
        this.log(
          "info",
          `Claude: ${result.verdict} (${result.confidence}) in ${result.latencyMs}ms`,
        );
      } else {
        this.deterministicFallbacks++;
        this.log("warn", `Fallback: ${result.verdict} (${result.confidence})`);
      }

      // Generate salt and compute commitment hash
      const salt = generateSalt();
      const commitmentHash = computeCommitmentHash(
        round.id,
        result.verdict,
        result.confidence,
        salt,
      );

      // SAFETY: Persist to disk BEFORE submitting on-chain
      this.commitments.store({
        roundId: round.id,
        verdict: result.verdict,
        confidence: result.confidence,
        salt,
        commitmentHash,
        height,
        timestamp: Date.now(),
        source: result.source,
      });

      // Sign and submit commitment
      const signature = await this.session.signTransaction({
        roundId: round.id,
        commitmentHash,
        validatorAddress: this.config.validator.validatorAddress,
      });

      await this.rpc.submitCommitment({
        roundId: round.id,
        commitmentHash,
        validatorAddress: this.config.validator.validatorAddress,
        signature,
      });

      // Mark as processed (equivocation prevention)
      this.rounds.markProcessed(round.id, height);
      this.roundsProcessed++;

      this.log("info", `Committed round ${round.id}: ${result.verdict}`);
      this.emit("committed", { roundId: round.id, verdict: result.verdict });
    } catch (err) {
      this.errorCount++;
      this.log(
        "error",
        `Failed to commit round ${round.id}: ${(err as Error).message}`,
      );
    }
  }

  /**
   * Process all reveal-phase rounds: look up stored commitment → reveal.
   */
  private async processRevealPhase(_height: bigint): Promise<void> {
    // Check all stored commitments for rounds now in reveal phase
    for (const roundId of this.commitments.roundIds()) {
      const commitment = this.commitments.get(roundId);
      if (!commitment) continue;

      try {
        const phase = await this.rpc.getRoundPhase(roundId);

        if (phase === "reveal") {
          await this.submitReveal(roundId);
        } else if (phase === "complete" || phase === "aggregation") {
          // Round finished — clean up
          this.commitments.delete(roundId);
          this.log("debug", `Cleaned up completed round ${roundId}`);
        }
      } catch (err) {
        this.log(
          "error",
          `Failed to check/reveal round ${roundId}: ${(err as Error).message}`,
        );
      }
    }
  }

  /**
   * Submit a reveal for a stored commitment.
   */
  private async submitReveal(roundId: string): Promise<void> {
    const commitment = this.commitments.get(roundId);
    if (!commitment) {
      this.log("warn", `No stored commitment for reveal of round ${roundId}`);
      return;
    }

    // SAFETY: Verify hash matches before revealing
    if (!this.commitments.verifyBeforeReveal(roundId)) {
      this.log(
        "error",
        `Hash verification failed for round ${roundId} — skipping reveal`,
      );
      this.errorCount++;
      return;
    }

    const signature = await this.session.signTransaction({
      roundId,
      verdict: commitment.verdict,
      confidence: commitment.confidence,
      salt: commitment.salt,
      validatorAddress: this.config.validator.validatorAddress,
    });

    await this.rpc.submitReveal({
      roundId,
      verdict: commitment.verdict,
      confidence: commitment.confidence,
      salt: commitment.salt,
      justification: `${commitment.source} evaluation`,
      validatorAddress: this.config.validator.validatorAddress,
      signature,
    });

    // Clean up after successful reveal
    this.commitments.delete(roundId);
    this.revealsSubmitted++;
    this.log(
      "info",
      `Revealed round ${roundId}: ${commitment.verdict} (${commitment.confidence})`,
    );
    this.emit("revealed", { roundId, verdict: commitment.verdict });
  }

  // ============ Logging ============

  private log(level: LogLevel, message: string): void {
    if (LOG_LEVELS[level] < LOG_LEVELS[this.config.logLevel]) return;

    const ts = new Date().toISOString();
    const prefix = `[${ts}] [${level.toUpperCase()}]`;
    const line = `${prefix} ${message}`;

    if (level === "error") {
      console.error(line);
    } else {
      console.log(line);
    }
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
