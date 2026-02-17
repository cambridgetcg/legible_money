package types

const (
	// ModuleName defines the module name.
	ModuleName = "schedule"

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
	ProcessKeyPrefix          = []byte{0x01} // {process_id} -> ScheduleProcess
	ByTimeIndexPrefix         = []byte{0x02} // {block_height}/{process_id} -> nil
	ByAccountIndexPrefix      = []byte{0x03} // {account}/{process_id} -> nil
	ByStatusIndexPrefix       = []byte{0x04} // {status}/{process_id} -> nil
	ByLinkedEntityIndexPrefix = []byte{0x05} // {entity_type}/{entity_id}/{process_id} -> nil
	WatchedKeyIndexPrefix     = []byte{0x06} // {state_key}/{process_id} -> nil
	ParamsKey                 = []byte{0x07}
	SequenceKey               = []byte{0x08} // auto-increment counter
)
