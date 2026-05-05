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

func TestMsgUpdateSpotMarketStatus_ActiveToFrozenBlocksOrders(t *testing.T) {
	k, _, bk, ctx := keepertest.LtrstdexKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	create, err := ms.CreateSpotMarket(sdk.WrapSDKContext(ctx), &types.MsgCreateSpotMarket{
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

	// Transition ACTIVE → PAUSED (which is effectively "frozen" in the
	// lifecycle state machine — PlaceOrder rejects PAUSED outright).
	_, err = ms.UpdateSpotMarketStatus(sdk.WrapSDKContext(ctx), &types.MsgUpdateSpotMarketStatus{
		Authority: k.GetAuthority(),
		MarketId:  create.MarketId,
		NewStatus: types.MarketStatus_MARKET_STATUS_PAUSED,
	})
	require.NoError(t, err)

	// New order attempt must be rejected with ErrMarketNotAcceptingOrders.
	owner := sdk.AccAddress([]byte("owner00000000000000u1"))
	fundAcc(bk, owner, "ultrst", 100)
	_, err = ms.PlaceSpotOrder(sdk.WrapSDKContext(ctx), &types.MsgPlaceSpotOrder{
		Owner:    owner.String(),
		MarketId: create.MarketId,
		Side:     types.Side_SIDE_SELL,
		Type:     types.OrderType_ORDER_TYPE_LIMIT,
		Tif:      types.TimeInForce_TIF_GTC,
		Price:    math.LegacyMustNewDecFromStr("1.00"),
		Quantity: math.LegacyMustNewDecFromStr("10"),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrMarketNotAcceptingOrders)

	// And the stored market record reflects the new status.
	m, ok := k.GetMarket(ctx, create.MarketId)
	require.True(t, ok)
	require.Equal(t, types.MarketStatus_MARKET_STATUS_PAUSED, m.Status)
}

func TestMsgUpdateSpotMarketStatus_NonGovRejected(t *testing.T) {
	k, _, _, ctx := keepertest.LtrstdexKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	create, err := ms.CreateSpotMarket(sdk.WrapSDKContext(ctx), &types.MsgCreateSpotMarket{
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

	notGov := sdk.AccAddress([]byte("notgov000000000000001")).String()
	_, err = ms.UpdateSpotMarketStatus(sdk.WrapSDKContext(ctx), &types.MsgUpdateSpotMarketStatus{
		Authority: notGov,
		MarketId:  create.MarketId,
		NewStatus: types.MarketStatus_MARKET_STATUS_PAUSED,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidAuthority)

	// Status must be unchanged.
	m, _ := k.GetMarket(ctx, create.MarketId)
	require.Equal(t, types.MarketStatus_MARKET_STATUS_ACTIVE, m.Status)
}
