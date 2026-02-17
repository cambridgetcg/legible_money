package types

const (
	// ModuleName defines the module name.
	ModuleName = "knowledge"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey defines the routing key.
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key.
	MemStoreKey = "mem_" + ModuleName

	// QuerierRoute defines the querier route.
	QuerierRoute = ModuleName

	// PortID is the IBC port ID for the knowledge oracle.
	PortID = "knowledge"

	// Version is the IBC module version for knowledge oracle.
	Version = "lgm-knowledge-1"
)

// Store key prefixes.
var (
	FactKeyPrefix              = []byte{0x01}
	ClaimKeyPrefix             = []byte{0x02}
	VerificationRoundKeyPrefix = []byte{0x03}
	FactReferenceKeyPrefix     = []byte{0x04}
	DomainFactIndexPrefix      = []byte{0x05}
	ParamsKey                  = []byte{0x06}
	// 0x07 used by domainKey in state.go
	IncomingRefIndexPrefix = []byte{0x08} // toFactId:fromFactId → reference lookup
	ContentHashIndexPrefix = []byte{0x09} // contentHash → claimId (duplicate detection)
	ClaimRoundIndexPrefix  = []byte{0x0a} // claimId → roundId (active round lookup)
	EquivocationKeyPrefix  = []byte{0x0b} // roundId:validator → equivocation evidence

	// Adversarial verification store keys
	ProvisionalChallengeKeyPrefix = []byte{0x0c} // challengeId → ProvisionalChallenge
	ChallengerCooldownKeyPrefix   = []byte{0x0d} // challenger:claimId → last challenge height
	PendingEvalClaimIndexPrefix   = []byte{0x0e} // height:claimId → index of claims awaiting eval
	SubmitterCalibrationPrefix    = []byte{0x0f} // submitterAddr → CalibrationRecord

	// Negative knowledge store keys
	CounterFactKeyPrefix             = []byte{0x10} // counterFactId → CounterFact
	CounterFactByFactIndexPrefix     = []byte{0x11} // targetFactId:counterFactId → lookup
	CounterFactByDomainIndexPrefix   = []byte{0x12} // domain:counterFactId → lookup
	CounterFactByHeightIndexPrefix   = []byte{0x13} // height:counterFactId → lookup (for expiry)
	FactNegationLinkPrefix           = []byte{0x14} // factId → max counter-fact confidence (cached)
	ContradictionCooldownPrefix      = []byte{0x15} // submitter:epoch → contradiction count
	FalsificationEpochPaidPrefix     = []byte{0x16} // epoch → total ulgm paid in falsification rewards
	FalsificationCarryForwardPrefix  = []byte{0x17} // epoch → ulgm carried forward from prior epoch cap overflow
	CounterFactChallengeKeyPrefix    = []byte{0x18} // counterFactChallengeId → CounterFactChallenge
	CounterFactChallengeWindowPrefix = []byte{0x19} // counterFactId → challenge window end height
	ExtendedParamsKey                = []byte{0x1a} // ExtendedParams (hand-written, JSON-encoded)
	PatronageRecordPrefix            = []byte{0x1b} // factId:patron → PatronageRecord
	PruningQueuePrefix               = []byte{0x1c} // graceEndBlock:factId → index for batch pruning
)
