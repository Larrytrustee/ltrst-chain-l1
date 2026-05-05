package keeper_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "ltrstchain/testutil/keeper"
	"ltrstchain/x/ltrstchain/keeper"
	"ltrstchain/x/ltrstchain/types"
)

// TestQueryPrivacyCommitmentHappyPath ensures the QueryPrivacy.Commitment gRPC
// handler returns a stored commitment with Found=true + matching bytes, and
// hex-decodes its wire parameter correctly.
func TestQueryPrivacyCommitmentHappyPath(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	qsrv := keeper.NewPrivacyQueryServerImpl(k)

	commit := randCommit(t)
	nullifier := randCommit(t)
	c := &types.ShieldedCommitment{
		Commit:    commit,
		Nullifier: nullifier,
		Protocol:  types.CommitProtocol,
	}
	require.NoError(t, k.AddCommitment(ctx, c))

	resp, err := qsrv.Commitment(ctx, &types.QueryCommitmentRequest{
		Commit: hex.EncodeToString(commit),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.Found)
	require.Equal(t, commit, resp.Commitment.Commit)
	require.Equal(t, nullifier, resp.Commitment.Nullifier)
}

// TestQueryPrivacyCommitmentNotFound ensures a missing commitment returns
// Found=false rather than a gRPC error, so clients can distinguish
// "not present" from a protocol fault.
func TestQueryPrivacyCommitmentNotFound(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	qsrv := keeper.NewPrivacyQueryServerImpl(k)

	resp, err := qsrv.Commitment(ctx, &types.QueryCommitmentRequest{
		Commit: hex.EncodeToString(randCommit(t)),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.Found)
}

// TestQueryPrivacyCommitmentInvalidHex ensures a non-hex commit parameter
// returns InvalidArgument, not a panic or Internal error.
func TestQueryPrivacyCommitmentInvalidHex(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	qsrv := keeper.NewPrivacyQueryServerImpl(k)

	_, err := qsrv.Commitment(ctx, &types.QueryCommitmentRequest{
		Commit: "not-hex!",
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
}

// TestQueryPrivacyCommitmentShortCommit rejects a hex value whose decoded
// length is not 32 bytes.
func TestQueryPrivacyCommitmentShortCommit(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	qsrv := keeper.NewPrivacyQueryServerImpl(k)

	_, err := qsrv.Commitment(ctx, &types.QueryCommitmentRequest{
		Commit: hex.EncodeToString([]byte{0x01, 0x02}),
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
}

// TestQueryPrivacyNullifierUsed returns true after a commitment is stored and
// false for a random unseen nullifier.
func TestQueryPrivacyNullifierUsed(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	qsrv := keeper.NewPrivacyQueryServerImpl(k)

	nullifier := randCommit(t)
	require.NoError(t, k.AddCommitment(ctx, &types.ShieldedCommitment{
		Commit:    randCommit(t),
		Nullifier: nullifier,
		Protocol:  types.CommitProtocol,
	}))

	// Stored nullifier should read as used.
	resp, err := qsrv.NullifierUsed(ctx, &types.QueryNullifierUsedRequest{
		Nullifier: hex.EncodeToString(nullifier),
	})
	require.NoError(t, err)
	require.True(t, resp.Used)

	// A fresh random nullifier should not.
	resp, err = qsrv.NullifierUsed(ctx, &types.QueryNullifierUsedRequest{
		Nullifier: hex.EncodeToString(randCommit(t)),
	})
	require.NoError(t, err)
	require.False(t, resp.Used)
}

// TestQueryPrivacyMerkleRootGenesis ensures the genesis root is 32 zero bytes
// before any commitment is stored.
func TestQueryPrivacyMerkleRootGenesis(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	qsrv := keeper.NewPrivacyQueryServerImpl(k)

	resp, err := qsrv.MerkleRoot(ctx, &types.QueryMerkleRootRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Root, types.CommitmentLength)
	require.Equal(t, make([]byte, types.CommitmentLength), resp.Root)
}

// TestQueryPrivacyMerkleRootAdvances ensures MerkleRoot returns the updated
// accumulator after a commitment is stored.
func TestQueryPrivacyMerkleRootAdvances(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	qsrv := keeper.NewPrivacyQueryServerImpl(k)

	require.NoError(t, k.AddCommitment(ctx, &types.ShieldedCommitment{
		Commit:    randCommit(t),
		Nullifier: randCommit(t),
		Protocol:  types.CommitProtocol,
	}))

	resp, err := qsrv.MerkleRoot(ctx, &types.QueryMerkleRootRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Root, types.CommitmentLength)
	// Advanced past the all-zero genesis root.
	require.NotEqual(t, make([]byte, types.CommitmentLength), resp.Root)
}
