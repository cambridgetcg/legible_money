package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/bvm/types"
)

// NewQueryCmd returns the query commands for the bvm module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "BVM module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryContractCmd(),
		NewQueryContractsByCreatorCmd(),
		NewQueryCodeCmd(),
		NewQueryContractStateCmd(),
		NewQueryScheduleCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query bvm module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the BVM module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.bvm.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryContractCmd returns the command to query a contract by address.
func NewQueryContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contract [address]",
		Short: "Query a deployed contract by address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryContractRequest{Address: args[0]}
			resp := &types.QueryContractResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.bvm.v1.Query/Contract", req, resp); err != nil {
				return fmt.Errorf("failed to query contract: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryContractsByCreatorCmd returns the command to list contracts by creator.
func NewQueryContractsByCreatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contracts-by-creator [creator-address]",
		Short: "List contracts deployed by a specific creator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetUint32("limit")
			offset, _ := cmd.Flags().GetUint32("offset")

			req := &types.QueryContractsByCreatorRequest{
				Creator: args[0],
				Offset:  offset,
				Limit:   limit,
			}
			resp := &types.QueryContractsByCreatorResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.bvm.v1.Query/ContractsByCreator", req, resp); err != nil {
				return fmt.Errorf("failed to query contracts: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	cmd.Flags().Uint32("limit", 20, "Maximum number of contracts to return")
	cmd.Flags().Uint32("offset", 0, "Number of contracts to skip")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// NewQueryCodeCmd returns the command to query contract code by hash.
func NewQueryCodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "code [code-hash]",
		Short: "Query contract bytecode by hash",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryCodeRequest{CodeHash: args[0]}
			resp := &types.QueryCodeResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.bvm.v1.Query/Code", req, resp); err != nil {
				return fmt.Errorf("failed to query code: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryContractStateCmd returns the command to query contract state.
func NewQueryContractStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contract-state [contract-address] [key]",
		Short: "Query a value from a contract's state",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryContractStateRequest{
				ContractAddress: args[0],
				Key:             args[1],
			}
			resp := &types.QueryContractStateResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.bvm.v1.Query/ContractState", req, resp); err != nil {
				return fmt.Errorf("failed to query contract state: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryScheduleCmd returns the command to query a schedule by ID.
func NewQueryScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule [schedule-id]",
		Short: "Query a BVM schedule by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryScheduleRequest{ScheduleId: args[0]}
			resp := &types.QueryScheduleResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.bvm.v1.Query/Schedule", req, resp); err != nil {
				return fmt.Errorf("failed to query schedule: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
