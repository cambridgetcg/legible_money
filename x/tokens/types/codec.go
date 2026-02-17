package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgTransferFrom{}, "tokens/TransferFrom", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "tokens/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateToken{},
		&MsgMintToken{},
		&MsgBurnToken{},
		&MsgTransferToken{},
		&MsgApproveToken{},
		&MsgTransferFrom{},
		&MsgPauseToken{},
		&MsgUnpauseToken{},
		&MsgDelegatePower{},
		&MsgWrapToken{},
		&MsgUnwrapToken{},
		&MsgAddLiquidity{},
		&MsgRemoveLiquidity{},
		&MsgSwap{},
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
