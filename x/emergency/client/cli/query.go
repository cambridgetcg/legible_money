package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/emergency/types"
)

// NewQueryCmd returns the query commands for the emergency module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Emergency module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryStatusCmd(),
		NewQueryActiveCeremonyCmd(),
		NewQueryCompletedCeremoniesCmd(),
		NewQueryAuditLogCmd(),
	)

	return queryCmd
}

// NewQueryStatusCmd returns the command to query the emergency status.
func NewQueryStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Query the current emergency status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryStatusRequest{}
			resp := &types.QueryStatusResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.emergency.v1.Query/Status", req, resp); err != nil {
				return fmt.Errorf("failed to query status: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryActiveCeremonyCmd returns the command to query the active ceremony.
func NewQueryActiveCeremonyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "active-ceremony",
		Short: "Query the currently active emergency ceremony",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryActiveCeremonyRequest{}
			resp := &types.QueryActiveCeremonyResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.emergency.v1.Query/ActiveCeremony", req, resp); err != nil {
				return fmt.Errorf("failed to query active ceremony: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryCompletedCeremoniesCmd returns the command to list completed ceremonies.
func NewQueryCompletedCeremoniesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completed-ceremonies",
		Short: "List completed (finalized/failed) ceremonies",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetUint32("limit")
			offset, _ := cmd.Flags().GetUint32("offset")

			req := &types.QueryCompletedCeremoniesRequest{
				Limit:  limit,
				Offset: offset,
			}
			resp := &types.QueryCompletedCeremoniesResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.emergency.v1.Query/CompletedCeremonies", req, resp); err != nil {
				return fmt.Errorf("failed to query completed ceremonies: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	cmd.Flags().Uint32("limit", 20, "Maximum number of ceremonies to return")
	cmd.Flags().Uint32("offset", 0, "Number of ceremonies to skip")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// NewQueryAuditLogCmd returns the command to query the emergency audit log.
func NewQueryAuditLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit-log",
		Short: "Query the emergency audit log",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetUint32("limit")
			offset, _ := cmd.Flags().GetUint32("offset")

			req := &types.QueryAuditLogRequest{
				Limit:  limit,
				Offset: offset,
			}
			resp := &types.QueryAuditLogResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.emergency.v1.Query/AuditLog", req, resp); err != nil {
				return fmt.Errorf("failed to query audit log: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	cmd.Flags().Uint32("limit", 20, "Maximum number of entries to return")
	cmd.Flags().Uint32("offset", 0, "Number of entries to skip")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
