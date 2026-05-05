package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"ltrstchain/x/bridge/types"
)

// UpdateBridgeAsset mutates an existing entry. Authority-gated.
// Empty-string fields are treated as "no change" — numeric/bool
// fields are gated by a companion `set_*` bool to avoid ambiguity
// with zero values.
func (k msgServer) UpdateBridgeAsset(goCtx context.Context, req *types.MsgUpdateBridgeAsset) (*types.MsgUpdateBridgeAssetResponse, error) {
	if req.Authority != k.GetAuthority() {
		return nil, errorsmod.Wrapf(types.ErrInvalidAuthority,
			"expected %s, got %s", k.GetAuthority(), req.Authority)
	}
	existing, ok := k.GetBridgeAsset(goCtx, req.IbcDenom)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrAssetNotFound, "%s", req.IbcDenom)
	}

	if req.DisplaySymbol != "" {
		existing.DisplaySymbol = req.DisplaySymbol
	}
	if req.DisplayName != "" {
		existing.DisplayName = req.DisplayName
	}
	if req.SetDecimals {
		existing.Decimals = req.Decimals
	}
	if req.SetSource {
		existing.Source = req.Source
	}
	if req.SourceChainId != "" {
		existing.SourceChainId = req.SourceChainId
	}
	if req.SourceDenom != "" {
		existing.SourceDenom = req.SourceDenom
	}
	if req.ChannelId != "" {
		existing.ChannelId = req.ChannelId
	}
	if req.SetVerified {
		existing.Verified = req.Verified
	}

	if err := types.ValidateBridgeAsset(existing); err != nil {
		return nil, err
	}
	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	existing.UpdatedHeight = sdkCtx.BlockHeight()
	if err := k.SetBridgeAsset(goCtx, existing); err != nil {
		return nil, err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeBridgeAssetUpdated,
		sdk.NewAttribute(types.AttrIBCDenom, existing.IbcDenom),
		sdk.NewAttribute(types.AttrVerified, boolStr(existing.Verified)),
	))
	return &types.MsgUpdateBridgeAssetResponse{}, nil
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
