package keeper_test

// ==================================================================================
// OpenClaw Channels Adversarial Probe Suite (OC-CH-1 through OC-CH-4)
//
// Attack surfaces tested:
// - Non-party channel update (third party modifies state)
// - Nonce rollback (submit lower nonce to revert spent amount)
// - Settle during active dispute (premature settlement)
// - Overspend (spent exceeds deposited)
// ==================================================================================

import (
	"testing"

	"github.com/legible-money/legible/x/channels/keeper"
	"github.com/legible-money/legible/x/channels/types"
)

// ==================================================================================
// OC-CH-1: Third-Party Channel State Manipulation
//
// ATTACK: Payer opens channel with Provider. Third party tries to submit state
// updates. If unguarded, attacker could drain the channel or deny service.
//
// DEFENSE: UpdateChannel checks msg.Sender is payer or provider.
// ==================================================================================

func TestOC_CH1_ThirdPartyStateManipulation(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, err := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender:        payer1,
		Receiver:      provider1,
		Deposit:       "5000000",
		TimeoutHeight: 10000,
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// ATTACK: Third party tries to update channel state
	attacker := testAddr("attacker1")
	_, err = msgServer.UpdateChannel(ctx, &types.MsgUpdateChannel{
		Sender:             attacker,
		ChannelId:          resp.ChannelId,
		Sequence:           1,
		NewBalanceReceiver: "5000000", // drain entire deposit to receiver
	})
	if err == nil {
		t.Fatal("OC-CH-1 FAIL: third party was able to update channel state!")
	}

	// Verify state unchanged
	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Spent != "0" {
		t.Fatalf("OC-CH-1 FAIL: channel state modified by attacker: spent=%s", ch.Spent)
	}

	t.Logf("OC-CH-1 PASS: third-party state manipulation correctly rejected")
}

// ==================================================================================
// OC-CH-2: Nonce Rollback Attack
//
// ATTACK: Submit state update with nonce=5, then try nonce=3 with lower spent.
// If unguarded, the payer could "undo" payments to the provider.
//
// DEFENSE: UpdateChannel checks msg.Sequence > ch.Nonce.
// ==================================================================================

func TestOC_CH2_NonceRollback(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender:        payer1,
		Receiver:      provider1,
		Deposit:       "5000000",
		TimeoutHeight: 10000,
	})

	// Legitimate update: nonce 5, spent 3M
	_, err := msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender:    payer1,
		ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{
			Nonce: 5,
			Spent: "3000000",
		},
	})
	if err != nil {
		t.Fatalf("first update: %v", err)
	}

	// ATTACK: Rollback to nonce 3 with lower spent
	_, err = msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender:    payer1,
		ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{
			Nonce: 3,
			Spent: "1000000", // trying to undo 2M in payments
		},
	})
	if err == nil {
		t.Fatal("OC-CH-2 FAIL: nonce rollback allowed! Payments could be reversed.")
	}

	// Verify original state preserved
	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Nonce != 5 || ch.Spent != "3000000" {
		t.Fatalf("OC-CH-2 FAIL: state corrupted: nonce=%d, spent=%s", ch.Nonce, ch.Spent)
	}

	t.Logf("OC-CH-2 PASS: nonce rollback correctly rejected")
}

// ==================================================================================
// OC-CH-3: Premature Settlement During Active Dispute
//
// ATTACK: Dispute a channel, then immediately try to settle before the
// dispute period expires. If unguarded, the disputed state is ignored.
//
// DEFENSE: SettleChannel checks currentBlock > ch.DisputeDeadline for disputed channels.
// ==================================================================================

func TestOC_CH3_PrematureSettlementDuringDispute(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender:        payer1,
		Receiver:      provider1,
		Deposit:       "5000000",
		TimeoutHeight: 10000,
	})

	// Provider disputes with higher spent amount
	_, err := msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer:               provider1,
		ChannelId:              resp.ChannelId,
		Sequence:               1,
		ClaimedBalanceReceiver: "4000000",
	})
	if err != nil {
		t.Fatalf("dispute: %v", err)
	}

	// ATTACK: Try to settle immediately (dispute period still active)
	_, err = msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1,
		ChannelId: resp.ChannelId,
	})
	if err == nil {
		t.Fatal("OC-CH-3 FAIL: premature settlement during active dispute!")
	}

	// Verify channel still disputed
	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusDisputed {
		t.Fatalf("OC-CH-3 FAIL: channel status changed during dispute: %s", ch.Status)
	}

	t.Logf("OC-CH-3 PASS: premature settlement during dispute correctly rejected")
}

// ==================================================================================
// OC-CH-4: Overspend Attack
//
// ATTACK: Try to set spent amount higher than deposited amount.
// If unguarded, the channel would owe more than it holds.
//
// DEFENSE: UpdateChannel checks spent <= deposited.
// ==================================================================================

func TestOC_CH4_OverspendAttack(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender:        payer1,
		Receiver:      provider1,
		Deposit:       "5000000",
		TimeoutHeight: 10000,
	})

	// ATTACK: Set spent amount higher than deposited
	_, err := msgServer.UpdateChannel(ctx, &types.MsgUpdateChannel{
		Sender:             provider1,
		ChannelId:          resp.ChannelId,
		Sequence:           1,
		NewBalanceReceiver: "9999999999", // way over deposited 5M
	})
	if err == nil {
		t.Fatal("OC-CH-4 FAIL: overspend allowed! Channel would owe more than it holds.")
	}

	// Verify channel state unchanged
	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Spent != "0" {
		t.Fatalf("OC-CH-4 FAIL: channel state corrupted by overspend: spent=%s", ch.Spent)
	}

	t.Logf("OC-CH-4 PASS: overspend correctly rejected")
}
