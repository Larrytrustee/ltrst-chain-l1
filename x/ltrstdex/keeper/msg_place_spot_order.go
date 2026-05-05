package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"ltrstchain/x/ltrstdex/types"
)

// PlaceSpotOrder handles a user's order submission. Calls into the
// core match engine k.PlaceOrder which does market lookup, collateral
// locking, matching, settlement, and resting of any remainder.
func (k msgServer) PlaceSpotOrder(goCtx context.Context, req *types.MsgPlaceSpotOrder) (*types.MsgPlaceSpotOrderResponse, error) {
	owner, err := sdk.AccAddressFromBech32(req.Owner)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAuthority, "invalid owner: "+err.Error())
	}

	result, err := k.PlaceOrder(
		goCtx,
		owner,
		req.MarketId,
		req.Side,
		req.Type,
		req.Tif,
		req.Price,
		req.Quantity,
		req.ExpiresAtBlock,
		req.ClientOrderId,
	)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSpotOrderPlaced,
		sdk.NewAttribute(types.AttrMarketID, strconv.FormatUint(req.MarketId, 10)),
		sdk.NewAttribute(types.AttrOrderID, result.OrderID),
		sdk.NewAttribute(types.AttrOwner, req.Owner),
		sdk.NewAttribute(types.AttrSide, req.Side.String()),
		sdk.NewAttribute(types.AttrPrice, req.Price.String()),
		sdk.NewAttribute(types.AttrQuantity, req.Quantity.String()),
		sdk.NewAttribute(types.AttrFilledQty, result.FilledQuantity.String()),
		sdk.NewAttribute(types.AttrRestingQty, result.RestingQuantity.String()),
	))

	return &types.MsgPlaceSpotOrderResponse{
		OrderId:         result.OrderID,
		FilledQuantity:  result.FilledQuantity,
		RestingQuantity: result.RestingQuantity,
	}, nil
}
