package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/runtime"

	"ltrstchain/x/ltrstdex/types"
)

// GetParams returns the current Params. Returns zero-value Params if
// none have been stored (e.g. before InitGenesis).
func (k Keeper) GetParams(ctx context.Context) (p types.Params) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return p
	}
	k.cdc.MustUnmarshal(bz, &p)
	return p
}

// SetParams writes a new Params value to the module store.
func (k Keeper) SetParams(ctx context.Context, p types.Params) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&p)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, bz)
	return nil
}
