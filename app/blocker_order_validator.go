// Package app provides startup validation for BeginBlocker/EndBlocker ordering.
//
// The Cosmos SDK ModuleManager accepts plain string slices for blocker ordering
// and performs NO dependency validation. A misplaced module compiles and starts
// without error but silently corrupts chain state. This file encodes the
// dependency graph as machine-checkable assertions that panic at app init.
package app

import (
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	legibleauthtypes "github.com/legible-money/legible/x/auth/types"
	capturedefensetypes "github.com/legible-money/legible/x/capture_defense/types"
	emergencytypes "github.com/legible-money/legible/x/emergency/types"
	knowledgetypes "github.com/legible-money/legible/x/knowledge/types"
	ontologytypes "github.com/legible-money/legible/x/ontology/types"
	qualificationtypes "github.com/legible-money/legible/x/qualification/types"
	legiblestakingtypes "github.com/legible-money/legible/x/staking/types"
	tokenstypes "github.com/legible-money/legible/x/tokens/types"
	vestingrewardstypes "github.com/legible-money/legible/x/vesting_rewards/types"
)

// blockerDep declares that Before must appear at a strictly lower index
// than After in the blocker order slice.
type blockerDep struct {
	Before string
	After  string
	Reason string
}

// beginBlockerDeps encodes the dependency graph documented in app.go:1050-1060.
// Each edge means: Before's BeginBlocker MUST execute before After's.
var beginBlockerDeps = []blockerDep{
	// upgrade must be first — halts chain before any module touches state
	{upgradetypes.ModuleName, legibleauthtypes.ModuleName, "upgrade halts chain before custom modules"},
	{upgradetypes.ModuleName, emergencytypes.ModuleName, "upgrade halts chain before custom modules"},
	{upgradetypes.ModuleName, vestingrewardstypes.ModuleName, "upgrade halts chain before custom modules"},
	{upgradetypes.ModuleName, tokenstypes.ModuleName, "upgrade halts chain before custom modules"},
	{upgradetypes.ModuleName, legiblestakingtypes.ModuleName, "upgrade halts chain before custom modules"},
	{upgradetypes.ModuleName, knowledgetypes.ModuleName, "upgrade halts chain before custom modules"},

	// Identity before staking
	{legibleauthtypes.ModuleName, legiblestakingtypes.ModuleName, "session cleanup before stake operations"},

	// Economic foundation
	{vestingrewardstypes.ModuleName, legiblestakingtypes.ModuleName, "minted rewards available for unbonding OriginPeriod credits"},
	{tokenstypes.ModuleName, legiblestakingtypes.ModuleName, "period sweep before OriginPeriod credit in unbonding"},

	// Emergency before safety modules
	{emergencytypes.ModuleName, capturedefensetypes.ModuleName, "halt status must be known for reputation decay"},
	{emergencytypes.ModuleName, knowledgetypes.ModuleName, "halt status checked before round selection"},

	// Domain knowledge pipeline
	{ontologytypes.ModuleName, qualificationtypes.ModuleName, "domain structure current before qualification checks"},
	{qualificationtypes.ModuleName, knowledgetypes.ModuleName, "validator eligibility set before round selection"},

	// Staking feeds knowledge
	{legiblestakingtypes.ModuleName, knowledgetypes.ModuleName, "validator set current for round phase transitions"},

	// Capture defense feeds knowledge
	{capturedefensetypes.ModuleName, knowledgetypes.ModuleName, "reputation state current before round selection"},
}

// ValidateBlockerOrder panics at app init if the declared dependency graph is
// violated by the actual ordering. Call immediately after SetOrderBeginBlockers.
//
// Modules absent from the order slice are silently skipped (stub modules with
// no-op blockers may be omitted by the module manager).
func ValidateBlockerOrder(order []string, deps []blockerDep) {
	index := make(map[string]int, len(order))
	for i, name := range order {
		index[name] = i
	}

	for _, dep := range deps {
		beforeIdx, beforeFound := index[dep.Before]
		afterIdx, afterFound := index[dep.After]

		if !beforeFound || !afterFound {
			continue // module not registered — skip
		}

		if beforeIdx >= afterIdx {
			panic(fmt.Sprintf(
				"BeginBlocker ordering violation: %q (position %d) must run BEFORE %q (position %d) — %s",
				dep.Before, beforeIdx, dep.After, afterIdx, dep.Reason,
			))
		}
	}
}
