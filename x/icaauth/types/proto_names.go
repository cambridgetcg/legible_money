package types

// XXX_MessageName methods for hand-written types — provides unique proto
// message names so the Cosmos SDK interface registry derives distinct type URLs.

func (*MsgRegisterInterchainAccount) XXX_MessageName() string            { return "legible.icaauth.v1.MsgRegisterInterchainAccount" }
func (*MsgRegisterInterchainAccountResponse) XXX_MessageName() string    { return "legible.icaauth.v1.MsgRegisterInterchainAccountResponse" }
func (*MsgSendICATransaction) XXX_MessageName() string                   { return "legible.icaauth.v1.MsgSendICATransaction" }
func (*MsgSendICATransactionResponse) XXX_MessageName() string           { return "legible.icaauth.v1.MsgSendICATransactionResponse" }
func (*MsgUpdateParams) XXX_MessageName() string                         { return "legible.icaauth.v1.MsgUpdateParams" }
func (*MsgUpdateParamsResponse) XXX_MessageName() string                 { return "legible.icaauth.v1.MsgUpdateParamsResponse" }
