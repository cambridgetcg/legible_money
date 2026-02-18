package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers concrete types on codec
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterValidator{}, "legible_staking/RegisterValidator", nil)
	cdc.RegisterConcrete(&MsgUpdateValidatorStake{}, "legible_staking/UpdateValidatorStake", nil)
	cdc.RegisterConcrete(&MsgDelegate{}, "legible_staking/Delegate", nil)
	cdc.RegisterConcrete(&MsgUndelegate{}, "legible_staking/Undelegate", nil)
	cdc.RegisterConcrete(&MsgRedelegate{}, "legible_staking/Redelegate", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "legible_staking/UpdateParams", nil)
}

// RegisterInterfaces registers the interfaces types with the interface registry
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterValidator{},
	)
}

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
)

func init() {
	RegisterCodec(amino)
	amino.Seal()
}

// Note: RegisterMsgServer and Msg_ServiceDesc are now defined in tx_grpc.pb.go (proto-generated).
// Note: RegisterQueryServer, QueryServer, and query handlers are now in query_grpc.pb.go (proto-generated).
