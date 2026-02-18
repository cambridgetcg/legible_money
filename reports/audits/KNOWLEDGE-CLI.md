# KNOWLEDGE-CLI: Knowledge Module CLI Gap Resolution

**Date**: 2026-02-18
**Files modified**: `x/knowledge/client/cli/tx.go`, `x/knowledge/client/cli/query.go`
**Build**: `go build ./...` — PASS

---

## Executive Summary

Closed all CLI gaps for the knowledge module. Every proto RPC with an implemented handler now has a corresponding CLI command.

| Metric | Before | After | Delta |
|--------|--------|-------|-------|
| TX CLI commands | 6 | 12 | +6 |
| Query CLI commands | 9 | 20 | +11 |
| **Total CLI surface** | **15** | **32** | **+17** |

---

## TX Command Mapping

| Proto RPC | Handler | CLI Command | Status |
|-----------|---------|-------------|--------|
| SubmitClaim | IMPL | `submit-claim` | existed |
| SubmitCommitment | IMPL | `submit-commitment` | existed |
| SubmitReveal | IMPL | `submit-reveal` | existed |
| ChallengeFact | IMPL | `challenge-fact` | existed |
| AddFact | IMPL | `add-fact` | **ADDED** |
| SubmitContradiction | IMPL | `submit-contradiction` | existed |
| PatronizeFact | IMPL | `patronize-fact` | existed |
| ProposeDomain | IMPL | `propose-domain` | **ADDED** |
| EndorseDomainProposal | IMPL | `endorse-domain` | **ADDED** |
| ChallengeDomainProposal | IMPL | `challenge-domain` | **ADDED** |
| RegisterStratum | IMPL (stub) | `register-stratum` | **ADDED** |
| UpdateParams | IMPL | — | skipped (governance-only) |
| ChallengeProvisionalFact | IMPL (hand-written) | `challenge-provisional` | **ADDED** |
| UpdateExtendedParams | IMPL (hand-written) | — | skipped (governance-only) |

### New TX Commands Detail

#### `add-fact [claim-text] [domain] [category] [confidence]`
Governance-gated fact creation (genesis/migration). Authority = sender.
Flags: `--logic-zone`, `--references`

#### `propose-domain [name] [description] [stake]`
Proposes a new ontological domain. Stake locked until proposal resolves.

#### `endorse-domain [domain-name]`
Endorses a proposed domain. Activation when MinEndorsements reached.

#### `challenge-domain [domain-name] [reason]`
Challenges a domain proposal with stated reason.

#### `register-stratum [name] [description]`
Registers an ontological stratum (Phase 2 feature, handler is a stub).

#### `challenge-provisional [claim-id] [stake] [reason]`
Challenges a provisionally accepted fact during its challenge window.
Flags: `--fact-id`, `--evidence`, `--counter-claim`

---

## Query Command Mapping

| Proto RPC | Keeper Impl | CLI Command | Status |
|-----------|-------------|-------------|--------|
| Params | YES | `params` | existed |
| Fact | YES | `fact` | existed |
| Facts | YES | `facts` | existed |
| FactsByDomain | YES | `facts-by-domain` | existed |
| Claim | YES | `claim` | existed |
| VerificationRound | YES | `round` | existed |
| ActiveRounds | YES | `active-rounds` | existed |
| Domain | YES | `domain` | existed |
| CounterFact | YES | `counter-fact` | existed |
| Domains | NO | `domains` | **ADDED** |
| DomainsByStratum | NO | `domains-by-stratum` | **ADDED** |
| DomainChildren | NO | `domain-children` | **ADDED** |
| DomainPath | NO | `domain-path` | **ADDED** |
| Stratum | NO | `stratum` | **ADDED** |
| Strata | NO | `strata` | **ADDED** |
| StratumRegistry | NO | `stratum-registry` | **ADDED** |
| ValidateDependency | NO | `validate-dependency` | **ADDED** |
| CalculateConfidence | NO | `calculate-confidence` | **ADDED** |
| DomainProposal | NO | `domain-proposal` | **ADDED** |
| DomainProposals | NO | `domain-proposals` | **ADDED** |

### New Query Commands Detail

#### `domains`
Lists domains with optional filters.
Flags: `--stratum`, `--parent`, `--pattern`, `--limit`, `--offset`

#### `domains-by-stratum [stratum-id]`
Lists all domains in a specific stratum.

#### `domain-children [domain-id]`
Lists child domains of a parent.
Flags: `--include-descendants`

#### `domain-path [domain-id]`
Returns the path from root to a domain.

#### `stratum [stratum-id]`
Queries a single stratum by ID.

#### `strata`
Lists all ontological strata.
Flags: `--min-level`, `--max-level`

#### `stratum-registry`
Returns the full stratum registry.

#### `validate-dependency [from-stratum] [to-stratum]`
Checks if a stratum dependency is valid.

#### `calculate-confidence [domain-id] [base-confidence]`
Calculates ontologically-adjusted confidence for a domain.

#### `domain-proposal [proposal-id]`
Queries a domain proposal with endorsements and challenges.

#### `domain-proposals`
Lists domain proposals.
Flags: `--limit`, `--offset`

---

## Keeper Implementation Status

The 11 new query commands call proto-generated gRPC client methods. However, the keeper query server only implements 9 of the 20 Query RPCs. The remaining 11 fall back to `UnimplementedQueryServer` and will return "rpc error: code = Unimplemented" at runtime until keeper methods are added.

| Keeper Has Query Server | Keeper Missing Query Server |
|------------------------|----------------------------|
| Params, Fact, Facts, FactsByDomain, Claim, VerificationRound, ActiveRounds, Domain, CounterFact | Stratum, Strata, StratumRegistry, Domains, DomainsByStratum, DomainChildren, DomainPath, ValidateDependency, CalculateConfidence, DomainProposal, DomainProposals |

**Future work**: Implement the 11 missing query server methods in `x/knowledge/keeper/grpc_query.go`.

---

## Build Output

```
$ go build ./...
(exit 0, no errors)
```

## CLI Help Output

```
$ legbled tx knowledge --help
Available Commands:
  add-fact              Add a fact via governance (genesis/migration)
  challenge-domain      Challenge a proposed domain
  challenge-fact        Challenge a verified fact
  challenge-provisional Challenge a provisionally accepted fact
  endorse-domain        Endorse a proposed domain
  patronize-fact        Stake tokens to protect a fact from pruning
  propose-domain        Propose a new knowledge domain
  register-stratum      Register an ontological stratum
  submit-claim          Submit a knowledge claim for verification
  submit-commitment     Submit a verification commitment (commit phase)
  submit-contradiction  Submit a counter-fact contradicting an existing fact
  submit-reveal         Reveal a verification verdict (reveal phase)

$ legbled query knowledge --help
Available Commands:
  active-rounds        List all active verification rounds
  calculate-confidence Calculate effective confidence for a domain
  claim                Query a claim by ID
  counter-fact         Query a counter-fact by ID
  domain               Query a knowledge domain by ID
  domain-children      Query child domains of a parent
  domain-path          Query the path from root to a domain
  domain-proposal      Query a domain proposal by ID
  domain-proposals     List domain proposals with optional status filter
  domains              List knowledge domains with optional filters
  domains-by-stratum   Query all domains in a specific stratum
  fact                 Query a fact by ID
  facts                List all verified facts
  facts-by-domain      Query facts in a specific domain
  params               Query the knowledge module parameters
  round                Query a verification round by ID
  strata               List all ontological strata
  stratum              Query a stratum by ID
  stratum-registry     Query the full stratum registry
  validate-dependency  Check if a stratum dependency is valid
```
