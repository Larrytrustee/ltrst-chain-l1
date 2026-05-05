package types

import (
	"bytes"
	"fmt"
	"math/big"

	"cosmossdk.io/math"
)

// EncodePrice turns a positive LegacyDec into the fixed-length
// big-endian byte representation used in orderbook keys. The
// representation is designed so that lexicographic byte comparison
// equals numeric comparison on the underlying decimal value.
//
// LegacyDec stores prices as a big.Int of the price * 10^18. We
// take the absolute value's BigInt bytes and left-pad to
// PriceEncodedLen bytes. All orderbook prices are strictly positive
// and must fit in PriceEncodedLen bytes, so sort order is preserved.
//
// Returns an error if p is nil, non-positive, or its underlying
// big.Int representation exceeds PriceEncodedLen bytes.
func EncodePrice(p math.LegacyDec) ([]byte, error) {
	if p.IsNil() {
		return nil, fmt.Errorf("encode price: nil LegacyDec")
	}
	if !p.IsPositive() {
		return nil, fmt.Errorf("encode price: price must be positive (got %s)", p.String())
	}
	raw := p.BigInt().Bytes()
	if len(raw) > PriceEncodedLen {
		return nil, fmt.Errorf(
			"encode price: raw bytes %d > %d (price too large)",
			len(raw), PriceEncodedLen,
		)
	}
	out := make([]byte, PriceEncodedLen)
	copy(out[PriceEncodedLen-len(raw):], raw)
	return out, nil
}

// MustEncodePrice panics on failure. For use after ValidateBasic has
// already ensured the price is well-formed.
func MustEncodePrice(p math.LegacyDec) []byte {
	b, err := EncodePrice(p)
	if err != nil {
		panic(err)
	}
	return b
}

// InvertPriceForBids takes an encoded price and returns its bit-wise
// complement. Used so that the buy side sorts DESC under a forward
// KV iterator.
func InvertPriceForBids(encoded []byte) []byte {
	out := make([]byte, len(encoded))
	for i, b := range encoded {
		out[i] = ^b
	}
	return out
}

// EncodeBidPrice is a convenience for the buy-side encoding used
// directly in OrderbookBidKey.
func EncodeBidPrice(p math.LegacyDec) ([]byte, error) {
	enc, err := EncodePrice(p)
	if err != nil {
		return nil, err
	}
	return InvertPriceForBids(enc), nil
}

// MustEncodeBidPrice is the panicking variant for post-validated prices.
func MustEncodeBidPrice(p math.LegacyDec) []byte {
	return InvertPriceForBids(MustEncodePrice(p))
}

// DecodePriceFromBidEncoding reverses EncodeBidPrice, returning the
// original positive LegacyDec price.
func DecodePriceFromBidEncoding(inverted []byte) (math.LegacyDec, error) {
	if len(inverted) != PriceEncodedLen {
		return math.LegacyDec{}, fmt.Errorf(
			"decode bid: expected %d bytes, got %d", PriceEncodedLen, len(inverted),
		)
	}
	raw := make([]byte, PriceEncodedLen)
	for i, b := range inverted {
		raw[i] = ^b
	}
	return decodeRawToDec(raw)
}

// DecodePriceFromAskEncoding reverses EncodePrice for sell-side keys.
func DecodePriceFromAskEncoding(raw []byte) (math.LegacyDec, error) {
	if len(raw) != PriceEncodedLen {
		return math.LegacyDec{}, fmt.Errorf(
			"decode ask: expected %d bytes, got %d", PriceEncodedLen, len(raw),
		)
	}
	return decodeRawToDec(raw)
}

func decodeRawToDec(raw []byte) (math.LegacyDec, error) {
	trimmed := bytes.TrimLeft(raw, "\x00")
	if len(trimmed) == 0 {
		return math.LegacyDec{}, fmt.Errorf("decode price: zero not allowed")
	}
	bi := new(big.Int).SetBytes(trimmed)
	return math.LegacyNewDecFromBigIntWithPrec(bi, math.LegacyPrecision), nil
}
