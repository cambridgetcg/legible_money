package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/autopoiesis/types"
)

// NewTxCmd returns the transaction commands for the autopoiesis module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Autopoiesis module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewActivateAutopoiesisCmd(),
		NewOverrideMultiplierCmd(),
		NewFreezeMultiplierCmd(),
	)

	return txCmd
}

// NewActivateAutopoiesisCmd creates a CLI command for MsgActivateAutopoiesis.
func NewActivateAutopoiesisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activate [active]",
		Short: "Enable or disable the autopoiesis engine (governance)",
		Long: `Enable or disable the autopoiesis self-tuning engine via governance.

active: true or false`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			active, err := strconv.ParseBool(args[0])
			if err != nil {
				return fmt.Errorf("invalid active value: must be 'true' or 'false'")
			}

			msg := &types.MsgActivateAutopoiesis{
				Authority: clientCtx.GetFromAddress().String(),
				Activate:  active,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewOverrideMultiplierCmd creates a CLI command for MsgOverrideMultiplier.
func NewOverrideMultiplierCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "override-multiplier [path] [value]",
		Short: "Manually set a specific multiplier value (governance)",
		Long: `Override a multiplier at the given path with a specific value.
Used for emergency corrections or initial calibration.

path: Multiplier path (e.g., "slash.system")
value: New multiplier value in BPS (1000000 = 1.0x)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			value, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid value: %w", err)
			}

			msg := &types.MsgOverrideMultiplier{
				Authority: clientCtx.GetFromAddress().String(),
				Path:      args[0],
				Value:     value,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewFreezeMultiplierCmd creates a CLI command for MsgFreezeMultiplier.
func NewFreezeMultiplierCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "freeze-multiplier [path] [frozen]",
		Short: "Freeze or unfreeze a multiplier (governance)",
		Long: `Freeze a specific multiplier at its current value, preventing the
autopoiesis engine from modifying it until unfrozen.

path: Multiplier path (e.g., "slash.system")
frozen: true = freeze, false = unfreeze`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			frozen, err := strconv.ParseBool(args[1])
			if err != nil {
				return fmt.Errorf("invalid frozen value: must be 'true' or 'false'")
			}

			msg := &types.MsgFreezeMultiplier{
				Authority: clientCtx.GetFromAddress().String(),
				Path:      args[0],
				Frozen:    frozen,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
