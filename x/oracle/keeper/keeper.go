// Package keeper implements the oracle module's state management.
package keeper

import (
	"encoding/json"
	"fmt"
	"math/bits"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/oracle/types"
)

// Keeper manages x/oracle state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService
	bankKeeper   types.BankKeeper
	authority    string
}

// NewKeeper creates a new oracle Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		bankKeeper:   bankKeeper,
		authority:    authority,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

// ── Prediction CRUD ──────────────────────────────────────────────────────────

func predictionKey(id string) []byte {
	return append(types.PredictionKeyPrefix, []byte(id)...)
}

func calibrationKey(address string) []byte {
	return append(types.ScoreboardKeyPrefix, []byte(address)...)
}

func trainingKey(id string) []byte {
	return append(types.ResolutionKeyPrefix, []byte(id)...)
}

// SetPrediction stores a prediction.
func (k Keeper) SetPrediction(ctx sdk.Context, p types.Prediction) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, _ := json.Marshal(p)
	_ = kvStore.Set(predictionKey(p.PredictionId), bz)
}

// GetPrediction retrieves a prediction by ID.
func (k Keeper) GetPrediction(ctx sdk.Context, id string) (types.Prediction, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(predictionKey(id))
	if err != nil || bz == nil {
		return types.Prediction{}, false
	}
	var p types.Prediction
	if err := json.Unmarshal(bz, &p); err != nil {
		return types.Prediction{}, false
	}
	return p, true
}

// IteratePredictions iterates over all predictions.
func (k Keeper) IteratePredictions(ctx sdk.Context, cb func(types.Prediction) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	end := append([]byte{}, types.PredictionKeyPrefix...)
	end[len(end)-1]++
	iter, err := kvStore.Iterator(types.PredictionKeyPrefix, end)
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var p types.Prediction
		if err := json.Unmarshal(iter.Value(), &p); err != nil {
			continue
		}
		if cb(p) {
			break
		}
	}
}

// ── Calibration CRUD ─────────────────────────────────────────────────────────

// GetCalibration returns a predictor's calibration record.
func (k Keeper) GetCalibration(ctx sdk.Context, address string) (types.CalibrationSummary, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(calibrationKey(address))
	if err != nil || bz == nil {
		return types.CalibrationSummary{}, false
	}
	var c types.CalibrationSummary
	if err := json.Unmarshal(bz, &c); err != nil {
		return types.CalibrationSummary{}, false
	}
	return c, true
}

// SetCalibration updates a predictor's calibration record.
func (k Keeper) SetCalibration(ctx sdk.Context, c types.CalibrationSummary) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, _ := json.Marshal(c)
	_ = kvStore.Set(calibrationKey(c.Address), bz)
}

// ── Prediction Submission ────────────────────────────────────────────────────

// SubmitPrediction validates and stores a new prediction, locking stake.
func (k Keeper) SubmitPrediction(ctx sdk.Context, msg types.MsgSubmitPrediction) (types.Prediction, error) {
	height := uint64(ctx.BlockHeight())
	params := k.GetParams(ctx)

	if msg.Probability > 1_000_000 {
		return types.Prediction{}, fmt.Errorf("probability must be <= 1,000,000 bps")
	}
	if msg.Probability == 0 || msg.Probability == 1_000_000 {
		return types.Prediction{}, fmt.Errorf("probability must be strictly between 0 and 1 (avoid 0%% or 100%% predictions)")
	}

	stake := sdk.NewInt64Coin("ulgm", int64(msg.StakeAmount))
	if uint64(stake.Amount.Int64()) < params.MinPredictionStake {
		return types.Prediction{}, fmt.Errorf("stake %d below minimum %d", msg.StakeAmount, params.MinPredictionStake)
	}

	horizonBlocks := msg.ResolutionBlock - height
	if horizonBlocks < params.MinHorizonBlocks {
		return types.Prediction{}, fmt.Errorf("horizon too short: %d blocks (min %d)", horizonBlocks, params.MinHorizonBlocks)
	}

	// Lock stake
	submitterAddr, err := sdk.AccAddressFromBech32(msg.Submitter)
	if err != nil {
		return types.Prediction{}, fmt.Errorf("invalid submitter: %w", err)
	}
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, submitterAddr, types.ModuleName,
		sdk.NewCoins(stake)); err != nil {
		return types.Prediction{}, fmt.Errorf("stake lock failed: %w", err)
	}

	// Generate deterministic ID
	id := fmt.Sprintf("pred-%s-%d", msg.Submitter[:8], height)

	p := types.Prediction{
		PredictionId:       id,
		Submitter:          msg.Submitter,
		Claim:              msg.Claim,
		Domain:             msg.Domain,
		Probability:        msg.Probability,
		Stake:              msg.StakeAmount,
		ResolutionType: msg.ResolutionType,
		ResolutionRef:      msg.ResolutionRef,
		ResolutionCriteria: msg.ResolutionCriteria,
		ResolutionBlock:    msg.ResolutionBlock,
		SubmittedAt:        height,
		Status:             types.PredictionActive,
	}

	k.SetPrediction(ctx, p)
	k.Logger(ctx).Info("prediction submitted", "id", id, "submitter", msg.Submitter[:16])
	return p, nil
}

// ── Resolution ───────────────────────────────────────────────────────────────

