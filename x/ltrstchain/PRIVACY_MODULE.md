# x/ltrstchain Privacy Module — Current State

**Last verified:** 2026-04-11
**Binary:** `ltrstchaind` (built at `/tmp/ltrst-bin/ltrstchaind`, ~124 MB)
**Tests:** 19/19 privacy tests passing across keeper + module packages
  - 7 in `x/ltrstchain/keeper/privacy_test.go` (keeper-level)
  - 5 in `x/ltrstchain/keeper/msg_shielded_transfer_test.go` (msg server)
  - 7 in `x/ltrstchain/keeper/query_privacy_test.go` (query server)
  - 6 in `x/ltrstchain/module/genesis_privacy_test.go` (InitGenesis/ExportGenesis)

This file is the single source of truth for what the chain-side privacy
module actually does today. It is the chain counterpart to
`lt_wallet_flutter/WALLET_PRIVACY.md` and must stay in sync with it.

## What is live in the compiled binary

The privacy module is fully wired into `ltrstchaind`. Wallets can submit a
`MsgShieldedTransfer` via the standard Cosmos SDK tx flow and query the
on-chain commitment state via three gRPC endpoints.

Verified by symbol + service descriptor grep:

```
T  ltrstchain/x/ltrstchain/keeper.(*Keeper).AddCommitment
T  ltrstchain/x/ltrstchain/keeper.(*Keeper).GetCommitment
T  ltrstchain/x/ltrstchain/keeper.(*Keeper).GetMerkleRoot
T  ltrstchain/x/ltrstchain/keeper.(*Keeper).IsNullifierUsed
T  ltrstchain/x/ltrstchain/keeper.(*Keeper).IterateCommitments
T  ltrstchain/x/ltrstchain/keeper.(*privacyMsgServer).ShieldedTransfer
T  ltrstchain/x/ltrstchain/keeper.(*privacyQueryServer).Commitment
T  ltrstchain/x/ltrstchain/keeper.(*privacyQueryServer).NullifierUsed
T  ltrstchain/x/ltrstchain/keeper.(*privacyQueryServer).MerkleRoot
T  ltrstchain/x/ltrstchain/module.InitGenesis
T  ltrstchain/x/ltrstchain/module.ExportGenesis
D  ltrstchain/x/ltrstchain/types._MsgPrivacy_serviceDesc
D  ltrstchain/x/ltrstchain/types._QueryPrivacy_serviceDesc
```

### Keeper methods

All defined in `x/ltrstchain/keeper/privacy.go` and backed by
`storeService.OpenKVStore(ctx)`. Commitments are marshaled with the
module's `codec.BinaryCodec` so serialization stays consensus-deterministic.

| Method              | What it does                                                                              |
|---------------------|-------------------------------------------------------------------------------------------|
| `AddCommitment`     | Validates → rejects double-spend → stores commit + nullifier → rolls root                 |
| `IsNullifierUsed`   | O(1) lookup against the nullifier set                                                     |
| `GetCommitment`     | Fetches stored commitment by its 32-byte commit key                                       |
| `GetMerkleRoot`     | Returns the current 32-byte rolling Merkle accumulator (zero-init genesis)                |
| `IterateCommitments`| Streams every stored commitment in commit-bytes order; used by `ExportGenesis`            |

### Msg service — `MsgPrivacy`

Defined in `proto/ltrstchain/ltrstchain/privacy.proto`, implemented in
`x/ltrstchain/keeper/msg_shielded_transfer.go`, registered in
`x/ltrstchain/module/module.go`'s `RegisterServices`.

| RPC                | Request                    | Response                                                       |
|--------------------|----------------------------|----------------------------------------------------------------|
| `ShieldedTransfer` | `MsgShieldedTransfer`      | `MsgShieldedTransferResponse{new_merkle_root, height}`         |

The handler:

1. Validates the submitter bech32 address and overwrites
   `commitment.submitter` with the tx signer — client-supplied values for
   this field are ignored for attribution.
