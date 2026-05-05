package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"ltrstchain/x/bridge/types"
)

// RegisterBridgeAsset adds a new entry. Authority-gated when
// Params.RequireGovernanceRegistration is true (the mainnet default);
// otherwise any signer may register.
func (k msgServer) RegisterBridgeAsset(goCtx context.Context, req *types.MsgRegisterBridgeAsset) (*types.MsgRegisterBridgeAssetResponse, error) {
	params := k.GetParams(goCtx)
	if params.RequireGovernanceRegistration && req.Authority != k.GetAuthority() {
		return nil, errorsmod.Wrapf(types.ErrInvalidAuthority,
			"gov-gated: expected %s, got %s", k.GetAuthority(), req.Authority)
	}
	if err := types.ValidateBridgeAsset(req.Asset); err != nil {
		return nil, err
	}
	if k.HasBridgeAsset(goCtx, req.Asset.IbcDenom) {
		return nil, errorsmod.Wrapf(types.ErrAssetAlreadyExists, "%s", req.Asset.IbcDenom)
	}
	// Respect max_assets cap.
	if params.MaxAssets > 0 {
		if k.CountBridgeAssets(goCtx)+1 > params.MaxAssets {
			return nil, errorsmod.Wrapf(types.ErrRegistryFull, "max=%d", params.MaxAssets)
		}
	}

	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	asset := req.Asset
	asset.RegisteredHeight = sdkCtx.BlockHeight()
	asset.UpdatedHeight = sdkCtx.BlockHeight()

	if err := k.SetBridgeAsset(goCtx, asset); err != nil {
		return nil, fmt.Errorf("set asset: %w", err)
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeBridgeAssetRegistered,
		sdk.NewAttribute(types.AttrIBCDenom, asset.IbcDenom),
		sdk.NewAttribute(types.AttrSymbol, asset.DisplaySymbol),
		sdk.NewAttribute(types.AttrSource, asset.Source.String()),
		sdk.NewAttribute(types.AttrSourceChain, asset.SourceChainId),
	))
	return &types.MsgRegisterBridgeAssetResponse{}, nil
}
