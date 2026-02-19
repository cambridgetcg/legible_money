# x/ontology — Knowledge Domain Hierarchy

## Purpose

Multi-stratum domain ontology with Godelian incompleteness awareness. Manages knowledge domains organized by epistemic confidence level (strata), defines logic zones with formal properties, and processes domain proposals through voting.

## Key Concepts

- **Strata**: 7 levels from Axiomatic (0, highest confidence) to Testimonial (6, lowest)
- **Domains**: Knowledge areas within strata with confidence ceilings
- **Logic Zones**: Formal system properties (decidability, completeness, Godel flag)
- **Incompleteness**: Explicit acknowledgment of Godelian limitations per logic zone
- **Domain Proposals**: Stake-backed proposals with voting

## CLI Commands

### Transactions
```bash
legbled tx ontology propose-domain [name] [stratum] --stake 5000000ulgm --from alice
legbled tx ontology vote-proposal [proposal-id] --support --from bob
legbled tx ontology update-domain [domain-id] --description "..." --from authority
legbled tx ontology register-logic-zone [name] --decidable --complete --from authority
legbled tx ontology acknowledge-incompleteness [logic-zone] --statement "..." --from authority
```

### Queries
```bash
legbled query ontology params
legbled query ontology stratum [level]
legbled query ontology all-strata
legbled query ontology domain [id]
legbled query ontology domains-by-stratum [level]
legbled query ontology all-domains
legbled query ontology proposal [id]
legbled query ontology confidence-ceiling [domain]
legbled query ontology logic-zone [name]
legbled query ontology all-logic-zones
```

## Parameters

Min proposal stake, voting periods, confidence ceiling algorithms.

## Events

`DomainProposed`, `DomainVoted`, `DomainCreated`, `LogicZoneRegistered`, `IncompletenessAcknowledged`

## Dependencies

StakingKeeper, KnowledgeKeeper

## EndBlock

Processes expired domain proposals (tally votes, create or reject domains).
