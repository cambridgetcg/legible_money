package types

import (
	"fmt"
)

// ---- Fact/Claim/Round Query Types (not in proto) ----

type QueryFactRequest struct {
	FactId string `json:"fact_id"`
}

func (r *QueryFactRequest) Reset()        {}
func (r *QueryFactRequest) String() string { return fmt.Sprintf("QueryFactRequest{%s}", r.FactId) }
func (r *QueryFactRequest) ProtoMessage()  {}

type QueryFactResponse struct {
	Fact *Fact `json:"fact"`
}

func (r *QueryFactResponse) Reset()        {}
func (r *QueryFactResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryFactResponse) ProtoMessage()  {}

type QueryFactsRequest struct {
	Limit  uint32 `json:"limit"`
	Offset uint32 `json:"offset"`
}

func (r *QueryFactsRequest) Reset()        {}
func (r *QueryFactsRequest) String() string { return "QueryFactsRequest" }
func (r *QueryFactsRequest) ProtoMessage()  {}

type QueryFactsResponse struct {
	Facts []*Fact `json:"facts"`
	Total uint64  `json:"total"`
}

func (r *QueryFactsResponse) Reset()        {}
func (r *QueryFactsResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryFactsResponse) ProtoMessage()  {}

type QueryFactsByDomainRequest struct {
	Domain string `json:"domain"`
}

func (r *QueryFactsByDomainRequest) Reset()        {}
func (r *QueryFactsByDomainRequest) String() string { return fmt.Sprintf("QueryFactsByDomain{%s}", r.Domain) }
func (r *QueryFactsByDomainRequest) ProtoMessage()  {}

type QueryFactsByDomainResponse struct {
	Facts []*Fact `json:"facts"`
}

func (r *QueryFactsByDomainResponse) Reset()        {}
func (r *QueryFactsByDomainResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryFactsByDomainResponse) ProtoMessage()  {}

type QueryClaimRequest struct {
	ClaimId string `json:"claim_id"`
}

func (r *QueryClaimRequest) Reset()        {}
func (r *QueryClaimRequest) String() string { return fmt.Sprintf("QueryClaimRequest{%s}", r.ClaimId) }
func (r *QueryClaimRequest) ProtoMessage()  {}

type QueryClaimResponse struct {
	Claim *Claim `json:"claim"`
}

func (r *QueryClaimResponse) Reset()        {}
func (r *QueryClaimResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryClaimResponse) ProtoMessage()  {}

type QueryVerificationRoundRequest struct {
	RoundId string `json:"round_id"`
}

func (r *QueryVerificationRoundRequest) Reset()        {}
func (r *QueryVerificationRoundRequest) String() string { return fmt.Sprintf("QueryRound{%s}", r.RoundId) }
func (r *QueryVerificationRoundRequest) ProtoMessage()  {}

type QueryVerificationRoundResponse struct {
	Round *VerificationRound `json:"round"`
}

func (r *QueryVerificationRoundResponse) Reset()        {}
func (r *QueryVerificationRoundResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryVerificationRoundResponse) ProtoMessage()  {}

type QueryActiveRoundsRequest struct{}

func (r *QueryActiveRoundsRequest) Reset()        {}
func (r *QueryActiveRoundsRequest) String() string { return "QueryActiveRoundsRequest" }
func (r *QueryActiveRoundsRequest) ProtoMessage()  {}

type QueryActiveRoundsResponse struct {
	Rounds []*VerificationRound `json:"rounds"`
}

func (r *QueryActiveRoundsResponse) Reset()        {}
func (r *QueryActiveRoundsResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryActiveRoundsResponse) ProtoMessage()  {}

// ---- Negative Knowledge Query Types (not in proto) ----

type QueryCounterFactRequest struct {
	CounterFactId string `json:"counter_fact_id"`
}

func (r *QueryCounterFactRequest) Reset()        {}
func (r *QueryCounterFactRequest) String() string { return fmt.Sprintf("QueryCounterFact{%s}", r.CounterFactId) }
func (r *QueryCounterFactRequest) ProtoMessage()  {}

type QueryCounterFactResponse struct {
	CounterFact *CounterFact `json:"counter_fact"`
}

func (r *QueryCounterFactResponse) Reset()        {}
func (r *QueryCounterFactResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryCounterFactResponse) ProtoMessage()  {}

type QueryCounterFactsByFactRequest struct {
	FactId string `json:"fact_id"`
}

func (r *QueryCounterFactsByFactRequest) Reset()        {}
func (r *QueryCounterFactsByFactRequest) String() string { return fmt.Sprintf("QueryCounterFactsByFact{%s}", r.FactId) }
func (r *QueryCounterFactsByFactRequest) ProtoMessage()  {}

type QueryCounterFactsByFactResponse struct {
	CounterFacts []*CounterFact `json:"counter_facts"`
}

func (r *QueryCounterFactsByFactResponse) Reset()        {}
func (r *QueryCounterFactsByFactResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryCounterFactsByFactResponse) ProtoMessage()  {}

type QueryBidirectionalConfidenceRequest struct {
	FactId string `json:"fact_id"`
}

func (r *QueryBidirectionalConfidenceRequest) Reset()        {}
func (r *QueryBidirectionalConfidenceRequest) String() string { return fmt.Sprintf("QueryBidirConf{%s}", r.FactId) }
func (r *QueryBidirectionalConfidenceRequest) ProtoMessage()  {}

type QueryBidirectionalConfidenceResponse struct {
	BidirectionalConfidence *BidirectionalConfidence `json:"bidirectional_confidence"`
}

func (r *QueryBidirectionalConfidenceResponse) Reset()        {}
func (r *QueryBidirectionalConfidenceResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryBidirectionalConfidenceResponse) ProtoMessage()  {}

type QueryCounterFactsByDomainRequest struct {
	Domain string `json:"domain"`
}

func (r *QueryCounterFactsByDomainRequest) Reset()        {}
func (r *QueryCounterFactsByDomainRequest) String() string { return fmt.Sprintf("QueryCounterFactsByDomain{%s}", r.Domain) }
func (r *QueryCounterFactsByDomainRequest) ProtoMessage()  {}

type QueryCounterFactsByDomainResponse struct {
	CounterFacts []*CounterFact `json:"counter_facts"`
}

func (r *QueryCounterFactsByDomainResponse) Reset()        {}
func (r *QueryCounterFactsByDomainResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryCounterFactsByDomainResponse) ProtoMessage()  {}
