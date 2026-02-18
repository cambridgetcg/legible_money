package types

import (
	"context"
	"fmt"
)

// --- Params Validation ---
// Methods below attach to protobuf-generated types from state.pb.go and genesis.pb.go.

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

// --- Genesis state validation ---

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() GenesisState {
	p := DefaultParams()
	return GenesisState{
		Params: &p,
		Flows:  []*ChannelFlow{},
	}
}

// Validate performs genesis state validation.
func (gs GenesisState) Validate() error {
	if gs.Params != nil {
		return gs.Params.Validate()
	}
	return nil
}

// --- Message Types (not in proto-generated files) ---

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

// Query types (QueryServer, QueryParamsRequest/Response, QueryChannelFlowRequest/Response)
// are now provided by the proto-generated query.pb.go and query_grpc.pb.go files.