2. Delegates the whole ShieldedCommitment to `Keeper.AddCommitment`, which
   runs `ValidateShieldedCommitment` → nullifier uniqueness check → store
   writes → rolling root update → event emission.
3. Returns the updated Merkle root + block height so the wallet can
   confirm the commitment landed without waiting for an indexer.

### Query service — `QueryPrivacy`

Defined in `proto/ltrstchain/ltrstchain/privacy.proto`, implemented in
`x/ltrstchain/keeper/query_privacy.go`, registered alongside MsgPrivacy.

| RPC              | REST path                                                  | Returns                                                 |
|------------------|------------------------------------------------------------|---------------------------------------------------------|
| `Commitment`     | `GET /ltrstchain/ltrstchain/privacy/commitment/{commit}`   | `{commitment, found, height}`                           |
| `NullifierUsed`  | `GET /ltrstchain/ltrstchain/privacy/nullifier/{nullifier}` | `{used: bool}`                                          |
| `MerkleRoot`     | `GET /ltrstchain/ltrstchain/privacy/merkle_root`           | `{root: 32 bytes}`                                      |

Commit and nullifier parameters are hex-encoded on the wire and must
decode to exactly 32 bytes; anything shorter returns `InvalidArgument`.
Not-found commitments return `found=false` rather than a gRPC error, so
clients can distinguish a missing commitment from a protocol error.

### Proto types

All generated from `proto/ltrstchain/ltrstchain/privacy.proto` into
`x/ltrstchain/types/privacy.pb.go`:

- **`ShieldedCommitment`** — commit, nullifier, field_commitments
  (`map<string, bytes>`), field_openings (`map<string, FieldOpening>`),
  protocol, submitter, height.
- **`FieldOpening`** — 16-byte salt, arbitrary-length value.
- **`MsgShieldedTransfer`** — `{submitter, commitment}` with
  `commitment` declared non-nullable.
- **`MsgShieldedTransferResponse`** — `{new_merkle_root, height}`.
- **`QueryCommitmentRequest/Response`**, **`QueryNullifierUsedRequest/Response`**,
  **`QueryMerkleRootRequest/Response`**.

### Validation helpers

Defined in `x/ltrstchain/types/privacy.go` as free functions (the generated
`ShieldedCommitment` is the authoritative data shape; no manual struct
declarations are needed):

- **`ValidateShieldedCommitment(*ShieldedCommitment)`** — static shape
  check + every opening hashes to its declared field commitment (constant-
  time comparison via `equalBytes`).
- **`HashFieldOpening(value, salt)`** — `SHA-256(value || salt)`, matches
  the wallet's `generateCommitment` field-commitment format.
- **`UpdateRollingRoot(prior, commit)`** — `SHA-256(prior || commit)`.
  Linear hash chain, not a full Merkle tree. Sufficient until a
  Poseidon-friendly tree is wired in.
- **`CommitmentKey`**, **`NullifierKey`**, **`MerkleRootKey`** — KV-store
  key builders.

### Storage layout

```
CommitmentPrefix = "pc/"   // pc/ || 32-byte commit    -> cdc.Marshal(ShieldedCommitment)
NullifierPrefix  = "pn/"   // pn/ || 32-byte nullifier -> {0x01}
MerkleRootPrefix = "pr/"   // pr/                      -> 32-byte root
```

### Protocol tag

Only commitments stamped with `LT-Shield-Commit-v2` are accepted. Matches
the wallet's `generateCommitment()` output (LT Shield Wallet v1.0.2+10).

## Test coverage

25 test cases covering every layer of the privacy module. All passing.

### `x/ltrstchain/keeper/privacy_test.go` — 7 keeper-level cases

1. **TestPrivacyAddCommitmentHappyPath** — stores, reads back, marks
   nullifier as used, merkle root advanced from zero.
2. **TestPrivacyRejectsDoubleSpend** — second commitment with same
   nullifier is rejected.
3. **TestPrivacyRejectsWrongProtocol** — non-`LT-Shield-Commit-v2`
   protocol tag is rejected.
4. **TestPrivacyRejectsShortCommit** — commit that isn't 32 bytes is
   rejected.
