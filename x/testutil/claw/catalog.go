package claw

// allModules lists every x/ module for coverage tracking.
var allModules = []string{
	"alignment",
	"auth",
	"autopoiesis",
	"billing",
	"bvm",
	"capture_challenge",
	"capture_defense",
	"channels",
	"claiming_pot",
	"compute_pool",
	"discovery",
	"disputes",
	"emergency",
	"evidence_mgmt",
	"gov",
	"home",
	"ibcratelimit",
	"icaauth",
	"knowledge",
	"ontology",
	"partnerships",
	"qualification",
	"research",
	"schedule",
	"staking",
	"tokens",
	"tree",
	"vesting_rewards",
}

// Catalog is the static registry of all adversarial probes.
// Hand-maintained because Go test packages are separate binaries;
// runtime registration won't work cross-package.
var Catalog = []Probe{
	// ====================================================================
	// knowledge module — OC-1 through OC-15
	// ====================================================================
	{
		ID: "OC-1", Name: "Whale Domination", Variant: OpenClaw, Severity: High,
		Status: Loophole, Module: "knowledge", File: "x/knowledge/keeper/openclaw_test.go",
		TestFunc: "TestOC1_WhaleDomination",
		Defense:  "Capture defense reputation weighting (not yet wired in test)",
	},
	{
		ID: "OC-2", Name: "Grief-by-Rejection", Variant: OpenClaw, Severity: High,
		Status: Partial, Module: "knowledge", File: "x/knowledge/keeper/openclaw_test.go",
		TestFunc: "TestOC2_GriefByRejection",
		Defense:  "VRF selection randomization, reputation decay, challenge mechanism",
	},
	{
		ID: "OC-3", Name: "Abstain-to-Avoid-Slash", Variant: OpenClaw, Severity: Medium,
		Status: Partial, Module: "knowledge", File: "x/knowledge/keeper/openclaw_test.go",
		TestFunc: "TestOC3_AbstainToAvoidSlash",
		Defense:  "No reward for abstaining; zero-cost participation risk remains",
	},
	{
		ID: "OC-4", Name: "Minimum Stake Spam", Variant: NanoClaw, Severity: High,
		Status: Loophole, Module: "knowledge", File: "x/knowledge/keeper/openclaw_test.go",
		TestFunc: "TestOC4_MinimumStakeSpam",
		Defense:  "Gas fees per tx provide primary defense; minimum claim stake floor recommended",
	},
	{
		ID: "OC-5", Name: "Tier 0 Reward Viability", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "knowledge", File: "x/knowledge/keeper/openclaw_test.go",
		TestFunc: "TestOC5_Tier0RewardViability",
		Defense:  "Tier 0 earns 0.1x multiplier (300k ulgm/round); viable but slow",
	},
	{
		ID: "OC-6", Name: "Counter-fact Profit Maximization", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "knowledge", File: "x/knowledge/keeper/openclaw_test.go",
		TestFunc: "TestOC6_CounterFactProfitMaximization",
		Defense:  "3x cap limits max profit; MinContradictionStakeRatio prevents dust attacks",
	},
	{
		ID: "OC-7", Name: "Split Verdict Stake Lock", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "knowledge", File: "x/knowledge/keeper/openclaw_test.go",
		TestFunc: "TestOC7_SplitVerdictStakeLock",
		Defense:  "Inconclusive verdict returns stake (C-1 fix)",
	},
	{
		ID: "OC-8", Name: "Difficulty Bonus Exploitation", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "knowledge", File: "x/knowledge/keeper/openclaw_test.go",
		TestFunc: "TestOC8_DifficultyBonusExploitation",
		Defense:  "Agreement ratio must exceed 77%; AH-4 method diversity check",
	},
	{
		ID: "OC-9", Name: "Research Tax Evasion", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "knowledge", File: "x/knowledge/keeper/openclaw_test.go",
		TestFunc: "TestOC9_ResearchTaxAnalysis",
		Defense:  "Tax applied uniformly at vesting layer; no evasion path",
	},
	{
		ID: "OC-10", Name: "Equivocation Risk/Reward", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "knowledge", File: "x/knowledge/keeper/openclaw_test.go",
		TestFunc: "TestOC10_EquivocationRiskReward",
		Defense:  "Duplicate commitments rejected; 100% slash makes equivocation irrational",
	},
	{
		ID: "OC-11", Name: "Quorum Manipulation", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "knowledge", File: "x/knowledge/keeper/openclaw_test.go",
		TestFunc: "TestOC11_QuorumManipulation",
		Defense:  "Missed reveal slash raised to 1% (C-2 fix)",
	},
	{
		ID: "OC-12", Name: "Sybil Attack Economics", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "knowledge", File: "x/knowledge/keeper/openclaw_test.go",
		TestFunc: "TestOC12_SybilAttackEconomics",
		Defense:  "VRF stake-weighting; Sybils need proportional stake; CaptureDefense",
	},
	{
		ID: "OC-13", Name: "Slash Rate Sensitivity", Variant: NanoClaw, Severity: Medium,
		Status: Defended, Module: "knowledge", File: "x/knowledge/keeper/openclaw_test.go",
		TestFunc: "TestOC13_SlashRateSensitivity",
		Defense:  "22% claim slash is moderate deterrent; missed-reveal raised to 1%",
	},
	{
		ID: "OC-14", Name: "Integer Truncation Attack", Variant: NanoClaw, Severity: Critical,
		Status: Partial, Module: "knowledge", File: "x/knowledge/keeper/openclaw_test.go",
		TestFunc: "TestOC14_IntegerTruncationAttack",
		Defense:  "Min validator stake above truncation threshold; claim floor needed",
	},
	{
		ID: "OC-15", Name: "Multi-round Accumulation", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "knowledge", File: "x/knowledge/keeper/openclaw_test.go",
		TestFunc: "TestOC15_MultiRoundAccumulation",
		Defense:  "Tier 0 negative EV at <85% accuracy is by design (reputation gate)",
	},

	// ====================================================================
	// auth module — OCP-1 through OCP-12
	// ====================================================================
	{
		ID: "OCP-1", Name: "DID Spoofing", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "auth", File: "x/auth/keeper/openclaw_privacy_test.go",
		TestFunc: "TestOCP1_DIDSpoofing",
		Defense:  "ValidateDIDDerivation enforces cryptographic binding",
	},
	{
		ID: "OCP-2", Name: "Session Capability Boundary", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "auth", File: "x/auth/keeper/openclaw_privacy_test.go",
		TestFunc: "TestOCP2_SessionCapabilityBoundary",
		Defense:  "Proto serialization preserves capability bits exactly",
	},
	{
		ID: "OCP-3", Name: "Session Expiry Enforcement", Variant: NanoClaw, Severity: Critical,
		Status: Defended, Module: "auth", File: "x/auth/keeper/openclaw_privacy_test.go",
		TestFunc: "TestOCP3_SessionExpiryEnforcement",
		Defense:  "Strict < comparison on ExpiresAtBlock",
	},
	{
		ID: "OCP-4", Name: "Frozen Account Isolation", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "auth", File: "x/auth/keeper/openclaw_privacy_test.go",
		TestFunc: "TestOCP4_FrozenAccountIsolation",
		Defense:  "Ante handler LegibleAccountDecorator rejects frozen account txs",
	},
	{
		ID: "OCP-5", Name: "DID Format Fuzzing", Variant: ZeroClaw, Severity: High,
		Status: Defended, Module: "auth", File: "x/auth/keeper/openclaw_privacy_test.go",
		TestFunc: "TestOCP5_DIDFormatFuzzing",
		Defense:  "Strict regex-like validation (prefix + length + hex charset)",
	},
	{
		ID: "OCP-6", Name: "DID Derivation Determinism", Variant: NanoClaw, Severity: Critical,
		Status: Defended, Module: "auth", File: "x/auth/keeper/openclaw_privacy_test.go",
		TestFunc: "TestOCP6_DIDDerivationDeterminism",
		Defense:  "DID = first 32 hex chars of pubkey (128-bit collision resistance)",
	},
	{
		ID: "OCP-7", Name: "Multi-Session Key Isolation", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "auth", File: "x/auth/keeper/openclaw_privacy_test.go",
		TestFunc: "TestOCP7_MultiSessionIsolation",
		Defense:  "SessionKeyKey(owner, hash) namespaces storage by owner address",
	},
	{
		ID: "OCP-8", Name: "Recovery Config Privacy", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "auth", File: "x/auth/keeper/openclaw_privacy_test.go",
		TestFunc: "TestOCP8_RecoveryConfigPrivacy",
		Defense:  "RecoveryConfigKey(address) namespaces by account",
	},
	{
		ID: "OCP-9", Name: "Account Type Immutability", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "auth", File: "x/auth/keeper/openclaw_privacy_test.go",
		TestFunc: "TestOCP9_AccountTypeImmutability",
		Defense:  "RegisterAccount rejects duplicate registration",
	},
	{
		ID: "OCP-10", Name: "Session Key Deletion", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "auth", File: "x/auth/keeper/openclaw_privacy_test.go",
		TestFunc: "TestOCP10_SessionDeletionEnforcement",
		Defense:  "DeleteSessionKey removes KV entry; no ghost references",
	},
	{
		ID: "OCP-11", Name: "DID Mapping Collision", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "auth", File: "x/auth/keeper/openclaw_privacy_test.go",
		TestFunc: "TestOCP11_DIDMappingCollision",
		Defense:  "ValidateDIDDerivation + duplicate check in RegisterAccount",
	},
	{
		ID: "OCP-12", Name: "Key Rotation Continuity", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "auth", File: "x/auth/keeper/openclaw_privacy_test.go",
		TestFunc: "TestOCP12_KeyRotationContinuity",
		Defense:  "4-layer architecture: identity anchor immutable, operational key rotates",
	},

	// ====================================================================
	// alignment module — OCA-T/U/B/J/C + MetaLoop
	// ====================================================================
	{
		ID: "OCA-T1", Name: "Colluding Validators TDI", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_T1_ColludingValidators",
		Defense:  "ComputeTDI detects unanimity + cross-validator correlation",
	},
	{
		ID: "OCA-T2", Name: "Zero Challenges", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_T2_ZeroChallenges",
		Defense:  "Counter-fact deficit produces non-zero TDI",
	},
	{
		ID: "OCA-T3", Name: "High Reversal Rate", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_T3_HighReversalRate",
		Defense:  "TDI rises proportionally with reversal rate",
	},
	{
		ID: "OCA-T4", Name: "TDI Recovery", Variant: OpenClaw, Severity: Low,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_T4_TDIRecovery",
		Defense:  "Healthy signals produce low TDI (negative feedback restores)",
	},
	{
		ID: "OCA-T5", Name: "Alignment Override Priority", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_T5_AlignmentOverridePriority",
		Defense:  "Corrections use priority 5 (higher than SSI=4)",
	},
	{
		ID: "OCA-U1", Name: "Isolated Facts DCI", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_U1_IsolatedFacts",
		Defense:  "100% orphan facts produce DCI=0",
	},
	{
		ID: "OCA-U2", Name: "Cross-Cited Facts DCI", Variant: OpenClaw, Severity: Low,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_U2_CrossCitedFacts",
		Defense:  "Heavy cross-citation produces high DCI",
	},
	{
		ID: "OCA-U3", Name: "Orphan Cap Correction", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_U3_OrphanCapCorrection",
		Defense:  "Low DCI triggers negative orphan cap delta",
	},
	{
		ID: "OCA-U4", Name: "Single-Domain Monoculture", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_U4_SingleDomainMonoculture",
		Defense:  "No cross-domain citations produce low DCI",
	},
	{
		ID: "OCA-B1", Name: "Multipliers at Bounds", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_B1_MultipliersAtBounds",
		Defense:  "Bound saturation produces low HI",
	},
	{
		ID: "OCA-B2", Name: "Oscillating Hormones", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_B2_OscillatingHormones",
		Defense:  "Oscillation detection reduces HI",
	},
	{
		ID: "OCA-B3", Name: "Beauty Freeze / Damping", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_B3_DampingCorrection",
		Defense:  "Low HI triggers damping correction",
	},
	{
		ID: "OCA-B4", Name: "HI Recovery", Variant: OpenClaw, Severity: Low,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_B4_HIRecovery",
		Defense:  "Harmonious signals produce high HI",
	},
	{
		ID: "OCA-J1", Name: "Whale Validators FI", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_J1_WhaleValidators",
		Defense:  "Extreme inequality produces low FI",
	},
	{
		ID: "OCA-J2", Name: "New Entrant Rejection", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_J2_NewEntrantRejection",
		Defense:  "Low new entrant success reduces FI",
	},
	{
		ID: "OCA-J3", Name: "Quadratic Boost Correction", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_J3_QuadraticBoostCorrection",
		Defense:  "Low FI triggers quadratic boost",
	},
	{
		ID: "OCA-J4", Name: "FI Recovery", Variant: OpenClaw, Severity: Low,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_J4_FIRecovery",
		Defense:  "Fair signals produce high FI",
	},
	{
		ID: "OCA-C1", Name: "Single-Domain Flooding", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_C1_SingleDomainFlooding",
		Defense:  "Zero diversity produces NI=0",
	},
	{
		ID: "OCA-C2", Name: "Cross-Domain Exploration", Variant: OpenClaw, Severity: Low,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_C2_CrossDomainExploration",
		Defense:  "Cross-domain signals produce high NI",
	},
	{
		ID: "OCA-C3", Name: "Stagnation Alarm", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_C3_StagnationAlarm",
		Defense:  "Low NI triggers cross-domain bonus",
	},
	{
		ID: "OCA-C4", Name: "NI Recovery", Variant: OpenClaw, Severity: Low,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestOCA_C4_NIRecovery",
		Defense:  "Creative signals produce high NI",
	},
	{
		ID: "OCA-ML", Name: "Meta-Loop Activation", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "alignment", File: "x/alignment/keeper/openclaw_test.go",
		TestFunc: "TestMetaLoop_Activation",
		Defense:  "Meta-loop runs at epoch 10 and 20 intervals",
	},

	// ====================================================================
	// autopoiesis module — OC-AP-1 through OC-AP-8
	// ====================================================================
	{
		ID: "OC-AP-1", Name: "SSI Metric Manipulation", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "autopoiesis", File: "x/autopoiesis/keeper/openclaw_autopoiesis_test.go",
		TestFunc: "TestOC_AP1_SSIMetricManipulation",
		Defense:  "Rate-limited to +/-5% per epoch (MaxChangePerEpochBps)",
	},
	{
		ID: "OC-AP-2", Name: "Oscillation Attack", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "autopoiesis", File: "x/autopoiesis/keeper/openclaw_autopoiesis_test.go",
		TestFunc: "TestOC_AP2_OscillationAttack",
		Defense:  "Quadratic damping limits swing between adjacent epochs",
	},
	{
		ID: "OC-AP-3", Name: "Bound Gaming", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "autopoiesis", File: "x/autopoiesis/keeper/openclaw_autopoiesis_test.go",
		TestFunc: "TestOC_AP3_BoundGaming",
		Defense:  "Consecutive bound-hit tracking triggers auto-proposal",
	},
	{
		ID: "OC-AP-4", Name: "Emergency Halt Bypass", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "autopoiesis", File: "x/autopoiesis/keeper/openclaw_autopoiesis_test.go",
		TestFunc: "TestOC_AP4_EmergencyHaltFreeze",
		Defense:  "Emergency halt freezes epoch advancement and meta-learning",
	},
	{
		ID: "OC-AP-5", Name: "Meta-Learning Rate Bounds", Variant: NanoClaw, Severity: Medium,
		Status: Defended, Module: "autopoiesis", File: "x/autopoiesis/keeper/openclaw_autopoiesis_test.go",
		TestFunc: "TestOC_AP5_MetaLearningRateBounds",
		Defense:  "adjustAdaptationRate enforces bounds; raw storage is unbounded by design",
	},
	{
		ID: "OC-AP-6", Name: "Proposal Spam Prevention", Variant: ZeroClaw, Severity: Medium,
		Status: Defended, Module: "autopoiesis", File: "x/autopoiesis/keeper/openclaw_autopoiesis_test.go",
		TestFunc: "TestOC_AP6_ProposalSpamPrevention",
		Defense:  "AutoProposalCooldownEpochs rate limiter (1 per 100 epochs)",
	},
	{
		ID: "OC-AP-7", Name: "Phantom Domain Stress", Variant: ZeroClaw, Severity: Low,
		Status: Defended, Module: "autopoiesis", File: "x/autopoiesis/keeper/openclaw_autopoiesis_test.go",
		TestFunc: "TestOC_AP7_PhantomDomainStress",
		Defense:  "Empty DVI list does not trigger proposals",
	},
	{
		ID: "OC-AP-8", Name: "Meta-Learning No Adaptations", Variant: ZeroClaw, Severity: Low,
		Status: Defended, Module: "autopoiesis", File: "x/autopoiesis/keeper/openclaw_autopoiesis_test.go",
		TestFunc: "TestOC_AP8_MetaLearningNoAdaptations",
		Defense:  "No-op with state update; no panic or spurious records",
	},

	// ====================================================================
	// bvm module — OC-BVM-1 through OC-BVM-18
	// ====================================================================
	{
		ID: "OC-BVM-1", Name: "Session Capability Blocked Stake", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM1_SessionCapability_StakeBlocked",
		Defense:  "checkCapability inspects execution context; nil = deny-all (C-1 fix)",
	},
	{
		ID: "OC-BVM-2", Name: "Anonymous Caller Denied", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM2_AnonymousCaller_AllAgentOpsDenied",
		Defense:  "No DID → caps nil → all agent opcodes denied",
	},
	{
		ID: "OC-BVM-3", Name: "Identity Key Full Capabilities", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM3_IdentityKeyAuth_FullCapabilities",
		Defense:  "DID + no session key → all-true capabilities (identity key owners)",
	},
	{
		ID: "OC-BVM-4", Name: "Scheduled Execution Full Caps", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM4_ScheduledExecution_FullCapabilities",
		Defense:  "ExecutePendingSchedules sets fullCaps (ZD-B01 fix)",
	},
	{
		ID: "OC-BVM-5", Name: "Cross-Contract State Isolation", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM5_CrossContractStateIsolation",
		Defense:  "State keys prefixed with contract address",
	},
	{
		ID: "OC-BVM-6", Name: "Value Refund on Revert", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM6_ValueRefundOnRevert",
		Defense:  "Refund from BVM module account on execution failure",
	},
	{
		ID: "OC-BVM-7", Name: "Value Transfer Success Retains", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM7_ValueTransfer_SuccessfulCallRetains",
		Defense:  "Value sent to module before execution; not refunded on success",
	},
	{
		ID: "OC-BVM-8", Name: "Schedule Cancel Authorization", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM8_ScheduleCancelAuth",
		Defense:  "CancelSchedule checks msg.Caller == schedule.Caller",
	},
	{
		ID: "OC-BVM-9", Name: "Governance State Update Auth", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM9_GovernanceStateUpdateAuth",
		Defense:  "UpdateContractState checks msg.Authority == k.GetAuthority()",
	},
	{
		ID: "OC-BVM-10", Name: "Code Dedup RefCount Lifecycle", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM10_CodeDedup_RefCountLifecycle",
		Defense:  "RefCount tracks references; code GC'd only at RefCount=0",
	},
	{
		ID: "OC-BVM-11", Name: "Schedule Limit Enforcement", Variant: ZeroClaw, Severity: Medium,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM11_ScheduleLimitEnforcement",
		Defense:  "MaxSchedulesPerContract enforced; cancel frees slots",
	},
	{
		ID: "OC-BVM-12", Name: "Bytecode Size Limit", Variant: ZeroClaw, Severity: Medium,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM12_BytecodeSizeLimit",
		Defense:  "MaxBytecodeSize enforced at DeployContract",
	},
	{
		ID: "OC-BVM-13", Name: "Schedule Creator-Only", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM13_ScheduleCreatorOnly",
		Defense:  "ScheduleContract checks msg.Caller == contract.Creator",
	},
	{
		ID: "OC-BVM-14", Name: "Gas Exhaustion Scheduled", Variant: ZeroClaw, Severity: High,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM14_GasExhaustion_ScheduledExecution",
		Defense:  "Gas metering; schedule marked executed regardless of success/failure",
	},
	{
		ID: "OC-BVM-15", Name: "Deterministic Contract Addresses", Variant: NanoClaw, Severity: High,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM15_DeterministicContractAddresses",
		Defense:  "Address = hash(deployer + blockHeight + nonce); nonce increments",
	},
	{
		ID: "OC-BVM-16", Name: "Overdue Schedule Execution", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM16_OverdueScheduleExecution",
		Defense:  "Range iterator catches all overdue schedules on chain resume",
	},
	{
		ID: "OC-BVM-17", Name: "Session Key Expiry in BVM", Variant: NanoClaw, Severity: High,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM17_SessionKeyExpiry",
		Defense:  "Expired session falls through to identity-key full access",
	},
	{
		ID: "OC-BVM-18", Name: "Insufficient Funds Payable Call", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "bvm", File: "x/bvm/keeper/openclaw_bvm_test.go",
		TestFunc: "TestOCBVM18_InsufficientFundsPayableCall",
		Defense:  "bankKeeper.SendCoins fails pre-execution; no partial execution",
	},

	// ====================================================================
	// gov module — OC-GOV-1 through OC-GOV-12
	// ====================================================================
	{
		ID: "OC-GOV-1", Name: "Quadratic Vote Sqrt Edge Cases", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "gov", File: "x/gov/keeper/openclaw_gov_test.go",
		TestFunc: "TestOC_GOV1_QuadraticVoteSqrtEdgeCases",
		Defense:  "BigInt integer sqrt verified at 0, 1, 4, 5, 9M, 10^18 boundaries",
	},
	{
		ID: "OC-GOV-2", Name: "Double Vote Prevention", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "gov", File: "x/gov/keeper/openclaw_gov_test.go",
		TestFunc: "TestOC_GOV2_DoubleVotePrevention",
		Defense:  "Duplicate vote (same or different option) rejected; different voter succeeds",
	},
	{
		ID: "OC-GOV-3", Name: "Stage Skip Attack", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "gov", File: "x/gov/keeper/openclaw_gov_test.go",
		TestFunc: "TestOC_GOV3_StageSkipAttack",
		Defense:  "AdvanceLIPStage enforces stake threshold before stage transition",
	},
	{
		ID: "OC-GOV-4", Name: "Proposal Spam by Min Stake", Variant: ZeroClaw, Severity: Medium,
		Status: Defended, Module: "gov", File: "x/gov/keeper/openclaw_gov_test.go",
		TestFunc: "TestOC_GOV4_ProposalSpamByMinStake",
		Defense:  "LIP counter increments monotonically; no ID collision after 20 submissions",
	},
	{
		ID: "OC-GOV-5", Name: "Vote After Period Ends", Variant: NanoClaw, Severity: High,
		Status: Defended, Module: "gov", File: "x/gov/keeper/openclaw_gov_test.go",
		TestFunc: "TestOC_GOV5_VoteAfterPeriodEnds",
		Defense:  "CastVote rejects votes after VotingEndsAt block",
	},
	{
		ID: "OC-GOV-6", Name: "Withdraw by Non-Proposer", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "gov", File: "x/gov/keeper/openclaw_gov_test.go",
		TestFunc: "TestOC_GOV6_WithdrawByNonProposer",
		Defense:  "WithdrawLIP checks msg.Proposer == lip.Proposer; non-proposer rejected",
	},
	{
		ID: "OC-GOV-7", Name: "Disbursement Timelock Race", Variant: NanoClaw, Severity: Critical,
		Status: Defended, Module: "gov", File: "x/gov/keeper/openclaw_gov_test.go",
		TestFunc: "TestOC_GOV7_DisbursementTimelockRace",
		Defense:  "ExecuteDisbursement checks blockHeight >= TimelockExpiry; off-by-one verified",
	},
	{
		ID: "OC-GOV-8", Name: "Category Stake Thresholds", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "gov", File: "x/gov/keeper/openclaw_gov_test.go",
		TestFunc: "TestOC_GOV8_CategoryStakeThresholds",
		Defense:  "Each LIP category has correct stake threshold: Consensus(1000), Economics(800), Interface(400), Process(200) LGM",
	},
	{
		ID: "OC-GOV-9", Name: "Terminal State Immutability", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "gov", File: "x/gov/keeper/openclaw_gov_test.go",
		TestFunc: "TestOC_GOV9_TerminalStateImmutability",
		Defense:  "Accepted LIP rejects advance, vote, and withdraw operations",
	},
	{
		ID: "OC-GOV-10", Name: "Upgrade Plan on Non-Consensus", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "gov", File: "x/gov/keeper/openclaw_gov_test.go",
		TestFunc: "TestOC_GOV10_UpgradePlanOnNonConsensus",
		Defense:  "AttachUpgradePlan rejects non-consensus category LIPs",
	},
	{
		ID: "OC-GOV-11", Name: "Vote on Non-Voting LIP", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "gov", File: "x/gov/keeper/openclaw_gov_test.go",
		TestFunc: "TestOC_GOV11_VoteOnNonVotingLIP",
		Defense:  "CastVote checks LIP stage == voting; rejects draft and review stage votes",
	},
	{
		ID: "OC-GOV-12", Name: "Disbursement Double Execution", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "gov", File: "x/gov/keeper/openclaw_gov_test.go",
		TestFunc: "TestOC_GOV12_DisbursementDoubleExecution",
		Defense:  "ExecuteDisbursement checks stage != executed; terminal state prevents replay",
	},

	// ====================================================================
	// staking module — OC-SK-1 through OC-SK-12
	// ====================================================================
	{
		ID: "OC-SK-1", Name: "Whale Stake Concentration", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "staking", File: "x/staking/keeper/openclaw_staking_test.go",
		TestFunc: "TestOC_SK1_WhaleStakeConcentration",
		Defense:  "Tier 0/1 use virtual stake for VRF selection; whale has no selection advantage until bonded",
	},
	{
		ID: "OC-SK-2", Name: "Slash Escalation Arithmetic", Variant: NanoClaw, Severity: High,
		Status: Defended, Module: "staking", File: "x/staking/keeper/openclaw_staking_test.go",
		TestFunc: "TestOC_SK2_SlashEscalationArithmetic",
		Defense:  "BigInt slash escalation: base * (1M + count*escalationBps) / 1M; no truncation across 5 slashes",
	},
	{
		ID: "OC-SK-3", Name: "Slash Per-Epoch Cap", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "staking", File: "x/staking/keeper/openclaw_staking_test.go",
		TestFunc: "TestOC_SK3_SlashPerEpochCap_R2",
		Defense:  "3rd slash silently skipped when MaxSlashesPerEpoch=2; resets at epoch boundary",
	},
	{
		ID: "OC-SK-4", Name: "Slash Decay Recovery", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "staking", File: "x/staking/keeper/openclaw_staking_test.go",
		TestFunc: "TestOC_SK4_SlashDecayRecovery_R1",
		Defense:  "SlashCount decays by 1 per clean epoch; 2 slashes recover to 0 after 2 clean epochs",
	},
	{
		ID: "OC-SK-5", Name: "Slash Floor Zero Amount", Variant: NanoClaw, Severity: Medium,
		Status: Defended, Module: "staking", File: "x/staking/keeper/openclaw_staking_test.go",
		TestFunc: "TestOC_SK5_SlashFloorZeroAmount_R4",
		Defense:  "Zero-amount slash and zero-stake validator slash don't increment slashCount",
	},
	{
		ID: "OC-SK-6", Name: "Sybil Registration Spam", Variant: ZeroClaw, Severity: High,
		Status: Defended, Module: "staking", File: "x/staking/keeper/openclaw_staking_test.go",
		TestFunc: "TestOC_SK6_SybilRegistrationSpam",
		Defense:  "100 min-stake sybils all tier 0 apprentice; 0 block producers; no VRF advantage",
	},
	{
		ID: "OC-SK-7", Name: "Redelegation Cooldown Bypass", Variant: NanoClaw, Severity: High,
		Status: Defended, Module: "staking", File: "x/staking/keeper/openclaw_staking_test.go",
		TestFunc: "TestOC_SK7_RedelegationCooldownBypass",
		Defense:  "Redelegate within cooldown (1111 blocks) rejected; off-by-one at exact boundary verified",
	},
	{
		ID: "OC-SK-8", Name: "Tier Demotion on Slash", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "staking", File: "x/staking/keeper/openclaw_staking_test.go",
		TestFunc: "TestOC_SK8_TierDemotionOnSlash",
		Defense:  "Guardian has MaxSlashCount=0; single slash disqualifies (slashCount=1 > 0)",
	},
	{
		ID: "OC-SK-9", Name: "Slash Own Before Delegated", Variant: NanoClaw, Severity: Critical,
		Status: Defended, Module: "staking", File: "x/staking/keeper/openclaw_staking_test.go",
		TestFunc: "TestOC_SK9_SlashOwnBeforeDelegated",
		Defense:  "Own stake consumed first; delegated absorbs overflow; boundary at exact own stake verified",
	},
	{
		ID: "OC-SK-10", Name: "Unbonding ID Collision", Variant: NanoClaw, Severity: High,
		Status: Defended, Module: "staking", File: "x/staking/keeper/openclaw_staking_test.go",
		TestFunc: "TestOC_SK10_UnbondingIdCollision",
		Defense:  "5 undelegations in same block all get unique IDs; all retrievable by ID",
	},
	{
		ID: "OC-SK-11", Name: "Block Producer Cap Enforcement", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "staking", File: "x/staking/keeper/openclaw_staking_test.go",
		TestFunc: "TestOC_SK11_BlockProducerCapEnforcement",
		Defense:  "MaxValidators limits block producers (tier 2+); verifiers (tier 0/1) unlimited",
	},
	{
		ID: "OC-SK-12", Name: "Accuracy Rate Precision", Variant: NanoClaw, Severity: High,
		Status: Defended, Module: "staking", File: "x/staking/keeper/openclaw_staking_test.go",
		TestFunc: "TestOC_SK12_AccuracyRatePrecision",
		Defense:  "GetAccuracyRate returns BPS; boundary at guardian 77% (770000 BPS) verified with off-by-one",
	},

	// ====================================================================
	// emergency module — OC-EM-1 through OC-EM-10
	// ====================================================================
	{
		ID: "OC-EM-1", Name: "Non-Guardian Halt Attempt", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "emergency", File: "x/emergency/keeper/openclaw_emergency_test.go",
		TestFunc: "TestOC_EM1_NonGuardianHaltAttempt",
		Defense:  "ProposeHalt rejects non-guardian and unknown addresses",
	},
	{
		ID: "OC-EM-2", Name: "Cooldown Bypass", Variant: NanoClaw, Severity: High,
		Status: Defended, Module: "emergency", File: "x/emergency/keeper/openclaw_emergency_test.go",
		TestFunc: "TestOC_EM2_CooldownBypass",
		Defense:  "CooldownBlocks=111 enforced; proposal at boundary-1 rejected; succeeds after elapsed",
	},
	{
		ID: "OC-EM-3", Name: "Proposal Spam Per Epoch", Variant: ZeroClaw, Severity: Medium,
		Status: Defended, Module: "emergency", File: "x/emergency/keeper/openclaw_emergency_test.go",
		TestFunc: "TestOC_EM3_ProposalSpamPerEpoch",
		Defense:  "MaxProposalsPerGuardianPerEpoch=1; same guardian blocked; different guardian succeeds",
	},
	{
		ID: "OC-EM-4", Name: "Precommit Without Prevote", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "emergency", File: "x/emergency/keeper/openclaw_emergency_test.go",
		TestFunc: "TestOC_EM4_PrecommitWithoutPrevote",
		Defense:  "Precommit requires prior prevote YES; guardian5 (no prevote) rejected",
	},
	{
		ID: "OC-EM-5", Name: "Duplicate Vote in Phase", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "emergency", File: "x/emergency/keeper/openclaw_emergency_test.go",
		TestFunc: "TestOC_EM5_DuplicateVoteInPhase",
		Defense:  "Duplicate prevote and vote-flip (YES->NO) both rejected",
	},
	{
		ID: "OC-EM-6", Name: "Halt from Wrong Status", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "emergency", File: "x/emergency/keeper/openclaw_emergency_test.go",
		TestFunc: "TestOC_EM6_HaltFromWrongStatus",
		Defense:  "ProposeHalt rejected when status is halted, resume_voting, or revert_voting",
	},
	{
		ID: "OC-EM-7", Name: "Revert Depth Exceeded", Variant: NanoClaw, Severity: Critical,
		Status: Defended, Module: "emergency", File: "x/emergency/keeper/openclaw_emergency_test.go",
		TestFunc: "TestOC_EM7_RevertDepthExceeded",
		Defense:  "MaxRevertDepth enforced; block 0 revert rejected; within-bound revert succeeds",
	},
	{
		ID: "OC-EM-8", Name: "Resume from Normal State", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "emergency", File: "x/emergency/keeper/openclaw_emergency_test.go",
		TestFunc: "TestOC_EM8_ResumeFromNormalState",
		Defense:  "ProposeResume and ProposeRevert rejected when status is normal",
	},
	{
		ID: "OC-EM-9", Name: "Quorum Manipulation", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "emergency", File: "x/emergency/keeper/openclaw_emergency_test.go",
		TestFunc: "TestOC_EM9_QuorumManipulation",
		Defense:  "75% stake quorum enforced; 40% and 60% stay prevote; 80% advances; precommit→halt",
	},
	{
		ID: "OC-EM-10", Name: "Ceremony Timeout Enforcement", Variant: NanoClaw, Severity: High,
		Status: Defended, Module: "emergency", File: "x/emergency/keeper/openclaw_emergency_test.go",
		TestFunc: "TestOC_EM10_CeremonyTimeoutEnforcement",
		Defense:  "Prevote deadline boundary: block 111 active, block 112 fails; status reverts to normal",
	},

	// ====================================================================
	// disputes module — OC-DS-1 through OC-DS-8
	// ====================================================================
	{
		ID: "OC-DS-1", Name: "Double-Vote Injection", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "disputes", File: "x/disputes/keeper/openclaw_disputes_test.go",
		TestFunc: "TestOC_DS1_DoubleVoteInjection",
		Defense:  "CastVote rejects duplicate arbiter votes",
	},
	{
		ID: "OC-DS-2", Name: "Non-Arbiter Vote", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "disputes", File: "x/disputes/keeper/openclaw_disputes_test.go",
		TestFunc: "TestOC_DS2_NonArbiterVote",
		Defense:  "CastVote checks voter is in dispute.Arbiters list",
	},
	{
		ID: "OC-DS-3", Name: "Vote After Deadline", Variant: NanoClaw, Severity: High,
		Status: Defended, Module: "disputes", File: "x/disputes/keeper/openclaw_disputes_test.go",
		TestFunc: "TestOC_DS3_VoteAfterDeadline",
		Defense:  "CastVote checks blockHeight <= VotingDeadline",
	},
	{
		ID: "OC-DS-4", Name: "Evidence After Deadline", Variant: NanoClaw, Severity: High,
		Status: Defended, Module: "disputes", File: "x/disputes/keeper/openclaw_disputes_test.go",
		TestFunc: "TestOC_DS4_EvidenceAfterDeadline",
		Defense:  "CommitEvidence checks blockHeight <= EvidenceDeadline",
	},
	{
		ID: "OC-DS-5", Name: "Unauthorized Submitter Evidence", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "disputes", File: "x/disputes/keeper/openclaw_disputes_test.go",
		TestFunc: "TestOC_DS5_UnauthorizedSubmitterEvidence",
		Defense:  "CommitEvidence verifies submitter is challenger or defender",
	},
	{
		ID: "OC-DS-6", Name: "Commitment-Reveal Mismatch", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "disputes", File: "x/disputes/keeper/openclaw_disputes_test.go",
		TestFunc: "TestOC_DS6_CommitmentRevealMismatch",
		Defense:  "RevealEvidence verifies hash(content+nonce) == commitmentHash",
	},
	{
		ID: "OC-DS-7", Name: "Bond Insufficiency Attack", Variant: ZeroClaw, Severity: High,
		Status: Defended, Module: "disputes", File: "x/disputes/keeper/openclaw_disputes_test.go",
		TestFunc: "TestOC_DS7_BondInsufficiency",
		Defense:  "InitiateDispute rejects zero and negative stake amounts",
	},
	{
		ID: "OC-DS-8", Name: "Max Active Disputes Flooding", Variant: ZeroClaw, Severity: Medium,
		Status: Defended, Module: "disputes", File: "x/disputes/keeper/openclaw_disputes_test.go",
		TestFunc: "TestOC_DS8_MaxActiveDisputesFlooding",
		Defense:  "MaxActiveDisputes param enforced; overflow rejected",
	},

	// ====================================================================
	// ontology module — OC-ON-1 through OC-ON-3
	// ====================================================================
	{
		ID: "OC-ON-1", Name: "Zero Stake Domain Proposal", Variant: ZeroClaw, Severity: High,
		Status: Defended, Module: "ontology", File: "x/ontology/keeper/openclaw_ontology_test.go",
		TestFunc: "TestOC_ON1_ZeroStakeDomainProposal",
		Defense:  "ProposeDomain rejects zero and negative stake amounts",
	},
	{
		ID: "OC-ON-2", Name: "Duplicate Domain Proposal", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "ontology", File: "x/ontology/keeper/openclaw_ontology_test.go",
		TestFunc: "TestOC_ON2_DuplicateDomainProposal",
		Defense:  "ProposeDomain rejects duplicate domain names",
	},
	{
		ID: "OC-ON-3", Name: "Invalid Stratum Reference", Variant: OpenClaw, Severity: Medium,
		Status: Defended, Module: "ontology", File: "x/ontology/keeper/openclaw_ontology_test.go",
		TestFunc: "TestOC_ON3_InvalidStratumReference",
		Defense:  "ProposeDomain validates stratum exists before allowing domain creation",
	},

	// ====================================================================
	// evidence_mgmt module — OC-EV-1 through OC-EV-3
	// ====================================================================
	{
		ID: "OC-EV-1", Name: "Duplicate Evidence Submission", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "evidence_mgmt", File: "x/evidence_mgmt/keeper/openclaw_evidence_test.go",
		TestFunc: "TestOC_EV1_DuplicateEvidenceSubmission",
		Defense:  "Keeper rejects duplicate content hashes per claim",
	},
	{
		ID: "OC-EV-2", Name: "Audit Score Bounds", Variant: NanoClaw, Severity: Medium,
		Status: Defended, Module: "evidence_mgmt", File: "x/evidence_mgmt/keeper/openclaw_evidence_test.go",
		TestFunc: "TestOC_EV2_AuditScoreBounds",
		Defense:  "Score validated at ValidateBasic layer (>1000000 rejected)",
	},
	{
		ID: "OC-EV-3", Name: "Oracle Registration Validation", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "evidence_mgmt", File: "x/evidence_mgmt/keeper/openclaw_evidence_test.go",
		TestFunc: "TestOC_EV3_OracleRegistrationValidation",
		Defense:  "RegisterOracle validates data types and minimum stake",
	},

	// ====================================================================
	// vesting_rewards module — OC-VR-1 through OC-VR-6
	// ====================================================================
	{
		ID: "OC-VR-1", Name: "Double Claim Prevention", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "vesting_rewards", File: "x/vesting_rewards/keeper/openclaw_vesting_test.go",
		TestFunc: "TestOC_VR1_DoubleClaimPrevention",
		Defense:  "ClaimRewards rejects duplicate claims for same vesting schedule",
	},
	{
		ID: "OC-VR-2", Name: "Cliff Bypass Attempt", Variant: NanoClaw, Severity: High,
		Status: Defended, Module: "vesting_rewards", File: "x/vesting_rewards/keeper/openclaw_vesting_test.go",
		TestFunc: "TestOC_VR2_CliffBypassAttempt",
		Defense:  "Cliff period enforced; claims before cliff rejected",
	},
	{
		ID: "OC-VR-3", Name: "Zero Amount Vesting", Variant: ZeroClaw, Severity: High,
		Status: Loophole, Module: "vesting_rewards", File: "x/vesting_rewards/keeper/openclaw_vesting_test.go",
		TestFunc: "TestOC_VR3_ZeroAmountVesting",
		Defense:  "LOOPHOLE: zero amount accepted at keeper level (ValidateBasic blocks it but keeper doesn't)",
	},
	{
		ID: "OC-VR-4", Name: "Negative Amount Vesting", Variant: NanoClaw, Severity: High,
		Status: Loophole, Module: "vesting_rewards", File: "x/vesting_rewards/keeper/openclaw_vesting_test.go",
		TestFunc: "TestOC_VR4_NegativeStakeVesting",
		Defense:  "LOOPHOLE: negative amounts accepted by keeper (ValidateBasic blocks at msg layer, but keeper lacks defense-in-depth)",
	},
	{
		ID: "OC-VR-5", Name: "Pause/Resume Timing Exploit", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "vesting_rewards", File: "x/vesting_rewards/keeper/openclaw_vesting_test.go",
		TestFunc: "TestOC_VR5_PauseResumeTimingExploit",
		Defense:  "Pause/resume timing correctly tracked; no extra vesting during pause",
	},
	{
		ID: "OC-VR-6", Name: "Falsify Completed Schedule", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "vesting_rewards", File: "x/vesting_rewards/keeper/openclaw_vesting_test.go",
		TestFunc: "TestOC_VR6_FalsifyCompletedSchedule",
		Defense:  "Completed schedules can be falsified (retroactive clawback is by design)",
	},

	// ====================================================================
	// partnerships module — OC-PT-1 through OC-PT-6
	// ====================================================================
	{
		ID: "OC-PT-1", Name: "Non-Participant Deposit", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "partnerships", File: "x/partnerships/keeper/openclaw_partnerships_test.go",
		TestFunc: "TestOC_PT1_NonParticipantDeposit",
		Defense:  "Deposit rejected from non-participants",
	},
	{
		ID: "OC-PT-2", Name: "Zero Amount Deposit", Variant: ZeroClaw, Severity: Medium,
		Status: Defended, Module: "partnerships", File: "x/partnerships/keeper/openclaw_partnerships_test.go",
		TestFunc: "TestOC_PT2_ZeroAmountDeposit",
		Defense:  "Zero amount rejected at ValidateBasic and handler",
	},
	{
		ID: "OC-PT-3", Name: "Self-Approval Operation", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "partnerships", File: "x/partnerships/keeper/openclaw_partnerships_test.go",
		TestFunc: "TestOC_PT3_SelfApprovalOperation",
		Defense:  "Self-approval prevented; requires different counterparty",
	},
	{
		ID: "OC-PT-4", Name: "Formation Window Expiry", Variant: NanoClaw, Severity: Medium,
		Status: Defended, Module: "partnerships", File: "x/partnerships/keeper/openclaw_partnerships_test.go",
		TestFunc: "TestOC_PT4_FormationWindowExpiry",
		Defense:  "Formation window enforced; actions after expiry rejected",
	},
	{
		ID: "OC-PT-5", Name: "Double Acceptance", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "partnerships", File: "x/partnerships/keeper/openclaw_partnerships_test.go",
		TestFunc: "TestOC_PT5_DoubleAcceptance",
		Defense:  "Double acceptance of partnership rejected",
	},
	{
		ID: "OC-PT-6", Name: "Unauthorized Dissolution", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "partnerships", File: "x/partnerships/keeper/openclaw_partnerships_test.go",
		TestFunc: "TestOC_PT6_UnauthorizedDissolution",
		Defense:  "Dissolution requires authorized participant",
	},

	// ====================================================================
	// home module — OC-HM-1 through OC-HM-4
	// ====================================================================
	{
		ID: "OC-HM-1", Name: "Non-Owner Key Registration", Variant: OpenClaw, Severity: Critical,
		Status: Defended, Module: "home", File: "x/home/keeper/openclaw_home_test.go",
		TestFunc: "TestOC_HM1_NonOwnerKeyRegistration",
		Defense:  "Key registration restricted to home owner",
	},
	{
		ID: "OC-HM-2", Name: "Session Limit Exceeded", Variant: ZeroClaw, Severity: Medium,
		Status: Defended, Module: "home", File: "x/home/keeper/openclaw_home_test.go",
		TestFunc: "TestOC_HM2_SessionLimitExceeded",
		Defense:  "MaxSessions per home enforced",
	},
	{
		ID: "OC-HM-3", Name: "Non-Owner Session Start", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "home", File: "x/home/keeper/openclaw_home_test.go",
		TestFunc: "TestOC_HM3_NonOwnerSessionStart",
		Defense:  "Session start requires registered, non-revoked key",
	},
	{
		ID: "OC-HM-4", Name: "Duplicate Home Creation", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "home", File: "x/home/keeper/openclaw_home_test.go",
		TestFunc: "TestOC_HM4_DuplicateHomeCreation",
		Defense:  "Per-account home creation limit enforced",
	},

	// ====================================================================
	// capture_defense module — OC-CD-1 through OC-CD-4
	// ====================================================================
	{
		ID: "OC-CD-1", Name: "Expired Qualification Bypass", Variant: NanoClaw, Severity: High,
		Status: Defended, Module: "capture_defense", File: "x/capture_defense/keeper/openclaw_capturedefense_test.go",
		TestFunc: "TestOC_CD1_ExpiredQualificationCheck",
		Defense:  "Expiration boundary correctly enforced at ExpiresAtBlock+1",
	},
	{
		ID: "OC-CD-2", Name: "Min Accuracy Threshold", Variant: NanoClaw, Severity: High,
		Status: Defended, Module: "capture_defense", File: "x/capture_defense/keeper/openclaw_capturedefense_test.go",
		TestFunc: "TestOC_CD2_MinAccuracyThreshold",
		Defense:  "77% accuracy threshold enforced with correct boundary (76% rejected, 77% accepted)",
	},
	{
		ID: "OC-CD-3", Name: "Duplicate Qualification", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "capture_defense", File: "x/capture_defense/keeper/openclaw_capturedefense_test.go",
		TestFunc: "TestOC_CD3_DuplicateQualification",
		Defense:  "Duplicate qualifications rejected; re-qualification after expiry allowed",
	},
	{
		ID: "OC-CD-4", Name: "Self-Endorsement", Variant: OpenClaw, Severity: High,
		Status: Defended, Module: "capture_defense", File: "x/capture_defense/keeper/openclaw_capturedefense_test.go",
		TestFunc: "TestOC_CD4_SelfEndorsement",
		Defense:  "Self-endorsement filtered from endorser count",
	},

	// ====================================================================
	// Cross-module probes (planned, not yet implemented)
	// ====================================================================
	{
		ID: "XM-1", Name: "Stake-Claim-Unstake Race", Variant: CrossModule, Severity: High,
		Status: NotTested, Module: "staking+knowledge",
		Defense: "Planned: verify unstake during active verification round",
	},
	{
		ID: "XM-2", Name: "Emergency Halt + Active Rounds", Variant: CrossModule, Severity: Critical,
		Status: NotTested, Module: "emergency+knowledge",
		Defense: "Planned: verify rounds resolve correctly during emergency halt",
	},
	{
		ID: "XM-3", Name: "Session Key + BVM + Knowledge", Variant: CrossModule, Severity: High,
		Status: NotTested, Module: "auth+bvm+knowledge",
		Defense: "Planned: verify session capabilities enforce across module boundaries",
	},
	{
		ID: "XM-4", Name: "Governance Parameter + Autopoiesis", Variant: CrossModule, Severity: High,
		Status: NotTested, Module: "gov+autopoiesis",
		Defense: "Planned: verify governance parameter changes interact safely with autopoiesis",
	},
	{
		ID: "XM-5", Name: "Vesting Rewards + Knowledge Tax", Variant: CrossModule, Severity: High,
		Status: NotTested, Module: "vesting_rewards+knowledge",
		Defense: "Planned: verify research tax consistency across reward paths",
	},
	{
		ID: "XM-6", Name: "Frozen Account + Active Claims", Variant: CrossModule, Severity: High,
		Status: NotTested, Module: "auth+knowledge",
		Defense: "Planned: verify claim/round behavior when submitter frozen mid-verification",
	},
	{
		ID: "XM-7", Name: "Validator Demotion + Active Rounds", Variant: CrossModule, Severity: High,
		Status: NotTested, Module: "staking+knowledge",
		Defense: "Planned: verify round completion when validator tier drops mid-round",
	},
	{
		ID: "XM-8", Name: "BVM Contract + Token Transfer", Variant: CrossModule, Severity: Critical,
		Status: NotTested, Module: "bvm+tokens",
		Defense: "Planned: verify BVM value transfer interacts safely with dual-ledger tokens",
	},
}

// ByModule returns all probes targeting the given module.
func ByModule(module string) []Probe {
	var result []Probe
	for _, p := range Catalog {
		if p.Module == module {
			result = append(result, p)
		}
	}
	return result
}

// ByVariant returns all probes of the given variant.
func ByVariant(v Variant) []Probe {
	var result []Probe
	for _, p := range Catalog {
		if p.Variant == v {
			result = append(result, p)
		}
	}
	return result
}

// ByStatus returns all probes with the given status.
func ByStatus(s Status) []Probe {
	var result []Probe
	for _, p := range Catalog {
		if p.Status == s {
			result = append(result, p)
		}
	}
	return result
}

// CoverageByModule returns probe count per module.
func CoverageByModule() map[string]int {
	counts := make(map[string]int)
	for _, p := range Catalog {
		counts[p.Module]++
	}
	return counts
}

// UncoveredModules returns modules with zero probes.
func UncoveredModules() []string {
	covered := CoverageByModule()
	var uncovered []string
	for _, m := range allModules {
		if covered[m] == 0 {
			uncovered = append(uncovered, m)
		}
	}
	return uncovered
}
