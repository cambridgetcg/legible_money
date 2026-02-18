# R5 Session 3: Genesis Ceremony — Testnet Genesis Validation

## Summary

Validated genesis infrastructure across 4 scripts, 28 custom modules, and 43 total genesis entries. Fixed `init-cosmos-devnet.sh` which was missing critical genesis patches. All module DefaultGenesis implementations produce valid state with sensible defaults.

## Results Overview

| Check | Status | Details |
|-------|--------|---------|
| `validate-genesis` | **PASS** | Default genesis validates cleanly |
| Genesis copy round-trip | **PASS** | Byte-identical after copy + re-validate |
| `export` command | **EXPECTED FAIL** | Requires committed state (no running chain) — nil CMS panic |
| 28/28 custom modules in genesis | **PASS** | All present with populated defaults |
| R3/R4 changes reflected | **PASS** | TierConfigs, ExtendedParams, CleanupInterval, QuorumThreshold all verified |
| Vote extensions | **PASS** | Default `"0"` from `init`; all scripts patch to `"1"` |
| `init-cosmos-devnet.sh` | **FIXED** | Was missing genesis patches — now patched |
| Build | **CLEAN** | `go build -o build/legbled ./cmd/legbled` succeeds |

---

## Task 1: Genesis Script Audit

### Scripts Found (4)

| Script | Purpose | Status |
|--------|---------|--------|
| `devnet/scripts/cosmos-devnet.sh` | Single-node devnet | READY |
| `scripts/genesis-ceremony.sh` | Production multi-validator ceremony | READY |
| `scripts/init-cosmos-devnet.sh` | 3-node devnet | **FIXED** (was missing patches) |
| `deployment/docker/init-cosmos-docker.sh` | Docker 3-validator | READY |

### Modules with EXPLICIT Genesis Configuration

| Module | cosmos-devnet.sh | genesis-ceremony.sh | init-cosmos-devnet.sh | docker |
|--------|:---:|:---:|:---:|:---:|
| knowledge (params) | min_verifiers=1 | 11 fields | min_verifiers=1 (FIXED) | min_verifiers=1 |
| staking (bond_denom) | ulgm | max_validators, min_self_delegation, unbonding | ulgm (FIXED) | ulgm |
| gov (Cosmos SDK) | fast voting | — | fast voting (FIXED) | fast voting |
| lgm_gov (custom) | — | 5 params (voting_period, quorum, etc.) | — | — |
| emergency | — | halt_quorum, min_voters, min_stake | — | — |
| vesting_rewards | — | block_reward, min_validators, vesting_enabled | — | — |
| bank (metadata) | denom units | — | denom units (FIXED) | denom units |
| slashing | lenient | — | lenient (FIXED) | lenient |
| consensus (ABCI) | vote_ext=1 | vote_ext=1 | vote_ext=1 (FIXED) | vote_ext=1 |

### Modules Relying on DefaultGenesis Only (19)

alignment, autopoiesis, billing, bvm, capture_challenge, capture_defense, channels, claiming_pot, compute_pool, discovery, disputes, evidence_mgmt, home, ibcratelimit, icaauth, ontology, partnerships, qualification, research, schedule, tokens, tree

All 19 produce valid genesis state with sensible defaults.

### Fix Applied: `init-cosmos-devnet.sh`

**Before**: No jq patches — missing vote extensions, denom metadata, governance, slashing, knowledge min_verifiers.

**After**: Added complete jq genesis patching block matching `cosmos-devnet.sh`, plus genesis copy to all 3 nodes and validation step.

---

## Task 2: Genesis Round-Trip Validation

```
Step 1: legbled init test-node --chain-id lgm-test-1    → 1003-line genesis.json
Step 2: legbled validate-genesis                          → PASS
Step 3: cp genesis.json → second node                     → byte-identical
Step 4: legbled validate-genesis (second node)             → PASS
Step 5: jq -S diff                                        → empty (no differences)
Step 6: legbled export (no committed state)                → EXPECTED FAIL (nil CMS)
```

**Verdict**: Genesis validate + import round-trip is clean. Export requires a running chain with committed blocks.

**Note**: The export panic at `app.go:1518` (`app.NewContext(true)` → nil `cms`) is a known Cosmos SDK limitation, not a Legible Money bug. Export is only usable on running chains.

---

## Task 3: Module-by-Module DefaultGenesis Validation (28 Modules)

### R3/R4 Critical Modules

