# x/ltrstdex — LTRST Chain Spot CLOB Module

Status: draft 1 (2026-04-16) — Phase 1 = spot limit/market orderbook.

## 1. Purpose

`x/ltrstdex` turns LTRST Chain into a real decentralized exchange with
its own on-chain central limit orderbook (CLOB), rather than using
another chain's DEX. Users place buy/sell limit orders against spot
markets (e.g. `LTRST/USDC`); the chain matches them in-block with
deterministic price–time priority and settles each fill directly
through `x/bank`.

This is Phase 1. Perpetuals, leverage, liquidations, subaccounts and
insurance fund live in later phases.

The module name is `ltrstdex` (shortens nicely to `dex` in messages
and stays parallel to `ltrstchain`).

## 2. Non-goals for Phase 1

The purpose of being scoped to a single phase is that each phase
builds and ships before the next one starts. Phase 1 is deliberately
small:

- No perpetuals / futures / options — spot only.
- No leverage, no liquidations, no insurance fund.
- No subaccounts — an order owner is a normal bech32 `ltrst…`
  account; collateral is locked and settled directly on the account
  balance via `x/bank`.
- No cross-collateral / portfolio margin.
- No market-maker rebates, no taker-per-tier fees, no affiliate
  rev-share — a single flat `maker_fee_bps` / `taker_fee_bps` per
  market.
- No gasless short-term orders — every order is a stateful order
  persisted in chain state.
- No indexer / full-node streaming surface — standard gRPC query
  service is the sole read API.
- Market creation is permissioned — only `x/gov` may create or pause
  a market in Phase 1 (prevents ticker-squatting during bootstrap).
  A permissionless factory can come after fee plumbing stabilizes.
- No IBC/bridging logic in this module — bridged assets arrive as
  normal `x/bank` coins (e.g. `ibc/…` denoms or `noble-uusdc`). The
  module does not know or care how a token got here.

## 3. Design principles

1. **Deterministic.** All matching happens inside `MsgPlaceSpotOrder`
   handling. No off-chain matching. No node-local state used in
   consensus paths. Two validators running the same binary against
   the same state must produce the same match set.
2. **Immediate matching.** When a taker order arrives, we walk the
   opposite book top-of-book first and fill until the order is
   exhausted, expires per TIF, or the book dries up. Anything left
   over (for GTC) is rested on the book in the same message handler.
   No end-of-block matching round.
3. **Stateful only.** There is no in-memory-only "short-term" order
   concept. Every order is stored in the KV store and gets a
   deterministic `order_id`. This is simpler than dYdX's hybrid
   design and fine for our expected order rate.
4. **x/bank as the settlement layer.** Collateral is locked by
   sending the relevant coins from the owner to the `ltrstdex`
   module account at order placement. On match, the module account
   pays out to the counterparty. Any taker fee is routed to the
   `fee_collector` module account (which distribution already reads
   from), not to a dex-specific pool. Unfilled / cancelled amounts
   return to the owner.
5. **Two-keeper surface.** Outside SDK-standard plumbing, the keeper
   only needs `AccountKeeper` (to resolve module account addresses)
   and `BankKeeper` (to move funds). No perps/pricing/stats/
   affiliates/subaccount keepers — those live in later phases if we
   add derivatives.
6. **Decimal discipline.** All prices, quantities and fee fractions
   are `cosmossdk.io/math.LegacyDec` serialized as strings so proto
   round-trips are exact and comparisons in Go code never float.
7. **Privacy keeper coexists.** The existing `x/ltrstchain` privacy
   module (shield commitments, nullifiers, Merkle root) stays
   untouched. The DEX worker at `dex.larrytrustee.ai` already
   mirrors `/privacy/{commit,commitment,nullifier_used,merkle_root}`
   1:1 against the chain; that contract is unchanged. Nothing in
   `x/ltrstdex` reads or writes the shield pool. A later phase can
   add a shielded-order variant if needed.

## 4. Storage layout

All prefixes are scoped to the `ltrstdex` store. The keeper holds
`storetypes.StoreKey` (KV) only — no memstore, no transient store.

