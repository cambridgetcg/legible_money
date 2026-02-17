package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/claiming_pot/types"
)

// NewQueryCmd returns the query commands for the claiming_pot module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Claiming pot module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryPotCmd(),
		NewQueryPotsCmd(),
		NewQueryPotsByCreatorCmd(),
		NewQueryEligibilityCmd(),
		NewQueryClaimRecordsCmd(),
		NewQueryParamsCmd(),
	)

	return queryCmd
}

// NewQueryPotCmd returns the command to query a single claiming pot.
func NewQueryPotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pot [pot-id]",
		Short: "Query a claiming pot by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryPotRequest{PotId: args[0]}
			res := &types.QueryPotResponse{}
			if err := clientCtx.Invoke(cmd.Context(),
				"/legible.claiming_pot.v1.Query/Pot",
				req, res,
			); err != nil {
				return fmt.Errorf("failed to query pot: %w", err)
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

// NewQueryPotsCmd returns the command to list claiming pots.
func NewQueryPotsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pots",
		Short: "List claiming pots, optionally filtered by status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			status, _ := cmd.Flags().GetString("status")
			limit, _ := cmd.Flags().GetUint32("limit")
			offset, _ := cmd.Flags().GetUint32("offset")

			req := &types.QueryPotsRequest{
				Status: status,
				Limit:  limit,
				Offset: offset,
			}
			res := &types.QueryPotsResponse{}
			if err := clientCtx.Invoke(cmd.Context(),
				"/legible.claiming_pot.v1.Query/Pots",
				req, res,
			); err != nil {
				return fmt.Errorf("failed to query pots: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return err
		},
	}

	cmd.Flags().String("status", "", "Filter by pot status (active, exhausted, expired, closed)")
	cmd.Flags().Uint32("limit", 0, "Maximum number of results to return")
	cmd.Flags().Uint32("offset", 0, "Number of results to skip")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryPotsByCreatorCmd returns the command to list pots by creator.
func NewQueryPotsByCreatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pots-by-creator [creator]",
		Short: "List claiming pots created by a specific address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetUint32("limit")
			offset, _ := cmd.Flags().GetUint32("offset")

			req := &types.QueryPotsByCreatorRequest{
				Creator: args[0],
				Limit:   limit,
				Offset:  offset,
			}
			res := &types.QueryPotsByCreatorResponse{}
			if err := clientCtx.Invoke(cmd.Context(),
				"/legible.claiming_pot.v1.Query/PotsByCreator",
				req, res,
			); err != nil {
				return fmt.Errorf("failed to query pots by creator: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return err
		},
	}

	cmd.Flags().Uint32("limit", 0, "Maximum number of results to return")
	cmd.Flags().Uint32("offset", 0, "Number of results to skip")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryEligibilityCmd returns the command to check claim eligibility.
func NewQueryEligibilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eligibility [pot-id] [claimer]",
		Short: "Check if an address is eligible to claim from a pot",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryEligibilityRequest{
				PotId:   args[0],
				Claimer: args[1],
			}
			res := &types.QueryEligibilityResponse{}
			if err := clientCtx.Invoke(cmd.Context(),
				"/legible.claiming_pot.v1.Query/Eligibility",
				req, res,
			); err != nil {
				return fmt.Errorf("failed to query eligibility: %w", err)
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

// NewQueryClaimRecordsCmd returns the command to list claim records for a pot.
func NewQueryClaimRecordsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-records [pot-id]",
		Short: "List claim records for a claiming pot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetUint32("limit")
			offset, _ := cmd.Flags().GetUint32("offset")

			req := &types.QueryClaimRecordsRequest{
				PotId:  args[0],
				Limit:  limit,
				Offset: offset,
			}
			res := &types.QueryClaimRecordsResponse{}
			if err := clientCtx.Invoke(cmd.Context(),
				"/legible.claiming_pot.v1.Query/ClaimRecords",
				req, res,
			); err != nil {
				return fmt.Errorf("failed to query claim records: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return err
		},
	}

	cmd.Flags().Uint32("limit", 0, "Maximum number of results to return")
	cmd.Flags().Uint32("offset", 0, "Number of results to skip")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryParamsCmd returns the command to query claiming_pot module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the claiming pot module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			res := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(),
				"/legible.claiming_pot.v1.Query/Params",
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
