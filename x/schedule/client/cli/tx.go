package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/schedule/types"
)

// NewTxCmd returns the transaction commands for the schedule module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Schedule module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewCreateScheduleCmd(),
		NewPauseScheduleCmd(),
		NewResumeScheduleCmd(),
		NewCancelScheduleCmd(),
		NewAddFeeCmd(),
	)

	return txCmd
}

// NewCreateScheduleCmd creates a CLI command for MsgCreateSchedule.
func NewCreateScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-schedule [target-address] [fee-per-execution] [prepaid-fee]",
		Short: "Create a new scheduled process",
		Long: `Create a new scheduled process that executes a contract call
at specified intervals or conditions.

Use flags to configure execution parameters and conditions.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			maxExec, _ := cmd.Flags().GetUint64("max-executions")
			callData, _ := cmd.Flags().GetString("call-data")
			transferValue, _ := cmd.Flags().GetString("transfer-value")
			linkedType, _ := cmd.Flags().GetString("linked-entity-type")
			linkedID, _ := cmd.Flags().GetString("linked-entity-id")

			msg := &types.MsgCreateSchedule{
				Creator:          clientCtx.GetFromAddress().String(),
				TargetAddress:    args[0],
				FeePerExecution:  args[1],
				PrepaidFee:       args[2],
				MaxExecutions:    maxExec,
				CallData:         callData,
				TransferValue:    transferValue,
				LinkedEntityType: linkedType,
				LinkedEntityId:   linkedID,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Uint64("max-executions", 0, "Maximum number of executions (0 = unlimited)")
	cmd.Flags().String("call-data", "", "Call data for the target contract")
	cmd.Flags().String("transfer-value", "0", "LGM value to transfer per execution (ulgm)")
	cmd.Flags().String("linked-entity-type", "", "Linked entity type (e.g., fact, contract)")
	cmd.Flags().String("linked-entity-id", "", "Linked entity ID")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewPauseScheduleCmd creates a CLI command for MsgPauseSchedule.
func NewPauseScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause-schedule [process-id]",
		Short: "Pause an active scheduled process",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgPauseSchedule{
				Creator:   clientCtx.GetFromAddress().String(),
				ProcessId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewResumeScheduleCmd creates a CLI command for MsgResumeSchedule.
func NewResumeScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume-schedule [process-id]",
		Short: "Resume a paused scheduled process",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgResumeSchedule{
				Creator:   clientCtx.GetFromAddress().String(),
				ProcessId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCancelScheduleCmd creates a CLI command for MsgCancelSchedule.
func NewCancelScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-schedule [process-id]",
		Short: "Cancel a scheduled process and refund remaining fee",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgCancelSchedule{
				Creator:   clientCtx.GetFromAddress().String(),
				ProcessId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewAddFeeCmd creates a CLI command for MsgAddFee.
func NewAddFeeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-fee [process-id] [amount]",
		Short: "Add prepaid fee to an existing schedule",
		Long: `Add additional prepaid fee (ulgm) to keep a scheduled process running.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgAddFee{
				Creator:   clientCtx.GetFromAddress().String(),
				ProcessId: args[0],
				Amount:    args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
