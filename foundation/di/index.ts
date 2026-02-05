/**
 * Dependency Injection Module
 *
 * ARCH-3: Dependency Injection for Infrastructure
 * Domain modules MUST NOT directly instantiate infrastructure.
 * Infrastructure MUST be injected via interfaces.
 *
 * This module provides:
 * - DI container for service registration/resolution
 * - Factory interfaces for domain services
 * - Mock implementations for testing
 * - Composition root for wiring dependencies
 */

// Container
export {
  ServiceKeys,
  type ServiceKey,
  type Lifecycle,
  type ServiceRegistration,
  type Scope,
  type Container,
  type ScopedContainer,
  createContainer,
  ContainerBuilder,
  containerBuilder,
} from "./container.js";

// Factories
export {
  type KnowledgeDependencies,
  type ConsensusDependencies,
  type RewardsDependencies,
  type DisputeDependencies,
  type IdentityDependencies,
  type KnowledgeServiceFactory,
  type ConsensusEngineFactory,
  type RewardsEngineFactory,
  type DisputeResolverFactory,
  type IdentityServiceFactory,
  type FactoryRegistry,
  type AppConfig,
  type AppServices,
  createFactoryRegistry,
  composeApplication,
  LazyService,
  lazy,
  ServiceLocator,
} from "./factories.js";

// Mocks
export {
  MockStateProvider,
  MockEventBus,
  type MockEnvironment,
  createMockEnvironment,
} from "./mocks.js";
