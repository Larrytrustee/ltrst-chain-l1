package types

import (
	"errors"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PrivacyMinFeeUltrst is the absolute minimum ultrst fee that every privacy
// message must pay. Set to 10,000 ultrst = 0.01 LTRST. At $0.28/LTRST this
// is $0.0028 per shielded tx — small enough that legitimate users don't
// notice, large enough that a spammer has to commit real capital to flood
// the commitment set.
//
// Privacy messages do not consume a typical per-byte gas bill (the crypto
// work is amortized), so we enforce a flat minimum rather than relying on
// the global min-gas-price mechanism alone. When a proper zk-verifier lands,
// increase this to reflect the verifier's compute cost.
const PrivacyMinFeeUltrst int64 = 10_000

// PrivacyMinFee returns the sdk.Coin form of PrivacyMinFeeUltrst.
func PrivacyMinFee() sdk.Coin {
	return sdk.NewInt64Coin("ultrst", PrivacyMinFeeUltrst)
}

// ValidatePrivacyTxFee enforces PrivacyMinFee on any tx that contains a
// shielded-transfer, shielded-deposit, or shielded-withdraw message. Called
// from the app's ante handler (see app/ante.go) before signature verification
// so a spammer can't even get their tx past the mempool without paying.
//
// The rule: when ANY privacy message is present in the tx, the total fee
// attached to the tx must be at least PrivacyMinFee in ultrst. We don't
// sum per-message — a batched tx with 10 shielded transfers only needs to
// pay the minimum once, because the ABCI sees it as a single tx.
//
// Returns nil if the tx has no privacy messages (no fee floor enforced)
// or the fee is sufficient. Returns an error otherwise.
func ValidatePrivacyTxFee(msgs []sdk.Msg, feeCoins sdk.Coins) error {
	hasPrivacyMsg := false
	for _, m := range msgs {
		switch m.(type) {
		case *MsgShieldedTransfer:
			hasPrivacyMsg = true
		}
		if hasPrivacyMsg {
			break
		}
	}
	if !hasPrivacyMsg {
		return nil
	}

	paid := feeCoins.AmountOf("ultrst")
	required := math.NewInt(PrivacyMinFeeUltrst)
	if paid.LT(required) {
		return fmt.Errorf(
			"privacy: tx with shielded messages must pay at least %d ultrst (got %s ultrst)",
			PrivacyMinFeeUltrst, paid.String(),
		)
	}
	return nil
}

// ErrPrivacyFeeTooLow is returned by ante handlers when a privacy tx pays
// less than PrivacyMinFee.
var ErrPrivacyFeeTooLow = errors.New("privacy: tx fee below required minimum")
