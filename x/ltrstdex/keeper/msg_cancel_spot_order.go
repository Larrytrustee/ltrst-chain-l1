package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"ltrstchain/x/ltrstdex/types"
)

// CancelSpotOrder cancels a single resting order. Authority may be
// the order owner or the module gov authority (for HALTED cleanup).
func (k msgServer) CancelSpotOrder(goCtx context.Context, req *types.MsgCancelSpotOrder) (*types.MsgCancelSpotOrderResponse, error) {
	auth, err := sdk.AccAddressFromBech32(req.Authority)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAuthority, "invalid authority: "+err.Error())
	}
	if err := k.CancelOrder(goCtx, auth, req.MarketId, req.OrderId); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSpotOrderCancelled,
		sdk.NewAttribute(types.AttrMarketID, strconv.FormatUint(req.MarketId, 10)),
		sdk.NewAttribute(types.AttrOrderID, req.OrderId),
		sdk.NewAttribute(types.AttrOwner, req.Authority),
	))

	return &types.MsgCancelSpotOrderResponse{}, nil
}
