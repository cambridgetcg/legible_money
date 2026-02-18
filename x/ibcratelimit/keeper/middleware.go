package keeper

import (
	"encoding/json"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcporttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/legible-money/legible/x/ibcratelimit/types"
)

// Compile-time interface assertions.
var (
	_ ibcporttypes.IBCModule   = IBCMiddleware{}
	_ ibcporttypes.ICS4Wrapper = ICS4Wrapper{}
)

// transferPacketData mirrors ibc-go's FungibleTokenPacketData for JSON unmarshalling
// without importing the full transfer module as a dependency in the middleware.
type transferPacketData struct {
	Denom    string `json:"denom"`
	Amount   string `json:"amount"`
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Memo     string `json:"memo,omitempty"`
}

// -------------------------------------------------------------------
// IBCMiddleware wraps an inner IBCModule and enforces rate limits on
// packet receive (inflow) and acknowledgement/timeout (outflow tracking).
// -------------------------------------------------------------------

// IBCMiddleware wraps an IBC module and enforces rate limits.
type IBCMiddleware struct {
	app    ibcporttypes.IBCModule
	keeper Keeper
}

// NewIBCMiddleware creates a new IBCMiddleware.
func NewIBCMiddleware(app ibcporttypes.IBCModule, k Keeper) IBCMiddleware {
	return IBCMiddleware{app: app, keeper: k}
}

// --- Channel lifecycle: delegate to inner module ---

func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	channelCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, version)
}

func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	channelCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, counterpartyVersion)
}

func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

func (im IBCMiddleware) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

func (im IBCMiddleware) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

func (im IBCMiddleware) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// --- Packet callbacks ---

// OnRecvPacket checks inflow rate limits before delegating to the inner module.
// If the inflow for the tracked denom exceeds the epoch limit, an error acknowledgement is returned.
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	params := im.keeper.GetParams(ctx)

	// Parse the transfer packet data
	var data transferPacketData
	if err := json.Unmarshal(packet.GetData(), &data); err != nil {
		// Not a transfer packet or malformed — delegate to inner module unchanged.
		return im.app.OnRecvPacket(ctx, packet, relayer)
	}

	// Parse the amount — all denoms are rate-limited (aggregate per channel)
	amount, ok := new(big.Int).SetString(data.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		return im.app.OnRecvPacket(ctx, packet, relayer)
	}

	// Get (or reset) the flow record for this channel
	flow := im.keeper.GetOrResetFlow(ctx, packet.GetDestChannel())

	// Check inflow limit using effective flow (sliding window)
	maxInflow, ok := new(big.Int).SetString(params.MaxInflowPerEpoch, 10)
	if !ok {
		maxInflow = new(big.Int) // zero — block all if misconfigured
	}

	effectiveInflow := im.keeper.GetEffectiveInflow(ctx, packet.GetDestChannel())

	newInflow := new(big.Int).Add(effectiveInflow, amount)
	if newInflow.Cmp(maxInflow) > 0 {
		// Inflow limit exceeded — return error ack without processing
		return channeltypes.NewErrorAcknowledgement(types.ErrInflowExceeded)
	}

	// Delegate to inner module first
	ack := im.app.OnRecvPacket(ctx, packet, relayer)

	// Only track the inflow if the inner module accepted the packet (successful ack)
	if ack != nil && ack.Success() {
		currentInflow, ok := new(big.Int).SetString(flow.InflowTotal, 10)
		if !ok {
			currentInflow = new(big.Int)
		}
		flow.InflowTotal = new(big.Int).Add(currentInflow, amount).String()
		im.keeper.SetFlow(ctx, flow)
	}

	return ack
}

// OnAcknowledgementPacket tracks confirmed outflow on successful acknowledgement.
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	// Delegate to inner module first
	if err := im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer); err != nil {
		return err
	}

	// Only track outflow on successful ack
	var ack channeltypes.Acknowledgement
	if err := json.Unmarshal(acknowledgement, &ack); err != nil {
		return nil // can't parse ack, nothing to track
	}

	// Check if ack is successful (has Result field, not Error)
	if ack.GetError() != "" {
		return nil // transfer failed on counterparty, no outflow to track
	}

	var data transferPacketData
	if err := json.Unmarshal(packet.GetData(), &data); err != nil {
		return nil
	}

	// All denoms are tracked (aggregate per channel)
	amount, ok := new(big.Int).SetString(data.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		return nil
	}

	flow := im.keeper.GetOrResetFlow(ctx, packet.GetSourceChannel())

	currentOutflow, ok := new(big.Int).SetString(flow.OutflowTotal, 10)
	if !ok {
		currentOutflow = new(big.Int)
	}

	flow.OutflowTotal = new(big.Int).Add(currentOutflow, amount).String()
	im.keeper.SetFlow(ctx, flow)

	return nil
}

// OnTimeoutPacket handles timed-out packets. No outflow was confirmed, so nothing to track.
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	return im.app.OnTimeoutPacket(ctx, packet, relayer)
}

// -------------------------------------------------------------------
// ICS4Wrapper wraps the inner ICS4Wrapper to check outflow limits
// BEFORE a packet is sent.
// -------------------------------------------------------------------

// ICS4Wrapper wraps an inner ICS4Wrapper and enforces outflow rate limits on SendPacket.
type ICS4Wrapper struct {
	channel ibcporttypes.ICS4Wrapper
	keeper  Keeper
}

// NewICS4Wrapper creates a new rate-limiting ICS4Wrapper.
func NewICS4Wrapper(channel ibcporttypes.ICS4Wrapper, k Keeper) ICS4Wrapper {
	return ICS4Wrapper{channel: channel, keeper: k}
}

// SendPacket checks outflow rate limits before delegating to the inner wrapper.
func (w ICS4Wrapper) SendPacket(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	params := w.keeper.GetParams(ctx)

	// Parse the transfer packet data
	var packetData transferPacketData
	if err := json.Unmarshal(data, &packetData); err != nil {
		// Not a transfer packet — pass through
		return w.channel.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	}

	// All denoms are rate-limited (aggregate per channel)
	amount, ok := new(big.Int).SetString(packetData.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		return w.channel.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	}

	// Ensure flow record exists (triggers epoch reset if needed)
	_ = w.keeper.GetOrResetFlow(ctx, sourceChannel)

	// Check outflow limit using effective flow (sliding window)
	maxOutflow, ok := new(big.Int).SetString(params.MaxOutflowPerEpoch, 10)
	if !ok {
		maxOutflow = new(big.Int) // zero — block all if misconfigured
	}

	effectiveOutflow := w.keeper.GetEffectiveOutflow(ctx, sourceChannel)

	newOutflow := new(big.Int).Add(effectiveOutflow, amount)
	if newOutflow.Cmp(maxOutflow) > 0 {
		return 0, types.ErrOutflowExceeded
	}

	// Delegate to inner wrapper — the actual outflow tracking is done in OnAcknowledgementPacket
	// when the counterparty confirms receipt. Here we only do the pre-send limit check.
	return w.channel.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

// WriteAcknowledgement delegates to the inner wrapper.
func (w ICS4Wrapper) WriteAcknowledgement(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
) error {
	return w.channel.WriteAcknowledgement(ctx, chanCap, packet, ack)
}

// GetAppVersion delegates to the inner wrapper.
func (w ICS4Wrapper) GetAppVersion(
	ctx sdk.Context,
	portID,
	channelID string,
) (string, bool) {
	return w.channel.GetAppVersion(ctx, portID, channelID)
}
