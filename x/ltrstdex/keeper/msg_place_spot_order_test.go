package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "ltrstchain/testutil/keeper"
	"ltrstchain/x/ltrstdex/keeper"
	"ltrstchain/x/ltrstdex/types"
)

// placeTestSetup is shared boilerplate: spins up a keeper, creates
// one LTRST/USDC market with a 30-bps taker fee, and returns the
// msgServer plus keeper and fakes so each test can seed balances.
func placeTestSetup(t testing.TB) (
	keeper.Keeper,
	types.MsgServer,
	*keepertest.FakeAccountKeeper,
	*keepertest.FakeBankKeeper,
	sdk.Context,
	uint64,
) {
	t.Helper()
	k, ak, bk, ctx := keepertest.LtrstdexKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	createReq := &types.MsgCreateSpotMarket{
		Authority:     k.GetAuthority(),
		Ticker:        "LTRST/USDC",
		BaseDenom:     "ultrst",
		QuoteDenom:    "uusdc",
		MinPriceTick:  math.LegacyMustNewDecFromStr("0.01"),
		MinQuantity:   math.LegacyMustNewDecFromStr("1"),
		MakerFeeBps:   0,
		TakerFeeBps:   30,
		InitialStatus: types.MarketStatus_MARKET_STATUS_ACTIVE,
	}
	resp, err := ms.CreateSpotMarket(sdk.WrapSDKContext(ctx), createReq)
	require.NoError(t, err)
	return k, ms, ak, bk, ctx, resp.MarketId
}

// fundAcc is a test helper that drops coins onto an address via the
// fake bank keeper.
func fundAcc(bk *keepertest.FakeBankKeeper, addr sdk.AccAddress, denom string, amount int64) {
	bk.SetBalance(addr, sdk.NewCoins(sdk.NewCoin(denom, math.NewInt(amount))))
}

func TestMsgPlaceSpotOrder_SellLimitGTC_HappyPath(t *testing.T) {
	k, ms, ak, bk, ctx, marketID := placeTestSetup(t)

	seller := sdk.AccAddress([]byte("seller00000000000001"))
	fundAcc(bk, seller, "ultrst", 100)

	req := &types.MsgPlaceSpotOrder{
		Owner:    seller.String(),
		MarketId: marketID,
		Side:     types.Side_SIDE_SELL,
		Type:     types.OrderType_ORDER_TYPE_LIMIT,
		Tif:      types.TimeInForce_TIF_GTC,
		Price:    math.LegacyMustNewDecFromStr("1.00"),
		Quantity: math.LegacyMustNewDecFromStr("10"),
	}
	resp, err := ms.PlaceSpotOrder(sdk.WrapSDKContext(ctx), req)
	require.NoError(t, err)
	require.NotEmpty(t, resp.OrderId)
	require.True(t, resp.FilledQuantity.IsZero(), "no cross, nothing filled")
	require.Equal(t, "10.000000000000000000", resp.RestingQuantity.String())

	// Base collateral must have moved from seller → module account.
	modAddr := ak.GetModuleAddress(types.ModuleName)
	modBal := bk.GetBalance(ctx, modAddr, "ultrst")
	require.True(t, modBal.Amount.Equal(math.NewInt(10)), "module should hold 10 ultrst lock")
	sellerBal := bk.GetBalance(ctx, seller, "ultrst")
	require.True(t, sellerBal.Amount.Equal(math.NewInt(90)), "seller should have 90 ultrst left")

	// Order must be persisted and indexed.
	o, ok := k.GetOrder(ctx, marketID, resp.OrderId)
	require.True(t, ok)
	require.Equal(t, types.Side_SIDE_SELL, o.Side)

	best, found := k.BestAsk(ctx, marketID)
	require.True(t, found)
	require.Equal(t, resp.OrderId, best.OrderId)
}

