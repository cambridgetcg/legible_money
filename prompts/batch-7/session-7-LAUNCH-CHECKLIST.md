# Session: LAUNCH-CHECKLIST — Final testnet launch preparation

You are working in the legible_money Cosmos SDK chain codebase at ~/Desktop/legible_money.

## Task: Final pre-launch checklist and documentation

This is a READ-ONLY verification + documentation session. Minimal code changes (docs only).

### Step 1: Verify the blocker is resolved
```bash
# Confirm maccPerms includes ALL required modules
grep -A 50 "maccPerms" app/app.go | grep -c "ModuleName\|\"treasury"
# Should be 33+ entries (27 from B6-7 + 6 new)
```

### Step 2: Build + test one final time
```bash
go build -o build/legbled ./cmd/legbled
go test ./... -count=1 -timeout 600s
go vet ./...
```

### Step 3: Version check
```bash
./build/legbled version --long
```
Verify name, version, commit are populated.

### Step 4: Genesis validation with production config
```bash
# Use the testnet genesis ceremony
bash scripts/testnet-genesis.sh init --chain-id lgm-testnet-1
bash scripts/testnet-genesis.sh add-validator val1 100000000000ulgm
bash scripts/testnet-genesis.sh finalize
bash scripts/testnet-genesis.sh verify
```

### Step 5: Create CHANGELOG.md
Document everything that was done across batches 0-7:

```markdown
# Changelog — Legible Money v0.1.0 (Phase D)

## Security Fixes
- BVM: Memory offset overflow protection (15 opcode sites)
- BVM: CALLCODE storage context corrected
- BVM: Gas bridge between SDK and VM
- BVM: Panic recovery for all execution paths
- BVM: Scheduled execution gas budget (5M/block)
- BVM: Capability propagation to sub-calls
- ABCI: False equivocation prevention (multi-block commitment)
- ABCI: Reveal hash verification against commitment
- ABCI: Panic recovery for all ABCI handlers
- ABCI: VEX injection size limits (2MB)
- Ante: SDK v0.50 proto type signer extraction
- Ante: ICA type URL correction
- Ante: Gas overflow protection
- Staking: Reputation uint64 underflow protection
- Staking: ContestedVerificationsCorrect tracking via hooks
- Staking: Guardian slash window from params (not hardcoded)
- Staking: Delegation reverse index for correct queries
- IBC: ICA transfers through rate limit middleware
- IBC: All-denom rate limiting (not just tracked denom)
- IBC: Sliding window epoch boundary protection
- IBC: AllowedMsgTypes enforcement per account
- IBC: SendTx actually sending
- IBC: MsgTransfer removed from default allowlist
- Channels: Ed25519 signature verification on all state mutations
- Channels: Proper closing state machine with dispute window
- Gov: uint64 overflow in vote tallying (math/big)
- Gov: QuorumThresholdBps enforcement
- Knowledge: Confidence calculation overflow protection

## Features
- 216 TX CLI commands + 163 Query CLI commands
- 214 Proto TX gRPC endpoints
- Full RegisterInterfaces across 28 modules
- 242 gas entries
- Token features: --mintable/--burnable/--pausable/--wrappable
- Complete event emission across all modules
- Production genesis ceremony with 410+ params
- Docker 3-validator testnet
- Proof of Truth verified end-to-end

## Infrastructure
- 2,535+ tests, 100% pass rate
- go vet clean (0 warnings)
- 28/28 modules fully wired in app.go
- All module accounts registered in maccPerms
```

### Step 6: Update docs/TESTNET-LAUNCH.md
Verify the launch documentation reflects the final state. Add any corrections from batch 6-7 findings:
- Account pubkey bootstrap requirement
- Challenge window duration for testing
- Gas and fee requirements for knowledge tx
- Which PoT path works (deterministic) vs needs more testing (arbiter)

### Step 7: Final git status
```bash
git status
git log --oneline -20
```
Document the final branch state.

## Report format
Write to reports/batch-7/SESSION-B7-7-LAUNCH-CHECKLIST.md:
- Build + test: PASS/FAIL
- Version output
- Genesis ceremony: PASS/FAIL
- CHANGELOG.md location
- docs/TESTNET-LAUNCH.md updates
- Final git status
- **VERDICT: READY FOR TESTNET / NOT READY**
