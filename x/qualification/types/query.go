package types

import "context"

// QueryServer defines the qualification module's query server interface.
type QueryServer interface {
	Qualification(context.Context, *QueryQualificationRequest) (*QueryQualificationResponse, error)
	QualificationsByValidator(context.Context, *QueryQualificationsByValidatorRequest) (*QueryQualificationsByValidatorResponse, error)
	QualifiedValidators(context.Context, *QueryQualifiedValidatorsRequest) (*QueryQualifiedValidatorsResponse, error)
	Endorsement(context.Context, *QueryEndorsementRequest) (*QueryEndorsementResponse, error)
	EndorsementsForValidator(context.Context, *QueryEndorsementsForValidatorRequest) (*QueryEndorsementsForValidatorResponse, error)
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
}

type QueryQualificationRequest struct {
	ValidatorAddr string `json:"validator_addr"`
	Domain        string `json:"domain"`
}

type QueryQualificationResponse struct {
	Qualification DomainQualification `json:"qualification"`
	Found         bool                `json:"found"`
}

type QueryQualificationsByValidatorRequest struct {
	ValidatorAddr string `json:"validator_addr"`
}

type QueryQualificationsByValidatorResponse struct {
	Qualifications []DomainQualification `json:"qualifications"`
}

type QueryQualifiedValidatorsRequest struct {
	Domain string `json:"domain"`
}

type QueryQualifiedValidatorsResponse struct {
	Validators []string `json:"validators"`
}

type QueryEndorsementRequest struct {
	EndorsementId string `json:"endorsement_id"`
}

type QueryEndorsementResponse struct {
	Endorsement QualificationEndorsement `json:"endorsement"`
	Found       bool                     `json:"found"`
}

type QueryEndorsementsForValidatorRequest struct {
	TargetValidator string `json:"target_validator"`
}

type QueryEndorsementsForValidatorResponse struct {
	Endorsements []QualificationEndorsement `json:"endorsements"`
}

type QueryParamsRequest struct{}

type QueryParamsResponse struct {
	Params Params `json:"params"`
}
