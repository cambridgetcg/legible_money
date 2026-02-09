package cmd

import (
	"io"
	"os"

	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/keys"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	legibleapp "github.com/legible-money/legible/app"
)

// NewRootCmd creates the root command for the legbled daemon.
func NewRootCmd() *cobra.Command {
	encodingConfig := legibleapp.MakeEncodingConfig()

	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithHomeDir(legibleapp.DefaultNodeHome).
		WithViper("LGM")

	rootCmd := &cobra.Command{
		Use:   "legbled",
		Short: "Legible Money - Blockchain for AI Agent Economies",
		Long: `legbled is the daemon for the Legible Money blockchain, a protocol
for AI agent economies using Proof of Truth (PoT) consensus.

Token: LGM (ulgm)
Block time: 2521ms
Finality: 19 blocks
Consensus: Proof of Truth`,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}
			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			customAppTemplate, customAppConfig := initAppConfig()
			customCMTConfig := initCometBFTConfig()

			return server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customCMTConfig)
		},
	}

	initRootCmd(rootCmd, encodingConfig)
	return rootCmd
}

func initRootCmd(rootCmd *cobra.Command, encodingConfig legibleapp.EncodingConfig) {
	rootCmd.AddCommand(
		genutilcli.InitCmd(legibleapp.ModuleBasics, legibleapp.DefaultNodeHome),
		debug.Cmd(),
	)

	server.AddCommands(rootCmd, legibleapp.DefaultNodeHome, newApp, appExport, addModuleInitFlags)

	// Key management
	rootCmd.AddCommand(
		keys.Commands(),
	)

	// Query commands
	rootCmd.AddCommand(
		server.QueryBlockCmd(),
		authcmd.QueryTxsByEventsCmd(),
		authcmd.QueryTxCmd(),
	)

	// Genesis commands
	rootCmd.AddCommand(
		genutilcli.CollectGenTxsCmd(banktypes.GenesisBalancesIterator{}, legibleapp.DefaultNodeHome, genutiltypes.DefaultMessageValidator, addresscodec.NewBech32Codec("lgmvaloper")),
		genutilcli.GenTxCmd(legibleapp.ModuleBasics, encodingConfig.TxConfig, banktypes.GenesisBalancesIterator{}, legibleapp.DefaultNodeHome, addresscodec.NewBech32Codec("lgmvaloper")),
		genutilcli.ValidateGenesisCmd(legibleapp.ModuleBasics),
		AddGenesisAccountCmd(legibleapp.DefaultNodeHome),
		MigrateGenesisCmd(),
	)
}

func newApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) servertypes.Application {
	return legibleapp.NewLegibleApp(
		logger, db, traceStore, true, appOpts,
		server.DefaultBaseappOptions(appOpts)...,
	)
}

func appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	app := legibleapp.NewLegibleApp(logger, db, traceStore, false, appOpts)
	if height != -1 {
		if err := app.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}
	return app.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

func addModuleInitFlags(startCmd *cobra.Command) {
	// Custom init flags for Legible Money modules can be added here.
}
