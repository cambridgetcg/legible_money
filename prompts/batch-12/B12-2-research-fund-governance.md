# B12-2 — Research Fund Governance

## The Problem
The knowledge module collects a 7% research tax on verification rewards and
sends it to the `research_fund` module account. But there's no mechanism to
propose, vote on, or disburse funds from this account. Money goes in; nothing
comes out.

## The Fix

### 1. Create Research Fund Governance in x/knowledge

Add research fund proposal types and a lightweight governance flow directly
in the knowledge module (not a separate module — it's small enough to embed).

### 2. Types

In `x/knowledge/types/types.go` or a new `research_fund.go`:

```go
// ResearchProposal represents a request to disburse from the research fund.
type ResearchProposal struct {
    ProposalId   string `json:"proposal_id"`
    Proposer     string `json:"proposer"`       // Must be a verified knowledge contributor
    Title        string `json:"title"`
    Description  string `json:"description"`
    Amount       string `json:"amount"`          // ulgm
    Recipient    string `json:"recipient"`       // Can be individual, partnership, or module
    Domain       string `json:"domain"`          // Knowledge domain this serves
    Status       string `json:"status"`          // proposed, voting, approved, rejected, executed
    VotesFor     uint64 `json:"votes_for"`
    VotesAgainst uint64 `json:"votes_against"`
    VotingEndsAt uint64 `json:"voting_ends_at"`
    CreatedAt    uint64 `json:"created_at"`
}

// Research fund constants
const (
    ResearchProposalDeposit     = "1000000"  // 1 LGM
    ResearchVotingPeriodBlocks  = 34272      // ~1 day
    ResearchQuorumBps           = 200_000    // 20% of eligible voters
    ResearchPassThresholdBps    = 600_000    // 60% yes to pass
    ResearchMaxDisburseBps      = 100_000    // Max 10% of fund per proposal
)
```

### 3. Eligibility

Not everyone can propose or vote:
- **Propose:** Must have ≥3 verified facts (proven knowledge contributor)
- **Vote:** Must have ≥1 verified fact OR be an active verifier (≥10 rounds)
- Weight: proportional to reputation score (ReputationScore in LegibleAccount)

### 4. Messages

```protobuf
message MsgProposeResearchFund {
    string proposer = 1;
    string title = 2;
    string description = 3;
    string amount = 4;
    string recipient = 5;
    string domain = 6;
}

message MsgVoteResearchProposal {
    string voter = 1;
    string proposal_id = 2;
    bool   vote_yes = 3;
}

message MsgExecuteResearchProposal {
    string executor = 1;
    string proposal_id = 2;
}
```

For testnet, implement these as Go types with hand-written ValidateBasic/GetSigners
(same pattern as existing types). Proto can come later.

### 5. Flow

```
1. Contributor submits proposal + 1 LGM deposit
2. Voting period: 1 day (~34k blocks)
3. Eligible voters cast votes (weighted by reputation)
4. If quorum (20%) + pass threshold (60%): approved
5. Anyone can execute after approval → funds transferred
6. If rejected: deposit returned (minus 10% to discourage spam)
7. Max 10% of fund per proposal → prevents treasury drain
```

### 6. Research Fund Balance Query

Add a query endpoint:
```bash
legbled query knowledge research-fund
```
Returns: balance, active proposals, total disbursed, total collected.

### 7. Partnership Integration

Research fund proposals can name a partnership as recipient. When executed,
the disbursement flows through `DistributeReward()` — so 22% goes to the
partnership's common pot, 78% split per partnership terms. This incentivizes
partnerships to propose research they'll jointly execute.

### 8. Tests

```go
func TestResearchFundProposal(t *testing.T)           // Happy path: propose → vote → execute
func TestResearchFundEligibility(t *testing.T)         // Ineligible proposer rejected
func TestResearchFundQuorum(t *testing.T)              // Below quorum → rejected
func TestResearchFundMaxDisburse(t *testing.T)         // >10% of fund → rejected
func TestResearchFundToPartnership(t *testing.T)       // Disbursement to partnership splits correctly
```

### 9. Run

```bash
go test ./x/knowledge/keeper/... -v -run TestResearchFund
go test ./x/knowledge/... -count=1
```

## Commit
```
feat(knowledge): research fund governance — propose, vote, disburse

The 7% research tax now has a disbursement mechanism. Knowledge
contributors (≥3 verified facts) can propose funding. Eligible
participants vote with reputation-weighted votes. 20% quorum +
60% pass threshold. Max 10% of fund per proposal. Supports
partnership recipients for joint research funding.
```
