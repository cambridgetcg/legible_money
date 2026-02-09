# x/vesting_rewards — Block Reward Emission Progress

> PP Task: Wire block reward emission with bank integration, BeginBlocker, and gRPC handlers
> Started: 2026-02-09
> Territory: PARTIAL (80% existed, 20% critical wiring missing)
> Status: **COMPLETE**

## Status Legend
- [ ] Not started
- [~] In progress
- [x] Complete
- [!] Blocked

## Task Checklist

### Phase 1: Infrastructure (keeper dependencies)
- [x] 1. Create `types/expected_keepers.go` — BankKeeper + StakingKeeper interfaces
- [x] 2. Update `keeper/keeper.go` — Add bankKeeper, stakingKeeper fields + NewKeeper params
- [x] 3. Update `app/app.go` — Pass BankKeeper + VestingStakingAdapter to NewKeeper

### Phase 2: Bank Integration (the critical missing piece)
- [x] 4. Update `keeper/rewards.go` — DistributeBlockReward mints coins via BankKeeper
- [x] 5. Update `keeper/vesting.go` — ClaimRewards sends coins via BankKeeper
- [x] 6. FalsifyClaim clawback — state updates already implemented (bank transfer deferred to x/disputes integration)

### Phase 3: BeginBlocker (reward emission trigger)
- [x] 7. Update `module.go` — BeginBlocker calls DistributeBlockReward each block

### Phase 4: gRPC Handler Wiring
- [x] 8. Update `types/codec.go` — Wire all 12 real handlers (7 Msg + 5 Query), replaced nil stubs

### Phase 5: Gas Costs
- [x] 9. `app/gas.go` — vesting gas costs already present (create_vesting through complete_vesting); added `block_reward` to SystemTxTypes

### Phase 6: Tests
- [x] 10. Update `keeper/keeper_test.go` — 33 total tests (22 existing + 11 new bank integration tests)

### Phase 7: Build Verification
- [x] 11. `go build ./x/vesting_rewards/...` — Clean
- [x] 12. `go vet ./x/vesting_rewards/...` — Clean
- [x] 13. `go test ./x/vesting_rewards/...` — 33/33 PASS
- [x] 14. `go test ./x/...` — 21/21 module suites PASS, zero regressions

## Test Summary (33 tests)

### Existing Tests (22)
- CreateVesting: Success, Unauthorized, InvalidCategory
- PauseVesting: Success, Unauthorized
- ResumeVesting: Success, NotPaused
- AccelerateVesting: Defense, Replication, InvalidType, Unauthorized
- CompleteVesting: ByRecipient, ByAuthority, Unauthorized, AlreadyCompleted, FalsifiedSchedule
- ValidateBasic: MsgCreateVesting (4 cases), MsgAccelerateVesting (4 cases)
- VestingFullLifecycle

### New Bank Integration Tests (11)
- DistributeBlockReward: MintAndDistribute, ValidatorScaling (3/22), EmptyBlock, NilBankKeeper
- ClaimRewards: SendsCoins, NilBankKeeper
- RouteReward: ResearchSplit (22%), InvalidAmount
- FalsifyClaim: ClawbackCalculation, AlreadyFalsified

## Architecture Notes

### Reward Flow
```
BeginBlocker (height > 1, proposer present)
  → DistributeBlockReward(producer, validatorCount, hasTransactions)
    → Scale reward: baseReward * min(1, active/target)
    → RouteReward: 78% producer, 22% research fund
    → BankKeeper.MintCoins(vesting_rewards, totalMinted)
    → BankKeeper.SendCoinsFromModuleToAccount(producer, 78%)
    → BankKeeper.SendCoinsFromModuleToModule(research_fund, 22%)
    → Record BlockRewardDistribution

EndBlocker
  → UpdateClaimableAmount for all active vesting schedules
```

### Key Design Decisions
- Block producer identified from `ctx.BlockHeader().ProposerAddress`
- Validator count from `stakingKeeper.GetActiveValidatorCount()` via VestingStakingAdapter
- `hasTransactions` check: `len(DataHash) > 0 && activeValidatorCount > 0`
- Module account `vesting_rewards` registered with `{Minter, Burner}` in app.go maccPerms
- Nil-safe bankKeeper guard: if nil, record distribution but don't mint (devnet safety)
- VestingStakingAdapter wraps LegibleStakingKeeper → `GetActiveValidatorCount()`
- Skip genesis block (height <= 1) and missing proposer

## Files Modified
| File | Status | Changes |
|------|--------|---------|
| `x/vesting_rewards/types/expected_keepers.go` | NEW | BankKeeper + StakingKeeper interfaces |
| `x/vesting_rewards/keeper/keeper.go` | MOD | Add bankKeeper/stakingKeeper fields, updated NewKeeper, added GetStakingKeeper() |
| `x/vesting_rewards/keeper/rewards.go` | MOD | MintCoins + SendCoins in DistributeBlockReward, ResearchFundModuleName const |
| `x/vesting_rewards/keeper/vesting.go` | MOD | SendCoinsFromModuleToAccount in ClaimRewards |
| `x/vesting_rewards/module.go` | MOD | BeginBlocker with reward distribution, EndBlocker with claimable updates |
| `x/vesting_rewards/types/codec.go` | MOD | 12 real gRPC handlers replacing nil stubs |
| `app/app.go` | MOD | VestingRewardsKeeper init with BankKeeper + VestingStakingAdapter |
| `app/staking_adapter.go` | MOD | Added VestingStakingAdapter |
| `app/gas.go` | MOD | `block_reward` added to SystemTxTypes |
| `x/vesting_rewards/keeper/keeper_test.go` | MOD | +11 new tests, mock BankKeeper/StakingKeeper, setupKeeperWithBank helper |
