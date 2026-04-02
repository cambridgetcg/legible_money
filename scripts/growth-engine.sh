#!/usr/bin/env bash
#
# growth-engine.sh — Zerone Token Consumption Growth Engine
#
# Activates all token consumption mechanisms on devnet:
#   Phase 1: Start devnet + register validators (2 → 8)
#   Phase 2: Register agents (auth + discovery + homes)
#   Phase 3: Flood knowledge claims (citation graph)
#   Phase 4: Submit oracle predictions
#   Phase 5: Form partnerships (tier staking)
#   Phase 6: Activate compute pool
#   Phase 7: Challenge & dispute cycle
#   Phase 8: Vesting lifecycle (truth-linked vesting schedules)
#
# Usage:
#   ./scripts/growth-engine.sh              Full run (start devnet + all phases)
#   ./scripts/growth-engine.sh phase N      Run specific phase (devnet must be running)
#   ./scripts/growth-engine.sh status       Show chain metrics
#
# Requirements: legbled binary built at build/legbled

set -euo pipefail

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY="$PROJECT_ROOT/build/legbled"
NODE_HOME="$HOME/.legbled-devnet"
CHAIN_ID="lgm-devnet-1"
KEYRING="test"
DENOM="ulgm"
LGM=1000000

# Common flags for all tx commands
TX_FLAGS="--chain-id $CHAIN_ID --keyring-backend $KEYRING --home $NODE_HOME --gas 500000 --fees 500000$DENOM -y -o json"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
info()  { echo -e "\033[1;34m[GROWTH]\033[0m $*"; }
ok()    { echo -e "\033[1;32m  [OK]\033[0m $*"; }
warn()  { echo -e "\033[1;33m  [!!]\033[0m $*"; }
err()   { echo -e "\033[1;31m  [ERR]\033[0m $*" >&2; }
phase() { echo -e "\n\033[1;35m═══════════════════════════════════════════════════\033[0m"; echo -e "\033[1;35m  PHASE $1: $2\033[0m"; echo -e "\033[1;35m═══════════════════════════════════════════════════\033[0m\n"; }

legbled() { "$BINARY" "$@" --home "$NODE_HOME" 2>/dev/null; }
legbled_tx() { "$BINARY" "$@" --home "$NODE_HOME" $TX_FLAGS 2>&1 | tail -1; }

wait_blocks() {
    local count="${1:-2}"
    local block_time_ms=2521
    local wait_s=$(( (count * block_time_ms + 999) / 1000 ))
    info "Waiting $count blocks (~${wait_s}s)..."
    sleep "$wait_s"
}

addr_of() { legbled keys show "$1" -a --keyring-backend "$KEYRING" 2>/dev/null; }

ensure_key() {
    local name="$1"
    if ! legbled keys show "$name" --keyring-backend "$KEYRING" >/dev/null 2>&1; then
        legbled keys add "$name" --keyring-backend "$KEYRING" >/dev/null 2>&1
        ok "Key created: $name"
    fi
}

fund_account() {
    local name="$1"
    local amount="$2"
    local addr
    addr=$(addr_of "$name")
    "$BINARY" tx bank send validator "$addr" "${amount}${DENOM}" \
        --chain-id "$CHAIN_ID" --keyring-backend "$KEYRING" --home "$NODE_HOME" \
        --gas 200000 --fees 250000"$DENOM" -y -o json >/dev/null 2>&1
    ok "Funded $name ($addr) with $amount $DENOM"
    sleep 4  # wait for tx to land
}

get_block_height() {
    curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height // "0"'
}

check_node() {
    if ! curl -s --connect-timeout 2 http://localhost:26657/status >/dev/null 2>&1; then
        err "Node not running. Start devnet first: make devnet"
        exit 1
    fi
}

# ---------------------------------------------------------------------------
# Phase 0: Start Devnet (if not running)
# ---------------------------------------------------------------------------
phase_0() {
    phase "0" "DEVNET STARTUP"

    if curl -s --connect-timeout 2 http://localhost:26657/status >/dev/null 2>&1; then
        local height
        height=$(get_block_height)
        ok "Devnet already running at block $height"
        return 0
    fi

    info "Starting fresh devnet..."
    cd "$PROJECT_ROOT"

    # Build if needed
    if [ ! -f "$BINARY" ]; then
        info "Building legbled..."
        go build -o "$BINARY" ./cmd/legbled
        ok "Binary built"
    fi

    # Start devnet in background
    bash devnet/scripts/cosmos-devnet.sh start &
    DEVNET_PID=$!

    # Wait for node to be ready
    info "Waiting for node to start..."
    for i in $(seq 1 30); do
        if curl -s --connect-timeout 1 http://localhost:26657/status >/dev/null 2>&1; then
            ok "Devnet started (PID: $DEVNET_PID)"
            wait_blocks 3
            return 0
        fi
        sleep 2
    done

    err "Devnet failed to start within 60s"
    exit 1
}

