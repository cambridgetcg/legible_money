package types

// XXX_MessageName methods for hand-written types — provides unique proto
// message names so the Cosmos SDK interface registry derives distinct type URLs.

func (*MsgDepositChannel) XXX_MessageName() string               { return "legible.channels.v1.MsgDepositChannel" }
func (*MsgDepositChannelResponse) XXX_MessageName() string       { return "legible.channels.v1.MsgDepositChannelResponse" }
func (*MsgUpdateState) XXX_MessageName() string                  { return "legible.channels.v1.MsgUpdateState" }
func (*MsgUpdateStateResponse) XXX_MessageName() string          { return "legible.channels.v1.MsgUpdateStateResponse" }
func (*MsgClaimExpired) XXX_MessageName() string                 { return "legible.channels.v1.MsgClaimExpired" }
func (*MsgClaimExpiredResponse) XXX_MessageName() string         { return "legible.channels.v1.MsgClaimExpiredResponse" }
func (*MsgUpdateParams) XXX_MessageName() string                 { return "legible.channels.v1.MsgUpdateParams" }
func (*MsgUpdateParamsResponse) XXX_MessageName() string         { return "legible.channels.v1.MsgUpdateParamsResponse" }
