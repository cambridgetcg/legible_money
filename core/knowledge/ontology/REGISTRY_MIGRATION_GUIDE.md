# Stratum Registry Migration Guide

This guide explains how to migrate from the old hardcoded stratum constants to the new registry-based approach.

## Why Registry-Based?

The old approach scattered stratum definitions across multiple constants:

```typescript
// OLD: Multiple places to update when adding a stratum
export const STRATUM_HIERARCHY = { fundamental: 0, physical: 1, ... };
export const STRATUM_CONFIDENCE_CEILING = { fundamental: 1.0, physical: 0.99, ... };
export const STRATUM_LOGIC_ZONE = { fundamental: 'peano', physical: 'peano', ... };
// ... 5+ more places
```

The new approach centralizes everything:

```typescript
// NEW: Single definition with all properties
const stratum: StratumDefinition = {
  id: 'physical',
  name: 'Physical',
  level: 1,
  confidenceCeiling: 0.99,
  defaultLogicZone: 'peano',
  defaultEpistemicCategory: 'formal',
  validDependencies: ['fundamental'],
  description: 'Classical and quantum mechanics, thermodynamics',
  exampleDomains: ['physics.mechanics', 'physics.thermodynamics'],
  canReferenceAll: false,
  metadata: { ... }
};
```

## Quick Start

### Using the Default Registry

```typescript
import { defaultRegistry } from './stratum_registry.js';

// Get stratum properties
const level = defaultRegistry.getLevel('physical');              // 1
const ceiling = defaultRegistry.getConfidenceCeiling('biological'); // 0.95
const deps = defaultRegistry.getValidDependencies('chemical');   // ['fundamental', 'physical']

// Check relationships
const isValid = defaultRegistry.isValidDependency('biological', 'chemical'); // true
const isHigher = defaultRegistry.isHigher('cognitive', 'biological');        // true
const distance = defaultRegistry.getDistance('social', 'fundamental');       // 5

// Get full definition
const stratum = defaultRegistry.get('cognitive');
console.log(stratum?.description);
console.log(stratum?.defaultLogicZone);
```

### Dependency Injection (Recommended for New Code)

```typescript
import { OntologyProvider, createOntologyProvider, StratumRegistry } from './stratum_registry.js';

class ProofOfTruthEngine {
  constructor(private ontology: OntologyProvider) {}

  validateClaim(claim: { stratum: string; confidence: number }) {
    const ceiling = this.ontology.getConfidenceCeiling(claim.stratum);
    if (claim.confidence > ceiling) {
      return { valid: false, reason: `Confidence exceeds ceiling ${ceiling}` };
    }
    return { valid: true };
  }
}

// Production: use default registry
const engine = new ProofOfTruthEngine(defaultOntologyProvider);

// Testing: use custom registry
const testRegistry = StratumRegistry.createEmpty();
testRegistry.register({ id: 'test', level: 0, ... }, 0);
const testEngine = new ProofOfTruthEngine(createOntologyProvider(testRegistry));
```

### Backward Compatibility (Existing Code)

Existing code continues to work unchanged:

```typescript
// This still works!
import {
  STRATUM_HIERARCHY,
  STRATUM_CONFIDENCE_CEILING,
  getStratumLevel,
  isValidStratumDependency
} from './types.js';

const level = STRATUM_HIERARCHY['physical'];  // 1
const ceiling = STRATUM_CONFIDENCE_CEILING['biological'];  // 0.95
```

These are now generated from the registry, ensuring consistency.

## Migration Patterns

### Pattern 1: Direct Constant Access → Registry Method

```typescript
// Before
import { STRATUM_HIERARCHY } from './types.js';
const level = STRATUM_HIERARCHY[stratum];

// After
import { defaultRegistry } from './stratum_registry.js';
const level = defaultRegistry.getLevel(stratum);
```

### Pattern 2: Switch on Stratum → Registry Lookup

```typescript
// Before
function getConfidenceCeiling(stratum: OntologicalStratum): number {
  switch (stratum) {
    case 'fundamental': return 1.0;
    case 'physical': return 0.99;
    // ... many cases
  }
}

// After
function getConfidenceCeiling(stratum: string): number {
  return defaultRegistry.getConfidenceCeiling(stratum);
}
```

### Pattern 3: Hardcoded Validation → Registry-Based

```typescript
// Before
const VALID_DEPS: Record<string, string[]> = {
  chemical: ['fundamental', 'physical'],
  biological: ['fundamental', 'physical', 'chemical'],
  // ...
};

function isValid(from: string, to: string): boolean {
  return VALID_DEPS[from]?.includes(to) ?? false;
}

// After
function isValid(from: string, to: string): boolean {
  return defaultRegistry.isValidDependency(from, to);
}
```

