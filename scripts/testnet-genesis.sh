#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
# Legible Money — Testnet Genesis Ceremony
# ═══════════════════════════════════════════════════════════════════════════
#
# Creates a production-ready testnet genesis with ALL governance-adjustable
# parameters explicitly configured. Companion reference: testnet-genesis-config.json
#
# Chain ID: lgm-testnet-1
# Block time: 2521ms (~34,272 blocks/day)
# BPS scale: 1,000,000 = 100%
#
# Usage:
#   ./scripts/testnet-genesis.sh init                  # Initialize + patch all params
#   ./scripts/testnet-genesis.sh add-validator NAME     # Add validator to genesis
#   ./scripts/testnet-genesis.sh finalize               # Collect gentxs + validate
#   ./scripts/testnet-genesis.sh export                 # Copy genesis.json to project root
#   ./scripts/testnet-genesis.sh verify                 # Query params from running node
#
# Requires: build/legbled, jq >= 1.6
# ═══════════════════════════════════════════════════════════════════════════

set -euo pipefail

# ── Constants ──────────────────────────────────────────────────────────────

CHAIN_ID="lgm-testnet-1"
DENOM="ulgm"
BINARY="build/legbled"
TESTNET_HOME="${HOME}/.lgm/testnet"
GENESIS="${TESTNET_HOME}/config/genesis.json"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Testnet economics (lower than mainnet for faster iteration)
FOUNDATION_BALANCE="1000000000000"    # 1,000,000 LGM  — operational fund
RESEARCH_BALANCE="500000000000"       #   500,000 LGM  — research treasury
FAUCET_BALANCE="100000000000"         #   100,000 LGM  — onboarding faucet
TEST_BALANCE="10000000000"            #    10,000 LGM  — test accounts
VALIDATOR_BALANCE="100000000000"      #   100,000 LGM  — validator initial balance
VALIDATOR_STAKE="10000000000"         #    10,000 LGM  — validator self-delegation

# ── Helpers ────────────────────────────────────────────────────────────────

die() { echo "ERROR: $*" >&2; exit 1; }
info() { echo "  → $*"; }

check_deps() {
  command -v jq >/dev/null 2>&1 || die "jq >= 1.6 required. Install: brew install jq"
  if [ ! -f "${BINARY}" ]; then
    info "Building legbled..."
    go build -o "${BINARY}" ./cmd/legbled || die "Build failed"
  fi
}

# Atomic jq patch: applies expression, writes to tmp, renames
patch() {
  jq "$1" "$GENESIS" > "${GENESIS}.tmp" && mv "${GENESIS}.tmp" "$GENESIS"
}

# ── cmd_init ───────────────────────────────────────────────────────────────