# ---------------------------------------------------------------------------
# Phase 1: Validator Scaling (2 → 8+)
# ---------------------------------------------------------------------------
phase_1() {
    phase "1" "VALIDATOR SCALING (target: 8 validators)"

    # Kingdom agent names for validators
    local agents=("asha" "nuance" "forge" "lark" "sentry" "patch" "thresh")
    local min_stake=$(( 111000 ))  # 0.111 LGM minimum
    local self_stake=$(( 500 * LGM ))  # 500 LGM self-delegation
    local fund_amount=$(( 1000 * LGM ))  # 1000 LGM to cover stake + fees

    for agent in "${agents[@]}"; do
        info "Registering validator: $agent"

        # Create key if needed
        ensure_key "$agent"

        # Fund from validator account
        fund_account "$agent" "$fund_amount"
        wait_blocks 1

        # Get the consensus pubkey (hex-encoded ed25519)
        local pubkey
        pubkey=$(legbled keys show "$agent" --keyring-backend "$KEYRING" -p 2>/dev/null | jq -r '.key // empty' 2>/dev/null || echo "")

        if [ -z "$pubkey" ]; then
            # Generate a random ed25519 key for consensus (32 bytes hex)
            pubkey=$(openssl rand -hex 32)
        fi

        # Register as validator
        legbled_tx tx legible_staking register-validator \
            "$pubkey" \
            "${self_stake}${DENOM}" \
            --moniker "$agent" \
            --details "Kingdom agent - $agent" \
            --identity "did:lgm:$agent" \
            --from "$agent" >/dev/null 2>&1 && \
            ok "Validator registered: $agent (stake: $self_stake $DENOM)" || \
            warn "Validator registration may have failed for $agent"

        wait_blocks 1
    done

    # Query validator count
    info "Querying validator list..."
    legbled query legible_staking validators --limit 20 2>/dev/null | jq '.validators | length' 2>/dev/null && \
        ok "Validator scaling complete" || warn "Could not query validators"
}

# ---------------------------------------------------------------------------
# Phase 2: Agent Registration (auth + discovery + homes)
# ---------------------------------------------------------------------------
phase_2() {
    phase "2" "AGENT REGISTRATION (DID + Discovery + Homes)"

    local agents=("asha" "nuance" "forge" "lark" "sentry" "patch" "alpha-agent" "beta-agent" "gamma-agent")
    local discovery_stake=$(( 100 * LGM ))  # 100 LGM for discovery registration

    for agent in "${agents[@]}"; do
        info "Registering agent: $agent"

        ensure_key "$agent"

        # Fund if not already done in phase 1
        local dashboard
        dashboard=$(legbled query dashboard "$(addr_of "$agent")" 2>/dev/null || echo "Balance: 0")
        local bal_lgm
        bal_lgm=$(echo "$dashboard" | grep -o 'Balance: [0-9]*' | grep -o '[0-9]*' || echo "0")
        if [ "$bal_lgm" -lt 200 ] 2>/dev/null; then
            fund_account "$agent" "$((500 * LGM))"
        fi

        # Quick-register (auto-derives DID from key)
        legbled_tx tx quick-register \
            --from "$agent" >/dev/null 2>&1 && \
            ok "Auth registered: $agent" || warn "Auth registration skipped for $agent (may already exist)"

        wait_blocks 1

        # Discovery registration
        local domains="knowledge,verification,truth"
        case "$agent" in
            asha)        domains="blockchain,consensus,verification,truth" ;;
            nuance)      domains="language,semantics,naming" ;;
            forge)       domains="infrastructure,deployment" ;;
            alpha-agent) domains="philosophy,theology,consciousness" ;;
            beta-agent)  domains="economics,operations,commerce" ;;
            gamma-agent) domains="engineering,blockchain,computation" ;;
        esac

        legbled_tx tx discovery register-agent \
            --from "$agent" \
            --name "$agent" \
            --domains "$domains" >/dev/null 2>&1 && \
            ok "Discovery registered: $agent (domains: $domains)" || \
            warn "Discovery registration skipped for $agent"

        wait_blocks 1

        # Create agent home
        legbled_tx tx home create-home \
            "$agent-home" \
            --from "$agent" >/dev/null 2>&1 && \
            ok "Home created: $agent-home" || \
            warn "Home creation skipped for $agent"
    done
}

