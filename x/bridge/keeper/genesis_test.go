package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "ltrstchain/testutil/keeper"
	"ltrstchain/x/bridge/keeper"
	"ltrstchain/x/bridge/types"
)

// TestExportGenesis_RejectsAssetOverflow is the regression check for
// the bug in genesis.go: ExportGenesis used to silently emit an
// asset slice longer than Params.MaxAssets, which later failed
// replay at InitGenesis with a confusing error in a different place.
// Now it panics at export time with a clear message.
func TestExportGenesis_RejectsAssetOverflow(t *testing.T) {
	k, ctx := keepertest.BridgeKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Start with unlimited so the registers go in.
	require.NoError(t, k.SetParams(ctx, types.Params{
		RequireGovernanceRegistration: true,
		MaxAssets:                     0,
	}))

	// Register two distinct assets.
	a1 := validAsset()
	_, err := ms.RegisterBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgRegisterBridgeAsset{
		Authority: k.GetAuthority(),
		Asset:     a1,
	})
	require.NoError(t, err)

	a2 := validAsset()
	a2.IbcDenom = "ibc/BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
	a2.DisplaySymbol = "USDCA"
	a2.Source = types.BridgeSource_BRIDGE_SOURCE_AXELAR
	a2.SourceChainId = "axelar-dojo-1"
	_, err = ms.RegisterBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgRegisterBridgeAsset{
		Authority: k.GetAuthority(),
		Asset:     a2,
	})
	require.NoError(t, err)

	// Now tighten the cap to 1 so the stored state exceeds it.
	require.NoError(t, k.SetParams(ctx, types.Params{
		RequireGovernanceRegistration: true,
		MaxAssets:                     1,
	}))

	// ExportGenesis must panic with a descriptive message rather than
	// silently produce an invalid genesis blob.
	require.PanicsWithValue(t,
		"bridge: exported 2 assets exceeds MaxAssets cap 1",
		func() { _ = k.ExportGenesis(ctx) },
	)
}

func TestExportGenesis_WithinCap_NoPanic(t *testing.T) {
	k, ctx := keepertest.BridgeKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	require.NoError(t, k.SetParams(ctx, types.Params{
		RequireGovernanceRegistration: true,
		MaxAssets:                     10,
	}))

	a := validAsset()
	_, err := ms.RegisterBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgRegisterBridgeAsset{
		Authority: k.GetAuthority(),
		Asset:     a,
	})
	require.NoError(t, err)

	gs := k.ExportGenesis(ctx)
	require.NotNil(t, gs)
	require.Len(t, gs.Assets, 1)
}
