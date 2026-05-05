package types

import (
	"context"

	
	sdk "github.com/cosmos/cosmos-sdk/types"
	
)



// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
    GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
    // Methods imported from account should be defined here
}

// BankKeeper defines the expected interface for the Bank module.
// The send-to-module / send-from-module methods are the primitives used by
// the shielded-pool value flow (deposit / withdraw) in keeper/shielded_pool.go.
type BankKeeper interface {
    SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
    SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
    SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
    GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}
