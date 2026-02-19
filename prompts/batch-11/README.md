# Batch 11 — Anti-Farming Hardening & Incentive Alignment

## Overview
Close 6 economic loopholes identified during the financial incentive audit.
The core issue: the knowledge module rewards truth-seeking but has gaps that
allow farming through genesis-axiom restating, sybil citation rings, domain
ceiling abuse, prophetic manipulation, difficulty bonus skew, and falsification
reward asymmetry.

## Sessions (6)
| ID | Focus | FARM ID | Priority |
|----|-------|---------|----------|
| B11-1 | Semantic novelty floor — block axiom-restating claims | FARM-10 | Critical |
| B11-2 | Citation concentration detection — sybil ring defense | FARM-11 | High |
| B11-3 | Domain stratum governance — require stratum in proposals | FARM-12 | High |
| B11-4 | Prophetic diversity gate — require diverse citers | FARM-13 | Medium |
| B11-5 | Difficulty bonus normalization — adjust by domain ceiling | FARM-14 | Medium |
| B11-6 | Falsification reward scaling — age×citation weighted | FARM-15 | Low |

## Run Order
All 6 are independent — can run in parallel.
B11-1 is the highest priority (exploitable on day 1 of genesis).

## Context for All Sessions
- Repo: `~/Desktop/legible_money`
- Latest commit: `eb8491c`
- Knowledge module: `x/knowledge/`
- Types: `x/knowledge/types/types.go` (ExtendedParams)
- Evaluation pipeline: `x/knowledge/evaluation/`
- Keeper: `x/knowledge/keeper/`
- Existing FARM defenses: FARM-1 through FARM-9, AH-3, AH-4
- Extended params pattern: add constant → add field to ExtendedParams → add default → use in keeper
- All BPS values use 1,000,000 = 100%
