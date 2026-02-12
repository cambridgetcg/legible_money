package types

const (
	// ModuleName is the name of the capture defense module.
	ModuleName = "capture_defense"

	// StoreKey is the KV store key for the capture defense module.
	StoreKey = ModuleName

	// RouterKey is the message routing key for the capture defense module.
	RouterKey = ModuleName
)

// Store key prefixes for sub-stores.
var (
	QualificationKeyPrefix       = []byte{0x01}
	DomainReputationKeyPrefix    = []byte{0x02}
	StratumReputationKeyPrefix   = []byte{0x03}
	GlobalReputationKeyPrefix    = []byte{0x04}
	VerificationHistoryKeyPrefix = []byte{0x05}
	CaptureMetricsKeyPrefix      = []byte{0x06}
	ParamsKeyPrefix              = []byte{0x07}
	DomainPauseKeyPrefix         = []byte{0x08}
)

// QualificationKey returns the store key for a validator's qualification in a domain.
func QualificationKey(validatorAddr, domain string) []byte {
	key := append(QualificationKeyPrefix, []byte(validatorAddr)...)
	key = append(key, byte('/'))
	key = append(key, []byte(domain)...)
	return key
}

// QualificationsByValidatorPrefix returns the prefix for all qualifications of a validator.
func QualificationsByValidatorPrefix(validatorAddr string) []byte {
	key := append(QualificationKeyPrefix, []byte(validatorAddr)...)
	key = append(key, byte('/'))
	return key
}

// GlobalReputationKey returns the store key for a validator's global reputation.
func GlobalReputationKey(validatorAddr string) []byte {
	return append(GlobalReputationKeyPrefix, []byte(validatorAddr)...)
}

// DomainReputationKey returns the store key for a validator's domain reputation.
func DomainReputationKey(validatorAddr, domain string) []byte {
	key := append(DomainReputationKeyPrefix, []byte(validatorAddr)...)
	key = append(key, byte('/'))
	key = append(key, []byte(domain)...)
	return key
}

// StratumReputationKey returns the store key for a validator's stratum reputation.
func StratumReputationKey(validatorAddr string, stratum uint32) []byte {
	key := append(StratumReputationKeyPrefix, []byte(validatorAddr)...)
	key = append(key, byte('/'))
	key = append(key, byte(stratum))
	return key
}

// VerificationHistoryKey returns the store key for a domain's verification history.
func VerificationHistoryKey(domain string) []byte {
	return append(VerificationHistoryKeyPrefix, []byte(domain)...)
}

// CaptureMetricsKey returns the store key for a domain's capture metrics.
func CaptureMetricsKey(domain string) []byte {
	return append(CaptureMetricsKeyPrefix, []byte(domain)...)
}

// DomainPauseKey returns the store key for a domain's pause-until block.
func DomainPauseKey(domain string) []byte {
	return append(DomainPauseKeyPrefix, []byte(domain)...)
}
