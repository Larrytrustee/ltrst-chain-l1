package keeper

import (
	"ltrstchain/x/ltrstdex/types"
)

// Keeper itself serves the x/ltrstdex gRPC queries. Per-query
// implementations live in query_*.go files in this package.
var _ types.QueryServer = Keeper{}
