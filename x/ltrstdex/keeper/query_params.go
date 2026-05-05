package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ltrstchain/x/ltrstdex/types"
)

// Params returns the module's current Params.
func (k Keeper) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	return &types.QueryParamsResponse{Params: k.GetParams(goCtx)}, nil
}