```
# Module-level
p                                              → Params
m/next                                         → uint64  (next market id)
o/next/{marketID}                              → uint64  (next per-market seq)

# Markets
m/{marketID:u64be}                             → SpotMarket
m/by_ticker/{ticker}                           → {marketID:u64be}

# Orders (authoritative record)
o/{marketID:u64be}/{orderID}                   → SpotOrder

# Orderbook index (sorted walk — all we need to find top-of-book)
# Buy side sorts price DESC (highest bid first).
# Sell side sorts price ASC  (lowest ask first).
# Within a price level: seq ASC (older orders first = time priority).
# {price:20-byte big-endian dec}  — we negate for buy side so DESC works with ASC iterator.
ob/{marketID:u64be}/b/{negPrice:20}/{seq:u64be}/{orderID}   → orderID (value ignored, key carries info)
ob/{marketID:u64be}/a/{price:20}/{seq:u64be}/{orderID}      → orderID

# Owner index (for cancel-by-owner and UI)
u/{owner:bytes}/{orderID}                      → {marketID:u64be}

# Fill history (optional Phase 1, useful for UI; capped or trimmed later)
f/{marketID:u64be}/{blockHeight:u64be}/{fillSeq:u64be}  → SpotFill
```

Rationale:

- The orderbook index (`ob/…`) is our match-engine accelerator. We
  never scan all orders for a market; we iterate `ob/{mid}/b/` or
  `ob/{mid}/a/` in key order and that gives us price-time priority
  directly. Deleting an order is `ob/…/{orderID}` by key.
- Fill history at `f/…` is a write-only ledger. Phase 1 keeps all
  fills; Phase 1.1 can add `MaxFillsPerMarket` param and evict.
- Ticker index (`m/by_ticker/…`) protects against duplicate tickers
  and lets queries resolve a human-readable pair to a market ID.

## 5. Proto surface

### 5.1 Core domain (`spot.proto`)

```proto
enum Side         { SIDE_UNSPECIFIED=0; SIDE_BUY=1; SIDE_SELL=2; }
enum OrderType    { ORDER_TYPE_UNSPECIFIED=0; ORDER_TYPE_LIMIT=1; ORDER_TYPE_MARKET=2; }
enum TimeInForce  { TIF_GTC=0; TIF_IOC=1; TIF_FOK=2; }
enum MarketStatus {
  MARKET_STATUS_UNSPECIFIED = 0;
  MARKET_STATUS_PRE_OPEN    = 1;  // listed, orders rejected
  MARKET_STATUS_ACTIVE      = 2;  // normal trading
  MARKET_STATUS_POST_ONLY   = 3;  // only maker orders accepted
  MARKET_STATUS_PAUSED      = 4;  // no new orders; existing orders stay
  MARKET_STATUS_HALTED      = 5;  // no new orders; existing orders auto-cancelled next block
}

message SpotMarket {
  uint64 id           = 1;
  string ticker       = 2;        // display "LTRST/USDC"
  string base_denom   = 3;        // e.g. "ultrst"
  string quote_denom  = 4;        // e.g. "ibc/HASH-for-usdc"
  string min_price_tick = 5;      // LegacyDec
  string min_quantity   = 6;      // LegacyDec (in base units)
  uint32 maker_fee_bps  = 7;      // 1 bps = 0.01 %
  uint32 taker_fee_bps  = 8;
  MarketStatus status = 9;
  int64  created_at_block = 10;   // block height at creation
}

message SpotOrder {
  string order_id = 1;            // hash(owner || market || seq)
  bytes  owner    = 2;            // sdk.AccAddress bytes
  uint64 market_id = 3;
  Side        side = 4;
  OrderType   type = 5;
  TimeInForce tif  = 6;
  string price             = 7;   // LegacyDec (ignored for MARKET)
  string quantity          = 8;   // LegacyDec, base units
  string filled_quantity   = 9;   // LegacyDec, base units
  string locked_amount     = 10;  // LegacyDec, locked in the quote denom (BUY) or base denom (SELL)
  int64  created_height    = 11;
  uint64 seq               = 12;  // per-market monotonic ordering key
  int64  expires_at_block  = 13;  // 0 = never
}

message SpotFill {
  uint64 fill_seq       = 1;
  uint64 market_id      = 2;
  string taker_order_id = 3;
  string maker_order_id = 4;
  bytes  taker          = 5;
  bytes  maker          = 6;
  Side   taker_side     = 7;
  string price          = 8;      // LegacyDec, quote per base
  string quantity       = 9;      // LegacyDec, base units
  string taker_fee      = 10;     // quote units paid by taker
  string maker_fee      = 11;     // quote units paid (or refunded, if negative) by maker — 0 in Phase 1
  int64  block_height   = 12;
}
```

