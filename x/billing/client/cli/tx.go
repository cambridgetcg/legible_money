package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/billing/types"
)

// NewTxCmd returns the transaction commands for the billing module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Billing module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewRegisterProviderCmd(),
		NewDeregisterProviderCmd(),
		NewRequestQuoteCmd(),
		NewExecutePaymentCmd(),
	)

	return txCmd
}

// NewRegisterProviderCmd creates a CLI command for MsgRegisterProvider.
func NewRegisterProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-provider [stake-amount] [domain1,domain2,...]",
		Short: "Register as a knowledge API provider",
		Long: `Register as a knowledge API provider by staking tokens and
declaring which knowledge domains you serve.

Minimum stake: 100,000,000 ulgm (100 LGM).

Note: execute-payment requires --gas 500000 (default 200k insufficient
due to WritePerByte overhead for fact access recording).

Example:
  legbled tx billing register-provider 100000000 logic,mathematics --from mykey`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			domains := strings.Split(args[1], ",")
			for i := range domains {
				domains[i] = strings.TrimSpace(domains[i])
			}

			msg := &types.MsgRegisterProvider{
				Sender:  clientCtx.GetFromAddress().String(),
				Stake:   args[0],
				Domains: domains,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewDeregisterProviderCmd creates a CLI command for MsgDeregisterProvider.
func NewDeregisterProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deregister-provider",
		Short: "Deregister as a knowledge API provider and refund stake",
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

// NewRequestQuoteCmd creates a CLI command for MsgRequestQuote.
func NewRequestQuoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "request-quote [provider-address] [fact-id1,fact-id2,...]",
		Short: "Request a price quote for querying facts",
		Long: `Request a price quote from a knowledge API provider for
accessing specific facts.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			factIDs := strings.Split(args[1], ",")
			for i := range factIDs {
				factIDs[i] = strings.TrimSpace(factIDs[i])
			}

			msg := &types.MsgRequestQuote{
				Sender:   clientCtx.GetFromAddress().String(),
				Provider: args[0],
				FactIds:  factIDs,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewExecutePaymentCmd creates a CLI command for MsgExecutePayment.
func NewExecutePaymentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute-payment [quote-id]",
		Short: "Execute a quoted knowledge payment",
		Long: `Execute a previously obtained quote to pay for knowledge access.
The quote must still be within its validity window.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgExecutePayment{
				Sender:  clientCtx.GetFromAddress().String(),
				QuoteId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
