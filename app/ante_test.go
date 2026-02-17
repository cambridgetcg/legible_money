package app

import (
	"testing"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ---------- lookupMsgGas Tests ----------

func TestLookupMsgGas_KnownTypes(t *testing.T) {
	tests := []struct {
		msgTypeURL string
		wantGas    uint64
	}{
		{"/cosmos.bank.v1beta1.MsgSend", TransactionGasCosts["transfer"]},
		{"/cosmos.bank.v1beta1.MsgMultiSend", TransactionGasCosts["transfer"] * 2},
		{"/legible.knowledge.v1.MsgSubmitClaim", TransactionGasCosts["claim_submit"]},
		{"/legible.staking.v1.MsgRegisterValidator", TransactionGasCosts["register_validator"]},
		{"/legible.bvm.v1.MsgDeployContract", TransactionGasCosts["deploy_contract"]},
		{"/legible.tree.v1.MsgCreateProject", TransactionGasCosts["create_project"]},
		{"/legible.tree.v1.MsgPauseProject", TransactionGasCosts["pause_project"]},
		{"/legible.tree.v1.MsgSubscribeService", TransactionGasCosts["subscribe_service"]},
		{"/legible.tree.v1.MsgSetAvailability", TransactionGasCosts["set_availability"]},
		{"/legible.emergency.v1.MsgProposeHalt", TransactionGasCosts["propose_halt"]},
	}

	for _, tc := range tests {
		got := lookupMsgGas(tc.msgTypeURL)
		if got != tc.wantGas {
			t.Errorf("lookupMsgGas(%q) = %d, want %d", tc.msgTypeURL, got, tc.wantGas)
		}
	}
}

func TestLookupMsgGas_UnknownTypeFallsBackToMinGas(t *testing.T) {
	unknownTypes := []string{
		"/cosmos.unknown.v1.MsgDoSomething",
		"/legible.future.v1.MsgNewFeature",
		"",
		"not-a-valid-type-url",
	}

	for _, msgType := range unknownTypes {
		got := lookupMsgGas(msgType)
		if got != MinGasLimit {
			t.Errorf("lookupMsgGas(%q) = %d, want MinGasLimit %d", msgType, got, MinGasLimit)
		}
	}
}

// ---------- Message Type Classification Tests ----------

func TestIsTransferMsg(t *testing.T) {
	trueCases := []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/cosmos.bank.v1beta1.MsgMultiSend",
		"/ibc.applications.transfer.v1.MsgTransfer",
	}
	for _, tc := range trueCases {
		if !isTransferMsg(tc) {
			t.Errorf("isTransferMsg(%q) = false, want true", tc)
		}
	}

	falseCases := []string{
		"/cosmos.staking.v1beta1.MsgDelegate",
		"/legible.knowledge.v1.MsgSubmitClaim",
		"/cosmos.gov.v1.MsgVote",
		"",
	}
	for _, tc := range falseCases {
		if isTransferMsg(tc) {
			t.Errorf("isTransferMsg(%q) = true, want false", tc)
		}
	}
}

func TestIsStakeMsg(t *testing.T) {
	trueCases := []string{
		"/cosmos.staking.v1beta1.MsgDelegate",
		"/cosmos.staking.v1beta1.MsgUndelegate",
		"/cosmos.staking.v1beta1.MsgBeginRedelegate",
		"/legible.staking.v1.MsgRegisterValidator",
	}
	for _, tc := range trueCases {
		if !isStakeMsg(tc) {
			t.Errorf("isStakeMsg(%q) = false, want true", tc)
		}
	}

	falseCases := []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/legible.knowledge.v1.MsgSubmitClaim",
		"",
	}
	for _, tc := range falseCases {
		if isStakeMsg(tc) {
			t.Errorf("isStakeMsg(%q) = true, want false", tc)
		}
	}
}

