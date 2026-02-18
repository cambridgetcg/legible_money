package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/billing/types"
)

// NewQueryCmd returns the query commands for the billing module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Billing module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryProviderCmd(),
		NewQueryQuoteCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query billing module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the billing module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.billing.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryProviderCmd returns the command to query a billing provider.
func NewQueryProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider [address]",
		Short: "Query a knowledge API provider by address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryProviderRequest{Address: args[0]}
			resp := &types.QueryProviderResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.billing.v1.Query/Provider", req, resp); err != nil {
				return fmt.Errorf("failed to query provider: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryQuoteCmd returns the command to query a quote by ID.
func NewQueryQuoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quote [quote-id]",
		Short: "Query an API quote by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			quoteID, err := parseQuoteID(args[0])
			if err != nil {
				return fmt.Errorf("invalid quote-id: %w", err)
			}

			req := &types.QueryQuoteRequest{QuoteId: quoteID}
			resp := &types.QueryQuoteResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.billing.v1.Query/Quote", req, resp); err != nil {
				return fmt.Errorf("failed to query quote: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// parseQuoteID parses a string to uint64.
func parseQuoteID(s string) (uint64, error) {
	var n uint64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
