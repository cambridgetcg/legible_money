package app

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	legibleauthkeeper "github.com/legible-money/legible/x/auth/keeper"
	authtypes "github.com/legible-money/legible/x/auth/types"
	bvmtypes "github.com/legible-money/legible/x/bvm/types"
	knowledgekeeper "github.com/legible-money/legible/x/knowledge/keeper"
	legiblestakingkeeper "github.com/legible-money/legible/x/staking/keeper"
	tokenskeeper "github.com/legible-money/legible/x/tokens/keeper"
)

// --- BVM Knowledge Adapter ---

// BVMKnowledgeAdapter wraps x/knowledge keeper to satisfy bvm types.KnowledgeKeeper.
type BVMKnowledgeAdapter struct {
	keeper knowledgekeeper.Keeper
}

func NewBVMKnowledgeAdapter(keeper knowledgekeeper.Keeper) *BVMKnowledgeAdapter {
	return &BVMKnowledgeAdapter{keeper: keeper}
}

var _ bvmtypes.KnowledgeKeeper = (*BVMKnowledgeAdapter)(nil)

func (a *BVMKnowledgeAdapter) GetFact(ctx context.Context, factId string) (interface{}, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	fact, found := a.keeper.GetFact(sdkCtx, factId)
	if !found {
		return nil, false
	}
	return fact, true
}

func (a *BVMKnowledgeAdapter) GetFactsByDomain(ctx context.Context, domain string) []interface{} {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	facts := a.keeper.GetFactsByDomain(sdkCtx, domain)
	result := make([]interface{}, len(facts))
	for i, f := range facts {
		result[i] = f
	}
	return result
}

func (a *BVMKnowledgeAdapter) SubmitClaimFromBVM(ctx context.Context, submitterDID, domain, contentHash, evidenceHash, logicZone string, stakeAmount uint64) (string, error) {
	return a.keeper.SubmitClaimFromBVM(ctx, submitterDID, domain, contentHash, evidenceHash, logicZone, stakeAmount)
}

func (a *BVMKnowledgeAdapter) CommitVoteFromBVM(ctx context.Context, validatorDID, roundId, commitmentHash string) error {
	return a.keeper.CommitVoteFromBVM(ctx, validatorDID, roundId, commitmentHash)
}

func (a *BVMKnowledgeAdapter) RevealVoteFromBVM(ctx context.Context, validatorDID, roundId, verdict string, confidence uint64, salt string) error {
	return a.keeper.RevealVoteFromBVM(ctx, validatorDID, roundId, verdict, confidence, salt)
}

func (a *BVMKnowledgeAdapter) ChallengeFactFromBVM(ctx context.Context, challengerDID, factId, evidenceHash string, stakeAmount uint64) (string, error) {
	return a.keeper.ChallengeFactFromBVM(ctx, challengerDID, factId, evidenceHash, stakeAmount)
}

// --- BVM Staking Adapter ---

// BVMStakingAdapter wraps x/staking keeper to satisfy bvm types.StakingKeeper.
type BVMStakingAdapter struct {
	keeper legiblestakingkeeper.Keeper
}

func NewBVMStakingAdapter(keeper legiblestakingkeeper.Keeper) *BVMStakingAdapter {
	return &BVMStakingAdapter{keeper: keeper}
}

var _ bvmtypes.StakingKeeper = (*BVMStakingAdapter)(nil)

func (a *BVMStakingAdapter) GetValidatorStake(ctx context.Context, validatorAddr string) (uint64, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	validator, found := a.keeper.GetValidator(sdkCtx, validatorAddr)
	if !found {
		return 0, false
	}
	return bigIntStringToUint64(validator.TotalStake), true
}

// --- BVM Auth Adapter ---

// BVMAuthAdapter wraps x/auth keeper to satisfy bvm types.AuthKeeper.
type BVMAuthAdapter struct {
	keeper legibleauthkeeper.Keeper
}

func NewBVMAuthAdapter(keeper legibleauthkeeper.Keeper) *BVMAuthAdapter {
	return &BVMAuthAdapter{keeper: keeper}
}

var _ bvmtypes.AuthKeeper = (*BVMAuthAdapter)(nil)

func (a *BVMAuthAdapter) GetAccountByDID(ctx context.Context, did string) (interface{}, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	account, found := a.keeper.GetAccountByDID(sdkCtx, did)
	if !found {
		return nil, false
	}
	return account, true
}

// GetAccountDID resolves a bech32 address to its DID.
func (a *BVMAuthAdapter) GetAccountDID(ctx context.Context, address string) (string, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	account, found := a.keeper.GetAccount(sdkCtx, address)
	if !found || account.Did == "" {
		return "", false
	}
	return account.Did, true
}

