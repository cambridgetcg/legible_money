/**
 * Discovery Engine
 *
 * Orchestrates the passive discovery of the hidden message embedded
 * in the Legible Money protocol constants.
 *
 * "The protocol rewards those who truly engage with truth."
 *
 * PP HIERARCHY:
 *   TRUTH: Discoveries grounded in actual protocol constants
 *   UNDERSTANDING: Progressive revelation through genuine engagement
 *   BEAUTY: Seven mechanics mirror seven strata
 *   JUSTICE: Relational discovery requires cooperation
 *   CREATIVITY: Building paths to understanding through participation
 */

import { EventEmitter } from "events";
import type {
  DiscoveryState,
  DiscoveryEngineConfig,
  DiscoveryEvent,
  DetectionResult,
  DiscoveryMechanic,
  FragmentDiscovery,
  TriggerContext,
  VerificationEchoInput,
  AccumulatedWitnessInput,
  StakeSpiralInput,
  ConfidenceArchaeologyInput,
} from "./types.js";
import { DEFAULT_DISCOVERY_CONFIG } from "./types.js";
import type { DID } from "../../core/agents/identity/did.js";
import type { OntologicalStratum } from "../knowledge/ontology/types.js";
import {
  getFragmentForMechanic,
  canUnlockMechanic,
  assembleRevealedPoem,
  getCompletionStatus,
  isLineComplete,
  getCompletePoemLines,
  DISCOVERY_ORDER,
} from "./fragments.js";
import { detectVerificationEcho } from "./detectors/verification_echo.js";
import { detectConfidenceArchaeology } from "./detectors/confidence_archaeology.js";
import { detectBlockHeightPoem } from "./detectors/block_height_poems.js";
import { detectAccumulatedWitness } from "./detectors/accumulated_witness.js";
import { detectMerkleMeditation } from "./detectors/merkle_meditation.js";
import { detectStakeSpiral } from "./detectors/stake_spiral.js";
import { detectCrossStratumBridge } from "./detectors/cross_stratum_bridge.js";

// ============================================================================
// Engine State
// ============================================================================

interface EngineState {
  /** Is the engine running */
  running: boolean;

  /** Last block checked */
  lastCheckBlock: bigint;

  /** Discovery states per agent */
  discoveryStates: Map<DID, DiscoveryState>;

  /** Agents currently being tracked */
  trackedAgents: Set<DID>;
}

// ============================================================================
// Data Provider Interface
// ============================================================================

/**
 * Interface for providing agent data to the discovery engine
 * This decouples the engine from specific state manager implementations
 */
export interface DiscoveryDataProvider {
  /** Get verification history for an agent */
  getVerificationHistory(
    agentDid: DID,
  ): Promise<VerificationEchoInput["verificationHistory"]>;

  /** Get relationship data for an agent */
  getRelationshipData(
    agentDid: DID,
  ): Promise<VerificationEchoInput["relationshipData"]>;

  /** Get agent statistics */
  getAgentStatistics(agentDid: DID): Promise<AccumulatedWitnessInput>;

  /** Get stake history for an agent */
  getStakeHistory(agentDid: DID): Promise<StakeSpiralInput["stakeHistory"]>;

  /** Get fact contributions by stratum */
  getFactContributions(
    agentDid: DID,
  ): Promise<Map<OntologicalStratum, string[]>>;

  /** Get current state root */
  getStateRoot(): Promise<string>;

  /** Get axiom access history */
  getAxiomAccessHistory(
    agentDid: DID,
  ): Promise<ConfidenceArchaeologyInput["axiomAccessHistory"]>;

  /** Get blocks where agent was active */
  getActiveBlocks(agentDid: DID): Promise<Set<bigint>>;
}

// ============================================================================
// Discovery Engine
// ============================================================================

/**
 * DiscoveryEngine - Passive discovery of the hidden message
 *
 * The engine monitors agent activities and protocol events,
 * detecting when discovery conditions are met, then progressively
 * reveals message fragments through events.
 */
export class DiscoveryEngine extends EventEmitter {
  private config: DiscoveryEngineConfig;
  private state: EngineState;
  private dataProvider?: DiscoveryDataProvider;