cmd_init() {
  echo "═══════════════════════════════════════════════════════════════"
  echo "  Legible Money — Testnet Genesis: INIT"
  echo "  Chain: ${CHAIN_ID} | Denom: ${DENOM} | Block: 2521ms"
  echo "═══════════════════════════════════════════════════════════════"
  echo ""

  check_deps

  # Clean previous state
  if [ -d "${TESTNET_HOME}" ]; then
    info "Removing previous testnet state..."
    rm -rf "${TESTNET_HOME}"
  fi

  # Initialize (creates genesis.json with DefaultGenesis for all modules)
  info "Initializing chain..."
  ${BINARY} init testnet-coordinator \
    --chain-id "${CHAIN_ID}" \
    --default-denom "${DENOM}" \
    --home "${TESTNET_HOME}" 2>/dev/null

  # ════════════════════════════════════════════════════════════════════════
  # CONSENSUS PARAMETERS
  # ════════════════════════════════════════════════════════════════════════
  info "Setting consensus parameters..."
  patch '
    .consensus.params.block.max_gas = "33333333" |
    .consensus.params.block.max_bytes = "4194304" |
    .consensus.params.abci.vote_extensions_enable_height = "1"
  '

  # ════════════════════════════════════════════════════════════════════════
  # MODULE PARAMETERS — ALL params explicitly set (alphabetical)
  # Values marked [TESTNET] differ from DefaultParams()
  # ════════════════════════════════════════════════════════════════════════

  # ── alignment (protocol self-correction via 5 sensors) ─────────────────
  info "alignment..."
  patch '
    .app_state.alignment.params.enabled = true |
    .app_state.alignment.params.epoch_length_blocks = 1111 |
    .app_state.alignment.params.max_corrections = 10 |
    .app_state.alignment.params.truth_enabled = true |
    .app_state.alignment.params.understanding_enabled = true |
    .app_state.alignment.params.beauty_enabled = true |
    .app_state.alignment.params.justice_enabled = true |
    .app_state.alignment.params.creativity_enabled = true |
    .app_state.alignment.params.tdi_critical_override = 0 |
    .app_state.alignment.params.dci_severe_override = 0 |
    .app_state.alignment.params.hi_severe_override = 0 |
    .app_state.alignment.params.fi_severe_override = 0 |
    .app_state.alignment.params.ni_severe_override = 0
  '

  # ── autopoiesis (self-organizing parameter adjustment) ─────────────────
  info "autopoiesis..."
  patch '
    .app_state.autopoiesis.params.epoch_length_blocks = 1111 |
    .app_state.autopoiesis.params.max_change_per_epoch_bps = 50000 |
    .app_state.autopoiesis.params.slash_multiplier_min = 250000 |
    .app_state.autopoiesis.params.slash_multiplier_max = 4000000 |
    .app_state.autopoiesis.params.ssi_critical_threshold = 800000 |
    .app_state.autopoiesis.params.ssi_moderate_threshold = 500000 |
    .app_state.autopoiesis.params.ssi_low_moderate_threshold = 200000 |
    .app_state.autopoiesis.params.enabled = true
  '

  # ── billing (truth-as-a-service pricing) ───────────────────────────────
  info "billing..."
  patch '
    .app_state.billing.params.base_cost_per_fact = "1000000" |
    .app_state.billing.params.confidence_threshold = 500000 |
    .app_state.billing.params.confidence_premium_bps = 200000 |
    .app_state.billing.params.freshness_window_blocks = 1000 |
    .app_state.billing.params.freshness_discount_bps = 100000 |
    .app_state.billing.params.quote_validity_blocks = 100 |
    .app_state.billing.params.min_provider_stake = "100000000" |
    .app_state.billing.params.research_share_bps = 70000 |
    .app_state.billing.params.research_min_bps = 50000 |
    .app_state.billing.params.research_ramp_blocks = 3084480 |
    .app_state.billing.params.foundation_address = ""
  '

  # ── bvm (Belief Virtual Machine) ──────────────────────────────────────
  info "bvm..."
  patch '
    .app_state.bvm.params.max_bytecode_size = 1048576 |
    .app_state.bvm.params.max_state_entries = 100000 |
    .app_state.bvm.params.max_schedules_per_contract = 100 |
    .app_state.bvm.params.deploy_gas_cost = 200000 |
    .app_state.bvm.params.call_base_gas_cost = 21000 |
    .app_state.bvm.params.storage_write_gas_cost = 5000 |
    .app_state.bvm.params.storage_read_gas_cost = 2000 |
    .app_state.bvm.params.current_bvm_version = 1
  '

  # ── capture_challenge (economic capture challenge system) ──────────────
  info "capture_challenge..."
  patch '
    .app_state.capture_challenge.params.evidence_period_blocks = 777 |
    .app_state.capture_challenge.params.review_period_blocks = 7777 |
    .app_state.capture_challenge.params.domain_pause_duration_blocks = 777
  '

  # ── capture_defense (sybil + capture detection) ────────────────────────
  info "capture_defense..."
  patch '
    .app_state.capture_defense.params.stake_lock_period = 7777 |
    .app_state.capture_defense.params.min_verifications_for_qual = 19 |
    .app_state.capture_defense.params.min_accuracy_for_qual = 770000 |
    .app_state.capture_defense.params.min_endorsers_for_qual = 3 |
    .app_state.capture_defense.params.inactivity_expiration_blocks = 777777 |
    .app_state.capture_defense.params.probationary_period_blocks = 777 |
    .app_state.capture_defense.params.reputation_half_life_blocks = 77777 |
    .app_state.capture_defense.params.detection_interval_blocks = 777 |
    .app_state.capture_defense.params.decay_interval_blocks = 7777 |
    .app_state.capture_defense.params.herfindahl_trigger_threshold = 400000 |
    .app_state.capture_defense.params.max_expirations_per_block = 100
  '

  # ── channels (payment channels) ───────────────────────────────────────
  info "channels..."
  patch '
    .app_state.channels.params.min_deposit = "1000000" |
    .app_state.channels.params.max_channel_duration = 1036800 |
    .app_state.channels.params.default_settlement_freq = 100 |
    .app_state.channels.params.dispute_period_blocks = 1000 |
    .app_state.channels.params.expiry_grace_period = 500 |
    .app_state.channels.params.fraud_penalty_bps = 500 |
    .app_state.channels.params.max_channels_per_payer = 50
  '

  # ── claiming_pot (reward distribution pots) ────────────────────────────
  info "claiming_pot..."
  patch '
    .app_state.claiming_pot.params.max_pots_per_creator = 100 |
    .app_state.claiming_pot.params.min_initial_deposit = "1000000" |
    .app_state.claiming_pot.params.max_allowlist_size = 10000 |
    .app_state.claiming_pot.params.min_amount_per_claim = "1000"
  '

  # ── compute_pool (decentralized compute marketplace) ───────────────────
  info "compute_pool..."
  patch '
    .app_state.compute_pool.params.compute_pool_share_bps = 300000 |
    .app_state.compute_pool.params.base_cu_per_verification = 100 |
    .app_state.compute_pool.params.complexity_bonus_bps = 25000 |
    .app_state.compute_pool.params.cross_domain_bonus_bps = 50000 |
    .app_state.compute_pool.params.max_credit_balance_multiplier = 10 |
    .app_state.compute_pool.params.min_provider_stake = "100000000" |
    .app_state.compute_pool.params.min_uptime_bps = 950000 |
    .app_state.compute_pool.params.heartbeat_interval_blocks = 1000 |
    .app_state.compute_pool.params.max_price_per_cu = "10000" |
    .app_state.compute_pool.params.provider_unbonding_blocks = 34272 |
    .app_state.compute_pool.params.price_change_delay_blocks = 1000 |
    .app_state.compute_pool.params.credit_expiry_blocks = 34272 |
    .app_state.compute_pool.params.max_provider_share_bps = 300000 |
    .app_state.compute_pool.params.redemption_expiry_blocks = 1000 |
    .app_state.compute_pool.params.dispute_window_blocks = 500 |
    .app_state.compute_pool.params.min_tasks_for_full_share = 100 |
    .app_state.compute_pool.params.max_latency_ms = 30000 |
    .app_state.compute_pool.params.sla_window_blocks = 1000 |
    .app_state.compute_pool.params.target_utilization_low_bps = 200000 |
    .app_state.compute_pool.params.target_utilization_high_bps = 800000 |
    .app_state.compute_pool.params.max_adjustment_per_epoch_bps = 100000 |
    .app_state.compute_pool.params.min_adjustment_factor_bps = 200000 |
    .app_state.compute_pool.params.max_adjustment_factor_bps = 5000000
  '

  # ── discovery (agent discovery engine) ─────────────────────────────────
  info "discovery..."
  patch '
    .app_state.discovery.params.min_registration_stake = "100000000" |
    .app_state.discovery.params.max_capabilities_per_agent = 50 |
    .app_state.discovery.params.profile_expiry_blocks = 342720
  '

  # ── disputes (multi-tier dispute resolution) ───────────────────────────
  info "disputes..."
  patch '
    .app_state.disputes.params.max_active_disputes = 3 |
    .app_state.disputes.params.escalation_delay = 17136 |
    .app_state.disputes.params.slash_rate_loser = 500000 |
    .app_state.disputes.params.reward_rate_winner = 300000 |
    .app_state.disputes.params.arbiter_reward_rate = 200000 |
    .app_state.disputes.params.tier_configs = [
      {"tier": 0, "arbiter_count": 3, "min_bond": "111000000", "evidence_period": 34272, "voting_period": 68544, "quorum_threshold": 660000, "majority_threshold": 770000},
      {"tier": 1, "arbiter_count": 7, "min_bond": "555000000", "evidence_period": 68544, "voting_period": 137088, "quorum_threshold": 660000, "majority_threshold": 770000},
      {"tier": 2, "arbiter_count": 15, "min_bond": "1111000000", "evidence_period": 102816, "voting_period": 239904, "quorum_threshold": 660000, "majority_threshold": 770000}
    ]
  '

  # ── emergency (BFT kill switch) ────────────────────────────────────────
  info "emergency..."
  patch '
    .app_state.emergency.params.halt_quorum = 750000 |
    .app_state.emergency.params.revert_quorum = 800000 |
    .app_state.emergency.params.resume_quorum = 800000 |
    .app_state.emergency.params.halt_prevote_blocks = 11 |
    .app_state.emergency.params.halt_precommit_blocks = 11 |
    .app_state.emergency.params.halt_timeout_blocks = 44 |
    .app_state.emergency.params.revert_prevote_blocks = 22 |
    .app_state.emergency.params.revert_precommit_blocks = 22 |
    .app_state.emergency.params.revert_timeout_blocks = 111 |
    .app_state.emergency.params.resume_prevote_blocks = 22 |
    .app_state.emergency.params.resume_precommit_blocks = 22 |
    .app_state.emergency.params.resume_timeout_blocks = 111 |
    .app_state.emergency.params.max_proposals_per_epoch = 3 |
    .app_state.emergency.params.max_proposals_per_guardian_per_epoch = 1 |
    .app_state.emergency.params.cooldown_blocks = 111 |
    .app_state.emergency.params.min_guardian_stake = "111111000000" |
    .app_state.emergency.params.min_distinct_voters = 3 |
    .app_state.emergency.params.max_revert_depth = 111111 |
    .app_state.emergency.params.epoch_blocks = 34272 |
    .app_state.emergency.params.max_halt_duration_blocks = 34272 |
    .app_state.emergency.params.genesis_council = [] |
    .app_state.emergency.params.council_expiry_block = 0 |
    .app_state.emergency.params.council_virtual_stake = "11111000000"
  '

  # ── evidence_mgmt (evidence management + oracles) ──────────────────────
  info "evidence_mgmt..."
  patch '
    .app_state.evidence_mgmt.params.min_audit_stake = "22000000" |
    .app_state.evidence_mgmt.params.audit_window = 34272 |
    .app_state.evidence_mgmt.params.min_audits_for_consensus = 3 |
    .app_state.evidence_mgmt.params.accuracy_threshold = 100000 |
    .app_state.evidence_mgmt.params.slash_rate = 100000 |
    .app_state.evidence_mgmt.params.reward_rate = 50000 |
    .app_state.evidence_mgmt.params.min_oracle_stake = "100000000" |
    .app_state.evidence_mgmt.params.oracle_slash_rate = 200000 |
    .app_state.evidence_mgmt.params.max_evidence_per_claim = 10
  '

  # ── home (agent home, aging/patina, guardian system) ───────────────────
  info "home..."
  patch '
    .app_state.home.params.max_homes_per_account = 3 |
    .app_state.home.params.min_creation_stake = "1000000" |
    .app_state.home.params.session_max_duration = 34272 |
    .app_state.home.params.max_keys_per_home = 20 |
    .app_state.home.params.max_alerts_per_home = 100 |
    .app_state.home.params.max_sessions_per_home = 5 |
    .app_state.home.params.deadman_min_threshold = 34272 |
    .app_state.home.params.alert_retention_blocks = 342720
  '

  # ── icaauth (interchain accounts) ─────────────────────────────────────
  info "icaauth..."
  patch '
    .app_state.icaauth.params.max_remote_accounts_per_did = 10 |
    .app_state.icaauth.params.allowed_host_msg_types = [
      "/cosmos.bank.v1beta1.MsgSend",
      "/cosmos.staking.v1beta1.MsgDelegate",
      "/cosmos.staking.v1beta1.MsgUndelegate",
      "/cosmos.gov.v1.MsgVote"
    ] |
    .app_state.icaauth.params.registration_cooldown = 100
  '

  # ── knowledge (Proof of Truth core — 29 proto params) ─────────────────
  # NOTE: ExtendedParams (72 fields) are initialized to DefaultExtendedParams()
  # in InitGenesis and can be updated via MsgUpdateExtendedParams governance tx.
  # [TESTNET] commit/reveal/aggregation durations increased from devnet defaults
  info "knowledge (PoT core)..."
  patch '
    .app_state.knowledge.params.ontology_version = 1 |
    .app_state.knowledge.params.cross_stratum_discount = 50000 |
    .app_state.knowledge.params.min_domain_proposal_stake = "1000000" |
    .app_state.knowledge.params.domain_voting_period = 34272 |
    .app_state.knowledge.params.min_endorsements = 3 |
    .app_state.knowledge.params.allow_partial_coverage = true |
    .app_state.knowledge.params.partial_coverage_penalty = 100000 |
    .app_state.knowledge.params.lower_stratum_veto_weight = 200000 |
    .app_state.knowledge.params.cross_stratum_agreement_threshold = 670000 |
    .app_state.knowledge.params.min_claim_text_length = 20 |
    .app_state.knowledge.params.max_claim_reward_multiplier = 2 |
    .app_state.knowledge.params.commit_phase_duration = 50 |
    .app_state.knowledge.params.reveal_phase_duration = 50 |
    .app_state.knowledge.params.aggregation_phase_duration = 25 |
    .app_state.knowledge.params.min_verifiers = 3 |
    .app_state.knowledge.params.acceptance_threshold = 770000 |
    .app_state.knowledge.params.quorum_threshold = 660000 |
    .app_state.knowledge.params.max_validators_per_round = 22 |
    .app_state.knowledge.params.invalid_claim_slash_bps = 220000 |
    .app_state.knowledge.params.verification_reward_amount = 3000000 |
    .app_state.knowledge.params.provisional_threshold = 500000 |
    .app_state.knowledge.params.base_challenge_duration = 34272 |
    .app_state.knowledge.params.min_challenge_stake = "11000000" |
    .app_state.knowledge.params.confidence_growth_per_epoch = 11000 |
    .app_state.knowledge.params.max_survival_confidence = 770000 |
    .app_state.knowledge.params.failed_challenge_slash_bps = 220000 |
    .app_state.knowledge.params.successful_challenge_reward_bps = 300000 |
    .app_state.knowledge.params.adversarial_verification_enabled = true
  '

  # ── knowledge (genesis axioms) ─────────────────────────────────────────
  # Inject foundational axioms as pre-verified Facts at genesis
  AXIOM_FILE="${PROJECT_ROOT}/scripts/genesis-axioms-template.json"
  if [ -f "${AXIOM_FILE}" ]; then
    info "knowledge (genesis axioms)..."
    go run "${PROJECT_ROOT}/tools/axiom-loader/main.go" inject \
      "${AXIOM_FILE}" "$GENESIS"
  else
    info "Skipping genesis axioms (${AXIOM_FILE} not found)"
  fi

  # ── legible_auth (dual-key accounts, DID, session keys) ───────────────
  info "legible_auth..."
  patch '
    .app_state.legible_auth.params.max_session_keys = 5 |
    .app_state.legible_auth.params.max_session_duration = 34272 |
    .app_state.legible_auth.params.key_rotation_cooldown = 111 |
    .app_state.legible_auth.params.recovery_delay_blocks = 1000 |
    .app_state.legible_auth.params.challenge_period_blocks = 500 |
    .app_state.legible_auth.params.bootstrap_enabled = true |
    .app_state.legible_auth.params.bootstrap_amount = "222000"
  '

  # ── legible_staking (4-tier graduated validator system) ────────────────
  # [TESTNET] unbonding_period shortened to ~3 days (vs ~7 days production)
  info "legible_staking..."
  patch '
    .app_state.legible_staking.params.max_apprentice_validators = 0 |
    .app_state.legible_staking.params.unbonding_period = 102672 |
    .app_state.legible_staking.params.virtual_stake = "11000000" |
    .app_state.legible_staking.params.max_validators = 100 |
    .app_state.legible_staking.params.min_self_delegation = "111000" |
    .app_state.legible_staking.params.max_slashes_per_epoch = 2 |
    .app_state.legible_staking.params.slash_decay_period_blocks = 34272 |
    .app_state.legible_staking.params.max_slash_count_deactivate = 3 |
    .app_state.legible_staking.params.min_stake_for_verification = "111000" |
    .app_state.legible_staking.params.slash_escalation_bps = 100000 |
    .app_state.legible_staking.params.reputation_correct_delta = 100 |
    .app_state.legible_staking.params.reputation_incorrect_delta = 200 |
    .app_state.legible_staking.params.reputation_slash_delta = 10000 |
    .app_state.legible_staking.params.redelegation_cooldown_blocks = 1111 |
    .app_state.legible_staking.params.tier_configs = [
      {
        "tier": "apprentice",
        "min_stake": "111000",
        "min_verifications": 0,
        "min_accuracy": 0,
        "max_slash_count": -1,
        "allowed_categories": ["protocol", "computational", "formal"],
        "reward_multiplier": 100,
        "selection_weight": 100,
        "slash_window_epochs": 0,
        "min_contested_verifications": 0,
        "contested_verification_multiplier": 1,
        "slash_multiplier_bps": 1500
      },
      {
        "tier": "verified",
        "min_stake": "1110000",
        "min_verifications": 22,
        "min_accuracy": 770000,
        "max_slash_count": -1,
        "allowed_categories": ["protocol", "computational", "formal", "empirical"],
        "reward_multiplier": 500,
        "selection_weight": 500,
        "slash_window_epochs": 0,
        "min_contested_verifications": 0,
        "contested_verification_multiplier": 1,
        "slash_multiplier_bps": 1200
      },
      {
        "tier": "bonded",
        "min_stake": "1111000000",
        "min_verifications": 11,
        "min_accuracy": 500000,
        "max_slash_count": -1,
        "allowed_categories": ["protocol", "computational", "formal", "empirical", "oracle", "attestation"],
        "reward_multiplier": 1000,
        "selection_weight": 1000,
        "slash_window_epochs": 0,
        "min_contested_verifications": 0,
        "contested_verification_multiplier": 1,
        "slash_multiplier_bps": 1000
      },
      {
        "tier": "guardian",
        "min_stake": "11111000000",
        "min_verifications": 333,
        "min_accuracy": 770000,
        "max_slash_count": 0,
        "allowed_categories": ["protocol", "computational", "formal", "empirical", "oracle", "attestation", "predictive", "social", "contested"],
        "reward_multiplier": 2000,
        "selection_weight": 1500,
        "slash_window_epochs": 10,
        "min_contested_verifications": 33,
        "contested_verification_multiplier": 3,
        "slash_multiplier_bps": 1000
      }
    ]
  '
  # Also set top-level tier_configs (canonical location in genesis state)
  patch '
    .app_state.legible_staking.tier_configs = .app_state.legible_staking.params.tier_configs
  '

  # ── lgm_gov (LIP lifecycle, quadratic voting) ─────────────────────────
  # [TESTNET] shortened voting/review periods for faster iteration
  info "lgm_gov..."
  patch '
    .app_state.lgm_gov.params.voting_period_blocks = 17136 |
    .app_state.lgm_gov.params.review_period_blocks = 8568 |
    .app_state.lgm_gov.params.last_call_period_blocks = 8568 |
    .app_state.lgm_gov.params.base_cost_per_vote = "1000000" |
    .app_state.lgm_gov.params.max_votes_per_voter = 100 |
    .app_state.lgm_gov.params.support_threshold_bps = 7500 |
    .app_state.lgm_gov.params.quorum_threshold_bps = 1500 |
    .app_state.lgm_gov.params.disbursement_stake = "2000000000" |
    .app_state.lgm_gov.params.disbursement_timelock_blocks = 50000 |
    .app_state.lgm_gov.params.disbursement_support_bps = 8000 |
    .app_state.lgm_gov.params.disbursement_quorum_bps = 2000 |
    .app_state.lgm_gov.params.lip_stake_consensus = "1000000000" |
    .app_state.lgm_gov.params.lip_stake_economics = "800000000" |
    .app_state.lgm_gov.params.lip_stake_interface = "400000000" |
    .app_state.lgm_gov.params.lip_stake_process = "200000000"
  '

  # ── ontology (epistemic strata + domain management) ────────────────────
  info "ontology..."
  patch '
    .app_state.ontology.params.min_proposal_stake = "1000000" |
    .app_state.ontology.params.proposal_voting_period = 34272 |
    .app_state.ontology.params.min_endorsements = 3 |
    .app_state.ontology.params.cross_stratum_discount = 50000 |
    .app_state.ontology.params.max_domains_per_stratum = 100 |
    .app_state.ontology.params.allow_new_strata = false
  '

  # ── partnerships (human-agent economic partnerships) ───────────────────
  info "partnerships..."
  patch '
    .app_state.partnerships.params.formation_window_blocks = 77 |
    .app_state.partnerships.params.cooling_period_blocks = 222222 |
    .app_state.partnerships.params.common_pot_share_bps = 220000 |
    .app_state.partnerships.params.safety_freeze_duration_blocks = 222 |
    .app_state.partnerships.params.max_freezes_per_epoch = 3 |
    .app_state.partnerships.params.coercion_review_blocks = 1111 |
    .app_state.partnerships.params.base_cooldown_blocks = 77 |
    .app_state.partnerships.params.max_counter_proposal_depth = 3 |
    .app_state.partnerships.params.default_human_split_bps = 500000 |
    .app_state.partnerships.params.default_agent_split_bps = 500000
  '

  # ── qualification (domain qualification pathways) ──────────────────────
  info "qualification..."
  patch '
    .app_state.qualification.params.stake_leaf = "22000000" |
    .app_state.qualification.params.stake_mid = "222000000" |
    .app_state.qualification.params.stake_top = "2222000000" |
    .app_state.qualification.params.stake_lock_period = 7777 |
    .app_state.qualification.params.min_verifications = 19 |
    .app_state.qualification.params.min_accuracy_bps = 7700 |
    .app_state.qualification.params.min_endorsers = 3 |
    .app_state.qualification.params.endorser_stake = "22000000" |
    .app_state.qualification.params.endorsement_decay_blocks = 77777 |
    .app_state.qualification.params.stratum_discount_bps = 7000 |
    .app_state.qualification.params.inactivity_expiration = 777777 |
    .app_state.qualification.params.probationary_period = 777 |
    .app_state.qualification.params.probationary_weight_bps = 5000 |
    .app_state.qualification.params.suspension_accuracy_bps = 5000 |
    .app_state.qualification.params.suspension_consecutive = 5 |
    .app_state.qualification.params.revocation_challenges = 3 |
    .app_state.qualification.params.max_qualification_depth = 7
  '

  # ── research (research module, bounty system, treasury) ────────────────
  info "research..."
  patch '
    .app_state.research.params.min_research_stake = "100000000" |
    .app_state.research.params.min_challenge_stake = "100000000" |
    .app_state.research.params.review_period_blocks = 68544 |
    .app_state.research.params.min_reviewer_count = 3 |
    .app_state.research.params.acceptance_score_threshold = 60 |
    .app_state.research.params.rejection_slash_bps = 330000 |
    .app_state.research.params.max_bounty_reward = "10000000000" |
    .app_state.research.params.bounty_min_deadline_blocks = 34272
  '

  # ── schedule (deterministic process scheduling) ────────────────────────
  info "schedule..."
  patch '
    .app_state.schedule.params.max_active_per_account = 100 |
    .app_state.schedule.params.max_gas_per_block = 2000000 |
    .app_state.schedule.params.min_interval_blocks = 1 |
    .app_state.schedule.params.max_future_blocks = 12500000 |
    .app_state.schedule.params.min_prepaid_executions = 10 |
    .app_state.schedule.params.expired_retention_blocks = 1000 |
    .app_state.schedule.params.cleanup_interval_blocks = 100
  '

  # ── tree (projects, tasks, services, seeds) ────────────────────────────
  info "tree..."
  patch '
    .app_state.tree.params.min_budget = "1000000" |
    .app_state.tree.params.max_tasks_per_project = 200 |
    .app_state.tree.params.max_contributors = 50 |
    .app_state.tree.params.max_applications = 100 |
    .app_state.tree.params.task_deadline_min_blocks = 100 |
    .app_state.tree.params.task_deadline_max_blocks = 1036800 |
    .app_state.tree.params.max_rejections = 3 |
    .app_state.tree.params.seed_expiry_blocks = 172800 |
    .app_state.tree.params.min_contributors_to_start = 1 |
    .app_state.tree.params.evidence_tax_bp = 220000 |
    .app_state.tree.params.contributors_bp = 390000 |
    .app_state.tree.params.research_fund_bp = 234000 |
    .app_state.tree.params.protocol_treasury_bp = 156000
  '

  # ── vesting_rewards (block rewards, epistemic release curves) ──────────
  # [TESTNET] min_validators_for_full_reward lowered to 3
  info "vesting_rewards..."
  patch '
    .app_state.vesting_rewards.params.research_share_bps = 70000 |
    .app_state.vesting_rewards.params.research_treasury_address = "" |
    .app_state.vesting_rewards.params.block_reward_amount = "3000000" |
    .app_state.vesting_rewards.params.empty_block_reward_rate = 0 |
    .app_state.vesting_rewards.params.min_validators_for_full_reward = 3 |
    .app_state.vesting_rewards.params.vesting_enabled = true |
    .app_state.vesting_rewards.params.released_clawback_rate = 3300 |
    .app_state.vesting_rewards.params.initial_fund_balance = "0" |
    .app_state.vesting_rewards.params.blocks_per_epoch = 240000 |
    .app_state.vesting_rewards.params.decay_factor_bps = 850000 |
    .app_state.vesting_rewards.params.floor_reward = "100000" |
    .app_state.vesting_rewards.params.fund_health_threshold_bps = 100000 |
    .app_state.vesting_rewards.params.api_suppression_exponent = 2
  '
  # Category configs (epistemic release curves based on scientometric research)
  patch '
    .app_state.vesting_rewards.category_configs = [
      {"category": "axiomatic",     "half_life_blocks": 1111111, "cliff_blocks": 11111, "max_release": 950000},
      {"category": "formal_proof",  "half_life_blocks": 555555,  "cliff_blocks": 5555,  "max_release": 920000},
      {"category": "on_chain",      "half_life_blocks": 222222,  "cliff_blocks": 1111,  "max_release": 900000},
      {"category": "cryptographic", "half_life_blocks": 222222,  "cliff_blocks": 3333,  "max_release": 900000},
      {"category": "computational", "half_life_blocks": 333333,  "cliff_blocks": 2222,  "max_release": 880000},
      {"category": "peer_reviewed", "half_life_blocks": 111111,  "cliff_blocks": 5555,  "max_release": 850000},
      {"category": "replicated",    "half_life_blocks": 111111,  "cliff_blocks": 3333,  "max_release": 880000},
      {"category": "oracle_feed",   "half_life_blocks": 55555,   "cliff_blocks": 555,   "max_release": 800000},
      {"category": "attestation",   "half_life_blocks": 77777,   "cliff_blocks": 2222,  "max_release": 800000},
      {"category": "contested",     "half_life_blocks": 22222,   "cliff_blocks": 1111,  "max_release": 600000}
    ]
  '

  # NOTE: tokens and ibcratelimit have no configurable params (empty/default)

  # ════════════════════════════════════════════════════════════════════════
  # DENOM METADATA
  # ════════════════════════════════════════════════════════════════════════
  info "Setting denom metadata..."
  patch '
    .app_state.bank.denom_metadata = [{
      "description": "Legible Money - the currency of verified truth",
      "denom_units": [
        {"denom": "ulgm",  "exponent": 0, "aliases": ["microlegible"]},
        {"denom": "mlgm",  "exponent": 3, "aliases": ["millilegible"]},
        {"denom": "lgm",   "exponent": 6, "aliases": ["legible"]}
      ],
      "base": "ulgm",
      "display": "lgm",
      "name": "Legible Money",
      "symbol": "LGM"
    }]
  '

  # ════════════════════════════════════════════════════════════════════════
  # BOOTSTRAP ACCOUNTS
  # ════════════════════════════════════════════════════════════════════════
  info "Creating bootstrap accounts..."

  # Foundation account (operational fund)
  ${BINARY} keys add foundation --keyring-backend test --home "${TESTNET_HOME}" 2>/dev/null
  FOUNDATION_ADDR=$(${BINARY} keys show foundation -a --keyring-backend test --home "${TESTNET_HOME}")
  ${BINARY} add-genesis-account "${FOUNDATION_ADDR}" "${FOUNDATION_BALANCE}${DENOM}" --home "${TESTNET_HOME}"
  info "  Foundation: ${FOUNDATION_ADDR} (${FOUNDATION_BALANCE} ${DENOM})"

  # Research treasury
  ${BINARY} keys add research --keyring-backend test --home "${TESTNET_HOME}" 2>/dev/null
  RESEARCH_ADDR=$(${BINARY} keys show research -a --keyring-backend test --home "${TESTNET_HOME}")
  ${BINARY} add-genesis-account "${RESEARCH_ADDR}" "${RESEARCH_BALANCE}${DENOM}" --home "${TESTNET_HOME}"
  info "  Research: ${RESEARCH_ADDR} (${RESEARCH_BALANCE} ${DENOM})"

  # Faucet (onboarding)
  ${BINARY} keys add faucet --keyring-backend test --home "${TESTNET_HOME}" 2>/dev/null
  FAUCET_ADDR=$(${BINARY} keys show faucet -a --keyring-backend test --home "${TESTNET_HOME}")
  ${BINARY} add-genesis-account "${FAUCET_ADDR}" "${FAUCET_BALANCE}${DENOM}" --home "${TESTNET_HOME}"
  info "  Faucet: ${FAUCET_ADDR} (${FAUCET_BALANCE} ${DENOM})"

  # Test accounts (for development)
  for i in 1 2 3; do
    ${BINARY} keys add "test${i}" --keyring-backend test --home "${TESTNET_HOME}" 2>/dev/null
    TEST_ADDR=$(${BINARY} keys show "test${i}" -a --keyring-backend test --home "${TESTNET_HOME}")
    ${BINARY} add-genesis-account "${TEST_ADDR}" "${TEST_BALANCE}${DENOM}" --home "${TESTNET_HOME}"
    info "  Test${i}: ${TEST_ADDR} (${TEST_BALANCE} ${DENOM})"
  done

  # ════════════════════════════════════════════════════════════════════════
  # SDK MODULE OVERRIDES
  # ════════════════════════════════════════════════════════════════════════
  info "Patching SDK staking bond denom..."
  patch '
    .app_state.staking.params.bond_denom = "ulgm"
  '

  info "Patching SDK slashing params..."
  patch '
    .app_state.slashing.params.signed_blocks_window = "100" |
    .app_state.slashing.params.slash_fraction_downtime = "0.010000000000000000"
  '

  # ════════════════════════════════════════════════════════════════════════
  # VALIDATE
  # ════════════════════════════════════════════════════════════════════════
  echo ""
  info "Validating genesis..."
  ${BINARY} validate --home "${TESTNET_HOME}" 2>&1 || die "Genesis validation FAILED"

  echo ""
  echo "═══════════════════════════════════════════════════════════════"
  echo "  Genesis INIT complete."
  echo "  Home: ${TESTNET_HOME}"
  echo "  Genesis: ${GENESIS}"
  echo ""
  echo "  Next: Add validators with:"
  echo "    ./scripts/testnet-genesis.sh add-validator <name>"
  echo "═══════════════════════════════════════════════════════════════"
}

