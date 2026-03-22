package vm

import "math/big"

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
