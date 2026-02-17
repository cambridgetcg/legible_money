package types

// XXX_MessageName methods for hand-written types — provides unique proto
// message names so the Cosmos SDK interface registry derives distinct type URLs.

func (*MsgUpdateValidatorStake) XXX_MessageName() string             { return "legible.staking.v1.MsgUpdateValidatorStake" }
func (*MsgUpdateValidatorStakeResponse) XXX_MessageName() string     { return "legible.staking.v1.MsgUpdateValidatorStakeResponse" }
func (*MsgDelegate) XXX_MessageName() string                         { return "legible.staking.v1.MsgDelegate" }
func (*MsgDelegateResponse) XXX_MessageName() string                 { return "legible.staking.v1.MsgDelegateResponse" }
func (*MsgUndelegate) XXX_MessageName() string                       { return "legible.staking.v1.MsgUndelegate" }
func (*MsgUndelegateResponse) XXX_MessageName() string               { return "legible.staking.v1.MsgUndelegateResponse" }
func (*MsgRedelegate) XXX_MessageName() string                       { return "legible.staking.v1.MsgRedelegate" }
func (*MsgRedelegateResponse) XXX_MessageName() string               { return "legible.staking.v1.MsgRedelegateResponse" }
func (*MsgUpdateParams) XXX_MessageName() string                     { return "legible.staking.v1.MsgUpdateParams" }
func (*MsgUpdateParamsResponse) XXX_MessageName() string             { return "legible.staking.v1.MsgUpdateParamsResponse" }
