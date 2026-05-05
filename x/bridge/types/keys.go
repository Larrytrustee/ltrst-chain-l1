package types

const (
	// ModuleName defines the module name.
	ModuleName = "bridge"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey is the message route for x/bridge.
	RouterKey = ModuleName

	// QuerierRoute is the querier route.
	QuerierRoute = ModuleName
)

// KV store prefixes. Keep each prefix a distinct byte-ish string so
// ranges never overlap.
var (
	// ParamsKey stores the encoded Params.
	ParamsKey = []byte("p")

	// BridgeAssetKeyPrefix is the prefix under which every registered
	// BridgeAsset is stored. Full key: a/{ibc_denom} → BridgeAsset.
	// ibc_denom is a bounded string (ibc/<64 hex>) so direct use as
	// key is safe.
	BridgeAssetKeyPrefix = []byte("a/")
)

// BridgeAssetKey builds the full KV key for an asset record.
func BridgeAssetKey(ibcDenom string) []byte {
	out := make([]byte, 0, len(BridgeAssetKeyPrefix)+len(ibcDenom))
	out = append(out, BridgeAssetKeyPrefix...)
	out = append(out, []byte(ibcDenom)...)
	return out
}
