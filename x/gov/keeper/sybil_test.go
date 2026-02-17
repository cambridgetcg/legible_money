package keeper_test

import (
	"encoding/json"
	"testing"

	"github.com/legible-money/legible/x/gov/keeper"
	"github.com/legible-money/legible/x/gov/types"
)

// ============ Sybil Detection Unit Tests ============

func TestRecordFunding_NewRecord(t *testing.T) {
	k, ctx := setupKeeper(t)

	sender := testAddr("whale__")
	recipient := testAddr("sybil1_")

	k.RecordFunding(ctx, sender, recipient, "1000000", 100)

	sources := k.GetFundingSources(ctx, recipient, 200, 480000)
	if len(sources) != 1 {
		t.Fatalf("expected 1 funding source, got %d", len(sources))
	}
	if sources[0].Sender != sender {
		t.Errorf("sender: got %s, want %s", sources[0].Sender, sender)
	}
	if sources[0].TotalAmount != "1000000" {
		t.Errorf("amount: got %s, want 1000000", sources[0].TotalAmount)
	}
	if sources[0].TransferCount != 1 {
		t.Errorf("count: got %d, want 1", sources[0].TransferCount)
	}
	if sources[0].FirstBlock != 100 {
		t.Errorf("first_block: got %d, want 100", sources[0].FirstBlock)
	}
}

func TestRecordFunding_UpdateExisting(t *testing.T) {
	k, ctx := setupKeeper(t)

	sender := testAddr("whale__")
	recipient := testAddr("sybil1_")

	k.RecordFunding(ctx, sender, recipient, "1000000", 100)
	k.RecordFunding(ctx, sender, recipient, "2000000", 200)

	sources := k.GetFundingSources(ctx, recipient, 300, 480000)
	if len(sources) != 1 {
		t.Fatalf("expected 1 funding source (merged), got %d", len(sources))
	}
	if sources[0].TotalAmount != "3000000" {
		t.Errorf("amount: got %s, want 3000000 (accumulated)", sources[0].TotalAmount)
	}
	if sources[0].TransferCount != 2 {
		t.Errorf("count: got %d, want 2", sources[0].TransferCount)
	}
	if sources[0].FirstBlock != 100 {
		t.Errorf("first_block should be original: got %d, want 100", sources[0].FirstBlock)
	}
	if sources[0].LastBlock != 200 {
		t.Errorf("last_block should be updated: got %d, want 200", sources[0].LastBlock)
	}
}

func TestRecordFunding_MultipleSenders(t *testing.T) {
	k, ctx := setupKeeper(t)

	whale1 := testAddr("whale1_")
	whale2 := testAddr("whale2_")
	recipient := testAddr("sybil1_")

	k.RecordFunding(ctx, whale1, recipient, "1000000", 100)
	k.RecordFunding(ctx, whale2, recipient, "2000000", 100)

	sources := k.GetFundingSources(ctx, recipient, 200, 480000)
	if len(sources) != 2 {
		t.Fatalf("expected 2 funding sources, got %d", len(sources))
	}
}

func TestGetFundingSources_WindowFiltering(t *testing.T) {
	k, ctx := setupKeeper(t)

	sender := testAddr("whale__")
	recipient := testAddr("sybil1_")

	// Record at block 100 with window of 100 blocks
	k.RecordFunding(ctx, sender, recipient, "1000000", 100)

	// Within window (current=150, window=100 → cutoff=50, record at 100 passes)
	sources := k.GetFundingSources(ctx, recipient, 150, 100)
	if len(sources) != 1 {
		t.Errorf("expected 1 source within window, got %d", len(sources))
	}

	// Outside window (current=250, window=100 → cutoff=150, record at 100 fails)
	sources = k.GetFundingSources(ctx, recipient, 250, 100)
	if len(sources) != 0 {
		t.Errorf("expected 0 sources outside window, got %d", len(sources))
	}
}

