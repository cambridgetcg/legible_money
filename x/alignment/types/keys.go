package types

const (
	ModuleName = "alignment"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

// Store key prefixes.
var (
	ParamsKey              = []byte{0x01} // Params (JSON)
	StateKey               = []byte{0x02} // AlignmentState (active, currentEpoch)
	ObservationKeyPrefix   = []byte{0x03} // epoch → AlignmentObservation (JSON)
	ScoreKeyPrefix         = []byte{0x04} // epoch → DimensionScores (JSON)
	CorrectionKeyPrefix    = []byte{0x05} // epoch → []CorrectionRecord (JSON)
	MetaKeyPrefix          = []byte{0x06} // epoch → AlignmentHealthIndex (JSON)
	CorrectionHistPrefix   = []byte{0x07} // epoch → []CorrectionOutcome for meta-loop tracking
)
