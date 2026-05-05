package ltrstchain_test

import (
	"testing"

	keepertest "ltrstchain/testutil/keeper"
	"ltrstchain/testutil/nullify"
	ltrstchain "ltrstchain/x/ltrstchain/module"
	"ltrstchain/x/ltrstchain/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:	types.DefaultParams(),
		
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.LtrstchainKeeper(t)
	ltrstchain.InitGenesis(ctx, k, genesisState)
	got := ltrstchain.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	

	// this line is used by starport scaffolding # genesis/test/assert
}
