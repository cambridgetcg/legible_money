package cmd

import (
	"time"

	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	cmtcfg "github.com/cometbft/cometbft/config"
)

// initAppConfig returns a custom app server configuration template and default values.
func initAppConfig() (string, interface{}) {
	srvCfg := serverconfig.DefaultConfig()

	// LGM uses ulgm as the minimum gas price denomination.
	srvCfg.MinGasPrices = "0.025ulgm"

	// Enable API by default for development.
	srvCfg.API.Enable = true
	srvCfg.API.Swagger = true

	return serverconfig.DefaultConfigTemplate, srvCfg
}

// initCometBFTConfig returns custom CometBFT configuration.
func initCometBFTConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()

	// Legible Money target block time: 2521ms.
	cfg.Consensus.TimeoutCommit = 2521 * time.Millisecond
	cfg.Consensus.TimeoutPropose = 2000 * time.Millisecond

	// ABCI++ vote extensions for PoT consensus are enabled via genesis
	// consensus params (consensus_params.abci.vote_extensions_enable_height = "1").
	// See genesis.go for default genesis configuration.

	return cfg
}