### 5.2 Params (`params.proto`)

```proto
message Params {
  // Maximum orders an account may have open across all markets at once.
  // Prevents state bloat without a global fee on open orders.
  uint32 max_open_orders_per_account = 1;
  // Governance-only market creation toggle. Phase 1 = true.
  bool   gov_only_market_creation   = 2;
  // How many blocks of fill history we retain per market. 0 = keep all.
  uint32 fill_retention_blocks      = 3;
}
```

### 5.3 Messages (`tx.proto`)

```
MsgCreateSpotMarket      (authority: gov in phase 1)
MsgUpdateSpotMarketStatus(authority: gov)
MsgUpdateParams          (authority: gov)
MsgPlaceSpotOrder        (authority: owner)
MsgCancelSpotOrder       (authority: owner OR gov)
MsgCancelAllSpotOrders   (authority: owner; optional market filter)
```

Validation notes:

- `MsgPlaceSpotOrder` verifies market exists and is in
  `ACTIVE` or `POST_ONLY`; if `POST_ONLY`, taker paths reject.
- `price` must be a positive multiple of `min_price_tick`.
- `quantity` must be a positive multiple of `min_quantity`.
- For MARKET orders `price` MUST be empty; match walks entire book
  up to available liquidity. Unmatched portion of a MARKET is dropped
  regardless of TIF (treated as IOC).
- For FOK, if the book can't completely fill the order at placement
  time, the whole order is rejected atomically with no state change.

### 5.4 Queries (`query.proto`)

```
Params              () → Params
SpotMarket          (market_id) → SpotMarket
SpotMarketByTicker  (ticker)    → SpotMarket
SpotMarkets         ()          → []SpotMarket
SpotOrder           (market_id, order_id) → SpotOrder
SpotOrdersByOwner   (owner, pagination)   → []SpotOrder
SpotOrdersByMarket  (market_id, side, pagination) → []SpotOrder
SpotOrderbook       (market_id, depth) → {bids: []Level, asks: []Level}
SpotFills           (market_id, pagination, block_range?) → []SpotFill
```

`Level` is `{price, total_base_quantity}` aggregated from `ob/…`
iteration.

## 6. Matching algorithm (reference pseudocode)

```
placeOrder(order):
  validateStaticly(order)
  market = getMarket(order.market_id)
  requireStatusAcceptsOrders(market, order)

  locked = computeLockAmount(order, market)           # BUY: price*qty in quote; SELL: qty in base
  k.bank.SendCoinsFromAccountToModule(owner, ModuleName, locked)

  if order.type == MARKET or (order.type == LIMIT and crossesBook(order, market)):
    filledQty, paid, received = walkOpposingBookAndFill(order, market)
    order.filled_quantity += filledQty
    # taker fee on value received (BUY) or value paid out to taker (SELL)
    takerFee = round(tradedQuote * market.taker_fee_bps / 10000)
    payFee(takerFee)                                   # to fee_collector

  remainingQty = order.quantity - order.filled_quantity
  if remainingQty == 0 or order.tif == IOC or order.type == MARKET:
    refundIfAny(order, locked, paid)
    if remainingQty == 0: markFilledAndPersistHistory(order)
    else:                 markCancelledAndRefund(order, remainingQty)
    return

  if order.tif == FOK and remainingQty > 0:
    # Spec says we can't even START if FOK won't complete.
    # We rollback by refunding locked and NOT persisting the order.
    refund(locked)
    fail("FOK unable to fill completely")

  # GTC remainder rests
  persistOrder(order)
  insertIntoOrderbookIndex(order)
```

Maker-vs-taker determination:

- In a single `MsgPlaceSpotOrder` handler invocation, the incoming
  order is the taker against every resting order it consumes.
  Resting orders are makers.
