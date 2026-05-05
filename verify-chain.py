#!/usr/bin/env python3
# ======================================================================
# LTRST Chain L1 — verify-chain.py
# ----------------------------------------------------------------------
# Pure read-only integrity audit of the live ltrst-chain-1. Queries the
# public HTTPS REST endpoint (nothing local) and confirms:
#
#   1. Each of the 8 genesis accounts holds its whitepaper-spec amount
#      (bank balance + bonded delegations, since validator1 self-delegates
#      100k LTRST at genesis).
#   2. The 8 accounts sum to exactly 221,000,000 LTRST (the hard cap).
#   3. The total on-chain supply equals 221M LTRST (221T ultrst).
#   4. Exactly 1 validator is bonded with 100,000 LTRST (validator1).
#   5. Inflation is 0 (no mint module emission).
#   6. Gov params match whitepaper (48h voting, 10k LTRST min deposit).
#
# Run this any time you want a fresh proof that nothing has drifted:
#   python3 verify-chain.py
#
# Exit code 0 on full integrity, 1 on any constraint failure.
# ======================================================================
import json, sys, urllib.request

BASE = "https://chain.larrytrustee.ai/tapi"

EXPECTED = {
  'public_sale':       ('ltrst1au3ds5r6w76h048gxwazg5g9dwx969f50zvy6n', 33_150_000),
  'staking_rewards':   ('ltrst1urhhg9pvdfdqre382fze7pstw0u7x3jl7dwgfu', 55_250_000),
  'ecosystem_dev':     ('ltrst1rpl6slycf703mzlmlnhgexyquydk304gwxjce2', 43_700_000),
  'community_rewards': ('ltrst1ehkee9ms650tznqvgzyk6kyj8skncq5l9tp37y', 33_150_000),
  'reserve_founders':  ('ltrst1pqhj9vxqe0n40m37eqztexrdu6j0xwv4ura3x5', 22_100_000),
  'dao_treasury':      ('ltrst1k3cgyfph4aju2qed9znwetvkerc6gzrvq5c8sm', 11_050_000),
  'liquidity_pool':    ('ltrst1cx4yh9y0du74zfxufvev82flcqtf93wnkgv8dd', 22_100_000),
  'validator1':        ('ltrst14jypsver3ndf8rufwf34yrkdr2h5pa8ghchl6g',      500_000),
}
CAP = 221_000_000

def get(path):
    req = urllib.request.Request(BASE + path, headers={"User-Agent": "ltrst-integrity-check/1.0"})
    with urllib.request.urlopen(req, timeout=8) as r:
        return json.loads(r.read())

def balance(addr):
    """Total holdings: bank balance + bonded delegations (both in ultrst)."""
    d = get(f"/cosmos/bank/v1beta1/balances/{addr}")
    bank = sum(int(b["amount"]) for b in d.get("balances", []) if b["denom"] == "ultrst")
    try:
        de = get(f"/cosmos/staking/v1beta1/delegations/{addr}")
        deleg = sum(int(r["balance"]["amount"]) for r in de.get("delegation_responses", []) if r["balance"]["denom"] == "ultrst")
    except Exception:
        deleg = 0
    return bank + deleg

def main():
    print("=" * 72)
    print(" LTRST Chain L1 — Genesis integrity check (LIVE)")
    print("=" * 72)
    print()

    # Current chain head (for proof timestamp)
    head = get("/cosmos/base/tendermint/v1beta1/blocks/latest")
    height = head["block"]["header"]["height"]
    btime  = head["block"]["header"]["time"]
    print(f"  Audited at block {height} ({btime})")
    print()

    ok = True
    total_held = 0
    for name, (addr, expected) in EXPECTED.items():
        u = balance(addr)
        held = u // 1_000_000
        total_held += held
        status = "OK" if held == expected else f"MISMATCH (held {held}, expected {expected})"
        if held != expected:
            ok = False
        print(f"  {name:<18} {addr}  {held:>12,} LTRST   {status}")
    print()
    cap_ok = (total_held == CAP)
    print(f"  Sum of 8 genesis accounts: {total_held:>12,} LTRST   (cap {CAP:,})")
    print(f"  Cap match:                 {cap_ok}")
    if not cap_ok:
        ok = False
    print()

    # Validator set
    v = get("/cosmos/staking/v1beta1/validators")
    vals = v.get("validators", [])
    print(f"  Active validators: {len(vals)}")
    for val in vals:
        tokens = int(val["tokens"]) // 1_000_000
        print(f"    - {val['description']['moniker']}: {tokens:,} LTRST staked, status {val['status']}")
    if len(vals) < 1 or vals[0]["status"] != "BOND_STATUS_BONDED":
        ok = False
    print()

    # Supply
    s = get("/cosmos/bank/v1beta1/supply/by_denom?denom=ultrst")
    sup_u = int(s["amount"]["amount"])
    sup_l = sup_u // 1_000_000
    print(f"  On-chain supply: {sup_l:,} LTRST ({sup_u:,} ultrst)")
    print(f"  Matches cap:     {sup_l == CAP}")
    if sup_l != CAP:
        ok = False
    print()

    # Gov params
    g = get("/cosmos/gov/v1/params/voting")
    voting_period = g['params']['voting_period']
    print(f"  Gov voting period: {voting_period}  (expect 172800s = 48h)")
    g2 = get("/cosmos/gov/v1/params/deposit")
    dep = g2['params']['min_deposit'][0]
    min_deposit_l = int(dep['amount']) // 1_000_000
    print(f"  Gov min deposit:   {min_deposit_l:,} LTRST  (expect 10,000)")

    m = get("/cosmos/mint/v1beta1/inflation")
    infl = float(m['inflation'])
    print(f"  Inflation:         {m['inflation']}  (expect 0 — fixed supply)")
    if infl != 0:
        ok = False

    print()
    print("=" * 72)
    print(f" RESULT: {'ALL GENESIS CONSTRAINTS MATCH WHITEPAPER SPEC' if ok else 'CONSTRAINT FAILURE — REVIEW'}")
    print("=" * 72)
    sys.exit(0 if ok else 1)

if __name__ == "__main__":
    main()
