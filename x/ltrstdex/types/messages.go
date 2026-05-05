package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Compile-time assertions that every msg satisfies sdk.Msg.
var (
	_ sdk.Msg = &MsgUpdateParams{}
	_ sdk.Msg = &MsgCreateSpotMarket{}
	_ sdk.Msg = &MsgUpdateSpotMarketStatus{}
	_ sdk.Msg = &MsgPlaceSpotOrder{}
	_ sdk.Msg = &MsgCancelSpotOrder{}
	_ sdk.Msg = &MsgCancelAllSpotOrders{}
)

// MaxClientOrderIDLen is the upper bound for the optional tag on a
// place-order message. Kept short to avoid state bloat.
const MaxClientOrderIDLen = 64

// MaxTickerLen bounds the human-readable ticker of a market.
const MaxTickerLen = 32

// -----------------------------------------------------------------------------
// MsgUpdateParams
// -----------------------------------------------------------------------------

func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(ErrInvalidAuthority, "invalid authority: "+err.Error())
	}
	if err := m.Params.Validate(); err != nil {
		return err
	}
	return nil
}

// -----------------------------------------------------------------------------
// MsgCreateSpotMarket
// -----------------------------------------------------------------------------

func (m *MsgCreateSpotMarket) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(ErrInvalidAuthority, "invalid authority: "+err.Error())
	}
	if m.Ticker == "" {
		return errorsmod.Wrap(ErrInvalidMarketParams, "ticker is required")
	}
	if len(m.Ticker) > MaxTickerLen {
		return errorsmod.Wrapf(ErrInvalidMarketParams,
			"ticker must be ≤ %d chars (got %d)", MaxTickerLen, len(m.Ticker))
	}
	if err := sdk.ValidateDenom(m.BaseDenom); err != nil {
		return errorsmod.Wrap(ErrInvalidMarketParams, "invalid base_denom: "+err.Error())
	}
	if err := sdk.ValidateDenom(m.QuoteDenom); err != nil {
		return errorsmod.Wrap(ErrInvalidMarketParams, "invalid quote_denom: "+err.Error())
	}
	if m.BaseDenom == m.QuoteDenom {
		return errorsmod.Wrap(ErrInvalidMarketParams, "base and quote denom must differ")
	}
	if m.MinPriceTick.IsNil() || !m.MinPriceTick.IsPositive() {
		return errorsmod.Wrap(ErrInvalidMarketParams, "min_price_tick must be positive")
	}
	if m.MinQuantity.IsNil() || !m.MinQuantity.IsPositive() {
		return errorsmod.Wrap(ErrInvalidMarketParams, "min_quantity must be positive")
	}
	if m.MakerFeeBps > 10_000 {
		return errorsmod.Wrapf(ErrInvalidMarketParams,
			"maker_fee_bps must be ≤ 10000 (got %d)", m.MakerFeeBps)
	}
	if m.TakerFeeBps > 10_000 {
		return errorsmod.Wrapf(ErrInvalidMarketParams,
			"taker_fee_bps must be ≤ 10000 (got %d)", m.TakerFeeBps)
	}
	switch m.InitialStatus {
	case MarketStatus_MARKET_STATUS_PRE_OPEN,
		MarketStatus_MARKET_STATUS_ACTIVE,
		MarketStatus_MARKET_STATUS_POST_ONLY:
		// ok
	default:
		return errorsmod.Wrapf(ErrInvalidMarketStatus,
			"initial_status must be PRE_OPEN, ACTIVE, or POST_ONLY (got %s)", m.InitialStatus)
	}
	return nil
}

// -----------------------------------------------------------------------------
// MsgUpdateSpotMarketStatus
// -----------------------------------------------------------------------------

func (m *MsgUpdateSpotMarketStatus) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(ErrInvalidAuthority, "invalid authority: "+err.Error())
	}
	if m.MarketId == 0 {
		return errorsmod.Wrap(ErrInvalidMarketParams, "market_id is required")
	}
	if m.NewStatus == MarketStatus_MARKET_STATUS_UNSPECIFIED {
		return errorsmod.Wrap(ErrInvalidMarketStatus, "new_status must be specified")
	}
	return nil
}

// -----------------------------------------------------------------------------
// MsgPlaceSpotOrder
// -----------------------------------------------------------------------------

func (m *MsgPlaceSpotOrder) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return errorsmod.Wrap(ErrInvalidAuthority, "invalid owner: "+err.Error())
	}
	if m.MarketId == 0 {
		return errorsmod.Wrap(ErrInvalidOrderParams, "market_id is required")
	}
	switch m.Side {
	case Side_SIDE_BUY, Side_SIDE_SELL:
	default:
		return errorsmod.Wrapf(ErrInvalidSide, "invalid side %s", m.Side)
	}
	switch m.Type {
	case OrderType_ORDER_TYPE_LIMIT:
		if m.Price.IsNil() || !m.Price.IsPositive() {
			return ErrLimitOrderNeedsPrice
		}
	case OrderType_ORDER_TYPE_MARKET:
		if !m.Price.IsNil() && !m.Price.IsZero() {
			return ErrMarketOrderNeedsNoPrice
		}
	default:
		return errorsmod.Wrapf(ErrInvalidOrderType, "invalid type %s", m.Type)
	}
	switch m.Tif {
	case TimeInForce_TIF_GTC, TimeInForce_TIF_IOC, TimeInForce_TIF_FOK:
	default:
		return errorsmod.Wrapf(ErrInvalidTIF, "invalid tif %s", m.Tif)
	}
	if m.Quantity.IsNil() || !m.Quantity.IsPositive() {
		return errorsmod.Wrap(ErrInvalidOrderParams, "quantity must be positive")
	}
	if m.ExpiresAtBlock < 0 {
		return errorsmod.Wrap(ErrInvalidOrderParams, "expires_at_block must be ≥ 0 (0 = never)")
	}
	if len(m.ClientOrderId) > MaxClientOrderIDLen {
		return errorsmod.Wrapf(ErrInvalidOrderParams,
			"client_order_id must be ≤ %d chars (got %d)", MaxClientOrderIDLen, len(m.ClientOrderId))
	}
	// Enforce IOC semantics on MARKET orders even if caller sends a different TIF.
	if m.Type == OrderType_ORDER_TYPE_MARKET && m.Tif == TimeInForce_TIF_FOK {
		return fmt.Errorf("market orders cannot use FOK — MARKET is IOC-equivalent")
	}
	return nil
}

// -----------------------------------------------------------------------------
// MsgCancelSpotOrder
// -----------------------------------------------------------------------------

func (m *MsgCancelSpotOrder) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(ErrInvalidAuthority, "invalid authority: "+err.Error())
	}
	if m.MarketId == 0 {
		return errorsmod.Wrap(ErrInvalidOrderParams, "market_id is required")
	}
	if m.OrderId == "" {
		return errorsmod.Wrap(ErrInvalidOrderParams, "order_id is required")
	}
	return nil
}

// -----------------------------------------------------------------------------
// MsgCancelAllSpotOrders
// -----------------------------------------------------------------------------

func (m *MsgCancelAllSpotOrders) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return errorsmod.Wrap(ErrInvalidAuthority, "invalid owner: "+err.Error())
	}
	// market_id == 0 is legal; it means "cancel across all markets".
	return nil
}
