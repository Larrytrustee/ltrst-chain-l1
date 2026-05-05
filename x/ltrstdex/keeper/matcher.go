package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"ltrstchain/x/ltrstdex/types"
)

// bpsDenom is the basis-points denominator (bps of 10000 = 100 %).
var bpsDenom = math.LegacyNewDec(10000)

// PlaceOrderResult is returned from PlaceOrder to the msg server.
type PlaceOrderResult struct {
	OrderID         string
	FilledQuantity  math.LegacyDec
	RestingQuantity math.LegacyDec
}

// PlaceOrder is the core match-and-rest entrypoint. It:
//
//  1. Loads and validates the market.
//  2. Creates the new SpotOrder record (seq, order_id, derived fields).
//  3. Locks collateral from owner to the ltrstdex module account.
//  4. Walks the opposite book matching fills.
//  5. Routes fill proceeds and taker fees via x/bank.
//  6. Decides what to do with the remainder (rest / cancel / reject).
//
// All state changes are committed to the underlying KV store via
// runtime.KVStoreAdapter on the context's KVStore, and any x/bank
// transfer failure aborts the whole message with an error (SDK
// transaction semantics roll back all mutations).
func (k Keeper) PlaceOrder(
	ctx context.Context,
	owner sdk.AccAddress,
	marketID uint64,
	side types.Side,
	orderType types.OrderType,
	tif types.TimeInForce,
	price, quantity math.LegacyDec,
	expiresAtBlock int64,
	clientOrderID string,
) (PlaceOrderResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// ------------------------------------------------------------------
	// 1. Market lookup + status gate
	// ------------------------------------------------------------------
	market, ok := k.GetMarket(ctx, marketID)
	if !ok {
		return PlaceOrderResult{}, errorsmod.Wrapf(types.ErrMarketNotFound, "market %d", marketID)
	}
	if err := k.MarketAcceptsOrder(market, orderType); err != nil {
		return PlaceOrderResult{}, err
	}

	// ------------------------------------------------------------------
	// 2. Tick / step validation
	// ------------------------------------------------------------------
	if orderType == types.OrderType_ORDER_TYPE_LIMIT {
		if !isMultipleOf(price, market.MinPriceTick) {
			return PlaceOrderResult{}, errorsmod.Wrapf(types.ErrInvalidPriceTick,
				"price %s not a multiple of min_price_tick %s",
				price.String(), market.MinPriceTick.String())
		}
	}
	if !isMultipleOf(quantity, market.MinQuantity) {
		return PlaceOrderResult{}, errorsmod.Wrapf(types.ErrInvalidQuantityStep,
			"quantity %s not a multiple of min_quantity %s",
			quantity.String(), market.MinQuantity.String())
	}

	// ------------------------------------------------------------------
	// 3. Open-orders cap
	// ------------------------------------------------------------------
	if err := k.AssertOwnerUnderMaxOpen(ctx, owner); err != nil {
		return PlaceOrderResult{}, err
	}

	// ------------------------------------------------------------------
	// 4. Build the order record
	// ------------------------------------------------------------------
	seq := k.NextOrderSeq(ctx, marketID)
	orderID := DeriveOrderID(owner, marketID, seq)

	// Compute upfront lock amount.
	lockAmount, lockDenom, err := k.computeLock(market, side, orderType, price, quantity)
	if err != nil {
		return PlaceOrderResult{}, err
	}

	order := types.SpotOrder{
		OrderId:        orderID,
		Owner:          owner.Bytes(),
		MarketId:       marketID,
		Side:           side,
		Type:           orderType,
		Tif:            tif,
		Price:          price,
		Quantity:       quantity,
		FilledQuantity: math.LegacyZeroDec(),
		LockedAmount:   lockAmount,
		CreatedHeight:  sdkCtx.BlockHeight(),
		Seq:            seq,
		ExpiresAtBlock: expiresAtBlock,
	}
	_ = clientOrderID // currently unused on-chain; reserved for future event emission

	// ------------------------------------------------------------------
	// 5. Lock collateral
	// ------------------------------------------------------------------
	lockCoin := sdk.NewCoin(lockDenom, lockAmount.TruncateInt())
	if !lockAmount.IsPositive() {
		return PlaceOrderResult{}, errorsmod.Wrap(types.ErrInvalidOrderParams, "computed lock amount is not positive")
	}
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, owner, types.ModuleName, sdk.NewCoins(lockCoin)); err != nil {
		return PlaceOrderResult{}, errorsmod.Wrapf(types.ErrInsufficientFunds,
			"lock %s: %s", lockCoin.String(), err.Error())
	}

	// ------------------------------------------------------------------
	// 6. Post-only safety
	// ------------------------------------------------------------------
	if market.Status == types.MarketStatus_MARKET_STATUS_POST_ONLY {
		if orderType == types.OrderType_ORDER_TYPE_LIMIT && k.orderCrossesBook(ctx, order) {
			// Refund and reject.
			if err := k.refund(ctx, owner, lockDenom, lockAmount); err != nil {
				return PlaceOrderResult{}, err
			}
			return PlaceOrderResult{}, types.ErrPostOnlyCrosses
		}
	}

	// ------------------------------------------------------------------
	// 7. FOK pre-check
	// ------------------------------------------------------------------
	if tif == types.TimeInForce_TIF_FOK && orderType == types.OrderType_ORDER_TYPE_LIMIT {
		if !k.canFillCompletely(ctx, order, market) {
			if err := k.refund(ctx, owner, lockDenom, lockAmount); err != nil {
				return PlaceOrderResult{}, err
			}
			return PlaceOrderResult{}, types.ErrFOKUnfillable
		}
	}

	// ------------------------------------------------------------------
	// 8. Match loop
	// ------------------------------------------------------------------
	filled, err := k.matchLoop(ctx, &order, market)
	if err != nil {
		return PlaceOrderResult{}, err
	}

	remaining := order.Quantity.Sub(order.FilledQuantity)

	// ------------------------------------------------------------------
	// 9. Handle remainder according to TIF / type
	// ------------------------------------------------------------------
	switch {
	case !remaining.IsPositive():
		// Fully filled. Any slack in locked_amount refunds.
		if err := k.refundRemainingLock(ctx, owner, &order, market); err != nil {
			return PlaceOrderResult{}, err
		}
		// Order does not persist after full fill; keep no record.
		k.DeleteOrder(ctx, order)

	case orderType == types.OrderType_ORDER_TYPE_MARKET || tif == types.TimeInForce_TIF_IOC:
		// Market orders and IOC refund the remainder lock and don't rest.
		if err := k.refundRemainingLock(ctx, owner, &order, market); err != nil {
			return PlaceOrderResult{}, err
		}
		k.DeleteOrder(ctx, order)

	default:
		// GTC rests. Persist the order and insert into the orderbook index.
		if err := k.SetOrder(ctx, order); err != nil {
			return PlaceOrderResult{}, err
		}
		if err := k.InsertOrderbookIndex(ctx, order); err != nil {
			return PlaceOrderResult{}, err
		}
	}

	_ = filled // kept for log/event expansion later

	return PlaceOrderResult{
		OrderID:         orderID,
		FilledQuantity:  order.FilledQuantity,
		RestingQuantity: math.LegacyMaxDec(math.LegacyZeroDec(), order.Quantity.Sub(order.FilledQuantity)),
	}, nil
}

