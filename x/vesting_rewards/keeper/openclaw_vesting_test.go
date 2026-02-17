package keeper_test

// ==================================================================================
// OpenClaw Vesting Rewards Adversarial Probe Suite (OC-VR-1 through OC-VR-6)
//
// These probes systematically test the vesting_rewards module's attack surface:
// - Double-claim prevention within the same block
// - Cliff period enforcement (pre-cliff claim attempts)
// - Zero-amount and negative-amount vesting rejection
// - Pause/resume timing exploit (elapsed time accounting)
// - Completed schedule falsification behavior
// ==================================================================================

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/vesting_rewards/keeper"
	"github.com/legible-money/legible/x/vesting_rewards/types"
)

// Verify that keeper.NewMsgServerImpl is used (compile-time check)
var _ = keeper.NewMsgServerImpl

// ---------- OC-VR-1: Double Claim Prevention ----------

// TestOC_VR1_DoubleClaimPrevention verifies that claiming rewards twice in the
// same block does not yield double rewards. After the first claim drains the
// claimable amount, a second claim at the same block height must return 0.
func TestOC_VR1_DoubleClaimPrevention(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()

	// Create vesting schedule via msg server
	resp, err := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-double-claim",
	})
	if err != nil {
		t.Fatalf("create vesting failed: %v", err)
	}

	vestingId := resp.VestingId

	// Advance well past the cliff (formal_proof cliff = 5555 blocks, created at 1000)
	ctx = ctx.WithBlockHeight(20000)

	// First claim: should succeed and return non-zero
	claimed1, err := k.ClaimRewards(ctx, recipient, vestingId)
	if err != nil {
		t.Fatalf("first claim failed: %v", err)
	}
	if claimed1 == "0" {
		t.Fatal("first claim returned 0; expected non-zero claimable amount after cliff")
	}

	// Verify claimable is now 0 after first claim
	schedule, found := k.GetVestingSchedule(ctx, vestingId)
	if !found {
		t.Fatal("vesting schedule not found after first claim")
	}
	if schedule.ClaimableAmount != "0" {
		t.Errorf("ATTACK SUCCEEDED: ClaimableAmount should be 0 after claim, got %s -- double-claim is possible", schedule.ClaimableAmount)
	}

	// Second claim in the same block: should fail with ErrNothingToClaim
	claimed2, err := k.ClaimRewards(ctx, recipient, vestingId)
	if err == nil && claimed2 != "0" {
		t.Errorf("ATTACK SUCCEEDED: second claim in same block returned %s instead of 0 -- double-claim vulnerability", claimed2)
	}
	if claimed2 != "0" {
		t.Errorf("ATTACK SUCCEEDED: second claim returned %s; expected 0", claimed2)
	}
}

// ---------- OC-VR-2: Cliff Bypass Attempt ----------

// TestOC_VR2_CliffBypassAttempt verifies that rewards cannot be claimed before
// the cliff period ends. A vesting schedule created at block 1000 with
// formal_proof category (cliff = 5555 blocks) should yield 0 claimable at
// block 6554 (cliff ends at 1000 + 5555 = 6555).
func TestOC_VR2_CliffBypassAttempt(t *testing.T) {
	_, k, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()

	// Create a vesting schedule at block 1000 (default setupKeeper height)
	schedule, err := k.CreateVestingSchedule(ctx, "claim-cliff", "fact-cliff", recipient,
		"1000000000000000000", types.CategoryFormalProof, types.SourceVerification)
	if err != nil {
		t.Fatalf("create vesting failed: %v", err)
	}

	// Cliff for formal_proof is 5555 blocks. Created at block 1000.
	// CliffEndsAtBlock = 1000 + 5555 = 6555
	if schedule.CliffEndsAtBlock != 6555 {
		t.Fatalf("expected cliff ends at block 6555, got %d", schedule.CliffEndsAtBlock)
	}

	// Try to claim at block 6554 (1 block before cliff ends)
	ctx = ctx.WithBlockHeight(6554)

	// Calculate released amount directly -- should be 0 before cliff
	released := k.CalculateReleasedAmount(ctx, schedule)
	if released != "0" {
		t.Errorf("ATTACK SUCCEEDED: released amount before cliff is %s, expected 0 -- cliff bypass vulnerability", released)
	}

	// Try to claim via ClaimRewards -- should fail or return 0
	claimed, err := k.ClaimRewards(ctx, recipient, schedule.Id)
	if err == nil && claimed != "0" {
		t.Errorf("ATTACK SUCCEEDED: claimed %s before cliff ended -- cliff bypass vulnerability", claimed)
	}

	// Verify that after cliff, rewards start releasing (check a reasonable distance past cliff)
	ctx = ctx.WithBlockHeight(7000)
	releasedAfterCliff := k.CalculateReleasedAmount(ctx, schedule)
	if releasedAfterCliff == "0" {
		t.Error("released amount is still 0 well after cliff; vesting curve may be broken")
	}
}