func TestSybilParams_DefaultAndCustom(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Defaults
	params := k.GetSybilParams(ctx)
	if !params.Enabled {
		t.Error("expected enabled by default")
	}
	if params.CorrelationWindow != 480000 {
		t.Errorf("window: got %d, want 480000", params.CorrelationWindow)
	}
	if params.DecayPerSourceBPS != 2000 {
		t.Errorf("decay: got %d, want 2000", params.DecayPerSourceBPS)
	}
	if params.MinPowerBPS != 1000 {
		t.Errorf("min: got %d, want 1000", params.MinPowerBPS)
	}

	// Custom
	custom := keeper.SybilParams{
		Enabled:           false,
		CorrelationWindow: 100000,
		DecayPerSourceBPS: 3000,
		MinPowerBPS:       500,
	}
	k.SetSybilParams(ctx, custom)
	got := k.GetSybilParams(ctx)
	if got.Enabled {
		t.Error("expected disabled")
	}
	if got.CorrelationWindow != 100000 {
		t.Errorf("window: got %d, want 100000", got.CorrelationWindow)
	}
}

func TestComputeSybilDecayBPS_NoSources(t *testing.T) {
	k, ctx := setupKeeper(t)
	createAndAdvanceToVoting(t, k, keeper.NewMsgServerImpl(k), ctx)

	// Voter with no funding sources → no decay
	decay := k.ComputeSybilDecayBPS(ctx, testAddr("clean__"), "LIP-1")
	if decay != 10000 {
		t.Errorf("expected 10000 (no decay), got %d", decay)
	}
}

func TestComputeSybilDecayBPS_NoExistingVotes(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	createAndAdvanceToVoting(t, k, ms, ctx)

	whale := testAddr("whale__")
	voter := testAddr("voter1_")

	// Fund voter from whale
	k.RecordFunding(ctx, whale, voter, "1000000", 50)

	// First voter on proposal → no one to correlate with → no decay
	decay := k.ComputeSybilDecayBPS(ctx, voter, "LIP-1")
	if decay != 10000 {
		t.Errorf("expected 10000 (first voter), got %d", decay)
	}
}

func TestGetAllFundingRecords(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.RecordFunding(ctx, testAddr("sender1"), testAddr("recip1_"), "100", 10)
	k.RecordFunding(ctx, testAddr("sender2"), testAddr("recip2_"), "200", 20)
	k.RecordFunding(ctx, testAddr("sender1"), testAddr("recip2_"), "300", 30)

	all := k.GetAllFundingRecords(ctx)
	if len(all) != 3 {
		t.Errorf("expected 3 records, got %d", len(all))
	}
}

// ============ OpenClaw Adversarial Probes: Sybil Detection ============

// OC-GOV-13: Sybil Split Attack Detection — A whale funds 5 accounts from the
// same source. When they vote on the same proposal, accounts 2-5 get decayed.
func TestOC_GOV13_SybilSplitAttackDetection(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	createAndAdvanceToVoting(t, k, ms, ctx)

	whale := testAddr("whale__")

	// Whale funds 5 sybil accounts
	sybils := make([]string, 5)
	for i := 0; i < 5; i++ {
		sybils[i] = testAddr("sybil" + string(rune('a'+i)))
		k.RecordFunding(ctx, whale, sybils[i], "10000000", 50)
	}

	// First sybil votes — no decay (first voter with these sources)
	resp1, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: sybils[0], ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "10000",
	})
	if err != nil {
		t.Fatalf("OC-GOV-13: first sybil vote failed: %v", err)
	}
	// sqrt(10000) = 100, no decay → 100
	if resp1.EffectiveVotingPower != "100" {
		t.Errorf("OC-GOV-13: first sybil should have full power, got %s (want 100)", resp1.EffectiveVotingPower)
	}

	// Second sybil votes — should get decayed (shares 1 source with first sybil)
	resp2, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: sybils[1], ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "10000",
	})
	if err != nil {
		t.Fatalf("OC-GOV-13: second sybil vote failed: %v", err)
	}
	// sqrt(10000) = 100, decay = 10000 - 1*2000 = 8000, effective = 100 * 8000 / 10000 = 80
	if resp2.EffectiveVotingPower != "80" {
		t.Errorf("OC-GOV-13: second sybil should be decayed to 80, got %s — sybil attack not detected", resp2.EffectiveVotingPower)
	}

	// Fifth sybil — shares 1 source with all 4 prior voters, maxShared still 1
	resp5, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: sybils[4], ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "10000",
	})
	if err != nil {
		t.Fatalf("OC-GOV-13: fifth sybil vote failed: %v", err)
	}
	// maxShared = 1 (each shares the whale as single source)
	if resp5.EffectiveVotingPower != "80" {
		t.Errorf("OC-GOV-13: fifth sybil should be decayed to 80, got %s", resp5.EffectiveVotingPower)
	}
}

