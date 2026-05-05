package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
    

	"ltrstchain/x/ltrstchain/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

        // the address capable of executing a MsgUpdateParams message. Typically, this
        // should be the x/gov module account.
        authority string

		// bankKeeper moves ultrst in/out of the shielded pool module account.
		// See keeper/shielded_pool.go for DepositToShieldedPool and
		// WithdrawFromShieldedPool. Optional — nil means shielded-pool value
		// flow is disabled (commitments still work, but no funds escrowed).
		bankKeeper types.BankKeeper
	}
)

func NewKeeper(
    cdc codec.BinaryCodec,
	storeService store.KVStoreService,
    logger log.Logger,
	authority string,
	bankKeeper types.BankKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		logger:       logger,
		bankKeeper:   bankKeeper,
	}
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}


