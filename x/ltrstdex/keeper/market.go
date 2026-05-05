package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/runtime"

	"ltrstchain/x/ltrstdex/types"
)

// NextMarketID reads the monotonic marketID counter (or 1 if unset).
func (k Keeper) NextMarketID(ctx context.Context) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.NextMarketIDKey)
	if len(bz) != 8 {
		return 1
	}
	return types.Uint64FromBytes(bz)
}

// setNextMarketID writes the marketID counter.
func (k Keeper) setNextMarketID(ctx context.Context, next uint64) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.NextMarketIDKey, types.Uint64Bytes(next))
}

// CreateMarket validates a new SpotMarket, assigns it the next
// available id, stores it, and updates the ticker index. Fails if
// the ticker or (base,quote) pair is already in use.
func (k Keeper) CreateMarket(ctx context.Context, m types.SpotMarket) (uint64, error) {
	// Ticker uniqueness
	if _, ok := k.GetMarketByTicker(ctx, m.Ticker); ok {
		return 0, errorsmod.Wrapf(types.ErrDuplicateTicker, "ticker %q already in use", m.Ticker)
	}
	// Denom-pair uniqueness
	if existing, ok := k.findMarketByDenoms(ctx, m.BaseDenom, m.QuoteDenom); ok {
		return 0, errorsmod.Wrapf(types.ErrDuplicateDenomPair,
			"denom pair %s/%s already used by market id %d",
			m.BaseDenom, m.QuoteDenom, existing.Id)
	}

	nextID := k.NextMarketID(ctx)
	if nextID == 0 {
		nextID = 1
	}
	m.Id = nextID
	k.setNextMarketID(ctx, nextID+1)

	if err := k.SetMarket(ctx, m); err != nil {
		return 0, err
	}
	return m.Id, nil
}

// SetMarket writes (or overwrites) a SpotMarket record and keeps the
// ticker index in sync.
func (k Keeper) SetMarket(ctx context.Context, m types.SpotMarket) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&m)
	if err != nil {
		return err
	}
	store.Set(types.MarketKey(m.Id), bz)
	store.Set(types.MarketByTickerKey(m.Ticker), types.Uint64Bytes(m.Id))
	return nil
}

// GetMarket returns a SpotMarket by id, and ok=false if not found.
func (k Keeper) GetMarket(ctx context.Context, id uint64) (m types.SpotMarket, found bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.MarketKey(id))
	if bz == nil {
		return m, false
	}
	k.cdc.MustUnmarshal(bz, &m)
	return m, true
}

// GetMarketByTicker resolves a ticker string to a SpotMarket.
func (k Keeper) GetMarketByTicker(ctx context.Context, ticker string) (types.SpotMarket, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.MarketByTickerKey(ticker))
	if len(bz) != 8 {
		return types.SpotMarket{}, false
	}
	return k.GetMarket(ctx, types.Uint64FromBytes(bz))
}

// IterateMarkets calls cb for every SpotMarket in increasing id
// order. cb returning true stops iteration.
func (k Keeper) IterateMarkets(ctx context.Context, cb func(types.SpotMarket) bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	it := store.Iterator(types.MarketKeyPrefix, storeEndKey(types.MarketKeyPrefix))
	defer it.Close()
	for ; it.Valid(); it.Next() {
		// skip the by_ticker index entries, they live under the
		// m/by_ticker/ sub-prefix — shorter keys (m/{u64be}) are
		// exactly 2 + 8 = 10 bytes; skip anything else.
		if len(it.Key()) != len(types.MarketKeyPrefix)+8 {
			continue
		}
		var m types.SpotMarket
		k.cdc.MustUnmarshal(it.Value(), &m)
		if cb(m) {
			return
		}
	}
}

// findMarketByDenoms is a linear scan used only at market creation
// time (not a hot path). Ensures no duplicate base/quote pair.
func (k Keeper) findMarketByDenoms(ctx context.Context, base, quote string) (types.SpotMarket, bool) {
	var match types.SpotMarket
	found := false
	k.IterateMarkets(ctx, func(m types.SpotMarket) bool {
		if m.BaseDenom == base && m.QuoteDenom == quote {
			match = m
			found = true
			return true
		}
		return false
	})
	return match, found
}

// UpdateMarketStatus transitions a market to new_status, respecting
// allowed transitions.
func (k Keeper) UpdateMarketStatus(ctx context.Context, marketID uint64, newStatus types.MarketStatus) error {
	m, ok := k.GetMarket(ctx, marketID)
	if !ok {
		return errorsmod.Wrapf(types.ErrMarketNotFound, "market %d", marketID)
	}
	if !isValidStatusTransition(m.Status, newStatus) {
		return errorsmod.Wrapf(types.ErrInvalidMarketStatus,
			"cannot transition from %s to %s", m.Status, newStatus)
	}
	m.Status = newStatus
	return k.SetMarket(ctx, m)
}

// isValidStatusTransition enforces the small state machine for
// market lifecycle. Any status can always re-enter itself.
func isValidStatusTransition(from, to types.MarketStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case types.MarketStatus_MARKET_STATUS_PRE_OPEN:
		return to == types.MarketStatus_MARKET_STATUS_ACTIVE ||
			to == types.MarketStatus_MARKET_STATUS_POST_ONLY ||
			to == types.MarketStatus_MARKET_STATUS_PAUSED ||
			to == types.MarketStatus_MARKET_STATUS_HALTED
	case types.MarketStatus_MARKET_STATUS_ACTIVE:
		return to == types.MarketStatus_MARKET_STATUS_POST_ONLY ||
			to == types.MarketStatus_MARKET_STATUS_PAUSED ||
			to == types.MarketStatus_MARKET_STATUS_HALTED
	case types.MarketStatus_MARKET_STATUS_POST_ONLY:
		return to == types.MarketStatus_MARKET_STATUS_ACTIVE ||
			to == types.MarketStatus_MARKET_STATUS_PAUSED ||
			to == types.MarketStatus_MARKET_STATUS_HALTED
	case types.MarketStatus_MARKET_STATUS_PAUSED:
		return to == types.MarketStatus_MARKET_STATUS_ACTIVE ||
			to == types.MarketStatus_MARKET_STATUS_POST_ONLY ||
			to == types.MarketStatus_MARKET_STATUS_HALTED
	case types.MarketStatus_MARKET_STATUS_HALTED:
		// Halted is terminal from the gov perspective in Phase 1.
		// A market delisted via HALTED stays halted.
		return false
	default:
		return false
	}
}

// MarketAcceptsOrder returns an error if the market's current status
// does not permit a newly-incoming order of the given order type.
// POST_ONLY permits LIMIT-GTC orders that don't cross; whether it
// crosses is checked by the matcher, not here.
func (k Keeper) MarketAcceptsOrder(m types.SpotMarket, ot types.OrderType) error {
	switch m.Status {
	case types.MarketStatus_MARKET_STATUS_ACTIVE:
		return nil
	case types.MarketStatus_MARKET_STATUS_POST_ONLY:
		if ot != types.OrderType_ORDER_TYPE_LIMIT {
			return errorsmod.Wrap(types.ErrMarketNotAcceptingOrders,
				"market is POST_ONLY; only LIMIT orders accepted")
		}
		return nil
	default:
		return errorsmod.Wrapf(types.ErrMarketNotAcceptingOrders,
			"market status %s does not accept new orders", m.Status)
	}
}

// storeEndKey returns the exclusive upper bound for a prefix-scan
// iterator. Equivalent to prefix.PrefixEndBytes from the SDK but
// scoped locally to avoid another import.
func storeEndKey(prefix []byte) []byte {
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}
