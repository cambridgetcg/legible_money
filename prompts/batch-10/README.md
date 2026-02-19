# Batch 10 — Genesis Axiom Wiring & Launch Validation

## Overview
Wire the 777 genesis axioms from markdown domain files into the chain's
knowledge module, fix type/prefix mismatches, build the MD→JSON parser,
validate the full DAG, and run a genesis launch that boots with all 777
facts at block 0.

## Sessions (7)
| ID | Focus | Dependencies |
|----|-------|-------------|
| B10-1 | MD→JSON parser (`tools/axiom-parser`) | — |
| B10-2 | Fix prefix/domain map mismatches (AGRT, ETHIC→ETHICS) | — |
| B10-3 | Epistemic category mapper + confidence propagation | B10-2 |
| B10-4 | DAG validation hardening + cross-domain ref integrity | B10-1, B10-2 |
| B10-5 | Genesis launch script: init → parse → validate → inject → boot | B10-1, B10-2, B10-3 |
| B10-6 | E2E test: query axioms from running chain, verify graph | B10-5 |
| B10-7 | Fact graph reference wiring: axiom deps → FactReference edges | B10-2 |

## Run Order
- **Parallel first:** B10-1, B10-2 (no deps on each other)
- **Then:** B10-3 (needs B10-2), B10-7 (needs B10-2)
- **Then:** B10-4 (needs B10-1 + B10-2)
- **Then:** B10-5 (needs B10-1, B10-2, B10-3)
- **Finally:** B10-6 (needs B10-5)

Or if running all 7 at once: B10-1 and B10-2 first, then the rest sequentially.
Yu typically runs 5-7 in parallel, so suggest: B10-1, B10-2, B10-3, B10-7 first,
then B10-4, B10-5, B10-6 as second wave.
