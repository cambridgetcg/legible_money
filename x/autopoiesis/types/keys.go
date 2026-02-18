package types

const (
	ModuleName = "autopoiesis"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

// Store key prefixes.
var (
	MultiplierKeyPrefix  = []byte{0x01} // path → MultiplierEntry (JSON)
	ObservationKeyPrefix = []byte{0x02} // epoch → EpochObservation (JSON)
	HormoneKeyPrefix     = []byte{0x03} // epoch → HormoneValues (JSON)
	AdaptationKeyPrefix  = []byte{0x04} // epoch → []Adaptation (JSON)
	ParamsKey            = []byte{0x05} // Params (JSON)
	StateKey             = []byte{0x06} // AutopoiesisState (active, currentEpoch)
	DVITrackingKeyPrefix = []byte{0x07} // domain → DVIValue (consecutive negative epoch tracking)
	FrozenKeyPrefix      = []byte{0x08} // path → frozen flag (0x01 = frozen)
	SSIKeyPrefix         = []byte{0x09} // epoch → SSIValue (JSON)
)
