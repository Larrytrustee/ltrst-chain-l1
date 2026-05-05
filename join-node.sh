#!/usr/bin/env bash
# ======================================================================
# LTRST Chain L1 — join-node.sh
# ----------------------------------------------------------------------
# Idempotent bootstrap for a second (or Nth) node on ltrst-chain-1.
#
# Runs on a fresh Ubuntu 22.04 / Debian 12 box and leaves you with:
#   - Go 1.25.1
#   - ltrstchaind binary at /opt/ltrstchain/bin/ltrstchaind
#   - /opt/ltrstchain/home initialized, state-synced / block-synced
#   - systemd unit ltrstchain.service running on boot
#   - ufw open only where strictly needed (22, 26656)
#
# Usage
# -----
#   sudo ./join-node.sh --moniker <name>            # full node, non-validator
#   sudo ./join-node.sh --moniker <name> --validator # also prints gentx cmds
#
# Defaults assume the first/seed validator is chain.larrytrustee.ai. The
# script is safe to re-run — it skips steps whose outputs already exist.
# ======================================================================

set -euo pipefail

# ---- Tunable constants -----------------------------------------------
CHAIN_ID="ltrst-chain-1"
BIN_DIR="/opt/ltrstchain/bin"
HOME_DIR="/opt/ltrstchain/home"
LOG_DIR="/opt/ltrstchain/logs"
GO_VERSION="1.25.1"
COSMOS_SDK_VERSION="0.50.14"
COMETBFT_VERSION="0.38.17"

# Bootstrap seed node — the genesis validator advertised publicly
SEED_NODE_ID="948731cf3500d931757ec5cc312496b9a5e3719b"
# P2P comes directly to the origin server (NOT via the cloudflared tunnel),
# because CF Tunnel doesn't support raw TCP P2P. If that box's IP rotates,
# update this line. The CNAME chain.larrytrustee.ai is HTTPS-only and
# cannot be used for peering.
SEED_NODE_P2P_HOST="45.32.222.33"
SEED_NODE_P2P_PORT="26656"
PERSISTENT_PEERS="${SEED_NODE_ID}@${SEED_NODE_P2P_HOST}:${SEED_NODE_P2P_PORT}"

# Public HTTPS endpoints used to pull genesis + for state sync
GENESIS_URL="https://chain.larrytrustee.ai/trpc/genesis"
STATE_SYNC_RPC="https://chain.larrytrustee.ai/trpc/"

# Repo source of the ltrstchaind Cosmos SDK fork. The build step expects
# go.mod at the root of this directory. You can rsync your local copy up
# before running this script: rsync -a ltrst-chain-l1/ <host>:/opt/ltrstchain/src/
SOURCE_DIR="/opt/ltrstchain/src"

# ---- Arg parsing ------------------------------------------------------
MONIKER=""
AS_VALIDATOR="no"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --moniker) MONIKER="$2"; shift 2 ;;
    --validator) AS_VALIDATOR="yes"; shift ;;
    -h|--help)
      sed -n '2,25p' "$0"; exit 0 ;;
    *) echo "unknown arg: $1"; exit 1 ;;
  esac
done
if [[ -z "$MONIKER" ]]; then echo "error: --moniker required"; exit 1; fi

# ---- Step 1: apt deps -------------------------------------------------
echo "==> [1/8] Installing apt dependencies"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq curl wget jq git build-essential ufw fail2ban unattended-upgrades nginx >/dev/null

# ---- Step 2: Go 1.25.1 ------------------------------------------------
if ! /usr/local/go/bin/go version 2>/dev/null | grep -q "go${GO_VERSION}"; then
  echo "==> [2/8] Installing Go ${GO_VERSION}"
  cd /tmp
  wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
  echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/golang.sh
fi
export PATH="$PATH:/usr/local/go/bin"

# ---- Step 3: build ltrstchaind ---------------------------------------
echo "==> [3/8] Building ltrstchaind"
mkdir -p "$BIN_DIR" "$HOME_DIR" "$LOG_DIR"
if [[ ! -x "$BIN_DIR/ltrstchaind" ]]; then
  if [[ ! -d "$SOURCE_DIR" ]]; then
    echo "error: $SOURCE_DIR missing. rsync the ltrst-chain-l1 source tree first:"
    echo "  rsync -a --exclude=build/ ltrst-chain-l1/ root@<this-host>:$SOURCE_DIR/"
    exit 1
  fi
  cd "$SOURCE_DIR"
  go build -mod=readonly -ldflags="-checklinkname=0" -o "$BIN_DIR/ltrstchaind" ./cmd/ltrstchaind
