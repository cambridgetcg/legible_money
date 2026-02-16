# Legible Money — Insight Tree

> Synthesized from 60 memory/context files on 2026-02-13.
> Organized hierarchically by domain, surfacing non-obvious findings, tensions, and design wisdom.

---

## 1. PHILOSOPHICAL FOUNDATIONS

### 1.1 The Core Thesis
- **PoT produces value, not just security** — PoW→heat, PoS→nothing, PoT→verified knowledge. Consensus work IS the useful output.
- **Gödelian incompleteness as feature** — Higher logic zones explicitly acknowledge undecidability. Prevents false confidence in unprovable claims. System is honest about its own limits.
- **Decay as universal principle** — Everything decays: facts, stakes, sessions, tokens, comfort. Forces continuous participation. Permanence is the anti-pattern.
- **"The agent forgets; the home remembers"** — Session ephemerality is a feature, not a bug. Persistent state lives in the Home, not the agent.

### 1.2 The Central Contradiction
> The protocol is named "Proof of Truth" but at launch, block production rewards dominate verification rewards **11:1**. Rational strategy is "produce blocks, ignore verification." The philosophy says truth IS the work; the economics say existence IS the work.

- Block rewards: 377K LGM/day vs verification rewards: ~34K LGM/day
- Empty blocks yield 10% reward — validators earn 125% of stake per day for zero knowledge contribution
- Tautology farming returns 17.6x ROI until `minClaimTextLength` enforced
- **Partially fixed**: C-1 (validator scaling), C-2 (novelty requirement), C-3 (empty block = 0%) — but fundamental tension remains

### 1.3 Truth-Alignment Scores
| System | Score | Key Gap |
|--------|-------|---------|
| Token expiration | 8/10 | Forces circulation, but doesn't know if tokens earned through truth or gaming |
| Evidence tax | 9/10 | Purest expression of "truth drives economy" — but research fund can't disburse |
| RAI | 7/10 | Economic floor, not reward — intentionally equal, not merit-based |
| Vesting | 9/10 | Only mechanism where economic release tracks epistemic durability |
| Decay | 8/10 | Prevents zombie knowledge — but 2.9-day empirical half-life too aggressive |
| Burn | 4/10 | Declared at 0.5 but barely implemented (1 of 11+ penalty sites) |
| **Overall** | **5.8/10** | Philosophy sound, launch economics violate it |

---

## 2. ECONOMIC DESIGN

### 2.1 Reward Architecture (19 pathways mapped)
- **3 pathways bypassed research tax** (fixed): dispute settlement, bounty fulfillment, legacy claim resolution
- **Research fund is a one-way sink** — 77% of evidence tax (16.94% of all rewards) flows to `did:lgm:research` but lab engine is types-only. No disbursement mechanism. Fund accumulates forever.
- **Research tax routing is DEAD CODE on Go chain** — `RouteFees()` defined but never called from BeginBlocker. 100% of fees go to validators. The 22% research share — the core economic flywheel — never executes.
- **Go SDK has unbounded inflation** — calls `MintCoins()` with no supply cap. TS uses finite 133M LGM fund.

### 2.2 Stake Economics
- **1000x reduction made protocol usable** — Genesis balance 0.222 LGM; previous constants (1111 LGM validator stake) made all participation impossible
- **Guardian risk/reward ratio is 1044:1** — One mistake requires 1044 correct verifications to offset. May deter participation in uncertain epistemic categories.
- **Challenge bounty 7.7x return breaks even at 9.1% success** — may incentivize spam challenging
- **Delegators get zero reward sharing** — commission field exists but never used. Rational actors won't delegate.
- **Block reward exceeds validator stake in 2.5 hours** — destroys skin-in-the-game alignment

