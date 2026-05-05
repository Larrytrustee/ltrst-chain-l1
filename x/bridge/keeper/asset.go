package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"ltrstchain/x/bridge/types"
)

// SetBridgeAsset writes an asset record, overwriting any existing
// entry at the same ibc_denom. Callers that need "create-only"
// semantics should use HasBridgeAsset first.
func (k Keeper) SetBridgeAsset(ctx context.Context, a types.BridgeAsset) error {
	if err := types.ValidateBridgeAsset(a); err != nil {
		return err
	}
	bz, err := k.cdc.Marshal(&a)
	if err != nil {
		return err
	}
	k.kvStore(ctx).Set(types.BridgeAssetKey(a.IbcDenom), bz)
	return nil
}

// GetBridgeAsset fetches an asset by ibc_denom. Second return is
// false when the entry does not exist.
func (k Keeper) GetBridgeAsset(ctx context.Context, ibcDenom string) (types.BridgeAsset, bool) {
	bz := k.kvStore(ctx).Get(types.BridgeAssetKey(ibcDenom))
	if bz == nil {
		return types.BridgeAsset{}, false
	}
	var a types.BridgeAsset
	k.cdc.MustUnmarshal(bz, &a)
	return a, true
}

// HasBridgeAsset returns true when an entry exists.
func (k Keeper) HasBridgeAsset(ctx context.Context, ibcDenom string) bool {
	return k.kvStore(ctx).Has(types.BridgeAssetKey(ibcDenom))
}

// DeleteBridgeAsset deletes the entry. Does not error when missing —
// callers that need the existence check should use HasBridgeAsset.
// Named Delete (not Remove) so it does not collide with the msgServer
// method RemoveBridgeAsset via the embedded Keeper.
func (k Keeper) DeleteBridgeAsset(ctx context.Context, ibcDenom string) {
	k.kvStore(ctx).Delete(types.BridgeAssetKey(ibcDenom))
}

// IterateBridgeAssets walks every registered asset in ibc_denom lex
// order and invokes cb. Return true from cb to stop early.
func (k Keeper) IterateBridgeAssets(ctx context.Context, cb func(types.BridgeAsset) bool) {
	kv := k.kvStore(ctx)
	ps := prefix.NewStore(kv, types.BridgeAssetKeyPrefix)
	it := ps.Iterator(nil, nil)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		var a types.BridgeAsset
		k.cdc.MustUnmarshal(it.Value(), &a)
		if cb(a) {
			return
		}
	}
}

// CountBridgeAssets returns the number of registered entries.
func (k Keeper) CountBridgeAssets(ctx context.Context) uint32 {
	var n uint32
	k.IterateBridgeAssets(ctx, func(types.BridgeAsset) bool {
		n++
		return false
	})
	return n
}

// assetStore returns the asset-prefixed store. Exposed for callers
// that need direct iterator access (e.g. paginated queries).
func (k Keeper) assetStore(ctx context.Context) storetypes.KVStore {
	kv := k.kvStore(ctx)
	return prefix.NewStore(kv, types.BridgeAssetKeyPrefix)
}
