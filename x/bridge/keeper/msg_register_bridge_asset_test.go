package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "ltrstchain/testutil/keeper"
	"ltrstchain/x/bridge/keeper"
	"ltrstchain/x/bridge/types"
)

// validAsset returns a canonical Noble USDC record. Tests that want
// to mutate a single field (symbol, source, ibc_denom) use this as
// their baseline and override just what they need.
func validAsset() types.BridgeAsset {
	return types.BridgeAsset{
		IbcDenom:      "ibc/498A0751C7A53C569BE0F84EE4E6FA3B9D6C3B2FBEDFBF37EA0F67F8E56BDBCEE",
		DisplaySymbol: "USDC",
		DisplayName:   "USD Coin (Noble)",
		Decimals:      6,
		Source:        types.BridgeSource_BRIDGE_SOURCE_NOBLE,
		SourceChainId: "noble-1",
		SourceDenom:   "uusdc",
		ChannelId:     "channel-0",
		Verified:      true,
	}
}

func TestMsgRegisterBridgeAsset_HappyPath(t *testing.T) {
	k, ctx := keepertest.BridgeKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	a := validAsset()
	_, err := ms.RegisterBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgRegisterBridgeAsset{
		Authority: k.GetAuthority(),
		Asset:     a,
	})
	require.NoError(t, err)

	stored, ok := k.GetBridgeAsset(ctx, a.IbcDenom)
	require.True(t, ok, "asset must be persisted")
	require.Equal(t, a.DisplaySymbol, stored.DisplaySymbol)
	require.Equal(t, a.Source, stored.Source)
	require.Equal(t, int64(ctx.BlockHeight()), stored.RegisteredHeight)
	require.Equal(t, int64(ctx.BlockHeight()), stored.UpdatedHeight)

	// Count must be exactly 1.
	require.Equal(t, uint32(1), k.CountBridgeAssets(ctx))
}

func TestMsgRegisterBridgeAsset_DuplicateRejected(t *testing.T) {
	k, ctx := keepertest.BridgeKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	a := validAsset()
	_, err := ms.RegisterBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgRegisterBridgeAsset{
		Authority: k.GetAuthority(),
		Asset:     a,
	})
	require.NoError(t, err)

	// Same ibc_denom, any other field change → still duplicate.
	dup := validAsset()
	dup.DisplayName = "USD Coin (different label)"
	_, err = ms.RegisterBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgRegisterBridgeAsset{
		Authority: k.GetAuthority(),
		Asset:     dup,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrAssetAlreadyExists)
}

func TestMsgRegisterBridgeAsset_NonGovRejectedWhenGovRequired(t *testing.T) {
	k, ctx := keepertest.BridgeKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// DefaultParams: RequireGovernanceRegistration = true.
	_, err := ms.RegisterBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgRegisterBridgeAsset{
		Authority: "cosmos1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxpvu8d6",
		Asset:     validAsset(),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidAuthority)
}

func TestMsgRegisterBridgeAsset_ExceedsMaxAssets(t *testing.T) {
	k, ctx := keepertest.BridgeKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Force a cap of exactly 1, so the second register trips it.
	require.NoError(t, k.SetParams(ctx, types.Params{
		RequireGovernanceRegistration: true,
		MaxAssets:                     1,
	}))

	a := validAsset()
	_, err := ms.RegisterBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgRegisterBridgeAsset{
		Authority: k.GetAuthority(),
		Asset:     a,
	})
	require.NoError(t, err)

	b := validAsset()
	b.IbcDenom = "ibc/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	_, err = ms.RegisterBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgRegisterBridgeAsset{
		Authority: k.GetAuthority(),
		Asset:     b,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrRegistryFull)
}

func TestMsgRegisterBridgeAsset_InvalidSource(t *testing.T) {
	k, ctx := keepertest.BridgeKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	a := validAsset()
	a.Source = types.BridgeSource_BRIDGE_SOURCE_UNSPECIFIED
	_, err := ms.RegisterBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgRegisterBridgeAsset{
		Authority: k.GetAuthority(),
		Asset:     a,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidSource)

	// An integer value outside the enum range is also rejected.
	a2 := validAsset()
	a2.Source = types.BridgeSource(999)
	_, err = ms.RegisterBridgeAsset(sdk.WrapSDKContext(ctx), &types.MsgRegisterBridgeAsset{
		Authority: k.GetAuthority(),
		Asset:     a2,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidSource)
}
