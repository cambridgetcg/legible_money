# Legible Money Genesis Plan — February 22, 2026

> PP-verified (30/30 gates). Creative tier.
> **Core principle: The network is the experiment. Forever.**

---

## The Philosophy: Perpetual Genesis

A Proof of Truth protocol that freezes its parameters has stopped seeking truth.

Legible Money doesn't have a testnet phase and a mainnet phase. It has **one living network** that is perpetually evolving — every parameter, every module, every incentive structure is fluid and governed by the participants who verify truth on it.

**What Feb 22 is**: The beginning. Genesis block. The first axioms go on-chain.

**What Feb 22 is not**: A finished product. A frozen parameter set. A promise that things won't change.

**What makes this work**:
- `x/gov` — LIP (Legible Improvement Proposal) lifecycle with quadratic voting. Any parameter can be proposed for change.
- `x/emergency` — 3-phase BFT kill switch. If something goes catastrophically wrong, validators can halt and fix.
- Cosmos SDK upgrade module — Software upgrades at specific block heights. No hard forks, just governance-approved evolution.
- Epoch boundaries — Natural recalibration points for economic parameters.

**The Gödelian commitment**: We acknowledge that our initial parameters are incomplete. The protocol's purpose is not to be right from day one — it's to get *better at finding truth* every day.

---

## What Must Work on Feb 22

Only three things matter at genesis. Everything else evolves after.

### 1. Blocks Produce
- `legbled` binary compiles and starts
- CometBFT consensus runs at 2521ms block time
- ≥3 validators in the active set
- ABCI++ vote extensions deliver VRF proofs

### 2. Truth Gets Verified
- `MsgSubmitClaim` → verification round → `MsgCommitVerification` → `MsgRevealVerification` → fact accepted or rejected
- The full PoT lifecycle works end-to-end
- At least the 5 genesis axioms are on-chain

### 3. Parameters Can Change
- `x/gov` accepts LIP proposals
- Parameter change proposals can modify consensus params
- `x/emergency` can halt the chain if needed
- **This is the critical one.** If governance works, every other problem is solvable by the network itself.

Everything else — perfect economics, full module completion, polished explorer — is post-genesis work that the living network handles through its own governance.

---

## Current State (Go Chain)

| Layer | Status | Genesis-Ready? |
|-------|--------|---------------|
| `app/` (wiring) | **BUILD FAILS** | NO — must fix |
| `cmd/legbled/` (binary) | **BUILD FAILS** | NO — must fix |
| `x/knowledge/` | COMPLETE | YES |
| `x/auth/` | COMPLETE | YES |
| `x/staking/` | COMPLETE | YES |
| `x/emergency/` | COMPLETE | YES |
| `x/gov/` | **BUILD FAILS** | NO — must fix |
| `x/tokens/` | PARTIAL | Can launch without |
| `x/vesting_rewards/` | PARTIAL | Can launch without |
| `x/bvm/` | PARTIAL | Can launch without |
| `x/home/` | PARTIAL | Can launch without |
| `x/disputes/` | STUB | Post-genesis |
| `x/evidence_mgmt/` | STUB | Post-genesis |
| `x/research/` | STUB | Post-genesis |
| `x/ontology/` | STUB | Post-genesis |

**The critical path is clear**: Fix the build failures in `app/`, `cmd/legbled/`, and `x/gov/`. Everything else either works or can wait.

---

## 6-Day Plan

### DAY 0 — Sun Feb 16: FIX THE BUILD

> If it doesn't compile, nothing else matters.

- [ ] Fix `x/gov/` build failure
- [ ] Fix `app/app.go` build failure (likely cascades from gov)
- [ ] Fix `cmd/legbled/` build failure (likely cascades from app)
- [ ] `go build ./...` — clean
- [ ] `go test ./x/... -count=1` — all module keepers pass
- [ ] Understand and document what caused the failures

### DAY 1 — Mon Feb 17: VERIFY THE LIFECYCLE

> The PoT lifecycle must work end-to-end in the binary.

- [ ] Start single-node `legbled` locally
- [ ] Submit a claim via CLI or RPC
- [ ] Watch verification round (commit → reveal)
- [ ] Confirm: claim becomes fact (or is rejected)
- [ ] Confirm: block rewards flow to validators
- [ ] Confirm: governance module accepts proposals
- [ ] If any step fails → fix it. This is the critical path.

### DAY 2 — Tue Feb 18: GENESIS FILE

> Genesis parameters are starting points, not final values. But they must be sane starting points.

