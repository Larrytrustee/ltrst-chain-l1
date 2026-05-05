package keeper

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"ltrstchain/x/ltrstchain/types"
)

// AddCommitment accepts a new shielded commitment, rejects a double-spend via
// the nullifier set, persists the commitment + nullifier, and updates the
// rolling Merkle accumulator root.
//
// The commitment is validated statically via types.ValidateShieldedCommitment
// before state is touched. Any field openings the sender chose to reveal must
// self-check against their declared field commitments.
func (k Keeper) AddCommitment(ctx context.Context, c *types.ShieldedCommitment) error {
	if err := types.ValidateShieldedCommitment(c); err != nil {
		return err
	}
	store := k.storeService.OpenKVStore(ctx)

	// Reject double-spend: if the nullifier has already been seen, the
	// transaction was already processed and must not replay.
	nullKey := types.NullifierKey(c.Nullifier)
	has, err := store.Has(nullKey)
	if err != nil {
		return fmt.Errorf("privacy: nullifier lookup: %w", err)
	}
	if has {
		return errors.New("privacy: nullifier already spent — double-spend rejected")
	}

	// Record block height so future queries can tell when a commitment landed.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	c.Height = sdkCtx.BlockHeight()

	commitKey := types.CommitmentKey(c.Commit)
	commitBytes, err := k.cdc.Marshal(c)
	if err != nil {
		return fmt.Errorf("privacy: marshal commitment: %w", err)
	}
	if err := store.Set(commitKey, commitBytes); err != nil {
		return fmt.Errorf("privacy: store commitment: %w", err)
	}

	// Mark nullifier as spent. Value is a single-byte sentinel; the set
	// membership is all that matters here.
	if err := store.Set(nullKey, []byte{1}); err != nil {
		return fmt.Errorf("privacy: store nullifier: %w", err)
	}

	// Update the rolling Merkle accumulator root.
	rootKey := types.MerkleRootKey()
	priorRoot, err := store.Get(rootKey)
	if err != nil {
		return fmt.Errorf("privacy: read merkle root: %w", err)
	}
	if priorRoot == nil {
		// Chain-genesis root is 32 zero bytes so the first commitment still
		// produces a well-defined new root.
		priorRoot = make([]byte, types.CommitmentLength)
	}
	newRoot := types.UpdateRollingRoot(priorRoot, c.Commit)
	if err := store.Set(rootKey, newRoot); err != nil {
		return fmt.Errorf("privacy: write merkle root: %w", err)
	}

	// Emit an event so watchers (wallet, explorer, indexer) see the commit.
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"privacy_commitment",
		sdk.NewAttribute("commit", types.HexCommit(c.Commit)),
		sdk.NewAttribute("nullifier", types.HexCommit(c.Nullifier)),
		sdk.NewAttribute("merkle_root", types.HexCommit(newRoot)),
		sdk.NewAttribute("height", fmt.Sprintf("%d", c.Height)),
		sdk.NewAttribute("protocol", c.Protocol),
	))
	return nil
}

// IsNullifierUsed returns true if the given nullifier has been recorded in
// any prior accepted commitment.
func (k Keeper) IsNullifierUsed(ctx context.Context, nullifier []byte) (bool, error) {
	if len(nullifier) != types.CommitmentLength {
		return false, errors.New("privacy: nullifier must be exactly 32 bytes")
	}
	store := k.storeService.OpenKVStore(ctx)
	return store.Has(types.NullifierKey(nullifier))
}

// GetCommitment fetches a stored commitment by its commit bytes. Returns
// (nil, nil) when the commitment does not exist.
func (k Keeper) GetCommitment(ctx context.Context, commit []byte) (*types.ShieldedCommitment, error) {
	if len(commit) != types.CommitmentLength {
		return nil, errors.New("privacy: commit must be exactly 32 bytes")
	}
	store := k.storeService.OpenKVStore(ctx)
	raw, err := store.Get(types.CommitmentKey(commit))
	if err != nil {
		return nil, fmt.Errorf("privacy: read commitment: %w", err)
	}
	if raw == nil {
		return nil, nil
	}
	var out types.ShieldedCommitment
	if err := k.cdc.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("privacy: unmarshal commitment: %w", err)
	}
	return &out, nil
}

// GetMerkleRoot returns the current rolling accumulator root. A 32-byte
// all-zero slice is returned when no commitments have been stored yet.
func (k Keeper) GetMerkleRoot(ctx context.Context) ([]byte, error) {
	store := k.storeService.OpenKVStore(ctx)
	root, err := store.Get(types.MerkleRootKey())
	if err != nil {
		return nil, err
	}
	if root == nil {
		return make([]byte, types.CommitmentLength), nil
	}
	return root, nil
}

// IterateCommitments walks every stored shielded commitment in ascending
// commit-bytes order, unmarshals each into a ShieldedCommitment, and invokes
// the provided callback. Iteration stops (and the iterator is closed) if the
// callback returns true. Used by ExportGenesis to snapshot the full privacy
// state without loading all entries into memory at once for very large sets.
func (k Keeper) IterateCommitments(ctx context.Context, cb func(c types.ShieldedCommitment) (stop bool)) error {
	store := k.storeService.OpenKVStore(ctx)
	prefix := []byte(types.CommitmentPrefix)
	iter, err := store.Iterator(prefix, prefixEnd(prefix))
	if err != nil {
		return fmt.Errorf("privacy: commitment iterator: %w", err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var c types.ShieldedCommitment
		if err := k.cdc.Unmarshal(iter.Value(), &c); err != nil {
			return fmt.Errorf("privacy: unmarshal commitment during iterate: %w", err)
		}
		if cb(c) {
			return nil
		}
	}
	return nil
}

// prefixEnd returns the exclusive upper bound for a prefix range scan,
// equivalent to storetypes.PrefixEndBytes. Inlined so the keeper doesn't need
// to pull in cosmossdk.io/store/types for a single call site.
func prefixEnd(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for {
		if end[len(end)-1] != 0xff {
			end[len(end)-1]++
			return end
		}
		end = end[:len(end)-1]
		if len(end) == 0 {
			return nil
		}
	}
}
