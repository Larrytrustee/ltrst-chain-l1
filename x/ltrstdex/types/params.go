package types

import (
	"fmt"
)

// DefaultMaxOpenOrdersPerAccount caps live-order count per owner.
const DefaultMaxOpenOrdersPerAccount uint32 = 200

// DefaultFillRetentionBlocks is ~ two weeks of 2-second blocks.
const DefaultFillRetentionBlocks uint32 = 604800

// DefaultGovOnlyMarketCreation — Phase 1 locks market creation to gov.
const DefaultGovOnlyMarketCreation bool = true

// NewParams creates a new Params instance.
func NewParams(maxOpenOrders uint32, govOnly bool, fillRetentionBlocks uint32) Params {
	return Params{
		MaxOpenOrdersPerAccount: maxOpenOrders,
		GovOnlyMarketCreation:   govOnly,
		FillRetentionBlocks:     fillRetentionBlocks,
	}
}

// DefaultParams returns the Phase-1 default parameter set.
func DefaultParams() Params {
	return NewParams(
		DefaultMaxOpenOrdersPerAccount,
		DefaultGovOnlyMarketCreation,
		DefaultFillRetentionBlocks,
	)
}

// Validate checks the params are internally consistent.
func (p Params) Validate() error {
	if p.MaxOpenOrdersPerAccount == 0 {
		return fmt.Errorf("max_open_orders_per_account must be > 0")
	}
	if p.MaxOpenOrdersPerAccount > 10_000 {
		return fmt.Errorf("max_open_orders_per_account must be <= 10000 (got %d)", p.MaxOpenOrdersPerAccount)
	}
	return nil
}
