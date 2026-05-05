package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"ltrstchain/x/bridge/types"
)

// RemoveBridgeAsset deletes an entry. Authority-only.
func (k msgServer) RemoveBridgeAsset(goCtx context.Context, req *types.MsgRemoveBridgeAsset) (*types.MsgRemoveBridgeAssetResponse, error) {
	if req.Authority != k.GetAuthority() {
		return nil, errorsmod.Wrapf(types.ErrInvalidAuthority,
			"expected %s, got %s", k.GetAuthority(), req.Authority)
	}
	if !k.HasBridgeAsset(goCtx, req.IbcDenom) {
		return nil, errorsmod.Wrapf(types.ErrAssetNotFound, "%s", req.IbcDenom)
	}
	k.DeleteBridgeAsset(goCtx, req.IbcDenom)

	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeBridgeAssetRemoved,
		sdk.NewAttribute(types.AttrIBCDenom, req.IbcDenom),
	))
	return &types.MsgRemoveBridgeAssetResponse{}, nil
}