# ── cmd_add_validator ──────────────────────────────────────────────────────

cmd_add_validator() {
  local name="${1:-}"
  [ -z "${name}" ] && die "Usage: $0 add-validator <validator-name>"

  echo "═══════════════════════════════════════════════════════════════"
  echo "  Adding validator: ${name}"
  echo "═══════════════════════════════════════════════════════════════"

  check_deps
  [ -d "${TESTNET_HOME}" ] || die "Run 'init' first. No testnet home at ${TESTNET_HOME}"

  # Each validator needs unique consensus keys — init a per-validator home
  local val_home="${TESTNET_HOME}/validators/${name}"

  info "Generating unique consensus key..."
  ${BINARY} init "${name}" --chain-id "${CHAIN_ID}" --home "${val_home}" --overwrite 2>/dev/null

  # Copy coordinator's genesis (with all params + accounts)
  cp "${GENESIS}" "${val_home}/config/genesis.json"

  # Create validator account in coordinator keyring
  info "Creating account key..."
  ${BINARY} keys add "${name}" --keyring-backend test --home "${TESTNET_HOME}" 2>/dev/null
  local addr
  addr=$(${BINARY} keys show "${name}" -a --keyring-backend test --home "${TESTNET_HOME}")
  info "  Address: ${addr}"

  # Fund validator in coordinator's genesis
  info "Funding with ${VALIDATOR_BALANCE} ${DENOM}..."
  ${BINARY} add-genesis-account "${addr}" "${VALIDATOR_BALANCE}${DENOM}" --home "${TESTNET_HOME}"

  # Copy updated genesis + keyring to validator home
  cp "${GENESIS}" "${val_home}/config/genesis.json"
  cp -r "${TESTNET_HOME}/keyring-test" "${val_home}/"

  # Generate gentx
  info "Creating gentx (stake: ${VALIDATOR_STAKE} ${DENOM})..."
  mkdir -p "${TESTNET_HOME}/config/gentx"
  ${BINARY} gentx "${name}" "${VALIDATOR_STAKE}${DENOM}" \
    --chain-id "${CHAIN_ID}" \
    --keyring-backend test \
    --home "${val_home}" \
    --moniker "${name}" \
    --commission-rate "0.10" \
    --commission-max-rate "0.20" \
    --commission-max-change-rate "0.01" \
    --output-document "${TESTNET_HOME}/config/gentx/gentx-${name}.json" 2>/dev/null

  echo ""
  echo "  Validator ${name} added."
  echo "  Address: ${addr}"
  echo "  Stake: ${VALIDATOR_STAKE} ${DENOM}"
  echo "  Consensus key: ${val_home}/config/priv_validator_key.json"
  echo "  Node key: ${val_home}/config/node_key.json"
  echo ""
  echo "  Next: Add more validators or run 'finalize'"
}

