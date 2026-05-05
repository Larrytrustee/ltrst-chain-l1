package ltrstchain

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"ltrstchain/x/ltrstchain/keeper"
	"ltrstchain/x/ltrstchain/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	// Replay any pre-existing privacy commitments (e.g. carried forward from
	// an upgrade). AddCommitment will validate protocol tags, reject any
	// duplicate nullifiers, and advance the rolling Merkle root in the same
	// order they appear in the genesis file — which is also the order
	// IterateCommitments produces during ExportGenesis.
	for i := range genState.PrivacyCommitments {
		c := genState.PrivacyCommitments[i]
		if err := k.AddCommitment(ctx, &c); err != nil {
			panic(fmt.Errorf("privacy init genesis: commitment %d: %w", i, err))
		}
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	// Snapshot every stored shielded commitment in commit-bytes order so an
	// exported genesis file round-trips back through InitGenesis deterministically.
	commitments := make([]types.ShieldedCommitment, 0)
	if err := k.IterateCommitments(ctx, func(c types.ShieldedCommitment) bool {
		commitments = append(commitments, c)
		return false
	}); err != nil {
		panic(fmt.Errorf("privacy export genesis: iterate commitments: %w", err))
	}
	genesis.PrivacyCommitments = commitments

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
