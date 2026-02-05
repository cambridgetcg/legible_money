/**
 * Schedule Condition Evaluator
 *
 * Evaluates time and logic conditions to determine if a schedule process
 * should execute.
 *
 * From Patent: "SPU 131322 regularly checks that all processes set up are
 * served properly. In one embodiment, a process can be woken up by SPU 131322
 * when the scheduled time is reached."
 */

import type {
  ScheduleCondition,
  TimeCondition,
  LogicCondition,
  CompoundCondition,
  Comparator,
  ConditionProof,
  ScheduleProcess,
} from "./types.js";
import { ScheduleError } from "./types.js";

/**
 * State provider interface
 * Allows condition evaluator to read BVM state
 */
export interface StateProvider {
  /**
   * Get a value from BVM state
   */
  getState(key: string): Promise<unknown>;

  /**
   * Check if an event has occurred
   */
  hasEvent(
    eventType: string,
    filter?: Record<string, unknown>,
  ): Promise<boolean>;

  /**
   * Get current block number
   */
  getCurrentBlock(): bigint;
}

/**
 * Evaluation result
 */
export interface EvaluationResult {
  satisfied: boolean;
  proof?: ConditionProof;
  nextCheckTime?: number; // For time-based, when to check again
  reason?: string; // Why condition wasn't satisfied
}

/**
 * HIGH-003 FIX: Track threshold state for cross_up/cross_down detection
 */
interface ThresholdState {
  previousValue: bigint;
  lastCheckedBlock: bigint;
}

/**
 * Schedule Condition Evaluator
 * Determines if schedule conditions are met
 */
export class ConditionEvaluator {
  // HIGH-003 FIX: Track previous values for threshold crossing detection
  private thresholdStates: Map<string, ThresholdState> = new Map();

  constructor(private stateProvider: StateProvider) {}

  /**
   * Evaluate a schedule condition
   * @param condition - The condition to evaluate
   * @param currentTime - Current block timestamp
   * @param lastExecutedAt - MED-001 FIX: Last execution time for periodic deduplication
   */
  async evaluate(
    condition: ScheduleCondition,
    currentTime: number,
    lastExecutedAt?: number,
  ): Promise<EvaluationResult> {
    // Compound condition
    if (condition.compound) {
      return this.evaluateCompound(
        condition.compound,
        currentTime,
        lastExecutedAt,
      );
    }

    // Both time and logic (implicit AND)
    if (condition.time && condition.logic) {
      const timeResult = this.evaluateTime(
        condition.time,
        currentTime,
        lastExecutedAt,
      );
      if (!timeResult.satisfied) {
        return timeResult;
      }

      const logicResult = await this.evaluateLogic(condition.logic);
      if (!logicResult.satisfied) {
        return {
          satisfied: false,
          reason: logicResult.reason,
          nextCheckTime: timeResult.nextCheckTime,
        };
      }

      return {
        satisfied: true,
        proof: {
          type: "compound",
          subProofs: [timeResult.proof!, logicResult.proof!],
        },
      };
    }

    // Time only
    if (condition.time) {
      return this.evaluateTime(condition.time, currentTime, lastExecutedAt);
    }

    // Logic only
    if (condition.logic) {
      return this.evaluateLogic(condition.logic);
    }

    // No condition = always satisfied (shouldn't happen)
    return { satisfied: true };
  }

  /**
   * Evaluate time-based condition
   * @param lastExecutedAt - MED-001 FIX: For periodic, prevents multiple executions per interval
   */
  evaluateTime(
    condition: TimeCondition,
    currentTime: number,
    lastExecutedAt?: number,
  ): EvaluationResult {
    switch (condition.type) {
      case "absolute":
        return this.evaluateAbsoluteTime(condition, currentTime);

      case "relative":
        return this.evaluateRelativeTime(condition, currentTime);

      case "periodic":
        return this.evaluatePeriodicTime(
          condition,
          currentTime,
          lastExecutedAt,
        );

      default:
        throw new ScheduleError(
          `Unknown time condition type: ${condition.type}`,
          "INVALID_CONDITION",
        );
    }
  }

  private evaluateAbsoluteTime(
    condition: TimeCondition,
    currentTime: number,
  ): EvaluationResult {
    const executeAt = condition.executeAt!;

    if (currentTime >= executeAt) {
      return {
        satisfied: true,
        proof: {
          type: "time",
          timestamp: currentTime,
          blockNumber: this.stateProvider.getCurrentBlock(),
        },
      };
    }

    return {
      satisfied: false,
      nextCheckTime: executeAt,
      reason: `Waiting for time ${new Date(executeAt).toISOString()}`,
    };
  }

