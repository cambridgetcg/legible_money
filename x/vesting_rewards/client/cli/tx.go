package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/vesting_rewards/types"
)

// NewTxCmd returns the transaction commands for the vesting_rewards module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Vesting rewards module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewClaimVestingCmd(),
		NewCompleteVestingCmd(),
		NewCreateVestingCmd(),
		NewFalsifyVestingCmd(),
		NewPauseVestingCmd(),
		NewResumeVestingCmd(),
		NewAccelerateVestingCmd(),
	)

	return txCmd
}

// NewClaimVestingCmd creates a CLI command for MsgClaimVesting.
func NewClaimVestingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-vesting [vesting-id]",
		Short: "Claim available vested rewards (omit vesting-id to claim all)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var vestingIds []string
			if len(args) > 0 && args[0] != "" {
				vestingIds = []string{args[0]}
			}

			msg := &types.MsgClaimVesting{
				Claimer:    clientCtx.GetFromAddress().String(),
				VestingIds: vestingIds,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCompleteVestingCmd creates a CLI command for MsgCompleteVesting.
func NewCompleteVestingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "complete-vesting [vesting-id]",
		Short: "Mark a vesting schedule as completed (recipient or authority)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgCompleteVesting{
				Authority: clientCtx.GetFromAddress().String(),
				VestingId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCreateVestingCmd creates a CLI command for MsgCreateVesting.
func NewCreateVestingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-vesting [beneficiary] [amount] [linked-fact-id]",
		Short: "Create a new truth-linked vesting schedule (authority only)",
		Long: `Create a new truth-linked vesting schedule. Only callable by module authority (governance).

Category values (--category flag):
  0 = unspecified
  1 = verification_reward
  2 = block_reward
  3 = bounty_reward
  4 = dispute_reward
  5 = research_grant
  6 = bootstrap`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			category, err := cmd.Flags().GetUint32("category")
			if err != nil {
				return fmt.Errorf("invalid category: %w", err)
			}
			if category > 6 {
				return fmt.Errorf("invalid category %d: must be 0-6", category)
			}

			startHeight, err := cmd.Flags().GetUint64("start-height")
			if err != nil {
				return fmt.Errorf("invalid start-height: %w", err)
			}

			msg := &types.MsgCreateVesting{
				Authority:    clientCtx.GetFromAddress().String(),
				Beneficiary:  args[0],
				Amount:       args[1],
				LinkedFactId: args[2],
				Category:     types.VestingCategory(category),
				StartHeight:  startHeight,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Uint32("category", 0, "Vesting category (0-6: unspecified, verification_reward, block_reward, bounty_reward, dispute_reward, research_grant, bootstrap)")
	cmd.Flags().Uint64("start-height", 0, "Block height at which vesting begins (0 for current height)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewFalsifyVestingCmd creates a CLI command for MsgFalsifyVesting.
func NewFalsifyVestingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "falsify-vesting [vesting-id] [reason]",
		Short: "Challenge a vesting schedule with falsification evidence (authority only)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			counterEvidenceHash, _ := cmd.Flags().GetString("counter-evidence-hash")

			msg := &types.MsgFalsifyVesting{
				Challenger:          clientCtx.GetFromAddress().String(),
				VestingId:           args[0],
				Reason:              args[1],
				CounterEvidenceHash: counterEvidenceHash,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("counter-evidence-hash", "", "Hash of counter-evidence data")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewPauseVestingCmd creates a CLI command for MsgPauseVesting.
func NewPauseVestingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause-vesting [vesting-id] [reason]",
		Short: "Pause a vesting schedule due to active challenge (authority only)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgPauseVesting{
				Authority: clientCtx.GetFromAddress().String(),
				VestingId: args[0],
				Reason:    args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewResumeVestingCmd creates a CLI command for MsgResumeVesting.
func NewResumeVestingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume-vesting [vesting-id]",
		Short: "Resume a paused vesting schedule (authority only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgResumeVesting{
				Authority: clientCtx.GetFromAddress().String(),
				VestingId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewAccelerateVestingCmd creates a CLI command for MsgAccelerateVesting.
func NewAccelerateVestingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accelerate-vesting [vesting-id] [acceleration-factor]",
		Short: "Accelerate a vesting schedule with evidence (authority only)",
		Long: `Accelerate a vesting schedule. Only callable by module authority.

The acceleration-factor is a uint32 multiplier (0-1000000) that maps to acceleration types:
  0-250000     = citation
  250001-500000 = corroboration
  500001-750000 = replication
  750001+       = defense`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			factor, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid acceleration-factor: %w", err)
			}

			msg := &types.MsgAccelerateVesting{
				Authority:          clientCtx.GetFromAddress().String(),
				VestingId:          args[0],
				AccelerationFactor: uint32(factor),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
