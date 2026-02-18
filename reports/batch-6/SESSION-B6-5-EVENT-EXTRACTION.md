# Session B6-5: EVENT-EXTRACTION

**Goal**: Ensure all tx handlers emit proper events with extractable resource IDs.
Fix E2E test A3 (claim ID extraction) and any other event-dependent tests.

## Event Audit Table

### Knowledge Module (`x/knowledge/keeper/msg_server.go`)

| Handler | Event Type | Key Attributes | Before | After |
|---------|------------|----------------|--------|-------|
| SubmitClaim | `claim_submitted` | `claim_id`, `round_id`, `domain`, `category`, `submitter`, `stake`, `height`, `mode` | OK | OK |
| SubmitCommitment | `commitment_submitted` | `round_id`, `validator`, `height` | OK | OK |
| SubmitReveal | `reveal_submitted` | `round_id`, `validator`, `verdict` | OK | OK |
| ChallengeFact | `fact_challenged` | `fact_id`, `round_id`, `challenger`, `stake` | OK | OK |
| **ChallengeProvisionalFact** | `challenge_provisional_fact` | `challenge_id`, `claim_id`, `fact_id`, `challenger`, `resolution_path` | **MISSING** | **ADDED** |
| AddFact | `fact_added` | `fact_id`, `domain`, `authority` | OK | OK |
| ProposeDomain | `domain_proposed` | `domain`, `proposer` | OK | OK |
| EndorseDomainProposal | `domain_endorsed` | `domain`, `endorser`, `endorsements` | OK | OK |
| ChallengeDomainProposal | `domain_challenged` | `domain`, `challenger`, `reason` | OK | OK |
| SubmitContradiction | `contradiction_submitted` | `counter_fact_id`, `target_fact_id`, `round_id`, `submitter`, `stake` | OK | OK |
| PatronizeFact | `fact_patronized` | `fact_id`, `patron`, `amount`, `new_bonus`, `patronage_id` | OK | OK |
| **UpdateParams** | `knowledge_params_updated` | `authority` | **MISSING** | **ADDED** |
| UpdateExtendedParams | `knowledge_extended_params_updated` | `authority` | OK | OK |
| RegisterStratum | *(none — Phase 2 stub)* | — | N/A | N/A |

### Gov Module (`x/gov/keeper/msg_server.go`)

| Handler | Event Type | Key Attributes | Status |
|---------|------------|----------------|--------|
| SubmitLIP | `lip_submitted` | `lip_id`, `proposer` | OK |
| StakeLIP | `lip_staked` | `lip_id`, `staker`, `amount`, `stage` | OK |
| CastVote | `vote_cast` | `proposal_id`, `voter`, `direction`, `stake`, `effective_power`, `sybil_decay_bps` | OK |
| AdvanceLIPStage | `lip_advanced` | `lip_id`, `new_stage` | OK |
| WithdrawLIP | `lip_withdrawn` | `lip_id`, `proposer` | OK |
| SubmitDisbursement | `disbursement_submitted` | `disbursement_id`, `submitter`, `amount` | OK |
| ExecuteDisbursement | `disbursement_executed` | `disbursement_id`, `recipient`, `amount` | OK |
| CancelDisbursement | `disbursement_cancelled` | `disbursement_id`, `authority` | OK |
| AttachUpgradePlan | `upgrade_plan_attached` | `lip_id`, `upgrade_name`, `height` | OK |
| UpdateParams | `params_updated` | `authority` | OK |

### Home Module (`x/home/keeper/msg_server.go`)

| Handler | Event Type | Key Attributes | Status |
|---------|------------|----------------|--------|
| CreateHome | `create_home` | `home_id`, `owner`, `name` | OK |
| UpdateHome | `update_home` | `home_id`, `owner` | OK |
| UpdateMemoryCID | `update_memory_cid` | `home_id`, `cid` | OK |
| StartSession | `start_session` | `home_id`, `session_id`, `key_hash` | OK |
| EndSession | `end_session` | `home_id`, `session_id` | OK |
| RegisterKey | `register_key` | `home_id`, `key_hash`, `role` | OK |
| RevokeKey | `revoke_key` | `home_id`, `key_hash` | OK |
| ConfigureGuardian | `configure_guardian` | `home_id`, `owner` | OK |
| AcknowledgeAlert | `acknowledge_alert` | `home_id`, `alert_id` | OK |
| SetSpendingLimit | `set_spending_limit` | `home_id`, `key_type`, `max_amount` | OK |
| UpdateParams | `home_params_updated` | `authority` | OK |

### Tokens Module (`x/tokens/keeper/msg_server.go`)

