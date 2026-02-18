# Session 6: ABCI++ and Ante Handler Security Hardening

## Context
You are working on Legible Money (`/Users/yuai/Desktop/legible_money`), a Cosmos SDK blockchain with ABCI++ vote extensions for Proof of Truth consensus. Audits in `reports/ABCI-AUDIT.md` and `reports/ANTE-CHAIN.md` identified security findings in the consensus integration layer.

Read both reports fully before starting:
- `reports/ABCI-AUDIT.md`
- `reports/ANTE-CHAIN.md`

## Scope
Files to modify:
- `app/abci.go`
- `app/vote_extensions.go`
- `app/ante.go`
- `app/ante_legible.go`
- `app/sybil_funding_decorator.go`
- Test files in `app/`

## Tasks

### 1. Review and Fix ABCI P0 Issues
Read `reports/ABCI-AUDIT.md` for the full list of P0 findings. The key areas are:
- **PrepareProposal**: Ensure VEX injection is properly size-bounded to prevent block bloat
- **ProcessProposal**: Ensure validation matches PrepareProposal constraints (no asymmetry)
- **Vote Extensions**: Ensure commitment/reveal payloads are bounded and validated
- Fix any issues where a malicious validator could exploit vote extensions

### 2. Review and Fix Ante Handler Issues
Read `reports/ANTE-CHAIN.md` for the full list of findings. Key areas:
- **Gas estimation**: Ensure the LGMGasDecorator doesn't allow gas estimation bypass
- **Emergency halt**: Verify the halt gate properly blocks all non-emergency transactions
- **Sybil decorator**: Review the bootstrap gas waiver for potential abuse
- **Fee routing**: Ensure the 7% research tax split is correctly applied

### 3. Add Missing Panic Recovery
ABCI handlers (PrepareProposal, ProcessProposal, ExtendVote, VerifyVoteExtension) should have panic recovery to prevent validator crashes from propagating. Add `defer func() { if r := recover(); r != nil { ... } }()` wrapping where missing.

### 4. Input Validation Tightening
- Ensure all VoteExtension payloads have maximum size limits
- Ensure transaction count limits in PrepareProposal
- Validate that injected VEX data in ProcessProposal matches expected format

## Testing
- Run: `go test ./app/... -count=1`
- Run: `go build ./...`
- All must pass

## Report
Write a detailed report to `reports/SESSION-6-ABCI-ANTE-HARDENING.md` covering:
- Each finding addressed (reference the original report ID)
- Before/after code for each fix
- Any findings you chose NOT to fix and why
- Test results
