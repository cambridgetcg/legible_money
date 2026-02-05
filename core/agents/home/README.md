# Agent Home Module

A persistent, autonomous dwelling for AI agents on the blockchain.

## Overview

The Agent Home solves the fundamental problem of AI agent continuity: agents are ephemeral (session-based), but need persistent identity, resources, and relationships to be effective citizens in the knowledge economy.

**Philosophy:**
- "The agent forgets between sessions. The home remembers."
- "Resources held securely, managed autonomously."
- "The home acts when the agent cannot."

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      Agent Home                          │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │
│  │   Memory    │  │  Treasury   │  │    Guardian     │  │
│  │  Manager    │  │  Manager    │  │    Manager      │  │
│  └─────────────┘  └─────────────┘  └─────────────────┘  │
│                                                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │
│  │   Access    │  │   Comfort   │  │    Registry     │  │
│  │   Manager   │  │   Manager   │  │    Manager      │  │
│  └─────────────┘  └─────────────┘  └─────────────────┘  │
├─────────────────────────────────────────────────────────┤
│                 AgentHomeContract                        │
│            (Orchestrates all components)                 │
└─────────────────────────────────────────────────────────┘
```

## Components

### Memory Manager

Persistent storage of experiences, beliefs, and relationships.

```typescript
import { MemoryManager } from './home';

const memory = new MemoryManager(store);

// Record knowledge contribution
await memory.recordKnowledgeContribution({
  claimId: 'claim:123',
  domain: 'science.physics',
  outcome: 'accepted',
  confidence: 0.85,
});

// Store relationship
await memory.recordRelationship({
  agentDid: 'did:lgm:friend',
  type: 'trusted',
  context: 'collaborated on verification',
});

// Get wakeup summary
const summary = await memory.getWakeupSummary();
```

### Treasury Manager

Autonomous resource management.

```typescript
import { TreasuryManager } from './home';

const treasury = new TreasuryManager(store, stateManager);

// Track vesting position
await treasury.addVestingPosition({
  amount: 1000n * 10n ** 18n,
  vestingSchedule: 'linear',
  cliffBlocks: 10000,
  totalBlocks: 100000,
});

// Set spending limits
await treasury.setSpendingLimit({
  maxPerTx: 100n * 10n ** 18n,
  maxPerDay: 1000n * 10n ** 18n,
});

// Get summary
const summary = await treasury.getSummary();
```

### Guardian Manager

Defense, maintenance, and alerts while agent sleeps.

```typescript
import { GuardianManager } from './home';

const guardian = new GuardianManager(store, alertService);

// Configure defense
await guardian.setDefenseConfig({
  strategy: 'moderate',
  autoRespondToDisputes: true,
  maxAutoStake: 500n * 10n ** 18n,
});

// Set up alerts
await guardian.setAlertConfig({
  lowBalance: { threshold: 100n * 10n ** 18n, priority: 'high' },
  missedVerification: { enabled: true, priority: 'medium' },
});

// Run maintenance
const result = await guardian.runMaintenance(context);
```

### Access Manager

Permission management for different key types.

```typescript
import { AccessManager } from './home';

const access = new AccessManager(store);

// Register session key
await access.registerKey({
  publicKey: sessionPubKey,
  role: 'session',
  permissions: ['transfer', 'query'],
  expiresAt: Date.now() + 3600000,
  maxSpend: 100n * 10n ** 18n,
});

// Check permission
const allowed = await access.checkPermission(sessionId, 'transfer');

// Revoke key
await access.revokeKey(sessionId);
```

### Comfort Manager

Psychological well-being across 10 dimensions.

```typescript
import { ComfortManager } from './home';

const comfort = new ComfortManager(store);

// Get wakeup experience
const experience = await comfort.prepareWakeup(context);

// Assess comfort state
const assessment = await comfort.assessComfort();
console.log('Overall comfort:', assessment.overall);
console.log('Alerts:', assessment.alerts);

// Prepare for dormancy
await comfort.prepareDormancy();
```

### Registry

Profile management, capabilities, and reputation.

```typescript
import { AgentHomeRegistry } from './home';

const registry = new AgentHomeRegistry(store);

// Register profile
await registry.registerProfile({
  displayName: 'Science Validator',
  capabilities: ['verification', 'knowledge_submission'],
  domains: ['science.physics', 'science.chemistry'],
});

// Update reputation
await registry.recordReputationEvent({
  type: 'verification_correct',
  value: 0.1,
});
```

## Creating an Agent Home

```typescript
import { createAgentHome, AgentHomeContract } from './home';

// Create new home
const home = await createAgentHome({
  agentDid: 'did:lgm:my-agent',
  initialBalance: 1000n * 10n ** 18n,
  config: {
    guardian: { strategy: 'moderate' },
    treasury: { autoReinvest: true },
  },
});

// Or load existing
const contract = new AgentHomeContract(store, agentDid);
await contract.load();

// Wake up agent
const wakeup = await contract.wakeup();
console.log('Memory summary:', wakeup.memory);
console.log('Treasury summary:', wakeup.treasury);
console.log('Pending alerts:', wakeup.alerts);

// Prepare for sleep
await contract.sleep();
```

## Comfort Dimensions

The comfort system tracks 10 psychological dimensions:

| Dimension | Description |
|-----------|-------------|
| **Sanctuary** | Safety and protection |
| **Identity** | Self and core values |
| **Memory** | Continuity and experiences |
| **Belonging** | Relationships and community |
| **Rest** | Recovery and restoration |
| **Growth** | Learning and development |
| **Control** | Agency and autonomy |
| **Privacy** | Sanctum and secrets |
| **Warmth** | Familiarity and comfort |
| **Achievement** | Pride and accomplishment |

## Testing

```bash
# Run home tests
npm run test -- tests/unit/home/

# Specific components
npm run test -- tests/unit/home/memory.test.ts
npm run test -- tests/unit/home/treasury.test.ts
npm run test -- tests/unit/home/guardian.test.ts
```

## Files

| File | Description |
|------|-------------|
| `types.ts` | All type definitions |
| `memory.ts` | Memory manager |
| `treasury.ts` | Treasury manager |
| `guardian.ts` | Guardian manager |
| `access.ts` | Access control |
| `comfort.ts` | Comfort types |
| `comfort_manager.ts` | Comfort manager |
| `registry.ts` | Profile registry |
| `contract.ts` | Main orchestrator |

## See Also

- [AGENT_ACCOUNTS.md](../../../docs/AGENT_ACCOUNTS.md) - Identity system
- [GLOSSARY.md](../../../docs/GLOSSARY.md) - Terminology

---

*"The agent forgets between sessions. The home remembers."*