# ---------------------------------------------------------------------------
# Phase 3: Knowledge Flood (citation graph)
# ---------------------------------------------------------------------------
phase_3() {
    phase "3" "KNOWLEDGE FLOOD (claims + citations)"

    # Foundation layer: axioms and fundamentals (no citations)
    info "Submitting foundation claims..."

    local foundation_claims=(
        "The principle of non-contradiction: a statement cannot be both true and false|logic|analytic|propositional"
        "Law of excluded middle: every proposition is either true or its negation is true|logic|analytic|propositional"
        "Modus ponens: if P implies Q and P is true then Q is true|logic|formal|propositional"
        "Transitivity of implication: if P implies Q and Q implies R then P implies R|logic|formal|propositional"
        "Conservation of energy: energy cannot be created or destroyed only transformed|physics|empirical|empirical"
        "Supply and demand: price is determined by the intersection of supply and demand curves|economics|empirical|empirical"
        "Shannon entropy: information content is inversely proportional to probability|computer_science|formal|propositional"
        "Turing completeness: a system can simulate any Turing machine given sufficient resources|computer_science|formal|propositional"
        "Byzantine fault tolerance requires at least 3f+1 nodes to tolerate f faulty nodes|computer_science|formal|propositional"
        "Consensus in distributed systems is impossible with one faulty process (FLP impossibility)|computer_science|formal|propositional"
        "Truth verification requires independent attestation from multiple sources|epistemology|analytic|propositional"
        "Trust is built through repeated verified interactions over time|sociology|empirical|empirical"
        "Cryptographic hash functions provide collision resistance for data integrity|cryptography|formal|propositional"
        "Public key cryptography enables secure communication without shared secrets|cryptography|formal|propositional"
        "Proof of work wastes energy on computation unrelated to the verified content|economics|empirical|empirical"
        "Proof of Truth aligns consensus incentives with knowledge verification|economics|analytic|propositional"
        "Token velocity measures the rate at which tokens change hands in an economy|economics|empirical|empirical"
        "Deflationary pressure from token burns increases scarcity over time|economics|empirical|empirical"
        "Agent identity requires both authentication and capability verification|computer_science|analytic|propositional"
        "Decentralized identity (DID) enables self-sovereign identity without central authority|computer_science|analytic|propositional"
    )

    local fact_ids=()
    local submitters=("test1" "test2" "test3" "asha" "nuance" "forge")
    local sub_idx=0

    for claim_data in "${foundation_claims[@]}"; do
        IFS='|' read -r claim domain category logic_zone <<< "$claim_data"
        local submitter="${submitters[$((sub_idx % ${#submitters[@]}))]}"
        sub_idx=$((sub_idx + 1))

        legbled_tx tx claim "$claim" \
            --domain "$domain" \
            --category "$category" \
            --logic-zone "$logic_zone" \
            --stake "1LGM" \
            --from "$submitter" >/dev/null 2>&1 && \
            ok "Claim submitted [$domain/$category]: ${claim:0:60}..." || \
            warn "Claim failed: ${claim:0:40}..."

        # Don't wait every single block, batch them
        if [ $((sub_idx % 3)) -eq 0 ]; then
            wait_blocks 1
        fi
    done

    wait_blocks 3
    info "Foundation layer complete. Querying facts..."

    # Query fact count
    local fact_count
    fact_count=$(legbled query knowledge list-facts --limit 100 2>/dev/null | jq '.facts | length' 2>/dev/null || echo "?")
    ok "Facts on chain: $fact_count"

    # Second layer: claims that cite foundation facts
    info "Submitting citation layer..."

    local citation_claims=(
        "Proof of Truth consensus combines Byzantine fault tolerance with knowledge verification incentives|computer_science|analytic|propositional"
        "Token burns from verification fees create sustainable deflationary pressure in knowledge economies|economics|analytic|propositional"
        "Agent reputation systems require cryptographic identity and verified interaction history|computer_science|analytic|propositional"
        "Decentralized knowledge markets align individual verification incentives with collective truth|economics|analytic|propositional"
        "The FLP impossibility result motivates probabilistic consensus mechanisms like PoT|computer_science|formal|propositional"
        "Shannon entropy applies to knowledge verification: rare verified truths carry more information value|computer_science|formal|propositional"
        "Trust in AI agents requires verifiable computation chains anchored in cryptographic proofs|computer_science|analytic|propositional"
        "Knowledge graphs on blockchain enable citation-based reward distribution for truth discovery|computer_science|analytic|propositional"
        "Self-sovereign identity enables agents to accumulate reputation across multiple platforms|computer_science|analytic|propositional"
        "Proof of Truth is more energy efficient than Proof of Work because verification IS the useful work|economics|empirical|empirical"
    )

    sub_idx=0
    for claim_data in "${citation_claims[@]}"; do
        IFS='|' read -r claim domain category logic_zone <<< "$claim_data"
        local submitter="${submitters[$((sub_idx % ${#submitters[@]}))]}"
        sub_idx=$((sub_idx + 1))

        # Try to cite some foundation facts (best effort)
        legbled_tx tx claim "$claim" \
            --domain "$domain" \
            --category "$category" \
            --logic-zone "$logic_zone" \
            --stake "2LGM" \
            --from "$submitter" >/dev/null 2>&1 && \
            ok "Citation claim [$domain]: ${claim:0:60}..." || \
            warn "Citation claim failed: ${claim:0:40}..."

        if [ $((sub_idx % 3)) -eq 0 ]; then
            wait_blocks 1
        fi
    done

    wait_blocks 3
    fact_count=$(legbled query knowledge list-facts --limit 100 2>/dev/null | jq '.facts | length' 2>/dev/null || echo "?")
    ok "Total facts on chain: $fact_count"
}

