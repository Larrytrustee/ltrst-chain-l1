package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
)

// DefaultGenesis returns the default ltrstdex genesis state for a
// brand-new chain: params at Phase-1 defaults, no markets, no orders.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:                DefaultParams(),
		Markets:               []SpotMarket{},
		Orders:                []SpotOrder{},
		NextMarketId:          1,
		NextOrderSeqByMarket:  map[uint64]uint64{},
	}
}

// Validate performs full static validation of the genesis state.
// Must reject any state that would let InitGenesis leave the store
// in a mutually-inconsistent state.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return errorsmod.Wrap(ErrInvalidGenesis, err.Error())
	}

	marketsByID := make(map[uint64]struct{}, len(gs.Markets))
	tickersSeen := make(map[string]struct{}, len(gs.Markets))
	denomPairsSeen := make(map[string]struct{}, len(gs.Markets))

	for i := range gs.Markets {
		m := gs.Markets[i]

		if m.Id == 0 {
			return errorsmod.Wrapf(ErrInvalidGenesis, "market id must be > 0 (market[%d])", i)
		}
		if m.Id >= gs.NextMarketId {
			return errorsmod.Wrapf(ErrInvalidGenesis,
				"market id %d >= next_market_id %d (must be strictly less)", m.Id, gs.NextMarketId)
		}
		if _, dup := marketsByID[m.Id]; dup {
			return errorsmod.Wrapf(ErrInvalidGenesis, "duplicate market id %d", m.Id)
		}
		marketsByID[m.Id] = struct{}{}

		if m.Ticker == "" {
			return errorsmod.Wrapf(ErrInvalidGenesis, "market %d has empty ticker", m.Id)
		}
		if _, dup := tickersSeen[m.Ticker]; dup {
			return errorsmod.Wrapf(ErrInvalidGenesis, "duplicate ticker %q", m.Ticker)
		}
		tickersSeen[m.Ticker] = struct{}{}

		if m.BaseDenom == "" || m.QuoteDenom == "" {
			return errorsmod.Wrapf(ErrInvalidGenesis, "market %d missing base/quote denom", m.Id)
		}
		if m.BaseDenom == m.QuoteDenom {
			return errorsmod.Wrapf(ErrInvalidGenesis, "market %d: base and quote denom must differ", m.Id)
		}
		denomPair := m.BaseDenom + "/" + m.QuoteDenom
		if _, dup := denomPairsSeen[denomPair]; dup {
			return errorsmod.Wrapf(ErrInvalidGenesis,
				"duplicate base/quote pair %q", denomPair)
		}
		denomPairsSeen[denomPair] = struct{}{}

		if m.MinPriceTick.IsNil() || !m.MinPriceTick.IsPositive() {
			return errorsmod.Wrapf(ErrInvalidGenesis,
				"market %d: min_price_tick must be positive", m.Id)
		}
		if m.MinQuantity.IsNil() || !m.MinQuantity.IsPositive() {
			return errorsmod.Wrapf(ErrInvalidGenesis,
				"market %d: min_quantity must be positive", m.Id)
		}
		if m.MakerFeeBps > 10_000 {
			return errorsmod.Wrapf(ErrInvalidGenesis,
				"market %d: maker_fee_bps must be <= 10000 (got %d)", m.Id, m.MakerFeeBps)
		}
		if m.TakerFeeBps > 10_000 {
			return errorsmod.Wrapf(ErrInvalidGenesis,
				"market %d: taker_fee_bps must be <= 10000 (got %d)", m.Id, m.TakerFeeBps)
		}
		if m.Status == MarketStatus_MARKET_STATUS_UNSPECIFIED {
			return errorsmod.Wrapf(ErrInvalidGenesis,
				"market %d: status must be specified", m.Id)
		}
	}

	ordersByID := make(map[string]struct{}, len(gs.Orders))
	for i := range gs.Orders {
		o := gs.Orders[i]
		if o.OrderId == "" {
			return errorsmod.Wrapf(ErrInvalidGenesis, "order[%d] has empty order_id", i)
		}
		if _, dup := ordersByID[o.OrderId]; dup {
			return errorsmod.Wrapf(ErrInvalidGenesis, "duplicate order_id %s", o.OrderId)
		}
		ordersByID[o.OrderId] = struct{}{}

		if _, known := marketsByID[o.MarketId]; !known {
			return errorsmod.Wrapf(ErrInvalidGenesis,
				"order %s references unknown market %d", o.OrderId, o.MarketId)
		}

		if o.Side != Side_SIDE_BUY && o.Side != Side_SIDE_SELL {
			return errorsmod.Wrapf(ErrInvalidGenesis,
				"order %s has invalid side %s", o.OrderId, o.Side)
		}
		if o.Quantity.IsNil() || !o.Quantity.IsPositive() {
			return errorsmod.Wrapf(ErrInvalidGenesis,
				"order %s quantity must be positive", o.OrderId)
		}
		if o.FilledQuantity.IsNil() {
			return errorsmod.Wrapf(ErrInvalidGenesis,
				"order %s filled_quantity must be non-nil (use zero if unfilled)", o.OrderId)
		}
		if o.FilledQuantity.GT(o.Quantity) {
			return errorsmod.Wrapf(ErrInvalidGenesis,
				"order %s filled_quantity > quantity", o.OrderId)
		}
	}

	for mid, nextSeq := range gs.NextOrderSeqByMarket {
		if _, ok := marketsByID[mid]; !ok {
			return errorsmod.Wrapf(ErrInvalidGenesis,
				"next_order_seq_by_market references unknown market %d", mid)
		}
		_ = nextSeq // zero is allowed (no orders yet on this market)
	}

	if gs.NextMarketId == 0 {
		return errorsmod.Wrap(ErrInvalidGenesis, "next_market_id must be > 0")
	}

	if len(gs.Orders) > 0 && len(gs.Markets) == 0 {
		return fmt.Errorf("ltrstdex genesis: orders present but no markets defined")
	}

	return nil
}
