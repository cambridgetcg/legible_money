# x/evidence_mgmt — Evidence Management & Oracles

## Purpose

Evidence submissions with multi-auditor scoring and oracle integration. Evidence is collected, audited by multiple independent stake-weighted raters, and consensus determines validity.

## Key Concepts

- **Evidence Types**: Various types submitted for claims
- **Multi-Auditor**: Independent raters score evidence with rationale
- **Oracle Integration**: Registered oracle providers submit typed data
- **Audit Consensus**: Aggregate audit scores determine validity

## CLI Commands

### Transactions
```bash
legbled tx evidence_mgmt submit-evidence [claim-id] [evidence-hash] --type [type] --from alice
legbled tx evidence_mgmt audit-evidence [evidence-id] --score 85 --rationale "..." --from auditor1
legbled tx evidence_mgmt register-oracle [name] --supported-types "weather,market" --from oracle1
legbled tx evidence_mgmt submit-oracle-data [evidence-id] [data-json] --from oracle1
```

### Queries
```bash
legbled query evidence_mgmt evidence [id]
legbled query evidence_mgmt evidence-by-claim [claim-id]
legbled query evidence_mgmt audit [id]
legbled query evidence_mgmt audits-for-evidence [evidence-id]
legbled query evidence_mgmt oracle [address]
legbled query evidence_mgmt active-oracles
legbled query evidence_mgmt consensus [evidence-id]
legbled query evidence_mgmt params
```

## Parameters

Audit threshold, oracle stake requirements, evidence expiry windows.

## Events

`EvidenceSubmitted`, `EvidenceAudited`, `OracleRegistered`, `OracleDataSubmitted`

## Dependencies

BankKeeper, EvidenceResolutionKeeper

## EndBlock

Expires stale evidence, prunes inactive oracles.