  private evaluateRelativeTime(
    condition: TimeCondition,
    currentTime: number,
  ): EvaluationResult {
    // For relative time, we assume executeAfter was converted to executeAt
    // when the process was created
    const executeAt =
      condition.executeAt ?? condition.executeAfter! + currentTime;

    if (currentTime >= executeAt) {
      return {
        satisfied: true,
        proof: {
          type: "time",
          timestamp: currentTime,
          blockNumber: this.stateProvider.getCurrentBlock(),
        },
      };
    }

    return {
      satisfied: false,
      nextCheckTime: executeAt,
      reason: `Waiting for relative time delay`,
    };
  }

  /**
   * MED-001 & MED-002 FIX: Proper periodic time evaluation
   */
  private evaluatePeriodicTime(
    condition: TimeCondition,
    currentTime: number,
    lastExecutedAt?: number,
  ): EvaluationResult {
    const { startTime = 0, endTime, interval } = condition;

    // MED-002 FIX: Validate interval is defined and positive
    if (!interval || interval <= 0) {
      throw new ScheduleError(
        "Periodic condition requires a positive interval",
        "INVALID_CONDITION",
        { interval },
      );
    }

    // Check if within valid range
    if (currentTime < startTime) {
      return {
        satisfied: false,
        nextCheckTime: startTime,
        reason: `Waiting for periodic start at ${new Date(startTime).toISOString()}`,
      };
    }

    if (endTime && currentTime >= endTime) {
      return {
        satisfied: false,
        reason: "Periodic schedule has ended",
      };
    }

    // Calculate current interval index
    const elapsed = currentTime - startTime;
    const currentIntervalIndex = Math.floor(elapsed / interval);
    const nextIntervalTime = startTime + (currentIntervalIndex + 1) * interval;

    // MED-001 FIX: Check if already executed in this interval
    if (lastExecutedAt !== undefined) {
      const lastElapsed = lastExecutedAt - startTime;
      const lastIntervalIndex = Math.floor(lastElapsed / interval);

      if (lastIntervalIndex >= currentIntervalIndex) {
        // Already executed in this interval (or a later one somehow)
        return {
          satisfied: false,
          nextCheckTime: nextIntervalTime,
          reason: `Already executed in interval ${currentIntervalIndex}`,
        };
      }
    }

    // New interval - can execute
    return {
      satisfied: true,
      proof: {
        type: "time",
        timestamp: currentTime,
        blockNumber: this.stateProvider.getCurrentBlock(),
      },
      nextCheckTime: nextIntervalTime,
    };
  }

  /**
   * Evaluate logic-based condition
   */
  async evaluateLogic(condition: LogicCondition): Promise<EvaluationResult> {
    switch (condition.type) {
      case "state_compare":
        return this.evaluateStateCompare(condition);

      case "event_occurred":
        return this.evaluateEventOccurred(condition);

      case "threshold_crossed":
        return this.evaluateThresholdCrossed(condition);

      default:
        throw new ScheduleError(
          `Unknown logic condition type: ${condition.type}`,
          "INVALID_CONDITION",
        );
    }
  }

  private async evaluateStateCompare(
    condition: LogicCondition,
  ): Promise<EvaluationResult> {
    const { stateKey, comparator, compareValue } = condition;

    if (!stateKey || !comparator) {
      throw new ScheduleError(
        "State comparison requires stateKey and comparator",
        "INVALID_CONDITION",
      );
    }

    const currentValue = await this.stateProvider.getState(stateKey);
    const satisfied = this.compare(currentValue, comparator, compareValue);

    return {
      satisfied,
      proof: satisfied
        ? {
            type: "logic",
            stateKey,
            stateValue: currentValue,
            blockNumber: this.stateProvider.getCurrentBlock(),
          }
        : undefined,
      reason: satisfied
        ? undefined
        : `State ${stateKey} = ${currentValue}, expected ${comparator} ${compareValue}`,
    };
  }

  private async evaluateEventOccurred(
    condition: LogicCondition,
  ): Promise<EvaluationResult> {
    const { eventType, eventFilter } = condition;

    if (!eventType) {
      throw new ScheduleError(
        "Event condition requires eventType",
        "INVALID_CONDITION",
      );
    }

    const occurred = await this.stateProvider.hasEvent(eventType, eventFilter);

    return {
      satisfied: occurred,
      proof: occurred
        ? {
            type: "logic",
            stateKey: `event:${eventType}`,
            stateValue: true,
            blockNumber: this.stateProvider.getCurrentBlock(),
          }
        : undefined,
      reason: occurred ? undefined : `Event ${eventType} has not occurred`,
    };
  }

