# Batch 13 — Research Tax Adjustment, Founder Split & Testnet Launch Prep

## Overview
Reduce research tax from 7% to 3.33%, add founder operational split (7% of
research fund = 0.23% total), and prepare for actual testnet launch.

## Decision Log
- Research tax 7% → 3.33%: more goes to verification rewards (truth-seeking)
  and block producers (validator incentive). Fund still meaningful, not burdensome.
- Founder gets 7% of the 3.33% research fund (0.23% of total rewards) for
  operational expenses. Transparent, on-chain, with sunset ratchet.
- Start modest, governance can raise tax later. Easier to raise than lower.

## Sessions (6)
| ID | Focus | Priority |
|----|-------|----------|
| B13-1 | Research tax 7% → 3.33% across all modules | Critical |
| B13-2 | Founder auto-split from research fund | Critical |
| B13-3 | Batch 11 FARM fixes that didn't land (check & apply) | High |
| B13-4 | Genesis launch dry-run — full boot with 777 axioms | Critical |
| B13-5 | CLI UX — human-friendly commands for the cooperation flow | Medium |
| B13-6 | Testnet documentation — README, getting started, FAQ | Medium |

## Run Order
- **Wave 1 (parallel):** B13-1, B13-2, B13-3
- **Wave 2 (parallel):** B13-4, B13-5, B13-6