// OC-GOV-14: No Decay for Independent Voters — Three voters funded by different
// sources should get zero decay.
func TestOC_GOV14_NoDecayForIndependentVoters(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	createAndAdvanceToVoting(t, k, ms, ctx)

	// Three independent funding sources
	sources := []string{testAddr("src_a__"), testAddr("src_b__"), testAddr("src_c__")}
	voters := []string{testAddr("indep_a"), testAddr("indep_b"), testAddr("indep_c")}

	for i := 0; i < 3; i++ {
		k.RecordFunding(ctx, sources[i], voters[i], "10000000", 50)
	}

	// All voters should get full power (no shared sources)
	for i, voter := range voters {
		resp, err := ms.CastVote(ctx, &types.MsgCastVote{
			Voter: voter, ProposalId: "LIP-1",
			Option: types.VoteOption_VOTE_OPTION_YES, Stake: "10000",
		})
		if err != nil {
			t.Fatalf("OC-GOV-14: voter %d failed: %v", i, err)
		}
		if resp.EffectiveVotingPower != "100" {
			t.Errorf("OC-GOV-14: independent voter %d should have full power (100), got %s — false positive decay", i, resp.EffectiveVotingPower)
		}
	}
}

// OC-GOV-15: Decay Window Boundary — Transfers inside the window cause decay;
// transfers outside the window do not.
func TestOC_GOV15_DecayWindowBoundary(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Set a short correlation window for testing
	k.SetSybilParams(ctx, keeper.SybilParams{
		Enabled:           true,
		CorrelationWindow: 100, // 100 blocks
		DecayPerSourceBPS: 2000,
		MinPowerBPS:       1000,
	})

	createAndAdvanceToVoting(t, k, ms, ctx)

	whale := testAddr("whale__")
	voter1 := testAddr("v1_wnd_")
	voter2 := testAddr("v2_wnd_")

	// Fund voter1 at block 50 (WITHIN window since ctx is block 100, window=100, cutoff=0)
	k.RecordFunding(ctx, whale, voter1, "10000000", 50)

	// Fund voter2 at block 50 — same whale, same time
	k.RecordFunding(ctx, whale, voter2, "10000000", 50)

	// Vote at block 100 — both within window
	_, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: voter1, ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "10000",
	})
	if err != nil {
		t.Fatalf("voter1 failed: %v", err)
	}

	resp2, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: voter2, ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "10000",
	})
	if err != nil {
		t.Fatalf("voter2 failed: %v", err)
	}
	if resp2.EffectiveVotingPower != "80" {
		t.Errorf("OC-GOV-15: voter2 within window should be decayed to 80, got %s", resp2.EffectiveVotingPower)
	}

	// Now test outside window: create new proposal and vote at block 300
	// The funding at block 50 should be outside the window (300-100=200 > 50)
	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice2_"),
		Title:        "Window Test 2",
		Description:  "Test",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})
	lip2, _ := k.GetLIP(ctx, "LIP-2")
	lip2.Category = string(types.CategoryProcess)
	k.SetLIP(ctx, lip2)
	ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Staker: testAddr("bob2___"), LipId: "LIP-2", Amount: "200000000",
	})

	params := k.GetParams(ctx)
	params.ReviewPeriodBlocks = 0
	k.SetParams(ctx, params)

	if _, err := ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{Authority: testAddr("alice2_"), LipId: "LIP-2"}); err != nil {
		t.Fatalf("advance review->last_call failed: %v", err)
	}
	if _, err := ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{Authority: testAddr("alice2_"), LipId: "LIP-2"}); err != nil {
		t.Fatalf("advance last_call->voting failed: %v", err)
	}

	voter3 := testAddr("v3_wnd_")
	voter4 := testAddr("v4_wnd_")
	k.RecordFunding(ctx, whale, voter3, "10000000", 50) // old funding
	k.RecordFunding(ctx, whale, voter4, "10000000", 50) // old funding

	// Vote at block 300 — funding at block 50 is outside window of 100
	ctx300 := ctx.WithBlockHeight(300)
	_, err = ms.CastVote(ctx300, &types.MsgCastVote{
		Voter: voter3, ProposalId: "LIP-2",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "10000",
	})
	if err != nil {
		t.Fatalf("voter3 failed: %v", err)
	}

	resp4, err := ms.CastVote(ctx300, &types.MsgCastVote{
		Voter: voter4, ProposalId: "LIP-2",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "10000",
	})
	if err != nil {
		t.Fatalf("voter4 failed: %v", err)
	}
	// Outside window → no shared sources → no decay
	if resp4.EffectiveVotingPower != "100" {
		t.Errorf("OC-GOV-15: voter4 outside window should have full power (100), got %s — window boundary wrong", resp4.EffectiveVotingPower)
	}
}

