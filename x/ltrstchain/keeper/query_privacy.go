package keeper

import (
	"context"
	"encoding/hex"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ltrstchain/x/ltrstchain/types"
)

// privacyQueryServer implements types.QueryPrivacyServer. Kept separate from
// the existing Query service (which owns Params) so the two can be registered
// independently without touching query.pb.go.
type privacyQueryServer struct {
	Keeper
}

// NewPrivacyQueryServerImpl returns a concrete QueryPrivacyServer backed by
// the provided Keeper.
func NewPrivacyQueryServerImpl(keeper Keeper) types.QueryPrivacyServer {
	return &privacyQueryServer{Keeper: keeper}
}

var _ types.QueryPrivacyServer = privacyQueryServer{}

// Commitment returns a stored commitment by its hex-encoded commit bytes.
// When no commitment exists for the given commit, Found=false is returned
// with an empty ShieldedCommitment so callers can distinguish "not found"
// from a zero-byte commit without inspecting gRPC error codes.
func (k privacyQueryServer) Commitment(goCtx context.Context, req *types.QueryCommitmentRequest) (*types.QueryCommitmentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "privacy: request is nil")
	}
	commit, err := hex.DecodeString(req.Commit)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "privacy: commit must be hex-encoded")
	}
	if len(commit) != types.CommitmentLength {
		return nil, status.Error(codes.InvalidArgument, "privacy: commit must decode to exactly 32 bytes")
	}

	c, err := k.GetCommitment(goCtx, commit)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if c == nil {
		return &types.QueryCommitmentResponse{
			Commitment: types.ShieldedCommitment{},
			Found:      false,
			Height:     0,
		}, nil
	}
	return &types.QueryCommitmentResponse{
		Commitment: *c,
		Found:      true,
		Height:     c.Height,
	}, nil
}

// NullifierUsed returns whether a given hex-encoded nullifier has already
// been recorded in a prior accepted commitment.
func (k privacyQueryServer) NullifierUsed(goCtx context.Context, req *types.QueryNullifierUsedRequest) (*types.QueryNullifierUsedResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "privacy: request is nil")
	}
	nullifier, err := hex.DecodeString(req.Nullifier)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "privacy: nullifier must be hex-encoded")
	}
	if len(nullifier) != types.CommitmentLength {
		return nil, status.Error(codes.InvalidArgument, "privacy: nullifier must decode to exactly 32 bytes")
	}

	used, err := k.IsNullifierUsed(goCtx, nullifier)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryNullifierUsedResponse{Used: used}, nil
}

// MerkleRoot returns the current 32-byte rolling accumulator root. The
// genesis root is 32 zero bytes.
func (k privacyQueryServer) MerkleRoot(goCtx context.Context, req *types.QueryMerkleRootRequest) (*types.QueryMerkleRootResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "privacy: request is nil")
	}
	root, err := k.GetMerkleRoot(goCtx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if root == nil {
		return nil, errors.New("privacy: merkle root is nil")
	}
	return &types.QueryMerkleRootResponse{Root: root}, nil
}
