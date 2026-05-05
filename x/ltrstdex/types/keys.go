package types

import (
	"encoding/binary"
)

const (
	// ModuleName defines the module name.
	ModuleName = "ltrstdex"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey is the message route for x/ltrstdex.
	RouterKey = ModuleName

	// QuerierRoute is the querier route.
	QuerierRoute = ModuleName
)

// KV store prefixes. Keep each prefix a distinct byte-ish string so
// ranges never overlap.
var (
	// ParamsKey stores the encoded Params.
	ParamsKey = []byte("p")

	// NextMarketIDKey holds the uint64 big-endian encoded next market
	// id to assign.
	NextMarketIDKey = []byte("m/next")

	// MarketKeyPrefix is the prefix under which SpotMarkets live.
	// Full key: m/{marketID:u64be} → SpotMarket
	MarketKeyPrefix = []byte("m/")

	// MarketByTickerKeyPrefix maps ticker strings to market ids.
	// Full key: m/by_ticker/{ticker} → marketID bytes (u64be)
	MarketByTickerKeyPrefix = []byte("m/by_ticker/")

	// NextOrderSeqKeyPrefix stores the per-market next order seq.
	// Full key: o/next/{marketID:u64be} → u64be seq
	NextOrderSeqKeyPrefix = []byte("o/next/")

	// OrderKeyPrefix stores the authoritative SpotOrder record.
	// Full key: o/{marketID:u64be}/{orderID} → SpotOrder
	OrderKeyPrefix = []byte("o/")

	// OrderbookBidKeyPrefix indexes buy-side orders by (negPrice, seq, orderID).
	// Full key: ob/{marketID:u64be}/b/{negPrice:20}/{seq:u64be}/{orderID}
	// Value:    orderID bytes (redundant; key carries all info)
	OrderbookBidKeyPrefix = []byte("ob/")
	OrderbookBidSideByte  = byte('b')

	// OrderbookAskKeyPrefix indexes sell-side orders by (price, seq, orderID).
	// Full key: ob/{marketID:u64be}/a/{price:20}/{seq:u64be}/{orderID}
	OrderbookAskKeyPrefix = []byte("ob/")
	OrderbookAskSideByte  = byte('a')

	// UserOrderKeyPrefix indexes live orders per owner for cancel-all / lookup.
	// Full key: u/{owner:bytes}/{orderID} → marketID (u64be)
	UserOrderKeyPrefix = []byte("u/")

	// FillKeyPrefix stores fill history per market.
	// Full key: f/{marketID:u64be}/{blockHeight:u64be}/{fillSeq:u64be}
	// Value:    SpotFill
	FillKeyPrefix = []byte("f/")

	// NextFillSeqKey holds the global monotonic fill seq (uint64 big-endian).
	NextFillSeqKey = []byte("f/next")
)

// PriceEncodedLen is the fixed byte-length of a LegacyDec encoded for
// orderbook keys. LegacyDec has 18 decimal places and integer part up
// to ~38 digits; we pad the big-endian absolute value to this many
// bytes so lexicographic order on bytes equals numeric order on
// positive decimals. 20 bytes = room for the full 10^57 range.
const PriceEncodedLen = 20

// Uint64Bytes encodes a uint64 in big-endian form for sorted keys.
func Uint64Bytes(x uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, x)
	return buf
}