// ResolvePrediction marks a prediction resolved and updates calibration scores.
func (k Keeper) ResolvePrediction(ctx sdk.Context, id string, outcome bool) error {
	p, found := k.GetPrediction(ctx, id)
	if !found {
		return fmt.Errorf("prediction not found: %s", id)
	}
	if p.Status != types.PredictionActive {
		return fmt.Errorf("prediction %s is not active (status: %s)", id, p.Status)
	}

	height := uint64(ctx.BlockHeight())
	p.ResolvedAt = height
	p.Outcome = outcome
	p.Status = types.PredictionResolved

	// Compute Brier score for this prediction
	horizonDays := (p.ResolvedAt - p.SubmittedAt) * 2521 / 86_400_000 // blocks to days (~2521ms)
	w := computeBrierWeightInt(horizonDays)
	pred := p.Probability             // in bps [0, 1_000_000]
	act := uint64(0)
	if outcome {
		act = 1_000_000
	}
	var diff int64
	if pred > act {
		diff = int64(pred - act)
	} else {
		diff = int64(act - pred)
	}
	// (p-o)^2 in bps^2, then scale back to bps
	// diff is in bps (0-1_000_000), squared is 0-1e12, divide by 1e6 to get bps result
	scoreNumer := uint64(diff*diff) / 1_000_000 // result in bps
	score := w * scoreNumer / 1_000             // w is × 1000 scaled
	p.BrierScore = score

	k.SetPrediction(ctx, p)
	k.updateCalibration(ctx, p, horizonDays)
	k.storeTrainingExample(ctx, p, horizonDays)
	k.redistributeStake(ctx, p, outcome)

	k.Logger(ctx).Info("prediction resolved",
		"id", id, "outcome", outcome, "brier_score", score)
	return nil
}

// computeBrierWeightInt returns floor(log2(1+horizonDays)) * 1000.
func computeBrierWeightInt(horizonDays uint64) uint64 {
	if horizonDays == 0 {
		return 1000
	}
	b := uint64(bits.Len64(horizonDays+1) - 1) // floor(log2(horizonDays+1))
	return b * 1000
}

// updateCalibration updates a predictor's per-tier calibration record.
func (k Keeper) updateCalibration(ctx sdk.Context, p types.Prediction, horizonDays uint64) {
	c, found := k.GetCalibration(ctx, p.Submitter)
	if !found {
		c = types.CalibrationSummary{
			Address:        p.Submitter,
			ActiveBonusBps: types.NoBonusBps,
		}
	}

	tier := types.ClassifyHorizon(uint64(float64(horizonDays) / 2521 * 86_400))
	score := p.BrierScore

	switch tier {
	case types.HorizonFlash:
		c.FlashHwbs = (c.FlashHwbs*c.FlashCount + score) / (c.FlashCount + 1)
		c.FlashCount++
	case types.HorizonTactical:
		c.TacticalHwbs = (c.TacticalHwbs*c.TacticalCount + score) / (c.TacticalCount + 1)
		c.TacticalCount++
	case types.HorizonStrategic:
		c.StrategicHwbs = (c.StrategicHwbs*c.StrategicCount + score) / (c.StrategicCount + 1)
		c.StrategicCount++
	case types.HorizonEpochal:
		c.EpochalHwbs = (c.EpochalHwbs*c.EpochalCount + score) / (c.EpochalCount + 1)
		c.EpochalCount++
	}

	c.LastUpdated = uint64(ctx.BlockHeight())
	k.SetCalibration(ctx, c)
}

// storeTrainingExample writes a Kwok training corpus entry.
func (k Keeper) storeTrainingExample(ctx sdk.Context, p types.Prediction, horizonDays uint64) {
	example := types.TrainingExample{
		PredictionId:  p.PredictionId,
		Submitter:     p.Submitter,
		PredictedProb: p.Probability,
		ActualOutcome: p.Outcome,
		HorizonTier:   types.ClassifyHorizon(uint64(float64(horizonDays) / 2521 * 86_400)),
		Domain:        p.Domain,
		ResolvedBlock: p.ResolvedAt,
	}
	// Compute delta (positive = overcautious, negative = overconfident)
	pred := int64(p.Probability)
	act := int64(0)
	if p.Outcome {
		act = 1_000_000
	}
	example.Delta = pred - act

	bz, _ := json.Marshal(example)
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Set(trainingKey(p.PredictionId), bz)
}

// redistributeStake transfers stake from wrong predictors to correct ones.
// Simple version: wrong predictions' stake goes to protocol pool for distribution.
func (k Keeper) redistributeStake(ctx sdk.Context, p types.Prediction, outcome bool) {
	// If prediction was correct (outcome matches probability direction):
	// probability > 500k and outcome=true → correct
	// probability < 500k and outcome=false → correct
	submitterAddr, err := sdk.AccAddressFromBech32(p.Submitter)
	if err != nil {
		return
	}

	coin := sdk.NewInt64Coin("ulgm", int64(p.Stake))

	correct := (p.Probability > 500_000 && outcome) || (p.Probability < 500_000 && !outcome)
	if correct {
		// Return stake to submitter
		_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, submitterAddr,
			sdk.NewCoins(coin))
	}
	// Wrong predictions: stake stays in module for redistribution (future: proportional to calibration)
}

// ── Params ───────────────────────────────────────────────────────────────────

// GetParams returns the oracle module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.DefaultParams()
}
