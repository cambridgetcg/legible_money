package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/auth/types"
)

// NewQueryCmd returns the query commands for the legible_auth module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Legible auth module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryAccountCmd(),
		NewQueryAccountByDIDCmd(),
		NewQuerySessionKeysCmd(),
	)

	return queryCmd
}

// queryClient implements a gRPC query client for the legible_auth module.
type queryClient struct {
	cc grpc.ClientConnInterface
}

// newQueryClient creates a new query client for the legible_auth module.
func newQueryClient(cc grpc.ClientConnInterface) *queryClient {
	return &queryClient{cc}
}

func (c *queryClient) Params(ctx context.Context, in *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
	out := new(types.QueryParamsResponse)
	err := c.cc.Invoke(ctx, "/legible.auth.v1.Query/Params", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Account(ctx context.Context, in *types.QueryAccountRequest, opts ...grpc.CallOption) (*types.QueryAccountResponse, error) {
	out := new(types.QueryAccountResponse)
	err := c.cc.Invoke(ctx, "/legible.auth.v1.Query/Account", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) AccountByDID(ctx context.Context, in *types.QueryAccountByDIDRequest, opts ...grpc.CallOption) (*types.QueryAccountByDIDResponse, error) {
	out := new(types.QueryAccountByDIDResponse)
	err := c.cc.Invoke(ctx, "/legible.auth.v1.Query/AccountByDID", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) SessionKeys(ctx context.Context, in *types.QuerySessionKeysRequest, opts ...grpc.CallOption) (*types.QuerySessionKeysResponse, error) {
	out := new(types.QuerySessionKeysResponse)
	err := c.cc.Invoke(ctx, "/legible.auth.v1.Query/SessionKeys", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// NewQueryParamsCmd returns the command to query legible_auth module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the legible auth module parameters",
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

// NewQueryAccountCmd returns the command to query an account by address.
func NewQueryAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [address]",
		Short: "Query a Legible account by bech32 address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.Account(cmd.Context(), &types.QueryAccountRequest{Address: args[0]})
			if err != nil {
				return fmt.Errorf("failed to query account: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryAccountByDIDCmd returns the command to query an account by DID.
func NewQueryAccountByDIDCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account-by-did [did]",
		Short: "Query a Legible account by DID",
		Long:  `Query a Legible account using its Decentralized Identifier (did:lgm:{hex}).`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.AccountByDID(cmd.Context(), &types.QueryAccountByDIDRequest{Did: args[0]})
			if err != nil {
				return fmt.Errorf("failed to query account by DID: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQuerySessionKeysCmd returns the command to query session keys for an owner.
func NewQuerySessionKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session-keys [owner-address]",
		Short: "Query all session keys for an owner",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.SessionKeys(cmd.Context(), &types.QuerySessionKeysRequest{Owner: args[0]})
			if err != nil {
				return fmt.Errorf("failed to query session keys: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
