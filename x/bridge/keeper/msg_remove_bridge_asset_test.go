package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "ltrstchain/testutil/keeper"
	"ltrstchain/x/bridge/keeper"
	"ltrstchain/x/bridge/types"
)

func TestMsgRemoveBridgeAsset_HappyPath(t *testing.T) {
	k, ctx := keepertest.BridgeKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	a := validAsset()
	_, err := ms.RegisterBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgRegisterBridgeAsset{
		Authority: k.GetAuthority(),
		Asset:     a,
	})
	require.NoError(t, err)

	_, err = ms.RemoveBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgRemoveBridgeAsset{
		Authority: k.GetAuthority(),
		IbcDenom:  a.IbcDenom,
	})
	require.NoError(t, err)

	// The query surface must now report "not found".
	_, ok := k.GetBridgeAsset(ctx, a.IbcDenom)
	require.False(t, ok, "asset must be deleted")
	require.False(t, k.HasBridgeAsset(ctx, a.IbcDenom))
	require.Equal(t, uint32(0), k.CountBridgeAssets(ctx))
}

func TestMsgRemoveBridgeAsset_AssetNotFound(t *testing.T) {
	k, ctx := keepertest.BridgeKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.RemoveBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgRemoveBridgeAsset{
		Authority: k.GetAuthority(),
		IbcDenom:  "ibc/NEVERWASREGISTEREDNEVERWASREGISTEREDNEVERWASREGISTEREDNEVERWASR",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrAssetNotFound)
}
