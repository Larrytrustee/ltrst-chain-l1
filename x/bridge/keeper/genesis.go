package keeper

import (
	"context"
	"fmt"

	"ltrstchain/x/bridge/types"
)

// InitGenesis loads the module state from a validated GenesisState.
// Writes params then every asset.
func (k Keeper) InitGenesis(ctx context.Context, gs types.GenesisState) error {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		return fmt.Errorf("bridge: init genesis params: %w", err)
	}
	for i := range gs.Assets {
		a := gs.Assets[i]
		if err := k.SetBridgeAsset(ctx, a); err != nil {
			return fmt.Errorf("bridge: init genesis asset %s: %w", a.IbcDenom, err)
		}
	}
	return nil
}

// ExportGenesis snapshots module state.
//
// Includes a self-consistency check: if Params.MaxAssets is non-zero
// and the current asset count exceeds it, the exported state would
// fail GenesisState.Validate on the replay side. We'd rather panic
// here — at a known, synchronous chain-upgrade boundary — than ship
// a genesis file that silently bricks subsequent InitGenesis calls.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	gs := &types.GenesisState{
		Params: k.GetParams(ctx),
		Assets: []types.BridgeAsset{},
	}
	k.IterateBridgeAssets(ctx, func(a types.BridgeAsset) bool {
		gs.Assets = append(gs.Assets, a)
		return false
	})
	if gs.Params.MaxAssets > 0 && uint32(len(gs.Assets)) > gs.Params.MaxAssets {
		panic(fmt.Sprintf(
			"bridge: exported %d assets exceeds MaxAssets cap %d",
			len(gs.Assets), gs.Params.MaxAssets,
		))
	}
	return gs
}