// matchLoop walks the opposite book and fills as much of `taker` as
// possible. Mutates taker.FilledQuantity and taker.LockedAmount.
// Returns the total base quantity filled.
func (k Keeper) matchLoop(ctx context.Context, taker *types.SpotOrder, market types.SpotMarket) (math.LegacyDec, error) {
	totalFilled := math.LegacyZeroDec()
	// Iterate the opposite side in price-time order and break when we
	// hit a resting order whose price no longer crosses or when taker
	// is fully filled.

	oppositeIsAsk := taker.Side == types.Side_SIDE_BUY

	walk := func(maker types.SpotOrder) (stop bool, err error) {
		// Price crossing check for LIMIT takers. MARKET takers take
		// anything.
		if taker.Type == types.OrderType_ORDER_TYPE_LIMIT {
			if taker.Side == types.Side_SIDE_BUY && maker.Price.GT(taker.Price) {
				return true, nil
			}
			if taker.Side == types.Side_SIDE_SELL && maker.Price.LT(taker.Price) {
				return true, nil
			}
		}

		takerRemaining := taker.Quantity.Sub(taker.FilledQuantity)
		if !takerRemaining.IsPositive() {
			return true, nil
		}
		makerRemaining := maker.Quantity.Sub(maker.FilledQuantity)
		if !makerRemaining.IsPositive() {
			return false, nil
		}

		fillQty := math.LegacyMinDec(takerRemaining, makerRemaining)
		if fillQty.IsZero() {
			return false, nil
		}

		// Settle and update orders.
		if err := k.settleFill(ctx, taker, &maker, market, fillQty); err != nil {
			return false, err
		}

		totalFilled = totalFilled.Add(fillQty)

		// Maker post-state: either persist with updated filled_quantity,
		// or remove if fully filled.
		if maker.FilledQuantity.GTE(maker.Quantity) {
			// Full fill — remove from book.
			if err := k.DeleteOrderbookIndex(ctx, maker); err != nil {
				return false, err
			}
			// Refund any lock slack still on the maker.
			ownerAddr := sdk.AccAddress(maker.Owner)
			if err := k.refundRemainingLock(ctx, ownerAddr, &maker, market); err != nil {
				return false, err
			}
			k.DeleteOrder(ctx, maker)
		} else {
			if err := k.SetOrder(ctx, maker); err != nil {
				return false, err
			}
		}

		// Taker fully filled? stop.
		if taker.FilledQuantity.GTE(taker.Quantity) {
			return true, nil
		}
		return false, nil
	}

	var loopErr error
	cb := func(maker types.SpotOrder) bool {
		stop, err := walk(maker)
		if err != nil {
			loopErr = err
			return true
		}
		return stop
	}
	if oppositeIsAsk {
		k.IterateAskSide(ctx, taker.MarketId, cb)
	} else {
		k.IterateBidSide(ctx, taker.MarketId, cb)
	}
	return totalFilled, loopErr
}

