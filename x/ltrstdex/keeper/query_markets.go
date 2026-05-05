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

// SpotMarket returns a single SpotMarket by id.
func (k Keeper) SpotMarket(goCtx context.Context, req *types.QuerySpotMarketRequest) (*types.QuerySpotMarketResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.MarketId == 0 {
		return nil, status.Error(codes.InvalidArgument, "market_id is required")
	}
	m, ok := k.GetMarket(goCtx, req.MarketId)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "market %d not found", req.MarketId)
	}
	return &types.QuerySpotMarketResponse{Market: m}, nil
}

// SpotMarketByTicker resolves a ticker to a SpotMarket.
func (k Keeper) SpotMarketByTicker(goCtx context.Context, req *types.QuerySpotMarketByTickerRequest) (*types.QuerySpotMarketResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.Ticker == "" {
		return nil, status.Error(codes.InvalidArgument, "ticker is required")
	}
	m, ok := k.GetMarketByTicker(goCtx, req.Ticker)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "market %q not found", req.Ticker)
	}
	return &types.QuerySpotMarketResponse{Market: m}, nil
}

// SpotMarkets returns all SpotMarket records with standard Cosmos pagination.
// Filters out the by-ticker index entries at iteration time.
func (k Keeper) SpotMarkets(goCtx context.Context, req *types.QuerySpotMarketsRequest) (*types.QuerySpotMarketsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	kv := runtime.KVStoreAdapter(k.storeService.OpenKVStore(goCtx))
	pStore := prefix.NewStore(kv, types.MarketKeyPrefix)

	var markets []types.SpotMarket
	page, err := query.FilteredPaginate(pStore, req.Pagination, func(key, value []byte, accumulate bool) (bool, error) {
		// Skip by-ticker index entries: real market keys are exactly 8 bytes (u64be id).
		if len(key) != 8 {
			return false, nil
		}
		if accumulate {
			var m types.SpotMarket
			if err := k.cdc.Unmarshal(value, &m); err != nil {
				return false, err
			}
			markets = append(markets, m)
		}
		return true, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QuerySpotMarketsResponse{Markets: markets, Pagination: page}, nil
}
