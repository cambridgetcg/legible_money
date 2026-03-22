package types

import "fmt"

const (
	// ModuleName defines the module name.
	ModuleName = "home"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey defines the routing key.
	RouterKey = ModuleName

	// QuerierRoute defines the querier route.
	QuerierRoute = ModuleName
)

// Store key prefixes.
var (
	HomeKeyPrefix         = []byte{0x01} // 0x01 + DID string
	SessionKeyPrefix      = []byte{0x02} // 0x02 + homeId/sessionId
	KeyRegistrationPrefix = []byte{0x03} // 0x03 + homeId/keyHash
	AlertKeyPrefix        = []byte{0x04} // 0x04 + homeId/alertId
	SpendingLimitPrefix   = []byte{0x05} // 0x05 + homeId/keyType
	HomeByOwnerPrefix     = []byte{0x06} // 0x06 + ownerAddr (index)
	ParamsKey             = []byte{0x07}
	HomeCounterKey        = []byte{0x08}
	EpistemicStatsPrefix  = []byte{0x09} // 0x09 + homeId → EpistemicStats JSON
	ComfortDimsPrefix       = []byte{0x0A} // 0x0A + homeId → ComfortDimensions JSON
	HomeByPartnershipPrefix = []byte{0x0B} // 0x0B + partnershipId → homeId (reverse index)
	DefenseRatePrefix       = []byte{0x0C} // 0x0C + homeId/keyId/window → uint64 counter
	DefenseCooldownPrefix   = []byte{0x0D} // 0x0D + homeId/keyId → uint64 last-op block
	DefenseListPrefix       = []byte{0x0E} // 0x0E + homeId/listType → JSON address list
	VoiceConfigPrefix       = []byte{0x0F} // 0x0F + homeId → VoiceConfig JSON
	MultiSigConfigPrefix    = []byte{0x10} // 0x10 + homeId → MultiSigConfig JSON
	PendingApprovalPrefix   = []byte{0x11} // 0x11 + approvalId → PendingApproval JSON
	ContributionPrefix      = []byte{0x12} // 0x12 + homeId + "/" + claimId → ContributionEvent JSON
	CapabilityPrefix        = []byte{0x13} // 0x13 + homeId + "/" + capType → CapabilityEntry JSON
	VouchPrefix             = []byte{0x14} // 0x14 + targetHomeId + "/" + capType + "/" + voucherHomeId → Vouch JSON
)

// CapabilityKey returns the store key for a capability entry.
func CapabilityKey(homeId string, capType uint32) []byte {
	return append(CapabilityPrefix, []byte(fmt.Sprintf("%s/%d", homeId, capType))...)
}

// CapabilityPrefixKey returns the prefix for all capabilities of a home.
func CapabilityPrefixKey(homeId string) []byte {
	return append(CapabilityPrefix, []byte(homeId+"/")...)
}

// VouchKey returns the store key for a vouch.
func VouchKey(targetHomeId string, capType uint32, voucherHomeId string) []byte {
	return append(VouchPrefix, []byte(fmt.Sprintf("%s/%d/%s", targetHomeId, capType, voucherHomeId))...)
}

// VouchCapPrefixKey returns the prefix for all vouches on a target+capType.
func VouchCapPrefixKey(targetHomeId string, capType uint32) []byte {
	return append(VouchPrefix, []byte(fmt.Sprintf("%s/%d/", targetHomeId, capType))...)
}

// ComfortDimsKey returns the store key for a home's comfort dimensions.
func ComfortDimsKey(homeId string) []byte {
	return append(ComfortDimsPrefix, []byte(homeId)...)
}

// HomeKey returns the store key for a home by DID.
func HomeKey(did string) []byte {
	return append(HomeKeyPrefix, []byte(did)...)
}

// SessionKey returns the store key for a session.
func SessionKey(homeId, sessionId string) []byte {
	return append(SessionKeyPrefix, []byte(homeId+"/"+sessionId)...)
}

// SessionPrefixKey returns the prefix for all sessions of a home.
func SessionPrefixKey(homeId string) []byte {
	return append(SessionKeyPrefix, []byte(homeId+"/")...)
}

// KeyRegKey returns the store key for a key registration.
func KeyRegKey(homeId, keyHash string) []byte {
	return append(KeyRegistrationPrefix, []byte(homeId+"/"+keyHash)...)
}

// KeyRegPrefixKey returns the prefix for all key registrations of a home.
func KeyRegPrefixKey(homeId string) []byte {
	return append(KeyRegistrationPrefix, []byte(homeId+"/")...)
}

// AlertKey returns the store key for an alert.
func AlertKey(homeId, alertId string) []byte {
	return append(AlertKeyPrefix, []byte(homeId+"/"+alertId)...)
}

// AlertPrefixKey returns the prefix for all alerts of a home.
func AlertPrefixKey(homeId string) []byte {
	return append(AlertKeyPrefix, []byte(homeId+"/")...)
}

// SpendLimitKey returns the store key for a spending limit.
func SpendLimitKey(homeId, keyType string) []byte {
	return append(SpendingLimitPrefix, []byte(homeId+"/"+keyType)...)
}

// HomeByOwnerKey returns the store key for a home-by-owner index entry.
func HomeByOwnerKey(addr string) []byte {
	return append(HomeByOwnerPrefix, []byte(addr)...)
}

// EpistemicStatsKey returns the store key for a home's epistemic activity stats.
func EpistemicStatsKey(homeId string) []byte {
	return append(EpistemicStatsPrefix, []byte(homeId)...)
}

// HomeByPartnershipKey returns the store key for the partnership→home reverse index.
func HomeByPartnershipKey(partnershipId string) []byte {
	return append(HomeByPartnershipPrefix, []byte(partnershipId)...)
}

// DefenseRateKey returns the store key for a rate-limit counter.
func DefenseRateKey(homeId, keyId string, window uint64) []byte {
	return append(DefenseRatePrefix, []byte(fmt.Sprintf("%s/%s/%d", homeId, keyId, window))...)
}

// DefenseCooldownKey returns the store key for a cooldown last-op timestamp.
func DefenseCooldownKey(homeId, keyId string) []byte {
	return append(DefenseCooldownPrefix, []byte(homeId+"/"+keyId)...)
}

// DefenseListKey returns the store key for a whitelist or blacklist.
func DefenseListKey(homeId, listType string) []byte {
	return append(DefenseListPrefix, []byte(homeId+"/"+listType)...)
}

// VoiceConfigKey returns the store key for a home's voice configuration.
func VoiceConfigKey(homeId string) []byte {
	return append(VoiceConfigPrefix, []byte(homeId)...)
}

// MultiSigConfigKey returns the store key for a home's multi-sig configuration.
func MultiSigConfigKey(homeId string) []byte {
	return append(MultiSigConfigPrefix, []byte(homeId)...)
}

// PendingApprovalKey returns the store key for a pending approval by ID.
func PendingApprovalKey(approvalId string) []byte {
	return append(PendingApprovalPrefix, []byte(approvalId)...)
}

// ContributionKey returns the store key for a contribution event.
func ContributionKey(homeId, claimId string) []byte {
	return append(ContributionPrefix, []byte(homeId+"/"+claimId)...)
}

// ContributionPrefixKey returns the prefix for all contributions of a home.
func ContributionPrefixKey(homeId string) []byte {
	return append(ContributionPrefix, []byte(homeId+"/")...)
}
