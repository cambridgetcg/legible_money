package types

const (
	// ModuleName defines the module name.
	ModuleName = "qualification"

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
	// 0x01 | validatorAddr | "/" | domain → DomainQualification JSON
	QualificationKeyPrefix = []byte{0x01}

	// 0x02 | endorsementId → QualificationEndorsement JSON
	EndorsementKeyPrefix = []byte{0x02}

	// 0x05 → Params JSON
	ParamsKey = []byte{0x05}

	// 0x10 | domain | "/" | validatorAddr → [] (domain→validators index)
	DomainValidatorIndexPrefix = []byte{0x10}

	// 0x11 | endorserAddr | "/" | endorsementId → [] (endorser index)
	EndorserIndexPrefix = []byte{0x11}

	// 0x12 | targetAddr | "/" | endorsementId → [] (target validator index)
	TargetIndexPrefix = []byte{0x12}

	// 0x20 → endorsement counter (uint64 big-endian)
	EndorsementCounterKey = []byte{0x20}
)

// QualificationKey returns the store key for a validator's domain qualification.
func QualificationKey(validatorAddr, domain string) []byte {
	key := append(QualificationKeyPrefix, []byte(validatorAddr)...)
	key = append(key, '/')
	key = append(key, []byte(domain)...)
	return key
}

// EndorsementKey returns the store key for an endorsement by ID.
func EndorsementKey(id string) []byte {
	return append(EndorsementKeyPrefix, []byte(id)...)
}

// DomainValidatorIndexKey returns the index key for domain→validator lookup.
func DomainValidatorIndexKey(domain, validatorAddr string) []byte {
	key := append(DomainValidatorIndexPrefix, []byte(domain)...)
	key = append(key, '/')
	key = append(key, []byte(validatorAddr)...)
	return key
}

// DomainValidatorPrefix returns the prefix for iterating all validators in a domain.
func DomainValidatorPrefix(domain string) []byte {
	key := append(DomainValidatorIndexPrefix, []byte(domain)...)
	key = append(key, '/')
	return key
}

// EndorserIndexKey returns the index key for endorser→endorsement lookup.
func EndorserIndexKey(endorserAddr, endorsementId string) []byte {
	key := append(EndorserIndexPrefix, []byte(endorserAddr)...)
	key = append(key, '/')
	key = append(key, []byte(endorsementId)...)
	return key
}

// TargetIndexKey returns the index key for target→endorsement lookup.
func TargetIndexKey(targetAddr, endorsementId string) []byte {
	key := append(TargetIndexPrefix, []byte(targetAddr)...)
	key = append(key, '/')
	key = append(key, []byte(endorsementId)...)
	return key
}

// TargetIndexPrefix returns the prefix for iterating all endorsements targeting a validator.
func TargetValidatorPrefix(targetAddr string) []byte {
	key := append(TargetIndexPrefix, []byte(targetAddr)...)
	key = append(key, '/')
	return key
}

// ValidatorQualificationPrefix returns the prefix for iterating all qualifications of a validator.
func ValidatorQualificationPrefix(validatorAddr string) []byte {
	key := append(QualificationKeyPrefix, []byte(validatorAddr)...)
	key = append(key, '/')
	return key
}