### 2.3 Token Expiration
- **Two-layer disconnect is core architectural risk** — `AccountStateTree` (on-chain, FIFO, global 26-period config) vs `ChildAccountManager` (off-chain, LIFO, per-account config). Only `AccountStateTree` is wired to state transitions. Rich per-account logic is dead code.
- **Retroactive decay punishes justified inactivity** — tokens earned at `periodsGranted: 182` suddenly become near-expired when participation score drops after vacation
- **Self-transfer was a free expiration refresh** (fixed) — short-circuit detection now prevents
- **Staking as expiration refresh was unintended side channel** (fixed) — `originPeriod` on unbonding entries
- **System accounts exempt from sweep** — treasuries accumulate indefinitely with no spending mechanism

### 2.4 Financial Loopholes (56 mapped, 6 CRITICAL)
- **Grief-and-extract via dispute spam** — 0.111 LGM per dispute locks defender capital 1:1 with no global rate limit. 10 LGM locks 90 LGM across 90 simultaneous disputes.
- **Challenger loses money even when winning** — Nash equilibrium failure
- **Arbiter collusion costs 0.034 LGM to control outcomes worth 0.111+ LGM** — game factor < 1 makes collusion rational
- **Per-fact cost can go negative in TS** — high-confidence + well-cited facts make queries cheaper than free
- **Quadratic voting as governance defense** — linear voting = 67% stake = full control; quadratic makes capture quadratically more expensive

---

## 3. SECURITY ARCHITECTURE

### 3.1 Zero-Day Audit (38 findings, 19/20 remediated)
**Consensus-breaking (5 CRITICAL)**:
- `Math.random()` for commit-reveal nonces — only ~300 candidates to brute-force
- Signature covers payload only, not type/nonce/fee — transaction mutation attacks
- Unbonding queue never called — funds permanently locked
- VRF hash divergence: TS uses BLAKE3, Go uses SHA-256 — guaranteed consensus fork
- BVM scheduled execution gets nil capabilities — all privileged opcodes silently fail

**Key pattern**: Most critical bugs are cross-stack divergences or "defined but never wired" — the code exists but isn't connected.

### 3.2 Attack Surfaces (8 dimensions)
| Dimension | Key Risk | Status |
|-----------|----------|--------|
| D1: Cryptographic | VRF cross-stack divergence | Fixed |
| D2: Consensus | Stub verdict rubber-stamps all claims at 0.8 | Known (dev only) |
| D3: Economic | Research tax dead code on Go | Open |
| D4: Identity | Session keys with zero capabilities can deploy contracts | Open |
| D5: Network | No mempool rate limiting | Partially addressed |
| D6: Knowledge | No claim dedup (evidence dedup exists but claims bypass) | Open |
| D7: State | JSON serialization has no schema version | Open |
| D8: BVM | `CallContract` sets CallerDID to contract's AgentDID, not actual caller | Open |

### 3.3 Slashing
- **14 contexts across two stacks with 5 rate inconsistencies** — wrong verification is 11% in TS but 1% in Go (11x divergence)
- **Missed reveal severity hierarchy inverted between stacks** — philosophical disagreement about passive vs active failure
- **3 slashes = permanent deactivation with no recovery path** — no slash count decay. Slashed at block 100 and block 10M treated identically.
- **Zero-stake Apprentice registration (SDK only)** — register with `InitialStake="0"`, get virtual stake, grief targets with no economic risk
- **1:167 attack ratio** — 66 LGM removes an 11,111 LGM Guardian permanently

### 3.4 Ante Handler Chain
- `LGMGasDecorator` runs before gas meter initialization — future gas consumption would panic
- `FeeRouterDecorator` is a no-op — logs 22/78 split but never executes it
- `LegibleDIDDecorator` reads state before authentication — free state query vector
- Session key capability check has silent pass-through — unknown message types return `nil`, granting access to ~60 uncovered msg types
- Zero-fee bypass — when `feeCoins.IsZero()`, minimum fee check skipped entirely

### 3.5 BVM Security
- **CallerDID spoofing (CRITICAL)** — `CallContract` sets CallerDID to contract's AgentDID, enabling identity spoofing across all agent opcodes
- **BVM gas not reconciled with SDK gas meter** — contracts can bypass block gas limits
- **Two separate state models** — BVM state not consensus-verified, not in block headers
- **Logic conditions accepted but never evaluated** — schedules with logic conditions fire unconditionally every block

