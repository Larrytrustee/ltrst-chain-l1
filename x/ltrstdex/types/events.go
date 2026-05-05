package types

// Event types and attribute keys emitted by x/ltrstdex.
const (
	EventTypeSpotMarketCreated = "spot_market_created"
	EventTypeSpotMarketStatus  = "spot_market_status"
	EventTypeSpotOrderPlaced   = "spot_order_placed"
	EventTypeSpotOrderCancelled = "spot_order_cancelled"
	EventTypeSpotFill          = "spot_fill"

	AttrMarketID   = "market_id"
	AttrOrderID    = "order_id"
	AttrOwner      = "owner"
	AttrSide       = "side"
	AttrPrice      = "price"
	AttrQuantity   = "quantity"
	AttrFilledQty  = "filled_quantity"
	AttrRestingQty = "resting_quantity"
	AttrTakerOrder = "taker_order_id"
	AttrMakerOrder = "maker_order_id"
	AttrTakerFee   = "taker_fee"
	AttrStatus     = "status"
)
