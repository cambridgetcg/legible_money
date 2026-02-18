package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/channels/types"
)

// NewQueryCmd returns the query commands for the channels module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Channels module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryChannelCmd(),
		NewQueryChannelsByPayerCmd(),
		NewQueryChannelsByProviderCmd(),
		NewQueryActiveDisputeCmd(),
		NewQueryParamsCmd(),
	)

	return queryCmd
}

// NewQueryChannelCmd returns the command for querying a channel by ID.
func NewQueryChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel [channel-id]",
		Short: "Query a payment channel by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryChannelRequest{
				ChannelId: args[0],
			}
			res := &types.QueryChannelResponse{}

			err = clientCtx.Invoke(cmd.Context(), "/legible.channels.v1.Query/Channel", req, res)
			if err != nil {
				return err
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

// NewQueryChannelsByPayerCmd returns the command for querying channels by payer address.
func NewQueryChannelsByPayerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channels-by-payer [payer]",
		Short: "Query all channels for a given payer address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, err := cmd.Flags().GetUint32("limit")
			if err != nil {
				return err
			}
			offset, err := cmd.Flags().GetUint32("offset")
			if err != nil {
				return err
			}

			req := &types.QueryChannelsByPayerRequest{
				Payer:  args[0],
				Limit:  limit,
				Offset: offset,
			}
			res := &types.QueryChannelsByPayerResponse{}

			err = clientCtx.Invoke(cmd.Context(), "/legible.channels.v1.Query/ChannelsByPayer", req, res)
			if err != nil {
				return err
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return err
		},
	}

	cmd.Flags().Uint32("limit", 0, "Maximum number of channels to return")
	cmd.Flags().Uint32("offset", 0, "Number of channels to skip")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryChannelsByProviderCmd returns the command for querying channels by provider address.
func NewQueryChannelsByProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channels-by-provider [provider]",
		Short: "Query all channels for a given provider address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, err := cmd.Flags().GetUint32("limit")
			if err != nil {
				return err
			}
			offset, err := cmd.Flags().GetUint32("offset")
			if err != nil {
				return err
			}

			req := &types.QueryChannelsByProviderRequest{
				Provider: args[0],
				Limit:    limit,
				Offset:   offset,
			}
			res := &types.QueryChannelsByProviderResponse{}

			err = clientCtx.Invoke(cmd.Context(), "/legible.channels.v1.Query/ChannelsByProvider", req, res)
			if err != nil {
				return err
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return err
		},
	}

	cmd.Flags().Uint32("limit", 0, "Maximum number of channels to return")
	cmd.Flags().Uint32("offset", 0, "Number of channels to skip")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryActiveDisputeCmd returns the command for querying an active dispute.
func NewQueryActiveDisputeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "active-dispute [channel-id]",
		Short: "Query the active dispute for a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryActiveDisputeRequest{
				ChannelId: args[0],
			}
			res := &types.QueryActiveDisputeResponse{}

			err = clientCtx.Invoke(cmd.Context(), "/legible.channels.v1.Query/ActiveDispute", req, res)
			if err != nil {
				return err
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

// NewQueryParamsCmd returns the command for querying the module parameters.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the channels module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			res := &types.QueryParamsResponse{}

			err = clientCtx.Invoke(cmd.Context(), "/legible.channels.v1.Query/Params", req, res)
			if err != nil {
				return err
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
