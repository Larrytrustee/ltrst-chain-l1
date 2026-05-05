package keeper

import "ltrstchain/x/bridge/types"

// Keeper implements QueryServer. The per-RPC methods live in their
// own files (query_params.go, query_assets.go).
var _ types.QueryServer = Keeper{}
