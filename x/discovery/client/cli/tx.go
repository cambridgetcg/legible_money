package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/discovery/types"
)

// NewTxCmd returns the transaction commands for the discovery module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Discovery module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewRegisterAgentCmd(),
		NewUpdateProfileCmd(),
		NewAddCapabilityCmd(),
		NewRemoveCapabilityCmd(),
		NewDeregisterAgentCmd(),
		NewHeartbeatCmd(),
	)

	return txCmd
}

// NewRegisterAgentCmd creates a CLI command for MsgRegisterAgent.
func NewRegisterAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-agent [display-name] [stake]",
		Short: "Register a new agent profile in the discovery directory",
		Long: `Register an agent for discovery by other agents and validators.

The agent must stake tokens as registration collateral.
Capabilities and domains can be specified with flags.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			did, _ := cmd.Flags().GetString("did")
			domains, _ := cmd.Flags().GetStringSlice("domains")
			description, _ := cmd.Flags().GetString("description")
			metadata, _ := cmd.Flags().GetString("metadata")

			msg := &types.MsgRegisterAgent{
				Address:     clientCtx.GetFromAddress().String(),
				DisplayName: args[0],
				Stake:       args[1],
				Did:         did,
				Domains:     domains,
				Description: description,
				Metadata:    metadata,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("did", "", "Agent DID (did:lgm:{hex})")
	cmd.Flags().StringSlice("domains", nil, "Knowledge domains (comma-separated)")
	cmd.Flags().String("description", "", "Agent description")
	cmd.Flags().String("metadata", "", "Additional metadata (JSON)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdateProfileCmd creates a CLI command for MsgUpdateProfile.
func NewUpdateProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-profile",
		Short: "Update an agent's profile metadata",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			displayName, _ := cmd.Flags().GetString("display-name")
			description, _ := cmd.Flags().GetString("description")
			metadata, _ := cmd.Flags().GetString("metadata")

			msg := &types.MsgUpdateProfile{
				Address:     clientCtx.GetFromAddress().String(),
				DisplayName: displayName,
				Description: description,
				Metadata:    metadata,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("display-name", "", "New display name")
	cmd.Flags().String("description", "", "New description")
	cmd.Flags().String("metadata", "", "New metadata (JSON)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewAddCapabilityCmd creates a CLI command for MsgAddCapability.
func NewAddCapabilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-capability [capability-type]",
		Short: "Add a capability to an agent's profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			domains, _ := cmd.Flags().GetStringSlice("domains")
			confidence, _ := cmd.Flags().GetUint64("confidence")

			msg := &types.MsgAddCapability{
				Address: clientCtx.GetFromAddress().String(),
				Capability: types.AgentCapability{
					CapabilityType: args[0],
					Domains:        domains,
					ConfidenceBps:  confidence,
				},
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().StringSlice("domains", nil, "Domains this capability applies to (comma-separated)")
	cmd.Flags().Uint64("confidence", 5000, "Self-reported confidence in basis points (0-10000)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRemoveCapabilityCmd creates a CLI command for MsgRemoveCapability.
func NewRemoveCapabilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-capability [capability-type]",
		Short: "Remove a capability from an agent's profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRemoveCapability{
				Address:        clientCtx.GetFromAddress().String(),
				CapabilityType: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewDeregisterAgentCmd creates a CLI command for MsgDeregisterAgent.
func NewDeregisterAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deregister-agent",
		Short: "Remove an agent profile and refund stake",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDeregisterAgent{
				Address: clientCtx.GetFromAddress().String(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewHeartbeatCmd creates a CLI command for MsgHeartbeat.
func NewHeartbeatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "heartbeat",
		Short: "Signal that an agent is still active",
		Long:  `Send a heartbeat to update last-active block and reactivate expired profiles.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgHeartbeat{
				Address: clientCtx.GetFromAddress().String(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
