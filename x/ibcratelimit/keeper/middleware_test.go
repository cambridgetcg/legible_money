package keeper_test

import (
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/legible-money/legible/x/ibcratelimit/keeper"
	"github.com/legible-money/legible/x/ibcratelimit/types"
)

// transferPacketData mirrors the middleware's local struct for building test data.
type transferPacketData struct {
	Denom    string `json:"denom"`
	Amount   string `json:"amount"`
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Memo     string `json:"memo,omitempty"`
}

func makeTransferData(denom, amount string) []byte {
	bz, _ := json.Marshal(transferPacketData{
		Denom:    denom,
		Amount:   amount,
		Sender:   "lgm1sender",
		Receiver: "cosmos1receiver",
	})
	return bz
}

// ============================================================
// Mock IBCModule — satisfies ibcporttypes.IBCModule
// ============================================================

type mockIBCModule struct {
	recvAck ibcexported.Acknowledgement // what OnRecvPacket returns
	recvErr error                       // what OnAcknowledgementPacket returns
}

func (m *mockIBCModule) OnChanOpenInit(
	_ sdk.Context, _ channeltypes.Order, _ []string,
	_, _ string, _ *capabilitytypes.Capability,
	_ channeltypes.Counterparty, _ string,
) (string, error) {
	return "", nil
}

func (m *mockIBCModule) OnChanOpenTry(
	_ sdk.Context, _ channeltypes.Order, _ []string,
	_, _ string, _ *capabilitytypes.Capability,
	_ channeltypes.Counterparty, _ string,
) (string, error) {
	return "", nil
}

func (m *mockIBCModule) OnChanOpenAck(_ sdk.Context, _, _, _, _ string) error { return nil }
func (m *mockIBCModule) OnChanOpenConfirm(_ sdk.Context, _, _ string) error {
	return nil
}
func (m *mockIBCModule) OnChanCloseInit(_ sdk.Context, _, _ string) error  { return nil }
func (m *mockIBCModule) OnChanCloseConfirm(_ sdk.Context, _, _ string) error {
	return nil
}

func (m *mockIBCModule) OnRecvPacket(_ sdk.Context, _ channeltypes.Packet, _ sdk.AccAddress) ibcexported.Acknowledgement {
	return m.recvAck
}

func (m *mockIBCModule) OnAcknowledgementPacket(_ sdk.Context, _ channeltypes.Packet, _ []byte, _ sdk.AccAddress) error {
	return m.recvErr
}

func (m *mockIBCModule) OnTimeoutPacket(_ sdk.Context, _ channeltypes.Packet, _ sdk.AccAddress) error {
	return nil
}

// ============================================================
// Mock ICS4Wrapper — satisfies ibcporttypes.ICS4Wrapper
// ============================================================

type mockICS4Wrapper struct {
	seq uint64 // packet sequence returned by SendPacket
}

func (m *mockICS4Wrapper) SendPacket(
	_ sdk.Context, _ *capabilitytypes.Capability,
	_, _ string, _ clienttypes.Height, _ uint64, _ []byte,
) (uint64, error) {
	m.seq++
	return m.seq, nil
}

func (m *mockICS4Wrapper) WriteAcknowledgement(
	_ sdk.Context, _ *capabilitytypes.Capability,
	_ ibcexported.PacketI, _ ibcexported.Acknowledgement,
) error {
	return nil
}

func (m *mockICS4Wrapper) GetAppVersion(_ sdk.Context, _, _ string) (string, bool) {
	return "ics20-1", true
}

// successAck is a simple successful acknowledgement.
type successAck struct{}

func (successAck) Success() bool                { return true }
func (successAck) Acknowledgement() []byte      { return []byte(`{"result":"AQ=="}`) }

// ============================================================
// IBCMiddleware — OnRecvPacket Tests
// ============================================================