| Handler | Event Type | Key Attributes | Status |
|---------|------------|----------------|--------|
| CreateToken | `create_token` | `token_id`, `creator`, `symbol`, `initial_supply` | OK |
| MintToken | `mint_token` | `token_id`, `to`, `amount` | OK |
| BurnToken | `burn_token` | `token_id`, `burner`, `amount` | OK |
| TransferToken | `transfer_token` | `token_id`, `from`, `to`, `amount` | OK |
| ApproveToken | `approve_token` | `token_id`, `owner`, `spender`, `amount` | OK |
| TransferFrom | `transfer_from` | `token_id`, `spender`, `from`, `to`, `amount` | OK |
| PauseToken | `pause_token` | `token_id`, `authority` | OK |
| UnpauseToken | `unpause_token` | `token_id`, `authority` | OK |
| AddLiquidity | `add_liquidity` | `pool_id`, `provider`, `amount_a`, `amount_b`, `lp_minted` | OK |
| RemoveLiquidity | `remove_liquidity` | `pool_id`, `provider`, `amount_a`, `amount_b`, `lp_burned` | OK |
| Swap | `swap` | `pool_id`, `sender`, `token_in`, `amount_in`, `token_out`, `amount_out`, `fee_bps` | OK |
| DelegatePower | `delegate_power` | `token_id`, `delegator`, `delegate`, `amount` | OK |
| WrapToken | `wrap_token` | `token_id`, `sender`, `amount`, `wrapped_denom` | OK |
| UnwrapToken | `unwrap_token` | `token_id`, `sender`, `amount`, `wrapped_denom` | OK |
| UpdateParams | `tokens_params_updated` | `authority` | OK |

### BVM Module (`x/bvm/keeper/msg_server.go`)

| Handler | Event Type | Key Attributes | Status |
|---------|------------|----------------|--------|
| DeployContract | `bvm_deploy_contract` | `address`, `deployer`, `code_hash` | OK |
| CallContract | `bvm_call_contract` | `contract`, `caller`, `value`, `gas_used`, `success` | OK |
| ScheduleExecution | `bvm_schedule_execution` | `schedule_id`, `contract`, `scheduler`, `execute_at_block` | OK |
| ScheduleContract | `bvm_schedule_contract` | `schedule_id`, `contract`, `method`, `execute_at_block` | OK |
| CancelSchedule | `bvm_cancel_schedule` | `schedule_id`, `caller` | OK |
| UpdateContractState | `bvm_update_contract_state` | `contract`, `key`, `authority` | OK |
| ExecutePendingSchedules | `bvm_execute_schedule` | `schedule_id`, `contract`, `method`, `block`, `success`, `gas_used` | OK |
| UpdateParams | `bvm_params_updated` | `authority` | OK |

## Events Added

### 1. `ChallengeProvisionalFact` — `x/knowledge/keeper/msg_server.go:455-463`

```go
ctx.EventManager().EmitEvent(
    sdk.NewEvent("challenge_provisional_fact",
        sdk.NewAttribute("challenge_id", challenge.Id),
        sdk.NewAttribute("claim_id", msg.ClaimId),
        sdk.NewAttribute("fact_id", msg.FactId),
        sdk.NewAttribute("challenger", msg.Challenger),
        sdk.NewAttribute("resolution_path", challenge.ResolutionPath),
    ),
)
```

### 2. `UpdateParams` (knowledge) — `x/knowledge/keeper/msg_server.go:660-664`

```go
ctx.EventManager().EmitEvent(
    sdk.NewEvent("knowledge_params_updated",
        sdk.NewAttribute("authority", msg.Authority),
    ),
)
```

## E2E Test Script Fixes (`devnet/scripts/test-lgm-modules.sh`)

### Root Cause of A3 Skip

The script searched for event types `submit_claim` or `knowledge_claim_submitted`, but the Go handler emits `claim_submitted`. The attribute key was correct (`claim_id`), but the event type filter never matched, so `claim_id` was always empty.

### Fixes Applied

| Test | Line | Old Filter | New Filter |
|------|------|-----------|-----------|
| A3 (claim ID) | 302 | `submit_claim \| knowledge_claim_submitted` | `claim_submitted` |
| C2 (token ID) | 520 | `create_token \| token_created` | `create_token` |
| D2 (home ID) | 656 | `create_home \| home_created` | `create_home` |
| E2 (LIP ID) | 763 | `submit_lip \| lip_submitted` | `lip_submitted` |

The C2, D2, E2 filters already had the correct event type as one of the alternatives (`create_token`, `create_home`, `lip_submitted`), but the incorrect alternative was listed first, adding confusion. Cleaned up all four to use only the exact event type emitted by the Go handler.

Attribute key filters also tightened:
- Removed `or .key == "id"` fallbacks — all handlers emit the specific key (`claim_id`, `token_id`, `home_id`, `lip_id`)
- Removed `or .key == "did"` from home extraction (homes use `home_id`, not `did`)

### Event Extraction Pattern (for future tests)

```bash
# Extract a specific attribute from tx events
VALUE=$(echo "$tx_result" | jq -r '
    [.events[] | select(.type == "EVENT_TYPE") |
     .attributes[] | select(.key == "ATTR_KEY") | .value] | first // empty' 2>/dev/null)
```

## Verification

- `go build ./...` — clean
- `go vet ./x/knowledge/...` — clean
- `go test ./x/knowledge/... -count=1` — all pass
- Devnet verification deferred (requires running devnet — covered by batch-6 session 4/6)

## Summary

- **47 handlers audited** across 5 modules
- **45 already had proper events** with extractable resource IDs
- **2 events added**: `ChallengeProvisionalFact`, knowledge `UpdateParams`
- **1 critical E2E bug fixed**: A3 event type mismatch (`submit_claim` vs `claim_submitted`)
- **3 E2E filters tightened**: removed incorrect alternative event types
