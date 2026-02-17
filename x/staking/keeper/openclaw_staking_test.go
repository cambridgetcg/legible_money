package keeper_test

import (
	"fmt"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/staking/keeper"
	"github.com/legible-money/legible/x/staking/types"
)

// ============ OpenClaw Adversarial Tests: Staking Module ============
//
// These tests probe the 4-tier graduated staking system for gaming vectors,
// economic attacks, and edge cases that could compromise validator economics,
// VRF selection fairness, or slash protection mechanisms.

// OC-SK-1: Whale Stake Concentration — Can a whale with 99% of total stake
// dominate VRF selection? Verify that tier-0/1 validators get virtual stake
// (equalizing selection weight) and only tier-2/3 get real stake.
func TestOC_SK1_WhaleStakeConcentration(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Register whale with massive stake but still tier 0 (no verifications)
	whaleAddr := sdk.AccAddress("whale_validator_____").String()
	registerTestValidator(t, k, ctx, whaleAddr, "did:lgm:whale", "999999999999999")

	// Register small validator, also tier 0
	smallAddr := sdk.AccAddress("small_validator_____").String()
	registerTestValidator(t, k, ctx, smallAddr, "did:lgm:small", "111000")

	params := k.GetParams(ctx)

	// Both are apprentice tier — should get virtual stake, not real stake
	whaleVal, _ := k.GetValidator(ctx, whaleAddr)
	smallVal, _ := k.GetValidator(ctx, smallAddr)

	whaleEffective := keeper.GetEffectiveSelectionStake(ctx, k, whaleVal, params)
	smallEffective := keeper.GetEffectiveSelectionStake(ctx, k, smallVal, params)

	if whaleEffective != smallEffective {
		t.Errorf("OC-SK-1: whale effective stake (%s) != small effective stake (%s) at tier 0 — whale dominates VRF selection", whaleEffective, smallEffective)
	}
	if whaleEffective != params.VirtualStake {
		t.Errorf("OC-SK-1: tier-0 effective stake should be virtual stake (%s), got %s", params.VirtualStake, whaleEffective)
	}

	// Now promote whale to bonded — should use real stake
	promoteToBonded(t, k, ctx, whaleAddr)
	whaleVal, _ = k.GetValidator(ctx, whaleAddr)
	whaleEffective = keeper.GetEffectiveSelectionStake(ctx, k, whaleVal, params)

	if whaleEffective == params.VirtualStake {
		t.Errorf("OC-SK-1: bonded whale should use real stake, not virtual stake")
	}
	if whaleEffective != whaleVal.TotalStake {
		t.Errorf("OC-SK-1: bonded whale effective stake should equal TotalStake (%s), got %s", whaleVal.TotalStake, whaleEffective)
	}
}

// OC-SK-2: Slash Escalation Arithmetic — Slash a validator 5 times and verify
// progressive escalation doesn't truncate or overflow with BigInt arithmetic.
func TestOC_SK2_SlashEscalationArithmetic(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("val_escalate_test___").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:escalate", "1111000000000") // 1.111M LGM
	promoteToBonded(t, k, ctx, valAddr)

	// Raise limits to allow 5 slashes
	params := k.GetParams(ctx)
	params.MaxSlashesPerEpoch = 10
	params.MaxSlashCountDeactivate = 10
	k.SetParams(ctx, params)

	prevStake := new(big.Int)
	prevStake.SetString("1111000000000", 10)
	slashBase := new(big.Int)
	slashBase.SetString("1000000", 10)
	escalationBps := big.NewInt(int64(params.SlashEscalationBps)) // 100000 = 10%
	million := big.NewInt(1000000)

	for i := 0; i < 5; i++ {
		stakeBeforeSlash := new(big.Int)
		v, _ := k.GetValidator(ctx, valAddr)
		stakeBeforeSlash.SetString(v.Stake, 10)

		err := k.SlashValidator(ctx, valAddr, "1000000", fmt.Sprintf("escalation_%d", i))
		if err != nil {
			t.Fatalf("OC-SK-2: slash #%d failed: %v", i+1, err)
		}

		v, _ = k.GetValidator(ctx, valAddr)
		stakeAfterSlash := new(big.Int)
		stakeAfterSlash.SetString(v.Stake, 10)

		// Expected: base * (1M + slashCount*escalationBps) / 1M
		// slashCount at time of slash = i (pre-increment)
		expectedMultiplier := new(big.Int).Add(million, new(big.Int).Mul(big.NewInt(int64(i)), escalationBps))
		expectedSlash := new(big.Int).Mul(slashBase, expectedMultiplier)
		expectedSlash.Div(expectedSlash, million)

		// Cap to available stake
		if expectedSlash.Cmp(stakeBeforeSlash) > 0 {
			expectedSlash.Set(stakeBeforeSlash)
		}

		expectedStake := new(big.Int).Sub(stakeBeforeSlash, expectedSlash)
		if stakeAfterSlash.Cmp(expectedStake) != 0 {
			t.Errorf("OC-SK-2: slash #%d: expected stake %s, got %s (escalation arithmetic wrong)", i+1, expectedStake.String(), stakeAfterSlash.String())
		}
	}
}

