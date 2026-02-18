package types

// XXX_MessageName methods for hand-written types — provides unique proto
// message names so the Cosmos SDK interface registry derives distinct type URLs.

func (*MsgRegisterProvider) XXX_MessageName() string         { return "legible.billing.v1.MsgRegisterProvider" }
func (*MsgRegisterProviderResponse) XXX_MessageName() string  { return "legible.billing.v1.MsgRegisterProviderResponse" }
func (*MsgDeregisterProvider) XXX_MessageName() string        { return "legible.billing.v1.MsgDeregisterProvider" }
func (*MsgDeregisterProviderResponse) XXX_MessageName() string { return "legible.billing.v1.MsgDeregisterProviderResponse" }
func (*MsgRequestQuote) XXX_MessageName() string              { return "legible.billing.v1.MsgRequestQuote" }
func (*MsgRequestQuoteResponse) XXX_MessageName() string      { return "legible.billing.v1.MsgRequestQuoteResponse" }
func (*MsgExecutePayment) XXX_MessageName() string            { return "legible.billing.v1.MsgExecutePayment" }
func (*MsgExecutePaymentResponse) XXX_MessageName() string    { return "legible.billing.v1.MsgExecutePaymentResponse" }
func (*MsgUpdateParams) XXX_MessageName() string              { return "legible.billing.v1.MsgUpdateParams" }
func (*MsgUpdateParamsResponse) XXX_MessageName() string      { return "legible.billing.v1.MsgUpdateParamsResponse" }