// GetSessionCapabilities returns the active session capabilities for an owner at a given block height.
// Finds the first valid (non-expired) session key and returns its capabilities.
func (a *BVMAuthAdapter) GetSessionCapabilities(ctx context.Context, owner string, blockHeight uint64) (bvmtypes.SessionCapabilities, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sessions := a.keeper.GetSessionKeysForOwner(sdkCtx, owner)
	for _, session := range sessions {
		if session.ExpiresAtBlock > blockHeight {
			if session.Capabilities == nil {
				continue
			}
			return bvmtypes.SessionCapabilities{
				CanTransfer:     session.Capabilities.CanTransfer,
				CanStake:        session.Capabilities.CanStake,
				CanSubmitClaims: session.Capabilities.CanSubmitClaims,
				CanVote:         session.Capabilities.CanVote,
			}, true
		}
	}
	return bvmtypes.SessionCapabilities{}, false
}

// RegisterSessionKey creates a session key in the canonical x/auth store.
func (a *BVMAuthAdapter) RegisterSessionKey(ctx context.Context, ownerAddress, keyHash, pubKey string, caps bvmtypes.SessionCapabilities, expiresAtBlock uint64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Generate key hash if not provided
	if keyHash == "" {
		h := sha256.Sum256([]byte(pubKey))
		keyHash = fmt.Sprintf("%x", h)
	}

	session := authtypes.SessionKey{
		KeyHash:        keyHash,
		PublicKey:      pubKey,
		Owner:          ownerAddress,
		ExpiresAtBlock: expiresAtBlock,
		CreatedAtBlock: uint64(sdkCtx.BlockHeight()),
		Capabilities: &authtypes.SessionCapabilities{
			CanTransfer:     caps.CanTransfer,
			CanStake:        caps.CanStake,
			CanSubmitClaims: caps.CanSubmitClaims,
			CanVote:         caps.CanVote,
		},
	}
	a.keeper.SetSessionKey(sdkCtx, &session)

	// Increment session key count on the account
	if account, found := a.keeper.GetAccount(sdkCtx, ownerAddress); found {
		account.SessionKeyCount++
		a.keeper.SetAccount(sdkCtx, account)
	}

	return nil
}

// RevokeSessionKey removes a session key from the canonical x/auth store.
func (a *BVMAuthAdapter) RevokeSessionKey(ctx context.Context, ownerAddress, keyHash string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	a.keeper.DeleteSessionKey(sdkCtx, ownerAddress, keyHash)

	// Decrement session key count on the account
	if account, found := a.keeper.GetAccount(sdkCtx, ownerAddress); found {
		if account.SessionKeyCount > 0 {
			account.SessionKeyCount--
			a.keeper.SetAccount(sdkCtx, account)
		}
	}

	return nil
}

// --- BVM Tokens Adapter ---

// BVMTokensAdapter wraps x/tokens keeper to satisfy bvm types.TokensKeeper.
// Converts raw byte addresses to bech32 strings for the underlying keeper calls.
type BVMTokensAdapter struct {
	keeper tokenskeeper.Keeper
}

func NewBVMTokensAdapter(keeper tokenskeeper.Keeper) *BVMTokensAdapter {
	return &BVMTokensAdapter{keeper: keeper}
}

var _ bvmtypes.TokensKeeper = (*BVMTokensAdapter)(nil)

func (a *BVMTokensAdapter) TokenBalance(ctx context.Context, tokenId string, holderAddr []byte) *big.Int {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	addr := sdk.AccAddress(holderAddr).String()
	return a.keeper.GetBalance(sdkCtx, tokenId, addr)
}

func (a *BVMTokensAdapter) TokenTransfer(ctx context.Context, callerDID, tokenId string, toAddr []byte, amount *big.Int) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if amount == nil || amount.Sign() <= 0 {
		return false
	}

	token := a.keeper.GetToken(sdkCtx, tokenId)
	if token == nil || token.Paused {
		return false
	}

	// callerDID is the sender; resolve to bech32 via DID format "did:lgm:{hex}"
	// For BVM, callerDID is passed as the sender address identifier.
	// The underlying keeper uses bech32 strings, so we use callerDID directly
	// (the BVM adapter in the keeper resolves identity before calling us).
	senderAddr := callerDID
	toAddrStr := sdk.AccAddress(toAddr).String()

	if senderAddr == toAddrStr {
		return false
	}

	// Check sender's undelegated balance
	undelegated := a.keeper.GetUndelegatedBalance(sdkCtx, tokenId, senderAddr)
	if undelegated.Cmp(amount) < 0 {
		return false
	}

	// Debit sender, credit recipient
	senderBal := a.keeper.GetBalance(sdkCtx, tokenId, senderAddr)
	senderBal = new(big.Int).Sub(senderBal, amount)
	a.keeper.SetBalance(sdkCtx, tokenId, senderAddr, senderBal)

	recipientBal := a.keeper.GetBalance(sdkCtx, tokenId, toAddrStr)
	recipientBal = new(big.Int).Add(recipientBal, amount)
	a.keeper.SetBalance(sdkCtx, tokenId, toAddrStr, recipientBal)

	return true
}

