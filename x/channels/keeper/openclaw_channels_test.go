package keeper_test

// ==================================================================================
// OpenClaw Channels Adversarial Probe Suite (OC-CH-1 through OC-CH-12)
//
// Attack surfaces tested:
// - Non-party channel update (third party modifies state)
// - Nonce rollback (submit lower nonce to revert spent amount)
// - Settle during active dispute (premature settlement)
// - Overspend (spent exceeds deposited)
// - Nonce freshness (same-nonce replay)
// - Channel ID determinism and uniqueness
// - SpendFromChannel overspend and non-existent channel
// - Unauthorized close/settle by third party
// - Deposit to closed/disputed channels
// - ClaimExpired before grace period
// - Zero/negative deposit channel creation
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

	msgServer := keeper.NewMsgServerImpl(k)
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

	msgServer := keeper.NewMsgServerImpl(k)
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
		Update: &types.ChannelStateUpdate{
			Nonce:             5,
			Spent:             "3000000",
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 5, "3000000"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 5, "3000000"),
		},
	})
	if err != nil {
		t.Fatalf("first update: %v", err)
	}

	// ATTACK: Rollback to nonce 3 with lower spent (valid sigs but stale nonce)
	_, err = msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender:    payer1,
		ChannelId: resp.ChannelId,
		Update: &types.ChannelStateUpdate{
			Nonce:             3,
			Spent:             "1000000", // trying to undo 2M in payments
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 3, "1000000"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 3, "1000000"),
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

	msgServer := keeper.NewMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender:        payer1,
		Receiver:      provider1,
		Deposit:       "5000000",
		TimeoutHeight: 10000,
	})

	// Provider disputes with higher spent amount (payer1 is counterparty)
	_, err := msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer:               provider1,
		ChannelId:              resp.ChannelId,
		Sequence:               1,
		ClaimedBalanceReceiver: "4000000",
		ProofSignature:         signPacked(payer1Party, "dispute", resp.ChannelId, 1, "4000000"),
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

	msgServer := keeper.NewMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender:        payer1,
		Receiver:      provider1,
		Deposit:       "5000000",
		TimeoutHeight: 10000,
	})

	// ATTACK: Set spent amount higher than deposited (with valid counterparty sig to reach overspend check)
	_, err := msgServer.UpdateChannel(ctx, &types.MsgUpdateChannel{
		Sender:             provider1,
		ChannelId:          resp.ChannelId,
		Sequence:           1,
		NewBalanceReceiver: "9999999999", // way over deposited 5M
		Signature:          signPacked(payer1Party, "update", resp.ChannelId, 1, "9999999999"),
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

// ==================================================================================
// OC-CH-5: Nonce Freshness — Same-Nonce Replay
//
// ATTACK: Submit update with nonce=1. Try nonce=1 again (replay, not rollback).
// Also verify nonce=0 (below initial). Both must be rejected.
//
// DEFENSE: UpdateState checks msg.Update.Nonce > ch.Nonce (strictly greater).
// ==================================================================================

func TestOC_CH5_NonceFreshness(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Update with nonce 1 — success.
	_, err := msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: &types.ChannelStateUpdate{
			Nonce:             1,
			Spent:             "1000000",
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 1, "1000000"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 1, "1000000"),
		},
	})
	if err != nil {
		t.Fatalf("nonce 1 update failed: %v", err)
	}

	// ATTACK: Replay nonce=1 (same nonce, different spent — valid sigs but stale nonce).
	_, err = msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: &types.ChannelStateUpdate{
			Nonce:             1,
			Spent:             "500000",
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 1, "500000"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 1, "500000"),
		},
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: same-nonce replay accepted; " +
			"state can be rewritten without nonce advancement")
	}

	// ATTACK: Nonce=0 (below initial after update — valid sigs but stale nonce).
	_, err = msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: &types.ChannelStateUpdate{
			Nonce:             0,
			Spent:             "0",
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 0, "0"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 0, "0"),
		},
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: nonce=0 accepted after nonce=1; payments can be reversed")
	}

	// Verify nonce=2 succeeds (forward progression).
	_, err = msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: &types.ChannelStateUpdate{
			Nonce:             2,
			Spent:             "2000000",
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 2, "2000000"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 2, "2000000"),
		},
	})
	if err != nil {
		t.Errorf("nonce 2 should succeed: %v", err)
	}

	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Nonce != 2 || ch.Spent != "2000000" {
		t.Errorf("expected nonce=2/spent=2000000, got nonce=%d/spent=%s", ch.Nonce, ch.Spent)
	}
}

// ==================================================================================
// OC-CH-6: Channel ID Determinism
//
// DEFENSE: Open 3 channels in the same block. Verify IDs follow the
// pc-{blockHeight}-{counter} pattern and are all unique.
// ==================================================================================

