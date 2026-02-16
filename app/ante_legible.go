package app

import (
	"context"
	"encoding/hex"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	legibleauthkeeper "github.com/legible-money/legible/x/auth/keeper"
	legibleauthtypes "github.com/legible-money/legible/x/auth/types"
	emergencykeeper "github.com/legible-money/legible/x/emergency/keeper"
	emergencytypes "github.com/legible-money/legible/x/emergency/types"
)

// msgWithSigners is implemented by hand-written Legible message types.
type msgWithSigners interface {
	GetSigners() []sdk.AccAddress
}

// ---------- BootstrapGasFreeDecorator ----------

// BootstrapGasFreeDecorator waives gas and fees for essential PoT transactions
// during the bootstrap period (first 480,000 blocks ≈ 14 days).
// This allows the network to begin verifying truth before any LGM is minted.
type BootstrapGasFreeDecorator struct{}

// NewBootstrapGasFreeDecorator creates a new BootstrapGasFreeDecorator.
func NewBootstrapGasFreeDecorator() BootstrapGasFreeDecorator {
	return BootstrapGasFreeDecorator{}
}

func (bgd BootstrapGasFreeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if ctx.BlockHeight() > BootstrapEndBlock {
		return next(ctx, tx, simulate)
	}

	// Check if ALL messages in the tx are bootstrap-eligible
	msgs := tx.GetMsgs()
	allEligible := len(msgs) > 0
	for _, msg := range msgs {
		if !BootstrapGasFreeTypes[sdk.MsgTypeURL(msg)] {
			allEligible = false
			break
		}
	}

	if !allEligible {
		return next(ctx, tx, simulate)
	}

	// Set infinite gas meter — bootstrap tx pays no gas
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	return next(ctx, tx, simulate)
}

// ---------- EmergencyHaltDecorator ----------

// EmergencyHaltDecorator blocks all non-emergency transactions when the chain
// is in a halted state. Emergency module messages (propose/vote halt/revert/resume)
// are always allowed so guardians can coordinate recovery.
//
// Placed early in the ante chain (after SetUpContext, before gas/fee processing)
// so halted transactions don't consume gas or fees.
type EmergencyHaltDecorator struct {
	ek emergencykeeper.Keeper
}

// NewEmergencyHaltDecorator creates a new EmergencyHaltDecorator.
func NewEmergencyHaltDecorator(ek emergencykeeper.Keeper) EmergencyHaltDecorator {
	return EmergencyHaltDecorator{ek: ek}
}

func (ehd EmergencyHaltDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	if !ehd.ek.IsHalted(ctx) {
		return next(ctx, tx, simulate)
	}

	// Chain is halted — only allow emergency module messages
	for _, msg := range tx.GetMsgs() {
		if !emergencytypes.IsEmergencyMsg(msg) {
			return ctx, emergencytypes.ErrChainHalted
		}
	}

	return next(ctx, tx, simulate)
}

// ---------- LegibleAccountDecorator ----------

// LegibleAccountDecorator enforces Legible-specific account constraints:
// 1. Frozen accounts cannot send transactions
// 2. Updates LastActiveBlock for registered Legible accounts
//
// Runs AFTER signature verification (signer is already authenticated).
type LegibleAccountDecorator struct {
	lak legibleauthkeeper.Keeper
}

// NewLegibleAccountDecorator creates a new LegibleAccountDecorator.
func NewLegibleAccountDecorator(lak legibleauthkeeper.Keeper) LegibleAccountDecorator {
	return LegibleAccountDecorator{lak: lak}
}

func (lad LegibleAccountDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	msgs := tx.GetMsgs()
	currentHeight := uint64(ctx.BlockHeight())

	for _, msg := range msgs {
		ms, ok := msg.(msgWithSigners)
		if !ok {
			continue
		}
		signers := ms.GetSigners()
		for _, signer := range signers {
			address := signer.String()

			account, found := lad.lak.GetAccount(ctx, address)
			if !found {
				// Not a registered Legible account — standard Cosmos account, skip
				continue
			}

			// Check frozen status
			if account.Flags.Frozen {
				return ctx, legibleauthtypes.ErrAccountFrozen
			}

			// Update last active block.
			// NOTE: This runs in the ante handler, so it persists even if the
			// message handler fails (DeliverTx mode). This is intentional:
			// submitting a signed transaction demonstrates account activity
			// regardless of execution outcome. The account holder proved key
			// possession and paid gas fees.
			if account.LastActiveBlock < currentHeight {
				account.LastActiveBlock = currentHeight
				lad.lak.SetAccount(ctx, account)
			}
		}
	}

	return next(ctx, tx, simulate)
}

