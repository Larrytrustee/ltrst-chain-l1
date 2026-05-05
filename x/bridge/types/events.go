package types

// Event types and attribute keys emitted by the x/bridge keeper.
// Strings are stable public API; bump ConsensusVersion if they
// change.
const (
	EventTypeBridgeAssetRegistered = "bridge_asset_registered"
	EventTypeBridgeAssetUpdated    = "bridge_asset_updated"
	EventTypeBridgeAssetRemoved    = "bridge_asset_removed"

	AttrIBCDenom    = "ibc_denom"
	AttrSymbol      = "symbol"
	AttrSource      = "source"
	AttrSourceChain = "source_chain"
	AttrVerified    = "verified"
)
