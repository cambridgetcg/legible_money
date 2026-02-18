# R4 Session 2: Complete Gas Coverage for 9 Partial Modules

## Context
R3-7 was supposed to fill gas gaps but didn't execute. 9 modules have handlers but incomplete gas entries in `app/gas.go`. Without gas costs, transactions consume only the base ante gas — making spam trivially cheap.

## Modules to Audit

1. **autopoiesis** — 4 handlers (UpdateParams, Activate, OverrideMultiplier, FreezeMultiplier)
2. **billing** — 5 handlers (RegisterProvider, DeregisterProvider, RequestQuote, ExecutePayment, UpdateParams)
3. **capture_challenge** — 4 handlers (SubmitChallenge, AddEvidence, ResolveChallenge, UpdateParams)
4. **capture_defense** — 3 handlers (RequestQualification, EndorseQualification, UpdateParams)
5. **discovery** — 7 handlers (RegisterAgent, UpdateProfile, AddCapability, RemoveCapability, DeregisterAgent, Heartbeat, UpdateParams)
6. **ontology** — 6 handlers (ProposeDomain, VoteDomainProposal, UpdateDomain, RegisterLogicZone, AcknowledgeIncompleteness, UpdateParams)
7. **partnerships** — 11 handlers (InitiatePartnership, AcceptPartnership, DepositToPot, DistributeReward, ProposeOperation, ApproveOperation, RejectOperation, SafetyFreeze, RaiseCoercionSignal, InitiateExit, UpdateParams + cold-start msgs)
8. **qualification** — 5 handlers (RequestQualification, EndorseValidator, RenewQualification, WithdrawQualification, UpdateParams)
9. **schedule** — 6 handlers (CreateSchedule, PauseSchedule, ResumeSchedule, CancelSchedule, AddFee, UpdateParams)

## Method

### For Each Module:
1. List every registered Msg handler (check `keeper/msg_server.go`)
2. Cross-reference against `app/gas.go` entries
3. For missing entries, add with costs following established scale:
   - UpdateParams: 40,000 (governance, infrequent)
   - Simple state reads/writes: 15,000-25,000
   - Complex writes (multi-key, cross-module): 40,000-60,000
   - Operations with iteration/computation: 60,000-100,000
   - Security-sensitive (challenges, freezes): 80,000-100,000
4. Use the naming convention `Gas<Module><Operation>` matching existing entries

### Reference Scale (from existing entries in app/gas.go)
```
provider_heartbeat:        15,000  (simple write)
update_compute_provider:   30,000  (moderate write)
register_compute_provider: 80,000  (complex, multi-key)
dispute_usage:             60,000  (creates dispute record)
```

## Verification
```bash
go build ./...
go test ./app/...
grep -c "Gas" app/gas.go  # count total entries before and after
```

## Report
Write to `/reports/batch-4/SESSION-R4-2-GAS-COVERAGE.md` with a complete table:
| Module | Operation | Gas Key | Gas Cost | Rationale | New/Existing |