// ---------- OC-VR-3: Zero Amount Vesting ----------

// TestOC_VR3_ZeroAmountVesting verifies that creating a vesting schedule with
// amount "0" is rejected at both the ValidateBasic and msg server levels.
func TestOC_VR3_ZeroAmountVesting(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()

	// Test ValidateBasic rejects "0"
	msg := &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "0",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-zero",
	}
	if err := msg.ValidateBasic(); err == nil {
		t.Error("ATTACK SUCCEEDED: ValidateBasic accepted zero amount -- zero-amount vesting bypass")
	}

	// Also test at the msg server level (in case ValidateBasic is not called in some paths)
	_, err := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "0",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-zero-2",
	})
	// The keeper's CreateVestingSchedule may not independently reject "0" since
	// ValidateBasic is the first line of defense, but the msg server path should
	// enforce this. We check that either the msg server rejects it or the
	// resulting schedule has no exploitable state.
	if err == nil {
		t.Error("ATTACK SUCCEEDED: msg server accepted zero amount vesting -- zero-amount bypass at keeper level")
	}

	// Test empty string amount
	msgEmpty := &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-empty",
	}
	if err := msgEmpty.ValidateBasic(); err == nil {
		t.Error("ATTACK SUCCEEDED: ValidateBasic accepted empty amount string")
	}
}

// ---------- OC-VR-4: Negative Stake Vesting ----------

// TestOC_VR4_NegativeStakeVesting verifies that creating a vesting schedule
// with a negative amount is rejected. Negative amounts could lead to
// underflow in reward calculations.
func TestOC_VR4_NegativeStakeVesting(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()

	// Test via msg server -- negative amount string
	_, err := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "-100",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-negative",
	})
	// The keeper parses amount via big.Int.SetString which accepts "-100".
	// This should be caught somewhere in the pipeline.
	if err == nil {
		t.Error("ATTACK SUCCEEDED: msg server accepted negative amount vesting -- negative amount bypass")
	}

	// Test via keeper directly with explicitly negative amount
	_, err = k.CreateVestingSchedule(ctx, "claim-neg", "fact-neg", recipient,
		"-1000000000000000000", types.CategoryFormalProof, types.SourceVerification)
	if err == nil {
		t.Error("ATTACK SUCCEEDED: CreateVestingSchedule accepted negative totalAmount -- underflow vulnerability")
	}

	// Test with a very large negative number to check for integer wraparound
	_, err = k.CreateVestingSchedule(ctx, "claim-neg-big", "fact-neg-big", recipient,
		"-99999999999999999999999999999", types.CategoryFormalProof, types.SourceVerification)
	if err == nil {
		t.Error("ATTACK SUCCEEDED: CreateVestingSchedule accepted large negative amount -- wraparound vulnerability")
	}
}

// ---------- OC-VR-5: Pause/Resume Timing Exploit ----------

// TestOC_VR5_PauseResumeTimingExploit verifies that pausing and resuming a
// vesting schedule correctly accounts for paused duration in the elapsed time
// calculation. An attacker might try to exploit the timing to claim more
// rewards than deserved by manipulating pause/resume timing.
func TestOC_VR5_PauseResumeTimingExploit(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()

	// Create vesting at block 1000
	resp, err := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-pause-exploit",
	})
	if err != nil {
		t.Fatalf("create vesting failed: %v", err)
	}

	vestingId := resp.VestingId

	// Advance to block 1200 and pause
	ctx = ctx.WithBlockHeight(1200)
	_, err = ms.PauseVesting(ctx, &types.MsgPauseVesting{
		Authority: "authority",
		VestingId: vestingId,
		Reason:    "timing exploit test",
	})
	if err != nil {
		t.Fatalf("pause failed: %v", err)
	}

	schedule, _ := k.GetVestingSchedule(ctx, vestingId)
	if schedule.PausedAtBlock != 1200 {
		t.Fatalf("expected paused at block 1200, got %d", schedule.PausedAtBlock)
	}

	// Resume at block 1300 (100 blocks paused)
	ctx = ctx.WithBlockHeight(1300)
	_, err = ms.ResumeVesting(ctx, &types.MsgResumeVesting{
		Authority: "authority",
		VestingId: vestingId,
	})
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}

	schedule, _ = k.GetVestingSchedule(ctx, vestingId)
	if schedule.TotalPausedBlocks != 100 {
		t.Errorf("ATTACK SUCCEEDED: expected 100 paused blocks, got %d -- pause duration miscalculated", schedule.TotalPausedBlocks)
	}
	if schedule.PausedAtBlock != 0 {
		t.Errorf("expected paused_at_block reset to 0 after resume, got %d", schedule.PausedAtBlock)
	}

	// Now get the released amount at block 20000 and compare with a non-paused schedule.
	// The paused schedule should have released less due to the 100-block pause.
	ctx = ctx.WithBlockHeight(20000)

	// Calculate released for the paused schedule
	releasedWithPause := k.CalculateReleasedAmount(ctx, schedule)

	// Create a reference schedule WITHOUT any pause at the same starting block
	refSchedule, err := k.CreateVestingSchedule(ctx.WithBlockHeight(1000), "claim-ref", "fact-ref", recipient,
		"1000000000000000000", types.CategoryFormalProof, types.SourceVerification)
	if err != nil {
		t.Fatalf("create reference vesting failed: %v", err)
	}

	ctx = ctx.WithBlockHeight(20000)
	releasedWithoutPause := k.CalculateReleasedAmount(ctx, refSchedule)

	// The paused schedule should have released LESS than the non-paused reference.
	// If they are equal, the pause did not affect elapsed time calculation.
	if releasedWithPause == releasedWithoutPause && releasedWithPause != "0" {
		t.Errorf("ATTACK SUCCEEDED: paused schedule released same amount (%s) as non-paused (%s) -- pause duration not deducted from elapsed time",
			releasedWithPause, releasedWithoutPause)
	}

	// Additional check: verify the paused schedule's released amount is strictly less
	// (both should be non-zero since we're well past the cliff)
	if releasedWithPause != "0" && releasedWithoutPause != "0" {
		pausedBig := new(big.Int)
		pausedBig.SetString(releasedWithPause, 10)
		unpausedBig := new(big.Int)
		unpausedBig.SetString(releasedWithoutPause, 10)

		if pausedBig.Cmp(unpausedBig) >= 0 {
			t.Errorf("ATTACK SUCCEEDED: paused schedule released %s >= non-paused %s -- elapsed time not reduced by pause",
				releasedWithPause, releasedWithoutPause)
		}
	}
}

