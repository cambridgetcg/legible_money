package types

const (
	ModuleName = "oracle"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	PredictionKeyPrefix  = []byte{0x01} // oracle/{id}
	ScoreboardKeyPrefix  = []byte{0x02} // oracle/scores/{address}
	ResolutionKeyPrefix  = []byte{0x03} // oracle/resolve/{id}
	ParamKeyPrefix       = []byte{0x04} // oracle/params
)
