# B12-7 — Session Key Capability Expansion

## The Problem
Session keys give agents operational autonomy — they can sign transactions
without the identity key. But the capability set may not cover all operations
an autonomous agent needs for knowledge work. Let's audit and expand.

## Current State

Check `app/ante_legible.go`, `checkCapability()` to see what capabilities are
defined. Also check what message types each capability authorizes.

```bash
grep -A 20 "func.*checkCapability" app/ante_legible.go
```

## Audit Checklist

An autonomous knowledge-working agent needs session keys that can:

| Operation | Capability Needed | Currently Supported? |
|-----------|------------------|---------------------|
| Submit claims | `knowledge.submit_claim` | Check |
| Challenge facts | `knowledge.challenge` | Check |
| Vote in verification | `knowledge.verify` | Check |
| Query chain state | (no tx needed) | ✓ (read-only) |
| Deposit to partnership pot | `partnerships.deposit` | Check |
| Propose pot operation | `partnerships.propose` | Check |
| Approve pot operation | `partnerships.approve` | Check |
| Safety freeze | `partnerships.freeze` | Check |
| Transfer tokens (small) | `bank.send` | Check |
| Stake/delegate | `staking.delegate` | Check |
| Vote in governance | `gov.vote` | Check |
| Submit research proposal | `knowledge.research_propose` | Probably missing |
| Vote on research proposal | `knowledge.research_vote` | Probably missing |

## The Fix

### 1. Audit Existing Capabilities

Read `app/ante_legible.go` and enumerate all currently supported capabilities.

### 2. Add Missing Capabilities

For each missing capability:
a) Define the capability string constant
b) Add the message type mapping in `checkCapability()`
c) Document the capability in a capabilities reference

### 3. Capability Groups (Presets)

Add preset capability bundles for common use cases:

```go
// CapabilityPresets defines common session key capability bundles.
var CapabilityPresets = map[string][]string{
    "knowledge-worker": {
        "knowledge.submit_claim",
        "knowledge.verify",
        "knowledge.challenge",
        "knowledge.research_vote",
    },
    "partnership-operator": {
        "partnerships.deposit",
        "partnerships.propose",
        "partnerships.approve",
        "partnerships.freeze",
    },
    "autonomous-agent": {
        // Full autonomy for agents that need to operate independently
        "knowledge.submit_claim",
        "knowledge.verify",
        "knowledge.challenge",
        "knowledge.research_vote",
        "partnerships.deposit",
        "partnerships.propose",
        "partnerships.approve",
        "bank.send",  // Subject to spending limits
    },
    "read-only": {
        // No tx capabilities — query only
    },
}
```

Usage:
```bash
legbled tx legible-auth create-session \
    --preset autonomous-agent \
    --duration 34272 \
    --from agent1 ...
```

### 4. Capability-Scoped Spending Limits

Session keys with `bank.send` should have an associated spending limit
(from the home module). Wire the capability check to also check the
home spending limit:

```go
// In checkCapability for bank.send:
if cap == "bank.send" {
    // Check home spending limit
    if home, found := k.homeKeeper.GetHomeByOwner(ctx, session.Owner); found {
        if err := k.homeKeeper.CheckAndDebitSpendingLimit(ctx, home.HomeId, "session", amount); err != nil {
            return err
        }
    }
}
```

### 5. Capability Expiry Independent of Session Expiry

Allow individual capabilities within a session to expire independently:

```go
type CapabilityGrant struct {
    Capability     string `json:"capability"`
    ExpiresAtBlock uint64 `json:"expires_at_block"` // 0 = same as session
    MaxUsesRemaining int64 `json:"max_uses_remaining"` // -1 = unlimited
}
```

This lets you create a session key that can submit claims forever but can only
do bank.send 5 times.

### 6. Tests

```go
func TestCapabilityPresets(t *testing.T)              // Preset bundles resolve correctly
func TestSessionKeyKnowledgeWork(t *testing.T)        // Session key can submit claims
func TestSessionKeyPartnershipOps(t *testing.T)       // Session key can propose/approve
func TestSessionKeyBankSendWithLimit(t *testing.T)    // bank.send respects spending limit
func TestSessionKeyDeniedCapability(t *testing.T)     // Unauthorized op rejected
func TestCapabilityExpiry(t *testing.T)               // Individual capability expiry
func TestCapabilityMaxUses(t *testing.T)              // Use count enforcement
```

### 7. Run

```bash
go test ./app/... -v -run TestCapability
go test ./x/auth/... -count=1
```

## Commit
```
feat(auth): expand session key capabilities for agent autonomy

Adds partnership and research fund capabilities to session key system.
Introduces capability presets (knowledge-worker, autonomous-agent, etc.)
for common agent roles. Session keys with bank.send enforce home
spending limits. Individual capabilities can have independent expiry
and use count limits.
```
