package types

import "context"

// MsgServer defines the autopoiesis message server interface.
type MsgServer interface {
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
	Activate(context.Context, *MsgActivateAutopoiesis) (*MsgActivateAutopoiesisResponse, error)
	OverrideMultiplier(context.Context, *MsgOverrideMultiplier) (*MsgOverrideMultiplierResponse, error)
	FreezeMultiplier(context.Context, *MsgFreezeMultiplier) (*MsgFreezeMultiplierResponse, error)
}
