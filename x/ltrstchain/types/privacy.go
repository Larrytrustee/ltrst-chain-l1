package types

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// LTRST Chain x/privacy — shielded pool primitives.
//
// The LT Shield Wallet produces Pedersen-style commitments and per-spend
// nullifiers for each transaction (see lt_wallet_flutter/WALLET_PRIVACY.md).
// This module stores them on-chain so the network can:
//
//   1. Record every commitment in a growing Merkle set (the "shielded pool").
//   2. Reject any transaction whose nullifier has already been spent.
//   3. Accept selective-disclosure openings for fields the sender chooses to
//      reveal (amount, sender, receiver) and verify they match the binding.
//
// Commitments and nullifiers are both 32-byte SHA-256 outputs, hex-encoded on
// the wire. Openings and field salts travel only when the sender opts in.
//
// The canonical wire type is ShieldedCommitment, generated from
// proto/ltrstchain/ltrstchain/privacy.proto. This file only holds the
// constants, keys, hashing helpers, and validation logic — the data shape
// itself lives in privacy.pb.go so protobuf stays authoritative.

const (
	// CommitmentPrefix is the KV-store prefix for stored commitments.
	// Key layout: CommitmentPrefix || 32-byte commitment bytes -> ShieldedCommitment
	CommitmentPrefix = "pc/"

	// NullifierPrefix is the KV-store prefix for spent nullifiers.
	// Key layout: NullifierPrefix || 32-byte nullifier bytes -> []byte{1}
	NullifierPrefix = "pn/"

	// MerkleRootPrefix stores the latest rolling Merkle accumulator root.
	// Key layout: MerkleRootPrefix -> 32-byte root
	MerkleRootPrefix = "pr/"

	// CommitmentLength is the required length (in bytes) of a commitment and
	// nullifier value. Both are SHA-256 outputs so both are exactly 32 bytes.
	CommitmentLength = 32

	// MaxFieldOpenings is the maximum number of selective-disclosure openings
	// a single shielded transfer can carry. Prevents memory-amplification DoS.
	MaxFieldOpenings = 8

	// CommitProtocol is the protocol identifier the wallet stamps on every
	// commitment. Only commitments with this protocol tag are accepted.
	CommitProtocol = "LT-Shield-Commit-v2"

	// FieldSaltLength is the required length of every selective-disclosure
	// salt. Matches the wallet's 16-byte crypto.getRandomValues() salts.
	FieldSaltLength = 16
)

// ValidateShieldedCommitment returns nil if the commitment's static shape is
// valid and every provided field opening hashes to the declared field
// commitment. Does NOT check nullifier uniqueness — that requires Keeper
// state access.
func ValidateShieldedCommitment(c *ShieldedCommitment) error {
	if c == nil {
		return errors.New("privacy: commitment is nil")
	}
	if len(c.Commit) != CommitmentLength {
		return errors.New("privacy: commit must be exactly 32 bytes")
	}
	if len(c.Nullifier) != CommitmentLength {
		return errors.New("privacy: nullifier must be exactly 32 bytes")
	}
	if c.Protocol != CommitProtocol {
		return errors.New("privacy: unsupported commit protocol " + c.Protocol)
	}
	if len(c.FieldCommitments) > MaxFieldOpenings {
		return errors.New("privacy: too many field commitments")
	}
	if len(c.FieldOpenings) > MaxFieldOpenings {
		return errors.New("privacy: too many field openings")
	}
	for name, fc := range c.FieldCommitments {
		if len(fc) != CommitmentLength {
			return errors.New("privacy: field commitment '" + name + "' is not 32 bytes")
		}
	}
	// Every opening must match its declared commitment.
	for name, opening := range c.FieldOpenings {
		if opening == nil {
			return errors.New("privacy: opening '" + name + "' is nil")
		}
		fc, ok := c.FieldCommitments[name]
		if !ok {
			return errors.New("privacy: opening '" + name + "' has no corresponding commitment")
		}
		if len(opening.Salt) != FieldSaltLength {
			return errors.New("privacy: opening salt for '" + name + "' must be 16 bytes")
		}
		computed := HashFieldOpening(opening.Value, opening.Salt)
		if !equalBytes(computed, fc) {
			return errors.New("privacy: opening for '" + name + "' does not match commitment")
		}
	}
	return nil
}

// HashFieldOpening computes SHA-256(value || salt) matching the wallet's
// generateCommitment field-commitment format.
func HashFieldOpening(value, salt []byte) []byte {
	h := sha256.New()
	h.Write(value)
	h.Write(salt)
	return h.Sum(nil)
}

// CommitmentKey builds a KV-store key for a commitment by its commit bytes.
func CommitmentKey(commit []byte) []byte {
	key := make([]byte, 0, len(CommitmentPrefix)+len(commit))
	key = append(key, []byte(CommitmentPrefix)...)
	key = append(key, commit...)
	return key
}

// NullifierKey builds a KV-store key for a nullifier.
func NullifierKey(nullifier []byte) []byte {
	key := make([]byte, 0, len(NullifierPrefix)+len(nullifier))
	key = append(key, []byte(NullifierPrefix)...)
	key = append(key, nullifier...)
	return key
}

// MerkleRootKey returns the singleton key for the rolling Merkle root.
func MerkleRootKey() []byte {
	return []byte(MerkleRootPrefix)
}

// HexCommit returns the hex encoding of a commitment for logging + event emission.
func HexCommit(b []byte) string {
	return hex.EncodeToString(b)
}

// UpdateRollingRoot returns a new rolling Merkle accumulator root by hashing
// the prior root with the new commitment. Not a full Merkle tree — a linear
// hash chain, sufficient until a full Poseidon-friendly tree is wired in.
func UpdateRollingRoot(prior, commit []byte) []byte {
	h := sha256.New()
	h.Write(prior)
	h.Write(commit)
	return h.Sum(nil)
}

// equalBytes is a constant-time byte comparison that matches the semantics of
// subtle.ConstantTimeCompare without pulling in the extra import for the
// single call-site.
func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	diff := byte(0)
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
