package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"ltrstchain/x/bridge/types"
)

// UpdateParams is authority-only; the authority bech32 is set at
// module init.
func (k msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if req.Authority != k.GetAuthority() {
		return nil, errorsmod.Wrapf(types.ErrInvalidAuthority,
			"expected %s, got %s", k.GetAuthority(), req.Authority)
	}
	if err := k.SetParams(goCtx, req.Params); err != nil {
		return nil, err
	}
	return &types.MsgUpdateParamsResponse{}, nil
}
