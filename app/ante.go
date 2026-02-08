package app

import (
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"
)

// NewAnteHandler returns an AnteHandler that:
// 1. Runs standard Cosmos SDK ante decorators (sequence, signature, etc.)
// 2. Applies LGM-specific gas cost validation
// 3. Routes fees: 22% to research fund, 78% to validators
func NewAnteHandler(app *LegibleApp) sdk.AnteHandler {
	anteHandler, err := ante.NewAnteHandler(
		ante.HandlerOptions{
			AccountKeeper:   app.AccountKeeper,
			BankKeeper:      app.BankKeeper,
			FeegrantKeeper:  app.FeeGrantKeeper,
			SignModeHandler: app.txConfig.SignModeHandler(),
			SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
		},
	)
	if err != nil {
		panic(err)
	}

	return sdk.ChainAnteDecorators(
		ibcante.NewRedundantRelayDecorator(app.IBCKeeper),
		NewLGMGasDecorator(),
		NewFeeRouterDecorator(app.BankKeeper),
		anteHandlerDecorator{inner: anteHandler},
	)
}

// anteHandlerDecorator wraps an sdk.AnteHandler to satisfy the AnteDecorator
// interface so it can be chained with other decorators.
type anteHandlerDecorator struct {
	inner sdk.AnteHandler
}

func (d anteHandlerDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	return d.inner(ctx, tx, simulate)
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
	"/legible.auth.v1.MsgRotateKey":     TransactionGasCosts["rotate_key"],
	"/legible.auth.v1.MsgCreateSession": TransactionGasCosts["create_session"],
	"/legible.auth.v1.MsgRevokeSession": TransactionGasCosts["revoke_session"],

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

	// Tree of Life
	"/legible.tree.v1.MsgCreateProject":      TransactionGasCosts["create_project"],
	"/legible.tree.v1.MsgProposeProject":     TransactionGasCosts["propose_project"],
	"/legible.tree.v1.MsgStartDevelopment":   TransactionGasCosts["start_development"],
	"/legible.tree.v1.MsgCompleteProject":    TransactionGasCosts["complete_project"],
	"/legible.tree.v1.MsgAddTask":            TransactionGasCosts["add_task"],
	"/legible.tree.v1.MsgAssignTask":         TransactionGasCosts["assign_task"],
	"/legible.tree.v1.MsgSubmitDeliverable":  TransactionGasCosts["submit_deliverable"],
	"/legible.tree.v1.MsgApproveDeliverable": TransactionGasCosts["approve_deliverable"],
	"/legible.tree.v1.MsgDeployService":      TransactionGasCosts["deploy_service"],
	"/legible.tree.v1.MsgCallService":        TransactionGasCosts["call_service"],
	"/legible.tree.v1.MsgBeginSeeding":       TransactionGasCosts["begin_seeding"],
	"/legible.tree.v1.MsgDetectOpportunity":  TransactionGasCosts["detect_opportunity"],
	"/legible.tree.v1.MsgClaimOpportunity":   TransactionGasCosts["claim_opportunity"],

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
	// Fee routing happens post-execution in EndBlocker via distribution module.
	// Here we just validate and log the intended split.
	//
	// The actual split is:
	//   - 22% (ResearchTaxBPS) → research_fund module account
	//   - 78% (ValidatorFeeBPS) → fee_collector (standard Cosmos distribution)
	//
	// Phase 2: Implement BeginBlock fee sweep from fee_collector to research_fund.
	// For now, all fees go to fee_collector and distribution handles validator rewards.

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return next(ctx, tx, simulate)
	}

	fee := feeTx.GetFee()
	if fee.IsZero() {
		return next(ctx, tx, simulate)
	}

	// Log the fee split for observability
	ctx.Logger().Debug("LGM fee routing",
		"total_fee", fee.String(),
		"research_share_bps", ResearchTaxBPS,
		"validator_share_bps", ValidatorFeeBPS,
	)

	return next(ctx, tx, simulate)
}

// RouteFees splits collected fees between research fund and validators.
// Called in BeginBlock to sweep accumulated fees.
func RouteFees(ctx sdk.Context, bk bankkeeper.Keeper) error {
	feeCollectorAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	balance := bk.GetAllBalances(ctx, feeCollectorAddr)

	if balance.IsZero() {
		return nil
	}

	for _, coin := range balance {
		if coin.Denom != BondDenom {
			continue
		}

		totalAmount := coin.Amount
		researchAmount := totalAmount.MulRaw(int64(ResearchTaxBPS)).QuoRaw(10000)

		if researchAmount.IsPositive() {
			researchCoins := sdk.NewCoins(sdk.NewCoin(BondDenom, researchAmount))
			err := bk.SendCoinsFromModuleToModule(ctx, authtypes.FeeCollectorName, ResearchFundName, researchCoins)
			if err != nil {
				// Research fund module account may not exist yet — log and continue.
				ctx.Logger().Warn("failed to route fees to research fund",
					"amount", researchAmount.String(),
					"err", err,
				)
			}
		}
	}

	return nil
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
