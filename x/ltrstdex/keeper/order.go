package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"ltrstchain/x/ltrstdex/types"
)

// NextOrderSeq returns the next monotonic sequence number for a
// market and advances the counter.
func (k Keeper) NextOrderSeq(ctx context.Context, marketID uint64) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := types.NextOrderSeqKey(marketID)
	bz := store.Get(key)
	var seq uint64 = 1
	if len(bz) == 8 {
		seq = types.Uint64FromBytes(bz)
	}
	store.Set(key, types.Uint64Bytes(seq+1))
	return seq
}

// PeekNextOrderSeq reads without advancing. Used by ExportGenesis.
func (k Keeper) PeekNextOrderSeq(ctx context.Context, marketID uint64) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.NextOrderSeqKey(marketID))
	if len(bz) != 8 {
		return 1
	}
	return types.Uint64FromBytes(bz)
}

// setNextOrderSeqRaw writes the next order seq for a market. Only
// used by InitGenesis; normal code uses NextOrderSeq.
func (k Keeper) setNextOrderSeqRaw(ctx context.Context, marketID, seq uint64) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.NextOrderSeqKey(marketID), types.Uint64Bytes(seq))
}

// DeriveOrderID produces a deterministic order_id string from the
// (owner, marketID, seq) tuple. Guaranteed unique per chain.
func DeriveOrderID(owner sdk.AccAddress, marketID, seq uint64) string {
	h := sha256.New()
	h.Write(owner.Bytes())
	h.Write(types.Uint64Bytes(marketID))
	h.Write(types.Uint64Bytes(seq))
	sum := h.Sum(nil)
	// 20 hex chars = 80 bits is more than enough for chain-wide uniqueness.
	return hex.EncodeToString(sum[:10])
}

// SetOrder writes (or overwrites) the authoritative SpotOrder record.
// It does NOT touch the orderbook index — the caller is responsible
// for calling InsertOrderbookIndex / DeleteOrderbookIndex when state
// transitions require it.
func (k Keeper) SetOrder(ctx context.Context, o types.SpotOrder) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&o)
	if err != nil {
		return err
	}
	store.Set(types.OrderKey(o.MarketId, o.OrderId), bz)
	// User index
	store.Set(types.UserOrderKey(o.Owner, o.OrderId), types.Uint64Bytes(o.MarketId))
	return nil
}

// GetOrder reads a SpotOrder by (marketID, orderID).
func (k Keeper) GetOrder(ctx context.Context, marketID uint64, orderID string) (types.SpotOrder, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.OrderKey(marketID, orderID))
	if bz == nil {
		return types.SpotOrder{}, false
	}
	var o types.SpotOrder
	k.cdc.MustUnmarshal(bz, &o)
	return o, true
}

// DeleteOrder removes both the authoritative record and the user
// index entry for an order. Caller is responsible for also removing
// the orderbook index entry if one exists.
func (k Keeper) DeleteOrder(ctx context.Context, o types.SpotOrder) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Delete(types.OrderKey(o.MarketId, o.OrderId))
	store.Delete(types.UserOrderKey(o.Owner, o.OrderId))
}

// IterateMarketOrders calls cb for every order stored under a market,
// in orderID-sorted order (not price-time). Mainly for ExportGenesis
// and the SpotOrdersByMarket query when side filtering is off.
func (k Keeper) IterateMarketOrders(ctx context.Context, marketID uint64, cb func(types.SpotOrder) bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	prefix := types.OrderKeyForMarketPrefix(marketID)
	it := store.Iterator(prefix, storeEndKey(prefix))
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var o types.SpotOrder
		k.cdc.MustUnmarshal(it.Value(), &o)
		if cb(o) {
			return
		}
	}
}

// IterateOwnerOrders calls cb for every live order owned by addr.
func (k Keeper) IterateOwnerOrders(ctx context.Context, owner sdk.AccAddress, cb func(types.SpotOrder) bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	prefix := types.UserOrderOwnerPrefix(owner.Bytes())
	it := store.Iterator(prefix, storeEndKey(prefix))
	defer it.Close()

	for ; it.Valid(); it.Next() {
		if len(it.Value()) != 8 {
			continue
		}
		marketID := types.Uint64FromBytes(it.Value())
		// Extract orderID from key: prefix + orderID (after the final '/').
		key := it.Key()
		if len(key) <= len(prefix) {
			continue
		}
		orderID := string(key[len(prefix):])
		o, ok := k.GetOrder(ctx, marketID, orderID)
		if !ok {
			continue
		}
		if cb(o) {
			return
		}
	}
}

// CountOwnerOrders returns the live-order count owned by addr.
// Used to enforce Params.MaxOpenOrdersPerAccount.
func (k Keeper) CountOwnerOrders(ctx context.Context, owner sdk.AccAddress) uint32 {
	var count uint32
	k.IterateOwnerOrders(ctx, owner, func(_ types.SpotOrder) bool {
		count++
		return false
	})
	return count
}

// AssertOwnerUnderMaxOpen returns ErrTooManyOpenOrders if adding one
// more order would exceed the configured max.
func (k Keeper) AssertOwnerUnderMaxOpen(ctx context.Context, owner sdk.AccAddress) error {
	p := k.GetParams(ctx)
	if p.MaxOpenOrdersPerAccount == 0 {
		return nil // not configured — no cap
	}
	if k.CountOwnerOrders(ctx, owner) >= p.MaxOpenOrdersPerAccount {
		return errorsmod.Wrapf(types.ErrTooManyOpenOrders,
			"account %s already has %d open orders (max %d)",
			owner.String(), k.CountOwnerOrders(ctx, owner), p.MaxOpenOrdersPerAccount)
	}
	return nil
}
