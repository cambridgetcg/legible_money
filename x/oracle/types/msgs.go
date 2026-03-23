package types

import "fmt"

// MsgSubmitPrediction is sent to stake a future claim.
type MsgSubmitPrediction struct {
	Submitter          string `protobuf:"bytes,1,opt,name=submitter,proto3" json:"submitter"`
	Claim              string `protobuf:"bytes,2,opt,name=claim,proto3" json:"claim"`
	Domain             string `protobuf:"bytes,3,opt,name=domain,proto3" json:"domain"`
	Probability        uint64 `protobuf:"varint,4,opt,name=probability,proto3" json:"probability"`
	StakeAmount        uint64 `protobuf:"varint,5,opt,name=stake_amount,json=stakeAmount,proto3" json:"stake_amount"`
	ResolutionType     string `protobuf:"bytes,6,opt,name=resolution_type,json=resolutionType,proto3" json:"resolution_type"`
	ResolutionRef      string `protobuf:"bytes,7,opt,name=resolution_ref,json=resolutionRef,proto3" json:"resolution_ref"`
	ResolutionCriteria string `protobuf:"bytes,8,opt,name=resolution_criteria,json=resolutionCriteria,proto3" json:"resolution_criteria"`
	ResolutionBlock    uint64 `protobuf:"varint,9,opt,name=resolution_block,json=resolutionBlock,proto3" json:"resolution_block"`
}

// Proto interface implementation (minimal — for CLI use)
func (m *MsgSubmitPrediction) Reset()         {}
func (m *MsgSubmitPrediction) String() string { return fmt.Sprintf("%+v", *m) }
func (m *MsgSubmitPrediction) ProtoMessage()  {}

// ValidateBasic performs basic validation.
func (m *MsgSubmitPrediction) ValidateBasic() error {
	if m.Submitter == "" {
		return fmt.Errorf("submitter cannot be empty")
	}
	if m.Probability == 0 || m.Probability >= 1_000_000 {
		return fmt.Errorf("probability must be strictly between 0 and 1,000,000 bps")
	}
	if m.StakeAmount == 0 {
		return fmt.Errorf("stake_amount cannot be zero")
	}
	if m.ResolutionBlock == 0 {
		return fmt.Errorf("resolution_block cannot be zero")
	}
	return nil
}

// XXX_MessageName implements the proto name interface for Cosmos SDK registration.
func (*MsgSubmitPrediction) XXX_MessageName() string { return "legible.oracle.v1.MsgSubmitPrediction" }