// settleFill performs the x/bank transfers for a single match between
// a taker and a maker at the maker's resting price. Mutates both
// order structs' FilledQuantity and LockedAmount.
func (k Keeper) settleFill(
	ctx context.Context,
	taker *types.SpotOrder,
	maker *types.SpotOrder,
	market types.SpotMarket,
	fillQty math.LegacyDec,
) error {
	execPrice := maker.Price
	quoteGross := execPrice.Mul(fillQty) // quote moved for base delivered

	// Which side is the taker?
	takerIsBuy := taker.Side == types.Side_SIDE_BUY

	takerFeeRate := math.LegacyNewDec(int64(market.TakerFeeBps)).Quo(bpsDenom)
	takerFee := quoteGross.Mul(takerFeeRate)
	// (Phase 1: maker fee is 0; we intentionally don't compute it.)

	takerAddr := sdk.AccAddress(taker.Owner)
	makerAddr := sdk.AccAddress(maker.Owner)
	feeCollector := authtypes.FeeCollectorName

	if takerIsBuy {
		// Base: module → taker (buyer)
		baseCoin := sdk.NewCoin(market.BaseDenom, fillQty.TruncateInt())
		if baseCoin.Amount.IsPositive() {
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, takerAddr, sdk.NewCoins(baseCoin)); err != nil {
				return errorsmod.Wrapf(types.ErrBankTransferFailed, "base to buyer: %s", err.Error())
			}
		}
		// Quote: module → maker (seller)
		quoteCoin := sdk.NewCoin(market.QuoteDenom, quoteGross.TruncateInt())
		if quoteCoin.Amount.IsPositive() {
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, makerAddr, sdk.NewCoins(quoteCoin)); err != nil {
				return errorsmod.Wrapf(types.ErrBankTransferFailed, "quote to seller: %s", err.Error())
			}
		}
		// Taker fee: module → fee_collector (in quote)
		feeCoin := sdk.NewCoin(market.QuoteDenom, takerFee.TruncateInt())
		if feeCoin.Amount.IsPositive() {
			if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, feeCollector, sdk.NewCoins(feeCoin)); err != nil {
				return errorsmod.Wrapf(types.ErrBankTransferFailed, "taker fee: %s", err.Error())
			}
		}

		// Taker's lock decreases by quoteGross + takerFee
		takerLockUsed := quoteGross.Add(takerFee)
		taker.LockedAmount = taker.LockedAmount.Sub(takerLockUsed)
		if taker.LockedAmount.IsNegative() {
			return errorsmod.Wrapf(types.ErrBankTransferFailed,
				"taker lock went negative: %s", taker.LockedAmount.String())
		}
		// Maker's lock decreases by fillQty base
		maker.LockedAmount = maker.LockedAmount.Sub(fillQty)
	} else {
		// Taker is SELL.
		// Base: module → maker (buyer) from seller's base lock
		baseCoin := sdk.NewCoin(market.BaseDenom, fillQty.TruncateInt())
		if baseCoin.Amount.IsPositive() {
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, makerAddr, sdk.NewCoins(baseCoin)); err != nil {
				return errorsmod.Wrapf(types.ErrBankTransferFailed, "base to buyer: %s", err.Error())
			}
		}
		// Quote: module → taker (seller), minus taker fee
		quoteProceeds := quoteGross.Sub(takerFee)
		quoteCoin := sdk.NewCoin(market.QuoteDenom, quoteProceeds.TruncateInt())
		if quoteCoin.Amount.IsPositive() {
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, takerAddr, sdk.NewCoins(quoteCoin)); err != nil {
				return errorsmod.Wrapf(types.ErrBankTransferFailed, "quote to seller: %s", err.Error())
			}
		}
		// Taker fee: module → fee_collector
		feeCoin := sdk.NewCoin(market.QuoteDenom, takerFee.TruncateInt())
		if feeCoin.Amount.IsPositive() {
			if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, feeCollector, sdk.NewCoins(feeCoin)); err != nil {
				return errorsmod.Wrapf(types.ErrBankTransferFailed, "taker fee: %s", err.Error())
			}
		}

		// Taker's lock (base) decreases by fillQty
		taker.LockedAmount = taker.LockedAmount.Sub(fillQty)
		// Maker's lock (quote) decreases by quoteGross (no maker fee in P1)
		maker.LockedAmount = maker.LockedAmount.Sub(quoteGross)
		if maker.LockedAmount.IsNegative() {
			return errorsmod.Wrapf(types.ErrBankTransferFailed,
				"maker lock went negative: %s", maker.LockedAmount.String())
		}
	}

	taker.FilledQuantity = taker.FilledQuantity.Add(fillQty)
	maker.FilledQuantity = maker.FilledQuantity.Add(fillQty)

	// Persist a SpotFill record.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	makerFeeZero := math.LegacyZeroDec()
	fill := types.SpotFill{
		FillSeq:       k.NextFillSeq(ctx),
		MarketId:      market.Id,
		TakerOrderId:  taker.OrderId,
		MakerOrderId:  maker.OrderId,
		Taker:         taker.Owner,
		Maker:         maker.Owner,
		TakerSide:     taker.Side,
		Price:         execPrice,
		Quantity:      fillQty,
		TakerFee:      takerFee,
		MakerFee:      makerFeeZero,
		BlockHeight:   sdkCtx.BlockHeight(),
	}
	if err := k.AppendFill(ctx, fill); err != nil {
		return fmt.Errorf("append fill: %w", err)
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSpotFill,
		sdk.NewAttribute(types.AttrMarketID, fmt.Sprintf("%d", market.Id)),
		sdk.NewAttribute(types.AttrTakerOrder, taker.OrderId),
		sdk.NewAttribute(types.AttrMakerOrder, maker.OrderId),
		sdk.NewAttribute(types.AttrPrice, execPrice.String()),
		sdk.NewAttribute(types.AttrQuantity, fillQty.String()),
		sdk.NewAttribute(types.AttrTakerFee, takerFee.String()),
	))
	return nil
}

