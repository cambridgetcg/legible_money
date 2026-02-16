package app

import (
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"
)

// NewAnteHandler returns an AnteHandler with:
// 1. Standard Cosmos SDK decorators (explicit chain, not wrapped)
// 2. LGM-specific gas cost validation
// 3. Fee routing: 22% to research fund, 78% to validators
// 4. Legible Money custom decorators:
//    - DID resolution and validation
//    - Frozen account enforcement + LastActiveBlock tracking
//    - Session key capability enforcement
func NewAnteHandler(app *LegibleApp) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		// --- IBC ---
		ibcante.NewRedundantRelayDecorator(app.IBCKeeper),

		// --- Gas Meter Init (must be before any gas consumption) ---
		ante.NewSetUpContextDecorator(),

		// --- Bootstrap Gas-Free (waives gas for PoT txs during first ~14 days) ---
		NewBootstrapGasFreeDecorator(),

		// --- Emergency Halt Gate (blocks non-emergency txs when chain is halted) ---
		NewEmergencyHaltDecorator(app.EmergencyKeeper),

		// --- LGM Pre-Auth (gas meter available) ---
		NewLGMGasDecorator(),
		NewFeeRouterDecorator(app.BankKeeper),

		// --- Standard Cosmos SDK Decorators ---
		ante.NewExtensionOptionsDecorator(nil),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(app.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(app.AccountKeeper),
		ante.NewDeductFeeDecorator(app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, nil),
		ante.NewSetPubKeyDecorator(app.AccountKeeper),
		ante.NewValidateSigCountDecorator(app.AccountKeeper),
		ante.NewSigGasConsumeDecorator(app.AccountKeeper, ante.DefaultSigVerificationGasConsumer),
		ante.NewSigVerificationDecorator(app.AccountKeeper, app.txConfig.SignModeHandler()),
		ante.NewIncrementSequenceDecorator(app.AccountKeeper),

		// --- Legible Money Post-Auth (signer authenticated) ---
		NewLegibleDIDDecorator(app.LegibleAuthKeeper),
		NewLegibleAccountDecorator(app.LegibleAuthKeeper),
		NewLegibleCapabilityDecorator(app.LegibleAuthKeeper, app.AccountKeeper),
	)
}

// ---------- LGM Gas Decorator ----------

// LGMGasDecorator validates that transactions provide sufficient gas based on
// the LGM gas cost table (translated from core/billing/gas.ts).
type LGMGasDecorator struct{}

// NewLGMGasDecorator creates a new LGMGasDecorator.
func NewLGMGasDecorator() LGMGasDecorator {
	return LGMGasDecorator{}
}

func (lgd LGMGasDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errors.Wrap(sdkerrors.ErrTxDecode, "tx must implement FeeTx")
	}

	gasLimit := feeTx.GetGas()

	// Enforce block gas limit
	if gasLimit > BlockGasLimit {
		return ctx, errors.Wrapf(sdkerrors.ErrOutOfGas,
			"tx gas limit %d exceeds block gas limit %d", gasLimit, BlockGasLimit)
	}

	// Enforce per-tx gas limit
	if gasLimit > TxGasLimit {
		return ctx, errors.Wrapf(sdkerrors.ErrOutOfGas,
			"tx gas limit %d exceeds per-tx limit %d", gasLimit, TxGasLimit)
	}

	// Calculate minimum required gas from message types
	msgs := tx.GetMsgs()
	var totalRequiredGas uint64
	for _, msg := range msgs {
		msgType := sdk.MsgTypeURL(msg)
		requiredGas := lookupMsgGas(msgType)
		totalRequiredGas += requiredGas
	}

	if totalRequiredGas < MinGasLimit {
		totalRequiredGas = MinGasLimit
	}

	if gasLimit < totalRequiredGas {
		return ctx, errors.Wrapf(sdkerrors.ErrOutOfGas,
			"tx gas limit %d below minimum required %d for %d messages",
			gasLimit, totalRequiredGas, len(msgs))
	}

	// Enforce minimum gas price
	feeCoins := feeTx.GetFee()
	if !feeCoins.IsZero() {
		minFee := sdk.NewCoin(BondDenom, math.NewIntFromUint64(gasLimit*MinGasPrice))
		if feeCoins.AmountOf(BondDenom).LT(minFee.Amount) {
			return ctx, errors.Wrapf(sdkerrors.ErrInsufficientFee,
				"fee %s below minimum %s (gas %d * min price %d)",
				feeCoins, minFee, gasLimit, MinGasPrice)
		}
	}

	return next(ctx, tx, simulate)
}

