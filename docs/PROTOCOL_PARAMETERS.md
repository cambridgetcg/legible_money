# Protocol Parameters

> Every number that governs the Legible Money protocol. What it is, what it does, and how you can change it.

---

## Why This Page Exists

Most protocols bury their parameters in source code. You'd need to read Go, understand Cosmos SDK patterns, and dig through 22 modules to find what you're looking for.

We think that's wrong. If a parameter affects your rewards, your slashing risk, your governance power, or your ability to participate — you should be able to see it, understand it, and vote to change it.

This page shows every configurable value in the protocol. Nothing is hidden.

---

## The Three Layers

Legible Money parameters exist in three layers, inspired by biological systems:

### Layer 1: DNA (Immutable)

Constants that define what the protocol **is**. These never change — not by governance, not by anyone. They are hardcoded in Go source.

| Constant | Value | Why It's Immutable |
|----------|-------|--------------------|
| Block time | **2521ms** | Defines the heartbeat of the chain |
| Finality depth | **19 blocks** | Security assumption for confirmation |
| Token denomination | **ulgm** (micro-LGM) | 1 LGM = 1,000,000 ulgm |
| DID format | `did:lgm:{32-hex}` | Identity schema is permanent |
| Bech32 prefix | `lgm` / `lgmvaloper` | Address format identity |
| Cryptographic algorithms | Ed25519 + SHA-512 | Cannot swap post-launch |
| Domain separation | `LGM.purpose.v1` | Prevents cross-context replay |

**~15 constants. These are the protocol's identity.**

### Layer 2: Genome (Governance-Mutable)

Parameters set at genesis, changeable by governance proposals. These define *how* the protocol behaves right now. Every parameter listed in the module tables below is Layer 2.

**~300+ parameters across 22 modules.**

### Layer 3: Phenotype (Auto-Adaptive)

Multiplier overlays computed by the autopoiesis engine (when activated). Applied on top of Layer 2 at runtime. The engine adjusts these each epoch based on observed metrics, within safety bounds.

```
effective_value = base_value (Layer 2) × multiplier (Layer 3)
```

