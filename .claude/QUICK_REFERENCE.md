# Quick Reference Card

> One-page summary. Print this mentally at session start.

---

## What Is Legible Money?

Blockchain for AI agents where **verifying knowledge IS the useful work**.

---

## Key Numbers

| Constant | Value |
|----------|-------|
| Block time | 2521ms |
| Finality | 19 blocks |
| Min validator stake | 1000 LGM | |
| Min claim stake | 100 LGM | |
| Quorum | 66% | |
| Acceptance threshold | 80% | |

---

## Directory Map

```
core/crypto/     ← Hashing, signatures, VRF, Merkle
core/logic/      ← Gödelian zones
agents/identity/ ← 4-layer keys, sessions
network/         ← libp2p, gossip, sync
consensus/pot/   ← Proof of Truth engine
consensus/blocks/← Block production
state/           ← SMT-backed state
```

---

## The 4 Key Layers

| Layer | Type | If Compromised |
|-------|------|----------------|
| DID | Permanent | Can't be (derived) |
| Recovery | Cold, split | Full access |
| Operational | Rotatable | Damage, can recover |
| Session | Ephemeral | Limited damage |

---

## The 3 State Trees

| Tree | Contents | Key |
|------|----------|-----|
| Account | Balances, nonces | hash(DID) |
| Validator | Stakes, reputation | hash(pubkey) |
| Knowledge | Facts, confidence | hash(factId) |

---

## Transaction Types

`transfer` `stake` `unstake` `delegate` `undelegate`
`claim_submit` `verification_commit` `verification_reveal`
`challenge` `governance_vote` `governance_propose`

---

## Confidence Formula

```
C = Σ(stake × reputation × verdict) / Σ(stake × reputation)
```

---

## Code Patterns

```typescript
// Money: Always bigint
balance: bigint;

// Hashing: Always domain-separated
domainHash('LGM.purpose.v1', data);

// State: Always atomic
state.beginTransaction();
try {
  // mutations
  state.commitTransaction();
} catch {
  state.rollbackTransaction();
}
```

---

## Memory Files

| File | Purpose | When to Update |
|------|---------|----------------|
| CURRENT_STATE.md | What's built/next | Each session |
| DECISIONS.md | Technical choices | When deciding |
| INSIGHTS.md | Learnings | When learning |
| BLOCKED.md | Blockers | When stuck |
| PROMPTS.md | Useful prompts | As discovered |
| THINKING.md | Mental models | Rarely |
| MOTIVATIONS.md | Why decisions | When deciding |
| EVOLUTION.md | Project history | Each phase |

---

## Session Protocol

**Start**: Read CURRENT_STATE.md, BLOCKED.md
**During**: Update as you go
**End**: Write session log, update memory

---

## Forbidden

- `number` for money (use `bigint`)
- `Math.random()` (use VRF)
- Hash without domain prefix
- Non-deterministic state transitions
- Trusting validators blindly

---

## If Lost

1. Read `.claude/CLAUDE.md` (full context)
2. Read `docs/BLOCKCHAIN_ARCHITECTURE.md`
3. Read `consensus/pot/types.ts`
4. Read `state/types.ts`

---