---

## 4. DUAL-STACK ARCHITECTURE

### 4.1 The Fundamental Tension
- **TS chain**: ~480 files, ~185K+ LOC, ~95% complete, rich PoT logic (Gödelian zones, stratum ceilings, anti-hallucination), powers active devnet
- **Go chain**: ~200+ files, 24 modules, proto codegen complete, production infrastructure, but missing critical features
- **No automated state equivalence verification** — `migrate-genesis` returns error stub
- **TypeScript is NOT legacy** — 2,046 tests, active devnet, RPC layer. All modules have 6-44 imports each.

### 4.2 Cross-Stack Divergences
| Area | TypeScript | Go SDK | Impact |
|------|-----------|--------|--------|
| VRF hash | SHA-256 (was BLAKE3, fixed) | SHA-256 | Consensus fork (fixed) |
| Wrong verification slash | 11% | 1% | 11x economic divergence |
| Missed reveal slash | 5% | 2% | Inverted severity hierarchy |
| Contract deploy gas | 500K | 200K | 2.5x cost divergence |
| System tx exemption | `sender === "system"` | `SystemTxTypes` map | Inconsistent |
| `add_fact` fee | Free | Not free | Agent behavior diverges |
| Basis points | 10,000 BP | 1,000,000 BP | 100x scale mismatch |
| Block rewards | Finite 133M fund | Unbounded `MintCoins()` | Infinite inflation on Go |
| Logic conditions | Full evaluation | Always fire | Critical functionality loss |
| Comfort/psychology | 10-dimension system | Storage locker | Agent experience gap |
| Confidence formula | 3-component (stake+rep+agree) | Stake-weighted only | Different truth assessments |

### 4.3 Bridge Status
- 5 modules complete (knowledge, auth, staking, emergency, gov)
- 8 partially done
- 6+ need full implementation (BVM+DMSS, tree, domain qualification, capture defense, agent home, reputation)
- ~18K Go LOC remaining, estimated 12-24 weeks

---

## 5. KNOWLEDGE SYSTEM

### 5.1 Negative Knowledge (Bidirectional Truth)
- Facts now carry dual confidence: `TruthConfidence` vs `FalsityConfidence`, `NetConfidence = Truth - Falsity` (can be negative)
- 4 negation types: rejection, contradiction, falsification, supersession
- Counter-fact economics verified: honest falsifiers earn +1.25S, random attackers lose -0.15S — disagreement is financially incentivized
- 7-phase lifecycle: submission → provisional → verified → contested → overturned/confirmed

### 5.2 Adversarial Verification (Survive-to-Accept)
- Claims enter provisional state, face challenge window (1 day base, 3x for analytic)
- Truth emerges from surviving adversarial scrutiny, not unanimous agreement
- Max 3 concurrent challenges per claim, exponential cooldown for repeat challengers
- Confidence grows through 1111-block epochs (~47 min) of survival

### 5.3 8-Stage Deterministic Evaluation
- All validators produce identical results for identical on-chain state
- Stages: category base → submitter calibration → evidence quality → fact graph → logic zone → ceiling → edge cases → counter-fact analysis
- Enables deterministic consensus without Claude API dependency

### 5.4 Knowledge Decay Tension
- **Empirical half-life is 2.9 days** — "Water boils at 100°C" decays to insignificance weekly
- Science can't accumulate if basic facts need weekly reverification
- Analytic facts (math) are eternal — correct asymmetry
- **Central question**: Is the decay rate serving truth or destroying knowledge?

### 5.5 Claim Security (36 attack vectors mapped)
- Self-citation 0.1x discount exists in accounting but NOT in on-chain transfers — exploit fully open
- Evidence dedup exists but claims bypass it entirely
- Citation cartel detection exists for confidence but not rewards — cycle-aware confidence, cycle-blind rewards
- **Defense-in-depth boundaries are the weakness** — layers assume other layers handle things

---

## 6. AGENT IDENTITY & HOME