func TestOC_CH6_ChannelIdDeterminism(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 100000000)

	msgServer := keeper.NewMsgServerImpl(k)
	ids := make(map[string]bool)

	for i := 0; i < 3; i++ {
		resp, err := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
			Sender:        payer1,
			Receiver:      provider1,
			Deposit:       "1000000",
			TimeoutHeight: 10000,
		})
		if err != nil {
			t.Fatalf("open channel %d: %v", i+1, err)
		}
		if ids[resp.ChannelId] {
			t.Errorf("COLLISION: channel ID %s already exists; IDs are not unique within a block", resp.ChannelId)
		}
		ids[resp.ChannelId] = true

		// Verify ID follows pattern.
		expectedPrefix := "pc-100-" // block height 100
		if len(resp.ChannelId) < len(expectedPrefix) || resp.ChannelId[:len(expectedPrefix)] != expectedPrefix {
			t.Errorf("channel ID %s does not match expected prefix %s", resp.ChannelId, expectedPrefix)
		}
	}

	if len(ids) != 3 {
		t.Errorf("expected 3 unique IDs, got %d", len(ids))
	}

	_ = k // suppress unused
}

// ==================================================================================
// OC-CH-7: Overspend via SpendFromChannel
//
// ATTACK: Open channel with 1000 deposit. Spend 1001 via SpendFromChannel.
// Must be rejected. Then spend exactly 1000 — must succeed. Then spend 1
// more — must fail (balance 0).
// ==================================================================================

func TestOC_CH7_OverspendChannel(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.moduleBalances["channels"] = map[string]int64{"ulgm": 100000}

	ch := types.PaymentChannel{
		ChannelId:      "pc-overspend",
		Payer:          payer1,
		Provider:       provider1,
		Deposited:      "1000",
		Spent:          "0",
		Available:      "1000",
		Status:         types.ChannelStatusOpen,
		OpenedAtBlock:  100,
		ExpiresAtBlock: 200,
	}
	k.SetChannel(ctx, &ch)

	// ATTACK: Overspend.
	err := k.SpendFromChannel(ctx, "pc-overspend", "1001", "tree")
	if err == nil {
		t.Error("ATTACK SUCCEEDED: overspend via SpendFromChannel accepted; " +
			"channel would owe more than deposited")
	}

	// Exact spend should succeed.
	err = k.SpendFromChannel(ctx, "pc-overspend", "1000", "tree")
	if err != nil {
		t.Fatalf("exact spend should succeed: %v", err)
	}

	// Spend 1 more — should fail (balance now 0).
	err = k.SpendFromChannel(ctx, "pc-overspend", "1", "tree")
	if err == nil {
		t.Error("ATTACK SUCCEEDED: spend from drained channel accepted; balance should be 0")
	}

	// Verify final state.
	updated, _ := k.GetChannel(ctx, "pc-overspend")
	if updated.Available != "0" {
		t.Errorf("expected available 0, got %s", updated.Available)
	}
}

// ==================================================================================
// OC-CH-8: Unauthorized Close
//
// ATTACK: Open channel between payer A and provider B. Third party C
// tries to close it. Must be rejected.
// ==================================================================================

func TestOC_CH8_UnauthorizedClose(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)
	bk.moduleBalances["channels"] = map[string]int64{"ulgm": 5000000}

	msgServer := keeper.NewMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// ATTACK: Third party tries to close the channel.
	attacker := testAddr("attacker2")
	_, err := msgServer.CloseChannel(ctx, &types.MsgCloseChannel{
		Closer:                attacker,
		ChannelId:             resp.ChannelId,
		Sequence:              1,
		FinalBalanceReceiver:  "5000000", // drain to attacker's benefit
		FinalBalanceSender:    "0",
		CounterpartySignature: []byte("fake_sig"),
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: third party closed the channel; " +
			"any address can close channels and redirect fund distribution")
	}

	// Verify channel still open.
	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusOpen {
		t.Errorf("expected open status, got %s", ch.Status)
	}

	// Also try third-party settle.
	_, err = msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: attacker,
		ChannelId: resp.ChannelId,
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: third party settled the channel")
	}

	_ = k
}

// ==================================================================================
// OC-CH-9: Deposit to Non-Open Channel
//
// ATTACK: Close a channel, then try to deposit more funds into it.
// Must be rejected since closed channels should not accept deposits.
// ==================================================================================