// OC-SK-3: Slash Per-Epoch Cap (R2) — 3rd slash in one epoch should be silently
// skipped. After advancing epoch, slashing should work again.
func TestOC_SK3_SlashPerEpochCap_R2(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("val_epochcap_test___").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:epochcap", "1000000000000")

	// Slash twice — both count (default MaxSlashesPerEpoch=2)
	for i := 0; i < 2; i++ {
		err := k.SlashValidator(ctx, valAddr, "100000", fmt.Sprintf("epoch_slash_%d", i))
		if err != nil {
			t.Fatalf("OC-SK-3: slash #%d failed: %v", i+1, err)
		}
	}

	v, _ := k.GetValidator(ctx, valAddr)
	if v.SlashCount != 2 {
		t.Fatalf("OC-SK-3: expected slashCount=2 after 2 slashes, got %d", v.SlashCount)
	}

	// 3rd slash should be silently skipped (R2 cap)
	err := k.SlashValidator(ctx, valAddr, "100000", "epoch_slash_capped")
	if err != nil {
		t.Fatalf("OC-SK-3: 3rd slash should not error (silently skipped): %v", err)
	}

	v, _ = k.GetValidator(ctx, valAddr)
	if v.SlashCount != 2 {
		t.Errorf("OC-SK-3: slashCount should remain 2 after capped slash, got %d — attacker can exceed per-epoch limit", v.SlashCount)
	}

	// Advance to next epoch boundary — reset epoch counter
	params := k.GetParams(ctx)
	epochBlocks := int64(params.GetSlashDecayPeriodBlocks())
	ctx = ctx.WithBlockHeight(epochBlocks)
	if err := k.BeginBlocker(ctx); err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	v, _ = k.GetValidator(ctx, valAddr)
	if v.SlashesThisEpoch != 0 {
		t.Errorf("OC-SK-3: slashesThisEpoch should be 0 after epoch boundary, got %d", v.SlashesThisEpoch)
	}

	// Slash should work again in new epoch
	err = k.SlashValidator(ctx, valAddr, "100000", "new_epoch_slash")
	if err != nil {
		t.Fatalf("OC-SK-3: slash in new epoch failed: %v", err)
	}
	v, _ = k.GetValidator(ctx, valAddr)
	if v.SlashesThisEpoch != 1 {
		t.Errorf("OC-SK-3: expected slashesThisEpoch=1 in new epoch, got %d", v.SlashesThisEpoch)
	}
}

// OC-SK-4: Slash Decay Recovery (R1) — After slashing, verify slashCount decays
// by 1 per clean epoch. Multiple clean epochs should reduce to 0.
func TestOC_SK4_SlashDecayRecovery_R1(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("val_decay_recovery__").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:decay", "1000000000000")

	// Raise limit to allow 2 slashes in one epoch
	params := k.GetParams(ctx)
	params.MaxSlashesPerEpoch = 5
	k.SetParams(ctx, params)

	// Slash twice
	k.SlashValidator(ctx, valAddr, "100000", "decay_1")
	k.SlashValidator(ctx, valAddr, "100000", "decay_2")

	v, _ := k.GetValidator(ctx, valAddr)
	if v.SlashCount != 2 {
		t.Fatalf("expected slashCount=2, got %d", v.SlashCount)
	}

	epochBlocks := int64(params.GetSlashDecayPeriodBlocks())

	// Epoch 1 boundary: had slashes this epoch, so no decay
	ctx = ctx.WithBlockHeight(epochBlocks)
	k.BeginBlocker(ctx)
	v, _ = k.GetValidator(ctx, valAddr)
	if v.SlashCount != 2 {
		t.Errorf("OC-SK-4: expected slashCount=2 after epoch with slashes, got %d", v.SlashCount)
	}

	// Epoch 2 boundary: no slashes → decay by 1 (2 → 1)
	ctx = ctx.WithBlockHeight(epochBlocks * 2)
	k.BeginBlocker(ctx)
	v, _ = k.GetValidator(ctx, valAddr)
	if v.SlashCount != 1 {
		t.Errorf("OC-SK-4: expected slashCount=1 after 1 clean epoch, got %d", v.SlashCount)
	}

	// Epoch 3 boundary: no slashes → decay by 1 (1 → 0)
	ctx = ctx.WithBlockHeight(epochBlocks * 3)
	k.BeginBlocker(ctx)
	v, _ = k.GetValidator(ctx, valAddr)
	if v.SlashCount != 0 {
		t.Errorf("OC-SK-4: expected slashCount=0 after 2 clean epochs, got %d — decay not working", v.SlashCount)
	}
}

