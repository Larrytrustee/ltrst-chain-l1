package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
)

// ValidateBridgeAsset enforces the shape-level invariants on a single
// BridgeAsset record. It is called from MsgRegisterBridgeAsset,
// MsgUpdateBridgeAsset, and GenesisState.Validate.
func ValidateBridgeAsset(a BridgeAsset) error {
	if a.IbcDenom == "" {
		return errorsmod.Wrap(ErrInvalidIBCDenom, "empty")
	}
	// IBC transfer always prefixes ibc/ and the hash is 64 hex chars,
	// for a total length of exactly 68. We accept anything starting
	// with "ibc/" of length ≤ 128 to stay permissive for future
	// encoding changes while still ruling out obvious garbage.
	if !strings.HasPrefix(a.IbcDenom, "ibc/") {
		return errorsmod.Wrapf(ErrInvalidIBCDenom, "must start with ibc/: %q", a.IbcDenom)
	}
	if len(a.IbcDenom) > 128 {
		return errorsmod.Wrapf(ErrInvalidIBCDenom, "length %d exceeds 128", len(a.IbcDenom))
	}
	if a.DisplaySymbol == "" || len(a.DisplaySymbol) > 32 {
		return errorsmod.Wrapf(ErrInvalidDisplaySym, "symbol %q", a.DisplaySymbol)
	}
	if a.DisplayName == "" || len(a.DisplayName) > 64 {
		return errorsmod.Wrapf(ErrInvalidDisplayName, "name %q", a.DisplayName)
	}
	if a.Decimals > 36 {
		return errorsmod.Wrapf(ErrInvalidDecimals, "%d > 36", a.Decimals)
	}
	if a.Source == BridgeSource_BRIDGE_SOURCE_UNSPECIFIED {
		return errorsmod.Wrap(ErrInvalidSource, "must be set")
	}
	if _, ok := BridgeSource_name[int32(a.Source)]; !ok {
		return errorsmod.Wrapf(ErrInvalidSource, "unknown enum value %d", a.Source)
	}
	if a.SourceChainId == "" || len(a.SourceChainId) > 64 {
		return errorsmod.Wrapf(ErrInvalidSourceChain, "chain %q", a.SourceChainId)
	}
	if a.SourceDenom == "" || len(a.SourceDenom) > 128 {
		return errorsmod.Wrapf(ErrInvalidSourceDenom, "denom %q", a.SourceDenom)
	}
	// channel_id is optional — some registry entries may point to
	// multi-channel bridges where canonical channel drifts. But if
	// set, it must look like "channel-N".
	if a.ChannelId != "" {
		if !strings.HasPrefix(a.ChannelId, "channel-") {
			return errorsmod.Wrapf(ErrInvalidChannel, "must start with channel-: %q", a.ChannelId)
		}
		if len(a.ChannelId) > 32 {
			return errorsmod.Wrapf(ErrInvalidChannel, "length %d exceeds 32", len(a.ChannelId))
		}
	}
	return nil
}