func TestMiddleware_OnRecvPacket_WithinLimit(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set small inflow limit
	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "10000000000",
		MaxInflowPerEpoch:  "1000",
		TrackedDenom:       "ulgm",
	})

	inner := &mockIBCModule{recvAck: successAck{}}
	mw := keeper.NewIBCMiddleware(inner, k)

	packet := channeltypes.NewPacket(
		makeTransferData("ulgm", "500"),
		1,
		"transfer", "channel-src",
		"transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000),
		0,
	)

	ack := mw.OnRecvPacket(ctx, packet, nil)
	if ack == nil || !ack.Success() {
		t.Fatal("expected successful ack for inflow within limit")
	}

	// Verify inflow was tracked
	flow, found := k.GetFlow(ctx, "channel-dest")
	if !found {
		t.Fatal("expected flow record to exist")
	}
	if flow.InflowTotal != "500" {
		t.Errorf("expected InflowTotal=500, got %s", flow.InflowTotal)
	}
}

func TestMiddleware_OnRecvPacket_ExceedsLimit(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "10000000000",
		MaxInflowPerEpoch:  "100",
		TrackedDenom:       "ulgm",
	})

	inner := &mockIBCModule{recvAck: successAck{}}
	mw := keeper.NewIBCMiddleware(inner, k)

	packet := channeltypes.NewPacket(
		makeTransferData("ulgm", "200"),
		1,
		"transfer", "channel-src",
		"transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000),
		0,
	)

	ack := mw.OnRecvPacket(ctx, packet, nil)
	if ack == nil {
		t.Fatal("expected non-nil ack")
	}
	if ack.Success() {
		t.Error("expected error ack when inflow exceeds limit")
	}
}

func TestMiddleware_OnRecvPacket_NonTrackedDenom(t *testing.T) {
	k, ctx := setupKeeper(t)

	inner := &mockIBCModule{recvAck: successAck{}}
	mw := keeper.NewIBCMiddleware(inner, k)

	// Use non-tracked denom
	packet := channeltypes.NewPacket(
		makeTransferData("uatom", "99999999"),
		1,
		"transfer", "channel-src",
		"transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000),
		0,
	)

	ack := mw.OnRecvPacket(ctx, packet, nil)
	if ack == nil || !ack.Success() {
		t.Error("expected passthrough for non-tracked denom")
	}
}

func TestMiddleware_OnRecvPacket_NonTransferPacket(t *testing.T) {
	k, ctx := setupKeeper(t)

	inner := &mockIBCModule{recvAck: successAck{}}
	mw := keeper.NewIBCMiddleware(inner, k)

	// Invalid JSON — not a transfer packet
	packet := channeltypes.NewPacket(
		[]byte(`not-json`),
		1,
		"transfer", "channel-src",
		"transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000),
		0,
	)

	ack := mw.OnRecvPacket(ctx, packet, nil)
	if ack == nil || !ack.Success() {
		t.Error("expected passthrough for non-transfer packet")
	}
}

