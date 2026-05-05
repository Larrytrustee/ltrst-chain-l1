package keeper

import (
	"context"

	"ltrstchain/x/bridge/types"
)

// Params returns the module parameters.
func (k Keeper) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	return &types.QueryParamsResponse{Params: k.GetParams(goCtx)}, nil
}
