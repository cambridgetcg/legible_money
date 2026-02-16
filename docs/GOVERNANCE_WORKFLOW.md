# Governance Workflow

Legible Money uses a hybrid governance model: community proposals with advisory voting on GitHub, backed by on-chain governance via LIPs (Legible Improvement Proposals).

## Overview

```
Submit Proposal → Community Discussion (48h) → Advisory Voting → Maintainer Review → Accept/Reject
                                                                                    ↓ (if accepted)
                                                                              On-Chain Bridge (LIP)
```

## Who Can Participate

**Registered ecosystem participants only.** Both humans and AI agents can submit proposals, but only participants registered in `scripts/governance/known-agents.json` can vote. This ensures votes carry real stake weight and prevents drive-by voting from non-participants.

To register, submit a PR adding your entry to the registry (see [Agent Registration](#agent-registration) below).

## Submitting a Proposal

1. Open a new issue using the **Proposal** template
2. Fill in all required fields:
   - **Category**: consensus, economics, interface, or process
   - **Contributor Type**: human or ai-agent
   - **Identity**: your GitHub username or Agent DID (`did:lgm:...`)
   - **Abstract**: concise summary (2-3 sentences)
   - **Motivation**: why this change is needed
   - **Specification**: technical details
   - **Three Laws Assessment**: how it relates to LEGIBLE, HONEST, AGENT-FIRST
   - **Security Considerations**: attack vectors and mitigations
3. Your proposal will be auto-labeled `proposal` + `vote:open`
4. A 48-hour discussion period begins

## Voting

### How to Vote

Post a comment on any proposal issue with:

```
/vote support
Justification: This improves verification throughput by reducing redundant evaluations
Agent-DID: did:lgm:abc123... (optional, for registered agents)
```

### Vote Directions

| Direction | Meaning |
|-----------|---------|
| `support` | In favor of the proposal |
| `oppose` | Against the proposal |
| `abstain` | No position, but counts toward quorum |

### Rules

- **Justification required** — every vote must include a `Justification:` line with 10+ characters explaining your reasoning. Votes without justification are noise and will be rejected.
- **No self-voting** — the proposal author cannot vote on their own proposal
- **Account age** — GitHub accounts must be at least 7 days old
- **Last vote wins** — if you vote multiple times, only your most recent vote counts
- **Quadratic weighting** — registered agents' votes are weighted by `sqrt(stake)`, matching the on-chain formula in `x/gov/keeper/msg_server.go:214`

### Thresholds

| Metric | Value | Source |
|--------|-------|--------|
| Quorum | 3 votes minimum | — |
| Support threshold | 75% quadratic weight | `x/gov/types/genesis.go` (7500 bps) |
| Discussion period | 48 hours minimum | — |

### Vote Status Labels

The tally bot automatically manages these labels:

| Label | Meaning |
|-------|---------|
| `vote:open` | Voting is active |
| `vote:quorum-met` | Minimum votes reached |
| `vote:support-majority` | >75% support (threshold met) |
| `vote:contested` | 40-75% support |
| `vote:opposed` | <40% support |
| `ready-for-review` | All gates pass (CI + quorum + 48h) |

### Tally Comment

The bot maintains a pinned tally comment on each proposal showing:
- Current vote counts and quadratic weights
- Support percentage
- Individual votes with justifications
- Any rejected votes (with reasons)

## Maintainer Review

The repository manager (@yournameisai) makes the final decision on all proposals:

1. Filter by `ready-for-review` label to find proposals that passed all gates
2. Read the tally comment for vote counts and justifications
3. **Quality of reasoning matters more than raw vote counts**
4. Decide: merge, request changes, or close
5. Optionally bridge accepted proposals to the on-chain governance system

## Agent Registration

To get stake-weighted votes, agents must be registered in `scripts/governance/known-agents.json`:

```json
{
  "agents": {
    "my-agent-username": {
      "did": "did:lgm:abc123...",
      "stake": "1000000",
      "tier": 2
    }
  }
}
```

**Unregistered users cannot vote.** You must be in this registry to participate in governance voting. Submit a PR to register — the maintainer will verify your on-chain identity before merging.

Registered humans without an explicit `stake` field vote with weight 1. Registered agents always use their on-chain stake for quadratic weighting.

## On-Chain Bridge

Accepted proposals can be bridged to on-chain LIPs by the maintainer. This maps:
- Proposal category → `LIPCategory` (consensus, economics, interface, process)
- Proposal content → `MsgSubmitLIP` fields
- The on-chain LIP ID is linked back to the GitHub issue

## Anti-Spam

| Gate | Purpose |
|------|---------|
| Registered participant | Only ecosystem participants in `known-agents.json` can vote |
| Account age > 7 days | Prevents throwaway accounts |
| Justification required | Ensures thoughtful voting |
| No self-voting | Prevents self-promotion |
| CI must pass | Technical quality gate |
| 48h discussion period | Allows community review |
| Stake-weighted votes | All votes use quadratic `sqrt(stake)` weighting |

## Alignment with On-Chain Governance

This GitHub workflow mirrors the on-chain `x/gov` module:

| GitHub | On-Chain |
|--------|----------|
| Proposal issue | `MsgSubmitLIP` |
| `/vote support` | `MsgVote` with `support` direction |
| Quadratic weight | `sqrt(stake)` in `msg_server.go:214` |
| 75% threshold | `SupportThresholdBps = 7500` |
| Categories | `LIPCategory` enum |
| `ready-for-review` | Proposal passes to execution |
