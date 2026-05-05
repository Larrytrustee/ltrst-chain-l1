package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "ltrstchain/testutil/keeper"
	"ltrstchain/x/bridge/keeper"
	"ltrstchain/x/bridge/types"
)

// registerAssetForUpdateTest is a shared setup helper that registers
// the canonical Noble USDC entry with the gov authority.
func registerAssetForUpdateTest(
	t testing.TB,
	k keeper.Keeper,
	ms types.MsgServer,
	ctx sdk.Context,
) types.BridgeAsset {
	t.Helper()
	a := validAsset()
	_, err := ms.RegisterBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgRegisterBridgeAsset{
		Authority: k.GetAuthority(),
		Asset:     a,
	})
	require.NoError(t, err)
	return a
}

func TestMsgUpdateBridgeAsset_HappyPath_PartialUpdate(t *testing.T) {
	k, ctx := keepertest.BridgeKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	original := registerAssetForUpdateTest(t, k, ms, ctx)

	// Mutate only the display_name — verifies all other fields stay put.
	_, err := ms.UpdateBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgUpdateBridgeAsset{
		Authority:   k.GetAuthority(),
		IbcDenom:    original.IbcDenom,
		DisplayName: "USD Coin (Noble, rebranded)",
	})
	require.NoError(t, err)

	stored, ok := k.GetBridgeAsset(ctx, original.IbcDenom)
	require.True(t, ok)
	require.Equal(t, "USD Coin (Noble, rebranded)", stored.DisplayName)

	// Every other field should be unchanged.
	require.Equal(t, original.DisplaySymbol, stored.DisplaySymbol)
	require.Equal(t, original.Decimals, stored.Decimals)
	require.Equal(t, original.Source, stored.Source)
	require.Equal(t, original.SourceChainId, stored.SourceChainId)
	require.Equal(t, original.SourceDenom, stored.SourceDenom)
	require.Equal(t, original.ChannelId, stored.ChannelId)
	require.Equal(t, original.Verified, stored.Verified)
}

func TestMsgUpdateBridgeAsset_AssetNotFound(t *testing.T) {
	k, ctx := keepertest.BridgeKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.UpdateBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgUpdateBridgeAsset{
		Authority:   k.GetAuthority(),
		IbcDenom:    "ibc/FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
		DisplayName: "Nope",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrAssetNotFound)
}

func TestMsgUpdateBridgeAsset_NonGovRejected(t *testing.T) {
	k, ctx := keepertest.BridgeKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	original := registerAssetForUpdateTest(t, k, ms, ctx)

	_, err := ms.UpdateBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgUpdateBridgeAsset{
		Authority:   "cosmos1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxpvu8d6",
		IbcDenom:    original.IbcDenom,
		DisplayName: "Unauthorised change",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidAuthority)

	// State must be untouched.
	stored, _ := k.GetBridgeAsset(ctx, original.IbcDenom)
	require.Equal(t, original.DisplayName, stored.DisplayName)
}