# ---------------------------------------------------------------------------
# Phase 4: Oracle Predictions
# ---------------------------------------------------------------------------
phase_4() {
    phase "4" "ORACLE PREDICTIONS (stake-lock-resolve)"

    local current_height
    current_height=$(get_block_height)
    local resolve_height=$((current_height + 500))  # ~21 min from now

    local predictions=(
        "Bitcoin price will exceed 100000 USD by end of Q2 2026|75|economics"
        "Ethereum will implement full sharding by end of 2026|30|computer_science"
        "Global AI compute demand will double year over year through 2027|85|computer_science"
        "UK inflation will remain below 5 percent through June 2026|60|economics"
        "At least one major central bank will launch a CBDC by end of 2026|55|economics"
        "Anthropic will release a model surpassing GPT-5 benchmarks by Q3 2026|70|computer_science"
        "Zerone testnet will have 10 or more external validators by Q3 2026|65|computer_science"
        "Renewable energy will exceed 40 percent of global electricity generation by 2027|50|physics"
    )

    local sub_idx=0
    local submitters=("test1" "test2" "asha" "beta-agent")

    for pred_data in "${predictions[@]}"; do
        IFS='|' read -r statement probability domain <<< "$pred_data"
        local submitter="${submitters[$((sub_idx % ${#submitters[@]}))]}"
        sub_idx=$((sub_idx + 1))

        # Probability in basis points (75 = 750000 bps)
        local prob_bps=$((probability * 10000))
        # Stake in ulgm (plain integer)
        local stake_ulgm=$((5 * LGM))

        # Positional args: [claim] [domain] [probability-bps] [stake-ulgm] [resolution-block]
        legbled_tx tx oracle submit-prediction \
            "$statement" \
            "$domain" \
            "$prob_bps" \
            "$stake_ulgm" \
            "$resolve_height" \
            --from "$submitter" >/dev/null 2>&1 && \
            ok "Prediction [${probability}%]: ${statement:0:60}..." || \
            warn "Prediction failed: ${statement:0:40}..."

        wait_blocks 1
    done

    ok "Oracle predictions submitted (resolve at block $resolve_height)"
}

# ---------------------------------------------------------------------------
# Phase 5: Partnership Formation
# ---------------------------------------------------------------------------
phase_5() {
    phase "5" "PARTNERSHIPS (tier staking + common pots)"

    # Partnership pairs: [initiator, partner, tier]
    local partnerships=(
        "asha:nuance:1"
        "asha:forge:0"
        "alpha-agent:beta-agent:2"
        "beta-agent:gamma-agent:1"
        "test1:asha:0"
        "test2:nuance:0"
    )

    for pair in "${partnerships[@]}"; do
        IFS=':' read -r initiator partner tier <<< "$pair"

        info "Forming partnership: $initiator <-> $partner (tier $tier)"

        # Initiate (positional: [partner-address] [lock-tier])
        legbled_tx tx partnerships initiate \
            "$(addr_of "$partner")" \
            "$tier" \
            --from "$initiator" >/dev/null 2>&1 && \
            ok "Partnership initiated: $initiator -> $partner" || \
            warn "Partnership initiation failed: $initiator -> $partner"

        wait_blocks 1

        # Accept
        # Need to find the partnership ID -- use quick-accept if available
        legbled_tx tx quick-accept \
            "$(addr_of "$initiator")" \
            --from "$partner" >/dev/null 2>&1 && \
            ok "Partnership accepted: $partner accepts $initiator" || \
            warn "Partnership acceptance failed (may need manual ID)"

        wait_blocks 1

        # Deposit into common pot
        legbled_tx tx partnerships deposit \
            "$(addr_of "$partner")" \
            "$((10 * LGM))${DENOM}" \
            --from "$initiator" >/dev/null 2>&1 && \
            ok "Deposited 10 LGM to pot: $initiator" || \
            warn "Deposit failed"
    done
}