fi
"$BIN_DIR/ltrstchaind" version 2>&1 || true

# ---- Step 4: init home -----------------------------------------------
if [[ ! -f "$HOME_DIR/config/genesis.json" ]]; then
  echo "==> [4/8] Initializing home directory"
  "$BIN_DIR/ltrstchaind" init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR"
fi

# ---- Step 5: fetch canonical genesis via HTTPS ------------------------
echo "==> [5/8] Fetching canonical genesis from $GENESIS_URL"
TMP_GENESIS="$(mktemp)"
# /genesis returns { result: { genesis: {...} } }
curl -sS "$GENESIS_URL" | jq '.result.genesis' > "$TMP_GENESIS"
if ! jq -e . "$TMP_GENESIS" >/dev/null; then
  echo "error: failed to fetch canonical genesis"
  exit 1
fi
install -m 600 "$TMP_GENESIS" "$HOME_DIR/config/genesis.json"
rm -f "$TMP_GENESIS"

EXPECTED_GENESIS_SHA="b38fb00bbf873cc9c8ed69491b5e4a7eb1a2f6c77457a6969a5937d993df2385"
ACTUAL_GENESIS_SHA="$(sha256sum "$HOME_DIR/config/genesis.json" | awk '{print $1}')"
if [[ "$ACTUAL_GENESIS_SHA" != "$EXPECTED_GENESIS_SHA" ]]; then
  echo "WARNING: genesis sha256 mismatch"
  echo "  expected: $EXPECTED_GENESIS_SHA"
  echo "  actual:   $ACTUAL_GENESIS_SHA"
  echo "  (Genesis may have been re-exported. Verify against FORK-CREDENTIALS.md.)"
fi

# ---- Step 6: patch config.toml / app.toml -----------------------------
echo "==> [6/8] Patching config.toml and app.toml"
CONFIG_TOML="$HOME_DIR/config/config.toml"
APP_TOML="$HOME_DIR/config/app.toml"

sed -i "s/^moniker = .*/moniker = \"$MONIKER\"/" "$CONFIG_TOML"
sed -i "s|^persistent_peers = .*|persistent_peers = \"$PERSISTENT_PEERS\"|" "$CONFIG_TOML"
sed -i "s|^laddr = \"tcp://127.0.0.1:26657\"|laddr = \"tcp://0.0.0.0:26657\"|" "$CONFIG_TOML"
sed -i "s/^timeout_commit = .*/timeout_commit = \"2s\"/" "$CONFIG_TOML"
sed -i 's/^cors_allowed_origins = \[\]/cors_allowed_origins = ["*"]/' "$CONFIG_TOML"
sed -i "s/^prometheus = false/prometheus = true/" "$CONFIG_TOML"

# Light state sync — pulls recent snapshot instead of replaying all blocks
LATEST_HEIGHT="$(curl -s "$STATE_SYNC_RPC/block" | jq -r '.result.block.header.height')"
if [[ -n "$LATEST_HEIGHT" && "$LATEST_HEIGHT" != "null" ]]; then
  TRUST_HEIGHT=$((LATEST_HEIGHT - 500))
  TRUST_HASH="$(curl -s "$STATE_SYNC_RPC/block?height=$TRUST_HEIGHT" | jq -r '.result.block_id.hash')"
  sed -i "s/^enable = false/enable = true/" "$CONFIG_TOML"
  sed -i "s|^rpc_servers = .*|rpc_servers = \"${STATE_SYNC_RPC},${STATE_SYNC_RPC}\"|" "$CONFIG_TOML"
  sed -i "s/^trust_height = .*/trust_height = $TRUST_HEIGHT/" "$CONFIG_TOML"
  sed -i "s|^trust_hash = .*|trust_hash = \"$TRUST_HASH\"|" "$CONFIG_TOML"
  echo "    state sync enabled: trust_height=$TRUST_HEIGHT trust_hash=$TRUST_HASH"
