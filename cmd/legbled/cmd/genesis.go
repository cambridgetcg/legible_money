package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
)

// AddGenesisAccountCmd returns a command to add a genesis account to genesis.json.
func AddGenesisAccountCmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-genesis-account [address_or_key_name] [coin][,[coin]]",
		Short: "Add a genesis account to genesis.json",
		Long: `Add a genesis account to genesis.json. The provided account must specify
the account address or key name and a list of initial coins. Coin amounts
are in ulgm (micro-LGM). Example:

  legbled add-genesis-account lgm1... 1000000000ulgm

This adds an account with 1,000 LGM to the genesis file.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config
			config.SetRoot(clientCtx.HomeDir)

			addr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return fmt.Errorf("invalid address: %w", err)
			}

			coins, err := sdk.ParseCoinsNormalized(args[1])
			if err != nil {
				return fmt.Errorf("invalid coins: %w", err)
			}

			// Create the account.
			baseAccount := authtypes.NewBaseAccount(addr, nil, 0, 0)
			genAccount := baseAccount

			genFile := config.GenesisFile()
			appState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genFile)
			if err != nil {
				return fmt.Errorf("failed to unmarshal genesis state: %w", err)
			}

			// Add account to auth genesis.
			authGenesis := authtypes.GetGenesisStateFromAppState(clientCtx.Codec, appState)
			accs, err := authtypes.UnpackAccounts(authGenesis.Accounts)
			if err != nil {
				return fmt.Errorf("failed to get accounts from genesis: %w", err)
			}
			if accs.Contains(addr) {
				return fmt.Errorf("account already exists in genesis: %s", addr)
			}
			accs = append(accs, genAccount)
			accs = authtypes.SanitizeGenesisAccounts(accs)

			genAccs, err := authtypes.PackAccounts(accs)
			if err != nil {
				return fmt.Errorf("failed to convert accounts: %w", err)
			}
			authGenesis.Accounts = genAccs
			authGenesisBz, err := clientCtx.Codec.MarshalJSON(&authGenesis)
			if err != nil {
				return fmt.Errorf("failed to marshal auth genesis: %w", err)
			}
			appState[authtypes.ModuleName] = authGenesisBz

			// Add balances to bank genesis.
			bankGenesis := banktypes.GetGenesisStateFromAppState(clientCtx.Codec, appState)
			bankGenesis.Balances = append(bankGenesis.Balances, banktypes.Balance{
				Address: addr.String(),
				Coins:   coins,
			})
			bankGenesis.Balances = banktypes.SanitizeGenesisBalances(bankGenesis.Balances)
			bankGenesis.Supply = bankGenesis.Supply.Add(coins...)

			bankGenesisBz, err := clientCtx.Codec.MarshalJSON(bankGenesis)
			if err != nil {
				return fmt.Errorf("failed to marshal bank genesis: %w", err)
			}
			appState[banktypes.ModuleName] = bankGenesisBz

			appStateJSON, err := json.Marshal(appState)
			if err != nil {
				return fmt.Errorf("failed to marshal app state: %w", err)
			}
			genDoc.AppState = appStateJSON

			return genutil.ExportGenesisFile(genDoc, genFile)
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "node's home directory")
	return cmd
}

// MigrateGenesisCmd returns a command to migrate TS chain state to Cosmos genesis.
// Phase 5: Full implementation. Phase 0: Stub only.
func MigrateGenesisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate-genesis [ts-state-export.json]",
		Short: "Migrate TypeScript chain state to Cosmos genesis format",
		Long: `Migrate the exported state from the TypeScript Legible Money chain
into a Cosmos SDK genesis file. This converts:

  - AccountStateTree   → x/auth accounts + x/bank balances
  - ValidatorStateTree → x/staking validators + delegations
  - KnowledgeStateTree → x/knowledge facts + references
  - DID addresses      → bech32 (lgm1...) with backward-compat mapping

Usage:
  legbled migrate-genesis ./ts-state-export.json --chain-id lgm-1

This command will be fully implemented in Phase 5 of the migration.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("migrate-genesis is not yet implemented (Phase 5)")
		},
	}

	cmd.Flags().String("chain-id", "lgm-1", "Chain ID for the new Cosmos chain")
	cmd.Flags().String("output", "genesis.json", "Output genesis file path")
	return cmd
}