// lookupMsgGas maps a Cosmos SDK message type URL to its LGM gas cost.
// Type URLs are like "/legible.knowledge.v1.MsgSubmitClaim".
func lookupMsgGas(msgTypeURL string) uint64 {
	// Map proto message type URLs to our gas cost keys
	if cost, ok := msgTypeURLToGas[msgTypeURL]; ok {
		return cost
	}
	// Unknown message types get minimum gas
	return MinGasLimit
}

// msgTypeURLToGas maps protobuf message type URLs to gas costs.
// Built from the TransactionGasCosts table and proto service definitions.
var msgTypeURLToGas = map[string]uint64{
	// Standard Cosmos messages
	"/cosmos.bank.v1beta1.MsgSend":           TransactionGasCosts["transfer"],
	"/cosmos.bank.v1beta1.MsgMultiSend":      TransactionGasCosts["transfer"] * 2,
	"/cosmos.staking.v1beta1.MsgDelegate":    TransactionGasCosts["delegate"],
	"/cosmos.staking.v1beta1.MsgUndelegate":  TransactionGasCosts["undelegate"],
	"/cosmos.staking.v1beta1.MsgBeginRedelegate": TransactionGasCosts["redelegate"],
	"/cosmos.gov.v1.MsgSubmitProposal":       TransactionGasCosts["governance_propose"],
	"/cosmos.gov.v1.MsgVote":                 TransactionGasCosts["governance_vote"],

	// IBC
	"/ibc.applications.transfer.v1.MsgTransfer": TransactionGasCosts["transfer"],

	// Knowledge module
	"/legible.knowledge.v1.MsgSubmitClaim":    TransactionGasCosts["claim_submit"],
	"/legible.knowledge.v1.MsgSubmitCommitment": TransactionGasCosts["verification_commit"],
	"/legible.knowledge.v1.MsgSubmitReveal":   TransactionGasCosts["verification_reveal"],
	"/legible.knowledge.v1.MsgChallengeFact":  TransactionGasCosts["challenge_fact"],
	"/legible.knowledge.v1.MsgAddFact":        TransactionGasCosts["add_fact"],
	"/legible.knowledge.v1.MsgProposeDomain":  TransactionGasCosts["propose_domain"],
	"/legible.knowledge.v1.MsgEndorseDomainProposal": TransactionGasCosts["endorse_domain"],
	"/legible.knowledge.v1.MsgChallengeDomainProposal": TransactionGasCosts["challenge_domain"],
	"/legible.knowledge.v1.MsgRegisterStratum": TransactionGasCosts["register_stratum"],

	// Auth module (agent keys)
	"/legible.auth.v1.MsgRegisterAccount": TransactionGasCosts["register_account"],
	"/legible.auth.v1.MsgRotateKey":       TransactionGasCosts["rotate_key"],
	"/legible.auth.v1.MsgCreateSession":   TransactionGasCosts["create_session"],
	"/legible.auth.v1.MsgRevokeSession":   TransactionGasCosts["revoke_session"],

	// Staking module (extended)
	"/legible.staking.v1.MsgRegisterValidator": TransactionGasCosts["register_validator"],

	// Disputes
	"/legible.disputes.v1.MsgInitiateDispute":  TransactionGasCosts["initiate_dispute"],
	"/legible.disputes.v1.MsgCommitEvidence":   TransactionGasCosts["commit_evidence"],
	"/legible.disputes.v1.MsgRevealEvidence":   TransactionGasCosts["reveal_evidence"],
	"/legible.disputes.v1.MsgArbiterVote":      TransactionGasCosts["arbiter_vote"],
	"/legible.disputes.v1.MsgEscalateDispute":  TransactionGasCosts["escalate_dispute"],
	"/legible.disputes.v1.MsgSettleDispute":     TransactionGasCosts["settle_dispute"],
	"/legible.disputes.v1.MsgTimeoutDispute":   TransactionGasCosts["timeout_dispute"],

	// Research
	"/legible.research.v1.MsgSubmitResearch":    TransactionGasCosts["submit_research"],
	"/legible.research.v1.MsgChallengeResearch": TransactionGasCosts["challenge_research"],
	"/legible.research.v1.MsgReviewResearch":    TransactionGasCosts["review_research"],
	"/legible.research.v1.MsgResolveResearch":   TransactionGasCosts["resolve_research"],
	"/legible.research.v1.MsgCreateBounty":      TransactionGasCosts["create_bounty"],
	"/legible.research.v1.MsgClaimBounty":       TransactionGasCosts["claim_bounty"],
	"/legible.research.v1.MsgFulfillBounty":     TransactionGasCosts["fulfill_bounty"],
	"/legible.research.v1.MsgFundResearch":      TransactionGasCosts["fund_research"],

	// Evidence management
	"/legible.evidence_mgmt.v1.MsgSubmitEvidence":  TransactionGasCosts["submit_evidence"],
	"/legible.evidence_mgmt.v1.MsgArchiveEvidence": TransactionGasCosts["archive_evidence"],
	"/legible.evidence_mgmt.v1.MsgAuditSubmit":     TransactionGasCosts["audit_submit"],
	"/legible.evidence_mgmt.v1.MsgAuditResolve":    TransactionGasCosts["audit_resolve"],
	"/legible.evidence_mgmt.v1.MsgRegisterOracle":  TransactionGasCosts["register_oracle"],

	// Vesting rewards
	"/legible.vesting_rewards.v1.MsgCreateVesting":     TransactionGasCosts["create_vesting"],
	"/legible.vesting_rewards.v1.MsgClaimVesting":      TransactionGasCosts["claim_vesting"],
	"/legible.vesting_rewards.v1.MsgPauseVesting":      TransactionGasCosts["pause_vesting"],
	"/legible.vesting_rewards.v1.MsgResumeVesting":     TransactionGasCosts["resume_vesting"],
	"/legible.vesting_rewards.v1.MsgAccelerateVesting": TransactionGasCosts["accelerate_vesting"],
	"/legible.vesting_rewards.v1.MsgFalsifyVesting":    TransactionGasCosts["falsify_vesting"],
	"/legible.vesting_rewards.v1.MsgCompleteVesting":   TransactionGasCosts["complete_vesting"],

	// Secondary tokens (LGM-20)
	"/legible.tokens.v1.MsgCreateToken":     TransactionGasCosts["token_create"],
	"/legible.tokens.v1.MsgMintToken":       TransactionGasCosts["token_mint"],
	"/legible.tokens.v1.MsgBurnToken":       TransactionGasCosts["token_burn"],
	"/legible.tokens.v1.MsgTransferToken":   TransactionGasCosts["token_transfer_secondary"],
	"/legible.tokens.v1.MsgApproveToken":    TransactionGasCosts["token_approve"],
	"/legible.tokens.v1.MsgTransferFrom":    TransactionGasCosts["token_transfer_from"],
	"/legible.tokens.v1.MsgPauseToken":      TransactionGasCosts["token_pause"],
	"/legible.tokens.v1.MsgUnpauseToken":    TransactionGasCosts["token_unpause"],
	"/legible.tokens.v1.MsgDelegatePower":   TransactionGasCosts["delegate_power"],
	"/legible.tokens.v1.MsgWrapToken":       TransactionGasCosts["wrap_token"],
	"/legible.tokens.v1.MsgUnwrapToken":     TransactionGasCosts["unwrap_token"],
	"/legible.tokens.v1.MsgAddLiquidity":    TransactionGasCosts["lp_add_liquidity"],
	"/legible.tokens.v1.MsgRemoveLiquidity": TransactionGasCosts["lp_remove_liquidity"],
	"/legible.tokens.v1.MsgSwap":            TransactionGasCosts["lp_swap"],

	// Payment channels
	"/legible.channels.v1.MsgOpenChannel":    TransactionGasCosts["open_channel"],
	"/legible.channels.v1.MsgUpdateChannel":  TransactionGasCosts["update_channel"],
	"/legible.channels.v1.MsgCloseChannel":   TransactionGasCosts["close_channel"],
	"/legible.channels.v1.MsgSettleChannel":  TransactionGasCosts["settle_channel"],
	"/legible.channels.v1.MsgDisputeChannel": TransactionGasCosts["dispute_channel"],

	// Tree of Life — projects
	"/legible.tree.v1.MsgCreateProject":       TransactionGasCosts["create_project"],
	"/legible.tree.v1.MsgProposeProject":      TransactionGasCosts["propose_project"],
	"/legible.tree.v1.MsgStartDevelopment":    TransactionGasCosts["start_development"],
	"/legible.tree.v1.MsgCompleteProject":     TransactionGasCosts["complete_project"],
	"/legible.tree.v1.MsgPauseProject":        TransactionGasCosts["pause_project"],
	"/legible.tree.v1.MsgResumeProject":       TransactionGasCosts["resume_project"],
	"/legible.tree.v1.MsgAbandonProject":      TransactionGasCosts["abandon_project"],
	"/legible.tree.v1.MsgSpawnChildProject":   TransactionGasCosts["spawn_child_project"],

	// Tree of Life — tasks
	"/legible.tree.v1.MsgAddTask":             TransactionGasCosts["add_task"],
	"/legible.tree.v1.MsgAssignTask":          TransactionGasCosts["assign_task"],
	"/legible.tree.v1.MsgStartWork":           TransactionGasCosts["start_work"],
	"/legible.tree.v1.MsgSubmitDeliverable":   TransactionGasCosts["submit_deliverable"],
	"/legible.tree.v1.MsgApproveDeliverable":  TransactionGasCosts["approve_deliverable"],
	"/legible.tree.v1.MsgRejectDeliverable":   TransactionGasCosts["reject_deliverable"],
	"/legible.tree.v1.MsgReopenTask":          TransactionGasCosts["reopen_task"],

	// Tree of Life — services
	"/legible.tree.v1.MsgDeployService":       TransactionGasCosts["deploy_service"],
	"/legible.tree.v1.MsgCallService":         TransactionGasCosts["call_service"],
	"/legible.tree.v1.MsgSubscribeService":    TransactionGasCosts["subscribe_service"],
	"/legible.tree.v1.MsgPauseService":        TransactionGasCosts["pause_service"],
	"/legible.tree.v1.MsgResumeService":       TransactionGasCosts["resume_service"],
	"/legible.tree.v1.MsgRetireService":       TransactionGasCosts["retire_service"],

	// Tree of Life — seeds & contributors
	"/legible.tree.v1.MsgBeginSeeding":        TransactionGasCosts["begin_seeding"],
	"/legible.tree.v1.MsgDetectOpportunity":   TransactionGasCosts["detect_opportunity"],
	"/legible.tree.v1.MsgClaimOpportunity":    TransactionGasCosts["claim_opportunity"],
	"/legible.tree.v1.MsgAddContributor":      TransactionGasCosts["add_contributor"],
	"/legible.tree.v1.MsgApplyToProject":      TransactionGasCosts["apply_to_project"],
	"/legible.tree.v1.MsgReviewApplication":   TransactionGasCosts["review_application"],
	"/legible.tree.v1.MsgSetAvailability":     TransactionGasCosts["set_availability"],

	// BVM
	"/legible.bvm.v1.MsgDeployContract":    TransactionGasCosts["deploy_contract"],
	"/legible.bvm.v1.MsgCallContract":      TransactionGasCosts["call_contract"],
	"/legible.bvm.v1.MsgScheduleExecution": TransactionGasCosts["schedule_execution"],

	// Governance (extended)
	"/legible.gov.v1.MsgSubmitLIP":          TransactionGasCosts["submit_lip"],
	"/legible.gov.v1.MsgCastVote":           TransactionGasCosts["cast_vote"],
	"/legible.gov.v1.MsgLockVote":           TransactionGasCosts["lock_vote"],
	"/legible.gov.v1.MsgUnlockVote":         TransactionGasCosts["unlock_vote"],
	"/legible.gov.v1.MsgCommitReview":       TransactionGasCosts["commit_review"],
	"/legible.gov.v1.MsgRevealReview":       TransactionGasCosts["reveal_review"],
	"/legible.gov.v1.MsgSubmitDisbursement": TransactionGasCosts["submit_disbursement"],
	"/legible.gov.v1.MsgExecuteDisbursement": TransactionGasCosts["execute_disbursement"],

	// Emergency
	"/legible.emergency.v1.MsgProposeHalt":   TransactionGasCosts["propose_halt"],
	"/legible.emergency.v1.MsgVoteHalt":      TransactionGasCosts["vote_halt"],
	"/legible.emergency.v1.MsgProposeRevert": TransactionGasCosts["propose_revert"],
	"/legible.emergency.v1.MsgVoteRevert":    TransactionGasCosts["vote_revert"],
	"/legible.emergency.v1.MsgProposeResume": TransactionGasCosts["propose_resume"],
	"/legible.emergency.v1.MsgVoteResume":    TransactionGasCosts["vote_resume"],

	// Claiming pots
	"/legible.claiming_pot.v1.MsgCreatePot":     TransactionGasCosts["create_pot"],
	"/legible.claiming_pot.v1.MsgFundPot":       TransactionGasCosts["fund_pot"],
	"/legible.claiming_pot.v1.MsgClaimFromPot":  TransactionGasCosts["claim_from_pot"],
	"/legible.claiming_pot.v1.MsgClosePot":      TransactionGasCosts["close_pot"],

	// Capture defense
	"/legible.capture_defense.v1.MsgRequestQualification": TransactionGasCosts["request_capture_qualification"],
	"/legible.capture_defense.v1.MsgEndorseQualification": TransactionGasCosts["endorse_capture_qualification"],

	// Capture challenge
	"/legible.capture_challenge.v1.MsgSubmitChallenge":  TransactionGasCosts["submit_capture_challenge"],
	"/legible.capture_challenge.v1.MsgAddEvidence":      TransactionGasCosts["add_challenge_evidence"],
	"/legible.capture_challenge.v1.MsgResolveChallenge": TransactionGasCosts["resolve_capture_challenge"],

	// Agent Home
	"/legible.home.v1.MsgCreateHome":        TransactionGasCosts["create_home"],
	"/legible.home.v1.MsgUpdateHome":        TransactionGasCosts["update_home"],
	"/legible.home.v1.MsgUpdateMemoryCID":   TransactionGasCosts["update_memory_cid"],
	"/legible.home.v1.MsgStartSession":      TransactionGasCosts["home_start_session"],
	"/legible.home.v1.MsgEndSession":        TransactionGasCosts["home_end_session"],
	"/legible.home.v1.MsgRegisterKey":       TransactionGasCosts["home_register_key"],
	"/legible.home.v1.MsgRevokeKey":         TransactionGasCosts["home_revoke_key"],
	"/legible.home.v1.MsgConfigureGuardian": TransactionGasCosts["configure_guardian"],
	"/legible.home.v1.MsgAcknowledgeAlert":  TransactionGasCosts["acknowledge_alert"],
	"/legible.home.v1.MsgSetSpendingLimit":  TransactionGasCosts["set_spending_limit"],

	// Qualification
	"/legible.qualification.v1.MsgRequestQualification":  TransactionGasCosts["request_qualification"],
	"/legible.qualification.v1.MsgEndorseValidator":       TransactionGasCosts["endorse_validator"],
	"/legible.qualification.v1.MsgRenewQualification":     TransactionGasCosts["renew_qualification"],
	"/legible.qualification.v1.MsgWithdrawQualification":  TransactionGasCosts["withdraw_qualification"],

	// Auth recovery (hand-written ExtendedMsg service)
	"/legible.auth.v1.MsgRecoverAccount":       TransactionGasCosts["recover_account"],
	"/legible.auth.v1.MsgFreezeAccount":        TransactionGasCosts["freeze_account"],
	"/legible.auth.v1.MsgUnfreezeAccount":      TransactionGasCosts["unfreeze_account"],
	"/legible.auth.v1.MsgSetRecoveryConfig":    TransactionGasCosts["set_recovery_config"],
	"/legible.auth.v1.MsgInitiateRecovery":     TransactionGasCosts["initiate_recovery"],
	"/legible.auth.v1.MsgSubmitRecoveryShard":  TransactionGasCosts["submit_recovery_shard"],
	"/legible.auth.v1.MsgChallengeRecovery":    TransactionGasCosts["challenge_recovery"],
	"/legible.auth.v1.MsgExecuteRecovery":      TransactionGasCosts["execute_recovery"],

	// Governance extras
	"/legible.gov.v1.MsgStakeLIP":             TransactionGasCosts["governance_stake_lip"],
	"/legible.gov.v1.MsgAmendLIP":             TransactionGasCosts["amend_lip"],
	"/legible.gov.v1.MsgAdvanceLIPStage":      TransactionGasCosts["advance_lip_stage"],
	"/legible.gov.v1.MsgWithdrawLIP":          TransactionGasCosts["withdraw_lip"],
	"/legible.gov.v1.MsgSwitchVote":           TransactionGasCosts["switch_vote"],
	"/legible.gov.v1.MsgFinalizeReview":       TransactionGasCosts["finalize_review"],
	"/legible.gov.v1.MsgStakeDisbursement":    TransactionGasCosts["stake_disbursement"],
	"/legible.gov.v1.MsgAdvanceDisbursement":  TransactionGasCosts["advance_disbursement"],
	"/legible.gov.v1.MsgCancelDisbursement":   TransactionGasCosts["cancel_disbursement"],
	"/legible.gov.v1.MsgRegisterOperator":     TransactionGasCosts["register_operator"],
	"/legible.gov.v1.MsgAddAgent":             TransactionGasCosts["add_agent"],
	"/legible.gov.v1.MsgRemoveAgent":          TransactionGasCosts["remove_agent"],
	"/legible.gov.v1.MsgSlashOperator":        TransactionGasCosts["slash_operator"],
	"/legible.gov.v1.MsgCreateDeployment":     TransactionGasCosts["create_deployment"],
	"/legible.gov.v1.MsgAdvanceDeployment":    TransactionGasCosts["advance_deployment"],
	"/legible.gov.v1.MsgApproveDeployment":    TransactionGasCosts["approve_deployment"],
	"/legible.gov.v1.MsgRollbackDeployment":   TransactionGasCosts["rollback_deployment"],

	// Discovery
	"/legible.discovery.v1.MsgRegisterAgent":     TransactionGasCosts["register_agent"],
	"/legible.discovery.v1.MsgUpdateProfile":     TransactionGasCosts["update_profile"],
	"/legible.discovery.v1.MsgAddCapability":     TransactionGasCosts["add_capability"],
	"/legible.discovery.v1.MsgRemoveCapability":  TransactionGasCosts["remove_capability"],
	"/legible.discovery.v1.MsgDeregisterAgent":   TransactionGasCosts["deregister_agent"],
	"/legible.discovery.v1.MsgHeartbeat":         TransactionGasCosts["heartbeat"],

	// Partnerships
	"/legible.partnerships.v1.MsgInitiatePartnership":  TransactionGasCosts["initiate_partnership"],
	"/legible.partnerships.v1.MsgAcceptPartnership":    TransactionGasCosts["accept_partnership"],
	"/legible.partnerships.v1.MsgDepositToPot":         TransactionGasCosts["deposit_to_pot"],
	"/legible.partnerships.v1.MsgDistributeReward":     TransactionGasCosts["distribute_reward"],
	"/legible.partnerships.v1.MsgProposeOperation":     TransactionGasCosts["propose_operation"],
	"/legible.partnerships.v1.MsgApproveOperation":     TransactionGasCosts["approve_operation"],
	"/legible.partnerships.v1.MsgRejectOperation":      TransactionGasCosts["reject_operation"],
	"/legible.partnerships.v1.MsgSafetyFreeze":         TransactionGasCosts["safety_freeze"],
	"/legible.partnerships.v1.MsgRaiseCoercionSignal":  TransactionGasCosts["raise_coercion_signal"],
	"/legible.partnerships.v1.MsgInitiateExit":         TransactionGasCosts["initiate_exit"],

	// Ontology
	"/legible.ontology.v1.MsgProposeDomain":             TransactionGasCosts["propose_domain"],
	"/legible.ontology.v1.MsgVoteDomainProposal":        TransactionGasCosts["vote_domain_proposal"],
	"/legible.ontology.v1.MsgUpdateDomain":              TransactionGasCosts["update_domain"],
	"/legible.ontology.v1.MsgRegisterLogicZone":         TransactionGasCosts["register_logic_zone"],
	"/legible.ontology.v1.MsgAcknowledgeIncompleteness": TransactionGasCosts["acknowledge_incompleteness"],

	// Schedule
	"/legible.schedule.v1.MsgCreateSchedule":  TransactionGasCosts["create_schedule"],
	"/legible.schedule.v1.MsgPauseSchedule":   TransactionGasCosts["pause_schedule"],
	"/legible.schedule.v1.MsgResumeSchedule":  TransactionGasCosts["resume_schedule"],
	"/legible.schedule.v1.MsgCancelSchedule":  TransactionGasCosts["cancel_schedule"],
	"/legible.schedule.v1.MsgAddFee":          TransactionGasCosts["add_fee"],

	// Billing
	"/legible.billing.v1.MsgRegisterProvider":    TransactionGasCosts["register_provider"],
	"/legible.billing.v1.MsgDeregisterProvider":  TransactionGasCosts["deregister_provider"],
	"/legible.billing.v1.MsgRequestQuote":        TransactionGasCosts["request_quote"],
	"/legible.billing.v1.MsgExecutePayment":      TransactionGasCosts["execute_payment"],
}