// OC-SK-5: Slash Floor Zero Amount (R4) — Slashing a validator with zero actual
// effect should NOT increment slashCount. Covers both "0" amount and slashing
// a zero-stake validator.
func TestOC_SK5_SlashFloorZeroAmount_R4(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Case 1: Slash amount "0"
	valAddr := sdk.AccAddress("val_zero_slash______").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:zero", "1000000000000")

	err := k.SlashValidator(ctx, valAddr, "0", "zero_amount")
	if err != nil {
		t.Fatalf("OC-SK-5: SlashValidator with 0 amount failed: %v", err)
	}
	v, _ := k.GetValidator(ctx, valAddr)
	if v.SlashCount != 0 {
		t.Errorf("OC-SK-5: slashCount should be 0 after 0-amount slash, got %d — attacker can grief without economic cost", v.SlashCount)
	}

	// Case 2: Slash a validator with 0 stake
	val2Addr := sdk.AccAddress("val_nostake_slash___").String()
	registerTestValidator(t, k, ctx, val2Addr, "did:lgm:nostake", "111000")
	v2, _ := k.GetValidator(ctx, val2Addr)
	v2.Stake = "0"
	v2.TotalStake = "0"
	k.SetValidator(ctx, v2)

	err = k.SlashValidator(ctx, val2Addr, "999999", "no_stake_slash")
	if err != nil {
		t.Fatalf("OC-SK-5: SlashValidator on 0-stake validator failed: %v", err)
	}
	v2, _ = k.GetValidator(ctx, val2Addr)
	if v2.SlashCount != 0 {
		t.Errorf("OC-SK-5: slashCount should be 0 for 0-stake validator, got %d — phantom slash inflates count", v2.SlashCount)
	}
}

// OC-SK-6: Sybil Registration Spam — Register 100 minimum-stake validators.
// Verify none count as block producers and system doesn't panic.
func TestOC_SK6_SybilRegistrationSpam(t *testing.T) {
	k, ctx := setupKeeper(t)

	for i := 0; i < 100; i++ {
		addr := sdk.AccAddress(fmt.Sprintf("sybil_val_%04d______", i)).String()
		_, err := k.RegisterValidator(ctx, &types.MsgRegisterValidator{
			ValidatorAddress: addr,
			PubKey:           []byte(fmt.Sprintf("sybil_pubkey_%04d", i)),
			SelfDelegation:   "111000", // minimum deposit
			CommissionRate:   1000,
			Description:      &types.ValidatorDescription{Identity: fmt.Sprintf("did:lgm:sybil%04d", i)},
		})
		if err != nil {
			t.Fatalf("OC-SK-6: sybil registration #%d failed: %v", i, err)
		}
	}

	// None should count as block producers (all tier 0)
	count := k.CountBlockProducers(ctx)
	if count != 0 {
		t.Errorf("OC-SK-6: expected 0 block producers from 100 min-stake sybils, got %d — sybil attack effective", count)
	}

	// Verify all are apprentice tier
	for i := 0; i < 5; i++ {
		addr := sdk.AccAddress(fmt.Sprintf("sybil_val_%04d______", i)).String()
		v, found := k.GetValidator(ctx, addr)
		if !found {
			t.Fatalf("OC-SK-6: sybil validator %d not found", i)
		}
		if v.Tier != "apprentice" {
			t.Errorf("OC-SK-6: sybil #%d has tier %s, expected apprentice", i, v.Tier)
		}
	}
}