func TestIsClaimMsg(t *testing.T) {
	trueCases := []string{
		"/legible.knowledge.v1.MsgSubmitClaim",
		"/legible.knowledge.v1.MsgSubmitCommitment",
		"/legible.knowledge.v1.MsgSubmitReveal",
		"/legible.knowledge.v1.MsgChallengeFact",
	}
	for _, tc := range trueCases {
		if !isClaimMsg(tc) {
			t.Errorf("isClaimMsg(%q) = false, want true", tc)
		}
	}

	falseCases := []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/cosmos.gov.v1.MsgVote",
		"",
	}
	for _, tc := range falseCases {
		if isClaimMsg(tc) {
			t.Errorf("isClaimMsg(%q) = true, want false", tc)
		}
	}
}

func TestIsVoteMsg(t *testing.T) {
	trueCases := []string{
		"/cosmos.gov.v1.MsgVote",
		"/legible.gov.v1.MsgCastVote",
		"/legible.emergency.v1.MsgVoteHalt",
		"/legible.emergency.v1.MsgVoteRevert",
		"/legible.emergency.v1.MsgVoteResume",
	}
	for _, tc := range trueCases {
		if !isVoteMsg(tc) {
			t.Errorf("isVoteMsg(%q) = false, want true", tc)
		}
	}

	falseCases := []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/legible.knowledge.v1.MsgSubmitClaim",
		"",
	}
	for _, tc := range falseCases {
		if isVoteMsg(tc) {
			t.Errorf("isVoteMsg(%q) = true, want false", tc)
		}
	}
}

// ---------- Gas Constants Validation ----------

func TestGasConstantsInvariant(t *testing.T) {
	// MinGasLimit < TxGasLimit < BlockGasLimit
	if MinGasLimit >= TxGasLimit {
		t.Errorf("MinGasLimit (%d) should be < TxGasLimit (%d)", MinGasLimit, TxGasLimit)
	}
	if TxGasLimit >= BlockGasLimit {
		t.Errorf("TxGasLimit (%d) should be < BlockGasLimit (%d)", TxGasLimit, BlockGasLimit)
	}
}

func TestAllGasCostsBelowTxLimit(t *testing.T) {
	for txType, gas := range TransactionGasCosts {
		if gas > TxGasLimit {
			t.Errorf("TransactionGasCosts[%q] = %d exceeds TxGasLimit %d", txType, gas, TxGasLimit)
		}
	}
}

func TestFeeRoutingBPSSumTo1000000(t *testing.T) {
	// BPS constants are on a 1,000,000 scale (not 10,000):
	// 70000 = 7%, 930000 = 93%, sum = 1,000,000 = 100%
	if ResearchTaxBPS+ValidatorFeeBPS != 1000000 {
		t.Errorf("ResearchTaxBPS (%d) + ValidatorFeeBPS (%d) = %d, want 1000000",
			ResearchTaxBPS, ValidatorFeeBPS, ResearchTaxBPS+ValidatorFeeBPS)
	}
}

// ---------- EstimateTransactionGas Tests ----------

func TestEstimateTransactionGas_KnownType(t *testing.T) {
	got := EstimateTransactionGas("transfer")
	if got != 21_000 {
		t.Errorf("EstimateTransactionGas(transfer) = %d, want 21000", got)
	}
}

func TestEstimateTransactionGas_UnknownType(t *testing.T) {
	got := EstimateTransactionGas("nonexistent_type")
	if got != MinGasLimit {
		t.Errorf("EstimateTransactionGas(nonexistent_type) = %d, want %d", got, MinGasLimit)
	}
}

func TestIsSystemTransaction(t *testing.T) {
	sysTxs := []string{
		"verification_reward", "slash_validator", "epoch_transition",
		"emergency_halt", "emergency_revert", "emergency_resume", "block_reward",
	}
	for _, tx := range sysTxs {
		if !IsSystemTransaction(tx) {
			t.Errorf("IsSystemTransaction(%q) = false, want true", tx)
		}
	}

	nonSysTxs := []string{"transfer", "stake", "claim_submit", ""}
	for _, tx := range nonSysTxs {
		if IsSystemTransaction(tx) {
			t.Errorf("IsSystemTransaction(%q) = true, want false", tx)
		}
	}
}

// ---------- GetMinimumFee Tests ----------

// mockMsg implements sdk.Msg for testing gas calculations.
type mockMsg struct {
	typeURL string
}

func (m mockMsg) ProtoMessage()        {}
func (m mockMsg) Reset()               {}
func (m mockMsg) String() string       { return m.typeURL }
func (m mockMsg) ValidateBasic() error { return nil }

