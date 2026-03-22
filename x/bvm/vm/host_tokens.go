package vm

import (
	"context"
	"math/big"
)

// TokenHost is an optional interface for LGM-20 token opcodes (0xA5-0xAC).
// If a HostFunctions implementation also satisfies TokenHost, the interpreter
// dispatches token opcodes to these methods; otherwise safe defaults are used
// (balance/allowance return 0, mutations return false).
type TokenHost interface {
	TokenBalance(callerDID string, tokenID string, address []byte) *big.Int
	TokenTransfer(callerDID string, tokenID string, to []byte, amount *big.Int) bool
	TokenMint(callerDID string, tokenID string, to []byte, amount *big.Int) bool
	TokenBurn(callerDID string, tokenID string, amount *big.Int) bool
	TokenApprove(callerDID string, tokenID string, spender []byte, amount *big.Int) bool
	TokenAllowance(callerDID string, tokenID string, spender []byte) *big.Int
	TokenPause(callerDID string, tokenID string) bool
	TokenUnpause(callerDID string, tokenID string) bool
}

// StubTokenHost is a no-op implementation of TokenHost that returns safe defaults.
// Balance and Allowance return 0; all mutations return false.
type StubTokenHost struct{}

func (s *StubTokenHost) TokenBalance(_ string, _ string, _ []byte) *big.Int         { return big.NewInt(0) }
func (s *StubTokenHost) TokenTransfer(_ string, _ string, _ []byte, _ *big.Int) bool { return false }
func (s *StubTokenHost) TokenMint(_ string, _ string, _ []byte, _ *big.Int) bool     { return false }
func (s *StubTokenHost) TokenBurn(_ string, _ string, _ *big.Int) bool               { return false }
func (s *StubTokenHost) TokenApprove(_ string, _ string, _ []byte, _ *big.Int) bool  { return false }
func (s *StubTokenHost) TokenAllowance(_ string, _ string, _ []byte) *big.Int        { return big.NewInt(0) }
func (s *StubTokenHost) TokenPause(_ string, _ string) bool                          { return false }
func (s *StubTokenHost) TokenUnpause(_ string, _ string) bool                        { return false }

// VMTokensKeeper mirrors x/bvm/types.TokensKeeper to avoid circular imports.
// The adapter in app/bvm_adapters.go wraps the real x/tokens keeper.
type VMTokensKeeper interface {
	TokenBalance(ctx context.Context, tokenId string, holderAddr []byte) *big.Int
	TokenTransfer(ctx context.Context, callerDID, tokenId string, toAddr []byte, amount *big.Int) bool
	TokenMint(ctx context.Context, callerDID, tokenId string, toAddr []byte, amount *big.Int) bool
	TokenBurn(ctx context.Context, callerDID, tokenId string, amount *big.Int) bool
	TokenApprove(ctx context.Context, callerDID, tokenId string, spenderAddr []byte, amount *big.Int) bool
	TokenAllowance(ctx context.Context, tokenId string, ownerAddr, spenderAddr []byte) *big.Int
	TokenPause(ctx context.Context, callerDID, tokenId string) bool
	TokenUnpause(ctx context.Context, callerDID, tokenId string) bool
}

// KeeperTokenHost implements TokenHost by delegating to a VMTokensKeeper.
// Created per-call with the current context so state access is deterministic.
type KeeperTokenHost struct {
	keeper VMTokensKeeper
	ctx    context.Context
}

// NewKeeperTokenHost creates a KeeperTokenHost wired to the given keeper and context.
func NewKeeperTokenHost(keeper VMTokensKeeper, ctx context.Context) *KeeperTokenHost {
	return &KeeperTokenHost{keeper: keeper, ctx: ctx}
}

func (h *KeeperTokenHost) TokenBalance(callerDID string, tokenID string, address []byte) *big.Int {
	if h.keeper == nil {
		return big.NewInt(0)
	}
	bal := h.keeper.TokenBalance(h.ctx, tokenID, address)
	if bal == nil {
		return big.NewInt(0)
	}
	return bal
}

func (h *KeeperTokenHost) TokenTransfer(callerDID string, tokenID string, to []byte, amount *big.Int) bool {
	if h.keeper == nil {
		return false
	}
	return h.keeper.TokenTransfer(h.ctx, callerDID, tokenID, to, amount)
}

func (h *KeeperTokenHost) TokenMint(callerDID string, tokenID string, to []byte, amount *big.Int) bool {
	if h.keeper == nil {
		return false
	}
	return h.keeper.TokenMint(h.ctx, callerDID, tokenID, to, amount)
}

func (h *KeeperTokenHost) TokenBurn(callerDID string, tokenID string, amount *big.Int) bool {
	if h.keeper == nil {
		return false
	}
	return h.keeper.TokenBurn(h.ctx, callerDID, tokenID, amount)
}

func (h *KeeperTokenHost) TokenApprove(callerDID string, tokenID string, spender []byte, amount *big.Int) bool {
	if h.keeper == nil {
		return false
	}
	return h.keeper.TokenApprove(h.ctx, callerDID, tokenID, spender, amount)
}

func (h *KeeperTokenHost) TokenAllowance(callerDID string, tokenID string, spender []byte) *big.Int {
	if h.keeper == nil {
		return big.NewInt(0)
	}
	// TokenAllowance needs the owner address — derive from callerDID
	// The callerDID identifies the owner for allowance queries from the VM.
	// The adapter resolves DID → address internally.
	al := h.keeper.TokenAllowance(h.ctx, tokenID, []byte(callerDID), spender)
	if al == nil {
		return big.NewInt(0)
	}
	return al
}

func (h *KeeperTokenHost) TokenPause(callerDID string, tokenID string) bool {
	if h.keeper == nil {
		return false
	}
	return h.keeper.TokenPause(h.ctx, callerDID, tokenID)
}

func (h *KeeperTokenHost) TokenUnpause(callerDID string, tokenID string) bool {
	if h.keeper == nil {
		return false
	}
	return h.keeper.TokenUnpause(h.ctx, callerDID, tokenID)
}
