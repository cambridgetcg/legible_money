package types

// PoT core query types are now proto-generated in query.pb.go.
// Only non-proto query types remain here.

// ---- Negative Knowledge Extended Query Types (not yet in proto) ----

import "fmt"

type QueryCounterFactsByFactRequest struct {
	FactId string `json:"fact_id"`
}

func (r *QueryCounterFactsByFactRequest) Reset() {}
func (r *QueryCounterFactsByFactRequest) String() string {
	return fmt.Sprintf("QueryCounterFactsByFact{%s}", r.FactId)
}
func (r *QueryCounterFactsByFactRequest) ProtoMessage() {}

type QueryCounterFactsByFactResponse struct {
	CounterFacts []*CounterFact `json:"counter_facts"`
}

func (r *QueryCounterFactsByFactResponse) Reset()         {}
func (r *QueryCounterFactsByFactResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryCounterFactsByFactResponse) ProtoMessage()  {}

type QueryBidirectionalConfidenceRequest struct {
	FactId string `json:"fact_id"`
}

func (r *QueryBidirectionalConfidenceRequest) Reset() {}
func (r *QueryBidirectionalConfidenceRequest) String() string {
	return fmt.Sprintf("QueryBidirConf{%s}", r.FactId)
}
func (r *QueryBidirectionalConfidenceRequest) ProtoMessage() {}

type QueryBidirectionalConfidenceResponse struct {
	BidirectionalConfidence *BidirectionalConfidence `json:"bidirectional_confidence"`
}

func (r *QueryBidirectionalConfidenceResponse) Reset()         {}
func (r *QueryBidirectionalConfidenceResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryBidirectionalConfidenceResponse) ProtoMessage()  {}

type QueryCounterFactsByDomainRequest struct {
	Domain string `json:"domain"`
}

func (r *QueryCounterFactsByDomainRequest) Reset() {}
func (r *QueryCounterFactsByDomainRequest) String() string {
	return fmt.Sprintf("QueryCounterFactsByDomain{%s}", r.Domain)
}
func (r *QueryCounterFactsByDomainRequest) ProtoMessage() {}

type QueryCounterFactsByDomainResponse struct {
	CounterFacts []*CounterFact `json:"counter_facts"`
}

func (r *QueryCounterFactsByDomainResponse) Reset()         {}
func (r *QueryCounterFactsByDomainResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryCounterFactsByDomainResponse) ProtoMessage()  {}

// ---- Provisional Challenge Query Types ----

type QueryProvisionalChallengeRequest struct {
	ChallengeId string `json:"challenge_id"`
}

func (r *QueryProvisionalChallengeRequest) Reset() {}
func (r *QueryProvisionalChallengeRequest) String() string {
	return fmt.Sprintf("QueryProvisionalChallenge{%s}", r.ChallengeId)
}
func (r *QueryProvisionalChallengeRequest) ProtoMessage() {}

type QueryProvisionalChallengeResponse struct {
	Challenge *ProvisionalChallenge `json:"challenge"`
}

func (r *QueryProvisionalChallengeResponse) Reset()         {}
func (r *QueryProvisionalChallengeResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryProvisionalChallengeResponse) ProtoMessage()  {}

type QueryChallengesByClaimRequest struct {
	ClaimId    string `json:"claim_id"`
	StatusFilter string `json:"status_filter,omitempty"` // "", "open", "resolved"
}

func (r *QueryChallengesByClaimRequest) Reset() {}
func (r *QueryChallengesByClaimRequest) String() string {
	return fmt.Sprintf("QueryChallengesByClaim{%s}", r.ClaimId)
}
func (r *QueryChallengesByClaimRequest) ProtoMessage() {}

type QueryChallengesByClaimResponse struct {
	Challenges []*ProvisionalChallenge `json:"challenges"`
}

func (r *QueryChallengesByClaimResponse) Reset()         {}
func (r *QueryChallengesByClaimResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryChallengesByClaimResponse) ProtoMessage()  {}
