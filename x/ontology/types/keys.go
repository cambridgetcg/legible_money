package types

const (
	// ModuleName defines the module name.
	ModuleName = "ontology"

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
	ParamsKey             = []byte{0x00}
	StratumKeyPrefix      = []byte{0x01} // {stratum_id} -> StratumProperties
	DomainKeyPrefix       = []byte{0x02} // {domain_name} -> Domain
	ProposalKeyPrefix     = []byte{0x03} // {proposal_id} -> DomainProposal
	LinkKeyPrefix         = []byte{0x04} // {source}/{target} -> CrossStratumLink
	DomainByStratumPrefix = []byte{0x05} // {stratum}/{domain} -> nil (index)
)
