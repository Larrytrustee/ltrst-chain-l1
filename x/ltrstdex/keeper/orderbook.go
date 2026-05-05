package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "cosmossdk.io/store/types"

	"ltrstchain/x/ltrstdex/types"
)

// InsertOrderbookIndex writes the (side, price, seq, orderID) entry
// that makes this order walkable in price-time priority.
func (k Keeper) InsertOrderbookIndex(ctx context.Context, o types.SpotOrder) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	switch o.Side {
	case types.Side_SIDE_BUY:
		priceEnc, err := types.EncodeBidPrice(o.Price)
		if err != nil {
			return err
		}
		key := types.OrderbookBidKey(o.MarketId, priceEnc, o.Seq, o.OrderId)
		store.Set(key, []byte(o.OrderId))
	case types.Side_SIDE_SELL:
		priceEnc, err := types.EncodePrice(o.Price)
		if err != nil {
			return err
		}
		key := types.OrderbookAskKey(o.MarketId, priceEnc, o.Seq, o.OrderId)
		store.Set(key, []byte(o.OrderId))
	default:
		return types.ErrInvalidSide
	}
	return nil
}

// DeleteOrderbookIndex removes the index entry for an order. Safe to
// call even if no entry exists (no-op).
func (k Keeper) DeleteOrderbookIndex(ctx context.Context, o types.SpotOrder) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	switch o.Side {
	case types.Side_SIDE_BUY:
		priceEnc, err := types.EncodeBidPrice(o.Price)
		if err != nil {
			return err
		}
		store.Delete(types.OrderbookBidKey(o.MarketId, priceEnc, o.Seq, o.OrderId))
	case types.Side_SIDE_SELL:
		priceEnc, err := types.EncodePrice(o.Price)
		if err != nil {
			return err
		}
		store.Delete(types.OrderbookAskKey(o.MarketId, priceEnc, o.Seq, o.OrderId))
	default:
		return types.ErrInvalidSide
	}
	return nil
}

// IterateBidSide walks buy orders for a market from best (highest
// price) to worst (lowest). Within a price level, older seq first.
// cb returning true stops iteration.
func (k Keeper) IterateBidSide(ctx context.Context, marketID uint64, cb func(types.SpotOrder) bool) {
	k.iterateBookSide(ctx, marketID, true, cb)
}

// IterateAskSide walks sell orders for a market from best (lowest
// price) to worst (highest). Within a price level, older seq first.
func (k Keeper) IterateAskSide(ctx context.Context, marketID uint64, cb func(types.SpotOrder) bool) {
	k.iterateBookSide(ctx, marketID, false, cb)
}

// iterateBookSide performs the shared walk over either side. For
// bids, price encoding is already inverted so forward iteration
// yields descending prices.
func (k Keeper) iterateBookSide(ctx context.Context, marketID uint64, bid bool, cb func(types.SpotOrder) bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	var prefix []byte
	if bid {
		prefix = types.OrderbookBidMarketPrefix(marketID)
	} else {
		prefix = types.OrderbookAskMarketPrefix(marketID)
	}
	it := store.Iterator(prefix, storeEndKey(prefix))
	defer it.Close()
	for ; it.Valid(); it.Next() {
		orderID := string(it.Value())
		if orderID == "" {
			continue
		}
		o, ok := k.GetOrder(ctx, marketID, orderID)
		if !ok {
			continue
		}
		if cb(o) {
			return
		}
	}
}

// BestBid returns the highest-priced resting buy order, if any.
func (k Keeper) BestBid(ctx context.Context, marketID uint64) (types.SpotOrder, bool) {
	var out types.SpotOrder
	found := false
	k.IterateBidSide(ctx, marketID, func(o types.SpotOrder) bool {
		out = o
		found = true
		return true
	})
	return out, found
}

// BestAsk returns the lowest-priced resting sell order, if any.
func (k Keeper) BestAsk(ctx context.Context, marketID uint64) (types.SpotOrder, bool) {
	var out types.SpotOrder
	found := false
	k.IterateAskSide(ctx, marketID, func(o types.SpotOrder) bool {
		out = o
		found = true
		return true
	})
	return out, found
}

// AggregateLevels walks one side and aggregates remaining quantity
// by price. Used by the SpotOrderbook query. Stops after `maxLevels`
// distinct price levels (0 = unlimited).
func (k Keeper) AggregateLevels(ctx context.Context, marketID uint64, bid bool, maxLevels uint32) []types.OrderbookLevel {
	var out []types.OrderbookLevel
	var current *types.OrderbookLevel

	appendCurrent := func() {
		if current != nil {
			out = append(out, *current)
			current = nil
		}
	}

	walk := func(o types.SpotOrder) bool {
		remaining := o.Quantity.Sub(o.FilledQuantity)
		if !remaining.IsPositive() {
			return false
		}
		if current == nil || !current.Price.Equal(o.Price) {
			appendCurrent()
			if maxLevels != 0 && uint32(len(out)) >= maxLevels {
				return true
			}
			current = &types.OrderbookLevel{
				Price:         o.Price,
				TotalQuantity: math.LegacyZeroDec(),
				OrderCount:    0,
			}
		}
		current.TotalQuantity = current.TotalQuantity.Add(remaining)
		current.OrderCount++
		return false
	}
	if bid {
		k.IterateBidSide(ctx, marketID, walk)
	} else {
		k.IterateAskSide(ctx, marketID, walk)
	}
	appendCurrent()
	if maxLevels != 0 && uint32(len(out)) > maxLevels {
		out = out[:maxLevels]
	}
	return out
}

// Ensure the storetypes import isn't pruned — used indirectly by the
// runtime.KVStoreAdapter iterator contract for prefix scans.
var _ = storetypes.StoreTypeIAVL