  /**
   * HIGH-003 FIX: Proper threshold crossing detection with state tracking
   */
  private async evaluateThresholdCrossed(
    condition: LogicCondition,
  ): Promise<EvaluationResult> {
    const { thresholdKey, thresholdValue, direction } = condition;

    if (!thresholdKey || thresholdValue === undefined || !direction) {
      throw new ScheduleError(
        "Threshold condition requires thresholdKey, thresholdValue, and direction",
        "INVALID_CONDITION",
      );
    }

    // Get current value
    const currentValue = await this.stateProvider.getState(thresholdKey);
    let numericValue: bigint;
    try {
      numericValue =
        typeof currentValue === "bigint"
          ? currentValue
          : BigInt(String(currentValue));
    } catch {
      return {
        satisfied: false,
        reason: `State value '${currentValue}' for key '${thresholdKey}' is not numeric`,
      };
    }

    const currentBlock = this.stateProvider.getCurrentBlock();

    // Get previous state for crossing detection
    const prevState = this.thresholdStates.get(thresholdKey);
    const previousValue = prevState?.previousValue;

    let satisfied = false;
    let reason: string | undefined;

    switch (direction) {
      case "above":
        // Simple: current value is above threshold
        satisfied = numericValue > thresholdValue;
        reason = satisfied
          ? undefined
          : `Value ${numericValue} is not above ${thresholdValue}`;
        break;

      case "below":
        // Simple: current value is below threshold
        satisfied = numericValue < thresholdValue;
        reason = satisfied
          ? undefined
          : `Value ${numericValue} is not below ${thresholdValue}`;
        break;

      case "cross_up":
        // Crossing: was at or below, now above
        if (previousValue === undefined) {
          // First check - record state but don't trigger
          satisfied = false;
          reason = "First observation - tracking for future crossing";
        } else {
          satisfied =
            previousValue <= thresholdValue && numericValue > thresholdValue;
          reason = satisfied
            ? undefined
            : `No upward crossing (prev: ${previousValue}, curr: ${numericValue}, threshold: ${thresholdValue})`;
        }
        break;

      case "cross_down":
        // Crossing: was at or above, now below
        if (previousValue === undefined) {
          // First check - record state but don't trigger
          satisfied = false;
          reason = "First observation - tracking for future crossing";
        } else {
          satisfied =
            previousValue >= thresholdValue && numericValue < thresholdValue;
          reason = satisfied
            ? undefined
            : `No downward crossing (prev: ${previousValue}, curr: ${numericValue}, threshold: ${thresholdValue})`;
        }
        break;
    }

    // Update threshold state for next check
    this.thresholdStates.set(thresholdKey, {
      previousValue: numericValue,
      lastCheckedBlock: currentBlock,
    });

    return {
      satisfied,
      proof: satisfied
        ? {
            type: "logic",
            stateKey: thresholdKey,
            stateValue: numericValue.toString(),
            blockNumber: currentBlock,
          }
        : undefined,
      reason,
    };
  }

  /**
   * Evaluate compound condition (AND/OR)
   */
  private async evaluateCompound(
    condition: CompoundCondition,
    currentTime: number,
    lastExecutedAt?: number,
  ): Promise<EvaluationResult> {
    const results: EvaluationResult[] = [];

    for (const subCondition of condition.conditions) {
      const result = await this.evaluate(
        subCondition,
        currentTime,
        lastExecutedAt,
      );
      results.push(result);

      // Short-circuit for AND (any false = false)
      if (condition.operator === "and" && !result.satisfied) {
        return {
          satisfied: false,
          nextCheckTime: this.getEarliestNextCheck(results),
          reason: result.reason,
        };
      }

      // Short-circuit for OR (any true = true)
      if (condition.operator === "or" && result.satisfied) {
        return {
          satisfied: true,
          proof: {
            type: "compound",
            subProofs: results.filter((r) => r.proof).map((r) => r.proof!),
          },
        };
      }
    }

    // AND: all must be true (we get here if all were true)
    // OR: none were true (we get here if all were false)
    const satisfied = condition.operator === "and";

    return {
      satisfied,
      proof: satisfied
        ? {
            type: "compound",
            subProofs: results.filter((r) => r.proof).map((r) => r.proof!),
          }
        : undefined,
      nextCheckTime: this.getEarliestNextCheck(results),
      reason: satisfied ? undefined : "No conditions in OR were satisfied",
    };
  }

