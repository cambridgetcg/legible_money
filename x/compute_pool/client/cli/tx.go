package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/compute_pool/types"
)

// NewTxCmd returns the transaction commands for the compute_pool module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Compute pool module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewRegisterComputeProviderCmd(),
		NewUpdateProviderCmd(),
		NewDeregisterProviderCmd(),
		NewProviderHeartbeatCmd(),
		NewRedeemCreditsCmd(),
		NewConfirmUsageCmd(),
		NewDisputeUsageCmd(),
	)

	return txCmd
}

// NewRegisterComputeProviderCmd creates a CLI command for MsgRegisterComputeProvider.
func NewRegisterComputeProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-provider [service-type] [endpoint] [price-per-cu] [stake-amount]",
		Short: "Register as a compute infrastructure provider",
		Long: `Register as a compute infrastructure provider.

Service types: ai_inference, gpu_compute, storage
Price and stake are in ulgm.`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			did, _ := cmd.Flags().GetString("did")

			msg := &types.MsgRegisterComputeProvider{
				Sender:      clientCtx.GetFromAddress().String(),
				ServiceType: args[0],
				Endpoint:    args[1],
				PricePerCu:  args[2],
				StakeAmount: args[3],
				Did:         did,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("did", "", "DID identifier for the provider")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewUpdateProviderCmd creates a CLI command for MsgUpdateProvider.
func NewUpdateProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-provider",
		Short: "Update compute provider configuration",
		Long: `Update endpoint, price, or pause/unpause a compute provider.
Use flags to specify which fields to update.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			endpoint, _ := cmd.Flags().GetString("endpoint")
			price, _ := cmd.Flags().GetString("price")
			pause, _ := cmd.Flags().GetBool("pause")
			unpause, _ := cmd.Flags().GetBool("unpause")

			msg := &types.MsgUpdateProvider{
				Sender:   clientCtx.GetFromAddress().String(),
				Endpoint: endpoint,
				Price:    price,
				Pause:    pause,
				Unpause:  unpause,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("endpoint", "", "New endpoint URL")
	cmd.Flags().String("price", "", "New price per compute unit (ulgm)")
	cmd.Flags().Bool("pause", false, "Pause the provider")
	cmd.Flags().Bool("unpause", false, "Unpause the provider")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewDeregisterProviderCmd creates a CLI command for MsgDeregisterProvider.
func NewDeregisterProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deregister-provider",
		Short: "Deregister as a compute provider and begin unbonding",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDeregisterProvider{
				Sender: clientCtx.GetFromAddress().String(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewProviderHeartbeatCmd creates a CLI command for MsgProviderHeartbeat.
func NewProviderHeartbeatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "heartbeat",
		Short: "Send a provider liveness heartbeat",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgProviderHeartbeat{
				Sender: clientCtx.GetFromAddress().String(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRedeemCreditsCmd creates a CLI command for MsgRedeemCredits.
func NewRedeemCreditsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redeem-credits [provider-address] [cu-amount]",
		Short: "Redeem compute credits against a provider",
		Long: `Spend compute credits to purchase compute units from a provider.
Creates a redemption ticket that must be confirmed after service delivery.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			cuAmount, err := parseUint64(args[1])
			if err != nil {
				return fmt.Errorf("invalid cu-amount: %w", err)
			}

			msg := &types.MsgRedeemCredits{
				Sender:   clientCtx.GetFromAddress().String(),
				Provider: args[0],
				CuAmount: cuAmount,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewConfirmUsageCmd creates a CLI command for MsgConfirmUsage.
func NewConfirmUsageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "confirm-usage [ticket-id] [actual-cu]",
		Short: "Confirm that compute was delivered for a redemption ticket",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			ticketID, err := parseUint64(args[0])
			if err != nil {
				return fmt.Errorf("invalid ticket-id: %w", err)
			}

			actualCU, err := parseUint64(args[1])
			if err != nil {
				return fmt.Errorf("invalid actual-cu: %w", err)
			}

			msg := &types.MsgConfirmUsage{
				Sender:   clientCtx.GetFromAddress().String(),
				TicketId: ticketID,
				ActualCu: actualCU,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewDisputeUsageCmd creates a CLI command for MsgDisputeUsage.
func NewDisputeUsageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dispute-usage [ticket-id] [reason]",
		Short: "Dispute a redemption ticket (compute not delivered)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			ticketID, err := parseUint64(args[0])
			if err != nil {
				return fmt.Errorf("invalid ticket-id: %w", err)
			}

			msg := &types.MsgDisputeUsage{
				Sender:   clientCtx.GetFromAddress().String(),
				TicketId: ticketID,
				Reason:   args[1],
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
