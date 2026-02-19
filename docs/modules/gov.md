# x/gov — Governance (LIP Lifecycle)

## Purpose

Protocol governance via Legible Improvement Proposals (LIPs) with quadratic voting and stake-weighted participation. Supports category-based stake requirements, 7-stage lifecycle, software upgrade scheduling via x/upgrade, and treasury disbursements with timelock.

## Key Concepts

- **LIP Stages**: draft → review → last_call → voting → accepted/rejected/withdrawn
- **Categories**: consensus, economics, interface, process (each with different stake requirements)
- **Quadratic Voting**: Vote cost scales quadratically; stake-weighted
- **Disbursements**: Treasury spending proposals with separate voting + timelock
- **BigInt Tallying**: Uses `math/big` to prevent uint64 overflow in vote sums

## CLI Commands

### Transactions
```bash
legbled tx lgm_gov submit-lip --title "LIP-1" --description "..." --category consensus --from alice
legbled tx lgm_gov stake-lip [lip-id] --amount 5000000ulgm --from bob
legbled tx lgm_gov advance-lip-stage [lip-id] --from alice
legbled tx lgm_gov cast-vote [lip-id] --support --votes 3 --from bob
legbled tx lgm_gov withdraw-lip [lip-id] --from alice
legbled tx lgm_gov attach-upgrade-plan [lip-id] --name "v0.2.0" --height 500000 --from authority
legbled tx lgm_gov submit-disbursement --title "..." --amount 1000000ulgm --recipient [addr] --from alice
legbled tx lgm_gov execute-disbursement [id] --from authority
legbled tx lgm_gov cancel-disbursement [id] --from alice
```

### Queries
```bash
legbled query lgm_gov params
legbled query lgm_gov lip [id]
legbled query lgm_gov lips
legbled query lgm_gov disbursement [id]
```

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| voting_period_blocks | 17,136 | ~12 hours voting |
| review_period_blocks | 8,568 | ~6 hours review |
| support_threshold_bps | 500,000 | 50% for acceptance |
| quorum_threshold_bps | 333,333 | 33.3% quorum |
| disbursement_timelock_blocks | 6,854 | ~4.8 hours after approval |

15 total governance-adjustable parameters.

## Events

`lip_submitted`, `lip_staked`, `vote_cast`, `lip_advanced`, `lip_withdrawn`, `lip_resolved`, `disbursement_submitted`, `disbursement_executed`, `disbursement_cancelled`, `upgrade_scheduled`, `params_updated`

## Dependencies

StakingKeeper (optional, quorum), BankKeeper (optional, treasury), UpgradeKeeper (optional, software upgrades)
