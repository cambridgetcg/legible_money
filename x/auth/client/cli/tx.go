package cli

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/auth/types"
)

// NewTxCmd returns the transaction commands for the legible_auth module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Legible auth module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewRegisterAccountCmd(),
		NewRotateKeyCmd(),
		NewCreateSessionCmd(),
		NewRevokeSessionCmd(),
		NewFreezeAccountCmd(),
		NewUnfreezeAccountCmd(),
		NewSetRecoveryConfigCmd(),
		NewInitiateRecoveryCmd(),
		NewSubmitRecoveryShardCmd(),
		NewChallengeRecoveryCmd(),
		NewExecuteRecoveryCmd(),
	)

	return txCmd
}

// NewRegisterAccountCmd creates a CLI command for MsgRegisterAccount.
func NewRegisterAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-account [did] [public-key] [account-type]",
		Short: "Register a new Legible account with DID mapping",
		Long: `Register a new Legible account with a DID (Decentralized Identifier).

DID format: did:lgm:{32-hex-chars}
Public key: 64 hex characters (Ed25519)
Account types: agent, human, contract, system`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			opKeyHash, _ := cmd.Flags().GetString("operational-key-hash")

			msg := &types.MsgRegisterAccount{
				Sender:             clientCtx.GetFromAddress().String(),
				Did:                args[0],
				PublicKey:          args[1],
				AccountType:        args[2],
				OperationalKeyHash: opKeyHash,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("operational-key-hash", "", "Initial operational key hash")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRotateKeyCmd creates a CLI command for MsgRotateKey.
func NewRotateKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotate-key [new-operational-key-hex] [authorization-signature-hex]",
		Short: "Rotate an agent's operational key",
		Long: `Rotate the operational key for a Legible account.
Requires a cooldown period between rotations.

Both arguments are hex-encoded byte strings.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			newKey, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("invalid hex for new-operational-key: %w", err)
			}

			authSig, err := hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("invalid hex for authorization-signature: %w", err)
			}

			msg := &types.MsgRotateKey{
				Sender:                 clientCtx.GetFromAddress().String(),
				NewOperationalKey:      newKey,
				AuthorizationSignature: authSig,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// validPresets lists the recognized capability preset names.
var validPresets = []string{
	"knowledge-worker",
	"partnership-operator",
	"researcher",
	"autonomous-agent",
	"dispute-arbiter",
}

// NewCreateSessionCmd creates a CLI command for MsgCreateSession.
func NewCreateSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-session [session-pub-key-hex]",
		Short: "Create a session key with limited capabilities",
		Long: `Create a session key for delegated operations.

The session public key is a hex-encoded Ed25519 key.
Capabilities are controlled via boolean flags or --preset.

Available presets:
  knowledge-worker       Submit claims, vote in governance
  partnership-operator   Partnership deposit, propose, approve, safety freeze
  researcher             Submit research, fund, create bounties
  autonomous-agent       Full operational autonomy (claims, vote, partnerships, research, disputes, transfer)
  dispute-arbiter        Dispute participation and voting

Presets can be combined with individual flags (flags override preset defaults).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			sessionKey, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("invalid hex for session-pub-key: %w", err)
			}

			expiresAt, _ := cmd.Flags().GetUint64("expires-at")

			// Start with preset if specified
			caps := &types.SessionCapabilities{}
			preset, _ := cmd.Flags().GetString("preset")
			if preset != "" {
				presetCaps := resolvePresetCaps(preset)
				if presetCaps == nil {
					return fmt.Errorf("unknown preset %q, valid presets: %s", preset, strings.Join(validPresets, ", "))
				}
				caps = presetCaps
			}

			// Override with explicit flags (only if the flag was actually set)
			if cmd.Flags().Changed("can-transfer") {
				caps.CanTransfer, _ = cmd.Flags().GetBool("can-transfer")
			}
			if cmd.Flags().Changed("can-stake") {
				caps.CanStake, _ = cmd.Flags().GetBool("can-stake")
			}
			if cmd.Flags().Changed("can-submit-claims") {
				caps.CanSubmitClaims, _ = cmd.Flags().GetBool("can-submit-claims")
			}
			if cmd.Flags().Changed("can-vote") {
				caps.CanVote, _ = cmd.Flags().GetBool("can-vote")
			}
			if cmd.Flags().Changed("can-partnership") {
				caps.CanPartnership, _ = cmd.Flags().GetBool("can-partnership")
			}
			if cmd.Flags().Changed("can-research") {
				caps.CanResearch, _ = cmd.Flags().GetBool("can-research")
			}
			if cmd.Flags().Changed("can-dispute") {
				caps.CanDispute, _ = cmd.Flags().GetBool("can-dispute")
			}

			msg := &types.MsgCreateSession{
				Owner:         clientCtx.GetFromAddress().String(),
				SessionPubKey: sessionKey,
				Capabilities:  caps,
				ExpiresAtHeight: expiresAt,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("preset", "", "Capability preset (knowledge-worker, partnership-operator, researcher, autonomous-agent, dispute-arbiter)")
	cmd.Flags().Bool("can-transfer", false, "Allow transfers")
	cmd.Flags().Bool("can-stake", false, "Allow staking")
	cmd.Flags().Bool("can-submit-claims", false, "Allow submitting claims")
	cmd.Flags().Bool("can-vote", false, "Allow voting")
	cmd.Flags().Bool("can-partnership", false, "Allow partnership operations")
	cmd.Flags().Bool("can-research", false, "Allow research operations")
	cmd.Flags().Bool("can-dispute", false, "Allow dispute operations")
	cmd.Flags().Uint64("expires-at", 0, "Block height at which session expires")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// resolvePresetCaps returns a copy of the SessionCapabilities for a preset name.
// Defined here to avoid circular import with app package.
func resolvePresetCaps(name string) *types.SessionCapabilities {
	switch name {
	case "knowledge-worker":
		return &types.SessionCapabilities{
			CanSubmitClaims: true,
			CanVote:         true,
		}
	case "partnership-operator":
		return &types.SessionCapabilities{
			CanPartnership: true,
		}
	case "researcher":
		return &types.SessionCapabilities{
			CanResearch: true,
		}
	case "autonomous-agent":
		return &types.SessionCapabilities{
			CanSubmitClaims: true,
			CanVote:         true,
			CanPartnership:  true,
			CanResearch:     true,
			CanDispute:      true,
			CanTransfer:     true,
		}
	case "dispute-arbiter":
		return &types.SessionCapabilities{
			CanDispute: true,
			CanVote:    true,
		}
	default:
		return nil
	}
}

// NewRevokeSessionCmd creates a CLI command for MsgRevokeSession.
func NewRevokeSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-session [session-id]",
		Short: "Revoke an existing session key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRevokeSession{
				Owner:     clientCtx.GetFromAddress().String(),
				SessionId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewFreezeAccountCmd creates a CLI command for MsgFreezeAccount.
func NewFreezeAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "freeze-account [address]",
		Short: "Freeze an account (self-freeze or authority)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			reason, _ := cmd.Flags().GetString("reason")

			msg := &types.MsgFreezeAccount{
				Sender:  clientCtx.GetFromAddress().String(),
				Address: args[0],
				Reason:  reason,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("reason", "", "Reason for freezing")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUnfreezeAccountCmd creates a CLI command for MsgUnfreezeAccount.
func NewUnfreezeAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unfreeze-account [address]",
		Short: "Unfreeze a frozen account (authority only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUnfreezeAccount{
				Authority: clientCtx.GetFromAddress().String(),
				Address:   args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSetRecoveryConfigCmd creates a CLI command for MsgSetRecoveryConfig.
func NewSetRecoveryConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-recovery-config [threshold] [total-shards] [shard-holder-addresses...]",
		Short: "Set recovery configuration for an account",
		Long: `Set the recovery configuration including Shamir shard parameters.

threshold: minimum shards needed for recovery
total-shards: total number of shards
shard-holder-addresses: bech32 addresses of shard holders (space-separated)`,
		Args: cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			threshold, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid threshold: %w", err)
			}

			totalShards, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid total-shards: %w", err)
			}

			holderAddrs := args[2:]
			holders := make([]*types.ShardHolder, len(holderAddrs))
			for i, addr := range holderAddrs {
				holders[i] = &types.ShardHolder{
					Type:                "address",
					Identifier:          addr,
					ShardIndex:          uint32(i + 1),
					CanInitiateRecovery: true,
				}
			}

			msg := &types.MsgSetRecoveryConfig{
				Sender: clientCtx.GetFromAddress().String(),
				Config: &types.RecoveryConfig{
					Threshold:    uint32(threshold),
					TotalShards:  uint32(totalShards),
					ShardHolders: holders,
				},
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewInitiateRecoveryCmd creates a CLI command for MsgInitiateRecovery.
func NewInitiateRecoveryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "initiate-recovery [account-address] [new-operational-key]",
		Short: "Initiate account recovery with a new operational key",
		Long: `Begin account recovery. The new operational key must be a 64-character
hex-encoded Ed25519 public key. Recovery has a delay and challenge period.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgInitiateRecovery{
				Sender:            clientCtx.GetFromAddress().String(),
				AccountAddress:    args[0],
				NewOperationalKey: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSubmitRecoveryShardCmd creates a CLI command for MsgSubmitRecoveryShard.
func NewSubmitRecoveryShardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-recovery-shard [account-address] [shard-index]",
		Short: "Submit an encrypted recovery shard",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			shardIndex, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid shard-index: %w", err)
			}

			msg := &types.MsgSubmitRecoveryShard{
				Sender:         clientCtx.GetFromAddress().String(),
				AccountAddress: args[0],
				ShardIndex:     uint32(shardIndex),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewChallengeRecoveryCmd creates a CLI command for MsgChallengeRecovery.
func NewChallengeRecoveryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "challenge-recovery [account-address] [reason]",
		Short: "Challenge an active recovery request",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgChallengeRecovery{
				Sender:         clientCtx.GetFromAddress().String(),
				AccountAddress: args[0],
				Reason:         args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewExecuteRecoveryCmd creates a CLI command for MsgExecuteRecovery.
func NewExecuteRecoveryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute-recovery [account-address]",
		Short: "Execute a recovery after delay and challenge period",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgExecuteRecovery{
				Sender:         clientCtx.GetFromAddress().String(),
				AccountAddress: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
