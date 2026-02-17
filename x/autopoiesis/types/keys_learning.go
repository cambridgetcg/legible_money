package types

// Store key prefixes for meta-learning (Loop 2) and auto-proposals (Loop 4).
var (
	LearningRecordKeyPrefix = []byte{0x08} // path → LearningRecord (JSON)
	LearningWindowKeyPrefix = []byte{0x09} // epoch → LearningWindow (JSON)
	MetaLearningStateKey    = []byte{0x0a} // MetaLearningState (JSON)
	ProposalTriggerKeyPrefix = []byte{0x0b} // trigger_type → ProposalTriggerState (JSON)
	AutoProposalStateKey     = []byte{0x0c} // AutoProposalState (JSON)
)
