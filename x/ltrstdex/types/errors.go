package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/ltrstdex module sentinel errors.
var (
	ErrInvalidAuthority     = sdkerrors.Register(ModuleName, 1100, "invalid authority for action")
	ErrGovOnlyMarketCreate  = sdkerrors.Register(ModuleName, 1101, "market creation is gov-only in this phase")
	ErrInvalidMarketParams  = sdkerrors.Register(ModuleName, 1102, "invalid market parameters")
	ErrDuplicateTicker      = sdkerrors.Register(ModuleName, 1103, "market ticker already in use")
	ErrDuplicateDenomPair   = sdkerrors.Register(ModuleName, 1104, "market with same base/quote denom pair already exists")
	ErrMarketNotFound       = sdkerrors.Register(ModuleName, 1105, "market not found")
	ErrMarketNotAcceptingOrders = sdkerrors.Register(ModuleName, 1106, "market status does not accept new orders")
	ErrInvalidMarketStatus  = sdkerrors.Register(ModuleName, 1107, "invalid or unsupported market status transition")

	ErrInvalidOrderParams   = sdkerrors.Register(ModuleName, 1200, "invalid order parameters")
	ErrInvalidPriceTick     = sdkerrors.Register(ModuleName, 1201, "price is not a multiple of min_price_tick")
	ErrInvalidQuantityStep  = sdkerrors.Register(ModuleName, 1202, "quantity is not a multiple of min_quantity")
	ErrOrderNotFound        = sdkerrors.Register(ModuleName, 1203, "order not found")
	ErrOrderAlreadyFilled   = sdkerrors.Register(ModuleName, 1204, "order already filled or cancelled")
	ErrPostOnlyCrosses      = sdkerrors.Register(ModuleName, 1205, "post-only order would cross the book and was rejected")
	ErrFOKUnfillable        = sdkerrors.Register(ModuleName, 1206, "FOK order could not be filled completely")
	ErrTooManyOpenOrders    = sdkerrors.Register(ModuleName, 1207, "account exceeded max_open_orders_per_account")
	ErrMarketOrderNeedsNoPrice = sdkerrors.Register(ModuleName, 1208, "market order must not specify a price")
	ErrLimitOrderNeedsPrice = sdkerrors.Register(ModuleName, 1209, "limit order must specify a positive price")
	ErrInsufficientLiquidity = sdkerrors.Register(ModuleName, 1210, "insufficient liquidity to fill order")
	ErrInvalidTIF           = sdkerrors.Register(ModuleName, 1211, "invalid or unsupported time-in-force")
	ErrInvalidSide          = sdkerrors.Register(ModuleName, 1212, "invalid or unspecified side")
	ErrInvalidOrderType     = sdkerrors.Register(ModuleName, 1213, "invalid or unspecified order type")
	ErrExpiredOrder         = sdkerrors.Register(ModuleName, 1214, "order has expired")

	ErrInsufficientFunds    = sdkerrors.Register(ModuleName, 1300, "account has insufficient spendable balance to lock collateral")
	ErrBankTransferFailed   = sdkerrors.Register(ModuleName, 1301, "x/bank transfer failed during order lock or settlement")

	ErrInvalidGenesis       = sdkerrors.Register(ModuleName, 1400, "invalid ltrstdex genesis state")
)