// computeLock returns (amount, denom) that should move from owner's
// balance to the module account when this order is placed.
func (k Keeper) computeLock(
	market types.SpotMarket,
	side types.Side,
	orderType types.OrderType,
	price, quantity math.LegacyDec,
) (math.LegacyDec, string, error) {
	takerFeeRate := math.LegacyNewDec(int64(market.TakerFeeBps)).Quo(bpsDenom)
	feeMultiplier := math.LegacyOneDec().Add(takerFeeRate)

	switch side {
	case types.Side_SIDE_BUY:
		var notional math.LegacyDec
		if orderType == types.OrderType_ORDER_TYPE_MARKET {
			// MARKET BUY: we don't know the execution price in advance.
			// Require the caller to have locked based on their own
			// understanding of available liquidity. Simplest: reject
			// MARKET-BUY at Phase 1 and fall back to requiring LIMIT
			// with high price. This keeps bank-lock semantics strict.
			return math.LegacyDec{}, "", errorsmod.Wrap(types.ErrInvalidOrderType,
				"MARKET BUY not supported in Phase 1 — use a LIMIT order with a high price instead")
		}
		notional = price.Mul(quantity).Mul(feeMultiplier)
		return notional.Ceil(), market.QuoteDenom, nil

	case types.Side_SIDE_SELL:
		// SELL locks base quantity regardless of type.
		return quantity, market.BaseDenom, nil
	default:
		return math.LegacyDec{}, "", types.ErrInvalidSide
	}
}