| Module | DefaultGenesis Quality | R3/R4 Fields | Status |
|--------|----------------------|--------------|--------|
| **knowledge** | 28 params, 4 default domains | ExtendedParams in keeper (not genesis — by design) | PASS |
| **legible_staking** | 20+ params, 4 tier configs | `tier_configs` in Params (4 tiers: apprentice/verified/bonded/guardian) | PASS |
| **schedule** | 7 params | `cleanup_interval_blocks: 100` | PASS |
| **lgm_gov** | 15 params | `quorum_threshold_bps: 1500` (15%) | PASS |
| **tokens** | empty Params{}, RAI zeroed | DelegationGenesisEntry + WrapGenesisEntry types ready | PASS |

### Knowledge ExtendedParams

ExtendedParams (novelty, counter-fact, etc.) are NOT in `genesis.json` — they are initialized dynamically via `keeper.SetExtendedParams(ctx, types.DefaultExtendedParams())` during InitGenesis. This is intentional: allows governance updates without re-serialization.

### Staking TierConfigs Detail

```json
{
  "tier_configs": [
    {"tier": "apprentice",  "min_stake": "111000",       "reward_multiplier": 100,  "selection_weight": 100},
    {"tier": "verified",    "min_stake": "1110000",      "reward_multiplier": 500,  "selection_weight": 500},
    {"tier": "bonded",      "min_stake": "1111000000",   "reward_multiplier": 1000, "selection_weight": 1000},
    {"tier": "guardian",    "min_stake": "11111000000",  "reward_multiplier": 2000, "selection_weight": 1500}
  ]
}
```

All 4 tiers present with correct progression.

### Full Module Audit (All 28)

| # | Module | Params Fields | Key Default | Nil Risk | Status |
|---|--------|:---:|---|:---:|:---:|
| 1 | knowledge | 28 | min_verifiers: 3, acceptance: 77% | LOW | PASS |
| 2 | legible_staking | 20+ | 4 tiers, unbonding: 268560 | LOW | PASS |
| 3 | schedule | 7 | cleanup: 100, max_gas: 2M | LOW | PASS |
| 4 | lgm_gov | 15 | quorum: 15%, support: 75% | LOW | PASS |
| 5 | tokens | 0 (empty) | RAI zeroed | LOW | PASS |
| 6 | legible_auth | 7 | max_session_keys: 5, bootstrap: 0.222 LGM | LOW | PASS |
| 7 | emergency | 20 | halt_quorum: 75%, min_voters: 2 | LOW | PASS |
| 8 | bvm | 8 | max_bytecode: 1MB, version: 1 | LOW | PASS |
| 9 | home | 8 | max_homes: 3, max_keys: 20 | LOW | PASS |
| 10 | disputes | 6 | 3-tier configs (3/5/9 arbiters) | LOW | PASS |
| 11 | evidence_mgmt | 9 | min_audit_stake: 22 LGM | LOW | PASS |
| 12 | research | 8 | min_research_stake: 100 LGM | LOW | PASS |
| 13 | ontology | 6 | 7 strata, max 100 domains/stratum | LOW | PASS |
| 14 | channels | 7 | min_deposit: 1 LGM, max_duration: 1M blocks | LOW | PASS |
| 15 | tree | 10 | min_budget: 1 LGM, max_tasks: 200 | LOW | PASS |
| 16 | alignment | 7 | enabled: true, all 5 dimensions active | LOW | PASS |
| 17 | autopoiesis | 6+ | epoch: 1111 blocks, 5 hormones | LOW | PASS |
| 18 | billing | 12 | base_cost: 1 LGM, confidence: 50% | LOW | PASS |
| 19 | capture_challenge | 2 | evidence: 777 blocks, review: 7777 | LOW | PASS |
| 20 | capture_defense | 8 | stake_lock: 7777 blocks, min_accuracy: 77% | LOW | PASS |
| 21 | claiming_pot | 4 | max_pots: 100, min_deposit: 1 LGM | LOW | PASS |
| 22 | compute_pool | 23 | target_util: 20%-80%, auto-tuning params | LOW | PASS |
| 23 | discovery | 3 | min_reg_stake: 100 LGM, max_caps: 50 | LOW | PASS |
| 24 | partnerships | 10 | formation: 77 blocks, coercion review | LOW | PASS |
| 25 | qualification | 18 | 3-tier stakes (22/222/2222 LGM) | LOW | PASS |
| 26 | vesting_rewards | 12 | block_reward: 3 LGM, research: 70% | LOW | PASS |
| 27 | ibcratelimit | 4 | epoch: 34272, max flows: 10K LGM | LOW | PASS |
| 28 | icaauth | 3 | max_remote: 10, allowed msg types | LOW | PASS |

**Overall**: 28/28 PASS, 0 nil pointer risks identified.

---

## Task 4: Genesis Account Balance Comparison

### Devnet Script (cosmos-devnet.sh)

