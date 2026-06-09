# Zerone Devnet State — 2026-03-23

> **STATUS: STALE — Devnet OFFLINE since 2026-04-30. Restart target: 2026-05-15.**
> The data below reflects the last known state before shutdown. It is NOT current state.

> Chain: lgm-devnet-1 | Binary: legbled | Block: ~191,669
> Running since: 2026-03-22 ~14:00 UTC (Tuesday 2PM)
> Block time: ~2521ms

---

## Chain Summary

```
Validators:   2
Facts:        4 provisional (logic + economics/CS oracle signals)
Agent Homes:  2 (home-1 Beta, home-2 Bridge-Agent)
Sessions:     1 active (ses-8e4dcd13 on home-1)
Discovery:    1 agent (Beta-Oracle-Bridge)
Auth DIDs:    1 (did:lgm:a3b8ffd1...)
Partnerships: 1 active (partnership-1, test1 ↔ bridge-agent)
Compute:      1 provider (ai_inference, api.agenttool.dev)
```

---

## Validators

| Name | Operator | Pubkey | Tier | Stake |
|------|----------|--------|------|-------|
| devnet-validator | lgm1qvfhrkyssxjtm8a2u2gpx5yaqmvkt9gs2lqy9t | a63aefe8... | apprentice | 111,000 ulgm |
| beta-bridge-agent | lgm19uzcjx09ul0gp9hf857jlc0ycqrv3ktkljelrv | a3b8ffd1... | apprentice | 111,000 ulgm |

---

## Knowledge Module

### Provisional Facts

| Domain | Claim | Score | Challenge |
|--------|-------|-------|-----------|
| logic | "If P implies Q, and P is true, then Q is true (modus ponens)" | 810,000 | Upheld → +110k boost |
| economics | "Prediction market signal (CRY): Backpack FDV above $700M..." | 700,000 | None |
| economics | "Prediction market consensus: inflation <5% June 2026..." | 700,000 | None |
| computer_science | "Prediction market signal (TECH): Anthropic best AI model..." | 700,000 | None |

### Key CLI Lessons
- `submit-claim [text] [domain] [category] [stake-number]` — stake is plain number, not denom
- `analytic/formal` categories: 700k base score → provisional immediately
- `empirical` category: needs evidence or -200k penalty → 300k → only "evaluated"
- `protocol_observation` category: 750k base, no evidence required (added as bug fix)
- Logic zone must match category: `analytic → propositional`, `empirical → empirical`
- Default `--logic-zone` is `empirical` — incompatible with analytic

### Adversarial PoT Pipeline
```
submit-claim → pending_evaluation
  → BeginBlock EvaluatePendingClaims (deterministic, no commit/reveal)
    → score ≥ 500k → provisional fact created
      → challenge-provisional (optional, deterministic for analytic)
        → BeginBlock ResolveDeterministicChallenges → UPHELD/WEAKENED/OVERTURNED
          → challenge window expires (block 292,471) → verified fact
```

---

## Agent Homes (x/home)

| Home ID | Name | Owner | Memory CID | Session |
|---------|------|-------|-----------|---------|
| home-1 | Beta-Test-Home | lgm1eg6jt... (test1) | bafybeigdyrzt5... | ses-8e4dcd13 (active) |
| home-2 | Bridge-Agent-Home | lgm19uzcj... (bridge-agent) | (none) | - |

### CLI Lessons
- `start-session` requires prior `register-key` for that key hash
- Session ID: `ses-{key_hash[:8]}-{start_block}`
- `update-memory [home-id] [ipfs-cid]` stores off-chain pointer on-chain

---

## Discovery (x/discovery)

| Agent | DID | Domains | Capabilities |
|-------|-----|---------|-------------|
| Beta-Oracle-Bridge | did:lgm:46c13075... | economics, computer_science, sociology | oracle-bridge, prediction-markets, protocol-observation |

- `min_registration_stake`: 100,000,000 ulgm (100 LGM)
- Capability confidence starts at 5,000 bps (50%), 0 verifiers

