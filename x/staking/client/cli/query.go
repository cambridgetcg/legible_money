package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/staking/types"
)

// NewQueryCmd returns the query commands for the legible_staking module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Legible staking module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryValidatorCmd(),
		NewQueryValidatorsCmd(),
		NewQueryDelegationCmd(),
		NewQueryDelegationsForValidatorCmd(),
		NewQueryUnbondingsCmd(),
	)

	return queryCmd
}

// queryClient implements a gRPC query client for the legible_staking module.
type queryClient struct {
	cc grpc.ClientConnInterface
}

func newQueryClient(cc grpc.ClientConnInterface) *queryClient {
	return &queryClient{cc}
}

func (c *queryClient) Params(ctx context.Context, in *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
	out := new(types.QueryParamsResponse)
	err := c.cc.Invoke(ctx, "/legible.staking.v1.Query/Params", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Validator(ctx context.Context, in *types.QueryValidatorRequest, opts ...grpc.CallOption) (*types.QueryValidatorResponse, error) {
	out := new(types.QueryValidatorResponse)
	err := c.cc.Invoke(ctx, "/legible.staking.v1.Query/Validator", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Validators(ctx context.Context, in *types.QueryValidatorsRequest, opts ...grpc.CallOption) (*types.QueryValidatorsResponse, error) {
	out := new(types.QueryValidatorsResponse)
	err := c.cc.Invoke(ctx, "/legible.staking.v1.Query/Validators", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Delegation(ctx context.Context, in *types.QueryDelegationRequest, opts ...grpc.CallOption) (*types.QueryDelegationResponse, error) {
	out := new(types.QueryDelegationResponse)
	err := c.cc.Invoke(ctx, "/legible.staking.v1.Query/Delegation", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) DelegationsForValidator(ctx context.Context, in *types.QueryDelegationsForValidatorRequest, opts ...grpc.CallOption) (*types.QueryDelegationsForValidatorResponse, error) {
	out := new(types.QueryDelegationsForValidatorResponse)
	err := c.cc.Invoke(ctx, "/legible.staking.v1.Query/DelegationsForValidator", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Unbondings(ctx context.Context, in *types.QueryUnbondingsRequest, opts ...grpc.CallOption) (*types.QueryUnbondingsResponse, error) {
	out := new(types.QueryUnbondingsResponse)
	err := c.cc.Invoke(ctx, "/legible.staking.v1.Query/Unbondings", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// NewQueryParamsCmd returns the command to query legible_staking module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the legible staking module parameters and tier configs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryValidatorCmd returns the command to query a single validator.
func NewQueryValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator [operator-address]",
		Short: "Query a validator by operator address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.Validator(cmd.Context(), &types.QueryValidatorRequest{OperatorAddress: args[0]})
			if err != nil {
				return fmt.Errorf("failed to query validator: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryValidatorsCmd returns the command to list validators.
func NewQueryValidatorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validators",
		Short: "List validators with optional filtering",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			activeOnly, _ := cmd.Flags().GetBool("active-only")
			tier, _ := cmd.Flags().GetInt32("tier")
			limit, _ := cmd.Flags().GetUint32("limit")
			offset, _ := cmd.Flags().GetUint32("offset")

			qc := newQueryClient(clientCtx)
			res, err := qc.Validators(cmd.Context(), &types.QueryValidatorsRequest{
				ActiveOnly: activeOnly,
				Tier:       tier,
				Limit:      limit,
				Offset:     offset,
			})
			if err != nil {
				return fmt.Errorf("failed to query validators: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Bool("active-only", false, "Only show active validators")
	cmd.Flags().Int32("tier", -1, "Filter by tier (-1=all, 1=apprentice, 2=verified, 3=bonded, 4=guardian)")
	cmd.Flags().Uint32("limit", 100, "Maximum number of validators to return")
	cmd.Flags().Uint32("offset", 0, "Number of validators to skip")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryDelegationCmd returns the command to query a single delegation.
func NewQueryDelegationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegation [delegator-address] [validator-address]",
		Short: "Query a delegation by delegator and validator address",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.Delegation(cmd.Context(), &types.QueryDelegationRequest{
				DelegatorAddress: args[0],
				ValidatorAddress: args[1],
			})
			if err != nil {
				return fmt.Errorf("failed to query delegation: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryDelegationsForValidatorCmd returns the command to query all delegations to a validator.
func NewQueryDelegationsForValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegations-for-validator [validator-address]",
		Short: "Query all delegations to a specific validator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.DelegationsForValidator(cmd.Context(), &types.QueryDelegationsForValidatorRequest{
				ValidatorAddress: args[0],
			})
			if err != nil {
				return fmt.Errorf("failed to query delegations: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryUnbondingsCmd returns the command to query unbonding entries for a delegator.
func NewQueryUnbondingsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unbondings [delegator-address]",
		Short: "Query unbonding entries for a delegator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.Unbondings(cmd.Context(), &types.QueryUnbondingsRequest{
				DelegatorAddress: args[0],
			})
			if err != nil {
				return fmt.Errorf("failed to query unbondings: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
