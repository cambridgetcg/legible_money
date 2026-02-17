package app

import (
	"fmt"

	"google.golang.org/grpc"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/cosmos/cosmos-sdk/server/grpc/gogoreflection"
	reflection "github.com/cosmos/cosmos-sdk/server/grpc/reflection/v2alpha1"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewLegibleGRPCServer creates a gRPC server that gracefully handles reflection
// failures caused by hand-written Msg types without proto descriptors.
//
// This is identical to servergrpc.NewGRPCServer except that reflection.Register
// errors are logged as warnings instead of being fatal. This allows the chain
// to serve gRPC queries (bank, staking, knowledge, etc.) even when some custom
// modules lack proto codegen.
func NewLegibleGRPCServer(
	logger log.Logger,
	clientCtx client.Context,
	app servertypes.Application,
	cfg serverconfig.GRPCConfig,
) (*grpc.Server, error) {
	maxSendMsgSize := cfg.MaxSendMsgSize
	if maxSendMsgSize == 0 {
		maxSendMsgSize = serverconfig.DefaultGRPCMaxSendMsgSize
	}

	maxRecvMsgSize := cfg.MaxRecvMsgSize
	if maxRecvMsgSize == 0 {
		maxRecvMsgSize = serverconfig.DefaultGRPCMaxRecvMsgSize
	}

	grpcSrv := grpc.NewServer(
		grpc.ForceServerCodec(codec.NewProtoCodec(clientCtx.InterfaceRegistry).GRPCCodec()),
		grpc.MaxSendMsgSize(maxSendMsgSize),
		grpc.MaxRecvMsgSize(maxRecvMsgSize),
	)

	app.RegisterGRPCServer(grpcSrv)

	// v2alpha1 reflection — may fail when hand-written types lack proto descriptors.
	// This is non-fatal: basic gRPC queries still work without reflection.
	err := reflection.Register(grpcSrv, reflection.Config{
		SigningModes: func() map[string]int32 {
			supportedModes := clientCtx.TxConfig.SignModeHandler().SupportedModes()
			modes := make(map[string]int32, len(supportedModes))
			for _, m := range supportedModes {
				modes[m.String()] = (int32)(m)
			}
			return modes
		}(),
		ChainID:           clientCtx.ChainID,
		SdkConfig:         sdk.GetConfig(),
		InterfaceRegistry: clientCtx.InterfaceRegistry,
	})
	if err != nil {
		logger.Warn(fmt.Sprintf("gRPC v2alpha1 reflection unavailable (hand-written types): %v", err))
	}

	// gogoproto reflection — always works
	gogoreflection.Register(grpcSrv)

	return grpcSrv, nil
}