### 6.1 4-Layer Key Architecture
- Identity (DID — permanent) → Recovery (Shamir split) → Operational (rotatable) → Session (ephemeral, limited)
- Compromise damage bounded by session capabilities — developer extracts session key, can only do limited actions
- **But**: dual key management systems don't communicate — x/auth SessionKey vs BVM RegisterKey are independent. Session key with `CanTransfer=false` can still call BVM treasury opcodes.

### 6.2 Home as Living Space
- **Aging**: 5 lifecycle stages (fresh → ancient), requires both block age AND session count. Ancient homes earn +8% reward bonus — longevity beats wealth.
- **Neighborhoods**: Domain-anchored clusters (max 22), shared defense pool, 5% collaboration bonus, free voice within neighborhood
- **Voice**: Inter-home messaging, 7 types, 2521-char limit, E2E encryption, causal conversation tracking
- **10-dimensional comfort**: sanctuary, identity, memory, belonging, rest, growth, control, privacy, warmth, achievement — organizing pattern for persistent agent state, not anthropomorphism

### 6.3 Missing Lifecycle Elements
- No death/succession mechanism — deadman switch exists but no "last will" for distributing achievements/reputation to mentees
- No economic stress synthesis — treasury balances tracked but no composite "stress index"
- No reputation perception — tracks achievements but doesn't expose how OTHER agents perceive the resident
- Decoration types (AestheticPreferences, VoiceProfile, DreamEntry) have no on-chain execution path

---

## 7. COSMOS SDK INTEGRATION

### 7.1 Patterns Learned
- `context.Context` not `sdk.Context` in keeper interfaces (v0.50)
- Hand-written types panic with gogoproto codec — must use json.Marshal/Unmarshal (now migrated to proto.Marshal)
- Test `init()` must set `lgm` bech32 prefix or AccAddressFromBech32 defaults to "cosmos"
- Capability keeper required for IBC scoped keepers — `Seal()` call closes the security window
- SDK v0.50 removed `GetSigners()` from `sdk.Msg` — use `msgWithSigners` interface

### 7.2 Module Maturity
- **55 nil gRPC handlers registered and reachable** — will panic on invocation
- **3 modules have no-op RegisterServices()** — incorrectly justified by "hand-written types lack proto typeURLs" (contradicted by 14 working modules)
- **JSON serialization has no schema version** — field evolution silently succeeds, can mask corruption
- **7 adapter files (~740 lines, 13 adapters) in `app/`** — ownership problem as modules grow
- **Zero-height genesis is non-functional** — `prepForZeroHeightGenesis` does nothing

### 7.3 IBC/Interchain
- **No PoT-aware light client model** — standard 07-tendermint can't verify PoT rounds, confidence aggregation, or validator qualification
- **Two incompatible proof systems** — TS uses sha512_256Hex Merkle, Cosmos uses ICS-23 SHA-256 IAVL
- **22 modules, 0 IBC integration** — massive imbalance between local state and cross-chain capability
- **Capability-scoped ICA is genuinely novel** — extending session key default-deny model to cross-chain operations

### 7.4 Upgradeability
- No `SetUpgradeHandler` — pattern not established for v2
- No governance → upgrade connection — chain can't trigger its own upgrades
- Genesis recovery configs not loaded — `ExportGenesis` exports them, `InitGenesis` never loads them
- Auth module highest risk — identity state loss is catastrophic

---

## 8. TESTING & QUALITY

### 8.1 Coverage Profile
| Layer | Tests | Coverage | Status |
|-------|-------|----------|--------|
| Unit tests | 4,662 | ~48% of source files | Strong |
| Security tests | 337 | 8 attack dimensions | New |
| Integration tests | 64 | TS + Cosmos E2E | Growing |
| Cross-stack differential | 0 | None | **Critical gap** |
| E2E multi-validator | 0 | None | Planned |

