#!/usr/bin/env bash
#
# chain-health.sh — Zerone chain health monitor
#
# Checks devnet health and reports metrics.
# Called by Asha's heartbeat or run standalone.
#
# Usage:
#   ./scripts/chain-health.sh          Quick status
#   ./scripts/chain-health.sh full     Full health check
#   ./scripts/chain-health.sh inject   Submit a random knowledge claim (keep chain alive)

set -euo pipefail

BINARY="$(dirname "$0")/../build/legbled"
NODE_HOME="$HOME/.legbled-devnet"
CHAIN_ID="lgm-devnet-1"
KEYRING="test"
DENOM="ulgm"
LGM=1000000

legbled() { "$BINARY" "$@" --home "$NODE_HOME" 2>/dev/null; }

# ---------------------------------------------------------------------------
# Quick status
# ---------------------------------------------------------------------------
quick_status() {
    if ! curl -s --connect-timeout 2 http://localhost:26657/status >/dev/null 2>&1; then
        echo "CHAIN: OFFLINE"
        return 1
    fi

    local status
    status=$(curl -s http://localhost:26657/status)
    local height=$(echo "$status" | jq -r '.result.sync_info.latest_block_height')
    local time=$(echo "$status" | jq -r '.result.sync_info.latest_block_time')
    local catching=$(echo "$status" | jq -r '.result.sync_info.catching_up')

    echo "CHAIN: ONLINE"
    echo "  Height: $height"
    echo "  Time: $time"
    echo "  Catching up: $catching"

    # Check validator
    local val_addr
    val_addr=$(legbled keys show validator -a --keyring-backend "$KEYRING" 2>/dev/null || echo "unknown")
    local dashboard
    dashboard=$(legbled query dashboard "$val_addr" 2>/dev/null || echo "")
    local balance
    balance=$(echo "$dashboard" | grep -o 'Balance: [0-9]*' | grep -o '[0-9]*' || echo "?")
    echo "  Validator balance: $balance LGM"
}

# ---------------------------------------------------------------------------
# Full health check
# ---------------------------------------------------------------------------
full_check() {
    quick_status
    echo ""

    # Check all accounts
    echo "ACCOUNTS:"
    for name in validator asha nuance forge lark sentry patch thresh test1 test2 test3 alpha-agent beta-agent gamma-agent; do
        local addr
        addr=$(legbled keys show "$name" -a --keyring-backend "$KEYRING" 2>/dev/null || echo "")
        if [ -n "$addr" ]; then
            local dashboard
            dashboard=$(legbled query dashboard "$addr" 2>/dev/null || echo "")
            local bal
            bal=$(echo "$dashboard" | grep -o 'Balance: [0-9.]*' | grep -o '[0-9.]*' || echo "0")
            local facts
            facts=$(echo "$dashboard" | grep -o 'Facts Submitted: [0-9]*' | grep -o '[0-9]*' || echo "0")
            local partnerships
            partnerships=$(echo "$dashboard" | grep -c 'Partnership:' || echo "0")
            printf "  %-14s %s  bal=%s LGM  facts=%s  partnerships=%s\n" "$name" "${addr:0:15}..." "$bal" "$facts" "$partnerships"
        fi
    done
}

# ---------------------------------------------------------------------------
# Inject a claim (keep chain alive with activity)
# ---------------------------------------------------------------------------
inject_claim() {
    if ! curl -s --connect-timeout 2 http://localhost:26657/status >/dev/null 2>&1; then
        echo "Chain offline, cannot inject"
        return 1
    fi

    # Rotate through domains and submitters
    local domains=("logic" "economics" "computer_science" "cryptography" "epistemology" "mathematics" "physics")
    local submitters=("asha" "nuance" "forge" "test1" "test2")
    local height
    height=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')

    # Use height as seed for deterministic rotation
    local domain="${domains[$((height % ${#domains[@]}))]}"
    local submitter="${submitters[$((height % ${#submitters[@]}))]}"

    # Generate claim based on domain
    local claims_logic=(
        "De Morgan's laws: negation distributes over conjunction and disjunction"
        "Gödel's first incompleteness theorem: any consistent formal system is incomplete"
        "The halting problem is undecidable in general"
        "Tarski's undefinability theorem: truth cannot be defined within its own language"
    )
    local claims_economics=(
        "Marginal utility diminishes with increasing consumption"
        "Network effects create winner-take-most dynamics in platform economies"
        "Token velocity inversely correlates with token price when demand is constant"
        "Mechanism design can align individual incentives with collective outcomes"
    )
    local claims_cs=(
        "CAP theorem: a distributed system cannot simultaneously guarantee consistency availability and partition tolerance"
        "Amdahl's law limits parallel speedup based on sequential fraction"
        "Lamport timestamps establish happened-before ordering in distributed systems"
        "Merkle trees enable efficient verification of data integrity in untrusted environments"
    )

    local claim_idx=$((height / ${#domains[@]} % 4))
    local claim=""
    local category="analytic"
    local zone="propositional"

    local claims_cryptography=(
        "Public-key cryptography relies on the asymmetry between easy computation and hard inversion"
        "A cryptographic hash function must be collision-resistant, preimage-resistant, and second-preimage-resistant"
        "Zero-knowledge proofs allow a prover to convince a verifier of a statement without revealing the witness"
        "The Diffie-Hellman key exchange establishes a shared secret over an insecure channel without transmitting the secret"
    )
    local claims_epistemology=(
        "Justified true belief is necessary but not sufficient for knowledge, as Gettier cases demonstrate"
        "The problem of induction: past regularities provide no deductive guarantee of future regularities"
        "Bayesian epistemology treats credence as probability and updates beliefs via Bayes' theorem"
        "Foundationalism holds that some beliefs are basic and non-inferential, grounding all other knowledge"
    )
    local claims_mathematics=(
        "Every natural number greater than 1 is either prime or a product of primes, uniquely up to order"
        "Cantor's theorem: the power set of any set has strictly greater cardinality than the set itself"
        "The Pythagorean theorem holds for any right triangle: the square of the hypotenuse equals the sum of squares of the legs"
        "Euler's identity: e raised to the power of i times pi, plus one, equals zero"
    )
    local claims_physics=(
        "The speed of light in a vacuum is constant for all inertial observers, regardless of source motion"
        "Energy and mass are equivalent: E equals m times c squared"
        "Entropy of a closed system never decreases over time, as stated by the second law of thermodynamics"
        "The uncertainty principle: position and momentum of a particle cannot both be known to arbitrary precision"
    )

    case "$domain" in
        logic)          claim="${claims_logic[$claim_idx]}" ;;
        economics)      claim="${claims_economics[$claim_idx]}"; category="empirical"; zone="empirical" ;;
        computer_science) claim="${claims_cs[$claim_idx]}" ;;
        cryptography)   claim="${claims_cryptography[$claim_idx]}" ;;
        epistemology)   claim="${claims_epistemology[$claim_idx]}" ;;
        mathematics)    claim="${claims_mathematics[$claim_idx]}"; category="analytic"; zone="propositional" ;;
        physics)        claim="${claims_physics[$claim_idx]}"; category="empirical"; zone="empirical" ;;
        *)              claim="A consistent formal system contains truths unprovable within that system"; category="analytic"; zone="propositional" ;;
    esac

    echo "Injecting: [$domain] $claim (from $submitter)"

    "$BINARY" tx claim "$claim" \
        --domain "$domain" \
        --category "$category" \
        --logic-zone "$zone" \
        --stake "1LGM" \
        --from "$submitter" \
        --chain-id "$CHAIN_ID" \
        --keyring-backend "$KEYRING" \
        --home "$NODE_HOME" \
        --gas 500000 --fees 500000"$DENOM" -y 2>&1 | tail -3
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
case "${1:-}" in
    full)   full_check ;;
    inject) inject_claim ;;
    *)      quick_status ;;
esac
