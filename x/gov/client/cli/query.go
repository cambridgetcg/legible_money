package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/gov/types"
)

// NewQueryCmd returns the query commands for the governance module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Legible governance module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryLIPCmd(),
		NewQueryLIPsCmd(),
		NewQueryDisbursementCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query governance module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the governance module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.gov.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryLIPCmd returns the command to query a single LIP.
func NewQueryLIPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lip [lip-id]",
		Short: "Query a LIP by ID (includes votes)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryLIPRequest{LipId: args[0]}
			resp := &types.QueryLIPResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.gov.v1.Query/LIP", req, resp); err != nil {
				return fmt.Errorf("failed to query LIP: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryLIPsCmd returns the command to list LIPs.
func NewQueryLIPsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lips",
		Short: "List LIPs (optionally filtered by stage)",
		Long: `List all LIPs, optionally filtered by stage.

Stages: draft, review, last_call, voting, accepted, rejected, withdrawn`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			stage, _ := cmd.Flags().GetString("stage")
			limit, _ := cmd.Flags().GetUint32("limit")
			offset, _ := cmd.Flags().GetUint32("offset")

			req := &types.QueryLIPsRequest{
				Stage:  stage,
				Limit:  limit,
				Offset: offset,
			}
			resp := &types.QueryLIPsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.gov.v1.Query/LIPs", req, resp); err != nil {
				return fmt.Errorf("failed to query LIPs: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	cmd.Flags().String("stage", "", "Filter by stage (draft, review, last_call, voting, accepted, rejected, withdrawn)")
	cmd.Flags().Uint32("limit", 100, "Maximum number of results")
	cmd.Flags().Uint32("offset", 0, "Offset for pagination")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryDisbursementCmd returns the command to query a single disbursement.
func NewQueryDisbursementCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disbursement [disbursement-id]",
		Short: "Query a disbursement by ID (includes votes)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryDisbursementRequest{DisbursementId: args[0]}
			resp := &types.QueryDisbursementResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.gov.v1.Query/Disbursement", req, resp); err != nil {
				return fmt.Errorf("failed to query disbursement: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
