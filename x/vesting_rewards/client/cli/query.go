package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/vesting_rewards/types"
)

// NewQueryCmd returns the query commands for the vesting_rewards module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Vesting rewards module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryVestingScheduleCmd(),
		NewQueryVestingSchedulesByRecipientCmd(),
		NewQueryActiveVestingSchedulesCmd(),
		NewQueryBlockRewardDistributionCmd(),
		NewQueryParamsCmd(),
	)

	return queryCmd
}

// NewQueryVestingScheduleCmd returns the command to query a single vesting schedule.
func NewQueryVestingScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule [vesting-id]",
		Short: "Query a vesting schedule by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryVestingScheduleRequest{VestingId: args[0]}
			res := &types.QueryVestingScheduleResponse{}
			if err := clientCtx.Invoke(cmd.Context(),
				"/legible.vesting_rewards.v1.Query/VestingSchedule",
				req, res,
			); err != nil {
				return fmt.Errorf("failed to query vesting schedule: %w", err)
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

// NewQueryVestingSchedulesByRecipientCmd returns the command to query vesting schedules by recipient.
func NewQueryVestingSchedulesByRecipientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedules-by-recipient [recipient-address]",
		Short: "Query all vesting schedules for a recipient",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryVestingSchedulesByRecipientRequest{Recipient: args[0]}
			res := &types.QueryVestingSchedulesByRecipientResponse{}
			if err := clientCtx.Invoke(cmd.Context(),
				"/legible.vesting_rewards.v1.Query/VestingSchedulesByRecipient",
				req, res,
			); err != nil {
				return fmt.Errorf("failed to query vesting schedules: %w", err)
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

// NewQueryActiveVestingSchedulesCmd returns the command to list active vesting schedules.
func NewQueryActiveVestingSchedulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "active-schedules",
		Short: "List all active vesting schedules",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetUint32("limit")
			offset, _ := cmd.Flags().GetUint32("offset")

			req := &types.QueryActiveVestingSchedulesRequest{
				Limit:  limit,
				Offset: offset,
			}
			res := &types.QueryActiveVestingSchedulesResponse{}
			if err := clientCtx.Invoke(cmd.Context(),
				"/legible.vesting_rewards.v1.Query/ActiveVestingSchedules",
				req, res,
			); err != nil {
				return fmt.Errorf("failed to query active vesting schedules: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return err
		},
	}

	cmd.Flags().Uint32("limit", 20, "Maximum number of schedules to return")
	cmd.Flags().Uint32("offset", 0, "Number of schedules to skip")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryBlockRewardDistributionCmd returns the command to query block reward distribution.
func NewQueryBlockRewardDistributionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block-reward [height]",
		Short: "Query block reward distribution at a specific height",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			height, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid block height: %w", err)
			}

			req := &types.QueryBlockRewardDistributionRequest{BlockHeight: height}
			res := &types.QueryBlockRewardDistributionResponse{}
			if err := clientCtx.Invoke(cmd.Context(),
				"/legible.vesting_rewards.v1.Query/BlockRewardDistribution",
				req, res,
			); err != nil {
				return fmt.Errorf("failed to query block reward distribution: %w", err)
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

// NewQueryParamsCmd returns the command to query vesting_rewards module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the vesting rewards module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			res := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(),
				"/legible.vesting_rewards.v1.Query/Params",
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
