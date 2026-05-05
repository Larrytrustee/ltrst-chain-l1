package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"ltrstchain/x/ltrstdex/types"
)

// CancelAllSpotOrders cancels every live order owned by the caller,
// optionally scoped to a single market (market_id == 0 ⇒ all markets).
func (k msgServer) CancelAllSpotOrders(goCtx context.Context, req *types.MsgCancelAllSpotOrders) (*types.MsgCancelAllSpotOrdersResponse, error) {
	owner, err := sdk.AccAddressFromBech32(req.Owner)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAuthority, "invalid owner: "+err.Error())
	}
	count, err := k.CancelAllOrdersForOwner(goCtx, owner, req.MarketId)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSpotOrderCancelled,
		sdk.NewAttribute(types.AttrOwner, req.Owner),
		sdk.NewAttribute(types.AttrMarketID, strconv.FormatUint(req.MarketId, 10)),
		sdk.NewAttribute("cancelled_count", strconv.FormatUint(uint64(count), 10)),
	))

	return &types.MsgCancelAllSpotOrdersResponse{CancelledCount: count}, nil
}
