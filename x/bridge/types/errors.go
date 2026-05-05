package types

import (
	errorsmod "cosmossdk.io/errors"
)

// Sentinel error definitions. All messages and keepers wrap one of
// these so consumers can switch on error codes.
var (
	ErrInvalidAuthority    = errorsmod.Register(ModuleName, 2, "invalid authority")
	ErrInvalidIBCDenom     = errorsmod.Register(ModuleName, 3, "invalid ibc denom")
	ErrInvalidDisplaySym   = errorsmod.Register(ModuleName, 4, "invalid display symbol")
	ErrInvalidDisplayName  = errorsmod.Register(ModuleName, 5, "invalid display name")
	ErrInvalidChannel      = errorsmod.Register(ModuleName, 6, "invalid channel id")
	ErrInvalidSource       = errorsmod.Register(ModuleName, 7, "invalid bridge source")
	ErrInvalidSourceChain  = errorsmod.Register(ModuleName, 8, "invalid source chain id")
	ErrInvalidSourceDenom  = errorsmod.Register(ModuleName, 9, "invalid source denom")
	ErrAssetAlreadyExists  = errorsmod.Register(ModuleName, 10, "bridge asset already registered")
	ErrAssetNotFound       = errorsmod.Register(ModuleName, 11, "bridge asset not found")
	ErrRegistryFull        = errorsmod.Register(ModuleName, 12, "bridge registry at max capacity")
	ErrInvalidDecimals     = errorsmod.Register(ModuleName, 13, "invalid decimals")
)