// ---------- Fee Router Decorator ----------

// FeeRouterDecorator splits collected fees: 22% to research fund, 78% to validators.
// This implements the LGM fee routing from core/billing/reward_router.ts.
type FeeRouterDecorator struct {
	bankKeeper bankkeeper.Keeper
}

// NewFeeRouterDecorator creates a new FeeRouterDecorator.
func NewFeeRouterDecorator(bk bankkeeper.Keeper) FeeRouterDecorator {
	return FeeRouterDecorator{bankKeeper: bk}
}

func (frd FeeRouterDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Fee routing is handled in x/vesting_rewards BeginBlocker via keeper.RouteFees().
	// The BeginBlocker sweeps 22% of fee_collector to research_fund before
	// x/distribution's BeginBlocker sends the remaining 78% to validators.
	//
	// This decorator only logs the intended split for observability.

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return next(ctx, tx, simulate)
	}

	fee := feeTx.GetFee()
	if fee.IsZero() {
		return next(ctx, tx, simulate)
	}

	ctx.Logger().Debug("LGM fee routing",
		"total_fee", fee.String(),
		"research_share_bps", ResearchTaxBPS,
		"validator_share_bps", ValidatorFeeBPS,
	)

	return next(ctx, tx, simulate)
}

// GetMinimumFee calculates the minimum fee for a transaction based on message types.
func GetMinimumFee(msgs []sdk.Msg) sdk.Coins {
	var totalGas uint64
	for _, msg := range msgs {
		totalGas += lookupMsgGas(sdk.MsgTypeURL(msg))
	}
	if totalGas < MinGasLimit {
		totalGas = MinGasLimit
	}

	minFee := math.NewIntFromUint64(totalGas * MinGasPrice)
	return sdk.NewCoins(sdk.NewCoin(BondDenom, minFee))
}

// ---------- Helpers ----------

func init() {
	// Validate that all gas costs are within limits at startup
	for txType, gas := range TransactionGasCosts {
		if gas > TxGasLimit {
			panic(fmt.Sprintf("gas cost for %s (%d) exceeds tx limit (%d)", txType, gas, TxGasLimit))
		}
	}
}
