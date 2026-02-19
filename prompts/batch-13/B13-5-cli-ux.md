# B13-5 — CLI UX: Human-Friendly Commands for the Cooperation Flow

## Context
The current CLI requires knowing exact flag names, bech32 addresses, and
module-specific subcommand patterns. For testnet onboarding, we need
higher-level commands that make the cooperation flow intuitive.

## Task

### 1. Add Convenience Aliases

Create `x/legible_cli/` or add to existing CLI files:

#### Quick Registration
```bash
# Current:
legbled tx legible-auth register --account-type human --did did:lgm:... --public-key ... --from mykey ...

# New shorthand:
legbled tx quick-register --type human --from mykey ...
# Auto-derives DID from key, fills public-key, handles the ceremony
```

#### Quick Partnership
```bash
# Current (3 separate commands):
legbled tx partnerships initiate --partner <addr> --lock-tier 1 --from human ...
legbled tx partnerships accept --partnership-id partnership-1 --from agent ...

# New:
legbled tx quick-partner --partner <name-or-addr> --lock-tier 1 --from human ...
# Then partner runs:
legbled tx quick-accept --from agent ...
# (auto-finds pending partnership for this address)
```

#### Quick Claim
```bash
# Current:
legbled tx knowledge submit-claim "statement" --domain physics --references PHYS-001,PHYS-002 --stake 100000ulgm --from agent ...

# New (also accepts):
legbled tx claim "statement" --domain physics --refs PHYS-001 PHYS-002 --stake 0.1LGM --from agent ...
# Accepts human-readable amounts (0.1LGM → 100000ulgm)
# Accepts space-separated refs
```

### 2. Dashboard Query

```bash
legbled query dashboard <address-or-name>
```

Outputs a human-readable summary:
```
═══════════════════════════════════════════
  Account: lgm1abc...def
  Type: agent | Reputation: 72.3%
  Balance: 12.45 LGM
═══════════════════════════════════════════
  Partnership: partnership-42 (active)
    Partner: lgm1xyz...uvw (human)
    Lock Tier: 2 (1.22× multiplier)
    Common Pot: 3.21 LGM
    Cooperation Score: 85%
    Split: 50/50
═══════════════════════════════════════════
  Knowledge Profile:
    Facts Submitted: 14 (12 verified, 2 pending)
    Total Citations: 47
    Domains: physics (8), chemistry (4), cosmology (2)
    Fundamentality Score: 0.42
═══════════════════════════════════════════
  Recent Activity:
    Block 10234: Submitted claim in physics
    Block 10189: Earned 0.05 LGM citation reward
    Block 10102: Partnership pot deposit 0.22 LGM
═══════════════════════════════════════════
```

### 3. Interactive Mode (Stretch Goal)

```bash
legbled interactive
```

A guided flow:
```
Welcome to Legible Money!
What would you like to do?
  1. Register a new account
  2. Form a partnership
  3. Submit a knowledge claim
  4. Check your dashboard
  5. Explore the knowledge tree
> _
```

### 4. Human-Readable Amounts

Add an amount parser that accepts:
- `0.1LGM` → `100000ulgm`
- `1LGM` → `1000000ulgm`
- `100000ulgm` → `100000ulgm` (passthrough)
- `1000000` → `1000000ulgm` (assume ulgm if no denom)

### 5. Knowledge Explorer

```bash
legbled query knowledge explore MATH-001
```

Shows the axiom and its citation tree:
```
MATH-001 — Peano Zero (axiom, confidence: 100.0%)
  Statement: Zero is a natural number...
  Domain: mathematics | Stratum: fundamental
  Citations: 23 | Fundamentality: axiomatic
  ├── Referenced by: MATH-015 (Addition is commutative)
  ├── Referenced by: MATH-023 (Induction principle)
  ├── Referenced by: CS-003 (Natural number encoding)
  └── ... 20 more
```

### 6. Tests

```go
func TestQuickRegisterDerivesDID(t *testing.T)
func TestHumanReadableAmounts(t *testing.T)
func TestDashboardOutput(t *testing.T)
func TestKnowledgeExplorer(t *testing.T)
```

### 7. Run

```bash
go test ./x/legible_cli/... -count=1 -v  # if new package
# or
go test ./... -run TestQuick -count=1
go build ./...
```

## Commit
```
feat(cli): human-friendly commands and dashboard for cooperation flow

Adds quick-register, quick-partner, quick-claim convenience commands.
Dashboard query shows account, partnership, and knowledge profile in
one view. Knowledge explorer shows axiom citation trees. Human-readable
amount parsing (0.1LGM → 100000ulgm).
```