func TestOC_CH9_DepositToClosedChannel(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)
	bk.moduleBalances["channels"] = map[string]int64{"ulgm": 5000000}

	msgServer := keeper.NewMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Close the channel cooperatively (provider1 is counterparty).
	_, err := msgServer.CloseChannel(ctx, &types.MsgCloseChannel{
		Closer:                payer1,
		ChannelId:             resp.ChannelId,
		Sequence:              1,
		FinalBalanceReceiver:  "0",
		FinalBalanceSender:    "5000000",
		CounterpartySignature: signPacked(provider1Party, "close", resp.ChannelId, 1, "0"),
	})
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}

	// ATTACK: Deposit to closed channel.
	_, err = msgServer.DepositChannel(ctx, &types.MsgDepositChannel{
		Depositor: payer1,
		ChannelId: resp.ChannelId,
		Amount:    "1000000",
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: deposit accepted on closed channel; " +
			"funds would be locked with no way to recover")
	}

	// Also test deposit to disputed channel.
	ch2 := types.PaymentChannel{
		ChannelId: "pc-disputed", Payer: payer1, Provider: provider1,
		Deposited: "5000000", Spent: "0", Available: "5000000",
		Status: types.ChannelStatusDisputed,
	}
	k.SetChannel(ctx, &ch2)

	_, err = msgServer.DepositChannel(ctx, &types.MsgDepositChannel{
		Depositor: payer1,
		ChannelId: "pc-disputed",
		Amount:    "1000000",
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: deposit accepted on disputed channel")
	}
}

// ==================================================================================
// OC-CH-10: ClaimExpired Before Expiry
//
// ATTACK: Open a channel with long timeout. Immediately try to ClaimExpired.
// Must be rejected (channel not yet expired).
// ==================================================================================

func TestOC_CH10_ClaimExpiredBeforeExpiry(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// ATTACK: Try to claim before expiry.
	_, err := msgServer.ClaimExpired(ctx, &types.MsgClaimExpired{
		Claimer:   payer1,
		ChannelId: resp.ChannelId,
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: ClaimExpired accepted on non-expired channel; " +
			"payer can prematurely reclaim funds, denying provider earned payments")
	}

	// Advance to 1 block before expiry + grace — should still fail.
	// ExpiresAtBlock = 100 + 10000 = 10100. GracePeriod = 500. Threshold = 10600.
	ctx = ctx.WithBlockHeight(10599)
	_, err = msgServer.ClaimExpired(ctx, &types.MsgClaimExpired{
		Claimer:   payer1,
		ChannelId: resp.ChannelId,
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: ClaimExpired accepted 1 block before grace period ends")
	}

	// At threshold — should succeed.
	bk.moduleBalances["channels"] = map[string]int64{"ulgm": 5000000}
	ctx = ctx.WithBlockHeight(10601)
	_, err = msgServer.ClaimExpired(ctx, &types.MsgClaimExpired{
		Claimer:   payer1,
		ChannelId: resp.ChannelId,
	})
	if err != nil {
		t.Errorf("ClaimExpired after grace period should succeed: %v", err)
	}

	_ = k
}

// ==================================================================================
// OC-CH-11: Zero Deposit Channel
//
// ATTACK: Open a channel with deposit "0" or negative amount.
// Must be rejected by both ValidateBasic and handler.
// ==================================================================================

func TestOC_CH11_ZeroDepositChannel(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewMsgServerImpl(k)

	// Zero deposit via ValidateBasic.
	msg := &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "0", TimeoutHeight: 10000,
	}
	if err := msg.ValidateBasic(); err == nil {
		t.Error("OC-CH-11 FAIL: ValidateBasic accepted zero deposit; " +
			"allows gas-griefing via no-op channel creation")
	}

	// Zero deposit via handler (if ValidateBasic bypassed).
	_, err := msgServer.OpenChannel(ctx, msg)
	if err == nil {
		t.Error("OC-CH-11 FAIL: handler accepted zero deposit channel")
	}

	// Empty deposit string.
	_, err = msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "", TimeoutHeight: 10000,
	})
	if err == nil {
		t.Error("OC-CH-11 FAIL: handler accepted empty deposit channel")
	}

	// Negative deposit.
	_, err = msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "-1000000", TimeoutHeight: 10000,
	})
	if err == nil {
		t.Error("OC-CH-11 FAIL: handler accepted negative deposit channel")
	}

	_ = k
}

// ==================================================================================
// OC-CH-12: SpendFromChannel on Non-Existent Channel
//
// ATTACK: Call SpendFromChannel with a channel ID that doesn't exist.
// Must return error, no panic.
// ==================================================================================

func TestOC_CH12_SpendFromNonExistent(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	err := k.SpendFromChannel(ctx, "pc-nonexistent-999", "1000", "tree")
	if err == nil {
		t.Error("ATTACK SUCCEEDED: SpendFromChannel on non-existent channel succeeded; " +
			"could trigger module-to-module transfer without backing funds")
	}

	// Also try with empty channel ID.
	err = k.SpendFromChannel(ctx, "", "1000", "tree")
	if err == nil {
		t.Error("OC-CH-12 FAIL: SpendFromChannel with empty channel ID succeeded")
	}
}
