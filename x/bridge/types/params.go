package types

// DefaultParams returns params suitable for a fresh mainnet: require
// governance registration, unlimited asset count. Devnets typically
// flip require_governance_registration off in their genesis.
func DefaultParams() Params {
	return Params{
		RequireGovernanceRegistration: true,
		MaxAssets:                     0, // unlimited
	}
}

// Validate enforces basic sanity — no fields have inter-field
// dependencies yet, but the hook is here for future additions.
func (p Params) Validate() error { return nil }