// refundRemainingLock sends any remaining locked_amount back to the
// owner and zeroes the field. Safe to call for orders that have no
// remaining lock (no-op).
func (k Keeper) refundRemainingLock(
	ctx context.Context,
	owner sdk.AccAddress,
	o *types.SpotOrder,
	market types.SpotMarket,
) error {
	if o.LockedAmount.IsNil() || !o.LockedAmount.IsPositive() {
		return nil
	}

	var denom string
	switch o.Side {
	case types.Side_SIDE_BUY:
		denom = market.QuoteDenom
	case types.Side_SIDE_SELL:
		denom = market.BaseDenom
	default:
		return types.ErrInvalidSide
	}
	return k.refund(ctx, owner, denom, o.LockedAmount)
}

// refund is the low-level "module → account" transfer used by cancel,
// post-only rejection, FOK failure, and remainder cleanup.
func (k Keeper) refund(ctx context.Context, to sdk.AccAddress, denom string, amount math.LegacyDec) error {
	if !amount.IsPositive() {
		return nil
	}
	coin := sdk.NewCoin(denom, amount.Ceil().TruncateInt())
	if !coin.Amount.IsPositive() {
		return nil
	}
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, to, sdk.NewCoins(coin)); err != nil {
		return errorsmod.Wrapf(types.ErrBankTransferFailed, "refund %s to %s: %s",
			coin.String(), to.String(), err.Error())
	}
	return nil
}

