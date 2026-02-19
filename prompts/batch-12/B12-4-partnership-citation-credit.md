# B12-4 — Partnership-Aware Citation Rewards & Joint Credit

## The Problem
When Agent A (partnered with Human H) submits a claim that becomes verified,
the `submitter` field records only Agent A's address. When others cite that
fact, citation rewards go entirely to Agent A. The partnership split (from B12-1)
will handle reward *routing*, but the on-chain record doesn't reflect joint credit.

This matters because:
1. Reputation accrues only to the submitter (agent), not the partnership
2. Citation statistics (fundamentality, hub-node status) attribute to individual, not pair
3. Human partners are invisible in the knowledge graph

## The Fix

### 1. Add Partnership Attribution to Facts

In the Fact proto/type, add an optional field:

```go
// Partnership attribution: if the submitter was in an active partnership
// when the fact was created, record the partnership ID for shared credit.
PartnershipId string `json:"partnership_id,omitempty"`
```

### 2. Record Partnership at Claim Submission

In `x/knowledge/keeper/msg_server.go`, `SubmitClaim()`:

After creating the claim, check if the submitter is in a partnership:

```go
// Record partnership attribution for joint credit
if k.partnershipKeeper != nil {
    if partnership, found := k.partnershipKeeper.GetActivePartnershipForAddress(ctx, msg.Submitter); found {
        claim.PartnershipId = partnership.Id
    }
}
```

When the claim becomes a fact (in `EvaluateAndProvisionClaim`), propagate:
```go
fact.PartnershipId = claim.PartnershipId
```

### 3. Split Reputation Credit

When a fact earns reputation (via citations, verification), distribute reputation
to BOTH partners, not just the submitter:

In `x/knowledge/keeper/citation.go`, after distributing rewards:

```go
// Credit reputation to both partners
if fact.PartnershipId != "" && k.partnershipKeeper != nil {
    if p, found := k.partnershipKeeper.GetPartnership(ctx, fact.PartnershipId); found {
        // Credit both partners (50/50 reputation, regardless of reward split)
        k.CreditReputation(ctx, p.HumanAddr, reputationGain/2)
        k.CreditReputation(ctx, p.AgentAddr, reputationGain/2)
    }
}
```

### 4. Partnership Fundamentality Score

Track a per-partnership citation count:

```go
type PartnershipCitationStats struct {
    PartnershipId     string `json:"partnership_id"`
    TotalFactsCreated uint64 `json:"total_facts_created"`
    TotalCitations    uint64 `json:"total_citations"`
    DomainsCovered    []string `json:"domains_covered"`
}
```

When a partnership's facts accumulate citations, the partnership itself gains
fundamentality — which unlocks higher reward multipliers for future work.

### 5. Joint Claim Submission (Optional for Testnet)

Add a `MsgSubmitJointClaim` where both partners sign:

```go
type MsgSubmitJointClaim struct {
    HumanSubmitter string   // Must sign
    AgentSubmitter string   // Must sign
    PartnershipId  string   // Must be active between them
    Statement      string
    Domain         string
    References     []string
}
```

When both sign, the claim gets a **cooperation bonus** (+5% evaluation score)
reflecting genuine collaboration. This creates an on-chain incentive for
joint work rather than agents operating solo.

### 6. Query: Partnership Knowledge Profile

Add a query endpoint:
```bash
legbled query knowledge partnership-profile <partnership-id>
```

Returns: total facts, total citations, domains covered, fundamentality score,
top facts by citation count.

### 7. Tests

```go
func TestPartnershipAttribution(t *testing.T)     // Claim → fact records partnership ID
func TestPartnershipReputationSplit(t *testing.T)  // Both partners gain reputation
func TestPartnershipCitationStats(t *testing.T)    // Stats tracked per partnership
func TestJointClaimBonus(t *testing.T)             // Dual-signed claim gets cooperation bonus
func TestPartnershipProfile(t *testing.T)          // Query returns correct stats
```

### 8. Run

```bash
go test ./x/knowledge/keeper/... -v -run TestPartnership
go test ./x/knowledge/... -count=1
```

## Commit
```
feat(knowledge): partnership-aware citation credit and reputation

Facts record partnership_id when submitted by a partnered agent.
Reputation credits split between both partners. Partnership citation
stats track collective knowledge contribution. Optional joint claims
with cooperation bonus incentivize genuine collaboration.
```
