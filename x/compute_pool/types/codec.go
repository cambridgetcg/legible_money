package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the module's types on the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterComputeProvider{}, "compute_pool/RegisterComputeProvider", nil)
	cdc.RegisterConcrete(&MsgUpdateProvider{}, "compute_pool/UpdateProvider", nil)
	cdc.RegisterConcrete(&MsgDeregisterProvider{}, "compute_pool/DeregisterProvider", nil)
	cdc.RegisterConcrete(&MsgProviderHeartbeat{}, "compute_pool/ProviderHeartbeat", nil)
	cdc.RegisterConcrete(&MsgRedeemCredits{}, "compute_pool/RedeemCredits", nil)
	cdc.RegisterConcrete(&MsgConfirmUsage{}, "compute_pool/ConfirmUsage", nil)
	cdc.RegisterConcrete(&MsgDisputeUsage{}, "compute_pool/DisputeUsage", nil)
	cdc.RegisterConcrete(&MsgUpdateComputeParams{}, "compute_pool/UpdateComputeParams", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "compute_pool/UpdateParams", nil)
}

// RegisterInterfaces registers the module's interface types.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
