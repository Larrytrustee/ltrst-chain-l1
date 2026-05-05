package keeper

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"ltrstchain/x/ltrstdex/keeper"
	"ltrstchain/x/ltrstdex/types"
)

// FakeAccountKeeper implements types.AccountKeeper with deterministic
// module-address derivation and a trivial GetAccount/ModuleAccount
// that never errors. Sufficient for message-handler testing: the real
// x/ltrstdex code only reads GetModuleAddress (via x/bank) to address
// module-owned coin pools.
type FakeAccountKeeper struct {
	// moduleAccs pre-derived at construction time.
	moduleAccs map[string]sdk.AccAddress
}

// NewFakeAccountKeeper pre-resolves the standard x/ltrstdex +
// x/auth.FeeCollectorName module addresses so the x/bank fake can
// route coins cleanly.
func NewFakeAccountKeeper() *FakeAccountKeeper {
	return &FakeAccountKeeper{
		moduleAccs: map[string]sdk.AccAddress{
			types.ModuleName:             authtypes.NewModuleAddress(types.ModuleName),
			authtypes.FeeCollectorName:   authtypes.NewModuleAddress(authtypes.FeeCollectorName),
			govtypes.ModuleName:          authtypes.NewModuleAddress(govtypes.ModuleName),
		},
	}
}

func (f *FakeAccountKeeper) GetModuleAddress(name string) sdk.AccAddress {
	if a, ok := f.moduleAccs[name]; ok {
		return a
	}
	// Fall back to the real derivation so any module we didn't
	// pre-seed still resolves deterministically.
	a := authtypes.NewModuleAddress(name)
	f.moduleAccs[name] = a
	return a
}

func (f *FakeAccountKeeper) GetModuleAccount(_ context.Context, name string) sdk.ModuleAccountI {
	addr := f.GetModuleAddress(name)
	// authtypes.NewModuleAccount requires a BaseAccount and name.
	base := authtypes.NewBaseAccount(addr, nil, 0, 0)
	return authtypes.NewModuleAccount(base, name)
}

func (f *FakeAccountKeeper) GetAccount(_ context.Context, addr sdk.AccAddress) sdk.AccountI {
	// Return a vanilla BaseAccount — plenty for simulation / genesis
	// checks, which is the only caller.
	return authtypes.NewBaseAccount(addr, nil, 0, 0)
}

// Compile-time assertion.
var _ types.AccountKeeper = (*FakeAccountKeeper)(nil)

// FakeBankKeeper is a minimal in-memory implementation of
// types.BankKeeper. It tracks coins at bech32 addresses in a flat
// map[addr][denom]math.Int. Module accounts are addressed via the
// companion FakeAccountKeeper.
type FakeBankKeeper struct {
	ak       *FakeAccountKeeper
	balances map[string]map[string]math.Int
}

// NewFakeBankKeeper returns an empty in-memory bank.
func NewFakeBankKeeper(ak *FakeAccountKeeper) *FakeBankKeeper {
	return &FakeBankKeeper{
		ak:       ak,
		balances: map[string]map[string]math.Int{},
	}
}

// SetBalance is a test-only helper to seed an account with coins.
func (b *FakeBankKeeper) SetBalance(addr sdk.AccAddress, coins sdk.Coins) {
	key := addr.String()
	m, ok := b.balances[key]
	if !ok {
		m = map[string]math.Int{}
		b.balances[key] = m
	}
	for _, c := range coins {
		m[c.Denom] = c.Amount
	}
}

// AddBalance adds coins to the account's existing balance (creating
// the entry if missing). Useful in tests that want to top up.
func (b *FakeBankKeeper) AddBalance(addr sdk.AccAddress, coins sdk.Coins) {
	key := addr.String()
	m, ok := b.balances[key]
	if !ok {
		m = map[string]math.Int{}
		b.balances[key] = m
	}
	for _, c := range coins {
		cur, ok := m[c.Denom]
		if !ok {
			cur = math.ZeroInt()
		}
		m[c.Denom] = cur.Add(c.Amount)
	}
}

