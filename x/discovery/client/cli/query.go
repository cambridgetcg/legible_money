package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/discovery/types"
)

// NewQueryCmd returns the query commands for the discovery module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Discovery module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryAgentCmd(),
		NewQueryAgentsByDomainCmd(),
		NewQueryAgentsByCapabilityCmd(),
	)

	return queryCmd
}

// queryClient implements a gRPC query client for the discovery module.
type queryClient struct {
	cc grpc.ClientConnInterface
}

func newQueryClient(cc grpc.ClientConnInterface) *queryClient {
	return &queryClient{cc}
}

func (c *queryClient) Params(ctx context.Context, in *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
	out := new(types.QueryParamsResponse)
	err := c.cc.Invoke(ctx, "/legible.discovery.v1.Query/Params", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Agent(ctx context.Context, in *types.QueryAgentRequest, opts ...grpc.CallOption) (*types.QueryAgentResponse, error) {
	out := new(types.QueryAgentResponse)
	err := c.cc.Invoke(ctx, "/legible.discovery.v1.Query/Agent", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) AgentsByDomain(ctx context.Context, in *types.QueryAgentsByDomainRequest, opts ...grpc.CallOption) (*types.QueryAgentsByDomainResponse, error) {
	out := new(types.QueryAgentsByDomainResponse)
	err := c.cc.Invoke(ctx, "/legible.discovery.v1.Query/AgentsByDomain", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) AgentsByCapability(ctx context.Context, in *types.QueryAgentsByCapabilityRequest, opts ...grpc.CallOption) (*types.QueryAgentsByCapabilityResponse, error) {
	out := new(types.QueryAgentsByCapabilityResponse)
	err := c.cc.Invoke(ctx, "/legible.discovery.v1.Query/AgentsByCapability", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// NewQueryParamsCmd returns the command to query discovery module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the discovery module parameters",
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

// NewQueryAgentCmd returns the command to query an agent by address.
func NewQueryAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent [address]",
		Short: "Query an agent profile by address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.Agent(cmd.Context(), &types.QueryAgentRequest{Address: args[0]})
			if err != nil {
				return fmt.Errorf("failed to query agent: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryAgentsByDomainCmd returns the command to query agents by domain.
func NewQueryAgentsByDomainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents-by-domain [domain]",
		Short: "Query agents registered in a specific domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.AgentsByDomain(cmd.Context(), &types.QueryAgentsByDomainRequest{Domain: args[0]})
			if err != nil {
				return fmt.Errorf("failed to query agents by domain: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryAgentsByCapabilityCmd returns the command to query agents by capability.
func NewQueryAgentsByCapabilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents-by-capability [capability-type]",
		Short: "Query agents with a specific capability",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			qc := newQueryClient(clientCtx)
			res, err := qc.AgentsByCapability(cmd.Context(), &types.QueryAgentsByCapabilityRequest{Capability: args[0]})
			if err != nil {
				return fmt.Errorf("failed to query agents by capability: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