func TestMiddleware_OnRecvPacket_CumulativeInflow(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "10000000000",
		MaxInflowPerEpoch:  "1000",
		TrackedDenom:       "ulgm",
	})

	inner := &mockIBCModule{recvAck: successAck{}}
	mw := keeper.NewIBCMiddleware(inner, k)

	// First packet: 600 inflow — OK
	pkt1 := channeltypes.NewPacket(
		makeTransferData("ulgm", "600"),
		1, "transfer", "channel-src", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack1 := mw.OnRecvPacket(ctx, pkt1, nil)
	if !ack1.Success() {
		t.Fatal("first packet should succeed")
	}

	// Second packet: 500 inflow — total 1100, exceeds 1000 limit
	pkt2 := channeltypes.NewPacket(
		makeTransferData("ulgm", "500"),
		2, "transfer", "channel-src", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack2 := mw.OnRecvPacket(ctx, pkt2, nil)
	if ack2.Success() {
		t.Error("second packet should fail — cumulative inflow exceeds limit")
	}
}

// ============================================================
// ICS4Wrapper — SendPacket Tests
// ============================================================

func TestICS4Wrapper_SendPacket_WithinLimit(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "5000",
		MaxInflowPerEpoch:  "10000000000",
		TrackedDenom:       "ulgm",
	})

	inner := &mockICS4Wrapper{}
	w := keeper.NewICS4Wrapper(inner, k)

	seq, err := w.SendPacket(
		ctx, nil,
		"transfer", "channel-0",
		clienttypes.NewHeight(0, 1000), 0,
		makeTransferData("ulgm", "3000"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if seq == 0 {
		t.Error("expected non-zero packet sequence")
	}
}

func TestICS4Wrapper_SendPacket_ExceedsLimit(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "100",
		MaxInflowPerEpoch:  "10000000000",
		TrackedDenom:       "ulgm",
	})

	inner := &mockICS4Wrapper{}
	w := keeper.NewICS4Wrapper(inner, k)

	_, err := w.SendPacket(
		ctx, nil,
		"transfer", "channel-0",
		clienttypes.NewHeight(0, 1000), 0,
		makeTransferData("ulgm", "200"),
	)
	if err == nil {
		t.Error("expected error when outflow exceeds limit")
	}
}

func TestICS4Wrapper_SendPacket_NonTrackedDenom(t *testing.T) {
	k, ctx := setupKeeper(t)

	inner := &mockICS4Wrapper{}
	w := keeper.NewICS4Wrapper(inner, k)

	_, err := w.SendPacket(
		ctx, nil,
		"transfer", "channel-0",
		clienttypes.NewHeight(0, 1000), 0,
		makeTransferData("uatom", "99999999"),
	)
	if err != nil {
		t.Errorf("expected passthrough for non-tracked denom, got %v", err)
	}
}

func TestICS4Wrapper_SendPacket_NonTransferData(t *testing.T) {
	k, ctx := setupKeeper(t)

	inner := &mockICS4Wrapper{}
	w := keeper.NewICS4Wrapper(inner, k)

	_, err := w.SendPacket(
		ctx, nil,
		"transfer", "channel-0",
		clienttypes.NewHeight(0, 1000), 0,
		[]byte(`not-json`),
	)
	if err != nil {
		t.Errorf("expected passthrough for non-transfer data, got %v", err)
	}
}

// ============================================================
// IBCMiddleware — OnAcknowledgementPacket Tests
// ============================================================

