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
	"github.com/cosmos/cosmos-sdk/client/rpc"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankcli "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	stakingcli "github.com/cosmos/cosmos-sdk/x/staking/client/cli"

	legibleapp "github.com/legible-money/legible/app"
	legibleauthcli "github.com/legible-money/legible/x/auth/client/cli"
	lgmgovcli "github.com/legible-money/legible/x/gov/client/cli"
	autopoiesiscli "github.com/legible-money/legible/x/autopoiesis/client/cli"
	billingcli "github.com/legible-money/legible/x/billing/client/cli"
	bvmcli "github.com/legible-money/legible/x/bvm/client/cli"
	capturechallengecli "github.com/legible-money/legible/x/capture_challenge/client/cli"
	capturedefensecli "github.com/legible-money/legible/x/capture_defense/client/cli"
	channelscli "github.com/legible-money/legible/x/channels/client/cli"
	claimingpotcli "github.com/legible-money/legible/x/claiming_pot/client/cli"
	computepoolcli "github.com/legible-money/legible/x/compute_pool/client/cli"
	discoverycli "github.com/legible-money/legible/x/discovery/client/cli"
	disputescli "github.com/legible-money/legible/x/disputes/client/cli"
	emergencycli "github.com/legible-money/legible/x/emergency/client/cli"
	evidencemgmtcli "github.com/legible-money/legible/x/evidence_mgmt/client/cli"
	homecli "github.com/legible-money/legible/x/home/client/cli"
	ibcratelimitcli "github.com/legible-money/legible/x/ibcratelimit/client/cli"
	icaauthcli "github.com/legible-money/legible/x/icaauth/client/cli"
	knowledgecli "github.com/legible-money/legible/x/knowledge/client/cli"
	ontologycli "github.com/legible-money/legible/x/ontology/client/cli"
	partnershipscli "github.com/legible-money/legible/x/partnerships/client/cli"
	qualificationcli "github.com/legible-money/legible/x/qualification/client/cli"
	researchcli "github.com/legible-money/legible/x/research/client/cli"
	schedulecli "github.com/legible-money/legible/x/schedule/client/cli"
	legiblestakingcli "github.com/legible-money/legible/x/staking/client/cli"
	tokenscli "github.com/legible-money/legible/x/tokens/client/cli"
	treecli "github.com/legible-money/legible/x/tree/client/cli"
	vestingrewardscli "github.com/legible-money/legible/x/vesting_rewards/client/cli"
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
	queryCmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	valAddrCodec := addresscodec.NewBech32Codec("lgmvaloper")
	addrCodec := addresscodec.NewBech32Codec("lgm")

	queryCmd.AddCommand(
		rpc.QueryEventForTxCmd(),
		server.QueryBlockCmd(),
		authcmd.QueryTxsByEventsCmd(),
		authcmd.QueryTxCmd(),
		server.QueryBlocksCmd(),
		server.QueryBlockResultsCmd(),
		// Legible custom modules
		legibleauthcli.NewQueryCmd(),
		autopoiesiscli.NewQueryCmd(),
		billingcli.NewQueryCmd(),
		bvmcli.NewQueryCmd(),
		capturechallengecli.NewQueryCmd(),
		capturedefensecli.NewQueryCmd(),
		channelscli.NewQueryCmd(),
		claimingpotcli.NewQueryCmd(),
		computepoolcli.NewQueryCmd(),
		discoverycli.NewQueryCmd(),
		disputescli.NewQueryCmd(),
		emergencycli.NewQueryCmd(),
		evidencemgmtcli.NewQueryCmd(),
		homecli.NewQueryCmd(),
		ibcratelimitcli.NewQueryCmd(),
		icaauthcli.NewQueryCmd(),
		knowledgecli.NewQueryCmd(),
		ontologycli.NewQueryCmd(),
		partnershipscli.NewQueryCmd(),
		qualificationcli.NewQueryCmd(),
		researchcli.NewQueryCmd(),
		schedulecli.NewQueryCmd(),
		legiblestakingcli.NewQueryCmd(),
		tokenscli.NewQueryCmd(),
		treecli.NewQueryCmd(),
		vestingrewardscli.NewQueryCmd(),
		lgmgovcli.NewQueryCmd(),
	)
	rootCmd.AddCommand(queryCmd)

	// Transaction commands
	txCmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	txCmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
		authcmd.GetSimulateCmd(),
		bankcli.NewTxCmd(addrCodec),
		stakingcli.NewTxCmd(valAddrCodec, addrCodec),
		govcli.NewTxCmd(nil),
		// Legible custom modules
		legibleauthcli.NewTxCmd(),
		autopoiesiscli.NewTxCmd(),
		billingcli.NewTxCmd(),
		bvmcli.NewTxCmd(),
		capturechallengecli.NewTxCmd(),
		capturedefensecli.NewTxCmd(),
		channelscli.NewTxCmd(),
		claimingpotcli.NewTxCmd(),
		computepoolcli.NewTxCmd(),
		discoverycli.NewTxCmd(),
		disputescli.NewTxCmd(),
		emergencycli.NewTxCmd(),
		evidencemgmtcli.NewTxCmd(),
		homecli.NewTxCmd(),
		ibcratelimitcli.NewTxCmd(),
		icaauthcli.NewTxCmd(),
		knowledgecli.NewTxCmd(),
		ontologycli.NewTxCmd(),
		partnershipscli.NewTxCmd(),
		qualificationcli.NewTxCmd(),
		researchcli.NewTxCmd(),
		schedulecli.NewTxCmd(),
		legiblestakingcli.NewTxCmd(),
		tokenscli.NewTxCmd(),
		treecli.NewTxCmd(),
		vestingrewardscli.NewTxCmd(),
		lgmgovcli.NewTxCmd(),
	)
	rootCmd.AddCommand(txCmd)

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
