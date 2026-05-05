package types

import "fmt"

// DefaultGenesis returns a GenesisState with default params and no
// pre-populated assets. Mainnet genesis overrides `assets` with the
// canonical Noble / Axelar / Wormhole USDC entries.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Assets: []BridgeAsset{},
	}
}

// Validate checks cross-field invariants: unique ibc_denom, every
// asset individually valid, and the params themselves valid.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("params: %w", err)
	}
	seen := map[string]struct{}{}
	for i := range gs.Assets {
		a := gs.Assets[i]
		if err := ValidateBridgeAsset(a); err != nil {
			return fmt.Errorf("asset[%d]: %w", i, err)
		}
		if _, ok := seen[a.IbcDenom]; ok {
			return fmt.Errorf("asset[%d]: duplicate ibc_denom %q", i, a.IbcDenom)
		}
		seen[a.IbcDenom] = struct{}{}
	}
	if gs.Params.MaxAssets > 0 && uint32(len(gs.Assets)) > gs.Params.MaxAssets {
		return fmt.Errorf("genesis has %d assets, exceeds max_assets %d",
			len(gs.Assets), gs.Params.MaxAssets)
	}
	return nil
}
