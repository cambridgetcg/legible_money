package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/partnerships/types"
)

// NewQueryCmd returns the query commands for the partnerships module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Partnerships module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryPartnershipCmd(),
		NewQueryPartnershipsByParticipantCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query partnerships module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the partnerships module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.partnerships.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryPartnershipCmd returns the command to query a single partnership.
func NewQueryPartnershipCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "partnership [partnership-id]",
		Short: "Query a partnership by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryPartnershipRequest{PartnershipId: args[0]}
			resp := &types.QueryPartnershipResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.partnerships.v1.Query/Partnership", req, resp); err != nil {
				return fmt.Errorf("failed to query partnership: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryPartnershipsByParticipantCmd returns the command to query partnerships by participant.
func NewQueryPartnershipsByParticipantCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "by-participant [address]",
		Short: "Query all partnerships for a participant address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryPartnershipsByParticipantRequest{Participant: args[0]}
			resp := &types.QueryPartnershipsByParticipantResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.partnerships.v1.Query/PartnershipsByParticipant", req, resp); err != nil {
				return fmt.Errorf("failed to query partnerships: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
