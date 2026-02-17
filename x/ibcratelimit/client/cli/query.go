package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/ibcratelimit/types"
)

// NewQueryCmd returns the query commands for the ibcratelimit module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "IBC rate limit module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryChannelFlowCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query ibcratelimit module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the IBC rate limit module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.ibcratelimit.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryChannelFlowCmd returns the command to query a channel's flow record.
func NewQueryChannelFlowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel-flow [channel-id]",
		Short: "Query the flow record for an IBC channel",
		Long: `Query the current inflow/outflow tracking record for a specific IBC channel.
Returns the flow amounts and whether rate limits have been hit.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryChannelFlowRequest{ChannelID: args[0]}
			resp := &types.QueryChannelFlowResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.ibcratelimit.v1.Query/ChannelFlow", req, resp); err != nil {
				return fmt.Errorf("failed to query channel flow: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
