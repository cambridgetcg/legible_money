package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	authtypes "github.com/legible-money/legible/x/auth/types"
	knowledgetypes "github.com/legible-money/legible/x/knowledge/types"
	partnershiptypes "github.com/legible-money/legible/x/partnerships/types"
)

// NewQuickRegisterCmd creates a convenience command for account registration.
// Auto-derives DID and public key from the keyring key specified by --from.
func NewQuickRegisterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quick-register",
		Short: "Register a Legible account (auto-derives DID from key)",
		Long: `Register a new Legible account with automatic DID derivation.

Derives the DID and public key from the keyring key specified by --from.
This is a convenience wrapper around 'legbled tx legible-auth register-account'.

Example:
  legbled tx quick-register --type human --from mykey`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			accountType, _ := cmd.Flags().GetString("type")
			if accountType == "" {
				accountType = "agent"
			}
			if accountType != "human" && accountType != "agent" && accountType != "contract" {
				return fmt.Errorf("invalid account type %q (must be human, agent, or contract)", accountType)
			}

			// Get public key from keyring
			info, err := clientCtx.Keyring.Key(clientCtx.FromName)
			if err != nil {
				return fmt.Errorf("failed to get key %q from keyring: %w", clientCtx.FromName, err)
			}

			pubKey, err := info.GetPubKey()
			if err != nil {
				return fmt.Errorf("failed to get public key: %w", err)
			}

			pubKeyHex := hex.EncodeToString(pubKey.Bytes())

			// DID = did:lgm:{first 32 hex chars of SHA-256(pubkey)}
			h := sha256.Sum256(pubKey.Bytes())
			did := "did:lgm:" + hex.EncodeToString(h[:16])

			msg := &authtypes.MsgRegisterAccount{
				Sender:      clientCtx.GetFromAddress().String(),
				Did:         did,
				PublicKey:   pubKeyHex,
				AccountType: accountType,
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Registering %s account:\n  Address: %s\n  DID:     %s\n  PubKey:  %s...\n\n",
				accountType, clientCtx.GetFromAddress().String(), did, pubKeyHex[:16])

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("type", "agent", "Account type (human, agent, contract)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewQuickPartnerCmd creates a convenience command for initiating partnerships.
func NewQuickPartnerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quick-partner [partner-address]",
		Short: "Initiate a partnership with a partner",
		Long: `Initiate a partnership formation with the specified partner address.
Defaults to lock tier 1 (Standard). Use --lock-tier to change.

Lock tiers:
  0 - Trial:     22,222 blocks, 1.00x multiplier
  1 - Standard:  77,777 blocks, 1.11x multiplier
  2 - Trusted:   222,222 blocks, 1.22x multiplier
  3 - Elite:     777,777 blocks, 1.44x multiplier
  4 - Sovereign: 1,111,111 blocks, 1.55x multiplier
  5 - Eternal:   2,222,222 blocks, 1.77x multiplier

Example:
  legbled tx quick-partner lgm1abc...def --lock-tier 2 --from mykey`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			lockTier, _ := cmd.Flags().GetUint32("lock-tier")

			msg := &partnershiptypes.MsgInitiatePartnership{
				Initiator: clientCtx.GetFromAddress().String(),
				Partner:   args[0],
				LockTier:  lockTier,
			}

			tierNames := []string{"Trial", "Standard", "Trusted", "Elite", "Sovereign", "Eternal"}
			tierName := "unknown"
			if int(lockTier) < len(tierNames) {
				tierName = tierNames[lockTier]
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Initiating partnership:\n  Partner:   %s\n  Lock Tier: %d (%s)\n\n",
				args[0], lockTier, tierName)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Uint32("lock-tier", 1, "Lock tier (0=Trial, 1=Standard, 2=Trusted, 3=Elite, 4=Sovereign, 5=Eternal)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewQuickAcceptCmd creates a convenience command for accepting partnerships.
func NewQuickAcceptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quick-accept [partnership-id]",
		Short: "Accept a pending partnership",
		Long: `Accept a pending partnership formation.

Example:
  legbled tx quick-accept partnership-1 --from agent-key`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &partnershiptypes.MsgAcceptPartnership{
				Acceptor:      clientCtx.GetFromAddress().String(),
				PartnershipId: args[0],
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Accepting partnership: %s\n\n", args[0])

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewClaimCmd creates a convenience command for submitting knowledge claims.
func NewClaimCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim [statement]",
		Short: "Submit a knowledge claim (human-friendly)",
		Long: `Submit a knowledge claim to the Proof of Truth verification pipeline.

Accepts human-readable amounts (0.1LGM, 1LGM, 100000ulgm) for the stake.
References can be specified with --refs flag (comma-separated or repeated).

Examples:
  legbled tx claim "Water boils at 100C at sea level" --domain physics --stake 0.1LGM --from agent
  legbled tx claim "1+1=2" --domain mathematics --category formal --stake 1LGM --refs MATH-001,MATH-002 --from agent`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			domain, _ := cmd.Flags().GetString("domain")
			category, _ := cmd.Flags().GetString("category")
			stakeRaw, _ := cmd.Flags().GetString("stake")
			logicZone, _ := cmd.Flags().GetString("logic-zone")
			refs, _ := cmd.Flags().GetStringSlice("refs")

			if domain == "" {
				return fmt.Errorf("--domain is required")
			}
			if stakeRaw == "" {
				return fmt.Errorf("--stake is required")
			}

			// Parse human-readable amount
			stakeUlgm, err := ParseHumanAmount(stakeRaw)
			if err != nil {
				return fmt.Errorf("invalid stake amount: %w", err)
			}

			msg := &knowledgetypes.MsgSubmitClaim{
				ClaimText:         args[0],
				Domain:            domain,
				EpistemicCategory: category,
				Submitter:         clientCtx.GetFromAddress().String(),
				StakeAmount:       stakeUlgm,
				LogicZone:         logicZone,
				References:        refs,
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Submitting claim:\n  Statement: %s\n  Domain:    %s\n  Stake:     %s\n\n",
				args[0], domain, FormatUlgmToLGM(stakeUlgm))

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("domain", "", "Knowledge domain (required)")
	cmd.Flags().String("category", "empirical", "Epistemic category (analytic, formal, empirical, etc.)")
	cmd.Flags().String("stake", "", "Stake amount (e.g., 0.1LGM, 100000ulgm) (required)")
	cmd.Flags().String("logic-zone", "empirical", "Logic zone (propositional, presburger, peano, empirical)")
	cmd.Flags().StringSlice("refs", nil, "Referenced fact IDs (comma-separated or repeated --refs)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