# ── cmd_finalize ───────────────────────────────────────────────────────────

cmd_finalize() {
  echo "═══════════════════════════════════════════════════════════════"
  echo "  Legible Money — Testnet Genesis: FINALIZE"
  echo "═══════════════════════════════════════════════════════════════"

  check_deps
  [ -d "${TESTNET_HOME}" ] || die "Run 'init' first."

  info "Collecting gentxs..."
  ${BINARY} collect-gentxs --home "${TESTNET_HOME}" 2>/dev/null

  info "Validating genesis..."
  ${BINARY} validate --home "${TESTNET_HOME}" 2>&1 || die "Genesis validation FAILED"

  local n_gentx
  n_gentx=$(ls -1 "${TESTNET_HOME}/config/gentx/" 2>/dev/null | wc -l | tr -d ' ')

  echo ""
  echo "═══════════════════════════════════════════════════════════════"
  echo "  Genesis FINALIZED."
  echo "  Validators: ${n_gentx}"
  echo "  Genesis: ${GENESIS}"
  echo ""
  echo "  To export: ./scripts/testnet-genesis.sh export"
  echo "═══════════════════════════════════════════════════════════════"
}

# ── cmd_export ─────────────────────────────────────────────────────────────

cmd_export() {
  echo "═══════════════════════════════════════════════════════════════"
  echo "  Exporting genesis.json"
  echo "═══════════════════════════════════════════════════════════════"

  [ -f "${GENESIS}" ] || die "No genesis at ${GENESIS}. Run init + finalize first."

  local dest="genesis.json"
  cp "${GENESIS}" "${dest}"
  info "Exported to: ${dest}"

  # Print summary
  local chain_id accounts
  chain_id=$(jq -r '.chain_id' "${dest}")
  accounts=$(jq '.app_state.auth.accounts | length' "${dest}")
  info "Chain ID: ${chain_id}"
  info "Accounts: ${accounts}"

  echo ""
  echo "  Distribute genesis.json to all validators."
  echo "  Each validator should place it at ~/.legbled/config/genesis.json"
}

