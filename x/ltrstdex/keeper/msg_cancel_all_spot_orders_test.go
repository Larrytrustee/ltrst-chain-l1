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

// twoMarketsSetup builds a keeper with two distinct spot markets so
// we can exercise the market-scoped cancel path.
func twoMarketsSetup(t testing.TB) (
	keeper.Keeper,
	types.MsgServer,
	*keepertest.FakeBankKeeper,
	sdk.Context,
	uint64, // marketA
	uint64, // marketB
) {
	t.Helper()
	k, _, bk, ctx := keepertest.LtrstdexKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	mkA, err := ms.CreateSpotMarket(sdk.WrapSDKContext(ctx), &types.MsgCreateSpotMarket{
		Authority:     k.GetAuthority(),
		Ticker:        "LTRST/USDC",
		BaseDenom:     "ultrst",
		QuoteDenom:    "uusdc",
		MinPriceTick:  math.LegacyMustNewDecFromStr("0.01"),
		MinQuantity:   math.LegacyMustNewDecFromStr("1"),
		TakerFeeBps:   30,
		InitialStatus: types.MarketStatus_MARKET_STATUS_ACTIVE,
	})
	require.NoError(t, err)

	mkB, err := ms.CreateSpotMarket(sdk.WrapSDKContext(ctx), &types.MsgCreateSpotMarket{
		Authority:     k.GetAuthority(),
		Ticker:        "BTC/USDC",
		BaseDenom:     "ubtc",
		QuoteDenom:    "uusdc",
		MinPriceTick:  math.LegacyMustNewDecFromStr("0.01"),
		MinQuantity:   math.LegacyMustNewDecFromStr("1"),
		TakerFeeBps:   30,
		InitialStatus: types.MarketStatus_MARKET_STATUS_ACTIVE,
	})
	require.NoError(t, err)

	return k, ms, bk, ctx, mkA.MarketId, mkB.MarketId
}

func TestMsgCancelAllSpotOrders_ScopedToMarket(t *testing.T) {
	k, ms, bk, ctx, mkA, mkB := twoMarketsSetup(t)

	owner := sdk.AccAddress([]byte("owner00000000000000c4"))
	// Enough of both base denoms for three sells.
	fundAcc(bk, owner, "ultrst", 100)
	bk.AddBalance(owner, sdk.NewCoins(sdk.NewCoin("ubtc", math.NewInt(100))))

	// Two orders on market A.
	_, err := ms.PlaceSpotOrder(sdk.WrapSDKContext(ctx), &types.MsgPlaceSpotOrder{
		Owner:    owner.String(),
		MarketId: mkA,
		Side:     types.Side_SIDE_SELL,
		Type:     types.OrderType_ORDER_TYPE_LIMIT,
		Tif:      types.TimeInForce_TIF_GTC,
		Price:    math.LegacyMustNewDecFromStr("1.00"),
		Quantity: math.LegacyMustNewDecFromStr("10"),
	})
	require.NoError(t, err)
	_, err = ms.PlaceSpotOrder(sdk.WrapSDKContext(ctx), &types.MsgPlaceSpotOrder{
		Owner:    owner.String(),
		MarketId: mkA,
		Side:     types.Side_SIDE_SELL,
		Type:     types.OrderType_ORDER_TYPE_LIMIT,
		Tif:      types.TimeInForce_TIF_GTC,
		Price:    math.LegacyMustNewDecFromStr("1.01"),
		Quantity: math.LegacyMustNewDecFromStr("5"),
	})
	require.NoError(t, err)

	// One order on market B.
	_, err = ms.PlaceSpotOrder(sdk.WrapSDKContext(ctx), &types.MsgPlaceSpotOrder{
		Owner:    owner.String(),
		MarketId: mkB,
		Side:     types.Side_SIDE_SELL,
		Type:     types.OrderType_ORDER_TYPE_LIMIT,
		Tif:      types.TimeInForce_TIF_GTC,
		Price:    math.LegacyMustNewDecFromStr("1.00"),
		Quantity: math.LegacyMustNewDecFromStr("3"),
	})
	require.NoError(t, err)

	resp, err := ms.CancelAllSpotOrders(sdk.WrapSDKContext(ctx), &types.MsgCancelAllSpotOrders{
		Owner:    owner.String(),
		MarketId: mkA,
	})
	require.NoError(t, err)
	require.Equal(t, uint32(2), resp.CancelledCount, "two orders on market A should be cancelled")

	// Market A should be empty.
	_, foundA := k.BestAsk(ctx, mkA)
	require.False(t, foundA)

	// Market B should still have its ask.
	_, foundB := k.BestAsk(ctx, mkB)
	require.True(t, foundB, "market B order must remain untouched")
}

func TestMsgCancelAllSpotOrders_NoOrdersIsNotAnError(t *testing.T) {
	_, ms, _, ctx, _, _ := twoMarketsSetup(t)

	owner := sdk.AccAddress([]byte("nowhere000000000000a1")).String()
	resp, err := ms.CancelAllSpotOrders(sdk.WrapSDKContext(ctx), &types.MsgCancelAllSpotOrders{
		Owner:    owner,
		MarketId: 0,
	})
	require.NoError(t, err, "cancel-all on empty book is a no-op, not an error")
	require.Equal(t, uint32(0), resp.CancelledCount)
}