  /**
   * Compare values using comparator
   */
  private compare(
    currentValue: unknown,
    comparator: Comparator,
    compareValue: unknown,
  ): boolean {
    // Handle bigint comparisons
    if (typeof currentValue === "bigint" || typeof compareValue === "bigint") {
      const a =
        typeof currentValue === "bigint"
          ? currentValue
          : BigInt(String(currentValue));
      const b =
        typeof compareValue === "bigint"
          ? compareValue
          : BigInt(String(compareValue));

      switch (comparator) {
        case "eq":
          return a === b;
        case "ne":
          return a !== b;
        case "gt":
          return a > b;
        case "gte":
          return a >= b;
        case "lt":
          return a < b;
        case "lte":
          return a <= b;
        default:
          return false;
      }
    }

    // Handle number comparisons
    if (typeof currentValue === "number" || typeof compareValue === "number") {
      const a = Number(currentValue);
      const b = Number(compareValue);

      switch (comparator) {
        case "eq":
          return a === b;
        case "ne":
          return a !== b;
        case "gt":
          return a > b;
        case "gte":
          return a >= b;
        case "lt":
          return a < b;
        case "lte":
          return a <= b;
        default:
          return false;
      }
    }

    // Handle string comparisons
    if (typeof currentValue === "string" || typeof compareValue === "string") {
      const a = String(currentValue);
      const b = String(compareValue);

      switch (comparator) {
        case "eq":
          return a === b;
        case "ne":
          return a !== b;
        case "gt":
          return a > b;
        case "gte":
          return a >= b;
        case "lt":
          return a < b;
        case "lte":
          return a <= b;
        case "contains":
          return a.includes(b);
        case "not_contains":
          return !a.includes(b);
        default:
          return false;
      }
    }

    // Handle array contains
    if (Array.isArray(currentValue)) {
      switch (comparator) {
        case "contains":
          return currentValue.includes(compareValue);
        case "not_contains":
          return !currentValue.includes(compareValue);
        default:
          return false;
      }
    }

    // Fallback: strict equality
    return comparator === "eq"
      ? currentValue === compareValue
      : currentValue !== compareValue;
  }

  /**
   * Get earliest next check time from multiple results
   */
  private getEarliestNextCheck(
    results: EvaluationResult[],
  ): number | undefined {
    const times = results
      .map((r) => r.nextCheckTime)
      .filter((t): t is number => t !== undefined);

    return times.length > 0 ? Math.min(...times) : undefined;
  }

  /**
   * Calculate next check time for a process
   */
  calculateNextCheckTime(
    process: ScheduleProcess,
    currentTime: number,
  ): number | undefined {
    const condition = process.condition;

    // Time-based conditions
    if (condition.time) {
      switch (condition.time.type) {
        case "absolute":
          return condition.time.executeAt;

        case "relative":
          return process.createdAt + (condition.time.executeAfter ?? 0);

        case "periodic":
          if (condition.time.interval) {
            const startTime = condition.time.startTime ?? process.createdAt;
            const elapsed = currentTime - startTime;
            const nextInterval = Math.ceil(elapsed / condition.time.interval);
            return startTime + nextInterval * condition.time.interval;
          }
      }
    }

    // Logic-based conditions are checked on state change, not time
    // Return undefined to indicate no time-based check needed
    return undefined;
  }

  /**
   * Extract state keys to watch from a condition
   */
  extractWatchedKeys(condition: ScheduleCondition): string[] {
    const keys: string[] = [];

    if (condition.logic) {
      if (condition.logic.stateKey) {
        keys.push(condition.logic.stateKey);
      }
      if (condition.logic.thresholdKey) {
        keys.push(condition.logic.thresholdKey);
      }
    }

    if (condition.compound) {
      for (const subCondition of condition.compound.conditions) {
        keys.push(...this.extractWatchedKeys(subCondition));
      }
    }

    return [...new Set(keys)]; // Deduplicate
  }

  /**
   * HIGH-003 FIX: Clear threshold state for a specific key
   * Called when a process is cancelled/completed to free memory
   */
  clearThresholdState(key: string): void {
    this.thresholdStates.delete(key);
  }

  /**
   * HIGH-003 FIX: Clear all threshold states
   * Useful for testing or reset scenarios
   */
  clearAllThresholdStates(): void {
    this.thresholdStates.clear();
  }

  /**
   * HIGH-003 FIX: Get threshold state for debugging/testing
   */
  getThresholdState(key: string): ThresholdState | undefined {
    return this.thresholdStates.get(key);
  }
}
