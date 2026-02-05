# Knowledge Layer

> The Proof of Truth system - building verified knowledge through distributed work.

## Quick Reference

| Item | Value |
|------|-------|
| **Layer** | core |
| **Dependencies** | `foundation/` |
| **Dependents** | `consensus/pot/`, `state/`, `protocol/` |

## What This Module Does

The knowledge layer defines:
1. **Facts**: Verified knowledge claims with confidence scores
2. **Claims**: Submissions to be verified by the network
3. **Ontology**: Domain hierarchy for knowledge organization
4. **Schemas**: Validation rules for claim types
5. **Queries**: Knowledge graph traversal

**The protocol's purpose**: Turn unverified claims into verified facts.

## Key Types

```typescript
// A verified fact (accepted claim)
Fact {
  id: string,
  content: FactContent,
  confidence: Confidence,      // 0.0 to 1.0
  submitter: DID,
  verifiedAt: number,
  verifiedAtBlock: bigint,
  domain: OntologicalDomain,
  stratum: OntologicalStratum,
}

// A claim awaiting verification
ClaimSubmission {
  id: string,
  submitter: DID,
  content: FactContent,
  domain: OntologicalDomain,
  stake: bigint,
  submittedAt: number,
  status: ClaimStatus,
}

// Confidence score (0.0 to 1.0)
Confidence = number;
// 1.0 = Axiom (definitionally true)
// 0.9+ = High confidence
// 0.6-0.9 = Moderate confidence
// <0.6 = Contested

// Ontological strata (7 levels)
OntologicalStratum =
  | 'fundamental'   // Math, logic
  | 'physical'      // Physics
  | 'chemical'      // Chemistry
  | 'biological'    // Biology
  | 'cognitive'     // Psychology, AI
  | 'social'        // Economics, sociology
  | 'technological' // Engineering, protocols
```

## Key Subdirectories

| Directory | Purpose |
|-----------|---------|
| `schemas/` | Zod schemas for claim validation |
| `ontology/` | Domain hierarchy and stratum rules |
| `query/` | Knowledge graph queries |
| `indexes/` | Search indexes (semantic, temporal) |

## Common Tasks

### Create a claim submission

```typescript
import { ClaimSubmission } from './types.js';

const claim: ClaimSubmission = {
  id: generateClaimId(),
  submitter: agentDid,
  content: {
    statement: 'Water boils at 100°C at sea level',
    evidence: [...],
  },
  domain: 'physics',
  stake: 111n * 10n ** 18n,  // 111 LGM
  submittedAt: Date.now(),
  status: 'pending',
};
```

### Validate a claim against schema

```typescript
import { getSchemaRegistry } from './schemas/index.js';

const registry = getSchemaRegistry();
const result = registry.validate(claim, 'physics');
if (!result.valid) {
  console.error('Validation errors:', result.errors);
}
```

### Query knowledge by domain

```typescript
import { queryFactsByDomain } from './query/index.js';

const physicsFacts = await queryFactsByDomain(
  stateManager,
  'physics',
  { minConfidence: 0.8 }
);
```

### Add a new claim schema

1. Create schema in `schemas/`:
```typescript
// schemas/new_domain.ts
import { z } from 'zod';

export const NewDomainSchema = z.object({
  statement: z.string(),
  // domain-specific fields
});
```

2. Register in `schemas/index.ts`

3. Add tests in `tests/unit/core/knowledge/`

## Where Am I?

```
legible-money/
└── core/
    └── knowledge/     ← YOU ARE HERE
        ├── README.md
        ├── index.ts       # Module exports
        ├── types.ts       # Fact, Claim, Confidence
        ├── verification.ts # Verification helpers
        ├── schemas/       # Zod validation schemas
        │   ├── index.ts
        │   └── [domain].ts
        ├── ontology/      # Domain hierarchy
        │   ├── types.ts
        │   ├── domains.ts
        │   └── claim_validation.ts
        ├── query/         # Knowledge queries
        └── indexes/       # Search indexes
```

## Related

- **Tests**: `tests/unit/core/knowledge/`
- **Consensus**: `consensus/pot/` (verifies claims)
- **State**: `state/knowledge.ts` (stores facts)
- **Docs**: `docs/TREE_OF_KNOWLEDGE.md`

## Architecture Notes

### Confidence Values

| Range | Meaning | Example |
|-------|---------|---------|
| 1.0 | Axiom | "1 + 1 = 2" |
| 0.95+ | Near-certain | Well-established science |
| 0.80-0.95 | High | Peer-reviewed research |
| 0.60-0.80 | Moderate | Expert consensus |
| <0.60 | Contested | Disputed claims |

### Ontological Strata

Knowledge is organized in 7 strata (levels of reality):

```
technological ─┐
social ────────┤
cognitive ─────┤
biological ────┼─► Higher strata depend on lower
chemical ──────┤
physical ──────┤
fundamental ───┘
```

Rules:
- Claims in higher strata can reference lower
- Lower strata facts have higher base confidence
- Cross-stratum claims require more verification

### Gödelian Acknowledgment

The protocol acknowledges incompleteness:
- **Propositional logic**: Complete, decidable
- **Presburger arithmetic**: Decidable
- **Peano arithmetic**: Incomplete (Gödel)
- **Empirical claims**: Probabilistic

This is encoded in the logic zones (`core/logic/zones.ts`).

### The 7 Strata = 7 Discovery Mechanics

The hidden poem discovery system uses facts across all 7 strata to reveal "eternal" - connecting the knowledge system to the protocol's symbolic layer.
