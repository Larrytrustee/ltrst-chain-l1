package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"ltrstchain/x/ltrstdex/types"
)

// CreateSpotMarket handles gov-gated creation of a new spot market.
// Behaviour depends on Params.GovOnlyMarketCreation:
//
//   - true  → authority MUST equal k.GetAuthority() (x/gov by default)
//   - false → any valid bech32 authority is accepted
//
// The message's InitialStatus is carried through unchanged.
func (k msgServer) CreateSpotMarket(goCtx context.Context, req *types.MsgCreateSpotMarket) (*types.MsgCreateSpotMarketResponse, error) {
	p := k.GetParams(goCtx)
	if p.GovOnlyMarketCreation && k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidAuthority,
			"gov-only market creation: expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	market := types.SpotMarket{
		Ticker:       req.Ticker,
		BaseDenom:    req.BaseDenom,
		QuoteDenom:   req.QuoteDenom,
		MinPriceTick: req.MinPriceTick,
		MinQuantity:  req.MinQuantity,
		MakerFeeBps:  req.MakerFeeBps,
		TakerFeeBps:  req.TakerFeeBps,
		Status:       req.InitialStatus,
	}

	marketID, err := k.CreateMarket(goCtx, market)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSpotMarketCreated,
		sdk.NewAttribute(types.AttrMarketID, strconv.FormatUint(marketID, 10)),
		sdk.NewAttribute("ticker", req.Ticker),
		sdk.NewAttribute("base_denom", req.BaseDenom),
		sdk.NewAttribute("quote_denom", req.QuoteDenom),
		sdk.NewAttribute(types.AttrStatus, req.InitialStatus.String()),
	))

	return &types.MsgCreateSpotMarketResponse{MarketId: marketID}, nil
}
