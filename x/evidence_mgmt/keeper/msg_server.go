package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/evidence_mgmt/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// SubmitEvidence handles MsgSubmitEvidence.
func (k msgServer) SubmitEvidence(goCtx context.Context, msg *types.MsgSubmitEvidence) (*types.MsgSubmitEvidenceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	height := uint64(ctx.BlockHeight())

	evidenceId := GenerateEvidenceID(msg.Submitter, msg.ClaimId, msg.ContentHash, height)

	evidence := types.Evidence{
		Id:           evidenceId,
		ClaimId:      msg.ClaimId,
		Submitter:    msg.Submitter,
		EvidenceType: msg.EvidenceType,
		ContentHash:  msg.ContentHash,
		Description:  msg.Description,
		SourceURI:    msg.SourceURI,
		SubmittedAt:  height,
	}

	if err := k.Keeper.SubmitEvidence(ctx, evidence); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"evidence_submitted",
			sdk.NewAttribute("evidence_id", evidenceId),
			sdk.NewAttribute("claim_id", msg.ClaimId),
			sdk.NewAttribute("submitter", msg.Submitter),
			sdk.NewAttribute("type", string(msg.EvidenceType)),
			sdk.NewAttribute("height", fmt.Sprintf("%d", height)),
		),
	)

	return &types.MsgSubmitEvidenceResponse{
		EvidenceId: evidenceId,
	}, nil
}

// AuditEvidence handles MsgAuditEvidence.
func (k msgServer) AuditEvidence(goCtx context.Context, msg *types.MsgAuditEvidence) (*types.MsgAuditEvidenceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	height := uint64(ctx.BlockHeight())

	auditId := GenerateAuditID(msg.Auditor, msg.EvidenceId, height)

	audit := types.EvidenceAudit{
		Id:          auditId,
		EvidenceId:  msg.EvidenceId,
		Auditor:     msg.Auditor,
		Stake:       msg.Stake,
		Score:       msg.Score,
		Rationale:   msg.Rationale,
		SubmittedAt: height,
	}

	if err := k.Keeper.SubmitAudit(ctx, audit); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"audit_submitted",
			sdk.NewAttribute("audit_id", auditId),
			sdk.NewAttribute("evidence_id", msg.EvidenceId),
			sdk.NewAttribute("auditor", msg.Auditor),
			sdk.NewAttribute("score", fmt.Sprintf("%d", msg.Score)),
			sdk.NewAttribute("stake", msg.Stake),
		),
	)

	return &types.MsgAuditEvidenceResponse{
		AuditId: auditId,
	}, nil
}

// RegisterOracle handles MsgRegisterOracle.
func (k msgServer) RegisterOracle(goCtx context.Context, msg *types.MsgRegisterOracle) (*types.MsgRegisterOracleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	height := uint64(ctx.BlockHeight())

	oracleId := GenerateOracleID(msg.Operator, msg.Name, height)

	oracle := types.Oracle{
		Id:           oracleId,
		Operator:     msg.Operator,
		Name:         msg.Name,
		Description:  msg.Description,
		DataTypes:    msg.DataTypes,
		Stake:        msg.Stake,
		RegisteredAt: height,
	}

	if err := k.Keeper.RegisterOracle(ctx, oracle); err != nil {
		return nil, err
	}

	return &types.MsgRegisterOracleResponse{
		OracleId: oracleId,
	}, nil
}

// SubmitOracleData handles MsgSubmitOracleData — oracle submits evidence for a claim.
func (k msgServer) SubmitOracleData(goCtx context.Context, msg *types.MsgSubmitOracleData) (*types.MsgSubmitOracleDataResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	height := uint64(ctx.BlockHeight())

	// Verify oracle exists and is active
	oracle, found := k.Keeper.GetOracle(ctx, msg.OracleId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrOracleNotFound, msg.OracleId)
	}
	if !oracle.IsActive {
		return nil, fmt.Errorf("%w: oracle %s is not active", types.ErrOracleInactive, msg.OracleId)
	}

	// Verify the oracle supports the claimed data type
	typeSupported := false
	for _, dt := range oracle.DataTypes {
		if dt == msg.DataType {
			typeSupported = true
			break
		}
	}
	if !typeSupported {
		return nil, fmt.Errorf("%w: oracle %s does not support data type %s",
			types.ErrInvalidEvidence, msg.OracleId, msg.DataType)
	}

	// Create evidence from oracle data
	evidenceId := GenerateEvidenceID(oracle.Operator, msg.ClaimId, msg.ContentHash, height)

	evidence := types.Evidence{
		Id:           evidenceId,
		ClaimId:      msg.ClaimId,
		Submitter:    oracle.Operator,
		EvidenceType: msg.DataType,
		ContentHash:  msg.ContentHash,
		Description:  fmt.Sprintf("Oracle data from %s (%s)", oracle.Name, msg.OracleId),
		SourceURI:    fmt.Sprintf("oracle://%s/%s", msg.OracleId, msg.ContentHash),
		SubmittedAt:  height,
	}

	if err := k.Keeper.SubmitEvidence(ctx, evidence); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_data_submitted",
			sdk.NewAttribute("oracle_id", msg.OracleId),
			sdk.NewAttribute("evidence_id", evidenceId),
			sdk.NewAttribute("claim_id", msg.ClaimId),
			sdk.NewAttribute("data_type", string(msg.DataType)),
		),
	)

	return &types.MsgSubmitOracleDataResponse{
		EvidenceId: evidenceId,
	}, nil
}

// UpdateParams handles MsgUpdateParams.
func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	k.SetParams(ctx, msg.Params)
	return &types.MsgUpdateParamsResponse{}, nil
}
