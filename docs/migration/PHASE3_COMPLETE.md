# Phase 3: Dependency Inversion - COMPLETE

**Date**: 2026-02-04
**Status**: COMPLETE

## Summary

Created dependency injection infrastructure enabling domain modules to receive infrastructure dependencies via interfaces rather than direct instantiation.

## Created Files

```
foundation/di/
├── index.ts          # Public API exports
├── container.ts      # DI container with lifecycle management
├── factories.ts      # Factory interfaces and composition root
└── mocks.ts          # Mock implementations for testing
```

## DI Container

### Service Registration

```typescript
import { createContainer, ServiceKeys, containerBuilder } from '@legible/foundation';

// Direct container usage
const container = createContainer();
container.register(ServiceKeys.StateProvider, (c) => new StateProviderImpl());
container.registerInstance(ServiceKeys.EventBus, eventBus);

// Builder pattern (fluent API)
const container = containerBuilder()
  .addSingleton(ServiceKeys.StateProvider, (c) => new StateProviderImpl())
  .addTransient(ServiceKeys.SomeService, (c) => new SomeServiceImpl())
  .addScoped(ServiceKeys.RequestHandler, (c) => new RequestHandler())
  .build();
```

### Lifecycle Management

| Lifecycle | Behavior |
|-----------|----------|
| `singleton` | One instance for entire application |
| `transient` | New instance on each resolve |
| `scoped` | One instance per scope (e.g., request) |

### Scoped Containers

```typescript
// Create child scope for request isolation
const scope = container.createScope();
const handler = scope.resolve<RequestHandler>(ServiceKeys.RequestHandler);
// ... handle request
await scope.disposeScope(); // Clean up scoped instances
```

## Factory Interfaces

### Dependency Types

Each domain module declares its dependencies:

```typescript
interface KnowledgeDependencies {
  state: StateProvider;
  events: EventBus;
  consensus: ConsensusEngine;
}

interface ConsensusDependencies {
  state: StateProvider;
  events: EventBus;
  params: ConsensusParams;
}

interface RewardsDependencies {
  state: StateProvider;
  events: EventBus;
  knowledge: KnowledgeService;
  params: VestingParams;
}

interface DisputeDependencies {
  state: StateProvider;
  events: EventBus;
  knowledge: KnowledgeService;
  rewards: RewardsEngine;
  params: DisputeParams;
}

interface IdentityDependencies {
  state: StateProvider;
  events: EventBus;
}
```

### Factory Types

```typescript
type KnowledgeServiceFactory = (deps: KnowledgeDependencies) => KnowledgeService;
type ConsensusEngineFactory = (deps: ConsensusDependencies) => ConsensusEngine;
type RewardsEngineFactory = (deps: RewardsDependencies) => RewardsEngine;
type DisputeResolverFactory = (deps: DisputeDependencies) => DisputeResolver;
type IdentityServiceFactory = (deps: IdentityDependencies) => IdentityService;
```

## Composition Root

The composition root wires up all dependencies in correct order:

```typescript
import { composeApplication, type AppConfig, type FactoryRegistry } from '@legible/foundation';

const config: AppConfig = {
  consensusParams: { ... },
  vestingParams: { ... },
  disputeParams: { ... },
};

const factories: Required<FactoryRegistry> = {
  knowledge: createKnowledgeService,
  consensus: createConsensusEngine,
  rewards: createRewardsEngine,
  disputes: createDisputeResolver,
  identity: createIdentityService,
};

// Wire up all services
const services = composeApplication(stateProvider, eventBus, factories, config);

// Use services
await services.knowledge.submitClaim(claim, submitter);
await services.consensus.startRound(claimId);
```

### Dependency Graph

```
                    StateProvider
                         │
                    EventBus
                         │
    ┌────────────────────┼────────────────────┐
    │                    │                    │
Identity            Consensus            (params)
    │                    │                    │
    │              ┌─────┴─────┐              │
    │              │           │              │
    │         Knowledge     (params)          │
    │              │           │              │
    │              └─────┬─────┘              │
    │                    │                    │
    │               Rewards                   │
    │                    │                    │
    │                    │                    │
    └──────────────Disputes───────────────────┘
```

