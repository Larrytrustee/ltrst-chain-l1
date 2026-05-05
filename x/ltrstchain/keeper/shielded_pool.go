package keeper

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"ltrstchain/x/ltrstchain/types"
)

// DepositToShieldedPool moves `amount` ultrst from `depositor` into the
// shielded-pool module account. The commitment is recorded via AddCommitment
// in the same atomic flow — either both state changes succeed, or the whole
// tx rolls back.
//
// Semantics:
//   1. Validate that the depositor has enough spendable balance.
//   2. Validate + record the ShieldedCommitment (reuses AddCommitment's
//      double-spend and opening-verification checks).
//   3. Transfer `amount` from the depositor's account to the shielded pool
//      module account via the bank keeper. After this, the depositor's
//      transparent balance is reduced and the pool's balance is increased.
//
// The caller is expected to be a Msg handler (MsgShieldedDeposit, once the
// proto layer lands) that has already verified the tx signer matches
// `depositor`. The keeper method itself trusts the supplied address.
//
// Returns an error if the commitment is invalid, the nullifier is already
// spent, the depositor lacks funds, or the bank keeper isn't wired
// (ShieldedPool value flow disabled — commitments still work).
func (k Keeper) DepositToShieldedPool(
	ctx context.Context,
	depositor sdk.AccAddress,
	amount sdk.Coins,
	c *types.ShieldedCommitment,
) error {
	if k.bankKeeper == nil {
		return errors.New("privacy: shielded-pool value flow is disabled (bankKeeper not wired)")
	}
	if c == nil {
		return errors.New("privacy: deposit requires a commitment")
	}
	if amount.IsZero() {
		return errors.New("privacy: deposit amount must be positive")
	}
	if !amount.IsValid() {
		return errors.New("privacy: deposit amount contains invalid coins")
	}

	// Record commitment FIRST — if this fails (double-spend, bad opening,
	// wrong protocol), the deposit is rejected before any bank movement.
	if err := k.AddCommitment(ctx, c); err != nil {
		return fmt.Errorf("privacy: deposit commitment rejected: %w", err)
	}

	// Move funds from depositor → shielded pool. Any bank-keeper error
	// (insufficient funds, blocked address, etc.) bubbles up and aborts
	// the whole tx thanks to Cosmos's msg-level state atomicity.
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, depositor, types.ShieldedPoolName, amount); err != nil {
		return fmt.Errorf("privacy: deposit bank transfer: %w", err)
	}

	// Emit a deposit event so explorers and indexers can see the shielded
	// pool's TVL grow. The commit itself is already emitted by AddCommitment
	// — this event is the pool-balance-delta counterpart.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"shielded_pool_deposit",
		sdk.NewAttribute("depositor", depositor.String()),
		sdk.NewAttribute("amount", amount.String()),
		sdk.NewAttribute("commit", types.HexCommit(c.Commit)),
	))
	return nil
}

// WithdrawFromShieldedPool moves `amount` ultrst from the shielded-pool
// module account to `recipient`. The caller must supply the nullifier that
// corresponds to a previously-deposited commitment; the nullifier is marked
// spent here so no future withdrawal can reuse it.
//
// Security model (interim, pre-zk):
//   - The authority (governance or a designated module authority) is the
//     only signer permitted to invoke withdrawals today.
//   - A proper zk-SNARK proof replaces the authority signature in a future
//     version: the prover demonstrates knowledge of a commitment opening
//     (value + salt + nullifier-preimage) without revealing which deposit.
//   - Until then, this method is intentionally restrictive: only the
//     module authority (matches k.GetAuthority()) can call it, to prevent
//     anyone from draining the shielded pool.
//
// Semantics:
//   1. Reject if nullifier already used (prevents replay).
//   2. Mark nullifier as used.
//   3. Transfer from pool → recipient.
//
// This method writes to the nullifier set, so a subsequent call with the
// same nullifier is rejected.
func (k Keeper) WithdrawFromShieldedPool(
	ctx context.Context,
	authority string,
	recipient sdk.AccAddress,
	amount sdk.Coins,
	nullifier []byte,
) error {
	if k.bankKeeper == nil {
		return errors.New("privacy: shielded-pool value flow is disabled (bankKeeper not wired)")
	}
	if authority != k.authority {
		return fmt.Errorf("privacy: withdraw authority mismatch — expected %q got %q", k.authority, authority)
	}
	if len(nullifier) != types.CommitmentLength {
		return errors.New("privacy: withdraw nullifier must be exactly 32 bytes")
	}
	if amount.IsZero() {
		return errors.New("privacy: withdraw amount must be positive")
	}
	if !amount.IsValid() {
		return errors.New("privacy: withdraw amount contains invalid coins")
	}

	// Reject double-spend on the nullifier set.
	used, err := k.IsNullifierUsed(ctx, nullifier)
	if err != nil {
		return fmt.Errorf("privacy: withdraw nullifier lookup: %w", err)
	}
	if used {
		return errors.New("privacy: withdraw nullifier already spent")
	}

	// Mark nullifier as spent BEFORE moving funds so a concurrent-tx
	// replay against the same nullifier hits the spent state immediately.
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(types.NullifierKey(nullifier), []byte{1}); err != nil {
		return fmt.Errorf("privacy: withdraw nullifier write: %w", err)
	}

	// Move funds from shielded pool → recipient.
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ShieldedPoolName, recipient, amount); err != nil {
		return fmt.Errorf("privacy: withdraw bank transfer: %w", err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"shielded_pool_withdraw",
		sdk.NewAttribute("recipient", recipient.String()),
		sdk.NewAttribute("amount", amount.String()),
		sdk.NewAttribute("nullifier", types.HexCommit(nullifier)),
	))
	return nil
}

// ShieldedPoolBalance returns the current ultrst balance of the shielded
// pool module account. Returns nil coins if the bank keeper isn't wired.
// Useful for explorers, dashboards, and test assertions. The caller must
// already hold a module-address lookup; we expose a string-free helper
// here so callers don't have to import authtypes just to query balance.
func (k Keeper) ShieldedPoolBalance(ctx context.Context, poolAddr sdk.AccAddress) sdk.Coins {
	if k.bankKeeper == nil {
		return nil
	}
	return k.bankKeeper.GetAllBalances(ctx, poolAddr)
}