// orderCrossesBook returns true if the given LIMIT order would cross
// the book immediately. Used for POST_ONLY rejection.
func (k Keeper) orderCrossesBook(ctx context.Context, o types.SpotOrder) bool {
	if o.Side == types.Side_SIDE_BUY {
		best, ok := k.BestAsk(ctx, o.MarketId)
		if !ok {
			return false
		}
		return !o.Price.LT(best.Price)
	}
	best, ok := k.BestBid(ctx, o.MarketId)
	if !ok {
		return false
	}
	return !o.Price.GT(best.Price)
}

// canFillCompletely is an O(book-depth) dry-run used for FOK.
// Returns true iff total maker quantity at crossing prices >= taker qty.
func (k Keeper) canFillCompletely(ctx context.Context, taker types.SpotOrder, market types.SpotMarket) bool {
	available := math.LegacyZeroDec()
	need := taker.Quantity

	check := func(maker types.SpotOrder) bool {
		if taker.Type == types.OrderType_ORDER_TYPE_LIMIT {
			if taker.Side == types.Side_SIDE_BUY && maker.Price.GT(taker.Price) {
				return true
			}
			if taker.Side == types.Side_SIDE_SELL && maker.Price.LT(taker.Price) {
				return true
			}
		}
		r := maker.Quantity.Sub(maker.FilledQuantity)
		if r.IsPositive() {
			available = available.Add(r)
		}
		return available.GTE(need)
	}

	if taker.Side == types.Side_SIDE_BUY {
		k.IterateAskSide(ctx, market.Id, check)
	} else {
		k.IterateBidSide(ctx, market.Id, check)
	}
	return available.GTE(need)
}

// CancelOrder removes a live order, refunds its remaining lock, and
// deletes its orderbook + user-index entries. Allowed to owner or
// to the module authority (e.g. for HALTED cleanup).
func (k Keeper) CancelOrder(
	ctx context.Context,
	authority sdk.AccAddress,
	marketID uint64,
	orderID string,
) error {
	o, ok := k.GetOrder(ctx, marketID, orderID)
	if !ok {
		return errorsmod.Wrapf(types.ErrOrderNotFound, "order %s on market %d", orderID, marketID)
	}
	ownerAddr := sdk.AccAddress(o.Owner)
	if !authority.Equals(ownerAddr) && authority.String() != k.GetAuthority() {
		return errorsmod.Wrap(types.ErrInvalidAuthority,
			"only the order owner or the gov authority may cancel this order")
	}
	m, ok := k.GetMarket(ctx, marketID)
	if !ok {
		return errorsmod.Wrapf(types.ErrMarketNotFound, "market %d", marketID)
	}
	if err := k.DeleteOrderbookIndex(ctx, o); err != nil {
		return err
	}
	if err := k.refundRemainingLock(ctx, ownerAddr, &o, m); err != nil {
		return err
	}
	k.DeleteOrder(ctx, o)
	return nil
}

// CancelAllOrdersForOwner cancels every live order owned by addr,
// optionally scoped to a single market (0 = all markets). Returns
// the count cancelled.
func (k Keeper) CancelAllOrdersForOwner(
	ctx context.Context,
	owner sdk.AccAddress,
	marketFilter uint64,
) (uint32, error) {
	var toCancel []types.SpotOrder
	k.IterateOwnerOrders(ctx, owner, func(o types.SpotOrder) bool {
		if marketFilter != 0 && o.MarketId != marketFilter {
			return false
		}
		toCancel = append(toCancel, o)
		return false
	})
	for i := range toCancel {
		if err := k.CancelOrder(ctx, owner, toCancel[i].MarketId, toCancel[i].OrderId); err != nil {
			return 0, err
		}
	}
	return uint32(len(toCancel)), nil
}

// isMultipleOf returns true if value is an exact integer multiple of step.
// Step must be positive.
func isMultipleOf(value, step math.LegacyDec) bool {
	if step.IsNil() || !step.IsPositive() {
		return false
	}
	// value / step should have no fractional part.
	quot := value.Quo(step)
	if !quot.Sub(quot.TruncateDec()).IsZero() {
		return false
	}
	return true
}