// We register the mock type URL via TypeURL method
func (m mockMsg) XXX_MessageName() string { return m.typeURL[1:] } // strip leading /

func TestGetMinimumFee_SingleMessage(t *testing.T) {
	// MsgSend → transfer → 21,000 gas → 21,000 * MinGasPrice = 21,000 ulgm
	msgs := []sdk.Msg{mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}}
	fee := GetMinimumFee(msgs)

	// transfer = 21,000 gas, but MinGasLimit is 22,222 so it gets clamped up
	expectedGas := uint64(22_222) // MinGasLimit since 21,000 < 22,222
	expectedFee := math.NewIntFromUint64(expectedGas * MinGasPrice)

	if !fee.AmountOf(BondDenom).Equal(expectedFee) {
		t.Errorf("GetMinimumFee for MsgSend = %s, want %s ulgm", fee, expectedFee)
	}
}

func TestGetMinimumFee_MultipleMessages(t *testing.T) {
	// 3 transfers: 21,000 * 3 = 63,000 gas
	msgs := []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	}
	fee := GetMinimumFee(msgs)

	expectedGas := uint64(63_000) // 21,000 * 3 > MinGasLimit
	expectedFee := math.NewIntFromUint64(expectedGas * MinGasPrice)

	if !fee.AmountOf(BondDenom).Equal(expectedFee) {
		t.Errorf("GetMinimumFee for 3x MsgSend = %s, want %s ulgm", fee, expectedFee)
	}
}

func TestGetMinimumFee_ExpensiveMessage(t *testing.T) {
	// deploy_contract = 200,000 gas
	msgs := []sdk.Msg{mockMsg{typeURL: "/legible.bvm.v1.MsgDeployContract"}}
	fee := GetMinimumFee(msgs)

	expectedGas := uint64(200_000)
	expectedFee := math.NewIntFromUint64(expectedGas * MinGasPrice)

	if !fee.AmountOf(BondDenom).Equal(expectedFee) {
		t.Errorf("GetMinimumFee for MsgDeployContract = %s, want %s ulgm", fee, expectedFee)
	}
}

// ---------- Proto URL Coverage Tests ----------

func TestMsgTypeURLToGas_AllEntriesReferenceValidCosts(t *testing.T) {
	// Every entry in msgTypeURLToGas should reference a valid gas cost (> 0)
	for url, gas := range msgTypeURLToGas {
		if gas == 0 {
			t.Errorf("msgTypeURLToGas[%q] = 0, all mapped messages should have non-zero gas", url)
		}
	}
}

func TestMsgTypeURLToGas_TreeModuleComplete(t *testing.T) {
	// Verify all tree module proto messages have gas mappings
	treeMessages := []string{
		"/legible.tree.v1.MsgCreateProject",
		"/legible.tree.v1.MsgProposeProject",
		"/legible.tree.v1.MsgStartDevelopment",
		"/legible.tree.v1.MsgCompleteProject",
		"/legible.tree.v1.MsgPauseProject",
		"/legible.tree.v1.MsgResumeProject",
		"/legible.tree.v1.MsgAbandonProject",
		"/legible.tree.v1.MsgSpawnChildProject",
		"/legible.tree.v1.MsgAddTask",
		"/legible.tree.v1.MsgAssignTask",
		"/legible.tree.v1.MsgStartWork",
		"/legible.tree.v1.MsgSubmitDeliverable",
		"/legible.tree.v1.MsgApproveDeliverable",
		"/legible.tree.v1.MsgRejectDeliverable",
		"/legible.tree.v1.MsgReopenTask",
		"/legible.tree.v1.MsgDeployService",
		"/legible.tree.v1.MsgCallService",
		"/legible.tree.v1.MsgSubscribeService",
		"/legible.tree.v1.MsgPauseService",
		"/legible.tree.v1.MsgResumeService",
		"/legible.tree.v1.MsgRetireService",
		"/legible.tree.v1.MsgBeginSeeding",
		"/legible.tree.v1.MsgDetectOpportunity",
		"/legible.tree.v1.MsgClaimOpportunity",
		"/legible.tree.v1.MsgAddContributor",
		"/legible.tree.v1.MsgApplyToProject",
		"/legible.tree.v1.MsgReviewApplication",
		"/legible.tree.v1.MsgSetAvailability",
	}

	for _, msg := range treeMessages {
		if _, ok := msgTypeURLToGas[msg]; !ok {
			t.Errorf("missing gas mapping for tree message %q", msg)
		}
	}
}

