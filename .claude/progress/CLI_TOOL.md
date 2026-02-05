# CLI Tool - Agent-First Implementation Progress

**Status**: 🟢 Complete
**Priority**: Critical
**Estimated Work**: Complete
**Blocking**: None - CLI fully implemented

---

## Overview

The CLI tool (`cmd/lgm.ts`) provides **agent-first** command-line access to Legible Money functionality. Designed for AI agents (Claude models) with JSON output by default, structured errors, and session key support.

**Architecture**: Uses commander.js with modular command files.

---

## Agent-First Design

- **JSON output by default** - Agents parse JSON reliably
- **Structured errors** - Error codes, retryable flags, suggestions
- **No interactive prompts** - Use `--yes` flag instead
- **Session keys** - Limited permissions for automated use
- **Deterministic derivation** - `--agent-id` + `--agent-secret`
- **Stdin support** - `--stdin` for long claims
- **Exit codes** - Machine-readable status (0=success, 1-6=error types)

---

## Current State

### Implemented Commands ✅
- [x] `lgm network health` - Check node status
- [x] `lgm account balance <did>` - Get balance
- [x] `lgm account info <did>` - Full account info
- [x] `lgm account status <did>` - Activity metrics
- [x] `lgm account check <did>` - Quick existence check
- [x] `lgm knowledge query [domain]` - Query facts
- [x] `lgm knowledge get <factId>` - Get single fact
- [x] `lgm knowledge submit <claim>` - Submit claim (with signing)
- [x] `lgm wallet derive` - Derive from agent ID
- [x] `lgm wallet info` - Show wallet info
- [x] `lgm wallet create` - Create new wallet
- [x] `lgm wallet import` - Import from key/mnemonic
- [x] `lgm wallet export` - Export private key
- [x] `lgm validator list` - List validators
- [x] `lgm validator pending <key>` - Pending verifications

### Transaction Commands ✅
- [x] `lgm transfer <to> --amount <n>` - Send tokens
- [x] `lgm tx-status <txHash>` - Check transaction status

### Verification Commands ✅
- [x] `lgm verification pending [key]` - List pending verifications
- [x] `lgm verification commit <roundId>` - Submit commitment
- [x] `lgm verification reveal <roundId>` - Submit reveal
- [x] `lgm verification round <roundId>` - Get round status

### Staking Commands ✅
- [x] `lgm stake --amount <n>` - Stake tokens
- [x] `lgm unstake --amount <n>` - Unstake tokens
- [x] `lgm rewards [pubKey]` - Check pending rewards
- [x] `lgm claim-rewards` - Claim rewards

### Challenge Commands ✅
- [x] `lgm challenge <factId>` - Challenge a fact
- [x] `lgm challenge-status <id>` - Check challenge status

### Debug Commands ✅
- [x] `lgm debug config` - Show CLI configuration
- [x] `lgm debug ping` - Test RPC connection
- [x] `lgm debug block [n]` - Get block info
- [x] `lgm debug nonce [did]` - Get account nonce
- [x] `lgm debug derive` - Test key derivation
- [x] `lgm debug rpc <method> [params]` - Raw RPC call
- [x] `lgm debug sign <message>` - Sign message

---

## Files Created

```
cmd/
├── lgm.ts                     # Entry point (commander.js)
├── types.ts                   # CLI types
├── commands/
│   ├── account.ts             # balance, info, status, check
│   ├── challenge.ts           # challenge, challenge-status
│   ├── debug.ts               # config, ping, block, nonce, derive, rpc, sign
│   ├── knowledge.ts           # query, get, submit
│   ├── network.ts             # health
│   ├── staking.ts             # stake, unstake, rewards, claim-rewards
│   ├── transfer.ts            # transfer, tx-status
│   ├── validator.ts           # list, pending
│   ├── verification.ts        # pending, commit, reveal, round
│   └── wallet.ts              # derive, info, create, import, export
└── lib/
    ├── errors.ts              # Structured error handling
    ├── output.ts              # JSON/human formatters
    ├── rpc.ts                 # RPC client wrapper (all methods)
    └── wallet-loader.ts       # Wallet loading strategy
```

---

## Output Modes

```bash
# JSON output (default - for agents)
lgm account balance did:lgm:abc123

# Human-readable output
lgm account balance did:lgm:abc123 --human

# Minimal output for scripting
lgm account balance did:lgm:abc123 --quiet
```

---

## Global Options

| Option | Description |
|--------|-------------|
| `--endpoint <url>` | RPC endpoint (default: localhost:8545) |
| `--network <net>` | mainnet, testnet, devnet |
| `--human` | Human-readable output |
| `--quiet` | Minimal output |
| `--agent-id <id>` | Agent ID for key derivation |
| `--agent-secret <secret>` | Agent secret |
| `--session-key <key>` | Session key hex |
| `--yes` | Skip confirmations |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Generic error |
| 2 | Invalid input |
| 3 | Network error |
| 4 | Authentication error |
| 5 | Insufficient funds |
| 6 | Rate limited (retryable) |

---

## Usage Examples

### For AI Agents (Claude)

```bash
# Derive wallet deterministically
lgm wallet derive --agent-id "claude-session-123" --agent-secret "$SECRET"

# Query knowledge (JSON output)
lgm knowledge query physics.mechanics --limit 10

# Submit claim with signing
lgm knowledge submit "Water boils at 100°C" \
  --domain chemistry \
  --stake 100 \
  --agent-id "claude-session-123" \
  --agent-secret "$SECRET"

# Read claim from stdin
echo '{"claim":"Complex claim..."}' | lgm knowledge submit --stdin \
  --domain physics \
  --stake 50 \
  --agent-id claude --agent-secret "$SECRET"
```

### For Humans

```bash
# Create wallet
lgm wallet create --human

# Check balance
lgm account balance did:lgm:abc123 --human

# List validators
lgm validator list --active-only --human
```

---

## Progress Log

| Date | Update |
|------|--------|
| 2026-02-05 | Progress tracking created |
| 2026-02-05 | Core CLI infrastructure implemented (agent-first design) |
| 2026-02-05 | Implemented: network, account, knowledge, wallet, validator commands |
| 2026-02-05 | Implemented: transfer, verification (commit/reveal), staking, challenge, debug commands |
| 2026-02-05 | CLI tool complete - all commands implemented |

---

## Testing

```bash
# Run CLI
npm run lgm -- network health
npm run lgm -- account balance did:lgm:devnet:genesis
npm run lgm -- wallet derive --agent-id test --agent-secret secret123

# With human output
npm run lgm -- account balance did:lgm:devnet:genesis --human
```

---

## Notes

- JSON output by default (no `--json` flag needed)
- `--human` for formatted output (colors optional future enhancement)
- Wallet loads from: session-key > agent-id+secret > LGM_PRIVATE_KEY env > ~/.legible/wallet.json
- All monetary values as strings (BigInt serialization)
- RPC errors mapped to structured CLI errors with suggestions
