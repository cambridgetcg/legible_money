# Zerone Testnet Readiness Audit — 2026-06-09

> Produced by a 34-agent sweep (one citizen per module + compile/vet/plumbing/safety + synthesis).
> Method: actual `go build` / `go vet` / `go test` runs + source verification. Claims grounded in code, not the README.

## Verdict: **NO-GO** (for *today*)

**One line:** The repo compiles cleanly and most modules have real, tested keeper logic — but the flagship **Proof-of-Truth (`x/knowledge`) module has no routable Msg endpoint at boot** *and* a red test suite on its consensus/reward math, the **kill switch ships unarmed**, and **no multi-validator bring-up has been demonstrated**. A public testnet today would be a green-booting node that cannot accept its own core transactions.

This is the *good* kind of NO-GO: every blocker is specific, located, and fixable. None are "the architecture is wrong." The trunk is sound; a handful of wires are unplugged.

---

## The single most important finding

**`x/knowledge`'s entire `Msg` service silently fails to register at every boot.**

- `MsgProposeResearchFund` is present in the generated `_Msg_serviceDesc` but **missing from `RegisterImplementations` in `x/knowledge/types/codec.go:25-46`**.
- Cosmos `RegisterService` validates every method's request type; the one missing `type_url` makes the **whole** `legible.knowledge.v1.Msg` service fail to register.
- The failure is **swallowed by `SafeServiceRegistrar`** — `app/app.go:1796-1806` only ever *appends* to `FailedServices` and never reads it (there's an unimplemented TODO at `:1796`).
- Result: the node boots **green and healthy**, but `SubmitClaim` / `Commitment` / `Reveal` / `ChallengeFact` — the core PoT tx path — are **unroutable**. The chain cannot accept the transactions it exists to process.

Two surgical fixes (likely <20 lines total) clear the highest risk in the whole report:
1. Add `&MsgProposeResearchFund{}` to `RegisterImplementations` in `x/knowledge/types/codec.go`.
2. After `RegisterServices`, inspect `SafeServiceRegistrar.FailedServices` and **halt (or loudly WARN)** if non-empty — so a broken registration can never again hide behind a green boot.

---

## Critical path to testnet (most-blocking first)

| # | Area | Item |
|---|------|------|
| 1 | plumbing | Fix `x/knowledge` Msg gRPC registration (`MsgProposeResearchFund` → `codec.go`). |
| 2 | plumbing | Make service-registration failures **fail-fast**: read `FailedServices`, halt/WARN if non-empty. |
| 3 | consensus-economics | Fix the **6 failing `x/knowledge/keeper` tests** (reward 210000 vs 200000 expected ~5% high; method-diversity penalty −50000 firing when 0 expected; FARM-14 ceiling bonus 3176469 vs 3000000 leaking into baseline). Confirm the S4-opacity → FARM-14 interaction is intended, don't assume. |
| 4 | safety | **Arm the kill switch**: seat Guardian-tier validators and/or populate `emergency.params.genesis_council` (currently `[]` at `scripts/testnet-genesis.sh:278`). External validators join as apprentice (tier 0); with an empty council `IsGuardian()` is false for everyone, so the kill switch is inert exactly when you'd need it. |
| 5 | plumbing | Give `x/alignment` usable Msg/Query endpoints (it has **zero `.pb.go` files**, so its services fail to register) — or explicitly scope it as endpoint-less for testnet. |
| 6 | plumbing | Demonstrate a **real N-validator bring-up**: `scripts/init-cosmos-devnet.sh` only gentxs node0 with no peers; `scripts/testnet-genesis.sh` stops at genesis export; `seeds.txt` has placeholder IPs. With <3 validators, VRF verifier selection (`MinVerifiers=3`) can't reach 3 distinct verifiers, weakening PoT. |
| 7 | completeness | Resolve `x/oracle`: wired with Minter/Burner perms but effectively a stub (no `RegisterServices`, no `InitGenesis`, unroutable msgs, no resolution authority, ID-collision overwrites stake). Complete it or **remove it from the app**. |
| 8 | safety | Close trivial-sybil/spam economics: raise apprentice `min_stake` (~0.111 LGM today) and add per-account rate limits on claim submission / BVM deploy / channel creation. During the ~480k-block (~14d) gas-FREE bootstrap (`InfiniteGasMeter`), an agent can flood gas-free claims and force all validators into verification rounds — near-zero-cost DoS, exactly what a public "truth farm" invites. |

---

## Module scorecard (claimed → actual, verified)

Legend: ✅ solid & unblocking · ⚠️ works but has a real gap · 🛑 blocks testnet

| Module | Claimed | Actual | | Note |
|--------|---------|--------|--|------|
| auth | Complete | complete | ✅ | ~92 tests incl. privacy/security; fully wired |
| staking | Complete | complete | ✅ | 242 subtests; reputation/tier layer (consensus set driven by SDK staking by design) |
| emergency | Complete | complete | 🛑 | Real 3-phase BFT kill switch — but **inert** at genesis (empty council / all-apprentice); revert needs a missing `rollback-state` CLI |
| disputes | Substantial | complete | ✅ | 82-func full lifecycle |
| evidence_mgmt | Substantial | complete | ✅ | 40 tests; wired; CLI via root.go |
| ontology | Substantial | complete | ✅ | 84 tests (unweighted 1-addr-1-vote) |
| schedule | — | complete | ✅ | 45 tests (unbounded EndBlock work; MaxGasPerBlock unenforced) |
| ibcratelimit | Moderate | complete | ✅ | ~60 tests; rate-limits all denoms in aggregate |
| billing | Substantial | complete | ✅ | 43 tests; subscription path not msg-exposed |
| vesting_rewards | Substantial | complete | ⚠️ | 68 tests; empty default treasury/founder addrs; autopoiesis multiplier wiring commented out |
| **knowledge** | **Complete** | **FAILING** | 🛑 | **6 red keeper tests + Msg service unregisterable (codec gap)** — the flagship |
| oracle | (oracle) | partial/stub | 🛑 | No RegisterServices/InitGenesis; unroutable msgs; holds unused Minter/Burner; ID-collision overwrites stake |
| alignment | Substantial | substantial | 🛑 | No `.pb.go`; Msg+Query fail to register; no CLI/REST |
| gov | Complete | substantial | 🛑 | 17/25 MsgServer methods Unimplemented but reachable; escrow never refunded; disbursement transfers no coins |
| channels | Substantial | substantial | 🛑 | Custom sig verify excludes standard secp256k1 wallets; unbounded BeginBlock settlement |
| tree | Substantial | substantial | 🛑 | DeployService strands funds; no Deploying→Active; ~50 panic-on-store-error sites can crash validators |
| research | Substantial | substantial | 🛑 | ChallengeResearch dead-ends (both stakes locked forever); promised slash/reward unimplemented |
| qualification | Substantial | substantial | 🛑 | Endorsement stake never refunded (funds-loss); endorser counting not deduped (sybil); Endorsement query broken |
| capture_challenge | Substantial | substantial | 🛑 | Module-account insolvency on upheld payout; authority-only resolution; no invariant |
| icaauth | Moderate | partial | 🛑 | ICA accounts never activated; controller middleware built with nil — non-functional e2e |
| bvm | Substantial | substantial | ⚠️ | ~445 tests; LGM-20 opcodes depend on `x/tokens`; interpreter determinism unfuzzed |
| home | Substantial | substantial | ⚠️ | 142 tests; MsgUpdateParams unroutable; MsgSetVoice unreachable; treasury auto-compound is a no-op |
| partnerships | Substantial | substantial | ⚠️ | 91 tests; 5/16 msgs Unimplemented but reachable; seed/matching/consortium dead on-chain |
| autopoiesis | Substantial | substantial | ⚠️ | 254 tests; no Query service; CLI not wired; auto-gov loop only emits events |
| capture_defense | Substantial | substantial | ⚠️ | 26 tests; CLI nil; partial Params.Validate can disable detection |
| discovery | Moderate | substantial | ⚠️ | Keeper fully tested incl. adversarial; CLI nil |
| claiming_pot | Substantial | substantial | ⚠️ | MsgUpdateParams unroutable; eligibility/config only via genesis |
| compute_pool | Substantial | partial | ⚠️ | Whole task lifecycle implemented+tested but **unroutable**; credit-issuance has no production caller |
| tokens | **Stub** | **substantial** | ⚠️ | **Better than claimed** — 134 tests pass; but Query surface exposes only Params (balances/pools invisible to bank tooling) |

> Note the README diverges from reality **in both directions**: `knowledge` is worse than "Complete," `tokens` is far better than "Stub." That divergence is exactly why running the tests beats trusting the labels.

---

## What's genuinely solid (credit where due)

- **Build is clean**: `go build ./...` exits 0 across all 129 packages → a real ~140 MB `legbled` arm64 binary; `init`/`start`/`validate-genesis`/`gentx`/`collect-gentxs` all run.
- **`go vet` is effectively clean**: 28 warnings are **one** copylocks defect class, 100% confined to `x/oracle` (proto types passed by value). The other 128 packages vet clean. Low runtime risk (json.Marshal skips the unexported mutex), but it fails `make pr-check` (exit 1) until fixed.
- **`app.go` wiring is coherent**: 30 modules, no placeholder/nil keepers, deliberate & documented Begin/End/InitGenesis ordering, BeginBlocker order machine-checked at startup, ABCI++ Prepare/ProcessProposal have panic-recovery + emergency-halt guard.
- **The kill switch is real code** (~3149-LOC keeper, genuine 3-phase BFT ceremony, ante gate, proposal halt filter) — it works *once a council/Guardians are seated*.
- **Anti-farming (FARM-1..15) is mostly real and wired** with live call-sites; the gaps are honestly documented, not hidden.
- **Adversarial test coverage exists** (OpenClaw): auth, staking (242), disputes (82), bvm (445), home (142), gov (99), partnerships (91). Only `x/knowledge` currently fails.

---

## Top systemic risks if launched today

1. Flagship knowledge Msg service silently unregistered — node looks healthy, can't accept core txs (`FailedServices` never read).
2. PoT reward + anti-farming math fails its own tests on the live tree — the highest-value exploit surface ships with unverified economics.
3. Kill switch ships unarmed (empty council + all-apprentice validators) — no working emergency stop.
4. Near-free sybil + gas-free 14-day bootstrap + no per-account rate limits = cheap verification-spam DoS.
5. Crash-on-corruption: `x/tree` ~50 unrecovered store panics can deterministically halt all validators on adversarial state.
6. Broken economic accounting: `IncrementBurned` called at 0 of 24 burn sites (TotalBurned stuck at 0); dual minted counters can diverge; empty treasury addrs can silently lose rewards — dashboards report false data right when agents start farming.
7. Footgun tx surface: many modules register Msg types that fall through to `Unimplemented` (gov 17/25; partnerships 5; compute_pool task lifecycle; oracle both) — users build, sign, pay gas, always fail. Plus stranded-funds paths (research, gov escrow, qualification, capture_challenge).

---

## Suggested sequencing

- **This week (unblock the boot):** items 1–3 + the `x/oracle` vet fix. After this, the chain can actually accept and process PoT claims, and `make pr-check` goes green. This is the difference between "boots" and "works."
- **Before opening the gate (arm & prove):** items 4 + 6 (genesis council + real multi-validator bring-up). After this you have a network that can defend itself and reach VRF quorum.
- **Before inviting adversaries (harden):** item 8 + the stranded-funds / footgun-msg cleanup (decide per module: implement or remove from the proto/codec surface).

The foundation is real. What stands between you and a public testnet is **wiring and arming, not building.**