// ---------- LegibleCapabilityDecorator ----------

// LegibleCapabilityDecorator enforces session key capabilities.
// If a tx is signed with a pubkey matching a registered session key,
// the decorator checks that the session key has the required capabilities
// for the message types in the tx.
//
// Runs AFTER signature verification and LegibleAccountDecorator.
type LegibleCapabilityDecorator struct {
	lak legibleauthkeeper.Keeper
	ak  AccountKeeperForLegible
}

// AccountKeeperForLegible is the Cosmos AccountKeeper interface needed by capability decorator.
type AccountKeeperForLegible interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

// NewLegibleCapabilityDecorator creates a new LegibleCapabilityDecorator.
func NewLegibleCapabilityDecorator(lak legibleauthkeeper.Keeper, ak AccountKeeperForLegible) LegibleCapabilityDecorator {
	return LegibleCapabilityDecorator{lak: lak, ak: ak}
}

func (lcd LegibleCapabilityDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	msgs := tx.GetMsgs()
	currentHeight := uint64(ctx.BlockHeight())

	for _, msg := range msgs {
		ms, ok := msg.(msgWithSigners)
		if !ok {
			continue
		}
		signers := ms.GetSigners()
		for _, signer := range signers {
			address := signer.String()

			// Get the Cosmos account to read the signing pubkey
			cosmosAcc := lcd.ak.GetAccount(ctx, signer)
			if cosmosAcc == nil {
				continue
			}
			pubKey := cosmosAcc.GetPubKey()
			if pubKey == nil {
				continue
			}

			// Check if the signing pubkey matches a registered session key
			signingKeyHex := hex.EncodeToString(pubKey.Bytes())
			session := lcd.findSessionByPubKey(ctx, address, signingKeyHex, currentHeight)
			if session == nil {
				// Not signing with a session key — operational/identity key, no restrictions
				continue
			}

			// Enforce session capabilities against message type
			if err := lcd.checkCapability(session, msg); err != nil {
				return ctx, err
			}
		}
	}

	return next(ctx, tx, simulate)
}

// findSessionByPubKey checks if the given pubkey matches any active session key for the owner.
func (lcd LegibleCapabilityDecorator) findSessionByPubKey(ctx sdk.Context, owner, pubKeyHex string, height uint64) *legibleauthtypes.SessionKey {
	sessions := lcd.lak.GetSessionKeysForOwner(ctx, owner)
	for _, session := range sessions {
		if session.ExpiresAtBlock <= height {
			continue // expired
		}
		if session.PublicKey == pubKeyHex {
			return &session
		}
	}
	return nil
}

// checkCapability verifies the session key has capability for the given message type.
// Default-deny: unrecognized message types are rejected for session keys.
func (lcd LegibleCapabilityDecorator) checkCapability(session *legibleauthtypes.SessionKey, msg sdk.Msg) error {
	msgType := sdk.MsgTypeURL(msg)

	// Map message types to required capabilities.
	// Any message type not covered here is denied by default.
	switch {
	case isTransferMsg(msgType):
		if !session.Capabilities.CanTransfer {
			return legibleauthtypes.ErrSessionCapabilityDenied
		}
		return nil
	case isStakeMsg(msgType):
		if !session.Capabilities.CanStake {
			return legibleauthtypes.ErrSessionCapabilityDenied
		}
		return nil
	case isClaimMsg(msgType):
		if !session.Capabilities.CanSubmitClaims {
			return legibleauthtypes.ErrSessionCapabilityDenied
		}
		return nil
	case isVoteMsg(msgType):
		if !session.Capabilities.CanVote {
			return legibleauthtypes.ErrSessionCapabilityDenied
		}
		return nil
	case isICAMsg(msgType):
		if !session.Capabilities.CanIca {
			return legibleauthtypes.ErrSessionCapabilityDenied
		}
		// ICA registration requires operational key — session keys can only
		// send transactions to already-registered ICAs.
		if msgType == "/legible.icaauth.v1.Msg/RegisterInterchainAccount" {
			return legibleauthtypes.ErrSessionCapabilityDenied
		}
		return nil
	default:
		// Default deny: session keys cannot perform unrecognized message types
		return legibleauthtypes.ErrSessionCapabilityDenied
	}
}

