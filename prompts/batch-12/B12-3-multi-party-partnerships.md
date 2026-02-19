# B12-3 — Multi-Party Partnership Support

## The Problem
Partnerships are strictly 1 human + 1 agent. Real knowledge work often involves
teams: a human researcher with 3 specialized agents, or a lab with 2 humans and
5 agents. The current system forces these into multiple bilateral partnerships,
losing the benefits of collective pot management and shared deliberation.

## The Fix — Consortia

### 1. Design: Consortium as Partnership Extension

Rather than rewriting partnerships, add a **Consortium** type that wraps multiple
participants with the same economic model:

```go
type Consortium struct {
    Id               string             `json:"id"`
    Name             string             `json:"name"`
    Status           string             `json:"status"`    // forming, active, dissolved
    Members          []*ConsortiumMember `json:"members"`
    CommonPotBalance string             `json:"common_pot_balance"`
    TotalEarned      string             `json:"total_earned"`
    SplitConfig      *ConsortiumSplit   `json:"split_config"`
    LockTier         uint32             `json:"lock_tier"`
    LockExpiresAt    uint64             `json:"lock_expires_at"`
    FormedAtBlock    uint64             `json:"formed_at_block"`
    RequiredApprovals uint32            `json:"required_approvals"` // For pot spending
    CooperationScore uint64             `json:"cooperation_score"`
}

type ConsortiumMember struct {
    Address     string `json:"address"`
    AccountType string `json:"account_type"` // human or agent
    ShareBps    uint64 `json:"share_bps"`    // Member's share of non-pot rewards (must sum to 1M)
    JoinedAt    uint64 `json:"joined_at"`
    Role        string `json:"role"`         // founder, member, observer
}

type ConsortiumSplit struct {
    CommonPotShareBps uint64 `json:"common_pot_share_bps"` // Still 22% default
    // Individual shares defined per-member in ConsortiumMember.ShareBps
}
```

### 2. Formation Flow

```
1. Founder creates consortium with initial members (2-11 max)
2. Each member must accept (like partnership accept)
3. Formation window: 2× standard partnership window
4. Unanimous acceptance required → active
5. Any member can propose adding new members (requires majority approval)
```

### 3. Deliberation

For pot spending, use the partnership deliberation system extended to N members:
- **Micro/Small (<5%):** Proposer + 1 other member
- **Medium (5-22%):** Proposer + majority
- **Large/Major (>22%):** Proposer + `RequiredApprovals` (default: all members)

Counter-proposals work the same way.

### 4. Reward Distribution

```go
func (k Keeper) DistributeConsortiumReward(ctx sdk.Context, consortium *Consortium, gross string, source string) error {
    // Same structure as partnership but N-way split
    // 1. Apply lock multiplier
    // 2. 22% to common pot
    // 3. 78% split per member ShareBps
}
```

### 5. Exit

Individual members can exit:
- Exiting member forfeits their share of the pot (based on exit penalty table)
- Remaining members' shares redistribute proportionally
- If membership drops below 2, consortium dissolves

### 6. Knowledge Module Integration

Update the `RouteReward()` function (from B12-1) to also check consortia:

```go
// In RouteReward, after partnership check:
if !found && k.partnershipKeeper != nil {
    consortium, cFound := k.partnershipKeeper.GetActiveConsortiumForAddress(ctx, addr.String())
    if cFound {
        return k.routeConsortiumReward(ctx, consortium, amount, sourceType)
    }
}
```

### 7. Priority: Partnership First

For v0.1 testnet, the bilateral partnership is the primary mechanism.
Consortia are an extension for v0.2 unless there's time. Design the interface
now so B12-1's `RouteReward` can be extended later without refactoring.

Actually, given testnet scope: **implement the data types and CRUD only**.
The formation/deliberation/reward flows can share 90% of the partnership code.
Write the types, storage, and basic formation. Skip deliberation N-way for now
(use binary approve/reject from each member).

### 8. Tests

```go
func TestConsortiumFormation(t *testing.T)       // Create, accept, activate
func TestConsortiumRewardSplit(t *testing.T)     // N-way split with pot
func TestConsortiumMemberExit(t *testing.T)      // Exit with penalty
func TestConsortiumDissolution(t *testing.T)     // Drop below minimum
func TestConsortiumMaxMembers(t *testing.T)      // 11 member cap
```

### 9. Run

```bash
go test ./x/partnerships/keeper/... -v -run TestConsortium
go test ./x/partnerships/... -count=1
```

## Commit
```
feat(partnerships): consortium support — multi-party partnerships

Adds Consortium type for 2-11 member partnerships. Supports mixed
human/agent membership with per-member share allocation. Formation
requires unanimous acceptance. Exit forfeits pot share. Designed
for integration with RouteReward for N-way knowledge reward splits.
```