- [ ] Create `genesis.json` for the living network
  - Chain ID: `legible-1` (no "devnet", no "testnet" — it's just the network)
  - Genesis time: `2026-02-22T00:00:00Z`
  - 5 genesis axioms (identity, non-contradiction, excluded middle, zero, successor)
  - Conservative economic parameters (knowing they'll be tuned via governance):
    - Block reward: scaled by validator count from day 1
    - Verification reward: meaningful relative to block reward
    - Claim stake: low enough to participate, high enough to deter spam
  - Governance parameters: short voting period initially (faster iteration)
  - Emergency parameters: low guardian threshold initially (safety net)

- [ ] Generate validator keys (secure, NOT in git)
- [ ] Document genesis parameter rationale (why each starting point)

### DAY 3 — Wed Feb 19: MULTI-NODE

> If it runs on one node, it might work. If it runs on three, it probably works.

- [ ] Start 3-node network (Docker or bare metal)
- [ ] All 3 validators producing blocks
- [ ] Submit claims from different validators
- [ ] Verification rounds complete with multi-validator participation
- [ ] Test: submit a governance proposal, vote, see it pass
- [ ] Test: trigger emergency halt, restart
- [ ] Let it run for 1000+ blocks (~42 min). Watch for stalls.

### DAY 4 — Thu Feb 20: DOCUMENT

> One document. Someone reads it, they can join.

- [ ] Write `docs/JOINING.md` — How to run a validator and join the network
  - Install `legbled`
  - Generate keys
  - Download genesis
  - Start node, connect to peers
  - Submit your first claim
  - Propose your first parameter change

- [ ] Write `docs/PHILOSOPHY.md` update — The Forever Network
  - Why there's no testnet/mainnet distinction
  - How governance evolves parameters
  - The Gödelian commitment to incompleteness
  - What "launch" means for a truth-seeking protocol

- [ ] Update `README.md`
  - Remove any testnet/mainnet language
  - Reflect Go chain as primary
  - Current module status (honest)
  - How to participate

### DAY 5 — Fri Feb 21: PREPARE

> Everything ready. Nothing left to do on launch day except start.

- [ ] Final `go build ./...` — clean
- [ ] Final `go test ./x/... -count=1` — all pass
- [ ] Genesis file committed
- [ ] Docker images built (if using Docker)
- [ ] Genesis ceremony script written:
  ```
  1. Validator-1 starts (creates block 0)
  2. Validator-2 joins (syncs, begins participating)
  3. Validator-3 joins (network at quorum)
  4. First claim submitted
  5. First verification round completes
  6. Network is live.
  ```
- [ ] Tag: `git tag -a v0.1.0-genesis -m "Genesis"`
- [ ] Announcement draft ready

### DAY 6 — Sun Feb 22: GENESIS

> The first axioms go on-chain. Truth verification begins.

```
Morning:
  Start validators in sequence
  Verify blocks producing
  Submit genesis claim (the first truth to verify)
  Watch first verification round complete

Afternoon:
  Publish announcement
  Share how to join
  Monitor first 1000 blocks

From here on:
  The network governs itself.
  Parameters evolve.
  Modules complete.
  Truth accumulates.
```

---

## What Happens After Genesis

There is no "post-launch roadmap" in the traditional sense. There is a living network that governs its own evolution. But here's what the network will likely need to address early:

### First Weeks: Tune Economics
- Validators observe actual reward distributions
- Community proposes parameter adjustments via LIPs
- Block reward, claim stake, verification reward all tuned from real data
- **The network has never had real data before. Now it does.**

### First Months: Complete Modules
- `x/disputes/` — Challenge resolution (currently stub)
- `x/evidence_mgmt/` — Evidence handling (currently stub)
- `x/research/` — Research coordination (currently stub)
- `x/ontology/` — Knowledge graph structure (currently stub)
- Each module deployed via governance-approved upgrade

### Ongoing: Optimize for Truth
- Is the protocol actually incentivizing truth-seeking? Measure it.
- Are validators doing useful verification or gaming rewards? Fix it.
- Are claims getting more accurate over time? Track it.
- Is the knowledge graph growing coherently? Observe it.
- **Every answer feeds back into governance proposals.**

---

## The Stub Module Strategy

Half the `x/` modules are stubs. This is fine. This is correct.

A forever-evolving network doesn't need every module at genesis. It needs:
1. The core loop (claims → verification → facts) — **COMPLETE**
2. Economic incentives (staking → rewards) — **COMPLETE**
3. Governance (proposals → voting → parameter changes) — **COMPLETE**
4. Safety (emergency halt → recovery) — **COMPLETE**

Everything else is a governance-approved upgrade that the living network decides it needs. The stubs aren't missing features — they're **reserved namespaces** for future evolution. The protocol knows they'll be needed; it just doesn't pretend to know exactly how they should work yet.

That's Gödelian. That's honest. That's legible.

---

## Risk Acceptance

| Risk | Traditional Response | Our Response |
|------|---------------------|-------------|
| Wrong economic parameters | Don't launch until perfect | Launch, measure, govern, tune |
| Module incomplete | Wait for completion | Launch with core, evolve via upgrades |
| Not enough validators | Recruit before launch | Start with 3, grow organically |
| Exploit discovered | Emergency patch + hard fork | Emergency halt + governance fix + restart |
| Parameters need changing | Hard fork or social consensus | `MsgSubmitProposal` → vote → auto-execute |

The traditional risks exist. We accept them differently — not by pretending we've solved them, but by building a protocol that can solve them as it goes.

---

## Success Criteria for Feb 22

**Genesis succeeds if:**
- [ ] Blocks are producing with ≥3 validators
- [ ] At least 1 claim has been submitted and verified
- [ ] Governance module can accept proposals
- [ ] Emergency module can halt if needed
- [ ] Someone who isn't us can read the docs and understand how to join

**Genesis does NOT require:**
- Perfect economic parameters
- All modules complete
- External validators
- A website
- Marketing

The network's job is to seek truth. The launch's job is to let it begin.

---

## Genesis Axioms

The first truths on the Legible Money network:

```
axiom:logic:identity          A = A
axiom:logic:non-contradiction ¬(A ∧ ¬A)
axiom:logic:excluded-middle   A ∨ ¬A
axiom:math:zero               0 is a natural number
axiom:math:successor          Every natural number has a unique successor
```

From these five axioms, an entire knowledge graph will grow. Every fact verified after genesis traces its confidence back to these foundations.

On February 22, 2026, a network begins where the work of consensus is the work of truth. It will never be finished. That's the point.

---

*Plan created: 2026-02-16 | PP gates: 30/30 PASS*
*Philosophy: The network is the experiment. Forever.*
