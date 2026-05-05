package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ltrstchain/x/bridge/types"
)

// BridgeAsset fetches a single record by ibc_denom.
func (k Keeper) BridgeAsset(goCtx context.Context, req *types.QueryBridgeAssetRequest) (*types.QueryBridgeAssetResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.IbcDenom == "" {
		return nil, status.Error(codes.InvalidArgument, "ibc_denom required")
	}
	a, ok := k.GetBridgeAsset(goCtx, req.IbcDenom)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "bridge asset %q not found", req.IbcDenom)
	}
	return &types.QueryBridgeAssetResponse{Asset: a}, nil
}

// BridgeAssets lists registered records with optional source_filter
// and verified_only filters. Pagination is standard x/base query.
func (k Keeper) BridgeAssets(goCtx context.Context, req *types.QueryBridgeAssetsRequest) (*types.QueryBridgeAssetsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	store := k.assetStore(goCtx)
	var out []types.BridgeAsset
	page, err := query.FilteredPaginate(store, req.Pagination, func(_, value []byte, accumulate bool) (bool, error) {
		var a types.BridgeAsset
		if err := k.cdc.Unmarshal(value, &a); err != nil {
			return false, errorsmod.Wrap(err, "unmarshal asset")
		}
		// source filter
		if req.SourceFilter != types.BridgeSource_BRIDGE_SOURCE_UNSPECIFIED && a.Source != req.SourceFilter {
			return false, nil
		}
		// verified filter
		if req.VerifiedOnly && !a.Verified {
			return false, nil
		}
		if accumulate {
			out = append(out, a)
		}
		return true, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryBridgeAssetsResponse{Assets: out, Pagination: page}, nil
}