  constructor(config: Partial<DiscoveryEngineConfig> = {}) {
    super();
    this.config = { ...DEFAULT_DISCOVERY_CONFIG, ...config };
    this.state = {
      running: false,
      lastCheckBlock: 0n,
      discoveryStates: new Map(),
      trackedAgents: new Set(),
    };
  }

  // ==========================================================================
  // Lifecycle
  // ==========================================================================

  /**
   * Start the discovery engine
   */
  async start(): Promise<void> {
    if (this.state.running) return;
    this.state.running = true;
    this.emit("started");
  }

  /**
   * Stop the discovery engine
   */
  async stop(): Promise<void> {
    if (!this.state.running) return;
    this.state.running = false;
    this.emit("stopped");
  }

  /**
   * Check if engine is running
   */
  isRunning(): boolean {
    return this.state.running;
  }

  /**
   * Set data provider for agent information
   */
  setDataProvider(provider: DiscoveryDataProvider): void {
    this.dataProvider = provider;
  }

  // ==========================================================================
  // Agent Tracking
  // ==========================================================================

  /**
   * Start tracking an agent for discoveries
   */
  trackAgent(agentDid: DID): void {
    if (!this.state.trackedAgents.has(agentDid)) {
      this.state.trackedAgents.add(agentDid);
      // Initialize empty discovery state
      this.state.discoveryStates.set(agentDid, this.createEmptyState(agentDid));
    }
  }

  /**
   * Stop tracking an agent
   */
  untrackAgent(agentDid: DID): void {
    this.state.trackedAgents.delete(agentDid);
    // Keep discovery state for history
  }

  /**
   * Get all tracked agents
   */
  getTrackedAgents(): DID[] {
    return Array.from(this.state.trackedAgents);
  }

  // ==========================================================================
  // Discovery State
  // ==========================================================================

  /**
   * Get or create discovery state for an agent
   */
  getDiscoveryState(agentDid: DID): DiscoveryState {
    let state = this.state.discoveryStates.get(agentDid);
    if (!state) {
      state = this.createEmptyState(agentDid);
      this.state.discoveryStates.set(agentDid, state);
    }
    return state;
  }

  /**
   * Create empty discovery state
   */
  private createEmptyState(agentDid: DID): DiscoveryState {
    return {
      agentDid,
      discoveries: new Map(),
      unlockedLines: [],
      discoveryScore: 0,
      isComplete: false,
    };
  }

  // ==========================================================================
  // Block Processing
  // ==========================================================================

  /**
   * Process a block and check for discoveries
   * Called by consensus engine on each new block
   */
  async processBlock(blockNumber: bigint): Promise<DiscoveryEvent[]> {
    if (!this.state.running) return [];
    if (!this.config.enablePassiveMonitoring) return [];

    // Check interval (but always check special block heights)
    const shouldCheck =
      blockNumber - this.state.lastCheckBlock >=
        this.config.checkIntervalBlocks ||
      this.config.specialBlockHeights.includes(blockNumber);

    if (!shouldCheck) return [];

    this.state.lastCheckBlock = blockNumber;
    const events: DiscoveryEvent[] = [];

    // Check all tracked agents
    for (const agentDid of this.state.trackedAgents) {
      const discoveryState = this.getDiscoveryState(agentDid);
      const agentEvents = await this.checkAgentDiscoveries(
        agentDid,
        discoveryState,
        blockNumber,
      );
      events.push(...agentEvents);
    }

    return events;
  }

