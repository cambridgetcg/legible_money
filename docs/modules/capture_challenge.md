# x/capture_challenge — Validator Capture Challenges

## Purpose

Challenge mechanism for detecting validator capture (coordinated voting, off-chain coordination, stake manipulation). Challengers stake tokens, submit evidence across 3 phases, with auto-funded bounty pools per domain.

## Key Concepts

- **Evidence Types**: coordinated_voting, timing_correlation, off_chain_coordination, false_fact_accepted, stake_manipulation, qualification_fraud, systematic_bias
- **3 Phases**: submission → evidence_period (777 blocks) → under_review (7,777 blocks)
- **Bounty Pools**: Per-domain pools auto-funded every 777 blocks
- **Stake**: Minimum 77 LGM, multiplied by 7.7x for bond

## CLI Commands

### Transactions
```bash
legbled tx capture_challenge submit-challenge --accused-validators [addrs] --evidence-json [json] --from alice
legbled tx capture_challenge add-evidence [challenge-id] [evidence-json] --from alice
legbled tx capture_challenge resolve-challenge [challenge-id] --verdict upheld --from authority
```

### Queries
```bash
legbled query capture_challenge challenge [id]
legbled query capture_challenge active-challenges [domain]
legbled query capture_challenge bounty-pool [domain]
```

## Events

`challenge_submitted`, `challenge_phase_advanced`, `evidence_added`, `challenge_resolved`, `bounty_auto_funded`

## Dependencies

CaptureDefenseKeeper, StakingKeeper, BankKeeper
