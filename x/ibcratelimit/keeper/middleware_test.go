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
	k.SetParams(ctx, types.Params{
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

	k.SetParams(ctx, types.Params{
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

	k.SetParams(ctx, types.Params{
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

	k.SetParams(ctx, types.Params{
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

	k.SetParams(ctx, types.Params{
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
