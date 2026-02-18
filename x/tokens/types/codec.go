package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateToken{}, "tokens/CreateToken", nil)
	cdc.RegisterConcrete(&MsgMintToken{}, "tokens/MintToken", nil)
	cdc.RegisterConcrete(&MsgBurnToken{}, "tokens/BurnToken", nil)
	cdc.RegisterConcrete(&MsgTransferToken{}, "tokens/TransferToken", nil)
	cdc.RegisterConcrete(&MsgApproveToken{}, "tokens/ApproveToken", nil)
	cdc.RegisterConcrete(&MsgTransferFrom{}, "tokens/TransferFrom", nil)
	cdc.RegisterConcrete(&MsgPauseToken{}, "tokens/PauseToken", nil)
	cdc.RegisterConcrete(&MsgUnpauseToken{}, "tokens/UnpauseToken", nil)
	cdc.RegisterConcrete(&MsgAddLiquidity{}, "tokens/AddLiquidity", nil)
	cdc.RegisterConcrete(&MsgRemoveLiquidity{}, "tokens/RemoveLiquidity", nil)
	cdc.RegisterConcrete(&MsgSwap{}, "tokens/Swap", nil)
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
