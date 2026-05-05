package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ltrstchain/x/ltrstdex/types"
)

// DefaultOrderbookDepth is the default page size for SpotOrderbook
// queries when no depth is supplied in the request. Matches the Phase
// 1 design note — UIs rarely show deeper than 50 levels per side.
const DefaultOrderbookDepth = uint32(50)

// SpotOrderbook returns aggregated best-first price levels for both
// sides of a market's orderbook. Levels sum remaining (non-filled)
// quantity per price. Depth bounds how many levels per side.
func (k Keeper) SpotOrderbook(goCtx context.Context, req *types.QuerySpotOrderbookRequest) (*types.QuerySpotOrderbookResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.MarketId == 0 {
		return nil, status.Error(codes.InvalidArgument, "market_id is required")
	}
	if _, ok := k.GetMarket(goCtx, req.MarketId); !ok {
		return nil, status.Errorf(codes.NotFound, "market %d not found", req.MarketId)
	}

	depth := req.Depth
	if depth == 0 {
		depth = DefaultOrderbookDepth
	}

	bids := k.AggregateLevels(goCtx, req.MarketId, true, depth)
	asks := k.AggregateLevels(goCtx, req.MarketId, false, depth)

	return &types.QuerySpotOrderbookResponse{Bids: bids, Asks: asks}, nil
}
