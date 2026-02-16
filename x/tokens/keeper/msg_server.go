package keeper

import (
	"github.com/legible-money/legible/x/tokens/types"
)

type msgServer struct {
	Keeper
	types.UnimplementedMsgServer
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
// All 14 LGM-20 methods return "not implemented" via UnimplementedMsgServer
// until the secondary token system is built out.
func NewMsgServerImpl(keeper Keeper) *msgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = (*msgServer)(nil)
