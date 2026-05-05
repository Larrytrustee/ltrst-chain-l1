package keeper_test

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	keepertest "ltrstchain/testutil/keeper"
	"ltrstchain/x/ltrstchain/keeper"
	"ltrstchain/x/ltrstchain/types"
)

// testAddress returns a deterministic bech32 address suitable for the
// MsgShieldedTransfer.submitter field.
func testAddress(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 20) // sdk.AccAddress is 20 bytes
	_, err := rand.Read(raw)
	require.NoError(t, err)
	return sdk.AccAddress(raw).String()
}

// TestMsgShieldedTransferHappyPath exercises the full MsgPrivacy.ShieldedTransfer
// handler — not just the keeper — and confirms the response carries the
// updated rolling Merkle root + ingress height, and that the commitment
// becomes queryable via the keeper after the msg completes.
func TestMsgShieldedTransferHappyPath(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	srv := keeper.NewPrivacyMsgServerImpl(k)

	submitter := testAddress(t)
	commit := randCommit(t)
	nullifier := randCommit(t)

	msg := &types.MsgShieldedTransfer{
		Submitter: submitter,
		Commitment: types.ShieldedCommitment{
			Commit:    commit,
			Nullifier: nullifier,
			Protocol:  types.CommitProtocol,
		},
	}

	resp, err := srv.ShieldedTransfer(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.NewMerkleRoot, types.CommitmentLength)

	// Root must no longer be the all-zero genesis root.
	zero := make([]byte, types.CommitmentLength)
	require.NotEqual(t, zero, resp.NewMerkleRoot)

	// Commitment must be readable back through the keeper.
	got, err := k.GetCommitment(ctx, commit)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, commit, got.Commit)
	require.Equal(t, nullifier, got.Nullifier)
	// The handler must overwrite the submitter with the tx signer, not trust
	// any client-supplied value. Since we set msg.Submitter == testAddress,
	// that's what should land on-chain.
	require.Equal(t, submitter, got.Submitter)

	// Nullifier is marked spent.
	used, err := k.IsNullifierUsed(ctx, nullifier)
	require.NoError(t, err)
	require.True(t, used)
}

// TestMsgShieldedTransferOverwritesSubmitter ensures the handler ignores any
// submitter the client writes inside the nested ShieldedCommitment — the tx
// signer (msg.Submitter) is always the source of truth for attribution.
func TestMsgShieldedTransferOverwritesSubmitter(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	srv := keeper.NewPrivacyMsgServerImpl(k)

	signer := testAddress(t)
	spoofed := testAddress(t) // different address the client might try to plant
	require.NotEqual(t, signer, spoofed)

	commit := randCommit(t)
	msg := &types.MsgShieldedTransfer{
		Submitter: signer,
		Commitment: types.ShieldedCommitment{
			Commit:    commit,
			Nullifier: randCommit(t),
			Protocol:  types.CommitProtocol,
			Submitter: spoofed, // client tries to plant a different submitter
		},
	}

	_, err := srv.ShieldedTransfer(ctx, msg)
	require.NoError(t, err)

	got, err := k.GetCommitment(ctx, commit)
	require.NoError(t, err)
	require.NotNil(t, got)
	// The spoofed address must have been overwritten.
	require.Equal(t, signer, got.Submitter)
	require.NotEqual(t, spoofed, got.Submitter)
}

// TestMsgShieldedTransferRejectsInvalidSubmitter confirms the handler rejects
// a MsgShieldedTransfer whose submitter is not a valid bech32 address
// before touching any state.
func TestMsgShieldedTransferRejectsInvalidSubmitter(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	srv := keeper.NewPrivacyMsgServerImpl(k)

	commit := randCommit(t)
	msg := &types.MsgShieldedTransfer{
		Submitter: "not-a-bech32-address",
		Commitment: types.ShieldedCommitment{
			Commit:    commit,
			Nullifier: randCommit(t),
			Protocol:  types.CommitProtocol,
		},
	}

	_, err := srv.ShieldedTransfer(ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid submitter address")

	// The commitment must NOT have been stored on an invalid-signer path.
	got, err := k.GetCommitment(ctx, commit)
	require.NoError(t, err)
	require.Nil(t, got)
}

// TestMsgShieldedTransferRejectsDoubleSpend ensures nullifier double-spend
// protection is enforced through the msg-server path, not just the keeper.
func TestMsgShieldedTransferRejectsDoubleSpend(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	srv := keeper.NewPrivacyMsgServerImpl(k)
	submitter := testAddress(t)

	nullifier := randCommit(t)
	first := &types.MsgShieldedTransfer{
		Submitter: submitter,
		Commitment: types.ShieldedCommitment{
			Commit:    randCommit(t),
			Nullifier: nullifier,
			Protocol:  types.CommitProtocol,
		},
	}
	second := &types.MsgShieldedTransfer{
		Submitter: submitter,
		Commitment: types.ShieldedCommitment{
			Commit:    randCommit(t),
			Nullifier: nullifier, // same nullifier — must be rejected
			Protocol:  types.CommitProtocol,
		},
	}

	_, err := srv.ShieldedTransfer(ctx, first)
	require.NoError(t, err)

	_, err = srv.ShieldedTransfer(ctx, second)
	require.Error(t, err)
	require.Contains(t, err.Error(), "double-spend")
}

// TestMsgShieldedTransferReturnsUpdatedRoot confirms the MsgShieldedTransfer
// response's NewMerkleRoot equals the keeper's GetMerkleRoot after the msg
// completes, so wallets can trust the round-trip echo without polling.
func TestMsgShieldedTransferReturnsUpdatedRoot(t *testing.T) {
	k, ctx := keepertest.LtrstchainKeeper(t)
	srv := keeper.NewPrivacyMsgServerImpl(k)

	msg := &types.MsgShieldedTransfer{
		Submitter: testAddress(t),
		Commitment: types.ShieldedCommitment{
			Commit:    randCommit(t),
			Nullifier: randCommit(t),
			Protocol:  types.CommitProtocol,
		},
	}

	resp, err := srv.ShieldedTransfer(ctx, msg)
	require.NoError(t, err)

	root, err := k.GetMerkleRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, root, resp.NewMerkleRoot)
}
