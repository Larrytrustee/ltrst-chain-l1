package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/runtime"

	"ltrstchain/x/ltrstdex/types"
)

// NextFillSeq reads-and-advances the global fill sequence counter.
func (k Keeper) NextFillSeq(ctx context.Context) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.NextFillSeqKey)
	var seq uint64 = 1
	if len(bz) == 8 {
		seq = types.Uint64FromBytes(bz)
	}
	store.Set(types.NextFillSeqKey, types.Uint64Bytes(seq+1))
	return seq
}

// AppendFill writes a fill history record at the canonical key path.
func (k Keeper) AppendFill(ctx context.Context, f types.SpotFill) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&f)
	if err != nil {
		return err
	}
	store.Set(types.FillKey(f.MarketId, uint64(f.BlockHeight), f.FillSeq), bz)
	return nil
}

// IterateFills walks every fill record under a market, in (height,seq)
// ascending order. cb returning true stops iteration.
func (k Keeper) IterateFills(ctx context.Context, marketID uint64, cb func(types.SpotFill) bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	prefix := types.FillMarketPrefix(marketID)
	it := store.Iterator(prefix, storeEndKey(prefix))
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var f types.SpotFill
		k.cdc.MustUnmarshal(it.Value(), &f)
		if cb(f) {
			return
		}
	}
}
