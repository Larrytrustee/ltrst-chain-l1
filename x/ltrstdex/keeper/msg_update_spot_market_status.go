package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"ltrstchain/x/ltrstdex/types"
)

// UpdateSpotMarketStatus transitions a market to a new lifecycle
// state. Always gov-only regardless of GovOnlyMarketCreation, since
// pausing/halting trading is a fundamentally privileged operation.
func (k msgServer) UpdateSpotMarketStatus(goCtx context.Context, req *types.MsgUpdateSpotMarketStatus) (*types.MsgUpdateSpotMarketStatusResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidAuthority,
			"invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}
	if err := k.UpdateMarketStatus(goCtx, req.MarketId, req.NewStatus); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSpotMarketStatus,
		sdk.NewAttribute(types.AttrMarketID, strconv.FormatUint(req.MarketId, 10)),
		sdk.NewAttribute(types.AttrStatus, req.NewStatus.String()),
	))

	return &types.MsgUpdateSpotMarketStatusResponse{}, nil
}