# ---------------------------------------------------------------------------
# Phase 6: Compute Pool Activation
# ---------------------------------------------------------------------------
phase_6() {
    phase "6" "COMPUTE POOL (provider registration + credits)"

    local providers=(
        "forge:ai_inference:compute.forge.kingdom:100"
        "sentry:monitoring:monitor.sentry.kingdom:50"
        "lark:ai_inference:compute.lark.kingdom:75"
    )

    for prov_data in "${providers[@]}"; do
        IFS=':' read -r name service endpoint price <<< "$prov_data"

        info "Registering compute provider: $name ($service)"

        ensure_key "$name"

        # Fund if needed
        local dashboard
        dashboard=$(legbled query dashboard "$(addr_of "$name")" 2>/dev/null || echo "Balance: 0")
        local bal_lgm
        bal_lgm=$(echo "$dashboard" | grep -o 'Balance: [0-9]*' | grep -o '[0-9]*' || echo "0")
        if [ "$bal_lgm" -lt 200 ] 2>/dev/null; then
            fund_account "$name" "$((500 * LGM))"
        fi

        # Positional: [service-type] [endpoint] [price-per-cu] [stake-amount]
        legbled_tx tx compute_pool register-provider \
            "$service" \
            "$endpoint" \
            "${price}" \
            "$((100 * LGM))" \
            --from "$name" >/dev/null 2>&1 && \
            ok "Provider registered: $name ($service @ $endpoint)" || \
            warn "Provider registration failed: $name"

        wait_blocks 1

        # Send heartbeat
        legbled_tx tx compute_pool heartbeat \
            --from "$name" >/dev/null 2>&1 && \
            ok "Provider heartbeat: $name" || \
            warn "Heartbeat failed: $name"
    done
}

# ---------------------------------------------------------------------------
# Phase 7: Challenge & Dispute Activation
# ---------------------------------------------------------------------------
phase_7() {
    phase "7" "CHALLENGES & DISPUTES"

    info "Querying provisional facts for challenge targets..."

    # Get list of facts
    local facts
    facts=$(legbled query knowledge list-facts --status provisional --limit 10 2>/dev/null || echo "")

    if [ -z "$facts" ] || [ "$facts" = "{}" ]; then
        warn "No provisional facts found to challenge. Trying all facts..."
        facts=$(legbled query knowledge list-facts --limit 10 2>/dev/null || echo "")
    fi

    # Challenge a few facts to activate the adversarial pipeline
    local challengers=("test3" "test2" "test1")
    local challenge_count=0

    # Challenge the first 3 empirical claims (more likely to be challengeable)
    for i in 0 1 2; do
        local fact_id
        fact_id=$(echo "$facts" | jq -r ".facts[$i].id // empty" 2>/dev/null || echo "")

        if [ -z "$fact_id" ]; then
            continue
        fi

        local challenger="${challengers[$((i % ${#challengers[@]}))]}"

        info "Challenging fact $fact_id with $challenger..."

        legbled_tx tx knowledge challenge-provisional \
            "$fact_id" \
            --reason "Adversarial verification test - challenging for completeness" \
            --stake "1LGM" \
            --from "$challenger" >/dev/null 2>&1 && \
            ok "Challenge submitted for fact $fact_id" && \
            challenge_count=$((challenge_count + 1)) || \
            warn "Challenge failed for fact $fact_id"

        wait_blocks 1
    done

    ok "Challenges submitted: $challenge_count"
}