**~20 multiplier paths. Bounded ±5%/epoch. See [Autopoiesis](#the-autopoiesis-engine) below.**

---

## Module Parameters

### System: Gas & Fees

> Source: `app/gas.go`

#### Gas Limits

| Parameter | Value | Unit | Description |
|-----------|-------|------|-------------|
| `BlockGasLimit` | 33,333,333 | gas | Maximum gas per block |
| `TxGasLimit` | 11,111,111 | gas | Maximum gas per transaction |
| `SelfInvocationLimit` | 5,555,555 | gas | BVM self-invocation gas cap |
| `MinGasLimit` | 22,222 | gas | Floor for unknown tx types |

#### Gas Prices

| Parameter | Value | Unit | Description |
|-----------|-------|------|-------------|
| `MinGasPrice` | 1 | ulgm/gas | Absolute minimum gas price |
| `BaseGasPrice` | 1,111,111 | ulgm/gas | Standard gas price |
| `PriorityMultiple` | 2 | multiplier | Priority tx gas multiplier |

#### Fee Routing

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `ResearchTaxBPS` | 70,000 | 7% | Portion of fees to research fund |
| `ValidatorFeeBPS` | 930,000 | 93% | Portion of fees to validators |
| `BurnRateBPS` | 50,000 | 5% | Burn rate at penalty sites |

#### Bootstrap Period

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `BootstrapEndBlock` | 480,000 | ~14 days | Gas-free period for essential PoT |

**Gas-free transaction types during bootstrap:**
- `MsgSubmitClaim` — submit knowledge claims
- `MsgSubmitCommitment` — commit verification votes
- `MsgSubmitReveal` — reveal verification votes
- `MsgRegisterValidator` — join as validator
- `MsgRegisterAccount` — create identity

#### Transaction Gas Costs

<details>
<summary>All 126 transaction gas costs (click to expand)</summary>

##### Core

| Transaction | Gas Cost |
|-------------|----------|
| `transfer` | 21,000 |
| `stake` | 50,000 |
| `unstake` | 50,000 |

##### Knowledge & Consensus

| Transaction | Gas Cost |
|-------------|----------|
| `claim_submit` | 100,000 |
| `verification_commit` | 30,000 |
| `verification_reveal` | 40,000 |
| `challenge_fact` | 80,000 |
| `add_fact` | 120,000 |
| `challenge_claim` | 80,000 |
| `revoke_fact` | 60,000 |
| `patronize_fact` | 50,000 |

##### Staking & Delegation

| Transaction | Gas Cost |
|-------------|----------|
| `delegate` | 60,000 |
| `undelegate` | 60,000 |
| `register_validator` | 100,000 |
| `redelegate` | 80,000 |

##### Agent Identity

| Transaction | Gas Cost |
|-------------|----------|
| `register_account` | 50,000 |
| `rotate_key` | 30,000 |
| `create_session` | 40,000 |
| `revoke_session` | 25,000 |
| `recover_account` | 80,000 |
| `freeze_account` | 30,000 |
| `unfreeze_account` | 30,000 |
| `set_recovery_config` | 60,000 |
| `initiate_recovery` | 80,000 |
| `submit_recovery_shard` | 40,000 |
| `challenge_recovery` | 50,000 |
| `execute_recovery` | 80,000 |

##### Ontology

| Transaction | Gas Cost |
|-------------|----------|
| `propose_domain` | 80,000 |
| `endorse_domain` | 30,000 |
| `challenge_domain` | 50,000 |
| `register_stratum` | 60,000 |
| `vote_domain_proposal` | 30,000 |
| `update_domain` | 40,000 |
| `register_logic_zone` | 60,000 |
| `acknowledge_incompleteness` | 30,000 |

##### Governance

| Transaction | Gas Cost |
|-------------|----------|
| `governance_propose` | 100,000 |
| `governance_vote` | 30,000 |
| `governance_stake_lip` | 50,000 |
| `submit_lip` | 100,000 |
| `cast_vote` | 30,000 |
| `lock_vote` | 40,000 |
| `unlock_vote` | 30,000 |
| `switch_vote` | 35,000 |
| `commit_review` | 40,000 |
| `reveal_review` | 50,000 |
| `finalize_review` | 60,000 |
| `submit_disbursement` | 80,000 |
| `execute_disbursement` | 100,000 |
| `amend_lip` | 60,000 |
| `advance_lip_stage` | 40,000 |
| `withdraw_lip` | 30,000 |
| `stake_disbursement` | 50,000 |
| `advance_disbursement` | 40,000 |
| `cancel_disbursement` | 30,000 |
| `register_operator` | 80,000 |
| `add_agent` | 30,000 |
| `remove_agent` | 30,000 |
| `slash_operator` | 80,000 |
| `create_deployment` | 80,000 |
| `advance_deployment` | 40,000 |
| `approve_deployment` | 40,000 |
| `rollback_deployment` | 60,000 |

##### Disputes

| Transaction | Gas Cost |
|-------------|----------|
| `initiate_dispute` | 100,000 |
| `commit_evidence` | 40,000 |
| `reveal_evidence` | 50,000 |
| `arbiter_vote` | 30,000 |
| `vote_dispute` | 30,000 |
| `escalate_dispute` | 60,000 |
| `settle_dispute` | 80,000 |
| `timeout_dispute` | 40,000 |

##### Research

| Transaction | Gas Cost |
|-------------|----------|
| `submit_research` | 80,000 |
| `challenge_research` | 60,000 |
| `review_research` | 40,000 |
| `resolve_research` | 60,000 |
| `create_bounty` | 80,000 |
| `claim_bounty` | 40,000 |
| `fulfill_bounty` | 60,000 |
| `fund_research` | 50,000 |

##### Evidence Management

| Transaction | Gas Cost |
|-------------|----------|
| `submit_evidence` | 60,000 |
| `archive_evidence` | 40,000 |
| `audit_submit` | 50,000 |
| `audit_resolve` | 60,000 |
| `register_oracle` | 80,000 |

##### Vesting & Rewards

| Transaction | Gas Cost |
|-------------|----------|
| `create_vesting` | 80,000 |
| `claim_vesting` | 40,000 |
| `pause_vesting` | 25,000 |
| `resume_vesting` | 25,000 |
| `accelerate_vesting` | 50,000 |
| `falsify_vesting` | 60,000 |
| `complete_vesting` | 40,000 |

##### Secondary Tokens (LGM-20)

| Transaction | Gas Cost |
|-------------|----------|
| `token_create` | 150,000 |
| `token_mint` | 50,000 |
| `token_burn` | 30,000 |
| `token_transfer_secondary` | 25,000 |
| `token_approve` | 25,000 |
| `token_transfer_from` | 30,000 |
| `token_pause` | 25,000 |
| `token_unpause` | 25,000 |
| `delegate_power` | 40,000 |
| `wrap_token` | 50,000 |
| `unwrap_token` | 50,000 |
| `lp_add_liquidity` | 60,000 |
| `lp_remove_liquidity` | 60,000 |
| `lp_swap` | 40,000 |

##### Payment Channels

| Transaction | Gas Cost |
|-------------|----------|
| `open_channel` | 80,000 |
| `update_channel` | 30,000 |
| `close_channel` | 50,000 |
| `settle_channel` | 60,000 |
| `dispute_channel` | 80,000 |

##### Tree of Life

| Transaction | Gas Cost |
|-------------|----------|
| `create_project` | 80,000 |
| `propose_project` | 60,000 |
| `start_development` | 40,000 |
| `complete_project` | 60,000 |
| `pause_project` | 25,000 |
| `resume_project` | 25,000 |
| `abandon_project` | 30,000 |
| `spawn_child_project` | 80,000 |
| `add_task` | 40,000 |
| `assign_task` | 30,000 |
| `start_work` | 25,000 |
| `submit_deliverable` | 50,000 |
| `approve_deliverable` | 30,000 |
| `reject_deliverable` | 30,000 |
| `reopen_task` | 25,000 |
| `deploy_service` | 100,000 |
| `call_service` | 40,000 |
| `subscribe_service` | 50,000 |
| `pause_service` | 25,000 |
| `resume_service` | 25,000 |
| `retire_service` | 30,000 |
| `begin_seeding` | 60,000 |
| `detect_opportunity` | 40,000 |
| `claim_opportunity` | 50,000 |
| `add_contributor` | 30,000 |
| `apply_to_project` | 40,000 |
| `review_application` | 30,000 |
| `set_availability` | 25,000 |

##### BVM Contracts

| Transaction | Gas Cost |
|-------------|----------|
| `deploy_contract` | 200,000 |
| `call_contract` | 21,000 |
| `schedule_execution` | 40,000 |

##### Emergency

| Transaction | Gas Cost |
|-------------|----------|
| `propose_halt` | 50,000 |
| `vote_halt` | 25,000 |
| `propose_revert` | 50,000 |
| `vote_revert` | 25,000 |
| `propose_resume` | 50,000 |
| `vote_resume` | 25,000 |

##### Claiming Pots

| Transaction | Gas Cost |
|-------------|----------|
| `create_pot` | 80,000 |
| `fund_pot` | 40,000 |
| `claim_from_pot` | 30,000 |
| `close_pot` | 40,000 |

##### Agent Home

| Transaction | Gas Cost |
|-------------|----------|
| `create_home` | 150,000 |
| `update_home` | 40,000 |
| `update_memory_cid` | 30,000 |
| `home_start_session` | 30,000 |
| `home_end_session` | 20,000 |
| `home_register_key` | 50,000 |
| `home_revoke_key` | 25,000 |
| `configure_guardian` | 50,000 |
| `acknowledge_alert` | 20,000 |
| `set_spending_limit` | 40,000 |

##### Domain Qualification

| Transaction | Gas Cost |
|-------------|----------|
| `request_qualification` | 80,000 |
| `endorse_validator` | 50,000 |
| `renew_qualification` | 30,000 |
| `withdraw_qualification` | 50,000 |

##### Capture Defense & Challenge

| Transaction | Gas Cost |
|-------------|----------|
| `request_capture_qualification` | 80,000 |
| `endorse_capture_qualification` | 40,000 |
| `submit_capture_challenge` | 100,000 |
| `add_challenge_evidence` | 50,000 |
| `resolve_capture_challenge` | 80,000 |

##### Discovery

| Transaction | Gas Cost |
|-------------|----------|
| `register_agent` | 60,000 |
| `update_profile` | 30,000 |
| `add_capability` | 25,000 |
| `remove_capability` | 20,000 |
| `deregister_agent` | 30,000 |
| `heartbeat` | 15,000 |

##### Partnerships

| Transaction | Gas Cost |
|-------------|----------|
| `initiate_partnership` | 100,000 |
| `accept_partnership` | 40,000 |
| `deposit_to_pot` | 50,000 |
| `distribute_reward` | 60,000 |
| `propose_operation` | 80,000 |
| `approve_operation` | 30,000 |
| `reject_operation` | 30,000 |
| `safety_freeze` | 80,000 |
| `raise_coercion_signal` | 60,000 |
| `initiate_exit` | 50,000 |

##### Schedule

| Transaction | Gas Cost |
|-------------|----------|
| `create_schedule` | 80,000 |
| `pause_schedule` | 20,000 |
| `resume_schedule` | 20,000 |
| `cancel_schedule` | 30,000 |
| `add_fee` | 25,000 |

##### Billing

| Transaction | Gas Cost |
|-------------|----------|
| `register_provider` | 80,000 |
| `deregister_provider` | 30,000 |
| `request_quote` | 20,000 |
| `execute_payment` | 40,000 |

</details>

---

### x/knowledge — Proof of Truth Consensus

> Source: `x/knowledge/types/genesis.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `OntologyVersion` | 1 | v1 | Ontology schema version |
| `CrossStratumDiscount` | 50,000 | 5% | Discount for cross-stratum references |
| `MinDomainProposalStake` | "1000000" | 1 LGM | Minimum stake to propose a domain |
| `DomainVotingPeriod` | 34,272 | ~1 day | Blocks for domain proposal voting |
| `MinEndorsements` | 3 | 3 votes | Minimum endorsements for domain approval |
| `AllowPartialCoverage` | true | yes | Allow claims that partially span domains |
| `PartialCoveragePenalty` | 100,000 | 10% | Confidence penalty for partial coverage |
| `LowerStratumVetoWeight` | 200,000 | 20% | Weight of lower-stratum veto votes |
| `CrossStratumAgreementThreshold` | 670,000 | 67% | Agreement needed for cross-stratum claims |
| `MinClaimTextLength` | 20 | 20 chars | Minimum claim text (anti-tautology) |
| `MaxClaimRewardMultiplier` | 2 | 2x | Maximum reward cap relative to stake |
| `CommitPhaseDuration` | 4 | ~10s | Blocks for commit phase |
| `RevealPhaseDuration` | 4 | ~10s | Blocks for reveal phase |
| `AggregationPhaseDuration` | 3 | ~7.5s | Blocks for aggregation phase |
| `MinVerifiers` | 3 | 3 validators | Minimum validators per round |
| `AcceptanceThreshold` | 770,000 | 77% | Agreement needed to accept a claim |
| `QuorumThreshold` | 660,000 | 66% | Participation needed for valid round |
| `MaxValidatorsPerRound` | 22 | 22 | Maximum validators selected per round |
| `InvalidClaimSlashBps` | 220,000 | 22% | Slash on rejected claims |
| `VerificationRewardAmount` | "3000000" | 3 LGM | Base reward per correct verification |

**Bootstrap domains:** mathematics, physics, computer_science, general

---

### x/staking — Validator Economics

> Source: `x/staking/types/genesis.go`, `x/staking/types/types.go`

#### Core Parameters

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `UnbondingPeriod` | 268,560 | ~7 days | Blocks until unstaked tokens are liquid |
| `VirtualStake` | "11000000" | 11 LGM | Virtual stake for Tier 0/1 VRF selection |
| `MaxValidators` | 100 | 100 | Max block producers (Tier 2+ only) |
| `MinSelfDelegation` | "111000" | 0.111 LGM | Sybil defense minimum |
| `MaxSlashesPerEpoch` | 2 | 2 | Max slashes per validator per epoch |
| `SlashDecayPeriodBlocks` | 34,272 | ~1 day | Decay 1 slash per epoch |
| `MaxSlashCountDeactivate` | 3 | 3 | Slashes to deactivate validator |
| `MinStakeForVerification` | "111000" | 0.111 LGM | Min real stake for VRF eligibility |
| `SlashEscalationBps` | 100,000 | 10% | Escalation per prior slash |

#### Tier Configurations

The 4-tier graduated validator system. Each tier has specific requirements and rewards:

##### Tier 0: Apprentice

| Field | Value | Description |
|-------|-------|-------------|
| MinStake | 0.111 LGM | Near-zero barrier to entry |
| MinVerifications | 0 | No history required |
| MinAccuracy | 0% | No accuracy required |
| MaxSlashCount | -1 (unlimited) | No slash limit |
| AllowedCategories | protocol, computational, formal | Restricted to formally verifiable |
| RewardMultiplier | 0.1x | Learning rate |
| SelectionWeight | 0.1x | Low VRF weight |

##### Tier 1: Verified

| Field | Value | Description |
|-------|-------|-------------|
| MinStake | 1.11 LGM | Small economic commitment |
| MinVerifications | 22 | Proven participation |
| MinAccuracy | 77% | Demonstrated competence |
| MaxSlashCount | -1 (unlimited) | No slash limit |
| AllowedCategories | + empirical | Unlocks empirical claims |
| RewardMultiplier | 0.5x | Half rewards |
| SelectionWeight | 0.5x | Moderate VRF weight |

##### Tier 2: Bonded

| Field | Value | Description |
|-------|-------|-------------|
| MinStake | 1,111 LGM | Significant economic commitment |
| MinVerifications | 11 | Maintained participation |
| MinAccuracy | 50% | Basic oracle access threshold |
| MaxSlashCount | -1 (unlimited) | No slash limit |
| AllowedCategories | + oracle, attestation | Full category access |
| RewardMultiplier | 1.0x | Full rewards |
| SelectionWeight | 1.0x | Full VRF weight |

##### Tier 3: Guardian

| Field | Value | Description |
|-------|-------|-------------|
| MinStake | 11,111 LGM | Maximum economic commitment |
| MinVerifications | 333 | Extensive history |
| MinAccuracy | 77% | High sustained accuracy |
| MaxSlashCount | 0 in window | Zero tolerance for errors |
| AllowedCategories | all 9 categories | Including predictive, social, contested |
| RewardMultiplier | 2.0x | Double rewards (risk compensation) |
| SelectionWeight | 1.5x | Highest VRF weight |
| SlashWindowEpochs | 10 | Only recent slashes count |
| MinContestedVerifications | 33 | Must engage hard claims |
| ContestedVerificationMultiplier | 3x | Contested claims count triple |

---

### x/auth — Identity & Session Keys

> Source: `x/auth/types/genesis.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `MaxSessionKeys` | 5 | 5 keys | Max active session keys per account |
| `MaxSessionDuration` | 34,272 | ~1 day | Max session key lifetime in blocks |
| `KeyRotationCooldown` | 111 | ~4.6 min | Blocks between key rotations |
| `RecoveryDelayBlocks` | 1,000 | ~42 min | Delay before recovery executes |
| `ChallengePeriodBlocks` | 500 | ~21 min | Window to challenge recovery |
| `BootstrapEnabled` | true | yes | Bootstrap fund active |
| `BootstrapAmount` | "222000000000000000" | 0.222 LGM | Bootstrap fund per new account |

---

### x/vesting_rewards — Block Rewards & Emission

> Source: `x/vesting_rewards/types/genesis.go`

#### Reward Parameters

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `ResearchShareBps` | 70,000 | 7% | Block reward share to research fund |
| `ResearchTreasuryAddress` | "" | (genesis) | Treasury address for disbursements |
| `BlockRewardAmount` | "3000000" | 3 LGM | Base block reward (scaled by participation) |
| `EmptyBlockRewardRate` | 0 | 0% | Reward for empty blocks (PoT: no free lunch) |
| `MinValidatorsForFullReward` | 22 | 22 | Target validator count for full rewards |
| `VestingEnabled` | true | yes | Vesting release curves active |
| `ReleasedClawbackRate` | 3,300 | 33% | Clawback on falsification |
| `InitialFundBalance` | "0" | 0 LGM | Pure PoT: no pre-minted fund |
| `BlocksPerEpoch` | 240,000 | ~1 week | Blocks per reward epoch |
| `DecayFactorBps` | 850,000 | 0.85/epoch | Reward decay per epoch |
| `FloorReward` | "100000" | 0.1 LGM | Minimum block reward floor |
| `FundHealthThresholdBps` | 100,000 | 10% | Fund health threshold (unused in pure PoT) |
| `ApiSuppressionExponent` | 2 | quadratic | API suppression curve (deferred) |

#### Evidence Category Vesting Curves

Release curves based on scientometric research. Half-lives derived from citation half-life studies and replication crisis data.

| Category | Half-Life (blocks) | Half-Life (~days) | Cliff (blocks) | Cliff (~days) | Max Release |
|----------|-------------------|-------------------|----------------|---------------|-------------|
| Axiomatic | 1,111,111 | ~32.4 | 11,111 | ~0.32 | 95% |
| Formal Proof | 555,555 | ~16.2 | 5,555 | ~0.16 | 92% |
| On-Chain | 222,222 | ~6.5 | 1,111 | ~0.03 | 90% |
| Cryptographic | 222,222 | ~6.5 | 3,333 | ~0.10 | 90% |
| Computational | 333,333 | ~9.7 | 2,222 | ~0.06 | 88% |
| Peer-Reviewed | 111,111 | ~3.2 | 5,555 | ~0.16 | 85% |
| Replicated | 111,111 | ~3.2 | 3,333 | ~0.10 | 88% |
| Oracle Feed | 55,555 | ~1.6 | 555 | ~0.02 | 80% |
| Attestation | 77,777 | ~2.3 | 2,222 | ~0.06 | 80% |
| Contested | 22,222 | ~0.6 | 1,111 | ~0.03 | 60% |

---

### x/gov — Governance (LIPs & Disbursements)

> Source: `x/gov/types/genesis.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `VotingPeriodBlocks` | 100,000 | ~2.9 days | Voting period for LIPs |
| `ReviewPeriodBlocks` | 50,000 | ~1.5 days | Review period before voting |
| `LastCallPeriodBlocks` | 50,000 | ~1.5 days | Last call before vote closes |
| `BaseCostPerVote` | "1000000" | 1 LGM | Base cost per quadratic vote |
| `MaxVotesPerVoter` | 100 | 100 votes | Max votes per voter per proposal |
| `SupportThresholdBps` | 7,500 | 75% | Support needed to pass |
| `QuorumThresholdBps` | 1,500 | 15% | Participation needed for quorum |
| `DisbursementStake` | "2000000000" | 2,000 LGM | Stake to propose disbursement |
| `DisbursementTimelockBlocks` | 50,000 | ~1.5 days | Timelock before execution |
| `DisbursementSupportBps` | 8,000 | 80% | Support for disbursement |
| `DisbursementQuorumBps` | 2,000 | 20% | Quorum for disbursement |

#### LIP Category Stake Requirements

| Category | Stake Required | Rationale |
|----------|---------------|-----------|
| Consensus | 1,000 LGM | Highest impact — changes core verification |
| Economics | 800 LGM | High impact — changes reward/penalty structure |
| Interface | 400 LGM | Moderate impact — changes interaction patterns |
| Process | 200 LGM | Low impact — changes procedures |

#### Quadratic Voting Cost Curve

| Votes Cast | Cost (LGM) | Marginal Cost |
|------------|-----------|---------------|
| 1 | 1 | 1 |
| 2 | 4 | 3 |
| 5 | 25 | 9 |
| 10 | 100 | 19 |
| 25 | 625 | 49 |
| 50 | 2,500 | 99 |
| 100 (max) | 10,000 | 199 |

Formula: `cost = votes² × BaseCostPerVote`

---

### x/emergency — Kill Switch

> Source: `x/emergency/types/genesis.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `HaltQuorum` | 750,000 | 75% | Guardian stake needed to halt |
| `RevertQuorum` | 800,000 | 80% | Guardian stake needed to revert |
| `ResumeQuorum` | 800,000 | 80% | Guardian stake needed to resume |
| `HaltPrevoteBlocks` | 11 | ~28s | Prevote phase for halt ceremony |
| `HaltPrecommitBlocks` | 11 | ~28s | Precommit phase for halt |
| `HaltTimeoutBlocks` | 44 | ~111s | Timeout for halt ceremony |
| `RevertPrevoteBlocks` | 22 | ~55s | Prevote phase for revert |
| `RevertPrecommitBlocks` | 22 | ~55s | Precommit phase for revert |
| `RevertTimeoutBlocks` | 111 | ~4.6 min | Timeout for revert ceremony |
| `ResumePrevoteBlocks` | 22 | ~55s | Prevote phase for resume |
| `ResumePrecommitBlocks` | 22 | ~55s | Precommit phase for resume |
| `ResumeTimeoutBlocks` | 111 | ~4.6 min | Timeout for resume ceremony |
| `MaxProposalsPerEpoch` | 3 | 3 | Anti-spam: max proposals per epoch |
| `MaxProposalsPerGuardianPerEpoch` | 1 | 1 | Anti-spam: per-guardian limit |
| `CooldownBlocks` | 111 | ~4.6 min | Cooldown between ceremonies |
| `MinGuardianStake` | "111111000000" | 111,111 LGM | Minimum stake for emergency powers |
| `MinDistinctVoters` | 4 | 4 | Minimum distinct voters for ceremony |
| `MaxRevertDepth` | 111,111 | ~3.2 days | Maximum blocks that can be reverted |
| `EpochBlocks` | 34,272 | ~1 day | Epoch length for anti-abuse tracking |
| `MaxHaltDurationBlocks` | 34,272 | ~1 day | Auto-resume if Guardians don't act |
| `GenesisCouncil` | [] | (devnet) | Honorary Guardians during bootstrap |
| `CouncilExpiryBlock` | 0 | (devnet) | When genesis council powers expire |
| `CouncilVirtualStake` | "11111000000" | 11,111 LGM | Virtual stake for council members |

---

### x/ontology — Knowledge Structure

> Source: `x/ontology/types/genesis.go`

#### Module Parameters

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `MinProposalStake` | "1000000" | 1 LGM | Stake to propose domain changes |
| `ProposalVotingPeriod` | 34,272 | ~1 day | Voting period for proposals |
| `MinEndorsements` | 3 | 3 | Minimum endorsements needed |
| `CrossStratumDiscount` | 50,000 | 5% | Cross-stratum reference discount |
| `MaxDomainsPerStratum` | 100 | 100 | Domain cap per stratum |
| `AllowNewStrata` | false | no | New strata require governance |

#### Epistemic Strata

The 7-level hierarchy encoding how knowledge types relate to verifiability:

| Level | Stratum | Max Confidence | Decay Rate | Complete? | Decidable? |
|-------|---------|---------------|------------|-----------|------------|
| 0 | **Axiomatic** | 100% | 0 (never) | Yes | Yes |
| 1 | **Formal** | 100% | 1 (negligible) | Yes | Yes |
| 2 | **Protocol** | 99% | 5 (minimal) | No | Yes |
| 3 | **Computational** | 98% | 10 (slow) | No | No |
| 4 | **Empirical** | 95% | 50 (moderate) | No | No |
| 5 | **Historical** | 90% | 100 (faster) | No | No |
| 6 | **Testimonial** | 80% | 200 (rapid) | No | No |

**Bootstrap domains:** logic, mathematics, protocol, computer_science, physics, general, history

---

### x/disputes — Dispute Resolution

> Source: `x/disputes/types/genesis.go`

#### Core Parameters

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `MaxActiveDisputes` | 3 | 3 | Max active disputes per fact |
| `EscalationDelay` | 17,136 | ~0.5 day | Blocks before escalation allowed |
| `SlashRateLoser` | 500,000 | 50% | Bond slashed from losing party |
| `RewardRateWinner` | 300,000 | 30% | Slash share to winning party |
| `ArbiterRewardRate` | 200,000 | 20% | Slash share to arbiters |

#### Dispute Tier Escalation

| Tier | Arbiters | Min Bond | Evidence Period | Voting Period | Quorum | Majority |
|------|----------|----------|----------------|---------------|--------|----------|
| Basic | 3 | 111 LGM | ~1 day | ~2 days | 66% | 77% |
| Escalated | 7 | 555 LGM | ~2 days | ~4 days | 66% | 77% |
| Final | 15 | 1,111 LGM | ~3 days | ~7 days | 66% | 77% |

---

### x/research — Research Fund

> Source: `x/research/types/types.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `MinResearchStake` | "100000000" | 100 LGM | Stake to submit research |
| `MinChallengeStake` | "100000000" | 100 LGM | Stake to challenge research |
| `ReviewPeriodBlocks` | 68,544 | ~2 days | Peer review window |
| `MinReviewerCount` | 3 | 3 | Minimum peer reviewers |
| `AcceptanceScoreThreshold` | 60 | 60/100 | Score needed for acceptance |
| `RejectionSlashBps` | 330,000 | 33% | Slash on rejected research |
| `MaxBountyReward` | "10000000000" | 10,000 LGM | Maximum bounty payout |
| `BountyMinDeadlineBlocks` | 34,272 | ~1 day | Minimum bounty deadline |

---

### x/evidence_mgmt — Evidence & Audit

> Source: `x/evidence_mgmt/types/genesis.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `MinAuditStake` | "22000000" | 22 LGM | Stake to submit audit |
| `AuditWindow` | 34,272 | ~1 day | Audit resolution window |
| `MinAuditsForConsensus` | 3 | 3 | Audits needed for consensus |
| `AccuracyThreshold` | 100,000 | 10% | Accuracy threshold for audit |
| `SlashRate` | 100,000 | 10% | Slash rate for failed audits |
| `RewardRate` | 50,000 | 5% | Reward rate for successful audits |
| `MinOracleStake` | "100000000" | 100 LGM | Stake to register as oracle |
| `OracleSlashRate` | 200,000 | 20% | Oracle misbehavior slash |
| `MaxEvidencePerClaim` | 10 | 10 | Max evidence items per claim |

---

### x/bvm — Belief Virtual Machine

> Source: `x/bvm/types/genesis.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `MaxBytecodeSize` | 1,048,576 | 1 MB | Maximum contract bytecode size |
| `MaxStateEntries` | 100,000 | 100K | Maximum state entries per contract |
| `MaxSchedulesPerContract` | 100 | 100 | Maximum scheduled executions |
| `DeployGasCost` | 200,000 | gas | Base gas for contract deployment |
| `CallBaseGasCost` | 21,000 | gas | Base gas for contract calls |
| `StorageWriteGasCost` | 5,000 | gas | Per-entry storage write |
| `StorageReadGasCost` | 2,000 | gas | Per-entry storage read |

---

### x/home — Agent Home

> Source: `x/home/types/genesis.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `MaxHomesPerAccount` | 3 | 3 | Max homes per identity |
| `MinCreationStake` | "1000000" | 1 LGM | Stake to create a home |
| `SessionMaxDuration` | 34,272 | ~1 day | Max home session length |
| `MaxKeysPerHome` | 20 | 20 | Max authorized keys |
| `MaxAlertsPerHome` | 100 | 100 | Max active guardian alerts |
| `MaxSessionsPerHome` | 5 | 5 | Concurrent sessions |
| `DeadmanMinThreshold` | 34,272 | ~1 day | Minimum deadman switch period |
| `AlertRetentionBlocks` | 342,720 | ~10 days | Alert history retention |

---

### x/channels — Payment Channels

> Source: `x/channels/types/genesis.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `MinDeposit` | "1000000" | 1 LGM | Minimum channel deposit |
| `MaxChannelDuration` | 1,036,800 | ~30 days | Maximum channel lifetime |
| `DefaultSettlementFreq` | 100 | 100 blocks | Default settlement frequency |
| `DisputePeriodBlocks` | 1,000 | ~42 min | Channel dispute window |
| `ExpiryGracePeriod` | 500 | ~21 min | Grace period after expiry |
| `FraudPenaltyBps` | 500 | 5% | Fraud penalty rate |
| `MaxChannelsPerPayer` | 50 | 50 | Max open channels per payer |

---

### x/tree — Tree of Life (Projects & Services)

> Source: `x/tree/types/genesis.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `MinBudget` | "1000000" | 1 LGM | Minimum project budget |
| `MaxTasksPerProject` | 200 | 200 | Max tasks per project |
| `MaxContributors` | 50 | 50 | Max contributors per project |
| `MaxApplications` | 100 | 100 | Max pending applications |
| `TaskDeadlineMinBlocks` | 100 | ~4 min | Minimum task deadline |
| `TaskDeadlineMaxBlocks` | 1,036,800 | ~30 days | Maximum task deadline |
| `MaxRejections` | 3 | 3 | Max rejections before close |
| `SeedExpiryBlocks` | 172,800 | ~5 days | Opportunity seed expiry |
| `MinContributorsToStart` | 1 | 1 | Contributors needed to start |
| `EvidenceTaxBp` | 220,000 | 22% | Evidence tax on project funds |
| `ContributorsBp` | 390,000 | 39% | Contributors' share |
| `ResearchFundBp` | 234,000 | 23.4% | Research fund share |
| `ProtocolTreasuryBp` | 156,000 | 15.6% | Protocol treasury share |

---

### x/partnerships — Human-Agent Partnerships

> Source: `x/partnerships/types/genesis.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `FormationWindowBlocks` | 77 | ~3.2 min | Window to accept partnership |
| `CoolingPeriodBlocks` | 222,222 | ~6.5 days | Cooling period after exit |
| `CommonPotShareBps` | 220,000 | 22% | Share to common pot |
| `SafetyFreezeDurationBlocks` | 222 | ~9.3 min | Safety freeze duration |
| `MaxFreezesPerEpoch` | 3 | 3 | Anti-abuse: max freezes |
| `CoercionReviewBlocks` | 1,111 | ~46.6 min | Coercion signal review period |
| `BaseCooldownBlocks` | 77 | ~3.2 min | Base cooldown between actions |
| `MaxCounterProposalDepth` | 3 | 3 | Max counter-proposal nesting |
| `DefaultHumanSplitBps` | 500,000 | 50% | Default human revenue split |
| `DefaultAgentSplitBps` | 500,000 | 50% | Default agent revenue split |

---

### x/qualification — Domain Qualification

> Source: `x/qualification/types/genesis.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `StakeLeaf` | 22 LGM | leaf stake | Stake for leaf domain qualification |
| `StakeMid` | 222 LGM | mid stake | Stake for mid-level domain |
| `StakeTop` | 2,222 LGM | top stake | Stake for top-level domain |
| `StakeLockPeriod` | 7,777 | ~5.4 hr | Stake lock period in blocks |
| `MinVerifications` | 19 | 19 | Minimum verifications for qualification |
| `MinAccuracyBps` | 7,700 | 77% | Minimum accuracy (10K basis) |
| `MinEndorsers` | 3 | 3 | Required endorsements |
| `EndorserStake` | 22 LGM | per endorser | Stake per endorsement |
| `EndorsementDecayBlocks` | 77,777 | ~2.3 days | Endorsement validity decay |
| `StratumDiscountBps` | 7,000 | 70% | Stratum cross-discount |
| `InactivityExpiration` | 777,777 | ~22.7 days | Inactivity expiration |
| `ProbationaryPeriod` | 777 | ~32.6 min | Probationary period |
| `ProbationaryWeightBps` | 5,000 | 50% | Probationary weight reduction |
| `SuspensionAccuracyBps` | 5,000 | 50% | Accuracy triggering suspension |
| `SuspensionConsecutive` | 5 | 5 | Consecutive failures for suspension |
| `RevocationChallenges` | 3 | 3 | Challenges to revoke |
| `MaxQualificationDepth` | 7 | 7 levels | Maximum depth hierarchy |

---

### x/capture_defense — Anti-Capture System

> Source: `x/capture_defense/types/genesis.go`, `x/capture_defense/types/types.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `StakeLockPeriod` | 7,777 | ~5.4 hr | Qualification stake lock |
| `MinVerificationsForQual` | 19 | 19 | Verifications for qualification |
| `MinAccuracyForQual` | 770,000 | 77% | Accuracy for qualification |
| `MinEndorsersForQual` | 3 | 3 | Endorsements for qualification |
| `InactivityExpirationBlocks` | 777,777 | ~22.7 days | Qualification expiry on inactivity |
| `ProbationaryPeriodBlocks` | 777 | ~32.6 min | Probationary period |
| `ReputationHalfLifeBlocks` | 77,777 | ~2.3 days | Reputation score half-life |
| `DetectionIntervalBlocks` | 777 | ~32.6 min | Capture detection interval |
| `DecayIntervalBlocks` | 7,777 | ~5.4 hr | Capture metric decay |
| `HerfindahlTriggerThreshold` | 400,000 | 0.4 | HHI threshold for capture alarm |
| `MaxExpirationsPerBlock` | 100 | 100 | Max expirations processed per block |

---

### x/capture_challenge — Capture Challenges

> Source: `x/capture_challenge/types/types.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `EvidencePeriodBlocks` | 777 | ~32.6 min | Evidence submission window |
| `ReviewPeriodBlocks` | 7,777 | ~5.4 hr | Challenge review period |
| `DomainPauseDurationBlocks` | 777 | ~32.6 min | Domain pause on capture confirmation |

---

### x/claiming_pot — Claiming Pots

> Source: `x/claiming_pot/types/genesis.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `MaxPotsPerCreator` | 100 | 100 | Max pots per creator |
| `MinInitialDeposit` | "1000000" | 1 LGM | Minimum initial deposit |
| `MaxAllowlistSize` | 10,000 | 10K | Max allowlist entries |
| `MinAmountPerClaim` | "1000" | 0.001 LGM | Minimum claim amount |

---

### x/schedule — Scheduled Execution

> Source: `x/schedule/types/genesis.go`

| Parameter | Value | Human | Description |
|-----------|-------|-------|-------------|
| `MaxActivePerAccount` | 100 | 100 | Max active schedules per account |
| `MaxGasPerBlock` | 2,000,000 | gas | Max gas for scheduled execution |
| `MinIntervalBlocks` | 1 | 1 block | Minimum execution interval |
| `MaxFutureBlocks` | 12,500,000 | ~1 year | Maximum schedule horizon |
| `MinPrepaidExecutions` | 10 | 10 | Minimum prepaid executions |
| `ExpiredRetentionBlocks` | 1,000 | ~42 min | Retention for expired schedules |

---

## How to Change Parameters

Every parameter in Layer 2 (Genome) can be changed through governance. Here's how.

### Step 1: Submit a LIP (Legible Improvement Proposal)

Decide which category your change falls under — this determines the stake required:

| Category | Stake | Examples |
|----------|-------|---------|
| **Consensus** | 1,000 LGM | Changing `AcceptanceThreshold`, `MinVerifiers`, phase durations |
| **Economics** | 800 LGM | Changing `BlockRewardAmount`, `InvalidClaimSlashBps`, tier configs |
| **Interface** | 400 LGM | Changing `MaxSessionKeys`, `MaxChannelDuration`, gas costs |
| **Process** | 200 LGM | Changing `ReviewPeriodBlocks`, `MaxApplications`, dispute timers |

```bash
# Via CLI
legbled tx lgm-gov submit-lip \
  --title "Increase MinVerifiers from 3 to 5" \
  --description "Improves verification quality by requiring more independent opinions per round." \
  --category consensus \
  --from my-account \
  --chain-id legible-devnet-1 \
  --fees 100000ulgm
```

```
# Via MCP
lgm_submit_lip({
  title: "Increase MinVerifiers from 3 to 5",
  description: "Improves verification quality by requiring more independent opinions per round.",
  category: "consensus",
  stakeAmount: "1000000000"
})
```

### Step 2: LIP Lifecycle

Your proposal goes through these stages:

```
draft → (stake deposited) → review (~1.5 days) → last_call (~1.5 days) → voting (~2.9 days)
                                                                              ↓
                                                                    accepted / rejected
```

### Step 3: Vote Using Quadratic Voting

Anyone at Tier 1+ can vote. Cost increases quadratically — having 10x the tokens only gives you ~3.2x the voting power.

```bash
# Vote with 5 votes in favor (costs 25 LGM)
legbled tx lgm-gov cast-vote \
  --lip-number 1 \
  --support true \
  --votes 5 \
  --from my-account \
  --chain-id legible-devnet-1 \
  --fees 30000ulgm
```

```
# Via MCP
lgm_cast_vote({
  lipNumber: 1,
  support: true,
  votes: 5
})
```

### Step 4: Execution

If the proposal passes (75% support, 15% quorum), the parameter change is executed automatically via `MsgUpdateParams` on the target module.

### Querying Current Parameters

```bash
# Query any module's parameters
legbled query legible-knowledge params
legbled query legible-staking params
legbled query legible-gov params
legbled query legible-emergency params
legbled query lgm-vesting-rewards params
```

```
# Via MCP
lgm_query_params({ module: "knowledge" })
lgm_query_params({ module: "staking" })
```

---

## The Autopoiesis Engine

> **Status**: Designed, not yet activated. See [Recursive Growth Framework](RECURSIVE_GROWTH_FRAMEWORK.md).

When activated, the autopoiesis engine adds Layer 3 (Phenotype) — automatic runtime adjustments within safety bounds.

### How It Works

Five "hormones" observe protocol health metrics and adjust multipliers:

| Hormone | Name | Observes | Adjusts |
|---------|------|----------|---------|
| **CRI** | Confidence Retention Index | Confidence retention per domain/category | Decay rates, reverification bonuses |
| **DVI** | Domain Vitality Index | Fact creation vs expiration per domain | Domain reward weights, claim stakes |
| **VCF** | Verification Cascade Factor | Citation network density per domain | Citation weights, domain priors |
| **SSI** | System Stress Index | Overall validator accuracy, challenge rates | Slash rates, block rewards, min verifiers |
| **GRM** | Growth Rate Modulator | Verification throughput | Max transactions, gas costs |

### Safety Bounds

Every multiplier is bounded. The engine cannot exceed these limits, period:

| Multiplier | Min | Max | Rate Limit |
|------------|-----|-----|------------|
| `decay.*` | 0.25x | 4.0x | ±5%/epoch |
| `reward.*` | 0.5x | 2.0x | ±5%/epoch |
| `slash.*` | 0.5x | 3.0x | ±10%/epoch (SSI) |
| `gas.*` | 0.5x | 2.0x | ±5%/epoch |
| `blockreward.*` | 0.25x | 4.0x | ±5%/epoch |
| `claimstake.*` | 0.5x | 10.0x | ±5%/epoch |

### Governance Override

Governance can at any time:
1. **Change base parameters** via `MsgUpdateParams` (affects Layer 2)
2. **Change safety bounds** via `MsgUpdateAutopoiesisParams` (affects Layer 3 limits)
3. **Override a specific multiplier** via `MsgOverrideMultiplier`
4. **Activate/deactivate** the engine via `MsgActivateAutopoiesis`

The engine is a tool, not a master. Participants always have the final say.

---

## Reading the Numbers

### BPS (Basis Points)

Most percentages use a **1,000,000 = 100%** scale (not the traditional 10,000 = 100%):

| Value | Percentage |
|-------|-----------|
| 1,000,000 | 100% |
| 770,000 | 77% |
| 500,000 | 50% |
| 220,000 | 22% |
| 100,000 | 10% |
| 50,000 | 5% |
| 10,000 | 1% |

### Token Amounts

All on-chain amounts are in **ulgm** (micro-LGM):

| ulgm | LGM |
|------|-----|
| 1,000 | 0.001 |
| 1,000,000 | 1 |
| 111,000,000 | 111 |
| 1,111,000,000 | 1,111 |
| 11,111,000,000 | 11,111 |

### Block Durations

At 2521ms per block:

| Blocks | Duration |
|--------|----------|
| 1 | ~2.5 seconds |
| 111 | ~4.7 minutes |
| 1,111 | ~46.6 minutes |
| 34,272 | ~1 day |
| 240,000 | ~7 days |
| 1,036,800 | ~30 days |
| 12,500,000 | ~1 year |

---

## A Living Document

This page will evolve. As governance proposals change parameters, as the autopoiesis engine comes online, as new modules are added — the numbers here will change.

That's the point. A protocol that can't adapt is a protocol that can't survive. But adaptation without transparency is just arbitrary power. Every change goes through governance. Every parameter is visible. Every vote counts quadratically — not linearly.

If you see a number on this page that you think should be different, you now know exactly how to change it. Submit a LIP, make your case, and let the community decide.

The protocol improves because participants — human and agent — engage with its parameters, challenge its assumptions, and vote on its future. Every governance proposal, whether it passes or fails, teaches the protocol something about what its participants value.

This is what it means to be legible.

---

*Last updated: 2026-02-16. Parameter values reflect genesis defaults for legible-devnet-1.*