func TestMiddleware_OnAckPacket_SuccessTracksOutflow(t *testing.T) {
	k, ctx := setupKeeper(t)

	inner := &mockIBCModule{}
	mw := keeper.NewIBCMiddleware(inner, k)

	packet := channeltypes.NewPacket(
		makeTransferData("ulgm", "300"),
		1, "transfer", "channel-0", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)

	// Successful ack — use the canonical IBC JSON format
	ackBytes := channeltypes.NewResultAcknowledgement([]byte("ok")).Acknowledgement()

	err := mw.OnAcknowledgementPacket(ctx, packet, ackBytes, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Outflow should be tracked on source channel
	flow, found := k.GetFlow(ctx, "channel-0")
	if !found {
		t.Fatal("expected flow record for source channel")
	}
	if flow.OutflowTotal != "300" {
		t.Errorf("expected OutflowTotal=300, got %s", flow.OutflowTotal)
	}
}

func TestMiddleware_OnAckPacket_InnerModuleError(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Inner module returns an error — middleware returns early, no tracking
	inner := &mockIBCModule{recvErr: types.ErrOutflowExceeded}
	mw := keeper.NewIBCMiddleware(inner, k)

	packet := channeltypes.NewPacket(
		makeTransferData("ulgm", "300"),
		1, "transfer", "channel-0", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)

	ackBytes := channeltypes.NewResultAcknowledgement([]byte("ok")).Acknowledgement()

	err := mw.OnAcknowledgementPacket(ctx, packet, ackBytes, nil)
	if err == nil {
		t.Fatal("expected error from inner module")
	}

	// No outflow should be tracked when inner module errors
	_, found := k.GetFlow(ctx, "channel-0")
	if found {
		t.Error("expected no flow record when inner module returns error")
	}
}

func TestMiddleware_OnAckPacket_MalformedAck(t *testing.T) {
	k, ctx := setupKeeper(t)

	inner := &mockIBCModule{}
	mw := keeper.NewIBCMiddleware(inner, k)

	packet := channeltypes.NewPacket(
		makeTransferData("ulgm", "300"),
		1, "transfer", "channel-0", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)

	// Malformed ack — can't be parsed, no outflow tracked
	err := mw.OnAcknowledgementPacket(ctx, packet, []byte(`not-json`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No outflow should be tracked for unparseable ack
	_, found := k.GetFlow(ctx, "channel-0")
	if found {
		t.Error("expected no flow record when ack is malformed")
	}
}

// ============================================================
// IBCMiddleware — Channel Lifecycle Delegation Tests
// ============================================================

func TestMiddleware_OnChanOpenInit_Delegates(t *testing.T) {
	k, ctx := setupKeeper(t)

	inner := &mockIBCModule{}
	mw := keeper.NewIBCMiddleware(inner, k)

	_, err := mw.OnChanOpenInit(
		ctx, channeltypes.UNORDERED, []string{"connection-0"},
		"transfer", "channel-0", nil,
		channeltypes.NewCounterparty("transfer", "channel-1"),
		"ics20-1",
	)
	if err != nil {
		t.Errorf("expected delegation to succeed, got %v", err)
	}
}

func TestMiddleware_OnTimeoutPacket_Delegates(t *testing.T) {
	k, ctx := setupKeeper(t)

	inner := &mockIBCModule{}
	mw := keeper.NewIBCMiddleware(inner, k)

	packet := channeltypes.NewPacket(
		makeTransferData("ulgm", "100"),
		1, "transfer", "channel-0", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)

	err := mw.OnTimeoutPacket(ctx, packet, nil)
	if err != nil {
		t.Errorf("expected timeout delegation to succeed, got %v", err)
	}

	// No outflow should be tracked on timeout
	_, found := k.GetFlow(ctx, "channel-0")
	if found {
		t.Error("expected no flow record on timeout")
	}
}

// ============================================================
// P0-2 Tests: Multi-Denom Rate Limiting
// ============================================================

func TestMiddleware_OnRecvPacket_NonTrackedDenom_NowLimited(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "10000000000",
		MaxInflowPerEpoch:  "100", // very low limit
		TrackedDenom:       "ulgm",
	})

	inner := &mockIBCModule{recvAck: successAck{}}
	mw := keeper.NewIBCMiddleware(inner, k)

	// Send a non-ulgm denom — should now be rate-limited (P0-2 fix)
	packet := channeltypes.NewPacket(
		makeTransferData("uatom", "200"),
		1,
		"transfer", "channel-src",
		"transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000),
		0,
	)

	ack := mw.OnRecvPacket(ctx, packet, nil)
	if ack == nil {
		t.Fatal("expected non-nil ack")
	}
	if ack.Success() {
		t.Error("expected non-ulgm denom to be rate-limited (exceeds 100 limit with 200 amount)")
	}
}

func TestICS4Wrapper_SendPacket_NonTrackedDenom_NowLimited(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "100", // very low limit
		MaxInflowPerEpoch:  "10000000000",
		TrackedDenom:       "ulgm",
	})

	inner := &mockICS4Wrapper{}
	w := keeper.NewICS4Wrapper(inner, k)

	// Non-ulgm denom outflow should now be rate-limited (P0-2 fix)
	_, err := w.SendPacket(
		ctx, nil,
		"transfer", "channel-0",
		clienttypes.NewHeight(0, 1000), 0,
		makeTransferData("uatom", "200"),
	)
	if err == nil {
		t.Error("expected non-ulgm denom outflow to be rate-limited")
	}
}