// OC-GOV-16: Decay Floor Enforcement — Many shared sources should clamp at
// MinPowerBPS, never reaching zero.
func TestOC_GOV16_DecayFloorEnforcement(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	createAndAdvanceToVoting(t, k, ms, ctx)

	voter1 := testAddr("floor1_")
	voter2 := testAddr("floor2_")

	// Fund both voters from 10 different whales (more than 5 needed to hit floor)
	for i := 0; i < 10; i++ {
		whale := testAddr("wh" + string(rune('a'+i)) + "____")
		k.RecordFunding(ctx, whale, voter1, "1000000", 50)
		k.RecordFunding(ctx, whale, voter2, "1000000", 50)
	}

	// First voter — no decay
	_, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: voter1, ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "10000",
	})
	if err != nil {
		t.Fatalf("voter1 failed: %v", err)
	}

	// Second voter — 10 shared sources × 2000 BPS = 20000, but clamped at MinPowerBPS=1000
	resp2, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: voter2, ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "10000",
	})
	if err != nil {
		t.Fatalf("voter2 failed: %v", err)
	}
	// sqrt(10000) = 100, decay clamped at 1000 BPS (10%), effective = 100 * 1000 / 10000 = 10
	if resp2.EffectiveVotingPower != "10" {
		t.Errorf("OC-GOV-16: voter2 should be clamped at floor (10), got %s — floor not enforced", resp2.EffectiveVotingPower)
	}

	// Verify power never goes to zero even with massive correlation
	if resp2.EffectiveVotingPower == "0" {
		t.Error("OC-GOV-16: voting power should never be zero — would silence legitimate voters")
	}
}

