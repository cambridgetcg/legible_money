package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterInterfaces registers the oracle module's interface types.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitPrediction{},
	)
}

// RegisterLegacyAminoCodec registers amino codec for oracle types.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitPrediction{}, "oracle/SubmitPrediction", nil)
}

var (
	Amino = codec.NewLegacyAmino()
)

func init() {
	RegisterLegacyAminoCodec(Amino)
}