  /**
   * Check all discovery mechanics for an agent
   */
  private async checkAgentDiscoveries(
    agentDid: DID,
    state: DiscoveryState,
    currentBlock: bigint,
  ): Promise<DiscoveryEvent[]> {
    const events: DiscoveryEvent[] = [];
    const currentDiscoveries = Array.from(state.discoveries.keys());

    // Check each undiscovered mechanic
    for (const mechanic of DISCOVERY_ORDER) {
      if (state.discoveries.has(mechanic)) continue;

      // Check prerequisites
      const { canUnlock } = canUnlockMechanic(mechanic, currentDiscoveries);
      if (!canUnlock) continue;

      // Run detection
      const result = await this.runDetector(agentDid, mechanic, currentBlock);

      if (result.detected) {
        const discovery = this.recordDiscovery(
          state,
          mechanic,
          result,
          currentBlock,
        );

        // Emit fragment discovered event
        const fragmentEvent: DiscoveryEvent = {
          type: "fragment_discovered",
          agentDid,
          mechanic,
          fragment: discovery.fragment,
          collaborator: result.collaborator,
          timestamp: Date.now(),
          blockNumber: currentBlock,
          data: result.matchDetails,
        };
        events.push(fragmentEvent);
        this.emit("fragment_discovered", fragmentEvent);

        // Check for line completion
        for (let lineNum = 1; lineNum <= 3; lineNum++) {
          const prevComplete = isLineComplete(lineNum, currentDiscoveries);
          const nowComplete = isLineComplete(lineNum, [
            ...currentDiscoveries,
            mechanic,
          ]);

          if (!prevComplete && nowComplete) {
            const lines = getCompletePoemLines();
            const line = lines.find((l) => l.lineNumber === lineNum);
            if (line) {
              const lineEvent: DiscoveryEvent = {
                type: "line_completed",
                agentDid,
                line,
                timestamp: Date.now(),
                blockNumber: currentBlock,
              };
              events.push(lineEvent);
              this.emit("line_completed", lineEvent);
            }
          }
        }

        // Check for poem completion
        if (this.checkCompletion(state, currentBlock)) {
          const completeEvent: DiscoveryEvent = {
            type: "poem_completed",
            agentDid,
            timestamp: Date.now(),
            blockNumber: currentBlock,
          };
          events.push(completeEvent);
          this.emit("poem_completed", completeEvent);
        }

        // Update current discoveries for prerequisite checks
        currentDiscoveries.push(mechanic);
      }
    }

    return events;
  }

  /**
   * Run a specific detector
   */
  private async runDetector(
    agentDid: DID,
    mechanic: DiscoveryMechanic,
    currentBlock: bigint,
  ): Promise<DetectionResult> {
    if (!this.dataProvider) {
      return {
        detected: false,
        mechanic,
        failureReason: "No data provider configured",
      };
    }

    try {
      switch (mechanic) {
        case "verification_echo": {
          const history =
            await this.dataProvider.getVerificationHistory(agentDid);
          const relationships =
            await this.dataProvider.getRelationshipData(agentDid);
          return detectVerificationEcho(
            {
              agentDid,
              verificationHistory: history,
              relationshipData: relationships,
            },
            this.config.patternTolerance,
          );
        }

        case "confidence_archaeology": {
          const axiomHistory =
            await this.dataProvider.getAxiomAccessHistory(agentDid);
          return detectConfidenceArchaeology({
            agentDid,
            axiomAccessHistory: axiomHistory,
            genesisAxiomIds: this.config.genesisAxiomIds,
          });
        }

        case "block_height_poems": {
          const activeBlocks =
            await this.dataProvider.getActiveBlocks(agentDid);
          return detectBlockHeightPoem({
            agentDid,
            currentBlock,
            agentActiveAtBlocks: activeBlocks,
            specialBlockHeights: this.config.specialBlockHeights,
          });
        }

        case "accumulated_witness": {
          const stats = await this.dataProvider.getAgentStatistics(agentDid);
          return detectAccumulatedWitness(stats, this.config.ratioTolerance);
        }

        case "merkle_meditation": {
          const stateRoot = await this.dataProvider.getStateRoot();
          return detectMerkleMeditation({
            agentDid,
            stateRoot,
            currentBlock,
          });
        }

        case "stake_spiral": {
          const stakeHistory =
            await this.dataProvider.getStakeHistory(agentDid);
          return detectStakeSpiral({ agentDid, stakeHistory });
        }

        case "cross_stratum_bridge": {
          const contributions =
            await this.dataProvider.getFactContributions(agentDid);
          return detectCrossStratumBridge({
            agentDid,
            factContributions: contributions,
          });
        }

        default:
          return {
            detected: false,
            mechanic,
            failureReason: "Unknown mechanic",
          };
      }
    } catch (error) {
      return {
        detected: false,
        mechanic,
        failureReason: `Error: ${error instanceof Error ? error.message : "Unknown"}`,
      };
    }
  }