- Taker fee is charged on the taker leg only. Maker fee is 0 in
  Phase 1 (the proto field is kept so we can turn it on without a
  migration). This mirrors the bootstrap incentive policy we've
  already locked for the fleet bots.

Fill pricing:

- Each maker fill prices at the maker's resting price, not the
  taker's limit price (standard CLOB behaviour).
- Taker pays / receives at the maker price; any improvement vs the
  taker's limit is captured by the taker.

## 7. Genesis

`GenesisState` contains:

- `params`
- `markets` — full SpotMarket list
- `orders` — full SpotOrder list (gets re-inserted into the
  orderbook index on `InitGenesis`)
- `next_market_id`, `next_order_seq_by_market` so seq numbers don't
  restart at 0 and collide with replayed IDs.

`ExportGenesis` dumps all three in a form `InitGenesis` can round-trip
deterministically.

Phase 1 ships `DefaultGenesis` with:

- `params.max_open_orders_per_account = 200`
- `params.gov_only_market_creation = true`
- `params.fill_retention_blocks = 604800`  (≈2 weeks at 2s blocks)
- `markets = []`
- `orders = []`

## 8. Keeper dependencies

```go
type AccountKeeper interface {
    GetModuleAddress(name string) sdk.AccAddress
    GetModuleAccount(ctx context.Context, name string) sdk.ModuleAccountI
}

type BankKeeper interface {
    SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
    SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
    SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
    SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}
```

That's it. Two interfaces. Compare this to dYdX's keeper which pulls
in 11 external keepers — the surface area difference is the point:
we're a spot book on top of x/bank, not a margin exchange.

## 9. Fees

- `maker_fee_bps` = 0 (Phase 1)
- `taker_fee_bps` = 10 (= 0.10 %) — configurable per-market at
  creation time by gov.
- Taker fee is transferred from the dex module account to the SDK
  `auth` fee_collector module account in the matched quote denom.
  `x/distribution` already disburses fee_collector balance to
  validators and delegators per staking rewards rules, so we piggy-
  back on that without wiring a dex-specific distributor.
- There are no exchange-side fee discounts, no referral/affiliate
  split, and no maker rebates in Phase 1. (Founder-side economics
  happen off-chain per existing `feedback_no_fee_mentions.md` policy.)

## 10. Module account + permissions

- Module name: `ltrstdex`
- Module account permissions: none (can receive coins from users but
  cannot mint or burn). Registered in `moduleAccPerms` in
  `app/app_config.go`. The account is NOT added to `blockAccAddrs` —
  it must be able to receive from users' transfers when they lock
  collateral.

## 11. App wiring checklist

- `proto/ltrstchain/ltrstdex/module/module.proto` defines the
  depinject `Module` config with
  `go_import: "ltrstchain/x/ltrstdex"`.
- Side-effect import in `app/app_config.go`:
  `_ "ltrstchain/x/ltrstdex/module"`.
- `ltrstdexmodulev1` alias imported from the generated
  `ltrstchain/api/ltrstchain/ltrstdex/module` path.
- Added to `genesisModuleOrder`, `beginBlockers`, `endBlockers`
  (endBlocker only — expires/cancels orders past `expires_at_block`).
- Added to `appConfig.Modules` with `Name: ltrstdextypes.ModuleName`.

## 12. Future phases

- Phase 2: bridge plumbing so `ultrst`/`uusdc` both exist liquidly —
  Noble IBC channel for USDC, Axelar for ETH/BTC/USDT, Wormhole
  Gateway for SOL/AVAX. No module changes, just IBC channels + docs.
- Phase 3: cosmpy client library replacing ccxt — the bot fleet
  submits `MsgPlaceSpotOrder` / `MsgCancelSpotOrder` directly instead
  of hitting CEX relays.
- Phase 4: perpetuals module (`x/ltrstperp`) reusing the same
  matching engine shape but with collateral, PnL, funding, insurance
  fund, liquidations. Launches as a separate module next to
  `x/ltrstdex` so the spot module stays simple.
- Phase 5: permissionless market factory once fee accrual and
  governance norms are settled. Gov-only flag in Params flips to
  false via `MsgUpdateParams`.