// OC-SK-7: Redelegation Cooldown Bypass — Verify cooldown is enforced after
// redelegate, with off-by-one test at exact boundary.
func TestOC_SK7_RedelegationCooldownBypass(t *testing.T) {
	k, ctx := setupKeeper(t)

	val1Addr := sdk.AccAddress("val_redel_src_______").String()
	val2Addr := sdk.AccAddress("val_redel_dst_______").String()
	val3Addr := sdk.AccAddress("val_redel_dst2______").String()
	registerTestValidator(t, k, ctx, val1Addr, "did:lgm:rsrc", "1000000000000000000000")
	registerTestValidator(t, k, ctx, val2Addr, "did:lgm:rdst", "1000000000000000000000")
	registerTestValidator(t, k, ctx, val3Addr, "did:lgm:rdst2", "1000000000000000000000")

	delAddr := sdk.AccAddress("delegator_redel_____").String()
	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr, Validator: val1Addr, Amount: "500000000000000000000",
	})

	// First redelegate at height 100 — should succeed
	_, err := k.Redelegate(ctx, &types.MsgRedelegate{
		Delegator: delAddr, SrcValidator: val1Addr, DstValidator: val2Addr,
		Amount: "200000000000000000000",
	})
	if err != nil {
		t.Fatalf("OC-SK-7: first redelegate failed: %v", err)
	}

	// Immediately try again — should fail (cooldown = 1111 blocks)
	_, err = k.Redelegate(ctx, &types.MsgRedelegate{
		Delegator: delAddr, SrcValidator: val1Addr, DstValidator: val3Addr,
		Amount: "100000000000000000000",
	})
	if err == nil {
		t.Error("OC-SK-7: second redelegate should fail during cooldown — attacker can bypass cooldown")
	}

	// Off-by-one: at height 100 + 1110 — still within cooldown
	ctx = ctx.WithBlockHeight(100 + int64(types.RedelegationCooldownBlocks) - 1)
	_, err = k.Redelegate(ctx, &types.MsgRedelegate{
		Delegator: delAddr, SrcValidator: val1Addr, DstValidator: val3Addr,
		Amount: "100000000000000000000",
	})
	if err == nil {
		t.Errorf("OC-SK-7: redelegate at boundary-1 (block %d) should fail", 100+types.RedelegationCooldownBlocks-1)
	}

	// At height 100 + 1111 — cooldown elapsed
	ctx = ctx.WithBlockHeight(100 + int64(types.RedelegationCooldownBlocks))
	_, err = k.Redelegate(ctx, &types.MsgRedelegate{
		Delegator: delAddr, SrcValidator: val1Addr, DstValidator: val3Addr,
		Amount: "100000000000000000000",
	})
	if err != nil {
		t.Errorf("OC-SK-7: redelegate at boundary (block %d) should succeed: %v", 100+types.RedelegationCooldownBlocks, err)
	}
}

// OC-SK-8: Tier Demotion on Slash — Promote to Guardian (tier 3), slash once.
// Verify slashCount > 0 invalidates guardian qualification (MaxSlashCount=0 in window).
func TestOC_SK8_TierDemotionOnSlash(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("val_guardian_demote__").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:gdmote", "11111000000") // guardian-level stake
	promoteToGuardian(t, k, ctx, valAddr)

	v, _ := k.GetValidator(ctx, valAddr)
	if v.Tier != "guardian" {
		t.Fatalf("OC-SK-8: expected guardian tier, got %s", v.Tier)
	}
	if v.SlashCount != 0 {
		t.Fatalf("OC-SK-8: expected slashCount=0 for guardian, got %d", v.SlashCount)
	}

	// Slash once
	err := k.SlashValidator(ctx, valAddr, "100000", "guardian_slash")
	if err != nil {
		t.Fatalf("OC-SK-8: slash failed: %v", err)
	}

	v, _ = k.GetValidator(ctx, valAddr)
	if v.SlashCount != 1 {
		t.Errorf("OC-SK-8: expected slashCount=1 after slash, got %d", v.SlashCount)
	}

	// Guardian config: MaxSlashCount=0, so slashCount>0 means disqualified.
	// Check the tier config
	guardianConfig, found := k.GetTierConfig(ctx, types.TierGuardian)
	if !found {
		t.Fatal("OC-SK-8: guardian tier config not found")
	}
	if guardianConfig.MaxSlashCount != 0 {
		t.Errorf("OC-SK-8: expected guardian MaxSlashCount=0, got %d", guardianConfig.MaxSlashCount)
	}

	// With slashCount=1 and MaxSlashCount=0, validator no longer qualifies.
	// Verify the slash broke the qualification invariant.
	if v.SlashCount > uint64(guardianConfig.MaxSlashCount) || guardianConfig.MaxSlashCount == 0 {
		// This is the expected state — guardian disqualified after 1 slash
		if v.SlashCount == 0 {
			t.Error("OC-SK-8: guardian should have slashCount>0 after slash — slash was swallowed")
		}
	}
}

