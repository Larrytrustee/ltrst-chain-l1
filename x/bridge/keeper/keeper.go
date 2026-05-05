package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"

	"ltrstchain/x/bridge/types"
)

// Keeper is the x/bridge state gateway. It owns the module's store
// service and serves as the implementation behind both MsgServer and
// QueryServer.
type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	logger       log.Logger

	// authority is the bech32 address (string form) of the only signer
	// allowed to call authority-gated messages.
	authority string
}

// NewKeeper constructs the keeper. The caller is depinject (see
// module/module.go ProvideModule).
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		logger:       logger.With("module", "x/"+types.ModuleName),
		authority:    authority,
	}
}

// GetAuthority returns the configured authority bech32 address.
func (k Keeper) GetAuthority() string { return k.authority }

// Logger returns the scoped logger for the module.
func (k Keeper) Logger() log.Logger { return k.logger }

// kvStore opens the module's KV store. We go through the
// runtime.KVStoreAdapter shim so the same Keeper works whether the
// chain is running on the runtime/v2 store-service or the legacy
// store wiring.
func (k Keeper) kvStore(ctx context.Context) storetypes.KVStore {
	return runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
}

// -------------------------------------------------------------------
// Params
// -------------------------------------------------------------------

// GetParams reads the stored Params; returns DefaultParams() if no
// record has been written yet (should only happen before genesis).
func (k Keeper) GetParams(ctx context.Context) types.Params {
	kv := k.kvStore(ctx)
	bz := kv.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var p types.Params
	k.cdc.MustUnmarshal(bz, &p)
	return p
}

// SetParams writes Params atomically.
func (k Keeper) SetParams(ctx context.Context, p types.Params) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	bz, err := k.cdc.Marshal(&p)
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}
	k.kvStore(ctx).Set(types.ParamsKey, bz)
	return nil
}