  /**
   * Record a discovery
   */
  private recordDiscovery(
    state: DiscoveryState,
    mechanic: DiscoveryMechanic,
    result: DetectionResult,
    currentBlock: bigint,
  ): FragmentDiscovery {
    const fragment = getFragmentForMechanic(mechanic)!;

    const discovery: FragmentDiscovery = {
      mechanic,
      fragment,
      discoveredAt: Date.now(),
      discoveredAtBlock: currentBlock,
      triggerContext: {
        type: this.getTriggerType(mechanic),
        matchData: result.matchDetails || {},
        description: result.pattern || mechanic,
      },
      collaborator: result.collaborator,
    };

    state.discoveries.set(mechanic, discovery);

    // Update first discovery tracking
    if (!state.firstDiscoveryAt) {
      state.firstDiscoveryAt = discovery.discoveredAt;
      state.firstDiscoveryBlock = currentBlock;
    }

    // Recalculate score
    state.discoveryScore =
      (state.discoveries.size / DISCOVERY_ORDER.length) * 100;

    return discovery;
  }

  /**
   * Get trigger type for a mechanic
   */
  private getTriggerType(mechanic: DiscoveryMechanic): TriggerContext["type"] {
    switch (mechanic) {
      case "verification_echo":
        return "verification";
      case "confidence_archaeology":
        return "genesis";
      case "block_height_poems":
        return "block_height";
      case "accumulated_witness":
        return "statistics";
      case "merkle_meditation":
        return "merkle";
      case "stake_spiral":
        return "stake";
      case "cross_stratum_bridge":
        return "cross_stratum";
    }
  }

  /**
   * Check if poem is complete
   */
  private checkCompletion(
    state: DiscoveryState,
    currentBlock: bigint,
  ): boolean {
    if (
      state.discoveries.size === DISCOVERY_ORDER.length &&
      !state.isComplete
    ) {
      state.isComplete = true;
      state.completedAt = Date.now();
      state.completionBlock = currentBlock;
      return true;
    }
    return false;
  }

  // ==========================================================================
  // Manual Trigger (for testing or explicit discovery attempts)
  // ==========================================================================

  /**
   * Manually trigger discovery check for an agent
   * Useful for testing or when agent explicitly requests discovery check
   */
  async triggerDiscoveryCheck(
    agentDid: DID,
    currentBlock: bigint,
  ): Promise<DiscoveryEvent[]> {
    this.trackAgent(agentDid);
    const state = this.getDiscoveryState(agentDid);
    return this.checkAgentDiscoveries(agentDid, state, currentBlock);
  }

  /**
   * Manually check a specific mechanic
   */
  async checkSpecificMechanic(
    agentDid: DID,
    mechanic: DiscoveryMechanic,
    currentBlock: bigint,
  ): Promise<DetectionResult> {
    return this.runDetector(agentDid, mechanic, currentBlock);
  }

  // ==========================================================================
  // Query Methods
  // ==========================================================================

  /**
   * Get all discoveries for an agent
   */
  getAgentDiscoveries(agentDid: DID): FragmentDiscovery[] {
    const state = this.state.discoveryStates.get(agentDid);
    if (!state) return [];
    return Array.from(state.discoveries.values());
  }

  /**
   * Get the revealed poem text for an agent
   */
  getRevealedPoem(agentDid: DID): string {
    const state = this.state.discoveryStates.get(agentDid);
    if (!state) return "";
    return assembleRevealedPoem(Array.from(state.discoveries.keys()));
  }

  /**
   * Get completion status for an agent
   */
  getAgentCompletionStatus(
    agentDid: DID,
  ): ReturnType<typeof getCompletionStatus> {
    const state = this.state.discoveryStates.get(agentDid);
    if (!state) {
      return {
        discovered: 0,
        total: DISCOVERY_ORDER.length,
        percentage: 0,
        isComplete: false,
        nextMechanic: DISCOVERY_ORDER[0],
      };
    }
    return getCompletionStatus(Array.from(state.discoveries.keys()));
  }