// OC-GOV-17: Feature Flag Toggle — When SybilParams.Enabled=false, no decay
// should be applied even with strong correlation.
func TestOC_GOV17_FeatureFlagToggle(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Disable sybil detection
	k.SetSybilParams(ctx, keeper.SybilParams{
		Enabled:           false,
		CorrelationWindow: 480000,
		DecayPerSourceBPS: 2000,
		MinPowerBPS:       1000,
	})

	createAndAdvanceToVoting(t, k, ms, ctx)

	whale := testAddr("whale__")
	voter1 := testAddr("toggl1_")
	voter2 := testAddr("toggl2_")

	// Fund both from same whale
	k.RecordFunding(ctx, whale, voter1, "10000000", 50)
	k.RecordFunding(ctx, whale, voter2, "10000000", 50)

	// Both should get full power since detection is disabled
	resp1, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: voter1, ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "10000",
	})
	if err != nil {
		t.Fatalf("voter1 failed: %v", err)
	}
	if resp1.EffectiveVotingPower != "100" {
		t.Errorf("OC-GOV-17: voter1 with flag off should have full power, got %s", resp1.EffectiveVotingPower)
	}

	resp2, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: voter2, ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "10000",
	})
	if err != nil {
		t.Fatalf("voter2 failed: %v", err)
	}
	if resp2.EffectiveVotingPower != "100" {
		t.Errorf("OC-GOV-17: voter2 with flag off should have full power (100), got %s — kill switch broken", resp2.EffectiveVotingPower)
	}
}

// OC-GOV-18: Genesis Roundtrip for Funding Records — Export and import sybil
// genesis state, verifying FundingRecords and SybilParams survive.
func TestOC_GOV18_GenesisRoundtripFundingRecords(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set custom sybil params
	customParams := keeper.SybilParams{
		Enabled:           true,
		CorrelationWindow: 100000,
		DecayPerSourceBPS: 3000,
		MinPowerBPS:       500,
	}
	k.SetSybilParams(ctx, customParams)

	// Create funding records
	k.RecordFunding(ctx, testAddr("sender1"), testAddr("recip1_"), "1000000", 100)
	k.RecordFunding(ctx, testAddr("sender2"), testAddr("recip1_"), "2000000", 150)
	k.RecordFunding(ctx, testAddr("sender1"), testAddr("recip2_"), "3000000", 200)

	// Export sybil genesis
	exported := k.ExportSybilGenesis(ctx)

	// Verify it's valid JSON
	var sg keeper.SybilGenesis
	if err := json.Unmarshal(exported, &sg); err != nil {
		t.Fatalf("OC-GOV-18: exported sybil genesis is not valid JSON: %v", err)
	}
	if len(sg.FundingRecords) != 3 {
		t.Errorf("OC-GOV-18: expected 3 funding records in export, got %d", len(sg.FundingRecords))
	}
	if sg.Params.DecayPerSourceBPS != 3000 {
		t.Errorf("OC-GOV-18: exported params.decay should be 3000, got %d", sg.Params.DecayPerSourceBPS)
	}

	// Import into fresh keeper
	k2, ctx2 := setupKeeper(t)
	k2.InitSybilGenesis(ctx2, exported)

	// Verify params survived
	gotParams := k2.GetSybilParams(ctx2)
	if gotParams.CorrelationWindow != 100000 {
		t.Errorf("OC-GOV-18: window: got %d, want 100000", gotParams.CorrelationWindow)
	}
	if gotParams.DecayPerSourceBPS != 3000 {
		t.Errorf("OC-GOV-18: decay: got %d, want 3000", gotParams.DecayPerSourceBPS)
	}
	if gotParams.MinPowerBPS != 500 {
		t.Errorf("OC-GOV-18: min: got %d, want 500", gotParams.MinPowerBPS)
	}

	// Verify funding records survived
	allRecords := k2.GetAllFundingRecords(ctx2)
	if len(allRecords) != 3 {
		t.Fatalf("OC-GOV-18: expected 3 funding records after import, got %d — records lost on genesis restart", len(allRecords))
	}

	// Verify specific record details
	sources := k2.GetFundingSources(ctx2, testAddr("recip1_"), 300, 480000)
	if len(sources) != 2 {
		t.Errorf("OC-GOV-18: expected 2 sources for recip1, got %d", len(sources))
	}

	sources2 := k2.GetFundingSources(ctx2, testAddr("recip2_"), 300, 480000)
	if len(sources2) != 1 {
		t.Errorf("OC-GOV-18: expected 1 source for recip2, got %d", len(sources2))
	}
}

