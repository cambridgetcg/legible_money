package cli

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/bvm/types"
)

// NewTxCmd returns the transaction commands for the bvm module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "BVM module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewDeployContractCmd(),
		NewCallContractCmd(),
		NewScheduleContractCmd(),
		NewCancelScheduleCmd(),
	)

	return txCmd
}

// NewDeployContractCmd creates a CLI command for MsgDeployContract.
func NewDeployContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy-contract [bytecode-hex-or-file]",
		Short: "Deploy a new BVM contract",
		Long: `Deploy a new contract to the Belief Virtual Machine.

The bytecode can be provided as a hex string or as a path to a file
containing hex-encoded bytecode (prefix with @).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var bytecode []byte
			if strings.HasPrefix(args[0], "@") {
				data, err := os.ReadFile(args[0][1:])
				if err != nil {
					return fmt.Errorf("failed to read bytecode file: %w", err)
				}
				bytecode, err = hex.DecodeString(strings.TrimSpace(string(data)))
				if err != nil {
					return fmt.Errorf("invalid hex in bytecode file: %w", err)
				}
			} else {
				bytecode, err = hex.DecodeString(args[0])
				if err != nil {
					return fmt.Errorf("invalid bytecode hex: %w", err)
				}
			}

			deposit, _ := cmd.Flags().GetString("deposit")
			ctorArgs, _ := cmd.Flags().GetString("constructor-args")

			var ctorBytes []byte
			if ctorArgs != "" {
				ctorBytes, err = hex.DecodeString(ctorArgs)
				if err != nil {
					return fmt.Errorf("invalid constructor-args hex: %w", err)
				}
			}

			msg := &types.MsgDeployContract{
				Deployer:        clientCtx.GetFromAddress().String(),
				Bytecode:        bytecode,
				ConstructorArgs: ctorBytes,
				InitialDeposit:  deposit,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("deposit", "0", "Initial deposit in ulgm")
	cmd.Flags().String("constructor-args", "", "Constructor arguments as hex")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewCallContractCmd creates a CLI command for MsgCallContract.
func NewCallContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call-contract [contract-address] [input-data-hex]",
		Short: "Call a deployed BVM contract",
		Long: `Execute a contract call in the Belief Virtual Machine.

Input data should be hex-encoded. Use --value to transfer LGM with the call.
Use --static for read-only calls that cannot modify state.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			inputData, err := hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("invalid input data hex: %w", err)
			}

			gasLimit, _ := cmd.Flags().GetUint64("gas-limit")
			value, _ := cmd.Flags().GetString("value")
			staticCall, _ := cmd.Flags().GetBool("static")

			msg := &types.MsgCallContract{
				Caller:          clientCtx.GetFromAddress().String(),
				ContractAddress: args[0],
				InputData:       inputData,
				Value:           value,
				GasLimit:        gasLimit,
				StaticCall:      staticCall,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Uint64("gas-limit", 100000, "Gas limit for the contract call")
	cmd.Flags().String("value", "0", "LGM value to transfer with the call (ulgm)")
	cmd.Flags().Bool("static", false, "Static call (read-only, no state changes)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewScheduleContractCmd creates a CLI command for MsgScheduleContract.
func NewScheduleContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule-contract [contract-address] [method] [execute-at-block]",
		Short: "Schedule a future contract execution",
		Long: `Schedule a contract method to execute at a specific future block height.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			block, err := parseUint64(args[2])
			if err != nil {
				return fmt.Errorf("invalid block height: %w", err)
			}

			payload, _ := cmd.Flags().GetString("payload")
			maxGas, _ := cmd.Flags().GetUint64("max-gas")

			msg := &types.MsgScheduleContract{
				Caller:          clientCtx.GetFromAddress().String(),
				ContractAddress: args[0],
				Method:          args[1],
				Payload:         payload,
				ExecuteAtBlock:  block,
				MaxGas:          maxGas,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("payload", "", "Payload data for the method call")
	cmd.Flags().Uint64("max-gas", 100000, "Maximum gas for the scheduled execution")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewCancelScheduleCmd creates a CLI command for MsgCancelSchedule.
func NewCancelScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-schedule [schedule-id]",
		Short: "Cancel a pending scheduled execution",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgCancelSchedule{
				Caller:     clientCtx.GetFromAddress().String(),
				ScheduleId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// parseUint64 parses a string to uint64.
func parseUint64(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	var n uint64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