func TestMsgPlaceSpotOrder_BuyLimitGTC_LocksQuote(t *testing.T) {
	_, ms, ak, bk, ctx, marketID := placeTestSetup(t)

	buyer := sdk.AccAddress([]byte("buyer0000000000000001"))
	// 1.00 * 10 * (1 + 0.003) = 10.03 → ceil → 11 uusdc lock
	fundAcc(bk, buyer, "uusdc", 1000)

	req := &types.MsgPlaceSpotOrder{
		Owner:    buyer.String(),
		MarketId: marketID,
		Side:     types.Side_SIDE_BUY,
		Type:     types.OrderType_ORDER_TYPE_LIMIT,
		Tif:      types.TimeInForce_TIF_GTC,
		Price:    math.LegacyMustNewDecFromStr("1.00"),
		Quantity: math.LegacyMustNewDecFromStr("10"),
	}
	resp, err := ms.PlaceSpotOrder(sdk.WrapSDKContext(ctx), req)
	require.NoError(t, err)
	require.NotEmpty(t, resp.OrderId)

	// Quote collateral locked in module account.
	modAddr := ak.GetModuleAddress(types.ModuleName)
	modBal := bk.GetBalance(ctx, modAddr, "uusdc")
	require.True(t, modBal.Amount.IsPositive(), "module should hold a uusdc lock")
	buyerBal := bk.GetBalance(ctx, buyer, "uusdc")
	require.True(t, buyerBal.Amount.LT(math.NewInt(1000)), "buyer balance must have decreased")
}

func TestMsgPlaceSpotOrder_IOCCrossesAndFills(t *testing.T) {
	k, ms, _, bk, ctx, marketID := placeTestSetup(t)

	// Maker: SELL 10 @ 1.00 (rests)
	maker := sdk.AccAddress([]byte("maker0000000000000001"))
	fundAcc(bk, maker, "ultrst", 100)
	makerReq := &types.MsgPlaceSpotOrder{
		Owner:    maker.String(),
		MarketId: marketID,
		Side:     types.Side_SIDE_SELL,
		Type:     types.OrderType_ORDER_TYPE_LIMIT,
		Tif:      types.TimeInForce_TIF_GTC,
		Price:    math.LegacyMustNewDecFromStr("1.00"),
		Quantity: math.LegacyMustNewDecFromStr("10"),
	}
	_, err := ms.PlaceSpotOrder(sdk.WrapSDKContext(ctx), makerReq)
	require.NoError(t, err)

	// Taker: BUY 5 @ 1.00 IOC — should fully cross and no remainder.
	taker := sdk.AccAddress([]byte("taker0000000000000001"))
	fundAcc(bk, taker, "uusdc", 1000)
	takerReq := &types.MsgPlaceSpotOrder{
		Owner:    taker.String(),
		MarketId: marketID,
		Side:     types.Side_SIDE_BUY,
		Type:     types.OrderType_ORDER_TYPE_LIMIT,
		Tif:      types.TimeInForce_TIF_IOC,
		Price:    math.LegacyMustNewDecFromStr("1.00"),
		Quantity: math.LegacyMustNewDecFromStr("5"),
	}
	resp, err := ms.PlaceSpotOrder(sdk.WrapSDKContext(ctx), takerReq)
	require.NoError(t, err)
	require.Equal(t, "5.000000000000000000", resp.FilledQuantity.String(), "taker should fully fill")
	require.True(t, resp.RestingQuantity.IsZero(), "IOC should not rest")

	// Taker buyer must now hold the 5 base coins.
	require.Equal(t, int64(5), bk.GetBalance(ctx, taker, "ultrst").Amount.Int64())

	// Maker is still on book with 5 remaining.
	best, found := k.BestAsk(ctx, marketID)
	require.True(t, found)
	require.Equal(t, "10.000000000000000000", best.Quantity.String())
	require.Equal(t, "5.000000000000000000", best.FilledQuantity.String())

	// Taker's order record must NOT persist (IOC without rest).
	_, ok := k.GetOrder(ctx, marketID, resp.OrderId)
	require.False(t, ok, "IOC orders with full fill do not persist")

	// A fill record must have been appended.
	var fillCount int
	k.IterateFills(ctx, marketID, func(types.SpotFill) bool {
		fillCount++
		return false
	})
	require.Equal(t, 1, fillCount, "exactly one fill recorded")
}

