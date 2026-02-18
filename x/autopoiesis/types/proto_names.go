package types

// XXX_MessageName methods for hand-written types — provides unique proto
// message names so the Cosmos SDK interface registry derives distinct type URLs.

func (*MsgUpdateParams) XXX_MessageName() string             { return "legible.autopoiesis.v1.MsgUpdateParams" }
func (*MsgUpdateParamsResponse) XXX_MessageName() string      { return "legible.autopoiesis.v1.MsgUpdateParamsResponse" }
func (*MsgActivateAutopoiesis) XXX_MessageName() string       { return "legible.autopoiesis.v1.MsgActivateAutopoiesis" }
func (*MsgActivateAutopoiesisResponse) XXX_MessageName() string { return "legible.autopoiesis.v1.MsgActivateAutopoiesisResponse" }
func (*MsgOverrideMultiplier) XXX_MessageName() string        { return "legible.autopoiesis.v1.MsgOverrideMultiplier" }
func (*MsgOverrideMultiplierResponse) XXX_MessageName() string { return "legible.autopoiesis.v1.MsgOverrideMultiplierResponse" }
func (*MsgFreezeMultiplier) XXX_MessageName() string          { return "legible.autopoiesis.v1.MsgFreezeMultiplier" }
func (*MsgFreezeMultiplierResponse) XXX_MessageName() string   { return "legible.autopoiesis.v1.MsgFreezeMultiplierResponse" }
