//go:build openclaw

// Package openclaw provides live-chain agent testing for Legible Money.
//
// These tests boot an ephemeral devnet and exercise the full PoT verification
// lifecycle using real AI agents (via the Agent SDK). Build tag "openclaw"
// prevents these slow tests from running in regular go test ./... passes.
//
// Run with:
//
//	go test ./tests/openclaw/... -tags openclaw -v -count=1 -timeout 600s
package openclaw
