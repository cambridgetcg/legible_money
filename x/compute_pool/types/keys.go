package types

const (
	// ModuleName defines the module name.
	ModuleName = "compute_pool"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey defines the routing key.
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key.
	MemStoreKey = "mem_" + ModuleName

	// QuerierRoute defines the querier route.
	QuerierRoute = ModuleName
)

// Store key prefixes.
var (
	ParamsKey              = []byte{0x00} // -> Params
	ProviderKeyPrefix      = []byte{0x01} // {address} -> ComputeProvider
	CreditBalancePrefix    = []byte{0x02} // {validatorAddr} -> ComputeCredit
	RedemptionTicketPrefix = []byte{0x03} // {ticketId} -> RedemptionTicket
	UsageReportPrefix      = []byte{0x04} // {ticketId} -> UsageReport
	EpochStatsPrefix       = []byte{0x05} // {epoch BE64} -> EpochStats
	ProviderEpochPrefix    = []byte{0x06} // {epoch BE64}/{address} -> ProviderEpochFlow
	TicketSequenceKey      = []byte{0x07} // -> uint64 (next ticket ID)
	AdjustmentFactorKey    = []byte{0x08} // -> uint64 (current CU adjustment factor in bps)
	TaskKeyPrefix          = []byte{0x09} // {taskId BE64} -> ComputeTask
	TaskSequenceKey        = []byte{0x0a} // -> uint64 (next task ID)
	AgentUsagePrefix       = []byte{0x0b} // {agentDID} -> AgentUsageRecord
	TasksByRequesterPrefix = []byte{0x0c} // {requesterDID}/{taskId BE64} -> []byte{1} (index)
	TasksByProviderPrefix  = []byte{0x0d} // {providerAddr}/{taskId BE64} -> []byte{1} (index)
)

// Governance-tunable defaults.
const (
	// DefaultComputePoolShareBps is the share of the verification pool
	// allocated to compute. 300000 = 30% of the verification pool.
	DefaultComputePoolShareBps = 300000

	// DefaultBaseCUPerVerification is the base compute units earned per valid verification.
	DefaultBaseCUPerVerification uint64 = 100

	// DefaultComplexityBonusBps is the additional CU per evidence item (bps of base).
	// 25000 = 0.25 × base per evidence item.
	DefaultComplexityBonusBps uint64 = 25000

	// DefaultCrossDomainBonusBps is the additional CU per cross-domain reference (bps of base).
	// 50000 = 0.5 × base per cross-domain ref.
	DefaultCrossDomainBonusBps uint64 = 50000

	// DefaultMaxCreditBalanceMultiplier limits how many epochs-worth of credits
	// a validator can accumulate.
	DefaultMaxCreditBalanceMultiplier uint64 = 10

	// DefaultMinProviderStake is the minimum stake (ulgm) to register as a compute provider.
	DefaultMinProviderStake = "100000000" // 100 LGM

	// DefaultMinUptimeBps is the minimum rolling uptime required to remain active.
	DefaultMinUptimeBps uint64 = 950000 // 95%

	// DefaultHeartbeatIntervalBlocks is the max blocks between heartbeats before suspension.
	DefaultHeartbeatIntervalBlocks uint64 = 1000

	// DefaultMaxPricePerCU is the ceiling price per compute unit (ulgm).
	DefaultMaxPricePerCU = "10000" // 0.01 LGM

	// DefaultProviderUnbondingBlocks is the unbonding period after deregistration.
	DefaultProviderUnbondingBlocks uint64 = 34272 // 1 epoch

	// DefaultPriceChangeDelayBlocks is the delay before a price change takes effect.
	DefaultPriceChangeDelayBlocks uint64 = 1000

	// DefaultCreditExpiryBlocks is how long unused credits last before expiring.
	DefaultCreditExpiryBlocks uint64 = 34272 // 1 epoch

	// DefaultMaxProviderShareBps limits any single provider to 30% of pool per epoch.
	DefaultMaxProviderShareBps uint64 = 300000

	// DefaultRedemptionExpiryBlocks is the timeout for unconfirmed redemption tickets.
	DefaultRedemptionExpiryBlocks uint64 = 1000

	// DefaultDisputeWindowBlocks is the window during which a redemption can be disputed.
	DefaultDisputeWindowBlocks uint64 = 500

	// DefaultMinTasksForFullShare is the minimum lifetime tasks before a provider
	// can receive >10% of pool per epoch.
	DefaultMinTasksForFullShare uint64 = 100

	// DefaultMaxLatencyMs is the maximum acceptable response latency.
	DefaultMaxLatencyMs uint64 = 30000

	// DefaultSLAWindowBlocks is the rolling window for SLA metric calculation.
	DefaultSLAWindowBlocks uint64 = 1000

	// BlocksPerEpoch is used for epoch boundary calculations.
	BlocksPerEpoch uint64 = 34272

	// ---------- Utilization-Based Auto-Tuning Defaults ----------

	// DefaultTargetUtilizationLowBps is the lower bound of the target utilization range.
	// Below this, the module increases credit issuance to encourage spending.
	// 200000 = 20%.
	DefaultTargetUtilizationLowBps uint64 = 200000

	// DefaultTargetUtilizationHighBps is the upper bound of the target utilization range.
	// Above this, the module decreases credit issuance to prevent pool drain.
	// 800000 = 80%.
	DefaultTargetUtilizationHighBps uint64 = 800000

	// DefaultMaxAdjustmentPerEpochBps is the maximum change to the adjustment factor
	// in a single epoch. Prevents oscillation. 100000 = 10%.
	DefaultMaxAdjustmentPerEpochBps uint64 = 100000

	// DefaultMinAdjustmentFactorBps is the floor for the adjustment factor.
	// Credits can never drop below 20% of their base value. 200000 = 0.2x.
	DefaultMinAdjustmentFactorBps uint64 = 200000

	// DefaultMaxAdjustmentFactorBps is the ceiling for the adjustment factor.
	// Credits can never exceed 500% of their base value. 5000000 = 5.0x.
	DefaultMaxAdjustmentFactorBps uint64 = 5000000

	// DefaultAdjustmentFactorBps is the initial (neutral) adjustment factor.
	// 1000000 = 1.0x (no adjustment).
	DefaultAdjustmentFactorBps uint64 = 1000000

	// ---------- Task Lifecycle Defaults ----------

	// DefaultTaskExpiryBlocks is the max blocks before an unassigned task expires.
	DefaultTaskExpiryBlocks uint64 = 500

	// DefaultTaskResultExpiryBlocks is the max blocks a provider has to submit a result.
	DefaultTaskResultExpiryBlocks uint64 = 200

	// DefaultSlashOnRejectBps is the provider stake slash on rejected result (500 = 0.5%).
	DefaultSlashOnRejectBps uint64 = 500
)
