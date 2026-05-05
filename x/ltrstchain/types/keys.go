package types

const (
	// ModuleName defines the module name
	ModuleName = "ltrstchain"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_ltrstchain"

	// ShieldedPoolName is the module account that custodies funds currently
	// locked behind shielded commitments. Value enters via DepositToShieldedPool
	// and can only exit via WithdrawFromShieldedPool, which requires revealing
	// a prior-recorded commitment opening + unused nullifier. Holds ultrst.
	ShieldedPoolName = "ltrstchain_shielded_pool"
)

var (
	ParamsKey = []byte("p_ltrstchain")
)



func KeyPrefix(p string) []byte {
    return []byte(p)
}