// ============ Additional Edge Case Tests ============

func TestComputeSybilDecayBPS_MultipleSharedSources(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	createAndAdvanceToVoting(t, k, ms, ctx)

	voter1 := testAddr("multi1_")
	voter2 := testAddr("multi2_")

	// Fund both voters from 3 common sources
	whales := []string{testAddr("wha____"), testAddr("whb____"), testAddr("whc____")}
	for _, whale := range whales {
		k.RecordFunding(ctx, whale, voter1, "1000000", 50)
		k.RecordFunding(ctx, whale, voter2, "1000000", 50)
	}

	// First voter — full power
	_, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: voter1, ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "10000",
	})
	if err != nil {
		t.Fatalf("voter1 failed: %v", err)
	}

	// Second voter — 3 shared sources × 2000 = 6000 decay → 4000 BPS remaining
	// sqrt(10000) = 100, effective = 100 * 4000 / 10000 = 40
	resp2, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: voter2, ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "10000",
	})
	if err != nil {
		t.Fatalf("voter2 failed: %v", err)
	}
	if resp2.EffectiveVotingPower != "40" {
		t.Errorf("expected 40 (3 shared sources × 20%% decay each = 60%% decay), got %s", resp2.EffectiveVotingPower)
	}
}

func TestComputeSybilDecayBPS_MinimumOnePower(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Set floor very low
	k.SetSybilParams(ctx, keeper.SybilParams{
		Enabled:           true,
		CorrelationWindow: 480000,
		DecayPerSourceBPS: 9000, // 90% per source — extreme
		MinPowerBPS:       100,  // 1% floor
	})

	createAndAdvanceToVoting(t, k, ms, ctx)

	whale := testAddr("whale__")
	voter1 := testAddr("min_v1_")
	voter2 := testAddr("min_v2_")

	k.RecordFunding(ctx, whale, voter1, "1000000", 50)
	k.RecordFunding(ctx, whale, voter2, "1000000", 50)

	_, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: voter1, ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "100",
	})
	if err != nil {
		t.Fatalf("voter1 failed: %v", err)
	}

	// sqrt(100) = 10, decay clamped at 100 BPS (1%), effective = 10 * 100 / 10000 = 0
	// But minimum is 1 when stake > 0
	resp2, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: voter2, ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "100",
	})
	if err != nil {
		t.Fatalf("voter2 failed: %v", err)
	}
	if resp2.EffectiveVotingPower == "0" {
		t.Error("voting power should never be zero when stake is positive — minimum 1 guarantee broken")
	}
	if resp2.EffectiveVotingPower != "1" {
		t.Errorf("expected minimum power of 1, got %s", resp2.EffectiveVotingPower)
	}
}

func TestInitSybilGenesis_NilInput(t *testing.T) {
	k, ctx := setupKeeper(t)

	// nil input should set defaults
	k.InitSybilGenesis(ctx, nil)

	params := k.GetSybilParams(ctx)
	if !params.Enabled {
		t.Error("expected defaults with nil input")
	}
	if params.CorrelationWindow != 480000 {
		t.Errorf("expected default window, got %d", params.CorrelationWindow)
	}
}

func TestInitSybilGenesis_CorruptInput(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Corrupt JSON should fall back to defaults
	k.InitSybilGenesis(ctx, []byte("not-json"))

	params := k.GetSybilParams(ctx)
	if !params.Enabled {
		t.Error("expected defaults with corrupt input")
	}
}
