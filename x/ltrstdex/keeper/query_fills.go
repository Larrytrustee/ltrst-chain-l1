package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ltrstchain/x/ltrstdex/types"
)

// SpotFills returns paginated fill history for a market, optionally
// filtered by a block-height window. Fills are stored (and iterated)
// in (height,seq) ascending order.
func (k Keeper) SpotFills(goCtx context.Context, req *types.QuerySpotFillsRequest) (*types.QuerySpotFillsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.MarketId == 0 {
		return nil, status.Error(codes.InvalidArgument, "market_id is required")
	}
	if req.MaxBlockHeight != 0 && req.MinBlockHeight > req.MaxBlockHeight {
		return nil, status.Error(codes.InvalidArgument,
			"min_block_height must be ≤ max_block_height")
	}

	kv := runtime.KVStoreAdapter(k.storeService.OpenKVStore(goCtx))
	pStore := prefix.NewStore(kv, types.FillMarketPrefix(req.MarketId))

	var fills []types.SpotFill
	page, err := query.FilteredPaginate(pStore, req.Pagination, func(key, value []byte, accumulate bool) (bool, error) {
		var f types.SpotFill
		if err := k.cdc.Unmarshal(value, &f); err != nil {
			return false, err
		}
		if req.MinBlockHeight != 0 && f.BlockHeight < req.MinBlockHeight {
			return false, nil
		}
		if req.MaxBlockHeight != 0 && f.BlockHeight > req.MaxBlockHeight {
			return false, nil
		}
		if accumulate {
			fills = append(fills, f)
		}
		return true, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QuerySpotFillsResponse{Fills: fills, Pagination: page}, nil
}