// SpendableCoins returns the full balance (no vesting semantics in the
// fake — spendable == total).
func (b *FakeBankKeeper) SpendableCoins(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	m, ok := b.balances[addr.String()]
	if !ok {
		return sdk.NewCoins()
	}
	out := sdk.NewCoins()
	for denom, amt := range m {
		if amt.IsPositive() {
			out = out.Add(sdk.NewCoin(denom, amt))
		}
	}
	return out
}

// GetBalance returns the amount of a specific denom held by addr.
func (b *FakeBankKeeper) GetBalance(_ context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	m, ok := b.balances[addr.String()]
	if !ok {
		return sdk.NewCoin(denom, math.ZeroInt())
	}
	amt, ok := m[denom]
	if !ok {
		return sdk.NewCoin(denom, math.ZeroInt())
	}
	return sdk.NewCoin(denom, amt)
}

// sub removes coins from an address, returning an error if insufficient.
func (b *FakeBankKeeper) sub(addr sdk.AccAddress, coins sdk.Coins) error {
	key := addr.String()
	m, ok := b.balances[key]
	if !ok {
		m = map[string]math.Int{}
		b.balances[key] = m
	}
	// Pre-check all denoms first so we either fully succeed or leave
	// state intact.
	for _, c := range coins {
		cur, ok := m[c.Denom]
		if !ok {
			cur = math.ZeroInt()
		}
		if cur.LT(c.Amount) {
			return fmt.Errorf("insufficient funds: %s has %s%s, need %s%s",
				addr.String(), cur.String(), c.Denom, c.Amount.String(), c.Denom)
		}
	}
	for _, c := range coins {
		cur := m[c.Denom]
		m[c.Denom] = cur.Sub(c.Amount)
	}
	return nil
}

// add deposits coins into an address.
func (b *FakeBankKeeper) add(addr sdk.AccAddress, coins sdk.Coins) {
	key := addr.String()
	m, ok := b.balances[key]
	if !ok {
		m = map[string]math.Int{}
		b.balances[key] = m
	}
	for _, c := range coins {
		cur, ok := m[c.Denom]
		if !ok {
			cur = math.ZeroInt()
		}
		m[c.Denom] = cur.Add(c.Amount)
	}
}

func (b *FakeBankKeeper) SendCoinsFromAccountToModule(_ context.Context, sender sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	if err := b.sub(sender, amt); err != nil {
		return err
	}
	b.add(b.ak.GetModuleAddress(recipientModule), amt)
	return nil
}

func (b *FakeBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipient sdk.AccAddress, amt sdk.Coins) error {
	sender := b.ak.GetModuleAddress(senderModule)
	if err := b.sub(sender, amt); err != nil {
		return err
	}
	b.add(recipient, amt)
	return nil
}

func (b *FakeBankKeeper) SendCoinsFromModuleToModule(_ context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	sender := b.ak.GetModuleAddress(senderModule)
	recipient := b.ak.GetModuleAddress(recipientModule)
	if err := b.sub(sender, amt); err != nil {
		return err
	}
	b.add(recipient, amt)
	return nil
}

// Compile-time assertion.
var _ types.BankKeeper = (*FakeBankKeeper)(nil)

// Silence unused-import warnings when the fakes don't need these
// helpers directly — they stay imported because the SDK expects the
// full expected-keeper surface.
var _ cryptotypes.PubKey = nil

// LtrstdexKeeper wires up a fresh in-memory store + fake keepers and
// returns the configured Keeper plus an sdk.Context for use in tests.
// Mirrors the pattern in LtrstchainKeeper above.
func LtrstdexKeeper(t testing.TB) (keeper.Keeper, *FakeAccountKeeper, *FakeBankKeeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	ak := NewFakeAccountKeeper()
	bk := NewFakeBankKeeper(ak)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		ak,
		bk,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Seed default params so PlaceOrder's AssertOwnerUnderMaxOpen works.
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))

	return k, ak, bk, ctx
}
