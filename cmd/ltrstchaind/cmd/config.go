package cmd

import (
	cmtcfg "github.com/cometbft/cometbft/config"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
)

// initCometBFTConfig helps to override default CometBFT Config values.
// return cmtcfg.DefaultConfig if no custom configuration is required for the application.
func initCometBFTConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()

	// ── RPC: bind all interfaces so workers / wallets can reach the node ──
	cfg.RPC.ListenAddress = "tcp://0.0.0.0:26657"

	// ── P2P: private single-validator chain — relax strict routability ──
	cfg.P2P.ListenAddress = "tcp://0.0.0.0:26656"
	cfg.P2P.AddrBookStrict = false   // required for private/local networks
	cfg.P2P.AllowDuplicateIP = true  // single-node may reconnect from same IP

	// ── Consensus: 2-second block time per whitepaper ──
	cfg.Consensus.TimeoutPropose = 1_000_000_000       // 1s
	cfg.Consensus.TimeoutProposeDelta = 500_000_000     // 500ms
	cfg.Consensus.TimeoutPrevote = 500_000_000          // 500ms
	cfg.Consensus.TimeoutPrevoteDelta = 500_000_000     // 500ms
	cfg.Consensus.TimeoutPrecommit = 500_000_000        // 500ms
	cfg.Consensus.TimeoutPrecommitDelta = 500_000_000   // 500ms
	cfg.Consensus.TimeoutCommit = 2_000_000_000         // 2s

	return cfg
}

// initAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	// The following code snippet is just for reference.
	type CustomAppConfig struct {
		serverconfig.Config `mapstructure:",squash"`
	}

	// Optionally allow the chain developer to overwrite the SDK's default
	// server config.
	srvCfg := serverconfig.DefaultConfig()
	// The SDK's default minimum gas price is set to "" (empty value) inside
	// app.toml. If left empty by validators, the node will halt on startup.
	// However, the chain developer can set a default app.toml value for their
	// validators here.
	//
	// In summary:
	// - if you leave srvCfg.MinGasPrices = "", all validators MUST tweak their
	//   own app.toml config,
	// - if you set srvCfg.MinGasPrices non-empty, validators CAN tweak their
	//   own app.toml to override, or use this default value.
	//
	// In tests, we set the min gas prices to 0.
	// srvCfg.MinGasPrices = "0stake"
	// srvCfg.BaseConfig.IAVLDisableFastNode = true // disable fastnode by default

	customAppConfig := CustomAppConfig{
		Config: *srvCfg,
	}

	customAppTemplate := serverconfig.DefaultConfigTemplate
	// Edit the default template file
	//
	// customAppTemplate := serverconfig.DefaultConfigTemplate + `
	// [wasm]
	// # This is the maximum sdk gas (wasm and storage) that we allow for any x/wasm "smart" queries
	// query_gas_limit = 300000
	// # This is the number of wasm vm instances we keep cached in memory for speed-up
	// # Warning: this is currently unstable and may lead to crashes, best to keep for 0 unless testing locally
	// lru_size = 0`

	return customAppTemplate, customAppConfig
}
