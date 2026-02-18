package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/icaauth/types"
)

// NewQueryCmd returns the query commands for the icaauth module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "ICA auth module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryInterchainAccountsCmd(),
		NewQueryParamsCmd(),
	)

	return queryCmd
}

// NewQueryInterchainAccountsCmd returns the command to query interchain accounts by DID.
func NewQueryInterchainAccountsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts [did]",
		Short: "Query interchain accounts for a DID",
		Long:  `Query all registered interchain accounts (ICAs) associated with the given DID.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryInterchainAccountsRequest{Did: args[0]}
			resp := &types.QueryInterchainAccountsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.icaauth.v1.Query/InterchainAccounts", req, resp); err != nil {
				return fmt.Errorf("failed to query interchain accounts: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryParamsCmd returns the command to query icaauth module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the icaauth module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.icaauth.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
