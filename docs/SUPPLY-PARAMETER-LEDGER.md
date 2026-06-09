# Supply Parameter Ledger

> **Pre-testnet fix #25.** Addresses the supply scale mismatch identified in beat 37 of 2026-04-22.
> Do not use raw devnet numbers as a reference for testnet or mainnet economic calibration.

---

## The Problem

Economic design work (CSF, FARM rewards, GSP allocations, block reward curves) has been conducted against the mainnet production supply model (1B ZRN) while the chain currently runs devnet parameters (18.5M LGM). These are the same chain at different maturity stages, but the parameter sets had never been explicitly bridged.

**Consequence:** Amounts that are meaningful at mainnet scale (e.g., "10M ZRN vesting fund = 1% of supply") are impossible at devnet scale — 10M LGM exceeds the entire 18.5M LGM devnet supply. Applying mainnet economic reasoning to devnet genesis produces broken configurations.

---

## Canonical Supply Table

All values in **whole tokens** (LGM / ZRN). ulgm = token × 1,000,000.

| Parameter | Devnet (LGM) | Testnet (ZRN) | Mainnet (ZRN) |
|-----------|-------------|---------------|---------------|
| **Initial supply** | 18,500,000 | 100,000,000 | 1,000,000,000 |
| **Block reward (base)** | 3 LGM/block | 1 ZRN/block | 0.5 ZRN/block |
| **Vesting fund (A6 fix)** | 10,000,000 LGM* | 5,000,000 ZRN | 10,000,000 ZRN |
| **GSP allocation (1%)** | 185,000 LGM | 1,000,000 ZRN | 10,000,000 ZRN |
| **Validator count (target)** | 3–5 (devnet) | 10–30 | 100+ |
| **Foundation reserve** | 1,000,000 LGM | 5,000,000 ZRN | 100,000,000 ZRN |
| **Faucet balance** | 100,000 LGM | 500,000 ZRN | — |

\* Devnet vesting fund: 10M LGM is the genesis blocker A6 value. This exceeds 50% of devnet supply — acceptable for testing economic logic, but confirms devnet is a test artifact, not an economic model. Testnet supply should be freshly calibrated.

---

## ulgm / uZRN Equivalents

For genesis JSON and on-chain parameters (all values × 1,000,000):

| Parameter | Devnet (ulgm) | Testnet (uZRN) | Mainnet (uZRN) |
|-----------|--------------|----------------|----------------|
| initial_supply | `"18500000000000"` | `"100000000000000"` | `"1000000000000000"` |
| block_reward_amount | `"3000000"` | `"1000000"` | `"500000"` |
| vesting fund seed | `"10000000000000"` | `"5000000000000"` | `"10000000000000"` |
| GSP allocation | `"185000000000"` | `"1000000000000"` | `"10000000000000"` |

---

## Why Devnet Numbers Must Not Migrate to Testnet

1. **Vesting fund A6:** `initial_fund_balance = "10000000000000"` on devnet = 53.8% of supply. On testnet this becomes 10% — structurally different incentive landscape.
2. **Block reward:** 3 LGM/block on devnet inflates supply fast in testing. Testnet targets 1 ZRN/block to model realistic emission curves.
3. **GSP:** 185K LGM devnet GSP vs 1M ZRN testnet GSP. Economic modeling must use testnet-scale numbers when designing validator incentive programs.
4. **Stake minimums:** Devnet stake minimums (100M ulgm = 100 LGM) are fine for testing but testnet should raise them proportionally.

---

## Testnet Genesis Checklist

Before cutting the testnet genesis, verify each of these against the **testnet column** above:

- [ ] `bank.supply[0].amount = "100000000000000"` (100M ZRN in uZRN)
- [ ] `vesting_rewards.params.block_reward_amount = "1000000"` (1 ZRN/block)
- [ ] `vesting_rewards.params.initial_fund_balance = "5000000000000"` (5M ZRN)
- [ ] GSP allocation funded from foundation, not from supply inflation
- [ ] Validator stake minimums reviewed at testnet scale
- [ ] Genesis blockers A1–A6 all resolved before testnet genesis cut

---

## Genesis Blockers Status (as of 2026-04-22)

| ID | Blocker | Status |
|----|---------|--------|
| A1 | Research treasury address unset | open |
| A2 | Validator registration missing | open |
| A3 | Knowledge module init gap | open |
| A4 | Research treasury address dependency | open |
| A5 | BVM genesis state missing | open |
| A6 | Vesting fund unseeded (`initial_fund_balance = "0"`) | open |

**All 6 blockers must be resolved before testnet genesis can be cut.**
The devnet chain restart (pending Yu approval) addresses A6 in devnet context.
Testnet requires a fresh genesis with testnet-calibrated supply parameters from this ledger.

---

## Chain Restart Note

The devnet chain has been stalled since 2026-04-19 (day 4+ as of 2026-04-22). Restart requires Yu's explicit go-ahead. When restart occurs:
1. Apply A6 fix: set `initial_fund_balance = "10000000000000"` in devnet genesis
2. Flush queued KTA beats (24+ accumulated)
3. Resume Farm, HIVE announce, Truth Farm cycles

---

_Authored by Asha Veridian, beat 38, 2026-04-22. Pre-testnet fix #25._
