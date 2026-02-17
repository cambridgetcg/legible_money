package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/home/types"
)

// NewTxCmd returns the transaction commands for the home module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Home module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewCreateHomeCmd(),
		NewUpdateHomeCmd(),
		NewUpdateMemoryCmd(),
		NewConfigureGuardianCmd(),
		NewRegisterKeyCmd(),
		NewRevokeKeyCmd(),
		NewStartSessionCmd(),
		NewEndSessionCmd(),
		NewSetSpendingLimitCmd(),
		NewAckAlertCmd(),
	)

	return txCmd
}

// NewCreateHomeCmd creates a CLI command for MsgCreateHome.
func NewCreateHomeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-home [name]",
		Short: "Create a new agent home",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgCreateHome{
				Owner: clientCtx.GetFromAddress().String(),
				Name:  args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdateHomeCmd creates a CLI command for MsgUpdateHome.
func NewUpdateHomeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-home [home-id]",
		Short: "Update a home's name or status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			name, _ := cmd.Flags().GetString("name")
			status, _ := cmd.Flags().GetString("status")

			msg := &types.MsgUpdateHome{
				Owner:  clientCtx.GetFromAddress().String(),
				HomeId: args[0],
				Name:   name,
				Status: status,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("name", "", "New home name")
	cmd.Flags().String("status", "", "New status (active, dormant, guarded, recovery, archived)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdateMemoryCmd creates a CLI command for MsgUpdateMemoryCID.
func NewUpdateMemoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-memory [home-id] [cid]",
		Short: "Update a home's off-chain memory CID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUpdateMemoryCID{
				Owner:  clientCtx.GetFromAddress().String(),
				HomeId: args[0],
				Cid:    args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewConfigureGuardianCmd creates a CLI command for MsgConfigureGuardian.
func NewConfigureGuardianCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure-guardian [home-id] [strategy]",
		Short: "Configure guardian defense settings",
		Long: `Configure the guardian for an agent home.

Strategies: aggressive, moderate, conservative, diplomatic`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			strategy := args[1]
			autoDefend, _ := cmd.Flags().GetBool("auto-defend")
			deadmanThreshold, _ := cmd.Flags().GetUint64("deadman-threshold")
			deadmanAction, _ := cmd.Flags().GetString("deadman-action")

			msg := &types.MsgConfigureGuardian{
				Owner:           clientCtx.GetFromAddress().String(),
				HomeId:          args[0],
				DefenseStrategy: strategy,
				AutoDefend:      autoDefend,
			}

			if deadmanThreshold > 0 {
				msg.Deadman = &types.DeadmanConfig{
					Enabled:             true,
					InactivityThreshold: deadmanThreshold,
					Action:              deadmanAction,
				}
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Bool("auto-defend", true, "Enable automatic defense")
	cmd.Flags().Uint64("deadman-threshold", 0, "Deadman inactivity threshold in blocks (0 = disabled)")
	cmd.Flags().String("deadman-action", "alert_guardians", "Deadman action (alert_guardians, transfer_to_beneficiary, donate_to_research)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRegisterKeyCmd creates a CLI command for MsgRegisterKey.
func NewRegisterKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-key [home-id] [key-hash] [key-type]",
		Short: "Register a key with the home",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			role, _ := cmd.Flags().GetString("role")
			perms, _ := cmd.Flags().GetStringSlice("permissions")
			expiresAt, _ := cmd.Flags().GetUint64("expires-at")

			msg := &types.MsgRegisterKey{
				Owner:       clientCtx.GetFromAddress().String(),
				HomeId:      args[0],
				KeyHash:     args[1],
				KeyType:     args[2],
				Role:        role,
				Permissions: perms,
				ExpiresAt:   expiresAt,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("role", "operator", "Key role (owner, operator, guest, guardian, viewer)")
	cmd.Flags().StringSlice("permissions", nil, "Permissions (comma-separated)")
	cmd.Flags().Uint64("expires-at", 0, "Block height at which key expires (0 = never)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRevokeKeyCmd creates a CLI command for MsgRevokeKey.
func NewRevokeKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-key [home-id] [key-hash]",
		Short: "Revoke a key from the home",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRevokeKey{
				Owner:   clientCtx.GetFromAddress().String(),
				HomeId:  args[0],
				KeyHash: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewStartSessionCmd creates a CLI command for MsgStartSession.
func NewStartSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start-session [home-id] [key-hash]",
		Short: "Start a session on a home",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			perms, _ := cmd.Flags().GetStringSlice("permissions")

			msg := &types.MsgStartSession{
				Signer:               clientCtx.GetFromAddress().String(),
				HomeId:               args[0],
				KeyHash:              args[1],
				RequestedPermissions: perms,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().StringSlice("permissions", nil, "Requested permissions (comma-separated)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewEndSessionCmd creates a CLI command for MsgEndSession.
func NewEndSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "end-session [home-id] [session-id]",
		Short: "End a session",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgEndSession{
				Signer:    clientCtx.GetFromAddress().String(),
				HomeId:    args[0],
				SessionId: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSetSpendingLimitCmd creates a CLI command for MsgSetSpendingLimit.
func NewSetSpendingLimitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-spending-limit [home-id] [key-type] [max-amount] [period-blocks]",
		Short: "Set a spending limit for a key type",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			periodBlocks, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid period-blocks: %w", err)
			}

			msg := &types.MsgSetSpendingLimit{
				Owner:        clientCtx.GetFromAddress().String(),
				HomeId:       args[0],
				KeyType:      args[1],
				MaxAmount:    args[2],
				PeriodBlocks: periodBlocks,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewAckAlertCmd creates a CLI command for MsgAcknowledgeAlert.
func NewAckAlertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ack-alert [home-id] [alert-id]",
		Short: "Acknowledge an alert",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgAcknowledgeAlert{
				Signer:  clientCtx.GetFromAddress().String(),
				HomeId:  args[0],
				AlertId: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// Unused import suppressor.
var (
	_ = strings.TrimSpace
	_ = strconv.Atoi
)
