package types

// XXX_MessageName methods for hand-written types — provides unique proto
// message names so the Cosmos SDK interface registry derives distinct type URLs.

func (*MsgOpenDispute) XXX_MessageName() string              { return "legible.disputes.v1.MsgOpenDispute" }
func (*MsgOpenDisputeResponse) XXX_MessageName() string      { return "legible.disputes.v1.MsgOpenDisputeResponse" }
func (*MsgSubmitEvidence) XXX_MessageName() string            { return "legible.disputes.v1.MsgSubmitEvidence" }
func (*MsgSubmitEvidenceResponse) XXX_MessageName() string    { return "legible.disputes.v1.MsgSubmitEvidenceResponse" }
func (*MsgVoteDispute) XXX_MessageName() string               { return "legible.disputes.v1.MsgVoteDispute" }
func (*MsgVoteDisputeResponse) XXX_MessageName() string       { return "legible.disputes.v1.MsgVoteDisputeResponse" }
func (*MsgUpdateParams) XXX_MessageName() string              { return "legible.disputes.v1.MsgUpdateParams" }
func (*MsgUpdateParamsResponse) XXX_MessageName() string      { return "legible.disputes.v1.MsgUpdateParamsResponse" }