// Uint64FromBytes reads 8 big-endian bytes back into uint64.
// Caller must ensure len(b) == 8.
func Uint64FromBytes(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

// MarketKey returns the KV key for a SpotMarket record.
func MarketKey(marketID uint64) []byte {
	return append(append([]byte{}, MarketKeyPrefix...), Uint64Bytes(marketID)...)
}

// MarketByTickerKey returns the key mapping a ticker to its market id.
func MarketByTickerKey(ticker string) []byte {
	return append(append([]byte{}, MarketByTickerKeyPrefix...), []byte(ticker)...)
}

// NextOrderSeqKey returns the per-market next-seq key.
func NextOrderSeqKey(marketID uint64) []byte {
	return append(append([]byte{}, NextOrderSeqKeyPrefix...), Uint64Bytes(marketID)...)
}

// OrderKey returns the authoritative SpotOrder record key.
func OrderKey(marketID uint64, orderID string) []byte {
	out := append([]byte{}, OrderKeyPrefix...)
	out = append(out, Uint64Bytes(marketID)...)
	out = append(out, '/')
	out = append(out, []byte(orderID)...)
	return out
}

// OrderKeyForMarketPrefix returns the prefix that iterates all orders
// of a given market.
func OrderKeyForMarketPrefix(marketID uint64) []byte {
	return append(append([]byte{}, OrderKeyPrefix...), Uint64Bytes(marketID)...)
}

// OrderbookBidKey returns the key for a buy-side orderbook index
// entry. negEncodedPrice must already be the 20-byte complement of
// the price so forward iteration yields descending prices.
func OrderbookBidKey(marketID uint64, negEncodedPrice []byte, seq uint64, orderID string) []byte {
	out := append([]byte{}, OrderbookBidKeyPrefix...)
	out = append(out, Uint64Bytes(marketID)...)
	out = append(out, '/', OrderbookBidSideByte, '/')
	out = append(out, negEncodedPrice...)
	out = append(out, '/')
	out = append(out, Uint64Bytes(seq)...)
	out = append(out, '/')
	out = append(out, []byte(orderID)...)
	return out
}

// OrderbookAskKey returns the key for a sell-side orderbook index
// entry. encodedPrice is the 20-byte big-endian pad of the price.
func OrderbookAskKey(marketID uint64, encodedPrice []byte, seq uint64, orderID string) []byte {
	out := append([]byte{}, OrderbookAskKeyPrefix...)
	out = append(out, Uint64Bytes(marketID)...)
	out = append(out, '/', OrderbookAskSideByte, '/')
	out = append(out, encodedPrice...)
	out = append(out, '/')
	out = append(out, Uint64Bytes(seq)...)
	out = append(out, '/')
	out = append(out, []byte(orderID)...)
	return out
}

// OrderbookBidMarketPrefix is the iterator prefix for walking all
// buy-side orders of a market (highest price first).
func OrderbookBidMarketPrefix(marketID uint64) []byte {
	out := append([]byte{}, OrderbookBidKeyPrefix...)
	out = append(out, Uint64Bytes(marketID)...)
	out = append(out, '/', OrderbookBidSideByte, '/')
	return out
}

// OrderbookAskMarketPrefix is the iterator prefix for walking all
// sell-side orders of a market (lowest price first).
func OrderbookAskMarketPrefix(marketID uint64) []byte {
	out := append([]byte{}, OrderbookAskKeyPrefix...)
	out = append(out, Uint64Bytes(marketID)...)
	out = append(out, '/', OrderbookAskSideByte, '/')
	return out
}

// UserOrderKey returns the per-owner order index key.
func UserOrderKey(owner []byte, orderID string) []byte {
	out := append([]byte{}, UserOrderKeyPrefix...)
	out = append(out, byte(len(owner)))
	out = append(out, owner...)
	out = append(out, '/')
	out = append(out, []byte(orderID)...)
	return out
}

// UserOrderOwnerPrefix is the iterator prefix for all orders of an
// owner.
func UserOrderOwnerPrefix(owner []byte) []byte {
	out := append([]byte{}, UserOrderKeyPrefix...)
	out = append(out, byte(len(owner)))
	out = append(out, owner...)
	out = append(out, '/')
	return out
}

// FillKey returns the fill-history key for a single fill event.
func FillKey(marketID uint64, blockHeight uint64, fillSeq uint64) []byte {
	out := append([]byte{}, FillKeyPrefix...)
	out = append(out, Uint64Bytes(marketID)...)
	out = append(out, '/')
	out = append(out, Uint64Bytes(blockHeight)...)
	out = append(out, '/')
	out = append(out, Uint64Bytes(fillSeq)...)
	return out
}

// FillMarketPrefix is the iterator prefix for all fills of a market.
func FillMarketPrefix(marketID uint64) []byte {
	return append(append([]byte{}, FillKeyPrefix...), Uint64Bytes(marketID)...)
}