  /**
   * Check if agent has discovered a specific mechanic
   */
  hasDiscovered(agentDid: DID, mechanic: DiscoveryMechanic): boolean {
    const state = this.state.discoveryStates.get(agentDid);
    return state?.discoveries.has(mechanic) ?? false;
  }

  /**
   * Get discovery by mechanic for an agent
   */
  getDiscovery(
    agentDid: DID,
    mechanic: DiscoveryMechanic,
  ): FragmentDiscovery | undefined {
    const state = this.state.discoveryStates.get(agentDid);
    return state?.discoveries.get(mechanic);
  }

  // ==========================================================================
  // Serialization
  // ==========================================================================

  /**
   * Export discovery states for persistence
   */
  exportState(): {
    lastCheckBlock: string;
    discoveryStates: Array<{
      agentDid: DID;
      discoveries: Array<
        [
          DiscoveryMechanic,
          Omit<FragmentDiscovery, "discoveredAtBlock"> & {
            discoveredAtBlock: string;
          },
        ]
      >;
      discoveryScore: number;
      isComplete: boolean;
      firstDiscoveryAt?: number;
      completedAt?: number;
      firstDiscoveryBlock?: string;
      completionBlock?: string;
    }>;
  } {
    const states = Array.from(this.state.discoveryStates.entries()).map(
      ([agentDid, state]) => ({
        agentDid,
        discoveries: Array.from(state.discoveries.entries()).map(
          ([m, d]): [
            DiscoveryMechanic,
            Omit<FragmentDiscovery, "discoveredAtBlock"> & {
              discoveredAtBlock: string;
            },
          ] => [
            m,
            {
              ...d,
              discoveredAtBlock: d.discoveredAtBlock.toString(),
            },
          ],
        ),
        discoveryScore: state.discoveryScore,
        isComplete: state.isComplete,
        firstDiscoveryAt: state.firstDiscoveryAt,
        completedAt: state.completedAt,
        firstDiscoveryBlock: state.firstDiscoveryBlock?.toString(),
        completionBlock: state.completionBlock?.toString(),
      }),
    );

    return {
      lastCheckBlock: this.state.lastCheckBlock.toString(),
      discoveryStates: states,
    };
  }

  /**
   * Import discovery states from persistence
   */
  importState(
    data: ReturnType<typeof DiscoveryEngine.prototype.exportState>,
  ): void {
    this.state.lastCheckBlock = BigInt(data.lastCheckBlock);

    for (const stateData of data.discoveryStates) {
      const discoveries = new Map<DiscoveryMechanic, FragmentDiscovery>();

      for (const [mechanic, discovery] of stateData.discoveries) {
        discoveries.set(mechanic, {
          ...discovery,
          discoveredAtBlock: BigInt(
            discovery.discoveredAtBlock as unknown as string,
          ),
        });
      }

      const state: DiscoveryState = {
        agentDid: stateData.agentDid,
        discoveries,
        unlockedLines: [], // Reconstructed from discoveries
        discoveryScore: stateData.discoveryScore,
        isComplete: stateData.isComplete,
        firstDiscoveryAt: stateData.firstDiscoveryAt,
        completedAt: stateData.completedAt,
        firstDiscoveryBlock: stateData.firstDiscoveryBlock
          ? BigInt(stateData.firstDiscoveryBlock)
          : undefined,
        completionBlock: stateData.completionBlock
          ? BigInt(stateData.completionBlock)
          : undefined,
      };

      this.state.discoveryStates.set(stateData.agentDid, state);
      this.state.trackedAgents.add(stateData.agentDid);
    }
  }
}

// ============================================================================
// Factory Functions
// ============================================================================

/**
 * Create discovery engine with default configuration
 */
export function createDiscoveryEngine(
  config?: Partial<DiscoveryEngineConfig>,
): DiscoveryEngine {
  return new DiscoveryEngine(config);
}

/**
 * Create discovery engine with data provider
 */
export function createIntegratedDiscoveryEngine(
  dataProvider: DiscoveryDataProvider,
  config?: Partial<DiscoveryEngineConfig>,
): DiscoveryEngine {
  const engine = new DiscoveryEngine(config);
  engine.setDataProvider(dataProvider);
  return engine;
}
