package keeper_test

import (
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "ltrstchain/testutil/keeper"
	"ltrstchain/x/ltrstchain/types"
)

// randCommit returns 32 random bytes suitable for a commitment or nullifier.
func randCommit(t *testing.T) []byte {
	t.Helper()
	b := make([]byte, types.CommitmentLength)
	_, err := rand.Read(b)
	require.NoError(t, err)
	return b
}

func TestPrivacyAddCommitmentHappyPath(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	c := &types.ShieldedCommitment{
		Commit:    randCommit(t),
		Nullifier: randCommit(t),
		Protocol:  types.CommitProtocol,
	}

	require.NoError(t, k.AddCommitment(ctx, c))

	got, err := k.GetCommitment(ctx, c.Commit)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, c.Commit, got.Commit)
	require.Equal(t, c.Nullifier, got.Nullifier)

	used, err := k.IsNullifierUsed(ctx, c.Nullifier)
	require.NoError(t, err)
	require.True(t, used)

	root, err := k.GetMerkleRoot(ctx)
	require.NoError(t, err)
	require.Len(t, root, types.CommitmentLength)
	// Root must have changed from the all-zero genesis root.
	zero := make([]byte, types.CommitmentLength)
	require.NotEqual(t, zero, root)
}

func TestPrivacyRejectsDoubleSpend(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	nullifier := randCommit(t)

	first := &types.ShieldedCommitment{
		Commit:    randCommit(t),
		Nullifier: nullifier,
		Protocol:  types.CommitProtocol,
	}
	second := &types.ShieldedCommitment{
		Commit:    randCommit(t), // different commit
		Nullifier: nullifier,     // same nullifier — must be rejected
		Protocol:  types.CommitProtocol,
	}

	require.NoError(t, k.AddCommitment(ctx, first))
	err := k.AddCommitment(ctx, second)
	require.Error(t, err)
	require.Contains(t, err.Error(), "double-spend")
}

func TestPrivacyRejectsWrongProtocol(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	c := &types.ShieldedCommitment{
		Commit:    randCommit(t),
		Nullifier: randCommit(t),
		Protocol:  "some-other-protocol",
	}
	err := k.AddCommitment(ctx, c)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported commit protocol")
}

func TestPrivacyRejectsShortCommit(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	c := &types.ShieldedCommitment{
		Commit:    []byte{0x01, 0x02}, // too short
		Nullifier: randCommit(t),
		Protocol:  types.CommitProtocol,
	}
	err := k.AddCommitment(ctx, c)
	require.Error(t, err)
	require.Contains(t, err.Error(), "32 bytes")
}

func TestPrivacyFieldOpeningValidation(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)

	// Build an "amount" field with a known salt + value and its matching
	// commitment so the opening self-validates.
	amountValue := []byte("1000")
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	require.NoError(t, err)

	amountCommit := types.HashFieldOpening(amountValue, salt)

	c := &types.ShieldedCommitment{
		Commit:    randCommit(t),
		Nullifier: randCommit(t),
		FieldCommitments: map[string][]byte{
			"amount": amountCommit,
		},
		FieldOpenings: map[string]*types.FieldOpening{
			"amount": {Value: amountValue, Salt: salt},
		},
		Protocol: types.CommitProtocol,
	}
	require.NoError(t, k.AddCommitment(ctx, c))
}

func TestPrivacyFieldOpeningMismatchRejected(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)

	// Build an opening whose value doesn't match the declared commitment.
	amountValue := []byte("1000")
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	require.NoError(t, err)

	// Commit to "9999" but try to disclose "1000" instead.
	wrongCommit := types.HashFieldOpening([]byte("9999"), salt)

	c := &types.ShieldedCommitment{
		Commit:    randCommit(t),
		Nullifier: randCommit(t),
		FieldCommitments: map[string][]byte{
			"amount": wrongCommit,
		},
		FieldOpenings: map[string]*types.FieldOpening{
			"amount": {Value: amountValue, Salt: salt},
		},
		Protocol: types.CommitProtocol,
	}
	err = k.AddCommitment(ctx, c)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not match commitment")
}

func TestPrivacyRollingRootMatchesExpected(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)

	commitA := randCommit(t)
	commitB := randCommit(t)

	require.NoError(t, k.AddCommitment(ctx, &types.ShieldedCommitment{
		Commit: commitA, Nullifier: randCommit(t), Protocol: types.CommitProtocol,
	}))
	require.NoError(t, k.AddCommitment(ctx, &types.ShieldedCommitment{
		Commit: commitB, Nullifier: randCommit(t), Protocol: types.CommitProtocol,
	}))

	// Expected: H(H(zero || commitA) || commitB)
	zero := make([]byte, types.CommitmentLength)
	first := sha256.Sum256(append(append([]byte{}, zero...), commitA...))
	second := sha256.Sum256(append(append([]byte{}, first[:]...), commitB...))

	got, err := k.GetMerkleRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, second[:], got)
}
