package keeper

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"ltrstchain/x/ltrstchain/types"
)

// privacyMsgServer implements the generated types.MsgPrivacyServer interface.
// It is kept separate from the existing Msg service (which owns UpdateParams)
// so the two Msg services can be registered independently.
type privacyMsgServer struct {
	Keeper
}

// NewPrivacyMsgServerImpl returns a concrete MsgPrivacyServer backed by the
// provided Keeper. Register it alongside the existing Msg service via
// types.RegisterMsgPrivacyServer(cfg.MsgServer(), NewPrivacyMsgServerImpl(k))
// from the module's RegisterServices method.
func NewPrivacyMsgServerImpl(keeper Keeper) types.MsgPrivacyServer {
	return &privacyMsgServer{Keeper: keeper}
}

var _ types.MsgPrivacyServer = privacyMsgServer{}

// ShieldedTransfer records a shielded commitment on-chain by delegating to
// Keeper.AddCommitment. The submitter address is overwritten with the signer
// declared by the tx's msg signer so client-supplied values cannot forge a
// different submitter.
func (k privacyMsgServer) ShieldedTransfer(goCtx context.Context, msg *types.MsgShieldedTransfer) (*types.MsgShieldedTransferResponse, error) {
	if msg == nil {
		return nil, errors.New("privacy: MsgShieldedTransfer is nil")
	}

	// The tx's declared signer wins — the commitment's own submitter field is
	// informational only and must not be trusted for attribution.
	if _, err := sdk.AccAddressFromBech32(msg.Submitter); err != nil {
		return nil, errors.New("privacy: invalid submitter address: " + err.Error())
	}

	// Work on a copy of the commitment to avoid mutating msg state past the
	// ABCI boundary. AddCommitment sets Height on the pointer it receives.
	c := msg.Commitment
	c.Submitter = msg.Submitter
	if err := k.AddCommitment(goCtx, &c); err != nil {
		return nil, err
	}

	newRoot, err := k.GetMerkleRoot(goCtx)
	if err != nil {
		return nil, err
	}
	return &types.MsgShieldedTransferResponse{
		NewMerkleRoot: newRoot,
		Height:        c.Height,
	}, nil
}

// ShieldedDeposit atomically records a commitment AND escrows the supplied
// coins into the shielded-pool module account. The tx signer (= depositor)
// is the only party whose balance is debited; the commitment's submitter
// field is overwritten with the signer for attribution consistency with
// ShieldedTransfer.
func (k privacyMsgServer) ShieldedDeposit(goCtx context.Context, msg *types.MsgShieldedDeposit) (*types.MsgShieldedDepositResponse, error) {
	if msg == nil {
		return nil, errors.New("privacy: MsgShieldedDeposit is nil")
	}

	depositor, err := sdk.AccAddressFromBech32(msg.Depositor)
	if err != nil {
		return nil, fmt.Errorf("privacy: invalid depositor address: %w", err)
	}

	// Normalize the commitment's submitter field to the tx signer (same
	// rule ShieldedTransfer uses — client-supplied values are informational
	// only and must not forge attribution).
	c := msg.Commitment
	c.Submitter = msg.Depositor

	if err := k.Keeper.DepositToShieldedPool(goCtx, depositor, msg.Amount, &c); err != nil {
		return nil, err
	}

	newRoot, err := k.GetMerkleRoot(goCtx)
	if err != nil {
		return nil, err
	}

	// Query the pool's post-deposit balance so the caller sees the updated
	// TVL without a separate RPC round-trip.
	poolAddr := authtypes.NewModuleAddress(types.ShieldedPoolName)
	poolBalance := k.Keeper.ShieldedPoolBalance(goCtx, poolAddr)

	return &types.MsgShieldedDepositResponse{
		NewMerkleRoot: newRoot,
		Height:        c.Height,
		PoolBalance:   poolBalance,
	}, nil
}

// ShieldedWithdraw releases coins from the shielded pool to the recipient.
// Interim authority-gated: the signer (authority in the msg) must match the
// module's configured authority. Once a zk-SNARK verifier is wired into
// the chain, this check is replaced with proof-of-knowledge verification.
func (k privacyMsgServer) ShieldedWithdraw(goCtx context.Context, msg *types.MsgShieldedWithdraw) (*types.MsgShieldedWithdrawResponse, error) {
	if msg == nil {
		return nil, errors.New("privacy: MsgShieldedWithdraw is nil")
	}

	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return nil, fmt.Errorf("privacy: invalid authority address: %w", err)
	}
	recipient, err := sdk.AccAddressFromBech32(msg.Recipient)
	if err != nil {
		return nil, fmt.Errorf("privacy: invalid recipient address: %w", err)
	}

	if err := k.Keeper.WithdrawFromShieldedPool(goCtx, msg.Authority, recipient, msg.Amount, msg.Nullifier); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	poolAddr := authtypes.NewModuleAddress(types.ShieldedPoolName)
	poolBalance := k.Keeper.ShieldedPoolBalance(goCtx, poolAddr)

	return &types.MsgShieldedWithdrawResponse{
		Height:      sdkCtx.BlockHeight(),
		PoolBalance: poolBalance,
	}, nil
}
