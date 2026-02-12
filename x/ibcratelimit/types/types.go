package types

import (
	"context"
	"fmt"
)

// ChannelFlow tracks transfer volume per channel per epoch.
type ChannelFlow struct {
	ChannelID       string `json:"channel_id"`
	Denom           string `json:"denom"`             // tracked denom (e.g., "ulgm")
	OutflowTotal    string `json:"outflow_total"`     // cumulative outflow this epoch (string for math.Int compat)
	InflowTotal     string `json:"inflow_total"`      // cumulative inflow this epoch
	EpochStartBlock uint64 `json:"epoch_start_block"` // block height at which the current epoch started
}

// Params configurable rate limit parameters.
type Params struct {
	EpochLengthBlocks  uint64 `json:"epoch_length_blocks"`   // default: 34272 (~1 day at 2521ms blocks)
	MaxOutflowPerEpoch string `json:"max_outflow_per_epoch"` // default: "10000000000" (10,000 LGM in ulgm)
	MaxInflowPerEpoch  string `json:"max_inflow_per_epoch"`  // default: "10000000000"
	TrackedDenom       string `json:"tracked_denom"`         // default: "ulgm"
}

// DefaultParams returns the default rate limit parameters.
func DefaultParams() Params {
	return Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "10000000000", // 10,000 LGM
		MaxInflowPerEpoch:  "10000000000",
		TrackedDenom:       "ulgm",
	}
}

// Validate performs basic validation of params.
func (p Params) Validate() error {
	if p.EpochLengthBlocks == 0 {
		return fmt.Errorf("epoch length blocks must be positive")
	}
	if p.TrackedDenom == "" {
		return fmt.Errorf("tracked denom cannot be empty")
	}
	return nil
}

// GenesisState is the genesis state for the ibcratelimit module.
type GenesisState struct {
	Params Params        `json:"params"`
	Flows  []ChannelFlow `json:"flows"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() GenesisState {
	return GenesisState{
		Params: DefaultParams(),
		Flows:  []ChannelFlow{},
	}
}

// Validate performs genesis state validation.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}

// --- Message Types ---

// MsgUpdateParams updates the rate limit module parameters.
type MsgUpdateParams struct {
	Authority string `json:"authority"` // governance authority
	Params    Params `json:"params"`
}

func (m *MsgUpdateParams) Reset()         { *m = MsgUpdateParams{} }
func (m *MsgUpdateParams) String() string { return fmt.Sprintf("MsgUpdateParams{%+v}", m.Params) }
func (m *MsgUpdateParams) ProtoMessage()  {}

func (m *MsgUpdateParams) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	return m.Params.Validate()
}

// MsgUpdateParamsResponse is the response to MsgUpdateParams.
type MsgUpdateParamsResponse struct{}

func (m *MsgUpdateParamsResponse) Reset()         { *m = MsgUpdateParamsResponse{} }
func (m *MsgUpdateParamsResponse) String() string { return "MsgUpdateParamsResponse{}" }
func (m *MsgUpdateParamsResponse) ProtoMessage()  {}

// --- Service Interfaces ---

// MsgServer defines the ibcratelimit message handler interface.
type MsgServer interface {
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
}

// QueryServer defines the ibcratelimit query interface.
type QueryServer interface {
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	ChannelFlow(context.Context, *QueryChannelFlowRequest) (*QueryChannelFlowResponse, error)
}

// --- Query Types ---

// QueryParamsRequest is the request for the Params query.
type QueryParamsRequest struct{}

func (m *QueryParamsRequest) Reset()         { *m = QueryParamsRequest{} }
func (m *QueryParamsRequest) String() string { return "QueryParamsRequest{}" }
func (m *QueryParamsRequest) ProtoMessage()  {}

// QueryParamsResponse is the response for the Params query.
type QueryParamsResponse struct {
	Params Params `json:"params"`
}

func (m *QueryParamsResponse) Reset()         { *m = QueryParamsResponse{} }
func (m *QueryParamsResponse) String() string { return fmt.Sprintf("QueryParamsResponse{%+v}", m.Params) }
func (m *QueryParamsResponse) ProtoMessage()  {}

// QueryChannelFlowRequest is the request for the ChannelFlow query.
type QueryChannelFlowRequest struct {
	ChannelID string `json:"channel_id"`
}

func (m *QueryChannelFlowRequest) Reset()         { *m = QueryChannelFlowRequest{} }
func (m *QueryChannelFlowRequest) String() string { return fmt.Sprintf("QueryChannelFlowRequest{%s}", m.ChannelID) }
func (m *QueryChannelFlowRequest) ProtoMessage()  {}

// QueryChannelFlowResponse is the response for the ChannelFlow query.
type QueryChannelFlowResponse struct {
	Flow  *ChannelFlow `json:"flow"`
	Found bool         `json:"found"`
}

func (m *QueryChannelFlowResponse) Reset()         { *m = QueryChannelFlowResponse{} }
func (m *QueryChannelFlowResponse) String() string { return fmt.Sprintf("QueryChannelFlowResponse{found:%v}", m.Found) }
func (m *QueryChannelFlowResponse) ProtoMessage()  {}
