package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"ltrstchain/x/ltrstdex/types"
)

// UpdateParams handles gov-gated updates to x/ltrstdex Params.
// Authority must match the module authority (defaults to x/gov).
func (k msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidAuthority,
			"invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}
	if err := k.SetParams(goCtx, req.Params); err != nil {
		return nil, err
	}
	return &types.MsgUpdateParamsResponse{}, nil
}
