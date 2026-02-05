/**
 * Service Factory Interfaces
 *
 * Factory functions for creating domain services with injected dependencies.
 * Each domain module provides a factory that accepts its dependencies.
 *
 * Pattern:
 *   function createXxxService(deps: XxxDependencies): XxxService
 */

import type {
  StateProvider,
  EventBus,
  KnowledgeService,
  ConsensusEngine,
  RewardsEngine,
  DisputeResolver,
  IdentityService,
  ConsensusParams,
  VestingParams,
  DisputeParams,
} from "../contracts/index.js";

// ============ Dependency Types ============

/**
 * Dependencies for KnowledgeService
 */
export interface KnowledgeDependencies {
  state: StateProvider;
  events: EventBus;
  consensus: ConsensusEngine;
}

/**
 * Dependencies for ConsensusEngine
 */
export interface ConsensusDependencies {
  state: StateProvider;
  events: EventBus;
  params: ConsensusParams;
}

/**
 * Dependencies for RewardsEngine
 */
export interface RewardsDependencies {
  state: StateProvider;
  events: EventBus;
  knowledge: KnowledgeService;
  params: VestingParams;
}

/**
 * Dependencies for DisputeResolver
 */
export interface DisputeDependencies {
  state: StateProvider;
  events: EventBus;
  knowledge: KnowledgeService;
  rewards: RewardsEngine;
  params: DisputeParams;
}

/**
 * Dependencies for IdentityService
 */
export interface IdentityDependencies {
  state: StateProvider;
  events: EventBus;
}

// ============ Factory Types ============

/**
 * Factory for creating KnowledgeService
 */
export type KnowledgeServiceFactory = (
  deps: KnowledgeDependencies,
) => KnowledgeService;

/**
 * Factory for creating ConsensusEngine
 */
export type ConsensusEngineFactory = (
  deps: ConsensusDependencies,
) => ConsensusEngine;

/**
 * Factory for creating RewardsEngine
 */
export type RewardsEngineFactory = (deps: RewardsDependencies) => RewardsEngine;

/**
 * Factory for creating DisputeResolver
 */
export type DisputeResolverFactory = (
  deps: DisputeDependencies,
) => DisputeResolver;

/**
 * Factory for creating IdentityService
 */
export type IdentityServiceFactory = (
  deps: IdentityDependencies,
) => IdentityService;

// ============ Factory Registry ============

/**
 * Registry of all service factories
 */
export interface FactoryRegistry {
  knowledge?: KnowledgeServiceFactory;
  consensus?: ConsensusEngineFactory;
  rewards?: RewardsEngineFactory;
  disputes?: DisputeResolverFactory;
  identity?: IdentityServiceFactory;
}

/**
 * Create a factory registry
 */
export function createFactoryRegistry(): FactoryRegistry {
  return {};
}

// ============ Composition Root ============

/**
 * Application configuration
 */
export interface AppConfig {
  consensusParams: ConsensusParams;
  vestingParams: VestingParams;
  disputeParams: DisputeParams;
}

/**
 * Composed application services
 */
export interface AppServices {
  state: StateProvider;
  events: EventBus;
  knowledge: KnowledgeService;
  consensus: ConsensusEngine;
  rewards: RewardsEngine;
  disputes: DisputeResolver;
  identity: IdentityService;
}

/**
 * Composition root - wires up all dependencies
 *
 * This is where the dependency graph is constructed.
 * Call this once at application startup.
 */
export function composeApplication(
  stateProvider: StateProvider,
  eventBus: EventBus,
  factories: Required<FactoryRegistry>,
  config: AppConfig,
): AppServices {
  // Build services in dependency order

  // 1. Identity has no domain dependencies
  const identity = factories.identity({
    state: stateProvider,
    events: eventBus,
  });

  // 2. Consensus depends on state, events, params
  const consensus = factories.consensus({
    state: stateProvider,
    events: eventBus,
    params: config.consensusParams,
  });

  // 3. Knowledge depends on state, events, consensus
  const knowledge = factories.knowledge({
    state: stateProvider,
    events: eventBus,
    consensus,
  });

  // 4. Rewards depends on state, events, knowledge
  const rewards = factories.rewards({
    state: stateProvider,
    events: eventBus,
    knowledge,
    params: config.vestingParams,
  });

  // 5. Disputes depends on state, events, knowledge, rewards
  const disputes = factories.disputes({
    state: stateProvider,
    events: eventBus,
    knowledge,
    rewards,
    params: config.disputeParams,
  });

  return {
    state: stateProvider,
    events: eventBus,
    knowledge,
    consensus,
    rewards,
    disputes,
    identity,
  };
}

// ============ Lazy Composition ============

/**
 * Lazy service wrapper - delays instantiation until first use
 */
export class LazyService<T> {
  private instance?: T;
  private readonly factory: () => T;

  constructor(factory: () => T) {
    this.factory = factory;
  }

  get value(): T {
    if (this.instance === undefined) {
      this.instance = this.factory();
    }
    return this.instance;
  }

  get isCreated(): boolean {
    return this.instance !== undefined;
  }
}

/**
 * Create a lazy service wrapper
 */
export function lazy<T>(factory: () => T): LazyService<T> {
  return new LazyService(factory);
}

// ============ Service Locator (Anti-Pattern Warning) ============

/**
 * Global service locator
 *
 * WARNING: This is provided for legacy code migration only.
 * Prefer constructor injection over service location.
 *
 * Usage:
 *   // At startup
 *   ServiceLocator.setContainer(container);
 *
 *   // In legacy code (avoid in new code!)
 *   const knowledge = ServiceLocator.get<KnowledgeService>(ServiceKeys.KnowledgeService);
 */
export class ServiceLocator {
  private static container?: {
    resolve<T>(key: symbol): T;
  };

  static setContainer(container: { resolve<T>(key: symbol): T }): void {
    ServiceLocator.container = container;
  }

  static get<T>(key: symbol): T {
    if (!ServiceLocator.container) {
      throw new Error(
        "ServiceLocator not initialized. Call setContainer() first.",
      );
    }
    return ServiceLocator.container.resolve<T>(key);
  }

  static tryGet<T>(key: symbol): T | undefined {
    try {
      return ServiceLocator.get<T>(key);
    } catch {
      return undefined;
    }
  }

  static clear(): void {
    ServiceLocator.container = undefined;
  }
}