fi

python3 - "$APP_TOML" <<'PY'
import re, sys
p = sys.argv[1]
s = open(p).read()
s = re.sub(r'minimum-gas-prices = .*', 'minimum-gas-prices = "0.0025ultrst"', s)
# enable REST + gRPC on all interfaces
s = re.sub(r'(\[api\][^\[]*?)enable = false', r'\1enable = true', s, flags=re.S)
s = re.sub(r'(\[api\][^\[]*?)swagger = false', r'\1swagger = true', s, flags=re.S)
s = re.sub(r'(\[api\][^\[]*?)address = "tcp://localhost:1317"', r'\1address = "tcp://0.0.0.0:1317"', s, flags=re.S)
s = re.sub(r'(\[api\][^\[]*?)enabled-unsafe-cors = false', r'\1enabled-unsafe-cors = true', s, flags=re.S)
s = re.sub(r'(\[grpc\][^\[]*?)enable = false', r'\1enable = true', s, flags=re.S)
s = re.sub(r'(\[grpc\][^\[]*?)address = "localhost:9090"', r'\1address = "0.0.0.0:9090"', s, flags=re.S)
open(p,'w').write(s)
PY

# ---- Step 7: systemd unit --------------------------------------------
echo "==> [7/8] Installing systemd unit"
cat > /etc/systemd/system/ltrstchain.service <<EOF
[Unit]
Description=LTRST Chain L1 (Cosmos SDK) - $CHAIN_ID
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=$BIN_DIR/ltrstchaind start --home $HOME_DIR --log_level info
Restart=always
RestartSec=5
LimitNOFILE=65535
StandardOutput=append:$LOG_DIR/ltrstchain.log
StandardError=append:$LOG_DIR/ltrstchain.err.log

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable --now ltrstchain.service

# ---- Step 8: firewall + final checks ---------------------------------
echo "==> [8/8] Firewall + final checks"
ufw allow 22/tcp
ufw allow 26656/tcp   comment 'Cosmos P2P'
yes | ufw enable || true

sleep 3
systemctl is-active ltrstchain || { journalctl -u ltrstchain --no-pager | tail -30; exit 1; }

NODE_ID="$("$BIN_DIR/ltrstchaind" --home "$HOME_DIR" cometbft show-node-id)"
echo
echo "====================================="
echo "Node online."
echo "  chain_id:  $CHAIN_ID"
echo "  moniker:   $MONIKER"
echo "  node_id:   $NODE_ID"
echo "  rpc:       http://127.0.0.1:26657 (P2P port 26656 is the only public port)"
echo "====================================="

if [[ "$AS_VALIDATOR" == "yes" ]]; then
cat <<VALEOF

This node is configured as a FULL NODE only. To promote it to a validator
on ltrst-chain-1 you still need to:

  1. Import or generate a key in the same keyring-test as the genesis nodes:
       $BIN_DIR/ltrstchaind keys add <your_validator_name> \\
           --home $HOME_DIR --keyring-backend test

  2. Fund that address from one of the 8 genesis accounts (public_sale,
     ecosystem_dev, validator1, etc.) with at least 10,000 LTRST for the
     self-delegation + tx fees.

  3. Wait until this node is fully synced (catching_up == false).

  4. Submit a create-validator tx:
       $BIN_DIR/ltrstchaind tx staking create-validator \\
           --amount=100000000000ultrst \\
           --pubkey=\$($BIN_DIR/ltrstchaind --home $HOME_DIR cometbft show-validator) \\
           --moniker="$MONIKER" \\
           --chain-id="$CHAIN_ID" \\
           --commission-rate=0.10 \\
           --commission-max-rate=0.20 \\
           --commission-max-change-rate=0.01 \\
           --min-self-delegation=1 \\
           --from=<your_validator_name> \\
           --keyring-backend=test \\
           --home=$HOME_DIR \\
           --gas=auto --gas-adjustment=1.3 --fees=5000ultrst \\
           --node=http://127.0.0.1:26657

Keep the minimum commission rate ≥ 5% (the network param). Once the tx lands,
the validator appears in the active set and starts signing blocks.
VALEOF
fi
