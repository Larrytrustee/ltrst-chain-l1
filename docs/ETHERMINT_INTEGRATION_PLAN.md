# LTRST Chain™ — Ethermint Integration Plan

**Status:** Design-stage, not yet implemented
**Target activation:** Q4 2026 (post external security audit, post 21-validator decentralization)
**Branch:** `ethermint-staging` (separate from active `dex-fork-v1`)
**Version of this doc:** v1.0 — 2026-04-20
**Maintainer:** Larry Trustee AI Inc.

---

## 1. Executive Summary

LTRST Chain™ is a pure Cosmos SDK v0.50 Layer-1 blockchain with CometBFT v0.38.17 consensus. This document describes the planned integration of the **Ethermint EVM-compatibility module** into the chain, following the precedent set by Evmos, Kava, Injective, and Cronos.

The goal is to enable Solidity-contract deployment and Ethereum-JSON-RPC access **natively on LTRST Chain**, without introducing any cross-chain bridge. EVM execution will share the same consensus, state, and finality guarantees as the existing Cosmos SDK modules.

This is a **planned** integration — no Ethermint code has been merged to mainnet-bound branches. Activation requires a governance-gated chain upgrade.

## 2. Architectural Target

### 2.1 Current state (not EVM-compatible)

- Consensus: CometBFT 0.38.17 (BFT, 2-second finality)
- Cosmos SDK 0.50.14
- Address format: bech32 `ltrst1...` (20-byte accounts)
- Key derivation: BIP-44 coin-type 118 (Cosmos standard)
- Native denom: `ultrst` (6 decimals)
- RPC: Tendermint RPC port 26657, Cosmos REST port 1317
- No Ethereum JSON-RPC endpoints exposed
- `eth_chainId` and related methods return `-32601 Method not found`

### 2.2 Target state (EVM-compatible via Ethermint)

- Consensus: unchanged (CometBFT 0.38.17)
- Cosmos SDK 0.50.14 with Ethermint EVM module co-located
- **Dual address format**: `ltrst1...` bech32 continues to work; `0x...` hex addresses supported for EVM accounts, with deterministic mapping between the two
- **Dual key derivation**: Cosmos coin-type 118 AND Ethereum coin-type 60 both valid
- Native denom unchanged (`ultrst`)
- **Dual RPC**: Tendermint RPC (26657) AND Ethereum JSON-RPC (8545) both served
- Solidity contracts deployable through `eth_sendTransaction`
- CosmWasm + EVM contracts share the same consensus layer and can call each other through adapter precompiles

## 3. Dependency Selection

Two candidate implementations, both actively maintained against Cosmos SDK v0.50:

| Candidate | Source | Notes |
|---|---|---|
| `github.com/cosmos/evm` | Official Cosmos project | Cleaner module boundaries, actively developed |
| `github.com/evmos/os` | Evmos-fork successor | Production-proven, closer to Ethermint legacy |

**Decision**: target `github.com/cosmos/evm` as primary, with `evmos/os` as fallback if module compatibility issues surface during test-fork phase.

## 4. Module Integration Points

The following files in `ltrst-chain-l1/` will require modification:

- `go.mod` — add `github.com/cosmos/evm` dependency
- `app/app.go` — register EVM keeper, wire module.Manager, add AnteHandler for EVM tx
- `app/app_config.go` — add EVM module config entries
- `cmd/ltrstchaind/root.go` — register EVM JSON-RPC server startup flag
- `app/genesis.go` — add EVM module initial-state seeding for genesis-upgrade path
- New file: `app/evm_hooks.go` — bridge hooks between `bank` keeper and EVM state for shared `ultrst` balance

All changes will be behind a `UPGRADE_NAME_EVMINT_V1` governance upgrade handler, so the binary can run in **either** mode (EVM-off, matching current mainnet) **or** EVM-on (post-upgrade) depending on chain height.

## 5. Key Derivation Compatibility

The critical compatibility question: existing `ltrst1...` accounts must not be disrupted by the upgrade.

### 5.1 Plan

- **Existing accounts stay valid.** All `ltrst1...` addresses continue to work unchanged. Their balances are preserved.
- **New EVM accounts derive deterministically** from the same seed phrase using BIP-44 coin-type 60, producing a `0x...` address alongside any existing `ltrst1...` address.
- Adapter logic maps `0x...` → `ltrst1...` so the EVM module and bank module see the same balance for cross-addressed users.

### 5.2 Risk: state collisions

Edge case where an EVM-derived `ltrst1...` address collides with an existing `ltrst1...` account created under coin-type 118. Probability ~1/2^160 per address — cryptographically negligible but flagged in the risk register.

## 6. State Migration

No state migration is required for existing `ultrst` balances, validators, staking, or DAO treasury. The EVM module adds NEW state (contract code, EVM storage trie) alongside existing Cosmos state.

## 7. Governance Upgrade Procedure

1. Core team prepares `ltrstchaind v2.0.0-evmint` binary from `ethermint-staging` branch
2. Test-net fork: spin up a private 3-validator devnet, upgrade to v2.0.0 at height N, run for 7 days under realistic load
3. External audit firm reviews the upgrade diff
4. Governance proposal submitted: `UPGRADE_NAME_EVMINT_V1` with upgrade height set 14 days out
5. Proposal deposit, voting period (14 days), quorum (10%), pass threshold (50% + 33% no-veto)
6. On-pass: all validators upgrade binary, chain halts at upgrade height, resumes with EVM live

## 8. Test Plan

Before the governance proposal:

- Unit tests for EVM keeper integration (goal: >85% coverage on new code)
- Devnet fork upgrade simulation (7-day soak)
- External security audit of the diff
- Bug bounty window on the upgraded devnet (14 days minimum)
- MetaMask integration smoke test
- CosmWasm ↔ EVM contract cross-call integration test

## 9. Timeline

| Phase | Target | Dependency |
|---|---|---|
| Design doc (this file) | ✅ 2026-04-20 | — |
| `ethermint-staging` branch with scaffolded module | Q3 2026 | External security audit of current chain complete |
| Devnet fork + 7-day soak | Q3 2026 | Branch ready |
| Audit of upgrade diff | Q4 2026 | Devnet soak clean |
| Governance proposal | Q4 2026 | Audit clean |
| Mainnet upgrade | Q4 2026 | Governance pass + 21 validators |
| Post-upgrade: dApp onboarding docs, MetaMask support | Q1 2027 | Mainnet upgrade complete |

**This is not a commitment; it is a target contingent on dependencies above.**

## 10. Why Bridge-Free

The native-integration choice is deliberate. Cross-chain bridges are the single largest attack vector in crypto — over $2 billion stolen from Ronin, Wormhole, Nomad, Harmony, and others in the 2022–2025 cycle. By co-locating EVM execution on LTRST Chain's own consensus, users gain Ethereum compatibility without the trust assumption of a bridge operator or multisig.

Bridge-free multi-VM architecture on a single consensus layer is the long-term direction. This document commits to the EVM half of that architecture. Solana-VM (SVM) co-location is under separate research evaluation and has no committed activation date.

## 11. What This Document Does NOT Authorize

- No modifications to the running `ltrstchaind` binary
- No edits to the `dex-fork-v1` branch
- No re-deployment of the validator at `45.32.222.33`
- No changes to Ethereum-JSON-RPC exposure on the live chain

This document is a **plan**. Execution of any step above requires explicit operator authorization.

---

*Drafted 2026-04-20 as audit preparatory material.*
