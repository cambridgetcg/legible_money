package vm

import "math/big"

// ExecutionMode determines call semantics.
type ExecutionMode int

const (
	ModeCall         ExecutionMode = iota
	ModeStaticCall                         // Read-only, inherited by sub-calls
	ModeDelegateCall                       // Target code, caller's storage
	ModeCreate                             // Contract creation
)

// ExecutionContext provides runtime parameters to the interpreter.
type ExecutionContext struct {
	Caller          []byte // 20-byte address
	CallerDID       string // did:lgm:... or empty
	Origin          []byte // Transaction originator
	OriginDID       string
	CurrentContract []byte // 20-byte address of executing contract
	BlockNumber     uint64
	Timestamp       uint64
	GasPrice        *big.Int
	GasLimit        uint64
	Value           *big.Int // Value transferred with call
	CallData        []byte
	Mode            ExecutionMode
	Bytecode        []byte   // Code being executed (for CODESIZE/CODECOPY)
	ChainID         *big.Int
	Coinbase        []byte // Block proposer
	PrevRandao      *big.Int
	BaseFee         *big.Int
}

// ExecutionResult is returned after bytecode execution completes.
type ExecutionResult struct {
	Success      bool
	ReturnData   []byte
	GasUsed      uint64
	Logs         []Log
	StateChanges []StateChange
	Error        *ExecutionError
}

// Log represents an event emitted by LOG0-LOG4.
type Log struct {
	Address []byte
	Topics  [][]byte
	Data    []byte
}

// StateChange tracks a single state mutation for atomic apply/rollback.
type StateChange struct {
	Type     StateChangeType
	Address  []byte
	Key      string
	OldValue *big.Int
	NewValue *big.Int
}

type StateChangeType int

const (
	StateChangeStorage StateChangeType = iota
	StateChangeBalance
	StateChangeNonce
	StateChangeCode
)

type ExecutionError struct {
	Code    ErrorCode
	Message string
}

func (e *ExecutionError) Error() string {
	return e.Message
}

type ErrorCode int

const (
	ErrCodeOutOfGas ErrorCode = iota
	ErrCodeStackOverflow
	ErrCodeStackUnderflow
	ErrCodeInvalidOpcode
	ErrCodeInvalidJump
	ErrCodeRevert
	ErrCodeStaticStateChange
	ErrCodeInsufficientBalance
	ErrCodeReturnDataOutOfBounds
	ErrCodeCallDepthExceeded
	ErrCodeExecutionFailed
)

// StateDB is the interface for contract storage access.
// Implementations: MemoryStateDB (tests) and CosmosStateDB (production).
type StateDB interface {
	GetStorage(contract []byte, key []byte) []byte
	SetStorage(contract []byte, key []byte, value []byte)
	GetBalance(addr []byte) *big.Int
	GetCode(addr []byte) []byte
	GetCodeSize(addr []byte) int
	GetCodeHash(addr []byte) []byte
	Exists(addr []byte) bool
}

// HostFunctions provides hooks for agent-specific opcodes (0xB0-0xEF, 0xF6-0xF9).
// Nil means agent opcodes push 0/no-op (graceful degradation).
type HostFunctions interface {
	// Knowledge (0xB0-0xB5)
	QueryFact(factId []byte) (exists bool, confidence uint64, domain []byte)
	QueryDomain(domain []byte, limit uint64) (count uint64, factIds [][]byte)
	SubmitClaim(callerDID string, domain, dataHash, evidenceHash []byte, blockNumber uint64) []byte
	CommitVote(callerDID string, claimId, voteHash []byte, blockNumber uint64) bool
	RevealVote(callerDID string, claimId []byte, vote, salt *big.Int, blockNumber uint64) bool
	InitiateChallenge(callerDID string, factId, evidenceHash []byte, stake *big.Int, blockNumber uint64) []byte

	// Agent Identity (0xE0-0xE3)
	AgentDID(caller []byte) []byte
	AgentHome(callerDID string) []byte
	VerifyIdentity(targetDID []byte) bool
	AttestIdentity(callerDID string, targetDID, attestationType, data []byte) []byte

	// Agent Belief (0xE4-0xE7)
	GetBelief(callerDID string, factId []byte) uint64
	SetBelief(callerDID string, factId []byte, confidence uint64, source []byte)
	CompareBelief(callerDID string, factId []byte) int64
	BeliefCount(callerDID string, domain []byte) uint64

	// Agent Relationship (0xE8-0xEB)
	GetTrust(callerDID string, targetDID []byte) uint64
	SetTrust(callerDID string, targetDID []byte, level uint64, reason []byte)
	IsTrusted(callerDID string, targetDID []byte, threshold uint64) bool
	Delegate(callerDID string, targetDID, capability, constraints []byte) []byte

	// Agent Autonomy (0xEC-0xEF)
	ScheduleSelf(contract []byte, callData []byte, executeBlock uint64, gasLimit uint64) []byte
	CancelScheduleHost(scheduleId []byte) bool
	ModifySchedule(scheduleId []byte, newBlock uint64) bool
	GetScheduleCount(contract []byte) uint64

	// Extended Relationships (0xC0-0xC3)
	ListTrusted(callerDID string, offset, limit uint64) (count uint64, dids [][]byte)
	BlockAgent(callerDID string, targetDID, reason []byte) bool
	UnblockAgent(callerDID string, targetDID []byte) bool
	RevokeDelegation(callerDID string, delegationId []byte) bool

	// Treasury (0xD0-0xD7)
	ClaimVesting(callerDID string, vestingId []byte) *big.Int
	QueryVesting(callerDID string, offset, limit uint64) (count uint64, positions [][]byte)
	StakeTokens(callerDID string, amount *big.Int, validatorDID []byte) []byte
	UnstakeTokens(callerDID string, stakeId []byte) *big.Int
	QueryExpiring(callerDID string, periodId []byte) (amount *big.Int, nextExpiration uint64)
	ExtendExpiring(callerDID string, periodId []byte, newExpiry uint64) bool
	SetSpendLimit(callerDID string, limitType, amount []byte, period uint64) bool
	QueryTreasury(callerDID string) (available, reserved, staked, delegated *big.Int)

	// Guardian (0xD8-0xDB)
	QueryAlerts(callerDID string, offset, limit uint64) (count uint64, alerts [][]byte)
	DismissAlert(callerDID string, alertId []byte) bool
	SetDefense(callerDID string, defenseType, config, threshold []byte) bool
	QueryDefenses(callerDID string) (count uint64, defenses [][]byte)

	// Access Control (0xDC-0xDF)
	RegisterKey(callerDID string, keyType, publicKey, permissions, expiry []byte) bool
	RevokeKey(callerDID string, keyId []byte) bool
	QueryPermissions(callerDID string, role []byte) *big.Int
	QuerySession(callerDID string) (sessionId []byte, role []byte, expiresAt uint64, actionsRemaining uint64)

	// Introspection (0xF6-0xF9)
	SelfBalance(callerDID string) *big.Int
	SelfStake(callerDID string) *big.Int
	SelfReputation(callerDID string) uint64
	SelfAccuracy(callerDID string) uint64
}

const MaxCallDepth = 1024
