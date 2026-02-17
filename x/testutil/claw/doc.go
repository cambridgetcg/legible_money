// Package claw provides the Claw adversarial testing framework for Legible Money.
//
// Claw probes systematically test the protocol's defenses against game-theoretic
// attacks, precision exploits, zero-cost vectors, and cross-module vulnerabilities.
//
// # Taxonomy
//
// OpenClaw (OC): Macro game theory probes — multi-actor economic scenarios,
// collusion, information asymmetry, strategic behavior. The original and most
// common variant.
//
// NanoClaw (NC): Precision attack probes — integer truncation, BPS arithmetic
// boundaries, off-by-one errors, type conversion edge cases.
//
// ZeroClaw (ZC): Zero-cost attack probes — empty work exploits, free information
// extraction, gas-less exploitation, zero-stake operations.
//
// CrossModule (XM): Cross-module attack probes — attacks spanning multiple
// Cosmos SDK modules (e.g., staking + knowledge race conditions).
//
// # Naming Convention
//
// New probes MUST follow the pattern: Test{Variant}_{Number}_{ShortName}
//
//	TestOC_16_ValidatorCycling
//	TestNC_1_SlashRateBoundary
//	TestZC_1_FreeStateQuery
//	TestXM_1_StakeClaimUnstakeRace
//
// Legacy probes use module-specific prefixes that predate this convention:
//
//	OC-1..15     — knowledge module (OpenClaw)
//	OCP-1..12    — auth module (OpenClaw Privacy)
//	OCA-T/U/B/J/C — alignment module (OpenClaw Alignment)
//	OC-AP-1..8   — autopoiesis module
//	OC-BVM-1..18 — bvm module
//
// # File Convention
//
//	openclaw_*.go    — OpenClaw probes
//	nanoclaw_*.go    — NanoClaw probes
//	zeroclaw_*.go    — ZeroClaw probes
//	crossmodule_*.go — CrossModule probes (in tests/cross_module/)
//
// # Catalog
//
// All probes are registered in [Catalog]. Use [ByModule], [ByVariant],
// [ByStatus], [CoverageByModule], and [UncoveredModules] to query the registry.
//
// # Reporting
//
// Use [Insight], [Economics], [Defense], and [Result] for structured test logging
// that follows a consistent format across all probe files.
package claw
