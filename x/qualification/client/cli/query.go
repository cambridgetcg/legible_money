package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/qualification/types"
)

// NewQueryCmd returns the query commands for the qualification module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Qualification module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryQualificationCmd(),
		NewQueryQualificationsByValidatorCmd(),
		NewQueryQualifiedValidatorsCmd(),
		NewQueryEndorsementCmd(),
		NewQueryEndorsementsForValidatorCmd(),
	)

	return queryCmd
}

// queryClient implements a gRPC query client for the qualification module.
type queryClient struct {
	cc grpc.ClientConnInterface
}

func newQueryClient(cc grpc.ClientConnInterface) *queryClient {
	return &queryClient{cc}
}

func (c *queryClient) Params(ctx context.Context, in *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
	out := new(types.QueryParamsResponse)
	err := c.cc.Invoke(ctx, "/legible.qualification.v1.Query/Params", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Qualification(ctx context.Context, in *types.QueryQualificationRequest, opts ...grpc.CallOption) (*types.QueryQualificationResponse, error) {
	out := new(types.QueryQualificationResponse)
	err := c.cc.Invoke(ctx, "/legible.qualification.v1.Query/Qualification", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) QualificationsByValidator(ctx context.Context, in *types.QueryQualificationsByValidatorRequest, opts ...grpc.CallOption) (*types.QueryQualificationsByValidatorResponse, error) {
	out := new(types.QueryQualificationsByValidatorResponse)
	err := c.cc.Invoke(ctx, "/legible.qualification.v1.Query/QualificationsByValidator", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) QualifiedValidators(ctx context.Context, in *types.QueryQualifiedValidatorsRequest, opts ...grpc.CallOption) (*types.QueryQualifiedValidatorsResponse, error) {
	out := new(types.QueryQualifiedValidatorsResponse)
	err := c.cc.Invoke(ctx, "/legible.qualification.v1.Query/QualifiedValidators", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Endorsement(ctx context.Context, in *types.QueryEndorsementRequest, opts ...grpc.CallOption) (*types.QueryEndorsementResponse, error) {
	out := new(types.QueryEndorsementResponse)
	err := c.cc.Invoke(ctx, "/legible.qualification.v1.Query/Endorsement", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) EndorsementsForValidator(ctx context.Context, in *types.QueryEndorsementsForValidatorRequest, opts ...grpc.CallOption) (*types.QueryEndorsementsForValidatorResponse, error) {
	out := new(types.QueryEndorsementsForValidatorResponse)
	err := c.cc.Invoke(ctx, "/legible.qualification.v1.Query/EndorsementsForValidator", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// NewQueryParamsCmd returns the command to query qualification module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the qualification module parameters",
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

			return clientCtx.PrintObjectLegacy(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryQualificationCmd returns the command to query a qualification.
func NewQueryQualificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qualification [validator-address] [domain]",
		Short: "Query a validator's qualification in a domain",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.Qualification(cmd.Context(), &types.QueryQualificationRequest{
				ValidatorAddr: args[0],
				Domain:        args[1],
			})
			if err != nil {
				return fmt.Errorf("failed to query qualification: %w", err)
			}

			return clientCtx.PrintObjectLegacy(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryQualificationsByValidatorCmd returns the command to query all qualifications for a validator.
func NewQueryQualificationsByValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qualifications-by-validator [validator-address]",
		Short: "Query all domain qualifications for a validator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.QualificationsByValidator(cmd.Context(), &types.QueryQualificationsByValidatorRequest{
				ValidatorAddr: args[0],
			})
			if err != nil {
				return fmt.Errorf("failed to query qualifications: %w", err)
			}

			return clientCtx.PrintObjectLegacy(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryQualifiedValidatorsCmd returns the command to query validators qualified in a domain.
func NewQueryQualifiedValidatorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qualified-validators [domain]",
		Short: "Query all validators qualified in a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.QualifiedValidators(cmd.Context(), &types.QueryQualifiedValidatorsRequest{
				Domain: args[0],
			})
			if err != nil {
				return fmt.Errorf("failed to query qualified validators: %w", err)
			}

			return clientCtx.PrintObjectLegacy(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryEndorsementCmd returns the command to query an endorsement by ID.
func NewQueryEndorsementCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "endorsement [endorsement-id]",
		Short: "Query an endorsement by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.Endorsement(cmd.Context(), &types.QueryEndorsementRequest{
				EndorsementId: args[0],
			})
			if err != nil {
				return fmt.Errorf("failed to query endorsement: %w", err)
			}

			return clientCtx.PrintObjectLegacy(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryEndorsementsForValidatorCmd returns the command to query endorsements targeting a validator.
func NewQueryEndorsementsForValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "endorsements-for-validator [validator-address]",
		Short: "Query all endorsements targeting a validator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.EndorsementsForValidator(cmd.Context(), &types.QueryEndorsementsForValidatorRequest{
				TargetValidator: args[0],
			})
			if err != nil {
				return fmt.Errorf("failed to query endorsements: %w", err)
			}

			return clientCtx.PrintObjectLegacy(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
