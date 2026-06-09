#!/usr/bin/env bash
#
# truth-farm.sh — Zerone Truth Farm
#
# Reads genesis axiom files, generates derivative claims with citations,
# and submits them to the devnet in domain-balanced batches.
#
# Usage:
#   ./scripts/truth-farm.sh seed          Seed axioms as live claims
#   ./scripts/truth-farm.sh cultivate     Submit derivative claims citing existing facts
#   ./scripts/truth-farm.sh harvest       Query and report verified fact yields
#   ./scripts/truth-farm.sh cycle         Run one full farm cycle (seed -> cultivate -> harvest)
#   ./scripts/truth-farm.sh status        Show farm metrics
#
# Requirements: legbled binary at build/legbled, devnet running

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
AXIOM_DIR="$PROJECT_ROOT/genesis-axioms"
STATE_FILE="$NODE_HOME/truth-farm-state.json"

# Common flags for all tx commands
TX_FLAGS="--chain-id $CHAIN_ID --keyring-backend $KEYRING --home $NODE_HOME --gas 500000 --fees 500000$DENOM -y -o json"

# Submitter accounts (rotated through)
SUBMITTERS=("asha" "nuance" "forge" "lark" "sentry" "patch" "test1" "test2" "test3")

# Lookup functions (bash 3.x compatible, no associative arrays)

# Domain mapping: file prefix -> chain domain
domain_of() {
    case "$1" in
        LOGIC) echo "logic" ;;
        ECON)  echo "economics" ;;
        CS)    echo "computer_science" ;;
        PHYS)  echo "physics" ;;
        BIO)   echo "biology" ;;
        MATH)  echo "mathematics" ;;
        INFO)  echo "computer_science" ;;
        CHEM)  echo "physics" ;;
        ETHIC) echo "philosophy" ;;
        PSYCH) echo "sociology" ;;
        PHIL)  echo "philosophy" ;;
        LING)  echo "linguistics" ;;
        COSM)  echo "physics" ;;
        THEO)  echo "philosophy" ;;
        AGRT)  echo "philosophy" ;;
        *)     echo "unknown" ;;
    esac
}

# Type -> epistemic category
category_of() {
    case "$1" in
        axiom)              echo "analytic" ;;
        definition)         echo "formal" ;;
        derived_claim)      echo "formal" ;;
        empirical_axiom)    echo "empirical" ;;
        measurement_fact)   echo "empirical" ;;
        regime_declaration) echo "formal" ;;
        meta)               echo "synthetic" ;;
        *)                  echo "synthetic" ;;
    esac
}

# Type -> logic zone
zone_of() {
    case "$1" in
        axiom)              echo "propositional" ;;
        definition)         echo "propositional" ;;
        derived_claim)      echo "first_order" ;;
        empirical_axiom)    echo "empirical" ;;
        measurement_fact)   echo "empirical" ;;
        regime_declaration) echo "propositional" ;;
        meta)               echo "modal" ;;
        *)                  echo "propositional" ;;
    esac
}

# Type -> stake amount
stake_of() {
    case "$1" in
        axiom)         echo "2LGM" ;;
        derived_claim) echo "2LGM" ;;
        *)             echo "1LGM" ;;
    esac
}

# ---------------------------------------------------------------------------
# Helpers (same patterns as growth-engine.sh)
# ---------------------------------------------------------------------------
info()  { echo -e "\033[1;34m[FARM]\033[0m $*"; }
ok()    { echo -e "\033[1;32m  [OK]\033[0m $*"; }
warn()  { echo -e "\033[1;33m  [!!]\033[0m $*"; }
err()   { echo -e "\033[1;31m  [ERR]\033[0m $*" >&2; }
header() { echo -e "\n\033[1;35m═══════════════════════════════════════════════════\033[0m"; echo -e "\033[1;35m  $1\033[0m"; echo -e "\033[1;35m═══════════════════════════════════════════════════\033[0m\n"; }

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

get_block_height() {
    curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height // "0"'
}