### Pattern 4: Class with OntologyProvider Injection

```typescript
// Before: tight coupling
class ClaimValidator {
  validate(claim: Claim): boolean {
    const ceiling = STRATUM_CONFIDENCE_CEILING[claim.stratum];
    return claim.confidence <= ceiling;
  }
}

// After: loose coupling, testable
class ClaimValidator {
  constructor(private ontology: OntologyProvider) {}

  validate(claim: Claim): boolean {
    const ceiling = this.ontology.getConfidenceCeiling(claim.stratum);
    return claim.confidence <= ceiling;
  }
}
```

## Adding New Strata

### Via Governance (Runtime)

```typescript
const result = defaultRegistry.register({
  id: 'ecological',
  name: 'Ecological',
  level: 3.5,  // Between biological and cognitive
  confidenceCeiling: 0.92,
  defaultLogicZone: 'empirical',
  defaultEpistemicCategory: 'empirical',
  validDependencies: ['biological', 'chemical', 'physical', 'fundamental'],
  description: 'Ecosystem dynamics, climate systems, environmental science',
  exampleDomains: ['ecology.climate', 'ecology.ecosystems'],
  canReferenceAll: false,
}, currentBlockNumber, 'LIP-42');

if (!result.success) {
  console.error('Registration failed:', result.errors);
}
```

### Via Migration (Schema Upgrade)

See `stratum_migration.ts` for migration utilities.

```typescript
import { MigrationRunner, migrateRegistry } from './stratum_migration.js';

// Define migration
const migration_v1_to_v2: StratumMigration = {
  fromVersion: 1,
  toVersion: 2,
  description: 'Add ecological stratum',

  mapStratum(old: string): string[] {
    return [old];  // No changes to existing
  },

  getNewStratumDefinition(id: string) {
    if (id === 'ecological') {
      return { /* definition */ };
    }
    return null;
  },

  migrateConfidence(old: number, oldStratum: string, newStratum: string): number {
    return old;  // No confidence changes
  }
};

// Apply migration
const { newRegistry, record } = migrateRegistry(
  defaultRegistry,
  migration_v1_to_v2,
  currentBlockNumber
);
```

## Testing with Custom Ontologies

```typescript
import { StratumRegistry, createOntologyProvider } from './stratum_registry.js';

describe('ClaimValidator', () => {
  it('should reject confidence above ceiling', () => {
    // Create minimal test registry
    const registry = StratumRegistry.createEmpty(1);
    registry.register({
      id: 'test',
      name: 'Test',
      level: 0,
      confidenceCeiling: 0.5,  // Low ceiling for testing
      defaultLogicZone: 'propositional',
      defaultEpistemicCategory: 'formal',
      validDependencies: [],
      description: 'Test stratum',
      exampleDomains: [],
      canReferenceAll: false,
    }, 0);

    const validator = new ClaimValidator(createOntologyProvider(registry));

    expect(validator.validate({ stratum: 'test', confidence: 0.6 })).toBe(false);
    expect(validator.validate({ stratum: 'test', confidence: 0.4 })).toBe(true);
  });
});
```

## File Structure

```
core/knowledge/ontology/
├── types.ts              # Domain types (unchanged)
├── stratum_registry.ts   # NEW: Registry implementation
├── stratum_migration.ts  # NEW: Migration utilities
├── stratum_compat.ts     # NEW: Backward compatibility layer
├── strata.ts             # Legacy functions (now thin wrappers)
├── tree.ts               # Domain tree (unchanged)
├── genesis.ts            # Genesis domains (unchanged)
└── index.ts              # Exports (updated to include registry)
```

## API Reference

### StratumRegistry

| Method | Description |
|--------|-------------|
| `get(id)` | Get stratum definition |
| `has(id)` | Check if stratum exists |
| `getAll()` | Get all strata |
| `getAllIds()` | Get all IDs in level order |
| `getLevel(id)` | Get numeric level |
| `getConfidenceCeiling(id)` | Get max confidence |
| `getValidDependencies(id)` | Get allowed dependencies |
| `isValidDependency(from, to)` | Check dependency validity |
| `isHigher(a, b)` | Compare levels |
| `isLower(a, b)` | Compare levels |
| `getDistance(a, b)` | Get level difference |
| `register(def, block)` | Add new stratum |
| `serialize()` | Export for storage |

### OntologyProvider

Same methods as StratumRegistry, but as an interface for dependency injection.

## Versioning

The registry tracks schema versions:

- `schemaVersion: 1` - Initial 7-stratum hierarchy
- Future versions will be documented here

Each stratum records which schema version it was defined in via `metadata.schemaVersion`.
