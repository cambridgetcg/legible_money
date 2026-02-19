# Batch 14 — Service Revenue Simplification & Treasury Unification

## Overview
Kill the 5-way evidence tax sub-split in both tree and billing modules.
Replace with a clean 4-way split: contributors get the majority, research
fund gets a fair share with guaranteed 7% founder split, protocol treasury
handles operations, and a burn mechanism provides deflation.

## Key Decisions
- Contributors take 55% (majority — they built the service)
- Research fund takes 13% (with 7% founder auto-split = 0.91% to founder)
- Protocol treasury takes 22% (operations, foundation, community, devs — governed as one)
- Burn takes 10% (deflationary pressure)
- The 5 separate treasury module accounts (research/foundation/community/developers/reserve) are eliminated
- All research fund deposits include the 7% founder auto-split (same as block rewards)
- The tree module's ResearchFundAddr and ProtocolTreasuryAddr are replaced with module accounts

## Sessions (5)
| ID | Focus | Priority |
|----|-------|----------|
| B14-1 | Tree module: simplify to 4-way split, kill evidence tax sub-split | Critical |
| B14-2 | Billing module: simplify to 4-way split, kill 5-way research split | Critical |
| B14-3 | Founder split on service revenue research deposits | Critical |
| B14-4 | Remove dead treasury module accounts | High |
| B14-5 | Integration test: unified revenue across all modules | High |

## Run Order
- **Wave 1 (parallel):** B14-1, B14-2 (independent modules)
- **Wave 2 (parallel):** B14-3, B14-4 (after wave 1)
- **Wave 3:** B14-5 (needs all others)
