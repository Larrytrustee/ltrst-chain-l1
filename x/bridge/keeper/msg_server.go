package keeper

import "ltrstchain/x/bridge/types"

// msgServer is the MsgServer implementation wrapping Keeper. We
// expose one method per file following the x/ltrstdex convention.
type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns a MsgServer over the given keeper.
func NewMsgServerImpl(k Keeper) types.MsgServer { return &msgServer{Keeper: k} }

var _ types.MsgServer = msgServer{}
