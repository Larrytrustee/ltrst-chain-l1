package keeper

import (
	"ltrstchain/x/ltrstdex/types"
)

// msgServer is the x/ltrstdex MsgServer implementation. It's a thin
// wrapper around Keeper — all real logic lives on Keeper methods
// (CreateMarket, UpdateMarketStatus, PlaceOrder, CancelOrder, …).
// Handler code per-message lives in msg_*.go files in this package.
type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns a types.MsgServer backed by the given keeper.
func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}

// Compile-time interface assertion.
var _ types.MsgServer = msgServer{}