check_node() {
    if ! curl -s --connect-timeout 2 http://localhost:26657/status >/dev/null 2>&1; then
        err "Node not running. Start devnet first: make devnet"
        exit 1
    fi
}

# Submitter rotation index (global file-based counter for subshell persistence)
SUBMITTER_IDX_FILE="/tmp/truth-farm-submitter-idx"
next_submitter() {
    local idx=0
    if [ -f "$SUBMITTER_IDX_FILE" ]; then
        idx=$(cat "$SUBMITTER_IDX_FILE" 2>/dev/null || echo "0")
    fi
    local sub="${SUBMITTERS[$idx]}"
    idx=$(( (idx + 1) % ${#SUBMITTERS[@]} ))
    echo "$idx" > "$SUBMITTER_IDX_FILE"
    echo "$sub"
}

# ---------------------------------------------------------------------------
# State management
# ---------------------------------------------------------------------------
init_state() {
    if [ ! -f "$STATE_FILE" ]; then
        mkdir -p "$(dirname "$STATE_FILE")"
        echo '{"seeded":[],"last_seed":"","total_cultivated":0,"total_harvested":0}' > "$STATE_FILE"
        info "Initialized farm state at $STATE_FILE"
    fi
}

read_state() {
    init_state
    cat "$STATE_FILE"
}

is_seeded() {
    local axiom_id="$1"
    local state
    state=$(read_state)
    echo "$state" | jq -r --arg id "$axiom_id" '.seeded | index($id) // empty' 2>/dev/null
}

mark_seeded() {
    local axiom_id="$1"
    local state
    state=$(read_state)
    echo "$state" | jq --arg id "$axiom_id" --arg date "$(date +%Y-%m-%d)" \
        '.seeded += [$id] | .last_seed = $date' > "${STATE_FILE}.tmp"
    mv "${STATE_FILE}.tmp" "$STATE_FILE"
}

increment_cultivated() {
    local count="${1:-1}"
    local state
    state=$(read_state)
    echo "$state" | jq --argjson n "$count" '.total_cultivated += $n' > "${STATE_FILE}.tmp"
    mv "${STATE_FILE}.tmp" "$STATE_FILE"
}

increment_harvested() {
    local count="${1:-1}"
    local state
    state=$(read_state)
    echo "$state" | jq --argjson n "$count" '.total_harvested += $n' > "${STATE_FILE}.tmp"
    mv "${STATE_FILE}.tmp" "$STATE_FILE"
}

# ---------------------------------------------------------------------------
# Axiom parsing
# ---------------------------------------------------------------------------
# Parses a genesis axiom markdown file and emits one line per axiom:
#   AXIOM_ID|TYPE|STATEMENT|DEPENDS
# e.g.:
#   LOGIC-001|definition|A countably infinite set Var = ...|MATH-012
parse_axiom_file() {
    local file="$1"
    local prefix
    prefix=$(basename "$file" .md)

    # Only parse .md files that have axiom entries
    if [ ! -f "$file" ] || [ "$prefix" = "README" ] || [ "$prefix" = "SCHEMA" ]; then
        return
    fi

    # Extract blocks inside ``` fences that contain axiom IDs
    local in_block=0
    local current_id=""
    local current_type=""
    local current_statement=""
    local current_depends=""

    while IFS= read -r line || [ -n "$line" ]; do
        # Track code fences
        case "$line" in
            '```'*)
                if [ "$in_block" -eq 0 ]; then
                    in_block=1
                else
                    # Closing fence -- emit current entry if any
                    if [ -n "$current_id" ]; then
                        current_statement=$(echo "$current_statement" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//;s/  */ /g' | head -c 200)
                        echo "${current_id}|${current_type}|${current_statement}|${current_depends}"
                        current_id=""
                        current_type=""
                        current_statement=""
                        current_depends=""
                    fi
                    in_block=0
                fi
                continue
                ;;
        esac

        if [ "$in_block" -eq 0 ]; then
            continue
        fi

        # Match axiom header lines like: LOGIC-001 (Definition, ...) or CS-005a (Meta, ...)
        # Use sed to extract the ID and type from header lines
        local parsed_id=""
        parsed_id=$(echo "$line" | sed -n 's/^[[:space:]]*\([A-Z]*-[0-9][0-9]*[a-z]*\)[[:space:]]*(.*$/\1/p')

        if [ -n "$parsed_id" ]; then
            # Emit previous entry
            if [ -n "$current_id" ]; then
                current_statement=$(echo "$current_statement" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//;s/  */ /g' | head -c 200)
                echo "${current_id}|${current_type}|${current_statement}|${current_depends}"
            fi

            current_id="$parsed_id"
            # Extract type from parenthetical: e.g., "(Definition," or "(Derived Claim,"
            local raw_type=""
            raw_type=$(echo "$line" | sed -n 's/^[^(]*(\([^,)]*\).*/\1/p' | tr '[:upper:]' '[:lower:]' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
            case "$raw_type" in
                axiom)              current_type="axiom" ;;
                definition)         current_type="definition" ;;
                "derived claim")    current_type="derived_claim" ;;
                "empirical axiom")  current_type="empirical_axiom" ;;
                "measurement fact") current_type="measurement_fact" ;;
                "regime declaration") current_type="regime_declaration" ;;
                meta)               current_type="meta" ;;
                *)                  current_type="meta" ;;
            esac
            current_statement=""
            current_depends=""
            continue
        fi

        # Match depends line: depends: [LOGIC-001, LOGIC-002]
        case "$line" in
            *depends:*)
                current_depends=$(echo "$line" | sed 's/.*depends:[[:space:]]*\[//;s/\].*//' | sed 's/[" ]//g')
                continue
                ;;
        esac

        # Match explicit type: field
        case "$line" in
            *type:*)
                local explicit_type=""
                explicit_type=$(echo "$line" | sed 's/.*type:[[:space:]]*//' | sed 's/[[:space:]]*$//')
                case "$explicit_type" in
                    axiom|definition|derived_claim|empirical_axiom|measurement_fact|regime_declaration|meta)
                        current_type="$explicit_type" ;;
                esac
                continue
                ;;
        esac

        # Skip metadata lines
        case "$line" in
            *domain:*|*confidence:*|*formal:*|*source_tradition:*|*notes:*|*regime:*|*evidence_ref:*|*derivation_ref:*|*approximation_tags:*)
                continue
                ;;
        esac

        # Accumulate statement text
        if [ -n "$current_id" ]; then
            local trimmed=""
            trimmed=$(echo "$line" | sed 's/^[[:space:]]*//')
            if [ -n "$trimmed" ]; then
                if [ -z "$current_statement" ]; then
                    current_statement="$trimmed"
                else
                    current_statement="$current_statement $trimmed"
                fi
            fi
        fi
    done < "$file"

    # Emit last entry if inside a block that was never closed
    if [ -n "$current_id" ]; then
        current_statement=$(echo "$current_statement" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//;s/  */ /g' | head -c 200)
        echo "${current_id}|${current_type}|${current_statement}|${current_depends}"
    fi
}