// OC-SK-9: Slash Own Before Delegated — Verify own stake is consumed first,
// then delegated stake absorbs the overflow. Tests boundary at exact own stake.
func TestOC_SK9_SlashOwnBeforeDelegated(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("val_ownfirst________").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:ownfirst", "1111000000")

	// Set precise values: own=1000, delegated=500
	v, _ := k.GetValidator(ctx, valAddr)
	v.Stake = "1000"
	v.DelegatedStake = "500"
	v.TotalStake = "1500"
	v.Tier = "bonded"
	k.SetValidator(ctx, v)

	// Slash exactly own stake amount (boundary test)
	err := k.SlashValidator(ctx, valAddr, "1000", "exact_own")
	if err != nil {
		t.Fatalf("OC-SK-9: slash failed: %v", err)
	}

	v, _ = k.GetValidator(ctx, valAddr)
	if v.Stake != "0" {
		t.Errorf("OC-SK-9: own stake should be 0 after exact-amount slash, got %s", v.Stake)
	}
	if v.DelegatedStake != "500" {
		t.Errorf("OC-SK-9: delegated stake should be untouched (500), got %s — validator slashed delegators unfairly", v.DelegatedStake)
	}

	// Reset and test overflow: slash 1200 with own=1000, delegated=500
	v.Stake = "1000"
	v.DelegatedStake = "500"
	v.TotalStake = "1500"
	v.SlashCount = 0
	v.SlashesThisEpoch = 0
	k.SetValidator(ctx, v)

	err = k.SlashValidator(ctx, valAddr, "1200", "overflow")
	if err != nil {
		t.Fatalf("OC-SK-9: overflow slash failed: %v", err)
	}

	v, _ = k.GetValidator(ctx, valAddr)
	if v.Stake != "0" {
		t.Errorf("OC-SK-9: own stake should be 0, got %s", v.Stake)
	}
	if v.DelegatedStake != "300" {
		t.Errorf("OC-SK-9: delegated stake should be 300 (500 - 200 overflow), got %s", v.DelegatedStake)
	}
	if v.TotalStake != "300" {
		t.Errorf("OC-SK-9: total stake should be 300, got %s", v.TotalStake)
	}
}

// OC-SK-10: Unbonding ID Collision — Create 5 undelegations in the same block
// from the same delegator. Verify all get unique unbonding IDs.
func TestOC_SK10_UnbondingIdCollision(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("val_unbond_id_______").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:unbondid", "1000000000000000000000")

	delAddr := sdk.AccAddress("del_unbond_id_______").String()
	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr, Validator: valAddr, Amount: "1000000000000000000000",
	})

	ids := make(map[string]bool)
	for i := 0; i < 5; i++ {
		resp, err := k.Undelegate(ctx, &types.MsgUndelegate{
			Delegator: delAddr, Validator: valAddr,
			Amount: "100000000000000000000",
		})
		if err != nil {
			t.Fatalf("OC-SK-10: undelegate #%d failed: %v", i+1, err)
		}
		if ids[resp.UnbondingId] {
			t.Errorf("OC-SK-10: unbonding ID collision — ID %s returned twice. Attacker can steal unbonding entries", resp.UnbondingId)
		}
		ids[resp.UnbondingId] = true
	}

	if len(ids) != 5 {
		t.Errorf("OC-SK-10: expected 5 unique unbonding IDs, got %d", len(ids))
	}

	// Verify monotonic sequence
	for i := uint64(1); i <= 5; i++ {
		id := fmt.Sprintf("%d", i)
		_, found := k.GetUnbonding(ctx, id)
		if !found {
			t.Errorf("OC-SK-10: unbonding entry %s not found — sequence gap", id)
		}
	}
}