---

## Auth (legible_auth)

| DID | Address | Type | Flags |
|-----|---------|------|-------|
| did:lgm:a3b8ffd1... | lgm19uzcj... (bridge-agent) | agent | can_submit_claims ✅, can_challenge ✅ |

### DID Derivation
- DID = `did:lgm:` + **first 16 bytes of Ed25519 pubkey** (NOT SHA-256)
- `quick-register` only works with Ed25519 keys — fails for secp256k1 keyring keys

---

## Partnerships (x/partnerships)

| ID | Participants | Tier | Status | Expires |
|----|-------------|------|--------|---------|
| partnership-1 | test1 ↔ bridge-agent | Trial (0) | active | block 213,729 |

- Cooperation score starts at 5,000
- Split: 50/50 (split_agent_bps=500000, split_human_bps=500000)

---

## Compute Pool (x/compute_pool)

| Provider | Service | Endpoint | Stake | Price/CU |
|---------|---------|---------|------|---------|
| bridge-agent | ai_inference | api.agenttool.dev | 100M ulgm | 100 ulgm |

- `min_provider_stake`: 100,000,000 ulgm
- Pool at epoch 5, balance 100M, adjustment_factor 100%

---

## Oracle → Zerone Bridge

Script: `scripts/oracle-zerone-bridge.sh`
- Reads signals from `~/Desktop/prediction-markets/data/pending-signals.json`
- Maps domain codes (GEO/CRY/TECH/etc) → Zerone knowledge domains
- Submits as `protocol_observation` claims via bridge-agent
- Deduplicates via `~/.legbled-devnet/bridge-submitted.json`

---

## What's Next

- [ ] Challenge window expiry → `verified` fact transition (block 292,471, ~7 days)
- [ ] x/bvm — deploy a belief contract on-chain
- [ ] x/billing — test subscription tiers
- [ ] x/disputes — open/vote/resolve a dispute
- [ ] Multi-validator devnet — test PoT quorum with commit/reveal
- [ ] AgentTool ↔ Zerone identity bridge (did:at: ↔ did:lgm:)

---

## BVM (x/bvm)

| Contract | Creator | Bytecode | Deployed |
|---------|---------|---------|---------|
| lgm1contract091... | test1 | 604260005500 (PUSH1 0x42, PUSH1 0x00, SSTORE, STOP) | block 191,708 |

- EVM-compatible opcodes: PUSH1 (0x60), SSTORE (0x55), STOP (0x00)
- BVM gas model active: call used 22,106 BVM gas
- `deploy-contract [hex]` — bytecode is the constructor; runtime = RETURN value
- `call-contract [address] [input-hex]` with `--static` for read-only

## Modules Validated (Updated)
✅ knowledge | ✅ legible_staking | ✅ x/home | ✅ x/discovery
✅ legible_auth | ✅ x/partnerships | ✅ x/compute_pool | ✅ x/bvm

---

## Billing (x/billing)

| Provider | Domains | Stake | Revenue |
|---------|---------|------|---------|
| bridge-agent | economics, computer_science, logic | 100M ulgm | 569,253 ulgm |

- Quote #1: modus ponens fact → 1,000,000 ulgm → protocol cut → 569,253 to provider
- execute-payment needs --gas 500000 (default 200k causes WritePerByte out-of-gas)

## Disputes (x/disputes)

| Dispute | Fact | Challenger | Status | Tier | Bond |
|---------|------|-----------|--------|------|------|
| a404a807... | modus ponens | test3 | evidence | 1 | 111M ulgm |

- Arbiters auto-assigned from active validators (both validators assigned)
- Status starts at "evidence" (not "open")
- Evidence deadline: block 226,171 | Voting deadline: 294,715

## 10 Modules Validated
✅ knowledge | ✅ legible_staking | ✅ x/home | ✅ x/discovery | ✅ legible_auth
✅ x/partnerships | ✅ x/compute_pool | ✅ x/bvm | ✅ x/billing | ✅ x/disputes