5. **TestPrivacyFieldOpeningValidation** — legit opening + matching
   field commitment is accepted.
6. **TestPrivacyFieldOpeningMismatchRejected** — opening whose
   `SHA-256(value || salt)` does not equal the declared field commitment
   is rejected.
7. **TestPrivacyRollingRootMatchesExpected** — two-commitment chain
   produces `H(H(zero || commitA) || commitB)`.

### `x/ltrstchain/keeper/msg_shielded_transfer_test.go` — 5 msg-server cases

1. **TestMsgShieldedTransferHappyPath** — MsgPrivacy handler stores the
   commitment, echoes back the new root, and the commitment is queryable.
2. **TestMsgShieldedTransferOverwritesSubmitter** — a client that plants a
   different submitter inside `ShieldedCommitment.Submitter` cannot spoof
   attribution; the tx signer always wins.
3. **TestMsgShieldedTransferRejectsInvalidSubmitter** — a non-bech32
   submitter is rejected before state is touched.
4. **TestMsgShieldedTransferRejectsDoubleSpend** — double-spend rejection
   is enforced through the msg-server path, not just the keeper.
5. **TestMsgShieldedTransferReturnsUpdatedRoot** — response's
   `NewMerkleRoot` matches `GetMerkleRoot` after the msg completes.

### `x/ltrstchain/keeper/query_privacy_test.go` — 7 query-server cases

1. **TestQueryPrivacyCommitmentHappyPath** — stored commitment is
   returned via hex-wire Commit with `Found=true`.
2. **TestQueryPrivacyCommitmentNotFound** — missing commitment returns
   `Found=false` with no gRPC error.
3. **TestQueryPrivacyCommitmentInvalidHex** — non-hex commit returns
   `codes.InvalidArgument`.
4. **TestQueryPrivacyCommitmentShortCommit** — hex that decodes to the
   wrong length returns `codes.InvalidArgument`.
5. **TestQueryPrivacyNullifierUsed** — seen nullifier returns true,
   unseen returns false.
6. **TestQueryPrivacyMerkleRootGenesis** — root before any commitment is
   32 zero bytes.
7. **TestQueryPrivacyMerkleRootAdvances** — root after a commitment has
   advanced past the zero root.

### `x/ltrstchain/module/genesis_privacy_test.go` — 6 init/export cases

1. **TestGenesisPrivacyInitEmpty** — InitGenesis with no commitments
   leaves the root at 32 zero bytes.
2. **TestGenesisPrivacyInitReplaysCommitments** — pre-existing
   commitments in the genesis file replay into keeper state with the
   correct rolling root (`H(H(zero || A) || B)`).
3. **TestGenesisPrivacyInitRejectsDuplicateNullifier** — InitGenesis
   panics when the genesis file declares two commitments sharing a
   nullifier, preventing boot into a double-spent state.
4. **TestGenesisPrivacyValidateRejectsDuplicateNullifier** — static
   `GenesisState.Validate()` also catches the duplicate before any
   state is touched.
5. **TestGenesisPrivacyValidateRejectsBadProtocol** — `Validate()`
   rejects any commitment whose protocol tag isn't
   `LT-Shield-Commit-v2`.
6. **TestGenesisPrivacyExportRoundTrip** — commitments added online
   round-trip through `ExportGenesis` → `InitGenesis` on a clean keeper
   and produce the same final root.

Run them with:

```
cd ltrst-chain-l1
COPYFILE_DISABLE=1 PATH=/tmp/go1.24/go/bin:$PATH \
  go test -count=1 -run 'TestPrivacy|TestMsgShielded|TestQueryPrivacy|TestGenesisPrivacy' -v \
    ./x/ltrstchain/keeper/... ./x/ltrstchain/module/...
```

## Alignment with the wallet (Layers 1, 2, 6)

