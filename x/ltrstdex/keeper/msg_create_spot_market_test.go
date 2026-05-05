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

// validCreateSpotMarketMsg returns a well-formed request with the gov
// authority as signer, suitable as a base for happy-path and
// error-path tests alike.
func validCreateSpotMarketMsg(authority string) *types.MsgCreateSpotMarket {
	return &types.MsgCreateSpotMarket{
		Authority:     authority,
		Ticker:        "LTRST/USDC",
		BaseDenom:     "ultrst",
		QuoteDenom:    "uusdc",
		MinPriceTick:  math.LegacyMustNewDecFromStr("0.01"),
		MinQuantity:   math.LegacyMustNewDecFromStr("1"),
		MakerFeeBps:   0,
		TakerFeeBps:   30,
		InitialStatus: types.MarketStatus_MARKET_STATUS_ACTIVE,
	}
}

func TestMsgCreateSpotMarket_HappyPath(t *testing.T) {
	k, _, _, ctx := keepertest.LtrstdexKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	req := validCreateSpotMarketMsg(k.GetAuthority())
	resp, err := ms.CreateSpotMarket(sdk.WrapSDKContext(ctx), req)
	require.NoError(t, err)
	require.Equal(t, uint64(1), resp.MarketId, "first market should be id=1")

	// Market must be persisted with the submitted spec.
	m, ok := k.GetMarket(ctx, resp.MarketId)
	require.True(t, ok)
	require.Equal(t, "LTRST/USDC", m.Ticker)
	require.Equal(t, "ultrst", m.BaseDenom)
	require.Equal(t, "uusdc", m.QuoteDenom)
	require.Equal(t, types.MarketStatus_MARKET_STATUS_ACTIVE, m.Status)

	// Next market id counter must have advanced to 2 (ready for the
	// second market creation, which exercises the "orderbook head"
	// by being independent of #1).
	require.Equal(t, uint64(2), k.NextMarketID(ctx))
}

func TestMsgCreateSpotMarket_NonGovRejectedWhenGovOnly(t *testing.T) {
	k, _, _, ctx := keepertest.LtrstdexKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// DefaultParams().GovOnlyMarketCreation == true.
	notGov := sdk.AccAddress([]byte("randomsignernotgov__")).String()
	req := validCreateSpotMarketMsg(notGov)
	_, err := ms.CreateSpotMarket(sdk.WrapSDKContext(ctx), req)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidAuthority)
}

func TestMsgCreateSpotMarket_DuplicateTickerRejected(t *testing.T) {
	k, _, _, ctx := keepertest.LtrstdexKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// First create succeeds.
	_, err := ms.CreateSpotMarket(sdk.WrapSDKContext(ctx), validCreateSpotMarketMsg(k.GetAuthority()))
	require.NoError(t, err)

	// Second create with same ticker but different denom pair must
	// still trip the ticker uniqueness check.
	dup := validCreateSpotMarketMsg(k.GetAuthority())
	dup.BaseDenom = "ubtc"
	dup.QuoteDenom = "ueth"
	_, err = ms.CreateSpotMarket(sdk.WrapSDKContext(ctx), dup)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrDuplicateTicker)
}

func TestMsgCreateSpotMarket_MinPriceTickNonPositiveRejected(t *testing.T) {
	k, _, _, ctx := keepertest.LtrstdexKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	_, _ = ms, ctx // this case only exercises ValidateBasic; keeper/msg-server
	// are constructed to assert the factory wiring compiles against the
	// full dependency graph, not because they're invoked here.

	req := validCreateSpotMarketMsg(k.GetAuthority())
	req.MinPriceTick = math.LegacyZeroDec()
	require.Error(t, req.ValidateBasic(), "zero tick should fail basic validation")

	// Negative tick is also rejected by ValidateBasic.
	req.MinPriceTick = math.LegacyMustNewDecFromStr("-0.01")
	require.Error(t, req.ValidateBasic())
}

func TestMsgCreateSpotMarket_MinQuantityNonPositiveRejected(t *testing.T) {
	k, _, _, ctx := keepertest.LtrstdexKeeper(t)
	_ = ctx
	_ = k

	req := validCreateSpotMarketMsg("cosmos1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxpvu8d6")
	req.MinQuantity = math.LegacyZeroDec()
	require.Error(t, req.ValidateBasic(), "zero quantity should fail basic validation")

	req.MinQuantity = math.LegacyMustNewDecFromStr("-1")
	require.Error(t, req.ValidateBasic())
}
