# LTRST Chain — The AI Blockchain for Legal, Financial, and Medical AI-Generated Projects and Tools

LTRST Chain (`ltrst-chain-1`) is a sovereign Layer-1 blockchain purpose-built to host **legal, financial, and medical AI-generated projects and tools** with cryptographic privacy guarantees at the protocol level. It is the source-of-truth ledger for the [LarryTrustee.AI](https://larrytrustee.ai) platform.

This repository contains the source code for the `ltrstchaind` validator binary.

## Why a Purpose-Built Chain

General-purpose blockchains were not designed for AI workloads that handle regulated personal information. LTRST Chain was. Its protocol-level privacy primitives — shielded commitments, a nullifier set, and a rolling Merkle accumulator — allow an AI model to process a client's legal trust, financial estate plan, or medical record while keeping the underlying payload off any server that could be subpoenaed, breached, or quietly mined.

The chain is built to host AI-generated projects and tools across three regulated domains:

- **Legal** — Living trusts, wills, beneficiary designations, and probate filings, generated and assembled by AI; AI-assisted contract drafting; signing-likeness verification for deepfake-era authentication of executed documents.
- **Financial** — On-chain estate-planning instruments, vesting and trust-account commitments, cryptographically anchored beneficiary designations, family-legacy asset transfer, and programmable financial directives intended to outlive their grantor.
- **Medical** — HIPAA-aligned medical-record commitments, AI-guided care directives, cryptographically anchored patient consent for AI-driven diagnosis or treatment recommendation, and family medical history preserved across generations.

Each domain is represented by a live consumer product on the [LarryTrustee.AI](https://larrytrustee.ai) platform, all anchored to this chain.

## Chain Identity

| Field | Value |
|---|---|
| Chain ID | `ltrst-chain-1` |
| Network | Mainnet (live since 2026-04-11) |
| Native denom | `ultrst` (1 LTRST = 1,000,000 ultrst) |
| Max supply | 221,000,000 LTRST (fixed, no inflation) |
| Bech32 prefix | `ltrst` / `ltrstvaloper` / `ltrstvalcons` |
| SLIP-0044 | 118 (Cosmos standard) |
| Block time | ~2 seconds |
| Cosmos SDK | 0.50 |
| CometBFT | 0.38.17 |
| IBC-go | 8.5.2 |

## Public Endpoints

| Service | URL |
|---|---|
| RPC (Tendermint) | `https://chain.larrytrustee.ai/trpc/` |
| REST (Cosmos) | `https://chain.larrytrustee.ai/tapi/` |
| Genesis file | `https://chain.larrytrustee.ai/trpc/genesis` |
| Explorer | `https://dex.larrytrustee.ai/explorer` |
| Cosmos chain-registry | [cosmos/chain-registry/ltrstchain](https://github.com/cosmos/chain-registry/tree/master/ltrstchain) |

## Modules

In addition to the standard Cosmos SDK modules (`auth`, `bank`, `staking`, `gov`, `distribution`, `slashing`, `mint`, `crisis`, `evidence`, `feegrant`, `ibc`, `ica`, `transfer`), LTRST Chain provides three first-party modules:

- **`x/ltrstchain`** — Privacy module providing shielded-transfer commitments, a nullifier set, and a rolling Merkle root. AI-generated tools record legal, financial, and medical commitments on-chain without exposing the underlying payload to validators. See [`x/ltrstchain/PRIVACY_MODULE.md`](x/ltrstchain/PRIVACY_MODULE.md).
- **`x/ltrstdex`** — Phase 1 design for an on-chain spot central-limit order book with deterministic price-time priority matching. See [`x/ltrstdex/DESIGN.md`](x/ltrstdex/DESIGN.md).
- **`x/bridge`** — IBC asset registry providing governance-gated registration of bridged assets entering the chain through IBC channels.

## Status

LTRST Chain is currently in **Phase 1**. Mainnet is live and producing blocks. The DEX at [dex.larrytrustee.ai](https://dex.larrytrustee.ai) operates in bootstrap mode with issuer-operated market-maker liquidity, and a formal public token sale is scheduled to follow listings on CoinMarketCap and CoinGecko. Until those listings are in place, the chain operates as a builder and developer environment for legal, financial, and medical AI-generated projects and tools.

## Build from Source

```bash
git clone https://github.com/Larrytrustee/ltrst-chain-l1.git
cd ltrst-chain-l1
make build       # produces build/ltrstchaind
```

Requires Go 1.25+.

## Run a Full Node

See [`join-node.sh`](join-node.sh) for an idempotent Ubuntu 22.04 / Debian 12 bootstrap script.

```bash
sudo ./join-node.sh --moniker my-node-name
```

To run a validator, append `--validator` and follow the printed `gentx` instructions.

## Verify Chain Integrity

```bash
python3 verify-chain.py
```

Read-only audit against the public REST endpoint. Confirms the 8 genesis allocations, the 221M LTRST hard cap, zero inflation, and governance parameters. The latest live audit is recorded in [`integrity-proof-20260411.txt`](integrity-proof-20260411.txt).

## Genesis Allocation

The whitepaper allocation, enforced in `config.yml`:

| Allocation | Share | Amount |
|---|---|---|
| Public sale | 15% | 33,150,000 LTRST |
| Staking rewards | 25% | 55,250,000 LTRST |
| Ecosystem development | 20% | 44,200,000 LTRST |
| Community rewards | 15% | 33,150,000 LTRST |
| Reserve / founders | 10% | 22,100,000 LTRST |
| DAO treasury | 5% | 11,050,000 LTRST |
| Liquidity pool | 10% | 22,100,000 LTRST |
| Genesis validator | — | 500,000 LTRST |
| **Total** | **100%** | **221,000,000 LTRST** |

## Roadmap

- **Phase 1 (live)** — privacy commitments, sovereign chain, single-validator bootstrap, AI-tool integration for legal, financial, and medical workloads.
- **Phase 2 (active)** — multi-validator decentralization, DEX (`x/ltrstdex`) Phase 1 spot CLOB, third-party exchange listings, formal public token sale.
- **Phase 3 (Q4 2026, design only)** — Ethermint EVM-compatibility layer behind a governance-gated upgrade. See [`docs/ETHERMINT_INTEGRATION_PLAN.md`](docs/ETHERMINT_INTEGRATION_PLAN.md).

## License

Apache License 2.0 — see [`LICENSE`](LICENSE).

## Contact

LarryTrustee.AI Inc. — [larrytrustee.ai](https://larrytrustee.ai)
