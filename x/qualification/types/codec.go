package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the qualification module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRequestQualification{}, "qualification/RequestQualification", nil)
	cdc.RegisterConcrete(&MsgEndorseValidator{}, "qualification/EndorseValidator", nil)
	cdc.RegisterConcrete(&MsgRenewQualification{}, "qualification/RenewQualification", nil)
	cdc.RegisterConcrete(&MsgWithdrawQualification{}, "qualification/WithdrawQualification", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "qualification/UpdateParams", nil)
}

// RegisterInterfaces registers the qualification module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRequestQualification{},
		&MsgEndorseValidator{},
		&MsgRenewQualification{},
		&MsgWithdrawQualification{},
		&MsgUpdateParams{},
	)
}

// Msg gRPC service descriptor, handlers, MsgServer interface, and RegisterMsgServer
// are now proto-generated in tx_grpc.pb.go.
// Query gRPC service descriptors are now proto-generated in query_grpc.pb.go.
