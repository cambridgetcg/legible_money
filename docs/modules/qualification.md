# x/qualification — Domain Qualifications

## Purpose

Domain-specific validator qualifications through 5 pathways. Enforces probationary periods, tracks endorsement decay, checks accuracy thresholds, and enables reputation-weighted qualification decisions.

## Key Concepts

- **5 Pathways**: stake_commitment, track_record, cross_reference, stratum_inheritance, genesis
- **Status**: active → probationary → suspended → revoked → expired
- **Endorsements**: Other qualified validators endorse candidates (min 3 required)
- **Accuracy**: 77% minimum accuracy for qualification (770,000 BPS)
- **Suspension**: Below 50% accuracy triggers automatic suspension

## CLI Commands

### Transactions
```bash
legbled tx qualification request-qualification [domain] --pathway stake_commitment --stake 5000000ulgm --from alice
legbled tx qualification endorse-validator [validator] [domain] --from qualified-validator
legbled tx qualification renew-qualification [domain] --from alice
legbled tx qualification withdraw-qualification [domain] --from alice
```

### Queries
```bash
legbled query qualification params
legbled query qualification qualification [validator] [domain]
legbled query qualification qualifications-by-validator [validator]
legbled query qualification qualified-validators [domain]
legbled query qualification endorsement [endorser] [validator] [domain]
legbled query qualification endorsements-for-validator [validator] [domain]
```

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| MinVerifications | 19 | Min verifications for track_record |
| MinAccuracyBps | 770,000 | 77% accuracy threshold |
| MinEndorsers | 3 | Min endorsements needed |
| SuspensionAccuracyBps | 500,000 | 50% triggers suspension |
| MaxQualificationDepth | 7 | Max inheritance depth |

## Events

`qualification_requested`, `qualification_granted`, `qualification_probationary`, `endorsement_received`, `endorsement_decayed`, `qualification_renewed`, `qualification_suspended`, `qualification_revoked`, `qualification_expired`

## Dependencies

Standalone validation logic (no explicit keeper dependencies).
