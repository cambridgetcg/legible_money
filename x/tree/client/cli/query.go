package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/tree/types"
)

// NewQueryCmd returns the query commands for the tree module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Tree module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryProjectCmd(),
		NewQueryProjectsByFounderCmd(),
		NewQueryTaskCmd(),
		NewQueryTasksByProjectCmd(),
		NewQueryServiceCmd(),
		NewQuerySeedCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query tree module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the tree module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			res := &types.QueryParamsResponse{}

			err = clientCtx.Invoke(cmd.Context(), "/legible.tree.v1.Query/Params", req, res)
			if err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}

			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryProjectCmd returns the command to query a single project.
func NewQueryProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project [project-id]",
		Short: "Query a project by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryProjectRequest{ProjectId: args[0]}
			res := &types.QueryProjectResponse{}

			err = clientCtx.Invoke(cmd.Context(), "/legible.tree.v1.Query/Project", req, res)
			if err != nil {
				return fmt.Errorf("failed to query project: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}

			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryProjectsByFounderCmd returns the command to query projects by founder.
func NewQueryProjectsByFounderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects-by-founder [founder-address]",
		Short: "Query projects created by a founder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetUint32("limit")
			offset, _ := cmd.Flags().GetUint32("offset")

			req := &types.QueryProjectsByFounderRequest{
				Founder: args[0],
				Limit:   limit,
				Offset:  offset,
			}
			res := &types.QueryProjectsByFounderResponse{}

			err = clientCtx.Invoke(cmd.Context(), "/legible.tree.v1.Query/ProjectsByFounder", req, res)
			if err != nil {
				return fmt.Errorf("failed to query projects: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}

			return clientCtx.PrintBytes(out)
		},
	}

	cmd.Flags().Uint32("limit", 20, "Maximum number of projects to return")
	cmd.Flags().Uint32("offset", 0, "Number of projects to skip")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// NewQueryTaskCmd returns the command to query a single task.
func NewQueryTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task [task-id]",
		Short: "Query a task by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryTaskRequest{TaskId: args[0]}
			res := &types.QueryTaskResponse{}

			err = clientCtx.Invoke(cmd.Context(), "/legible.tree.v1.Query/Task", req, res)
			if err != nil {
				return fmt.Errorf("failed to query task: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}

			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryTasksByProjectCmd returns the command to query tasks by project.
func NewQueryTasksByProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tasks-by-project [project-id]",
		Short: "Query tasks belonging to a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetUint32("limit")
			offset, _ := cmd.Flags().GetUint32("offset")

			req := &types.QueryTasksByProjectRequest{
				ProjectId: args[0],
				Limit:     limit,
				Offset:    offset,
			}
			res := &types.QueryTasksByProjectResponse{}

			err = clientCtx.Invoke(cmd.Context(), "/legible.tree.v1.Query/TasksByProject", req, res)
			if err != nil {
				return fmt.Errorf("failed to query tasks: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}

			return clientCtx.PrintBytes(out)
		},
	}

	cmd.Flags().Uint32("limit", 20, "Maximum number of tasks to return")
	cmd.Flags().Uint32("offset", 0, "Number of tasks to skip")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// NewQueryServiceCmd returns the command to query a single service.
func NewQueryServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service [service-id]",
		Short: "Query a service by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryServiceRequest{ServiceId: args[0]}
			res := &types.QueryServiceResponse{}

			err = clientCtx.Invoke(cmd.Context(), "/legible.tree.v1.Query/Service", req, res)
			if err != nil {
				return fmt.Errorf("failed to query service: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}

			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQuerySeedCmd returns the command to query an opportunity seed.
func NewQuerySeedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed [seed-id]",
		Short: "Query an opportunity seed by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QuerySeedRequest{SeedId: args[0]}
			res := &types.QuerySeedResponse{}

			err = clientCtx.Invoke(cmd.Context(), "/legible.tree.v1.Query/Seed", req, res)
			if err != nil {
				return fmt.Errorf("failed to query seed: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}

			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