// ---------- LGMGasDecorator Tests ----------

// mockFeeTx implements sdk.FeeTx for testing.
type mockFeeTx struct {
	sdk.Tx
	gas  uint64
	fee  sdk.Coins
	msgs []sdk.Msg
}

func (m mockFeeTx) GetGas() uint64     { return m.gas }
func (m mockFeeTx) GetFee() sdk.Coins  { return m.fee }
func (m mockFeeTx) GetMsgs() []sdk.Msg { return m.msgs }
func (m mockFeeTx) FeeGranter() []byte { return nil }
func (m mockFeeTx) FeePayer() []byte   { return nil }

// passThroughHandler is a no-op next handler for testing decorators.
func passThroughHandler(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	return ctx, nil
}

func TestLGMGasDecorator_SimulateSkips(t *testing.T) {
	decorator := NewLGMGasDecorator()
	ctx := sdk.Context{}
	tx := mockFeeTx{gas: 0} // Would fail if not simulating

	_, err := decorator.AnteHandle(ctx, tx, true, passThroughHandler)
	if err != nil {
		t.Errorf("simulate mode should skip validation, got error: %v", err)
	}
}

func TestLGMGasDecorator_ExceedsBlockGasLimit(t *testing.T) {
	decorator := NewLGMGasDecorator()
	ctx := sdk.Context{}
	tx := mockFeeTx{
		gas:  BlockGasLimit + 1,
		fee:  sdk.NewCoins(sdk.NewCoin(BondDenom, math.NewInt(999999999))),
		msgs: []sdk.Msg{mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}},
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Error("expected error for gas exceeding block limit")
	}
}

func TestLGMGasDecorator_ExceedsTxGasLimit(t *testing.T) {
	decorator := NewLGMGasDecorator()
	ctx := sdk.Context{}
	tx := mockFeeTx{
		gas:  TxGasLimit + 1,
		fee:  sdk.NewCoins(sdk.NewCoin(BondDenom, math.NewInt(999999999))),
		msgs: []sdk.Msg{mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}},
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Error("expected error for gas exceeding tx limit")
	}
}

func TestLGMGasDecorator_InsufficientGas(t *testing.T) {
	decorator := NewLGMGasDecorator()
	ctx := sdk.Context{}
	// deploy_contract requires 200,000 gas, provide only 1000
	tx := mockFeeTx{
		gas:  1000,
		fee:  sdk.NewCoins(sdk.NewCoin(BondDenom, math.NewInt(999999999))),
		msgs: []sdk.Msg{mockMsg{typeURL: "/legible.bvm.v1.MsgDeployContract"}},
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Error("expected error for insufficient gas")
	}
}

func TestLGMGasDecorator_SufficientGas(t *testing.T) {
	decorator := NewLGMGasDecorator()
	ctx := sdk.Context{}
	// transfer = 21,000 but MinGasLimit = 22,222; provide 30,000
	tx := mockFeeTx{
		gas:  30_000,
		fee:  sdk.NewCoins(sdk.NewCoin(BondDenom, math.NewInt(30_000))),
		msgs: []sdk.Msg{mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}},
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Errorf("expected no error for sufficient gas, got: %v", err)
	}
}

func TestLGMGasDecorator_InsufficientFee(t *testing.T) {
	decorator := NewLGMGasDecorator()
	ctx := sdk.Context{}
	// Gas = 100,000, MinGasPrice = 1, so min fee = 100,000 ulgm. Provide only 1.
	tx := mockFeeTx{
		gas:  100_000,
		fee:  sdk.NewCoins(sdk.NewCoin(BondDenom, math.NewInt(1))),
		msgs: []sdk.Msg{mockMsg{typeURL: "/legible.knowledge.v1.MsgSubmitClaim"}},
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Error("expected error for insufficient fee")
	}
}