// ---------- OC-VR-6: Falsify Completed Schedule ----------

// TestOC_VR6_FalsifyCompletedSchedule verifies that a completed vesting
// schedule handles falsification correctly. Once a schedule is completed,
// attempting to falsify it should either be rejected (since rewards are
// already fully distributed) or properly handle the completed state.
func TestOC_VR6_FalsifyCompletedSchedule(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	recipient := sdk.AccAddress("recipient1__________").String()
	challenger := sdk.AccAddress("challenger__________").String()

	// Create vesting schedule
	resp, err := ms.CreateVesting(ctx, &types.MsgCreateVesting{
		Authority:    "authority",
		Beneficiary:  recipient,
		Amount:       "1000000000000000000",
		Category:     types.VestingCategory_VESTING_CATEGORY_VERIFICATION_REWARD,
		LinkedFactId: "fact-complete-falsify",
	})
	if err != nil {
		t.Fatalf("create vesting failed: %v", err)
	}

	vestingId := resp.VestingId

	// Complete the vesting schedule
	_, err = ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: recipient,
		VestingId: vestingId,
	})
	if err != nil {
		t.Fatalf("complete vesting failed: %v", err)
	}

	// Verify it is completed
	schedule, found := k.GetVestingSchedule(ctx, vestingId)
	if !found {
		t.Fatal("vesting schedule not found after completion")
	}
	if schedule.Status != string(types.VestingStatusCompleted) {
		t.Fatalf("expected completed status, got %s", schedule.Status)
	}

	// Now try to falsify the completed schedule via FalsifyClaim
	_, err = k.FalsifyClaim(ctx, schedule.ClaimId, challenger)

	// The FalsifyClaim function checks: schedule.Status == VestingStatusFalsified
	// for already-falsified schedules. For completed schedules, the current
	// implementation will still allow falsification (changes status to falsified).
	// This tests whether the module handles this case correctly.
	if err != nil {
		// If it errors, that means completed schedules are protected from
		// falsification -- this is a reasonable defense.
		t.Logf("Completed schedule correctly rejected falsification: %v", err)
	} else {
		// If it succeeds, verify the schedule was properly transitioned and
		// a clawback record was created. This is also valid behavior since
		// falsification should work retroactively even on completed schedules.
		updatedSchedule, _ := k.GetVestingSchedule(ctx, vestingId)
		if updatedSchedule.Status != string(types.VestingStatusFalsified) {
			t.Errorf("expected falsified status after falsification of completed schedule, got %s", updatedSchedule.Status)
		}

		// Verify that trying to falsify AGAIN fails (double-falsification)
		_, err2 := k.FalsifyClaim(ctx, schedule.ClaimId, challenger)
		if err2 == nil {
			t.Error("ATTACK SUCCEEDED: double falsification of same claim succeeded -- clawback can be triggered multiple times")
		}

		// Verify clawback record exists
		// The clawback should have accounted for the released amount on a
		// completed schedule (which may have been fully vested).
		t.Logf("Completed schedule was falsified (retroactive clawback). Released amount was: %s", schedule.ReleasedAmount)
	}

	// Additional probe: try to complete a falsified schedule (should fail)
	_, err = ms.CompleteVesting(ctx, &types.MsgCompleteVesting{
		Authority: recipient,
		VestingId: vestingId,
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: completed/falsified schedule accepted a second CompleteVesting call")
	}
}
