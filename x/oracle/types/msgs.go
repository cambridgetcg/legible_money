package types

// MsgSubmitPrediction is sent to stake a future claim.
type MsgSubmitPrediction struct {
	Submitter          string `json:"submitter"`
	Claim              string `json:"claim"`
	Domain             string `json:"domain"`
	Probability        uint64 `json:"probability"`        // bps [1, 999_999]
	StakeAmount        uint64 `json:"stake_amount"`       // ulgm
	ResolutionType     string `json:"resolution_type"`
	ResolutionRef      string `json:"resolution_ref"`     // optional fact/feed ID
	ResolutionCriteria string `json:"resolution_criteria"`
	ResolutionBlock    uint64 `json:"resolution_block"`
}