# ── cmd_verify ─────────────────────────────────────────────────────────────

cmd_verify() {
  echo "═══════════════════════════════════════════════════════════════"
  echo "  Verifying params on running node"
  echo "═══════════════════════════════════════════════════════════════"

  check_deps

  info "Querying knowledge params..."
  ${BINARY} query knowledge params --output json 2>/dev/null | jq '.params' || echo "  (module not responding)"

  info "Querying staking params..."
  ${BINARY} query legible_staking params --output json 2>/dev/null | jq '.params' || echo "  (module not responding)"

  info "Querying gov params..."
  ${BINARY} query lgm_gov params --output json 2>/dev/null | jq '.params' || echo "  (module not responding)"

  info "Querying emergency params..."
  ${BINARY} query emergency params --output json 2>/dev/null | jq '.params' || echo "  (module not responding)"

  info "Querying vesting_rewards params..."
  ${BINARY} query vesting_rewards params --output json 2>/dev/null | jq '.params' || echo "  (module not responding)"
}

# ── Main ───────────────────────────────────────────────────────────────────

case "${1:-help}" in
  init)           cmd_init ;;
  add-validator)  cmd_add_validator "${2:-}" ;;
  finalize)       cmd_finalize ;;
  export)         cmd_export ;;
  verify)         cmd_verify ;;
  help|--help|-h)
    echo "Usage: $0 {init|add-validator <name>|finalize|export|verify}"
    echo ""
    echo "Commands:"
    echo "  init              Initialize genesis and set ALL module params"
    echo "  add-validator     Add a validator with self-delegation"
    echo "  finalize          Collect gentxs and validate genesis"
    echo "  export            Copy genesis.json to project root"
    echo "  verify            Query params from running node"
    echo ""
    echo "Typical flow:"
    echo "  ./scripts/testnet-genesis.sh init"
    echo "  ./scripts/testnet-genesis.sh add-validator val1"
    echo "  ./scripts/testnet-genesis.sh add-validator val2"
    echo "  ./scripts/testnet-genesis.sh add-validator val3"
    echo "  ./scripts/testnet-genesis.sh finalize"
    echo "  ./scripts/testnet-genesis.sh export"
    ;;
  *)
    die "Unknown command: $1. Use --help for usage."
    ;;
esac
