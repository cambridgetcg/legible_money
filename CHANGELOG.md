# Changelog — Legible Money v0.1.0 (Phase D)

## Security Fixes

### BVM (Belief VM)
- Memory offset overflow protection (15 opcode sites)
- CALLCODE storage context corrected
- Gas bridge between SDK and VM
- Panic recovery for all execution paths
- Scheduled execution gas budget (5M/block)
- Capability propagation to sub-calls
- Cross-call depth limit enforcement

### ABCI++
- False equivocation prevention (multi-block commitment)
- Reveal hash verification against commitment
- Panic recovery for all ABCI handlers
- VEX injection size limits (2MB)

### Ante Handler Chain (21 decorators)
- SDK v0.50 proto type signer extraction (tx-level, not per-message)
- ICA type URL correction
- Gas overflow protection
- Bootstrap gas-free period (first 480,000 blocks)
- Sybil funding detection decorator
- Emergency halt decorator

### Staking
- Reputation uint64 underflow protection
- ContestedVerificationsCorrect tracking via hooks
- Guardian slash window from params (not hardcoded)
- Delegation reverse index for correct queries

### IBC
- ICA transfers through rate limit middleware
- All-denom rate limiting (not just tracked denom)
- Sliding window epoch boundary protection
- AllowedMsgTypes enforcement per account
- SendTx actually sending
- MsgTransfer removed from default allowlist

### Channels
- Ed25519 signature verification on all state mutations
- Proper closing state machine with dispute window

### Governance
- uint64 overflow in vote tallying (math/big)
- QuorumThresholdBps enforcement
- BPS arithmetic for all vote ratios (4 locations)
- Sybil detection integration

### Knowledge
- Confidence calculation overflow protection (safeMulDiv with math/big)
- VRF-based verifier selection (ECVRF-Ed25519-SHA512-Elligator2)

## Features

- 235 gas entries across all transaction types
- Token features: --mintable/--burnable/--pausable/--wrappable CLI flags
- Full RegisterInterfaces across 28+ modules
- Complete event emission across all modules
- Production genesis ceremony with 300+ params
- Docker 3-validator testnet
- Proof of Truth verified end-to-end (Phase D agent integration)
- OpenClaw adversarial probe framework (46 probes across social modules)
- Knowledge ExtendedParams (72 governance-adjustable parameters)

## Infrastructure

- 2,818 test functions across 129 test files, 37 packages — 100% pass rate
- go vet clean (0 warnings)
- 28+ modules fully wired in app.go
- Makefile with ldflags for binary versioning
- Agent SDK for external AI evaluators (Go client)
- PicoClaw plugin system for lightweight agent integration

## Module Status

### Complete
- knowledge, auth, staking, emergency, gov, channels, ibcratelimit, icaauth

### Substantial
- tokens, vesting_rewards, bvm, home, disputes, evidence_mgmt, research
- ontology, tree, alignment, autopoiesis, billing, capture_challenge
- capture_defense, claiming_pot, compute_pool, discovery, partnerships
- qualification, schedule

## Known Issues (Pre-Launch Blocker)

- **6 module accounts missing from maccPerms**: ontology, research, schedule, discovery, bvm, treasury_protocol. Any runtime `SendCoins*` call for these modules will panic. Fix: add entries to `maccPerms` in `app/app.go`.

## Dependencies

- Cosmos SDK v0.50.15
- CometBFT v0.38.20
- IBC-Go v8.8.0
- Go 1.26.0