// ============================================================
// P0-3 Tests: Sliding Window Epoch Reset
// ============================================================

func TestSlidingWindow_EpochBoundary_NoDoubleBurst(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Short epoch for testing (10 blocks)
	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  10,
		MaxOutflowPerEpoch: "1000",
		MaxInflowPerEpoch:  "1000",
		TrackedDenom:       "ulgm",
	})

	inner := &mockIBCModule{recvAck: successAck{}}
	mw := keeper.NewIBCMiddleware(inner, k)

	// Phase 1: Use 900/1000 of inflow limit during first epoch (block 100, epoch starts at 100)
	pkt1 := channeltypes.NewPacket(
		makeTransferData("ulgm", "900"),
		1, "transfer", "channel-src", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack1 := mw.OnRecvPacket(ctx, pkt1, nil)
	if !ack1.Success() {
		t.Fatal("first packet should succeed (900 of 1000)")
	}

	// Phase 2: Advance to just past epoch boundary (block 111, epoch started at 100, length 10)
	// The previous epoch's 900 should carry over partially
	ctx = ctx.WithBlockHeight(111)

	// At block 111, we are 1 block into the new epoch (111-110=1).
	// Carry-over = 900 * (10-1)/10 = 810
	// So effective inflow = 0 (current) + 810 (carry) = 810
	// A 200 transfer would push to 1010 > 1000 limit

	pkt2 := channeltypes.NewPacket(
		makeTransferData("ulgm", "200"),
		2, "transfer", "channel-src", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack2 := mw.OnRecvPacket(ctx, pkt2, nil)
	if ack2.Success() {
		t.Error("expected rejection: sliding window carry-over (810) + new (200) = 1010 > 1000 limit")
	}
}

func TestSlidingWindow_DecayOverTime(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  10,
		MaxOutflowPerEpoch: "1000",
		MaxInflowPerEpoch:  "1000",
		TrackedDenom:       "ulgm",
	})

	inner := &mockIBCModule{recvAck: successAck{}}
	mw := keeper.NewIBCMiddleware(inner, k)

	// Fill up to 900 in first epoch
	pkt1 := channeltypes.NewPacket(
		makeTransferData("ulgm", "900"),
		1, "transfer", "channel-src", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack1 := mw.OnRecvPacket(ctx, pkt1, nil)
	if !ack1.Success() {
		t.Fatal("first packet should succeed")
	}

	// Advance to 5 blocks into the new epoch
	// Carry-over = 900 * (10-5)/10 = 450
	ctx = ctx.WithBlockHeight(115)

	pkt2 := channeltypes.NewPacket(
		makeTransferData("ulgm", "500"),
		2, "transfer", "channel-src", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack2 := mw.OnRecvPacket(ctx, pkt2, nil)
	if !ack2.Success() {
		t.Error("should succeed: carry-over (450) + new (500) = 950 < 1000")
	}
}

func TestSlidingWindow_FullEpochLater_NoCarryOver(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  10,
		MaxOutflowPerEpoch: "1000",
		MaxInflowPerEpoch:  "1000",
		TrackedDenom:       "ulgm",
	})

	inner := &mockIBCModule{recvAck: successAck{}}
	mw := keeper.NewIBCMiddleware(inner, k)

	// Fill up to 900 in first epoch
	pkt1 := channeltypes.NewPacket(
		makeTransferData("ulgm", "900"),
		1, "transfer", "channel-src", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack1 := mw.OnRecvPacket(ctx, pkt1, nil)
	if !ack1.Success() {
		t.Fatal("first packet should succeed")
	}

	// Advance to 2+ full epochs later — no carry-over
	ctx = ctx.WithBlockHeight(130)

	pkt2 := channeltypes.NewPacket(
		makeTransferData("ulgm", "999"),
		2, "transfer", "channel-src", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack2 := mw.OnRecvPacket(ctx, pkt2, nil)
	if !ack2.Success() {
		t.Error("should succeed: no carry-over after 2+ full epochs, 999 < 1000")
	}
}

// ============================================================
// Multi-Denom Rate Limiting Tests
// ============================================================

// TestMultiDenom_AggregateLimit verifies that transfers of different denoms
// aggregate against the same per-channel limit.
func TestMultiDenom_AggregateLimit(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "10000000000",
		MaxInflowPerEpoch:  "1000",
		TrackedDenom:       "ulgm",
	})

	inner := &mockIBCModule{recvAck: successAck{}}
	mw := keeper.NewIBCMiddleware(inner, k)

	// 500 of denom A (ulgm)
	pkt1 := channeltypes.NewPacket(
		makeTransferData("ulgm", "500"),
		1, "transfer", "channel-src", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack1 := mw.OnRecvPacket(ctx, pkt1, nil)
	if !ack1.Success() {
		t.Fatal("500 of denom A should succeed")
	}

	// 500 of denom B (uatom) — aggregate total = 1000
	pkt2 := channeltypes.NewPacket(
		makeTransferData("uatom", "500"),
		2, "transfer", "channel-src", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack2 := mw.OnRecvPacket(ctx, pkt2, nil)
	if !ack2.Success() {
		t.Fatal("500 of denom B should succeed (total 1000 = limit)")
	}

	// Verify aggregate flow
	flow, found := k.GetFlow(ctx, "channel-dest")
	if !found {
		t.Fatal("expected flow record")
	}
	if flow.InflowTotal != "1000" {
		t.Errorf("expected aggregate inflow=1000, got %s", flow.InflowTotal)
	}

	// One more of any denom should fail
	pkt3 := channeltypes.NewPacket(
		makeTransferData("ibc/ABC123", "1"),
		3, "transfer", "channel-src", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack3 := mw.OnRecvPacket(ctx, pkt3, nil)
	if ack3.Success() {
		t.Error("should reject: aggregate 1001 > 1000 limit")
	}
}

// TestMultiDenom_DifferentChannels verifies flow for one channel doesn't count
// against another channel's limits.
func TestMultiDenom_DifferentChannels(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "10000000000",
		MaxInflowPerEpoch:  "1000",
		TrackedDenom:       "ulgm",
	})

	inner := &mockIBCModule{recvAck: successAck{}}
	mw := keeper.NewIBCMiddleware(inner, k)

	// 800 on channel-0
	pkt1 := channeltypes.NewPacket(
		makeTransferData("ulgm", "800"),
		1, "transfer", "channel-src", "transfer", "channel-0",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack1 := mw.OnRecvPacket(ctx, pkt1, nil)
	if !ack1.Success() {
		t.Fatal("800 on channel-0 should succeed")
	}

	// 800 on channel-1 — independent channel, separate flow
	pkt2 := channeltypes.NewPacket(
		makeTransferData("ulgm", "800"),
		2, "transfer", "channel-src", "transfer", "channel-1",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack2 := mw.OnRecvPacket(ctx, pkt2, nil)
	if !ack2.Success() {
		t.Fatal("800 on channel-1 should succeed (independent channel)")
	}

	// Each channel has its own flow
	flow0, _ := k.GetFlow(ctx, "channel-0")
	flow1, _ := k.GetFlow(ctx, "channel-1")
	if flow0.InflowTotal != "800" {
		t.Errorf("channel-0: expected inflow=800, got %s", flow0.InflowTotal)
	}
	if flow1.InflowTotal != "800" {
		t.Errorf("channel-1: expected inflow=800, got %s", flow1.InflowTotal)
	}

	// 300 more on channel-0 exceeds its limit (800+300=1100 > 1000)
	pkt3 := channeltypes.NewPacket(
		makeTransferData("ulgm", "300"),
		3, "transfer", "channel-src", "transfer", "channel-0",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack3 := mw.OnRecvPacket(ctx, pkt3, nil)
	if ack3.Success() {
		t.Error("channel-0: should reject 800+300=1100 > 1000")
	}

	// But 200 on channel-1 is still fine (800+200=1000)
	pkt4 := channeltypes.NewPacket(
		makeTransferData("ulgm", "200"),
		4, "transfer", "channel-src", "transfer", "channel-1",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack4 := mw.OnRecvPacket(ctx, pkt4, nil)
	if !ack4.Success() {
		t.Error("channel-1: should succeed 800+200=1000 = limit")
	}
}

// TestMultiDenom_NativeToken verifies that LGM (ulgm) transfers are correctly
// rate-limited.
func TestMultiDenom_NativeToken(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "10000000000",
		MaxInflowPerEpoch:  "500",
		TrackedDenom:       "ulgm",
	})

	inner := &mockIBCModule{recvAck: successAck{}}
	mw := keeper.NewIBCMiddleware(inner, k)

	// Under limit → OK
	pkt1 := channeltypes.NewPacket(
		makeTransferData("ulgm", "400"),
		1, "transfer", "channel-src", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack1 := mw.OnRecvPacket(ctx, pkt1, nil)
	if !ack1.Success() {
		t.Fatal("400 ulgm should succeed (limit 500)")
	}

	// Over limit → rejected
	pkt2 := channeltypes.NewPacket(
		makeTransferData("ulgm", "200"),
		2, "transfer", "channel-src", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack2 := mw.OnRecvPacket(ctx, pkt2, nil)
	if ack2.Success() {
		t.Error("400+200=600 > 500 limit: should reject")
	}
}

// TestMultiDenom_IBCDenom verifies that IBC denominations (ibc/hash...) are
// correctly rate-limited.
func TestMultiDenom_IBCDenom(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "10000000000",
		MaxInflowPerEpoch:  "500",
		TrackedDenom:       "ulgm",
	})

	inner := &mockIBCModule{recvAck: successAck{}}
	mw := keeper.NewIBCMiddleware(inner, k)

	ibcDenom := "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"

	// Under limit → OK
	pkt1 := channeltypes.NewPacket(
		makeTransferData(ibcDenom, "400"),
		1, "transfer", "channel-src", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack1 := mw.OnRecvPacket(ctx, pkt1, nil)
	if !ack1.Success() {
		t.Fatal("400 ibc/... should succeed (limit 500)")
	}

	// Over limit → rejected
	pkt2 := channeltypes.NewPacket(
		makeTransferData(ibcDenom, "200"),
		2, "transfer", "channel-src", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ack2 := mw.OnRecvPacket(ctx, pkt2, nil)
	if ack2.Success() {
		t.Error("400+200=600 > 500 limit: IBC denoms should be rate-limited")
	}

	// Verify flow tracked
	flow, found := k.GetFlow(ctx, "channel-dest")
	if !found {
		t.Fatal("expected flow record")
	}
	if flow.InflowTotal != "400" {
		t.Errorf("expected inflow=400 (first successful), got %s", flow.InflowTotal)
	}
}

// ============================================================
// ICA Rate Limit Integration Tests
// ============================================================

// TestICA_TransferThroughRateLimit verifies that an ICA-initiated transfer is
// counted against rate limits (same middleware path as direct transfers).
func TestICA_TransferThroughRateLimit(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "1000",
		MaxInflowPerEpoch:  "10000000000",
		TrackedDenom:       "ulgm",
	})

	innerW := &mockICS4Wrapper{}
	w := keeper.NewICS4Wrapper(innerW, k)

	// ICA-initiated transfer (goes through same SendPacket path)
	seq, err := w.SendPacket(
		ctx, nil,
		"transfer", "channel-0",
		clienttypes.NewHeight(0, 1000), 0,
		makeTransferData("ulgm", "500"),
	)
	if err != nil {
		t.Fatalf("ICA transfer within limit should succeed: %v", err)
	}
	if seq == 0 {
		t.Error("expected non-zero sequence")
	}

	// Track outflow via ack (outflow tracked on acknowledgement)
	innerMod := &mockIBCModule{}
	mw := keeper.NewIBCMiddleware(innerMod, k)

	packet := channeltypes.NewPacket(
		makeTransferData("ulgm", "500"),
		seq, "transfer", "channel-0", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ackBytes := channeltypes.NewResultAcknowledgement([]byte("ok")).Acknowledgement()
	err = mw.OnAcknowledgementPacket(ctx, packet, ackBytes, nil)
	if err != nil {
		t.Fatalf("ack should succeed: %v", err)
	}

	// Verify outflow tracked
	flow, found := k.GetFlow(ctx, "channel-0")
	if !found {
		t.Fatal("expected flow record")
	}
	if flow.OutflowTotal != "500" {
		t.Errorf("expected outflow=500, got %s", flow.OutflowTotal)
	}
}

// TestICA_ExceedsLimit_Rejected verifies that an ICA transfer exceeding the
// rate limit is rejected at the SendPacket level.
func TestICA_ExceedsLimit_Rejected(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "500",
		MaxInflowPerEpoch:  "10000000000",
		TrackedDenom:       "ulgm",
	})

	innerW := &mockICS4Wrapper{}
	w := keeper.NewICS4Wrapper(innerW, k)

	// ICA-initiated transfer exceeding limit
	_, err := w.SendPacket(
		ctx, nil,
		"transfer", "channel-0",
		clienttypes.NewHeight(0, 1000), 0,
		makeTransferData("ulgm", "600"),
	)
	if err == nil {
		t.Error("ICA transfer exceeding limit should be rejected")
	}
}

// TestICA_AndDirectTransfer_SharedLimit verifies that direct transfers and
// ICA transfers share the same per-channel rate limit pool.
func TestICA_AndDirectTransfer_SharedLimit(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "1000",
		MaxInflowPerEpoch:  "10000000000",
		TrackedDenom:       "ulgm",
	})

	innerW := &mockICS4Wrapper{}
	w := keeper.NewICS4Wrapper(innerW, k)
	innerMod := &mockIBCModule{}
	mw := keeper.NewIBCMiddleware(innerMod, k)

	// Direct transfer: send 800 (80% of 1000 limit)
	_, err := w.SendPacket(
		ctx, nil,
		"transfer", "channel-0",
		clienttypes.NewHeight(0, 1000), 0,
		makeTransferData("ulgm", "800"),
	)
	if err != nil {
		t.Fatalf("direct transfer (800) should succeed: %v", err)
	}

	// Track outflow via ack
	packet := channeltypes.NewPacket(
		makeTransferData("ulgm", "800"),
		1, "transfer", "channel-0", "transfer", "channel-dest",
		clienttypes.NewHeight(0, 1000), 0,
	)
	ackBytes := channeltypes.NewResultAcknowledgement([]byte("ok")).Acknowledgement()
	err = mw.OnAcknowledgementPacket(ctx, packet, ackBytes, nil)
	if err != nil {
		t.Fatalf("ack should succeed: %v", err)
	}

	// ICA transfer: try 300 (30%) → rejected (800+300=1100 > 1000)
	_, err = w.SendPacket(
		ctx, nil,
		"transfer", "channel-0",
		clienttypes.NewHeight(0, 1000), 0,
		makeTransferData("ulgm", "300"),
	)
	if err == nil {
		t.Error("ICA transfer should be rejected: 800+300=1100 > 1000 shared limit")
	}

	// But 200 fits (800+200=1000)
	_, err = w.SendPacket(
		ctx, nil,
		"transfer", "channel-0",
		clienttypes.NewHeight(0, 1000), 0,
		makeTransferData("ulgm", "200"),
	)
	if err != nil {
		t.Errorf("200 should succeed (800+200=1000): %v", err)
	}
}