func isTransferMsg(msgType string) bool {
	return msgType == "/cosmos.bank.v1beta1.MsgSend" ||
		msgType == "/cosmos.bank.v1beta1.MsgMultiSend" ||
		msgType == "/ibc.applications.transfer.v1.MsgTransfer"
}

func isStakeMsg(msgType string) bool {
	return msgType == "/cosmos.staking.v1beta1.MsgDelegate" ||
		msgType == "/cosmos.staking.v1beta1.MsgUndelegate" ||
		msgType == "/cosmos.staking.v1beta1.MsgBeginRedelegate" ||
		msgType == "/legible.staking.v1.MsgRegisterValidator"
}

func isClaimMsg(msgType string) bool {
	return msgType == "/legible.knowledge.v1.MsgSubmitClaim" ||
		msgType == "/legible.knowledge.v1.MsgSubmitCommitment" ||
		msgType == "/legible.knowledge.v1.MsgSubmitReveal" ||
		msgType == "/legible.knowledge.v1.MsgChallengeFact"
}

func isICAMsg(msgType string) bool {
	return msgType == "/legible.icaauth.v1.Msg/RegisterInterchainAccount" ||
		msgType == "/legible.icaauth.v1.Msg/SendICATransaction"
}

func isVoteMsg(msgType string) bool {
	return msgType == "/cosmos.gov.v1.MsgVote" ||
		msgType == "/legible.gov.v1.MsgCastVote" ||
		msgType == "/legible.emergency.v1.MsgVoteHalt" ||
		msgType == "/legible.emergency.v1.MsgVoteRevert" ||
		msgType == "/legible.emergency.v1.MsgVoteResume"
}

// ---------- LegibleDIDDecorator ----------

// LegibleDIDDecorator validates DID references in transactions.
// If a tx memo contains "did:lgm:", it validates the DID resolves to the sender
// and emits indexing events.
//
// Runs AFTER signature verification (post-auth) to prevent unauthenticated
// state reads. DID validation is an additional constraint, not a prerequisite.
type LegibleDIDDecorator struct {
	lak legibleauthkeeper.Keeper
}

// NewLegibleDIDDecorator creates a new LegibleDIDDecorator.
func NewLegibleDIDDecorator(lak legibleauthkeeper.Keeper) LegibleDIDDecorator {
	return LegibleDIDDecorator{lak: lak}
}

func (ldd LegibleDIDDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	// Check if tx has a memo with DID reference
	memoTx, ok := tx.(sdk.TxWithMemo)
	if !ok {
		return next(ctx, tx, simulate)
	}

	memo := memoTx.GetMemo()
	if len(memo) < 8 || memo[:8] != "did:lgm:" {
		return next(ctx, tx, simulate)
	}

	// Extract DID from memo (format: "did:lgm:{64hex}")
	did := memo
	if len(did) > 72 { // "did:lgm:" + 64 hex = 72
		did = memo[:72]
	}

	if err := legibleauthtypes.ValidateDID(did); err != nil {
		return ctx, legibleauthtypes.ErrDIDResolutionFailed
	}

	// Verify DID resolves to a known address
	address, found := ldd.lak.GetAddressForDID(ctx, did)
	if !found {
		return ctx, legibleauthtypes.ErrDIDResolutionFailed
	}

	// Validate that the DID resolves to one of the message signers
	msgs := tx.GetMsgs()
	senderMatch := false
	for _, msg := range msgs {
		if ms, ok := msg.(msgWithSigners); ok {
			for _, signer := range ms.GetSigners() {
				if signer.String() == address {
					senderMatch = true
					break
				}
			}
		}
		if senderMatch {
			break
		}
	}

	if !senderMatch {
		return ctx, legibleauthtypes.ErrDIDResolutionFailed
	}

	// Emit DID event for indexing
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"did_reference",
			sdk.NewAttribute("did", did),
			sdk.NewAttribute("address", address),
		),
	)

	return next(ctx, tx, simulate)
}
