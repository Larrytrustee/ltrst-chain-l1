package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"ltrstchain/x/ltrstdex/types"
)

// Keeper manages x/ltrstdex state — spot markets, orders, the
// orderbook index, and fill history — and settles fills through
// x/bank.
type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	logger       log.Logger

	// authority is the bech32 address allowed to execute gov-gated
	// messages (UpdateParams, CreateSpotMarket, UpdateSpotMarketStatus).
	// Defaults to the x/gov module account.
	authority string

	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
}

// NewKeeper constructs a new ltrstdex Keeper. Panics if authority is
// not a valid bech32 address.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("ltrstdex: invalid authority address %q: %s", authority, err))
	}
	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		logger:        logger,
		authority:     authority,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
	}
}

// GetAuthority returns the module's governance authority address.
func (k Keeper) GetAuthority() string { return k.authority }

// Logger returns a module-scoped logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
