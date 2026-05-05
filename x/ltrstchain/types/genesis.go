package types

import (
	"errors"
	// this line is used by starport scaffolding # genesis/types/import
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params:             DefaultParams(),
		PrivacyCommitments: []ShieldedCommitment{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate

	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// Validate every pre-loaded privacy commitment statically and reject any
	// duplicate nullifiers so genesis can never start in a double-spent state.
	seenNullifiers := make(map[string]struct{}, len(gs.PrivacyCommitments))
	for i := range gs.PrivacyCommitments {
		c := gs.PrivacyCommitments[i]
		if err := ValidateShieldedCommitment(&c); err != nil {
			return errors.New("privacy genesis: " + err.Error())
		}
		key := string(c.Nullifier)
		if _, dup := seenNullifiers[key]; dup {
			return errors.New("privacy genesis: duplicate nullifier in commitments list")
		}
		seenNullifiers[key] = struct{}{}
	}
	return nil
}
