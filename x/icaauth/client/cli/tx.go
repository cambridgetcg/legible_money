package cli

import (
	"encoding/hex"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/icaauth/types"
)

// NewTxCmd returns the transaction commands for the icaauth module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "ICA auth module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewRegisterInterchainAccountCmd(),
		NewSendICATransactionCmd(),
	)

	return txCmd
}

// NewRegisterInterchainAccountCmd creates a CLI command for MsgRegisterInterchainAccount.
func NewRegisterInterchainAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register [connection-id]",
		Short: "Register an interchain account on a remote chain",
		Long: `Register an interchain account (ICA) on a remote chain via the specified
IBC connection. Must be signed by the account's operational key (not a session key).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			version, _ := cmd.Flags().GetString("version")
			allowedMsgs, _ := cmd.Flags().GetStringSlice("allowed-msgs")

			msg := &types.MsgRegisterInterchainAccount{
				Owner:        clientCtx.GetFromAddress().String(),
				ConnectionId: args[0],
				Version:      version,
				AllowedMsgs:  allowedMsgs,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("version", "", "ICA version string")
	cmd.Flags().StringSlice("allowed-msgs", nil, "Proto URL allowlist for remote messages (comma-separated)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewSendICATransactionCmd creates a CLI command for MsgSendICATransaction.
func NewSendICATransactionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-tx [connection-id] [msg-bytes-hex]",
		Short: "Send a transaction to a remote chain via ICA",
		Long: `Send a serialized CosmosTx to a remote chain through an interchain account.

The msg-bytes argument must be hex-encoded serialized CosmosTx bytes.
The timeout can be specified with the --timeout flag in nanoseconds.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msgBytes, err := hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("invalid hex-encoded msg bytes: %w", err)
			}

			timeoutNanos, _ := cmd.Flags().GetUint64("timeout")

			msg := &types.MsgSendICATransaction{
				Owner:        clientCtx.GetFromAddress().String(),
				ConnectionId: args[0],
				MsgBytes:     msgBytes,
				TimeoutNanos: timeoutNanos,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Uint64("timeout", 600000000000, "Relative timeout in nanoseconds (default 10 minutes)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