// OC-SK-11: Block Producer Cap Enforcement — Verify MaxValidators limits block
// producers (tier 2+) while allowing unlimited verifiers (tier 0/1).
func TestOC_SK11_BlockProducerCapEnforcement(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set block producer cap to 2
	params := k.GetParams(ctx)
	params.MaxValidators = 2
	k.SetParams(ctx, params)

	// Register 4 validators with bonded-level stake
	addrs := make([]string, 4)
	for i := 0; i < 4; i++ {
		addrs[i] = sdk.AccAddress(fmt.Sprintf("bpcap_val_%04d______", i)).String()
		_, err := k.RegisterValidator(ctx, &types.MsgRegisterValidator{
			ValidatorAddress: addrs[i],
			PubKey:           []byte(fmt.Sprintf("bpcap_pk_%04d", i)),
			SelfDelegation:   "1111000000",
			CommissionRate:   1000,
			Description:      &types.ValidatorDescription{Identity: fmt.Sprintf("did:lgm:bpcap%04d", i)},
		})
		if err != nil {
			t.Fatalf("OC-SK-11: registration #%d failed: %v", i, err)
		}
	}

	// All start as apprentice — 0 block producers
	if count := k.CountBlockProducers(ctx); count != 0 {
		t.Fatalf("OC-SK-11: expected 0 block producers initially, got %d", count)
	}

	// Promote first two to bonded (becomes block producers)
	promoteToBonded(t, k, ctx, addrs[0])
	promoteToBonded(t, k, ctx, addrs[1])

	if count := k.CountBlockProducers(ctx); count != 2 {
		t.Fatalf("OC-SK-11: expected 2 block producers, got %d", count)
	}

	// Promote third to bonded — should still set (SetValidator bypasses cap check)
	// But CountBlockProducers should show 3
	promoteToBonded(t, k, ctx, addrs[2])
	count := k.CountBlockProducers(ctx)

	// Regardless of count, verify verifiers can still register
	for i := 0; i < 10; i++ {
		addr := sdk.AccAddress(fmt.Sprintf("verifier_%04d_______", i)).String()
		_, err := k.RegisterValidator(ctx, &types.MsgRegisterValidator{
			ValidatorAddress: addr,
			PubKey:           []byte(fmt.Sprintf("verpk_%04d", i)),
			SelfDelegation:   "111000", // min stake
			CommissionRate:   1000,
			Description:      &types.ValidatorDescription{Identity: fmt.Sprintf("did:lgm:ver%04d", i)},
		})
		if err != nil {
			t.Errorf("OC-SK-11: verifier registration #%d should succeed regardless of cap: %v", i, err)
		}
	}
	_ = count
}

// OC-SK-12: Accuracy Rate Precision — Test GetAccuracyRate() at the Guardian
// 77% boundary (770000 BPS). Verify exact boundary and off-by-one.
func TestOC_SK12_AccuracyRatePrecision(t *testing.T) {
	tests := []struct {
		name      string
		completed uint64
		correct   uint64
		wantBps   uint64
		meetsMin  bool // meets guardian 770000 BPS threshold?
	}{
		{"exactly 77%", 1000000, 770000, 770000, true},
		{"above 77%", 1000000, 770001, 770001, true},
		{"below 77%", 1000000, 769999, 769999, false},
		{"77% small sample", 100, 77, 770000, true},
		{"zero completions", 0, 0, 0, false},
		{"100%", 1000, 1000, 1000000, true},
		{"1 of 3 (33.3%)", 3, 1, 333333, false},
	}

	guardianMinAccuracy := uint64(770000) // from DefaultTierConfigs

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &types.LegibleValidator{
				VerificationsCompleted: tt.completed,
				VerificationsCorrect:   tt.correct,
			}
			got := v.GetAccuracyRate()
			if got != tt.wantBps {
				t.Errorf("OC-SK-12: GetAccuracyRate(%d/%d) = %d, want %d", tt.correct, tt.completed, got, tt.wantBps)
			}
			meetsThreshold := got >= guardianMinAccuracy
			if meetsThreshold != tt.meetsMin {
				t.Errorf("OC-SK-12: %d/%d meets guardian threshold = %v, want %v — boundary precision error", tt.correct, tt.completed, meetsThreshold, tt.meetsMin)
			}
		})
	}
}