func TestMsgPlaceSpotOrder_MarketFrozenRejectsNewOrders(t *testing.T) {
	k, ms, _, bk, ctx, marketID := placeTestSetup(t)

	// Pause the market via the gov-gated status update.
	_, err := ms.UpdateSpotMarketStatus(sdk.WrapSDKContext(ctx), &types.MsgUpdateSpotMarketStatus{
		Authority: k.GetAuthority(),
		MarketId:  marketID,
		NewStatus: types.MarketStatus_MARKET_STATUS_PAUSED,
	})
	require.NoError(t, err)

	owner := sdk.AccAddress([]byte("owner00000000000000a1"))
	fundAcc(bk, owner, "ultrst", 100)
	_, err = ms.PlaceSpotOrder(sdk.WrapSDKContext(ctx), &types.MsgPlaceSpotOrder{
		Owner:    owner.String(),
		MarketId: marketID,
		Side:     types.Side_SIDE_SELL,
		Type:     types.OrderType_ORDER_TYPE_LIMIT,
		Tif:      types.TimeInForce_TIF_GTC,
		Price:    math.LegacyMustNewDecFromStr("1.00"),
		Quantity: math.LegacyMustNewDecFromStr("10"),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrMarketNotAcceptingOrders)
}

func TestMsgPlaceSpotOrder_MarketNotFound(t *testing.T) {
	_, ms, _, bk, ctx, _ := placeTestSetup(t)

	owner := sdk.AccAddress([]byte("owner00000000000000a2"))
	fundAcc(bk, owner, "ultrst", 100)
	_, err := ms.PlaceSpotOrder(sdk.WrapSDKContext(ctx), &types.MsgPlaceSpotOrder{
		Owner:    owner.String(),
		MarketId: 9999,
		Side:     types.Side_SIDE_SELL,
		Type:     types.OrderType_ORDER_TYPE_LIMIT,
		Tif:      types.TimeInForce_TIF_GTC,
		Price:    math.LegacyMustNewDecFromStr("1.00"),
		Quantity: math.LegacyMustNewDecFromStr("10"),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrMarketNotFound)
}

func TestMsgPlaceSpotOrder_PriceNotAlignedToTick(t *testing.T) {
	_, ms, _, bk, ctx, marketID := placeTestSetup(t)

	owner := sdk.AccAddress([]byte("owner00000000000000a3"))
	fundAcc(bk, owner, "ultrst", 100)
	_, err := ms.PlaceSpotOrder(sdk.WrapSDKContext(ctx), &types.MsgPlaceSpotOrder{
		Owner:    owner.String(),
		MarketId: marketID,
		Side:     types.Side_SIDE_SELL,
		Type:     types.OrderType_ORDER_TYPE_LIMIT,
		Tif:      types.TimeInForce_TIF_GTC,
		// 0.015 is NOT a multiple of tick 0.01 (it's 1.5 ticks).
		Price:    math.LegacyMustNewDecFromStr("0.015"),
		Quantity: math.LegacyMustNewDecFromStr("10"),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidPriceTick)
}

func TestMsgPlaceSpotOrder_QuantityBelowMin(t *testing.T) {
	_, ms, _, bk, ctx, marketID := placeTestSetup(t)

	owner := sdk.AccAddress([]byte("owner00000000000000a4"))
	fundAcc(bk, owner, "ultrst", 100)
	_, err := ms.PlaceSpotOrder(sdk.WrapSDKContext(ctx), &types.MsgPlaceSpotOrder{
		Owner:    owner.String(),
		MarketId: marketID,
		Side:     types.Side_SIDE_SELL,
		Type:     types.OrderType_ORDER_TYPE_LIMIT,
		Tif:      types.TimeInForce_TIF_GTC,
		Price:    math.LegacyMustNewDecFromStr("1.00"),
		// 0.5 is below the 1 min_quantity step.
		Quantity: math.LegacyMustNewDecFromStr("0.5"),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidQuantityStep)
}

func TestMsgPlaceSpotOrder_InsufficientBalance(t *testing.T) {
	_, ms, _, bk, ctx, marketID := placeTestSetup(t)

	owner := sdk.AccAddress([]byte("owner00000000000000a5"))
	// Seed only 3 ultrst but try to lock 10.
	fundAcc(bk, owner, "ultrst", 3)

	_, err := ms.PlaceSpotOrder(sdk.WrapSDKContext(ctx), &types.MsgPlaceSpotOrder{
		Owner:    owner.String(),
		MarketId: marketID,
		Side:     types.Side_SIDE_SELL,
		Type:     types.OrderType_ORDER_TYPE_LIMIT,
		Tif:      types.TimeInForce_TIF_GTC,
		Price:    math.LegacyMustNewDecFromStr("1.00"),
		Quantity: math.LegacyMustNewDecFromStr("10"),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInsufficientFunds)
}