| Wallet side                                   | Chain side                                     |
|-----------------------------------------------|------------------------------------------------|
| `generateCommitment()` → 32-byte `commitment` | `ShieldedCommitment.Commit` accepted as-is     |
| `generateCommitment()` → 32-byte `nullifier`  | `IsNullifierUsed` + double-spend rejection     |
| `fieldCommitments.amount / .sender / .receiver` | `ShieldedCommitment.FieldCommitments` map    |
| `opening.amount / .sender / .receiver`        | `FieldOpenings` map, hash-verified on ingress  |
| `protocol: 'LT-Shield-Commit-v2'`             | `CommitProtocol = "LT-Shield-Commit-v2"`       |

Both halves use `SHA-256(value || salt)` with 16-byte salts — the keeper
recomputes on every ingress and rejects mismatched openings.

## Known gaps (to close before a "privacy-complete" chain claim)

1. **Linear hash chain, not a full Merkle tree.** `UpdateRollingRoot`
   is a collision-resistant accumulator but does not support inclusion
   proofs. When zk-SNARK verification lands on-chain, upgrade to a
   Poseidon-friendly sparse tree.
2. **Private→private transfers do not yet move funds.** The keeper
   records the commitment, but `x/bank` balances stay transparent.
   Shielded-pool value flow requires wiring AddCommitment to a bank
   hook (or a dedicated shielded-pool module account).
3. **Ante handler / fee bypass prevention.** The current
   `MsgShieldedTransfer` is free beyond the base gas cost. When the
   shielded pool moves funds, add a minimum fee check so spam isn't free.

### Gaps closed on 2026-04-11

- ~~**Genesis initial-state.**~~ Resolved. `GenesisState.PrivacyCommitments`
  is now a first-class field; `Validate()` rejects bad protocol tags and
  duplicate nullifiers; `InitGenesis` replays every commitment through
  `AddCommitment` (reusing the same double-spend + opening-verify checks
  as the online path); `ExportGenesis` snapshots the full commitment set
  via `IterateCommitments`. Round-trip confirmed by
  `TestGenesisPrivacyExportRoundTrip`.

## How to regenerate proto after editing `privacy.proto`

The proto toolchain is installed under `/tmp/ltrst-bin/` (buf, gogo,
pulsar, grpc-gateway, openapiv2, grpc). To regenerate:

```
cd ltrst-chain-l1/proto
# remove macOS AppleDouble metadata files so buf doesn't choke
find . -name "._*" -delete
COPYFILE_DISABLE=1 PATH=/tmp/ltrst-bin:/tmp/go1.24/go/bin:$PATH \
  buf generate --template buf.gen.gogo.yaml
# buf writes into a shadow tree — copy the privacy files back out:
cp ltrstchain/x/ltrstchain/types/privacy.pb.go ../x/ltrstchain/types/
cp ltrstchain/x/ltrstchain/types/privacy.pb.gw.go ../x/ltrstchain/types/
# clean up the shadow tree so `go build ./...` doesn't pick it up:
rm -rf ltrstchain/x
rm -f ltrstchain/ltrstchain/module/module.pb.go
```

## Change log

- **2026-04-11** — Added `x/ltrstchain/types/privacy.go` (constants,
  validators, key builders, rolling root, hash helpers). Added
  `x/ltrstchain/keeper/privacy.go` (AddCommitment, IsNullifierUsed,
  GetCommitment, GetMerkleRoot). Added `privacy_test.go` with 7 tests,
  all passing. Installed buf + gogo + pulsar + grpc-gateway +
  protoc-gen-gocosmos under `/tmp/ltrst-bin/`. Generated
  `x/ltrstchain/types/privacy.pb.go` from
  `proto/ltrstchain/ltrstchain/privacy.proto`. Refactored manual
  Commitment/FieldOpening types into free functions over generated
  `ShieldedCommitment`. Implemented `MsgPrivacyServer.ShieldedTransfer`
  and `QueryPrivacyServer.{Commitment, NullifierUsed, MerkleRoot}`.
  Registered both services in `module.RegisterServices`. Built
  `ltrstchaind` binary (~130 MB) and verified `_MsgPrivacy_serviceDesc`,
  `_QueryPrivacy_serviceDesc`, and all privacy server methods are linked.
