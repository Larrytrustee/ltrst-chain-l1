package keeper

import (
	"context"
	"fmt"

	"ltrstchain/x/ltrstdex/types"
)

// InitGenesis loads the x/ltrstdex state from a GenesisState. The
// caller (module.InitGenesis) must have already validated the state.
//
//  1. Write Params.
//  2. Write NextMarketID counter.
//  3. Replay every market record (main + by-ticker index).
//  4. Replay the per-market next-order-seq counters.
//  5. Replay every order — writes the authoritative record, the
//     user-owner index, and (if the order still has open quantity)
//     re-inserts it into the orderbook index so matching resumes in
//     the same price-time priority as before the export.
func (k Keeper) InitGenesis(ctx context.Context, gs types.GenesisState) error {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		return fmt.Errorf("ltrstdex: init genesis params: %w", err)
	}

	// Next-market-id — default to 1 if unset (DefaultGenesis already
	// sets 1, but belt-and-suspenders).
	next := gs.NextMarketId
	if next == 0 {
		next = 1
	}
	k.setNextMarketID(ctx, next)

	for i := range gs.Markets {
		if err := k.SetMarket(ctx, gs.Markets[i]); err != nil {
			return fmt.Errorf("ltrstdex: init genesis market %d: %w", gs.Markets[i].Id, err)
		}
	}

	for marketID, seq := range gs.NextOrderSeqByMarket {
		k.setNextOrderSeqRaw(ctx, marketID, seq)
	}

	for i := range gs.Orders {
		o := gs.Orders[i]
		if err := k.SetOrder(ctx, o); err != nil {
			return fmt.Errorf("ltrstdex: init genesis order %s: %w", o.OrderId, err)
		}
		// Re-insert into orderbook index only if still open.
		if o.FilledQuantity.LT(o.Quantity) {
			if err := k.InsertOrderbookIndex(ctx, o); err != nil {
				return fmt.Errorf("ltrstdex: init genesis order %s index: %w", o.OrderId, err)
			}
		}
	}

	return nil
}

// ExportGenesis snapshots the module's entire state into a GenesisState.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	gs := &types.GenesisState{
		Params:               k.GetParams(ctx),
		NextMarketId:         k.NextMarketID(ctx),
		NextOrderSeqByMarket: map[uint64]uint64{},
	}

	k.IterateMarkets(ctx, func(m types.SpotMarket) bool {
		gs.Markets = append(gs.Markets, m)
		gs.NextOrderSeqByMarket[m.Id] = k.PeekNextOrderSeq(ctx, m.Id)
		k.IterateMarketOrders(ctx, m.Id, func(o types.SpotOrder) bool {
			gs.Orders = append(gs.Orders, o)
			return false
		})
		return false
	})

	// Keep slices non-nil for deterministic JSON output.
	if gs.Markets == nil {
		gs.Markets = []types.SpotMarket{}
	}
	if gs.Orders == nil {
		gs.Orders = []types.SpotOrder{}
	}

	return gs
}
