# x/research — Research Bounties & Peer Review

## Purpose

Research submission with peer review voting, bounty system for incentivizing investigation, and treasury funding. Supports submission, challenges, multi-reviewer consensus, and bounty fulfillment.

## CLI Commands

### Transactions
```bash
legbled tx research submit-research [title] [content-hash] --domain [domain] --from alice
legbled tx research challenge-research [id] --stake 500000ulgm --from bob
legbled tx research review-research [id] --score 85 --verdict accept --from reviewer1
legbled tx research resolve-research [id] --from authority
legbled tx research create-bounty [title] --reward 10000000ulgm --deadline 100000 --from alice
legbled tx research claim-bounty [bounty-id] --from researcher1
legbled tx research fulfill-bounty [bounty-id] --from authority
legbled tx research fund-research --amount 5000000ulgm --from alice
```

### Queries
```bash
legbled query research research [id]
legbled query research submissions
legbled query research bounty [id]
legbled query research params
```

## Events

`ResearchSubmitted`, `ResearchChallenged`, `ResearchReviewed`, `ResearchResolved`, `BountyCreated`, `BountyClaimed`, `BountyFulfilled`

## Dependencies

BankKeeper, TreasuryKeeper

## BeginBlock

Expires bounties past deadline, returns locked rewards to creators.
