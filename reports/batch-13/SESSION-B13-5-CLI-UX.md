# B13-5 Session Report: CLI UX — Human-Friendly Commands

## Summary

Added human-friendly CLI commands for the cooperation flow: convenience
transaction shortcuts, a dashboard query, a knowledge explorer, and a
human-readable amount parser.

## Changes

### Human-Readable Amount Parser
- **`cmd/legbled/cmd/amounts.go`**: `ParseHumanAmount()`, `FormatUlgmToLGM()`,
  `ParseHumanCoin()` — converts between human-readable LGM and ulgm
  - `0.1LGM` → `100000`, `1LGM` → `1000000`, `100000ulgm` → `100000`
  - Uses `math/big` for precision, handles up to 6 decimal places
  - Case-insensitive, whitespace-tolerant

### Convenience Transaction Commands
- **`cmd/legbled/cmd/quick_tx.go`**: 4 convenience commands
  1. **`quick-register`**: Auto-derives DID from keyring key, fills public-key
     - `--type` flag (human/agent/contract), defaults to agent
     - DID = `did:lgm:` + hex(SHA-256(pubkey))[:32]
  2. **`quick-partner [addr]`**: Initiates partnership, defaults to tier 1
     - `--lock-tier` flag with descriptive help text
  3. **`quick-accept [id]`**: Accepts pending partnership
  4. **`claim [statement]`**: Submit knowledge claim with human-readable amounts
     - `--domain`, `--stake` (required), `--category`, `--logic-zone`, `--refs`
     - Parses `0.1LGM` → `100000` ulgm internally

### Dashboard Query
- **`cmd/legbled/cmd/dashboard.go`**: `legbled query dashboard [address]`
  - Queries account (legible-auth), balance (bank), partnerships, and knowledge
  - Shows human-readable output with account type, reputation, balance in LGM,
    partnership details (tier, pot, cooperation score, split), and knowledge
    profile (facts submitted, citations, domains)

### Knowledge Explorer
- **`cmd/legbled/cmd/explore.go`**: `legbled query explore [fact-id]`
  - Shows fact details: statement, domain, confidence, submitter, citations
  - Resolves references (what this fact depends on) with live queries
  - Shows citing facts (what depends on this fact)

### Wiring
- **`cmd/legbled/cmd/root.go`**: Added 6 new commands
  - txCmd: `quick-register`, `quick-partner`, `quick-accept`, `claim`
  - queryCmd: `dashboard`, `explore`

## Tests (38 cases, all passing)

### cmd/legbled/cmd (3 test functions)
- `TestParseHumanAmount` — 25 cases: LGM conversion, ulgm passthrough, bare
  numbers, case insensitivity, whitespace, error conditions
- `TestFormatUlgmToLGM` — 7 cases: formatting and invalid input
- `TestParseHumanCoin` — 6 cases: coin string generation

## Verification
```
go build ./...           # clean
go vet ./cmd/legbled/... # clean
go test ./cmd/legbled/cmd/... -count=1 -v  # 38/38 pass
```

## Files Changed (6 new)
- `cmd/legbled/cmd/amounts.go` (amount parser)
- `cmd/legbled/cmd/amounts_test.go` (38 test cases)
- `cmd/legbled/cmd/quick_tx.go` (4 convenience tx commands)
- `cmd/legbled/cmd/dashboard.go` (dashboard query)
- `cmd/legbled/cmd/explore.go` (knowledge explorer)
- `cmd/legbled/cmd/root.go` (wiring)
- `reports/batch-13/SESSION-B13-5-CLI-UX.md` (this report)
