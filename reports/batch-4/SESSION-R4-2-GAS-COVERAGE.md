# R4 Session 2: Gas Coverage Sweep Report

**Date**: 2026-02-18
**Scope**: 9 modules audited for missing gas entries in `app/gas.go`
**Result**: 9 missing `UpdateParams` entries added; all operational handlers already covered

## Summary

| Metric | Value |
|--------|-------|
| Modules audited | 9 |
| Total handlers across 9 modules | 51 |
| Pre-existing gas entries | 42 (operational handlers) |
| Missing entries found | 9 (all UpdateParams) |
| Entries added | 9 |
| Total gas map entries (after) | 242 |
| Build | PASS |
| Tests (`go test ./app/...`) | PASS |

## Finding

Every module had gas entries for all operational message handlers. The systematic gap was **UpdateParams** — the governance-gated parameter update handler present in every module but missing from the gas map. Without these entries, UpdateParams transactions would fall through to `MinGasLimit` (22,222 gas) instead of the standard governance cost of 40,000.

## Complete Audit Table

| Module | Operation | Gas Key | Gas Cost | Rationale | New/Existing |
|--------|-----------|---------|----------|-----------|--------------|
| **autopoiesis** | Activate | `activate_autopoiesis` | 40,000 | Simple state toggle | Existing |
| | OverrideMultiplier | `override_multiplier` | 50,000 | Writes multiplier + validation | Existing |
| | FreezeMultiplier | `freeze_multiplier` | 30,000 | Simple state write | Existing |
| | **UpdateParams** | **`update_autopoiesis_params`** | **40,000** | **Governance, infrequent** | **NEW** |
| **billing** | RegisterProvider | `register_provider` | 80,000 | Multi-key write, provider record | Existing |
| | DeregisterProvider | `deregister_provider` | 30,000 | Delete state | Existing |
| | RequestQuote | `request_quote` | 20,000 | Read-heavy, minimal write | Existing |
| | ExecutePayment | `execute_payment` | 40,000 | Cross-module bank transfer | Existing |
| | **UpdateParams** | **`update_billing_params`** | **40,000** | **Governance, infrequent** | **NEW** |
| **capture_challenge** | SubmitChallenge | `submit_capture_challenge` | 100,000 | Security-sensitive, creates challenge record | Existing |
| | AddEvidence | `add_challenge_evidence` | 50,000 | Writes evidence to challenge | Existing |
| | ResolveChallenge | `resolve_capture_challenge` | 80,000 | Multi-key resolution + slashing | Existing |
| | **UpdateParams** | **`update_capture_challenge_params`** | **40,000** | **Governance, infrequent** | **NEW** |
| **capture_defense** | RequestQualification | `request_capture_qualification` | 80,000 | Multi-key write, creates qualification record | Existing |
| | EndorseQualification | `endorse_capture_qualification` | 40,000 | Writes endorsement | Existing |
| | **UpdateParams** | **`update_capture_defense_params`** | **40,000** | **Governance, infrequent** | **NEW** |
| **discovery** | RegisterAgent | `register_agent` | 60,000 | Creates agent profile + indexes | Existing |
| | UpdateProfile | `update_profile` | 30,000 | Simple state update | Existing |
| | AddCapability | `add_capability` | 25,000 | Append to capability list | Existing |
| | RemoveCapability | `remove_capability` | 20,000 | Remove from list | Existing |
| | DeregisterAgent | `deregister_agent` | 30,000 | Delete state | Existing |
| | Heartbeat | `heartbeat` | 15,000 | Simple timestamp write | Existing |
| | **UpdateParams** | **`update_discovery_params`** | **40,000** | **Governance, infrequent** | **NEW** |
| **ontology** | ProposeDomain | `propose_domain` | 80,000 | Creates proposal record | Existing |
| | VoteDomainProposal | `vote_domain_proposal` | 30,000 | Simple vote write | Existing |
| | UpdateDomain | `update_domain` | 40,000 | Modifies domain metadata | Existing |
| | RegisterLogicZone | `register_logic_zone` | 60,000 | Creates zone + indexes | Existing |
| | AcknowledgeIncompleteness | `acknowledge_incompleteness` | 30,000 | Simple state write | Existing |
| | **UpdateParams** | **`update_ontology_params`** | **40,000** | **Governance, infrequent** | **NEW** |
| **partnerships** | InitiatePartnership | `initiate_partnership` | 100,000 | Complex multi-key, escrow setup | Existing |
| | AcceptPartnership | `accept_partnership` | 40,000 | State transition + notification | Existing |
| | DepositToPot | `deposit_to_pot` | 50,000 | Bank transfer + state update | Existing |
| | DistributeReward | `distribute_reward` | 60,000 | Multi-recipient distribution | Existing |
| | ProposeOperation | `propose_operation` | 80,000 | Creates proposal + approval tracking | Existing |
| | ApproveOperation | `approve_operation` | 30,000 | Simple approval write | Existing |
| | RejectOperation | `reject_operation` | 30,000 | Simple rejection write | Existing |
| | SafetyFreeze | `safety_freeze` | 80,000 | Security-sensitive, freezes partnership | Existing |
| | RaiseCoercionSignal | `raise_coercion_signal` | 60,000 | Security signal + state change | Existing |
| | InitiateExit | `initiate_exit` | 50,000 | Begins exit process + unbonding | Existing |
| | **UpdateParams** | **`update_partnerships_params`** | **40,000** | **Governance, infrequent** | **NEW** |
| **qualification** | RequestQualification | `request_qualification` | 80,000 | Multi-key write, qualification record | Existing |
| | EndorseValidator | `endorse_validator` | 50,000 | Cross-module staking check + endorsement | Existing |
| | RenewQualification | `renew_qualification` | 30,000 | Simple expiry extension | Existing |
| | WithdrawQualification | `withdraw_qualification` | 50,000 | Delete + stake return | Existing |
| | **UpdateParams** | **`update_qualification_params`** | **40,000** | **Governance, infrequent** | **NEW** |
| **schedule** | CreateSchedule | `create_schedule` | 80,000 | Complex record creation + scheduling | Existing |
| | PauseSchedule | `pause_schedule` | 20,000 | Simple state toggle | Existing |
| | ResumeSchedule | `resume_schedule` | 20,000 | Simple state toggle | Existing |
| | CancelSchedule | `cancel_schedule` | 30,000 | Delete + cleanup | Existing |
| | AddFee | `add_fee` | 25,000 | Append fee entry | Existing |
| | **UpdateParams** | **`update_schedule_params`** | **40,000** | **Governance, infrequent** | **NEW** |

## Gas Cost Justification

All 9 new entries use **40,000 gas** — the established convention for governance parameter updates:
- `update_extended_params` (knowledge): 40,000
- `update_compute_params` (compute_pool): 40,000
- `advance_lip_stage` (gov): 40,000

UpdateParams handlers are governance-gated (authority check), infrequent, and perform a single store write. 40,000 is appropriate — high enough to prevent spam, low enough to not burden legitimate governance.

## Verification

```
go build ./...     # PASS
go test ./app/...  # PASS (0.993s)
```

## Files Modified

- `app/gas.go` — 9 new entries added to `TransactionGasCosts` map
