package keeper_test

// ==================================================================================
// OpenClaw Schedule Adversarial Probe Suite (OC-SCHED-1 through OC-SCHED-3)
//
// Attack surfaces tested:
// - Non-creator cancels schedule (steals refund)
// - Non-creator pauses schedule (denial of service)
// - UpdateParams authorization bypass
// ==================================================================================

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/schedule/keeper"
	"github.com/legible-money/legible/x/schedule/types"
)

// ==================================================================================
// OC-SCHED-1: Non-Creator Cancel Attack (Fund Theft)
//
// ATTACK: Creator sets up a schedule with prepaid fees. Attacker cancels it
// and the refund goes to the creator, but the attacker denies the service.
// More critically: if authorization is missing, attacker could redirect refund.
//
// DEFENSE: CancelSchedule checks msg.Creator == process.Creator.
// ==================================================================================

func TestOC_SCHED1_NonCreatorCancel(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create a schedule with prepaid fees
	process := types.ScheduleProcess{
		Id:              "sched-target",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "500000",
		FeePerExecution: "100",
		CreatedAtBlock:  50,
		NextExecuteAt:   200,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimeAbsolute,
				ExecuteAtBlock: 200,
			},
		},
	}
	k.SetProcess(ctx, process)

	// ATTACK: Non-creator tries to cancel and steal refund
	attackerAddr := sdk.AccAddress("attacker____________").String()
	_, err := msgServer.CancelSchedule(ctx, &types.MsgCancelSchedule{
		Creator:   attackerAddr,
		ProcessId: "sched-target",
	})
	if err == nil {
		t.Fatal("OC-SCHED-1 FAIL: non-creator was able to cancel schedule!")
	}

	// Verify schedule still active
	p, found := k.GetProcess(ctx, "sched-target")
	if !found {
		t.Fatal("process not found")
	}
	if p.Status != types.StatusActive {
		t.Fatalf("OC-SCHED-1 FAIL: schedule status changed by attacker: %s", p.Status)
	}
	if p.RemainingFee != "500000" {
		t.Fatalf("OC-SCHED-1 FAIL: remaining fee changed: %s", p.RemainingFee)
	}

	t.Logf("OC-SCHED-1 PASS: non-creator cancel correctly rejected")
}

// ==================================================================================
// OC-SCHED-2: Non-Creator Pause Attack (Denial of Service)
//
// ATTACK: Attacker pauses someone else's schedule to deny service.
// If unguarded, any user could disable scheduled operations.
//
// DEFENSE: PauseSchedule checks msg.Creator == process.Creator.
// ==================================================================================

func TestOC_SCHED2_NonCreatorPause(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	process := types.ScheduleProcess{
		Id:              "sched-target",
		Creator:         testCreator,
		Status:          types.StatusActive,
		RemainingFee:    "1000000",
		FeePerExecution: "100",
		CreatedAtBlock:  50,
		NextExecuteAt:   200,
		Condition: &types.ScheduleCondition{
			TimeCondition: &types.TimeCondition{
				Type:           types.TimePeriodic,
				IntervalBlocks: 10,
			},
		},
	}
	k.SetProcess(ctx, process)

	// ATTACK: Non-creator tries to pause
	attackerAddr := sdk.AccAddress("attacker____________").String()
	_, err := msgServer.PauseSchedule(ctx, &types.MsgPauseSchedule{
		Creator:   attackerAddr,
		ProcessId: "sched-target",
	})
	if err == nil {
		t.Fatal("OC-SCHED-2 FAIL: non-creator was able to pause schedule!")
	}

	// Verify still active
	p, _ := k.GetProcess(ctx, "sched-target")
	if p.Status != types.StatusActive {
		t.Fatalf("OC-SCHED-2 FAIL: schedule paused by attacker: %s", p.Status)
	}

	t.Logf("OC-SCHED-2 PASS: non-creator pause correctly rejected")
}

// ==================================================================================
// OC-SCHED-3: UpdateParams Authorization Bypass
//
// ATTACK: Non-authority tries to update schedule parameters.
// If unguarded, anyone could change max active schedules or fee limits.
//
// DEFENSE: UpdateParams checks msg.Authority == k.authority.
// ==================================================================================

func TestOC_SCHED3_UpdateParamsAuthBypass(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// ATTACK: Random user tries to update params
	attackerAddr := sdk.AccAddress("attacker____________").String()
	_, err := msgServer.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: attackerAddr,
		Params: &types.Params{
			MaxActivePerAccount:  999999,
			MinPrepaidExecutions: 1, // absurdly low
		},
	})
	if err == nil {
		t.Fatal("OC-SCHED-3 FAIL: unauthorized user was able to update params!")
	}

	// Verify params unchanged
	params := k.GetParams(ctx)
	if params.MaxActivePerAccount != 100 {
		t.Fatalf("OC-SCHED-3 FAIL: params modified: MaxActive=%d", params.MaxActivePerAccount)
	}

	t.Logf("OC-SCHED-3 PASS: param update authorization enforced")
}
