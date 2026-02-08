// Package main is the entry point for the Legible Money blockchain daemon.
//
// Binary name: legbled (Legible + "d" for daemon)
//
// Usage:
//
//	legbled init [moniker] --chain-id lgm-1
//	legbled start
//	legbled keys add [name]
//	legbled tx bank send [from] [to] [amount]
//	legbled query bank balances [address]
//	legbled tx ibc-transfer transfer [src-port] [src-channel] [receiver] [amount]
package main

import (
	"os"

	"cosmossdk.io/log"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/legible-money/legible/app"
	"github.com/legible-money/legible/cmd/legbled/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		log.NewNopLogger().Error("failure when running app", "err", err)
		os.Exit(1)
	}
}
