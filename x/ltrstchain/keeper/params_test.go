package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

    keepertest "ltrstchain/testutil/keeper"
    "ltrstchain/x/ltrstchain/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