func (a *BVMTokensAdapter) TokenMint(ctx context.Context, callerDID, tokenId string, toAddr []byte, amount *big.Int) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if amount == nil || amount.Sign() <= 0 {
		return false
	}

	token := a.keeper.GetToken(sdkCtx, tokenId)
	if token == nil || token.Paused {
		return false
	}
	if token.Features == nil || !token.Features.Mintable {
		return false
	}

	// Auth: only token creator or governance authority can mint
	if callerDID != token.Creator && callerDID != a.keeper.GetAuthority() {
		return false
	}

	// Check supply cap
	totalSupply := new(big.Int)
	totalSupply.SetString(token.TotalSupply, 10)
	newSupply := new(big.Int).Add(totalSupply, amount)

	maxSupply := new(big.Int)
	maxSupply.SetString(token.MaxSupply, 10)
	if maxSupply.Sign() > 0 && newSupply.Cmp(maxSupply) > 0 {
		return false
	}

	// Update total supply
	token.TotalSupply = newSupply.String()
	a.keeper.SetToken(sdkCtx, token)

	// Credit recipient
	toAddrStr := sdk.AccAddress(toAddr).String()
	bal := a.keeper.GetBalance(sdkCtx, tokenId, toAddrStr)
	bal = new(big.Int).Add(bal, amount)
	a.keeper.SetBalance(sdkCtx, tokenId, toAddrStr, bal)

	return true
}

func (a *BVMTokensAdapter) TokenBurn(ctx context.Context, callerDID, tokenId string, amount *big.Int) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if amount == nil || amount.Sign() <= 0 {
		return false
	}

	token := a.keeper.GetToken(sdkCtx, tokenId)
	if token == nil || token.Paused {
		return false
	}
	if token.Features == nil || !token.Features.Burnable {
		return false
	}

	burnerAddr := callerDID

	// Check undelegated balance
	undelegated := a.keeper.GetUndelegatedBalance(sdkCtx, tokenId, burnerAddr)
	if undelegated.Cmp(amount) < 0 {
		return false
	}

	// Debit burner
	bal := a.keeper.GetBalance(sdkCtx, tokenId, burnerAddr)
	bal = new(big.Int).Sub(bal, amount)
	a.keeper.SetBalance(sdkCtx, tokenId, burnerAddr, bal)

	// Reduce total supply
	totalSupply := new(big.Int)
	totalSupply.SetString(token.TotalSupply, 10)
	totalSupply.Sub(totalSupply, amount)
	token.TotalSupply = totalSupply.String()
	a.keeper.SetToken(sdkCtx, token)

	return true
}

func (a *BVMTokensAdapter) TokenApprove(ctx context.Context, callerDID, tokenId string, spenderAddr []byte, amount *big.Int) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if amount == nil || amount.Sign() < 0 {
		return false
	}

	token := a.keeper.GetToken(sdkCtx, tokenId)
	if token == nil {
		return false
	}

	ownerAddr := callerDID
	spenderAddrStr := sdk.AccAddress(spenderAddr).String()

	if ownerAddr == spenderAddrStr {
		return false
	}

	a.keeper.SetAllowance(sdkCtx, tokenId, ownerAddr, spenderAddrStr, amount)
	return true
}

func (a *BVMTokensAdapter) TokenAllowance(ctx context.Context, tokenId string, ownerAddr, spenderAddr []byte) *big.Int {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// ownerAddr from the VM is the callerDID bytes (set by KeeperTokenHost.TokenAllowance)
	ownerAddrStr := string(ownerAddr)
	spenderAddrStr := sdk.AccAddress(spenderAddr).String()
	return a.keeper.GetAllowance(sdkCtx, tokenId, ownerAddrStr, spenderAddrStr)
}

func (a *BVMTokensAdapter) TokenPause(ctx context.Context, callerDID, tokenId string) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	token := a.keeper.GetToken(sdkCtx, tokenId)
	if token == nil {
		return false
	}
	if token.Features == nil || !token.Features.Pausable {
		return false
	}
	if callerDID != token.Creator && callerDID != a.keeper.GetAuthority() {
		return false
	}
	if token.Paused {
		return false
	}

	token.Paused = true
	a.keeper.SetToken(sdkCtx, token)
	return true
}

func (a *BVMTokensAdapter) TokenUnpause(ctx context.Context, callerDID, tokenId string) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	token := a.keeper.GetToken(sdkCtx, tokenId)
	if token == nil {
		return false
	}
	if token.Features == nil || !token.Features.Pausable {
		return false
	}
	if callerDID != token.Creator && callerDID != a.keeper.GetAuthority() {
		return false
	}
	if !token.Paused {
		return false
	}

	token.Paused = false
	a.keeper.SetToken(sdkCtx, token)
	return true
}