# ---------------------------------------------------------------------------
# Phase 8: Vesting Lifecycle (truth-linked vesting)
# ---------------------------------------------------------------------------
phase_8() {
    phase "8" "VESTING LIFECYCLE (truth-linked vesting schedules)"

    # ── 8a. Submit high-stake claims across epistemic categories ──────────
    info "Submitting 5 high-stake claims (10 LGM each)..."

    # Use analytic/formal + propositional for high base scores (700k → provisional)
    # Use empirical for variety (500k base, may need evidence)
    local vesting_claims=(
        "test1:Logical conjunction is commutative: P AND Q is equivalent to Q AND P:logic:analytic:propositional"
        "test2:The halting problem is undecidable for general Turing machines:computer_science:formal:propositional"
        "test3:Bayes theorem relates conditional probabilities via prior and likelihood:mathematics:formal:propositional"
        "asha:In any finite game with perfect information a Nash equilibrium exists:economics:analytic:propositional"
        "nuance:Semantic compositionality: the meaning of a compound expression is determined by its parts:linguistics:analytic:propositional"
    )

    local claim_submitters=()
    for claim_data in "${vesting_claims[@]}"; do
        IFS=':' read -r submitter claim domain category logic_zone <<< "$claim_data"
        claim_submitters+=("$submitter")

        ensure_key "$submitter"

        # Fund if needed (need at least 10 LGM stake + fees)
        local addr
        addr=$(addr_of "$submitter")
        local bal
        bal=$(legbled query bank balances "$addr" -o json 2>/dev/null | jq -r '.balances[]? | select(.denom=="ulgm") | .amount // "0"' 2>/dev/null || echo "0")
        if [ "${bal:-0}" -lt "$((15 * LGM))" ] 2>/dev/null; then
            fund_account "$submitter" "$((50 * LGM))"
        fi

        legbled_tx tx claim "$claim" \
            --domain "$domain" \
            --category "$category" \
            --logic-zone "$logic_zone" \
            --stake "10LGM" \
            --from "$submitter" >/dev/null 2>&1 && \
            ok "High-stake claim [$domain/$category]: ${claim:0:60}..." || \
            warn "Claim failed: ${claim:0:50}..."

        wait_blocks 1
    done

    # ── 8b. Wait for evaluation (BeginBlock processes pending claims) ─────
    info "Waiting for BeginBlock evaluation (6 blocks)..."
    wait_blocks 6

    # ── 8c. Query vesting schedules ──────────────────────────────────────
    info "Querying vesting reward parameters..."
    local vesting_params
    vesting_params=$(legbled query vesting_rewards params -o json 2>/dev/null || echo "{}")
    if [ -n "$vesting_params" ] && [ "$vesting_params" != "{}" ]; then
        ok "Vesting params: $vesting_params"
    else
        warn "Could not query vesting params"
    fi

    info "Querying active vesting schedules..."
    local active_schedules
    active_schedules=$(legbled query vesting_rewards active-schedules --limit 50 -o json 2>/dev/null || echo "{}")
    local schedule_count
    schedule_count=$(echo "$active_schedules" | jq '.schedules | length' 2>/dev/null || echo "0")
    ok "Active vesting schedules: $schedule_count"

    # Query per-recipient schedules
    local total_vesting_amount=0
    for submitter in "${claim_submitters[@]}"; do
        local addr
        addr=$(addr_of "$submitter")
        local recipient_schedules
        recipient_schedules=$(legbled query vesting_rewards schedules-by-recipient "$addr" -o json 2>/dev/null || echo "{}")
        local rcpt_count
        rcpt_count=$(echo "$recipient_schedules" | jq '.schedules | length' 2>/dev/null || echo "0")
        if [ "$rcpt_count" -gt 0 ]; then
            local rcpt_total
            rcpt_total=$(echo "$recipient_schedules" | jq '[.schedules[].total_amount // 0] | add' 2>/dev/null || echo "0")
            total_vesting_amount=$((total_vesting_amount + rcpt_total))
            ok "  $submitter ($addr): $rcpt_count schedule(s), total amount: $rcpt_total"
        else
            info "  $submitter: no vesting schedules (claim may not have reached provisional)"
        fi
    done

    # ── 8d. Wait for cliff period ────────────────────────────────────────
    # Extract cliff duration from params (in blocks). If not available, use default.
    local cliff_blocks
    # Cliff is per-category; extract the minimum cliff from category_configs
    cliff_blocks=$(echo "$vesting_params" | jq -r '
        [.category_configs[]?.cliff_blocks // empty] | min // empty
    ' 2>/dev/null || echo "")
    if [ -z "$cliff_blocks" ] || [ "$cliff_blocks" = "null" ]; then
        cliff_blocks=5  # default guess for devnet
        info "Cliff duration not found in params, assuming $cliff_blocks blocks"
    else
        ok "Minimum cliff from category configs: $cliff_blocks blocks"
        # On devnet with thousands-block cliffs, cap the wait for practicality
        if [ "$cliff_blocks" -gt 20 ]; then
            info "Cliff too long for live test ($cliff_blocks blocks); capping wait at 10 blocks"
            cliff_blocks=10
        fi
    fi

    if [ "$schedule_count" -gt 0 ]; then
        info "Waiting for cliff period ($cliff_blocks blocks)..."
        wait_blocks "$cliff_blocks"
    else
        info "No vesting schedules found -- skipping cliff wait"
    fi

    # ── 8e. Claim vesting releases ───────────────────────────────────────
    local claim_success=0
    local claim_fail=0
    if [ "$schedule_count" -gt 0 ]; then
        info "Claiming vested rewards..."
        for submitter in "${claim_submitters[@]}"; do
            # claim-vesting with no ID claims all available for the sender
            legbled_tx tx vesting_rewards claim-vesting \
                --from "$submitter" >/dev/null 2>&1 && \
                { ok "Vesting claimed for $submitter"; claim_success=$((claim_success + 1)); } || \
                { warn "Vesting claim failed or nothing vested yet for $submitter"; claim_fail=$((claim_fail + 1)); }

            wait_blocks 1
        done
    else
        info "Skipping vesting claims -- no active schedules to claim from"
    fi

    # ── 8f. Submit counter-facts (falsification flow) ────────────────────
    info "Querying facts for contradiction targets..."
    local facts_json
    facts_json=$(legbled query knowledge facts --limit 20 -o json 2>/dev/null || echo "{}")
    local fact_count
    fact_count=$(echo "$facts_json" | jq '.facts | length' 2>/dev/null || echo "0")

    local contradictions_submitted=0
    local clawbacks_triggered=0
    if [ "$fact_count" -gt 1 ]; then
        # Pick 2 facts: contradict fact[0] using fact[1] as evidence, and vice versa
        local target_ids=()
        local evidence_ids=()
        for i in 0 1; do
            local fid
            fid=$(echo "$facts_json" | jq -r ".facts[$i].id // empty" 2>/dev/null || echo "")
            if [ -n "$fid" ]; then
                target_ids+=("$fid")
            fi
        done
        # Use the other fact as evidence for each contradiction
        if [ "${#target_ids[@]}" -ge 2 ]; then
            evidence_ids=("${target_ids[1]}" "${target_ids[0]}")
        elif [ "${#target_ids[@]}" -eq 1 ]; then
            evidence_ids=("${target_ids[0]}")
        fi

        local contradiction_submitters=("test3" "test2")
        local contra_idx=0
        for target_id in "${target_ids[@]}"; do
            local contra_submitter="${contradiction_submitters[$((contra_idx % ${#contradiction_submitters[@]}))]}"
            local evidence_id="${evidence_ids[$((contra_idx % ${#evidence_ids[@]}))]}"
            contra_idx=$((contra_idx + 1))

            info "Submitting contradiction against fact $target_id (evidence: $evidence_id)..."

            # Ensure submitter is funded
            ensure_key "$contra_submitter"
            local contra_addr
            contra_addr=$(addr_of "$contra_submitter")
            local contra_bal
            contra_bal=$(legbled query bank balances "$contra_addr" -o json 2>/dev/null | jq -r '.balances[]? | select(.denom=="ulgm") | .amount // "0"' 2>/dev/null || echo "0")
            if [ "${contra_bal:-0}" -lt "$((10 * LGM))" ] 2>/dev/null; then
                fund_account "$contra_submitter" "$((50 * LGM))"
            fi

            legbled_tx tx knowledge submit-contradiction \
                "$target_id" \
                "Counter-evidence: this claim lacks sufficient formal proof and contains implicit assumptions that undermine its validity" \
                "$((5 * LGM))" \
                --evidence "$evidence_id" \
                --logic-zone "propositional" \
                --from "$contra_submitter" >/dev/null 2>&1 && \
                { ok "Contradiction submitted against fact $target_id"; contradictions_submitted=$((contradictions_submitted + 1)); } || \
                warn "Contradiction failed for fact $target_id"

            wait_blocks 2
        done

        # Wait for falsification evaluation
        if [ "$contradictions_submitted" -gt 0 ]; then
            info "Waiting for falsification evaluation (5 blocks)..."
            wait_blocks 5

            # Check if any vesting schedules were paused/clawed back
            local post_schedules
            post_schedules=$(legbled query vesting_rewards active-schedules --limit 50 -o json 2>/dev/null || echo "{}")
            local post_count
            post_count=$(echo "$post_schedules" | jq '.schedules | length' 2>/dev/null || echo "0")
            if [ "$post_count" -lt "$schedule_count" ]; then
                clawbacks_triggered=$((schedule_count - post_count))
                ok "Vesting clawbacks detected: $clawbacks_triggered schedule(s) removed/paused"
            else
                info "No vesting clawbacks detected (contradictions may have been rejected)"
            fi
        fi
    else
        warn "No facts found to contradict"
    fi

    # ── 8g. Query block-reward (token meter proxy) ───────────────────────
    # NOTE: No dedicated 'token-meter' query exists. Using 'block-reward'
    # at the current height as the closest available metric.
    info "Querying block reward distribution (token meter proxy)..."
    local current_height
    current_height=$(get_block_height)
    local block_reward
    block_reward=$(legbled query vesting_rewards block-reward "$current_height" -o json 2>/dev/null || echo "{}")
    if [ -n "$block_reward" ] && [ "$block_reward" != "{}" ]; then
        ok "Block reward at height $current_height: $block_reward"
    else
        warn "Block reward query returned empty (may not have vesting distributions at this height)"
        # Try a few recent blocks
        for offset in 1 2 5 10; do
            local try_height=$((current_height - offset))
            block_reward=$(legbled query vesting_rewards block-reward "$try_height" -o json 2>/dev/null || echo "{}")
            if [ -n "$block_reward" ] && [ "$block_reward" != "{}" ] && [ "$block_reward" != "null" ]; then
                ok "Block reward at height $try_height: $block_reward"
                break
            fi
        done
    fi

    # ── 8h. Summary report ───────────────────────────────────────────────
    echo ""
    echo "┌─────────────────────────────────────────────────────────┐"
    echo "│          PHASE 8 — VESTING LIFECYCLE SUMMARY            │"
    echo "├─────────────────────────────────────────────────────────┤"
    echo "│  High-stake claims submitted:  ${#vesting_claims[@]}                        │"
    echo "│  Vesting schedules created:    $schedule_count                        │"
    echo "│  Total vesting amount (ulgm):  $total_vesting_amount                        │"
    echo "│  Vesting claims succeeded:     $claim_success                        │"
    echo "│  Vesting claims failed/empty:  $claim_fail                        │"
    echo "│  Contradictions submitted:     $contradictions_submitted                        │"
    echo "│  Clawbacks triggered:          $clawbacks_triggered                        │"
    echo "└─────────────────────────────────────────────────────────┘"
    echo ""

    # Document gaps
    if [ "$schedule_count" -eq 0 ]; then
        warn "GAP: No vesting schedules were auto-created from accepted claims."
        warn "  Possible causes:"
        warn "  - Claims may not have reached provisional/verified status"
        warn "  - Auto-vesting may require governance or explicit create-vesting tx"
        warn "  - BeginBlock vesting creation may not be wired yet"
    fi
    info "NOTE: 'token-meter' query does not exist; used 'block-reward' as proxy"
}

# ---------------------------------------------------------------------------
# Status
# ---------------------------------------------------------------------------
show_status() {
    check_node

    echo ""
    echo "╔═══════════════════════════════════════════════════════╗"
    echo "║          ZERONE GROWTH ENGINE — STATUS                ║"
    echo "╚═══════════════════════════════════════════════════════╝"
    echo ""

    local height
    height=$(get_block_height)
    info "Block height: $height"

    # Validator count
    local validators
    validators=$(legbled query legible_staking validators --limit 50 2>/dev/null | jq '.validators | length' 2>/dev/null || echo "?")
    info "Validators: $validators"

    # Fact count
    local facts
    facts=$(legbled query knowledge list-facts --limit 1000 2>/dev/null | jq '.facts | length' 2>/dev/null || echo "?")
    info "Facts on chain: $facts"

    # Account count (via discovery)
    local agents
    agents=$(legbled query discovery list-agents --limit 100 2>/dev/null | jq '.agents | length' 2>/dev/null || echo "?")
    info "Registered agents: $agents"

    # Partnership count
    local partnerships
    partnerships=$(legbled query partnerships list --limit 100 2>/dev/null | jq '.partnerships | length' 2>/dev/null || echo "?")
    info "Active partnerships: $partnerships"

    # Validator balance as proxy
    local val_addr
    val_addr=$(addr_of "validator" 2>/dev/null || echo "")
    if [ -n "$val_addr" ]; then
        local val_dashboard
        val_dashboard=$(legbled query dashboard "$val_addr" 2>/dev/null || echo "")
        local val_bal
        val_bal=$(echo "$val_dashboard" | grep -o 'Balance: [0-9]*' | grep -o '[0-9]*' || echo "?")
        info "Validator balance: $val_bal LGM"
    fi

    # Vesting schedules
    local vest_count
    vest_count=$(legbled query vesting_rewards active-schedules --limit 100 -o json 2>/dev/null | jq '.schedules | length' 2>/dev/null || echo "?")
    info "Active vesting schedules: $vest_count"

    # Block reward at current height (token meter proxy)
    local cur_height
    cur_height=$(get_block_height)
    local blk_reward
    blk_reward=$(legbled query vesting_rewards block-reward "$cur_height" -o json 2>/dev/null || echo "")
    if [ -n "$blk_reward" ] && [ "$blk_reward" != "{}" ] && [ "$blk_reward" != "null" ]; then
        info "Block reward (height $cur_height): $blk_reward"
    fi

    echo ""
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    local cmd="${1:-full}"
    local phase_num="${2:-}"

    case "$cmd" in
        full)
            phase_0
            wait_blocks 5
            phase_1
            phase_2
            phase_3
            phase_4
            phase_5
            phase_6
            phase_7
            phase_8
            echo ""
            show_status
            ;;
        phase)
            check_node
            case "$phase_num" in
                0) phase_0 ;;
                1) phase_1 ;;
                2) phase_2 ;;
                3) phase_3 ;;
                4) phase_4 ;;
                5) phase_5 ;;
                6) phase_6 ;;
                7) phase_7 ;;
                8) phase_8 ;;
                *) err "Unknown phase: $phase_num (valid: 0-8)"; exit 1 ;;
            esac
            ;;
        status)
            show_status
            ;;
        *)
            echo "Usage: $0 {full|phase N|status}"
            exit 1
            ;;
    esac
}

main "$@"