# ---------------------------------------------------------------------------
# SEED: Parse axioms and submit as claims
# ---------------------------------------------------------------------------
do_seed() {
    header "TRUTH FARM: SEED"
    check_node
    init_state

    local total_seeded=0
    local total_skipped=0
    local total_failed=0

    # Find all axiom markdown files
    for axiom_file in "$AXIOM_DIR"/*.md; do
        local prefix
        prefix=$(basename "$axiom_file" .md)

        # Skip non-axiom files
        if [ "$prefix" = "README" ] || [ "$prefix" = "SCHEMA" ]; then
            continue
        fi

        local chain_domain
        chain_domain=$(domain_of "$prefix")
        if [ "$chain_domain" = "unknown" ]; then
            warn "No domain mapping for $prefix, skipping"
            continue
        fi

        info "Parsing $prefix.md -> domain=$chain_domain"

        # Parse and submit each axiom
        while IFS='|' read -r axiom_id axiom_type statement depends; do
            # Skip empty lines
            [ -z "$axiom_id" ] && continue

            # Check if already seeded
            if [ -n "$(is_seeded "$axiom_id")" ]; then
                total_skipped=$((total_skipped + 1))
                continue
            fi

            # Map type to category and zone
            local category
            category=$(category_of "$axiom_type")
            local zone
            zone=$(zone_of "$axiom_type")
            local stake
            stake=$(stake_of "$axiom_type")

            # Pick submitter
            local submitter
            submitter=$(next_submitter)

            # Truncate statement for the claim (chain has limits)
            local claim_text
            claim_text=$(echo "$statement" | head -c 180)

            # Build refs flag from depends
            local refs_flag=""
            if [ -n "$depends" ]; then
                # Use first dependency as ref (chain expects fact IDs, but we cite axiom IDs as symbolic refs)
                refs_flag="--refs $depends"
            fi

            info "Seeding: $axiom_id ($axiom_type) via $submitter"

            # Submit claim
            local result
            result=$("$BINARY" tx knowledge submit-claim "$claim_text" \
                "$chain_domain" \
                "$category" \
                "$stake" \
                --logic-zone "$zone" \
                --from "$submitter" \
                $refs_flag \
                $TX_FLAGS 2>&1 | tail -1) || true

            # Check for success
            local code
            code=$(echo "$result" | jq -r '.code // "999"' 2>/dev/null || echo "999")

            if [ "$code" = "0" ] || [ "$code" = "null" ]; then
                mark_seeded "$axiom_id"
                total_seeded=$((total_seeded + 1))
                ok "$axiom_id seeded ($chain_domain/$category)"
            else
                total_failed=$((total_failed + 1))
                warn "$axiom_id failed (code=$code)"
            fi

            # Brief pause to avoid sequence mismatch (3s ~ 1 block)
            sleep 3

        done < <(parse_axiom_file "$axiom_file")
    done

    echo ""
    info "Seed complete: $total_seeded seeded, $total_skipped skipped (already seeded), $total_failed failed"
}

# ---------------------------------------------------------------------------
# CULTIVATE: Generate derivative claims from existing facts
# ---------------------------------------------------------------------------
do_cultivate() {
    header "TRUTH FARM: CULTIVATE"
    check_node
    init_state

    local total_cultivated=0

    # Query existing facts on chain
    info "Querying existing facts on chain..."
    local facts_json
    facts_json=$(legbled query knowledge facts --limit 100 -o json 2>/dev/null || echo '{"facts":[]}')

    local fact_count
    fact_count=$(echo "$facts_json" | jq '.facts | length' 2>/dev/null || echo "0")
    info "Found $fact_count facts on chain"

    if [ "$fact_count" -eq 0 ]; then
        warn "No facts found. Run 'seed' first."
        return 0
    fi

    # For each fact, generate a derivative claim
    local max_derivatives=20
    local count=0

    while [ $count -lt "$fact_count" ] && [ $count -lt "$max_derivatives" ]; do
        local fact_id
        fact_id=$(echo "$facts_json" | jq -r ".facts[$count].id // empty" 2>/dev/null || echo "")
        local fact_body
        fact_body=$(echo "$facts_json" | jq -r ".facts[$count].body // .facts[$count].statement // empty" 2>/dev/null || echo "")
        local fact_domain
        fact_domain=$(echo "$facts_json" | jq -r ".facts[$count].domain // \"logic\"" 2>/dev/null || echo "logic")
        local fact_confidence
        fact_confidence=$(echo "$facts_json" | jq -r ".facts[$count].confidence // \"0.9\"" 2>/dev/null || echo "0.9")

        if [ -z "$fact_id" ] || [ -z "$fact_body" ]; then
            count=$((count + 1))
            continue
        fi

        # Only cultivate high-confidence facts
        local conf_int
        conf_int=$(echo "$fact_confidence" | sed 's/\.//' | sed 's/^0*//' | head -c 2)
        if [ "${conf_int:-0}" -lt 80 ] 2>/dev/null; then
            count=$((count + 1))
            continue
        fi

        # Generate derivative claim text
        local short_body
        short_body=$(echo "$fact_body" | head -c 80)
        local derivative="Given that ${short_body}, it follows that this principle extends to related domains and has further implications for systematic knowledge construction"

        # Pick submitter and build derivative
        local submitter
        submitter=$(next_submitter)

        info "Cultivating from fact $fact_id via $submitter"

        local result
        result=$("$BINARY" tx knowledge submit-claim "$derivative" \
            "$fact_domain" \
            "formal" \
            "$LGM" \
            --logic-zone "propositional" \
            --refs "$fact_id" \
            --from "$submitter" \
            $TX_FLAGS 2>&1 | tail -1) || true

        local code
        code=$(echo "$result" | jq -r '.code // "999"' 2>/dev/null || echo "999")

        if [ "$code" = "0" ] || [ "$code" = "null" ]; then
            total_cultivated=$((total_cultivated + 1))
            ok "Derivative from $fact_id cultivated"
        else
            warn "Derivative from $fact_id failed (code=$code)"
        fi

        sleep 1
        count=$((count + 1))
    done

    increment_cultivated "$total_cultivated"
    echo ""
    info "Cultivate complete: $total_cultivated new derivative claims"
}

