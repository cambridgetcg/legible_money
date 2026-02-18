package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/tokens/types"
)

// NewQueryCmd returns the query commands for the tokens module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Tokens module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query tokens module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the tokens module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			res := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(),
				fmt.Sprintf("/%s.%s.v1.Query/Params", "legible", types.ModuleName),
				req, res,
			); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return err
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
