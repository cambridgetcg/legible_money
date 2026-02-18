package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/compute_pool/types"
)

// NewQueryCmd returns the query commands for the compute_pool module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Compute pool module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryProviderCmd(),
		NewQueryActiveProvidersCmd(),
		NewQueryCreditBalanceCmd(),
		NewQueryRedemptionTicketCmd(),
		NewQueryPoolStatsCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query compute_pool module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the compute pool module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.compute_pool.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryProviderCmd returns the command to query a compute provider.
func NewQueryProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider [address]",
		Short: "Query a compute provider by address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryProviderRequest{Address: args[0]}
			resp := &types.QueryProviderResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.compute_pool.v1.Query/Provider", req, resp); err != nil {
				return fmt.Errorf("failed to query provider: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryActiveProvidersCmd returns the command to list active providers.
func NewQueryActiveProvidersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "active-providers",
		Short: "List all active compute providers",
		Long: `List all active compute providers, optionally filtered by service type.

Service types: ai_inference, gpu_compute, storage`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			serviceType, _ := cmd.Flags().GetString("service-type")

			req := &types.QueryActiveProvidersRequest{ServiceType: serviceType}
			resp := &types.QueryActiveProvidersResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.compute_pool.v1.Query/ActiveProviders", req, resp); err != nil {
				return fmt.Errorf("failed to query active providers: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	cmd.Flags().String("service-type", "", "Filter by service type (ai_inference, gpu_compute, storage)")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// NewQueryCreditBalanceCmd returns the command to query credit balance.
func NewQueryCreditBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credit-balance [validator-address]",
		Short: "Query a validator's compute credit balance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryCreditBalanceRequest{ValidatorAddr: args[0]}
			resp := &types.QueryCreditBalanceResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.compute_pool.v1.Query/CreditBalance", req, resp); err != nil {
				return fmt.Errorf("failed to query credit balance: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryRedemptionTicketCmd returns the command to query a redemption ticket.
func NewQueryRedemptionTicketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ticket [ticket-id]",
		Short: "Query a redemption ticket by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			ticketID, err := parseTicketID(args[0])
			if err != nil {
				return fmt.Errorf("invalid ticket-id: %w", err)
			}

			req := &types.QueryRedemptionTicketRequest{TicketId: ticketID}
			resp := &types.QueryRedemptionTicketResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.compute_pool.v1.Query/RedemptionTicket", req, resp); err != nil {
				return fmt.Errorf("failed to query ticket: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryPoolStatsCmd returns the command to query pool statistics.
func NewQueryPoolStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool-stats",
		Short: "Query current compute pool balance and epoch statistics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryPoolStatsRequest{}
			resp := &types.QueryPoolStatsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.compute_pool.v1.Query/PoolStats", req, resp); err != nil {
				return fmt.Errorf("failed to query pool stats: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// parseTicketID parses a string to uint64.
func parseTicketID(s string) (uint64, error) {
	var n uint64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