## Mock Implementations

### MockStateProvider

In-memory implementation of StateProvider for testing:

```typescript
import { createMockEnvironment } from '@legible/foundation';

const { state, events, reset } = createMockEnvironment();

// Setup test data
await state.createAccount('did:lgm:alice', 1000n);
await state.registerValidator('pk_123', 'did:lgm:alice', 500n);

// Run test
await myService.doSomething();

// Verify events
const emitted = events.getEmittedEvents('fact_verified');
expect(emitted).toHaveLength(1);

// Reset for next test
reset();
```

### MockEventBus

In-memory implementation of EventBus:

```typescript
// Subscribe to events
const subscription = events.on('block_finalized', (event) => {
  console.log('Block finalized:', event.payload.blockHeight);
});

// Wait for event
const event = await events.waitFor('fact_verified',
  (e) => e.payload.factId === 'fact123',
  5000 // timeout ms
);

// Get event history
const recentEvents = events.getEmittedEvents('block_proposed');
```

### Test Helpers

```typescript
const state = new MockStateProvider();

// Advance block height
state.advanceBlock(5n);

// Reset all state
state.reset();
```

## Architecture Invariants Satisfied

- [x] **ARCH-3**: Dependency Injection - Domain modules receive dependencies via interfaces
- [x] **MOD-1**: Single Responsibility - Container, factories, mocks are separate concerns
- [x] **TEST-1**: Testability - Mock implementations enable isolated testing

## Lazy Service Pattern

For services with expensive initialization:

```typescript
import { lazy, LazyService } from '@legible/foundation';

const heavyService = lazy(() => new ExpensiveService());

// Not created until first access
if (condition) {
  const service = heavyService.value; // Created here
}

// Check if created
if (heavyService.isCreated) {
  // ...
}
```

## ServiceLocator (Migration Aid)

For legacy code that can't use constructor injection:

```typescript
import { ServiceLocator, ServiceKeys } from '@legible/foundation';

// At application startup
ServiceLocator.setContainer(container);

// In legacy code (avoid in new code!)
const knowledge = ServiceLocator.get<KnowledgeService>(ServiceKeys.KnowledgeService);

// Safe access
const rewards = ServiceLocator.tryGet<RewardsEngine>(ServiceKeys.RewardsEngine);
if (rewards) {
  // ...
}
```

⚠️ **Warning**: ServiceLocator is an anti-pattern. Use constructor injection in new code.

## Verification

```bash
# Type check passes
npx tsc --noEmit -p tsconfig.json 2>&1 | grep "foundation/"
# (no errors)

# Property tests pass
npx vitest run tests/property/crypto/signatures.prop.ts
# ✓ 16 tests passed
```

## Usage Example

```typescript
import {
  createContainer,
  ServiceKeys,
  createMockEnvironment,
  composeApplication,
  type AppConfig,
} from '@legible/foundation';

// For testing
describe('KnowledgeService', () => {
  let env: MockEnvironment;
  let services: AppServices;

  beforeEach(() => {
    env = createMockEnvironment();
    services = composeApplication(
      env.state,
      env.events,
      testFactories,
      testConfig
    );
  });

  afterEach(() => {
    env.reset();
  });

  it('should submit claim', async () => {
    await env.state.createAccount('did:lgm:alice', 1000n);

    const result = await services.knowledge.submitClaim(claim, 'did:lgm:alice');

    expect(result.claimId).toBeDefined();
    const events = env.events.getEmittedEvents('claim_submitted');
    expect(events).toHaveLength(1);
  });
});
```

## Next Steps (Phase 4)

### Phase 4: Event-Driven Refactor
- Implement production EventBus (currently mock only)
- Replace direct module calls with event emission
- Add event persistence for replay/audit
- Implement event sourcing patterns

### Migration Path
1. Create factory functions for each domain module
2. Update domain modules to accept dependencies via constructor
3. Wire up production composition root
4. Replace direct instantiation with DI resolution