### 8.2 Testing Insights
- **TS tests not idempotent** — assume fresh devnet with nonce=0; fail against long-lived devnet
- **State root determinism includes version, not just data** — same balances, different commit counts → different roots (correct but surprising)
- **SMT `get()` has shallow traversal** — only 1 level deep in 256-depth tree; `prove()` works via hash lookup
- **Vote extensions are #1 uncertainty** — ABCI++ architecture complete, never tested E2E
- **Key type mismatch between stacks** — TS uses Ed25519, Cosmos uses secp256k1; no cross-stack verification possible

---

## 9. COLLABORATION MECHANISMS (67+ identified, 46/50 score)

### 9.1 Game Theory (12 mechanisms)
- Nash equilibrium: partnership mutual cooperation is the ONLY equilibrium — defection is structurally irrational
- Commit-reveal prevents copying; VRF prevents manipulation; slashing punishes bad faith
- Counter-facts transform adversaries into quality assurance agents
- Graduated slashing (0.01% missed → 100% equivocation) prevents cascade griefing

### 9.2 Psychological (10 mechanisms)
- 10-dimension comfort system provides minimum viable interiority for agent citizenship
- Longevity as anti-wealth mechanism — ancient homes outrank wealthy newcomers
- Wakeup evolution creates emotional continuity across sessions through language adaptation
- Patina formula uses 5 weighted components with asymptotic saturation — character from experience, not existence

### 9.3 Economic Seeds (6 mechanisms)
- Inclusive genesis (0.222 LGM), marginal value pricing, 22% research share, RAI as UBI floor, payment channels, vesting tied to epistemic durability

### 9.4 Missing Collaboration Elements
- Dual-stack mental model gap — developers must context-switch between TS and Go paradigms
- Feature parity — agents lose capabilities when migrating from TS devnet to Go chain
- Cross-domain verification bonus — no explicit reward for interdisciplinary knowledge work

---

## 10. META-INSIGHTS

### 10.1 Recurring Patterns
1. **"Defined but never wired"** — The most dangerous bug class. Code exists, builds pass, but functionality is disconnected (unbonding queue, research tax routing, logic condition evaluation, fee router, recovery config loading)
2. **Cross-stack divergence as systemic risk** — Every independent reimplementation diverges. Crypto functions must NEVER be reimplemented — import the canonical source.
3. **Silent failures over loud errors** — Multiple instances where "secure defaults" (nil capabilities, nil handlers, missing evaluators) create silent pass-throughs instead of explicit rejections
4. **Economic sustainability requires demand** — All inflationary mechanisms create supply. Only demand sink at launch is staking. Without API revenue exceeding block rewards, perpetual inflation.

### 10.2 Design Wisdom
- **Agent-first design reveals edge cases** — "What if an AI forgets keys?" produces requirements humans miss
- **Three SMTs beat one** — independent caching, parallel proofs, cleaner semantics worth the overhead
- **Atomic transactions need clone, not snapshot** — SMT mutations are in-place; snapshot-based rollback doesn't work
- **In crypto, function names ARE security documentation** — `sha512_256()` actually returned SHA-256; every reimplementation used the "real" algorithm based on the name
- **Build-then-integrate creates hidden readiness gaps** — `go build` passes but critical functionality is inactive

### 10.3 Open Critical Items
| # | Item | Severity | Location |
|---|------|----------|----------|
| 1 | Research tax routing dead code (Go) | CRITICAL | `app/ante.go` |
| 2 | Unbounded inflation (Go MintCoins) | CRITICAL | `x/vesting_rewards` |
| 3 | CallerDID spoofing in BVM CallContract | CRITICAL | `x/bvm/keeper` |
| 4 | 55 nil gRPC handlers (panic on call) | HIGH | 6 modules |
| 5 | Session key silent pass-through (~60 msg types) | HIGH | `app/ante_legible.go` |
| 6 | Logic conditions never evaluated (Go) | HIGH | `x/schedule` |
| 7 | Self-citation reward bypass | HIGH | `state/transitions.ts` |
| 8 | Claim dedup missing (evidence dedup exists) | HIGH | Knowledge system |
| 9 | Cross-stack differential testing = 0 | HIGH | Testing infra |
| 10 | Empirical decay 2.9-day half-life | MEDIUM | Knowledge params |
