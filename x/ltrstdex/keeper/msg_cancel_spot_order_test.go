package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	// placeTestSetup / FakeBankKeeper helpers live in msg_place_spot_order_test.go
	// in this same package, so no direct keepertest import is needed here.

	"ltrstchain/x/ltrstdex/keeper"
	"ltrstchain/x/ltrstdex/types"
)

func TestMsgCancelSpotOrder_HappyPath(t *testing.T) {
	k, ms, ak, bk, ctx, marketID := placeTestSetup(t)

	owner := sdk.AccAddress([]byte("canceler0000000000001"))
	fundAcc(bk, owner, "ultrst", 100)

	placeResp, err := ms.PlaceSpotOrder(sdk.WrapSDKContext(ctx), &types.MsgPlaceSpotOrder{
		Owner:    owner.String(),
		MarketId: marketID,
		Side:     types.Side_SIDE_SELL,
		Type:     types.OrderType_ORDER_TYPE_LIMIT,
		Tif:      types.TimeInForce_TIF_GTC,
		Price:    math.LegacyMustNewDecFromStr("1.00"),
		Quantity: math.LegacyMustNewDecFromStr("10"),
	})
	require.NoError(t, err)

	// Confirm lock is held by module.
	modAddr := ak.GetModuleAddress(types.ModuleName)
	require.Equal(t, int64(10), bk.GetBalance(ctx, modAddr, "ultrst").Amount.Int64())
	require.Equal(t, int64(90), bk.GetBalance(ctx, owner, "ultrst").Amount.Int64())

	// Cancel.
	_, err = ms.CancelSpotOrder(sdk.WrapSDKContext(ctx), &types.MsgCancelSpotOrder{
		Authority: owner.String(),
		MarketId:  marketID,
		OrderId:   placeResp.OrderId,
	})
	require.NoError(t, err)

	// Escrow must have come back.
	require.Equal(t, int64(0), bk.GetBalance(ctx, modAddr, "ultrst").Amount.Int64())
	require.Equal(t, int64(100), bk.GetBalance(ctx, owner, "ultrst").Amount.Int64())

	// Order must be gone from storage and from the orderbook index.
	_, ok := k.GetOrder(ctx, marketID, placeResp.OrderId)
	require.False(t, ok, "order record should be deleted")
	_, foundBest := k.BestAsk(ctx, marketID)
	require.False(t, foundBest, "orderbook ask side should be empty")
}

func TestMsgCancelSpotOrder_NonOwnerRejected(t *testing.T) {
	_, ms, _, bk, ctx, marketID := placeTestSetup(t)

	owner := sdk.AccAddress([]byte("owner00000000000000c1"))
	fundAcc(bk, owner, "ultrst", 100)

	placeResp, err := ms.PlaceSpotOrder(sdk.WrapSDKContext(ctx), &types.MsgPlaceSpotOrder{
		Owner:    owner.String(),
		MarketId: marketID,
		Side:     types.Side_SIDE_SELL,
		Type:     types.OrderType_ORDER_TYPE_LIMIT,
		Tif:      types.TimeInForce_TIF_GTC,
		Price:    math.LegacyMustNewDecFromStr("1.00"),
		Quantity: math.LegacyMustNewDecFromStr("10"),
	})
	require.NoError(t, err)

	// Another valid bech32 address tries to cancel.
	intruder := sdk.AccAddress([]byte("intruder000000000000c")).String()
	_, err = ms.CancelSpotOrder(sdk.WrapSDKContext(ctx), &types.MsgCancelSpotOrder{
		Authority: intruder,
		MarketId:  marketID,
		OrderId:   placeResp.OrderId,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidAuthority)
}

// Ensure the happy-path uses the Keeper correctly — also included so
// the imports aren't dead-code if the happy-path test is skipped.
var _ = keeper.NewMsgServerImpl

func TestMsgCancelSpotOrder_NotFound(t *testing.T) {
	_, ms, _, _, ctx, marketID := placeTestSetup(t)

	owner := sdk.AccAddress([]byte("owner00000000000000c2")).String()
	_, err := ms.CancelSpotOrder(sdk.WrapSDKContext(ctx), &types.MsgCancelSpotOrder{
		Authority: owner,
		MarketId:  marketID,
		OrderId:   "deadbeefdeadbeef",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrOrderNotFound)
}
