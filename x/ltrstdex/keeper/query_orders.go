package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ltrstchain/x/ltrstdex/types"
)

// SpotOrder returns a single SpotOrder by (marketID, orderID).
func (k Keeper) SpotOrder(goCtx context.Context, req *types.QuerySpotOrderRequest) (*types.QuerySpotOrderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.MarketId == 0 || req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "market_id and order_id are required")
	}
	o, ok := k.GetOrder(goCtx, req.MarketId, req.OrderId)
	if !ok {
		return nil, status.Errorf(codes.NotFound,
			"order %s on market %d not found", req.OrderId, req.MarketId)
	}
	return &types.QuerySpotOrderResponse{Order: o}, nil
}

// SpotOrdersByOwner lists every live order owned by a bech32 address
// across all markets, paginated. Each returned record is the full
// SpotOrder (not the index pointer).
func (k Keeper) SpotOrdersByOwner(goCtx context.Context, req *types.QuerySpotOrdersByOwnerRequest) (*types.QuerySpotOrdersByOwnerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	owner, err := sdk.AccAddressFromBech32(req.Owner)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid owner: "+err.Error())
	}

	kv := runtime.KVStoreAdapter(k.storeService.OpenKVStore(goCtx))
	ownerPrefix := types.UserOrderOwnerPrefix(owner.Bytes())
	pStore := prefix.NewStore(kv, ownerPrefix)

	var orders []types.SpotOrder
	page, err := query.FilteredPaginate(pStore, req.Pagination, func(key, value []byte, accumulate bool) (bool, error) {
		if len(value) != 8 {
			return false, nil
		}
		marketID := types.Uint64FromBytes(value)
		orderID := string(key)
		o, ok := k.GetOrder(goCtx, marketID, orderID)
		if !ok {
			return false, nil
		}
		if accumulate {
			orders = append(orders, o)
		}
		return true, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QuerySpotOrdersByOwnerResponse{Orders: orders, Pagination: page}, nil
}

// SpotOrdersByMarket lists every live order on a market, optionally
// filtered to one side. Iteration is by orderID (not price-time
// priority) — for the orderbook snapshot use SpotOrderbook instead.
func (k Keeper) SpotOrdersByMarket(goCtx context.Context, req *types.QuerySpotOrdersByMarketRequest) (*types.QuerySpotOrdersByMarketResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.MarketId == 0 {
		return nil, status.Error(codes.InvalidArgument, "market_id is required")
	}

	kv := runtime.KVStoreAdapter(k.storeService.OpenKVStore(goCtx))
	pStore := prefix.NewStore(kv, types.OrderKeyForMarketPrefix(req.MarketId))

	var orders []types.SpotOrder
	page, err := query.FilteredPaginate(pStore, req.Pagination, func(key, value []byte, accumulate bool) (bool, error) {
		var o types.SpotOrder
		if err := k.cdc.Unmarshal(value, &o); err != nil {
			return false, err
		}
		if req.SideFilter != types.Side_SIDE_UNSPECIFIED && o.Side != req.SideFilter {
			return false, nil
		}
		if accumulate {
			orders = append(orders, o)
		}
		return true, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QuerySpotOrdersByMarketResponse{Orders: orders, Pagination: page}, nil
}
