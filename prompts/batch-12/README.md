# Batch 12 — Account Architecture Audit, Cooperation Hardening & E2E Flow Testing

## Overview
Audit of the cryptographic account system, partnership economics, and
human-agent cooperation flows. Fix gaps, add missing joint-account
primitives, and build E2E tests for the practical flow from account
creation through knowledge work to shared rewards.

## Sessions (7)
| ID | Focus | Priority |
|----|-------|----------|
| B12-1 | Partnership reward integration — wire knowledge rewards into partnerships | Critical |
| B12-2 | Research fund governance — treasury with proposal-based disbursement | High |
| B12-3 | Multi-party partnership support — 3+ participant partnerships | Medium |
| B12-4 | Partnership-aware citation rewards — joint credit for partnered work | High |
| B12-5 | Agent home → partnership integration — auto-wire homes to partnerships | Medium |
| B12-6 | E2E flow test: register → partner → submit → verify → earn → split | Critical |
| B12-7 | Session key capability expansion — agent operational autonomy | Medium |

## Run Order
- **Wave 1 (parallel):** B12-1, B12-2, B12-7 (independent)
- **Wave 2 (parallel):** B12-3, B12-4, B12-5 (B12-1 should land first for 4)
- **Wave 3:** B12-6 (needs all others for comprehensive E2E)