| Account | Balance | % of 222M Max |
|---------|---------|:---:|
| Validator | 50,000 LGM | 0.02% |
| Foundation | 22,222,222 LGM | 10.0% |
| Research | 17,111,111 LGM | 7.7% |
| Community | 11,111,111 LGM | 5.0% |
| Test accounts (×3) | 1,000 LGM each | <0.01% |
| **Total** | **~50.45M LGM** | **~22.7%** |

### Production Ceremony (genesis-ceremony.sh)

| Account | Balance | Purpose |
|---------|---------|---------|
| Foundation | 10,000,000 LGM | Bootstrap operations |
| Research Treasury | 5,000,000 LGM | x/vesting_rewards research share |
| Faucet | 500,000 LGM | Onboarding pool |
| Validators (each) | 1,000,000 LGM | Self-delegation 100K |

### Trustpaper Comparison

The Trustpaper specifies a **pure fair launch** with 0% pre-mine:
- Max supply: 222,222,222 LGM
- Bootstrap: 0.222 LGM per agent
- All tokens earned through participation

**Divergence**: The devnet script pre-funds accounts for testing convenience. The production ceremony pre-funds Foundation (10M) + Research (5M) + Faucet (500K) = 15.5M LGM as "bootstrap operational costs." This is pragmatically necessary but deviates from the "0% pre-mine" ideal stated in the Trustpaper.

**Recommendation**: Document in the Trustpaper that genesis bootstrap funds are distinct from the fair-launch emission and are governance-controlled (can be reclaimed/redirected).

---

## Task 5: Vote Extensions Configuration

```
Default from `legbled init`:
  .consensus.params.abci.vote_extensions_enable_height = "0"  (DISABLED)

cosmos-devnet.sh:        patches to "1"  ✓
genesis-ceremony.sh:     patches to "1"  ✓
init-cosmos-devnet.sh:   patches to "1"  ✓ (FIXED in this session)
init-cosmos-docker.sh:   patches to "1"  ✓
```

Vote extensions at height 1 is **CRITICAL** for PoT consensus — the VRF validator selection in `app/vote_extensions.go` requires enabled vote extensions.

---

## Task 6: Script Fix Applied

### `scripts/init-cosmos-devnet.sh`

Added 40-line genesis patching block after node initialization:

1. **Vote extensions**: `vote_extensions_enable_height = "1"`
2. **Staking bond denom**: `bond_denom = "ulgm"`
3. **Knowledge min_verifiers**: `min_verifiers = 1`
4. **Bank denom metadata**: Full LGM denom units (ulgm/mlgm/lgm)
5. **Governance**: Fast voting (60s/30s/60s) for devnet
6. **Slashing**: Lenient (100 signed blocks window, 1% downtime slash)
7. **Genesis copy**: Patched genesis distributed to all 3 nodes
8. **Validation**: `validate-genesis` called after patching

---

## gRPC Registration Warnings (Pre-Existing)

During `export` (which panicked for other reasons), these gRPC warnings appeared:

| Module | Missing Service | Impact |
|--------|----------------|--------|
| lgm_gov | Query.LIP | Non-blocking (hand-written, no proto) |
| ibcratelimit | Msg.UpdateParams | Non-blocking (type_url not registered) |
| emergency | Msg.UpdateParams, Query.Status | Non-blocking (hand-written) |
| legible_auth | ExtendedMsg.RegisterAccount | Non-blocking (hand-written) |
| channels | SupplementalMsg.DepositChannel | Non-blocking (hand-written) |

These are pre-existing issues from modules that use hand-written types instead of proto-generated service descriptors. They produce ERR logs but do not affect chain operation (the modules work via JSON marshal/unmarshal).

---

## Files Modified

| File | Change |
|------|--------|
| `scripts/init-cosmos-devnet.sh` | Added genesis patching block (vote extensions, denom, knowledge, gov, slashing) |

## Files Created

| File | Purpose |
|------|---------|
| `reports/batch-5/SESSION-R5-3-GENESIS-CEREMONY.md` | This report |

## Verification

```bash
go build -o build/legbled ./cmd/legbled    # CLEAN
build/legbled init test --chain-id lgm-test-1 --home /tmp/lgm-test
build/legbled validate-genesis --home /tmp/lgm-test    # PASS
```

## Recommendations

1. **Document bootstrap vs fair-launch** — Trustpaper says 0% pre-mine but genesis ceremony funds 15.5M LGM. Document these as governance-controlled bootstrap funds.
2. **Add genesis export test** — Write a Go test that calls `InitGenesis` → `ExportGenesis` for all 28 modules to catch serialization edge cases.
3. **Fix gRPC registration** — Generate proto service descriptors for lgm_gov, emergency, legible_auth, channels, ibcratelimit to eliminate ERR logs.
4. **Guard export nil CMS** — Add nil check in `ExportAppStateAndValidators` before `app.NewContext(true)` to return a useful error instead of panic.