# ---------------------------------------------------------------------------
# HARVEST: Query verified facts and report yields
# ---------------------------------------------------------------------------
do_harvest() {
    header "TRUTH FARM: HARVEST"
    check_node
    init_state

    # Query all facts
    info "Querying fact yields..."
    local facts_json
    facts_json=$(legbled query claim list-facts --limit 500 -o json 2>/dev/null || echo '{"facts":[]}')

    local total_facts
    total_facts=$(echo "$facts_json" | jq '.facts | length' 2>/dev/null || echo "0")

    # Count per domain
    info "Counting per-domain yields..."
    local domains_raw
    domains_raw=$(echo "$facts_json" | jq -r '[.facts[].domain // "unknown"] | group_by(.) | map({domain: .[0], count: length}) | .[]' 2>/dev/null || echo "")

    # Domain yield table
    echo ""
    echo -e "\033[1;36m  Domain Yields:\033[0m"
    echo "  ────────────────────────────────────"
    if [ -n "$domains_raw" ]; then
        echo "$facts_json" | jq -r '
            [.facts[].domain // "unknown"]
            | group_by(.)
            | map({domain: .[0], count: length})
            | sort_by(-.count)
            | .[]
            | "  \(.domain)\t\(.count) facts"
        ' 2>/dev/null || echo "  (unable to parse domain data)"
    else
        echo "  (no facts found)"
    fi
    echo "  ────────────────────────────────────"
    echo -e "  \033[1mTotal facts: $total_facts\033[0m"

    # Citation depth (count facts with refs)
    local cited_count
    cited_count=$(echo "$facts_json" | jq '[.facts[] | select(.refs != null and (.refs | length) > 0)] | length' 2>/dev/null || echo "0")
    local avg_citations="0"
    if [ "$total_facts" -gt 0 ]; then
        avg_citations=$(echo "scale=2; $cited_count * 100 / $total_facts" | bc 2>/dev/null || echo "0")
    fi

    echo ""
    echo -e "\033[1;36m  Citation Metrics:\033[0m"
    echo "  ────────────────────────────────────"
    echo "  Facts with citations: $cited_count"
    echo "  Citation density:     ${avg_citations}%"
    echo "  ────────────────────────────────────"

    # Token consumption estimate
    local est_fees=$(( total_facts * 500000 ))
    local est_stakes=$(( total_facts * 1 * LGM ))
    local est_total=$(( est_fees + est_stakes ))
    local est_lgm=$(( est_total / LGM ))

    echo ""
    echo -e "\033[1;36m  Token Consumption:\033[0m"
    echo "  ────────────────────────────────────"
    echo "  Est. fees burned:     ${est_fees} ${DENOM}"
    echo "  Est. staked:          ${est_stakes} ${DENOM}"
    echo "  Est. total consumed:  ~${est_lgm} LGM"
    echo "  ────────────────────────────────────"

    increment_harvested "$total_facts"
    echo ""
    ok "Harvest complete"
}

# ---------------------------------------------------------------------------
# STATUS: Show farm state
# ---------------------------------------------------------------------------
do_status() {
    header "TRUTH FARM: STATUS"
    init_state

    local state
    state=$(read_state)

    local seeded_count
    seeded_count=$(echo "$state" | jq '.seeded | length' 2>/dev/null || echo "0")
    local last_seed
    last_seed=$(echo "$state" | jq -r '.last_seed // "never"' 2>/dev/null || echo "never")
    local total_cultivated
    total_cultivated=$(echo "$state" | jq -r '.total_cultivated // 0' 2>/dev/null || echo "0")
    local total_harvested
    total_harvested=$(echo "$state" | jq -r '.total_harvested // 0' 2>/dev/null || echo "0")

    echo -e "\033[1;36m  Farm State ($STATE_FILE):\033[0m"
    echo "  ────────────────────────────────────"
    echo "  Axioms seeded:      $seeded_count"
    echo "  Last seed date:     $last_seed"
    echo "  Total cultivated:   $total_cultivated"
    echo "  Total harvested:    $total_harvested"
    echo "  ────────────────────────────────────"

    # Show seeded axioms by domain prefix
    echo ""
    echo -e "\033[1;36m  Seeded by Domain:\033[0m"
    echo "  ────────────────────────────────────"
    echo "$state" | jq -r '
        .seeded
        | map(split("-")[0])
        | group_by(.)
        | map({prefix: .[0], count: length})
        | sort_by(-.count)
        | .[]
        | "  \(.prefix)\t\(.count) axioms"
    ' 2>/dev/null || echo "  (none yet)"
    echo "  ────────────────────────────────────"

    # Show available axiom files
    echo ""
    echo -e "\033[1;36m  Available Axiom Files:\033[0m"
    echo "  ────────────────────────────────────"
    for f in "$AXIOM_DIR"/*.md; do
        local base
        base=$(basename "$f" .md)
        if [ "$base" = "README" ] || [ "$base" = "SCHEMA" ]; then
            continue
        fi
        local count
        count=$(parse_axiom_file "$f" | wc -l | tr -d ' ')
        local domain
        domain=$(domain_of "$base")
        printf "  %-8s -> %-18s (%s axioms)\n" "$base" "$domain" "$count"
    done
    echo "  ────────────────────────────────────"

    # Chain status if available
    if curl -s --connect-timeout 2 http://localhost:26657/status >/dev/null 2>&1; then
        local height
        height=$(get_block_height)
        echo ""
        echo -e "\033[1;32m  Chain: ONLINE (block $height)\033[0m"
    else
        echo ""
        echo -e "\033[1;31m  Chain: OFFLINE\033[0m"
    fi
}

# ---------------------------------------------------------------------------
# CYCLE: seed -> wait -> cultivate -> wait -> harvest
# ---------------------------------------------------------------------------
do_cycle() {
    header "TRUTH FARM: FULL CYCLE"
    check_node

    info "Phase 1/3: Seeding axioms..."
    do_seed

    info "Waiting for claims to land on chain..."
    wait_blocks 3

    info "Phase 2/3: Cultivating derivatives..."
    do_cultivate

    info "Waiting for derivatives to land on chain..."
    wait_blocks 3

    info "Phase 3/3: Harvesting yields..."
    do_harvest

    echo ""
    ok "Full farm cycle complete"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
case "${1:-}" in
    seed)      do_seed ;;
    cultivate) do_cultivate ;;
    harvest)   do_harvest ;;
    cycle)     do_cycle ;;
    status)    do_status ;;
    *)
        echo "Usage: $0 {seed|cultivate|harvest|cycle|status}"
        echo ""
        echo "  seed       Seed genesis axioms as live claims"
        echo "  cultivate  Submit derivative claims citing existing facts"
        echo "  harvest    Query and report verified fact yields"
        echo "  cycle      Run one full farm cycle (seed -> cultivate -> harvest)"
        echo "  status     Show farm metrics"
        exit 1
        ;;
esac
