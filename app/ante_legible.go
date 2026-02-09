package app

import (
	"context"
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"

	legibleauthkeeper "github.com/legible-money/legible/x/auth/keeper"
	legibleauthtypes "github.com/legible-money/legible/x/auth/types"
)

// msgWithSigners is implemented by hand-written Legible message types.
type msgWithSigners interface {
	GetSigners() []sdk.AccAddress
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

			// Update last active block
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
func (lcd LegibleCapabilityDecorator) checkCapability(session *legibleauthtypes.SessionKey, msg sdk.Msg) error {
	msgType := sdk.MsgTypeURL(msg)

	// Map message types to required capabilities
	switch {
	case isTransferMsg(msgType):
		if !session.Capabilities.CanTransfer {
			return legibleauthtypes.ErrSessionCapabilityDenied
		}
	case isStakeMsg(msgType):
		if !session.Capabilities.CanStake {
			return legibleauthtypes.ErrSessionCapabilityDenied
		}
	case isClaimMsg(msgType):
		if !session.Capabilities.CanSubmitClaims {
			return legibleauthtypes.ErrSessionCapabilityDenied
		}
	case isVoteMsg(msgType):
		if !session.Capabilities.CanVote {
			return legibleauthtypes.ErrSessionCapabilityDenied
		}
	}

	return nil
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
// Runs early in the chain (before sig verification) for DID validation.
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
