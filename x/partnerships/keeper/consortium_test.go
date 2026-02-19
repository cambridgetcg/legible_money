package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/partnerships/types"
)

// -----------------------------------------------------------------------
// Tests: Consortium Formation
// -----------------------------------------------------------------------

func TestConsortiumFormation(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	founder := testAddr(1)
	member2 := testAddr(2)
	member3 := testAddr(3)

	id, err := k.CreateConsortium(ctx, founder, "Test Consortium",
		[]string{founder, member2, member3},
		[]string{"human", "agent", "agent"},
		[]uint64{400000, 300000, 300000},
		0, // lock tier 0
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty consortium ID")
	}

	// Verify forming status, founder accepted, others not
	c, found := k.GetConsortium(ctx, id)
	if !found {
		t.Fatal("consortium not found")
	}
	if c.Status != types.StatusForming {
		t.Errorf("expected forming, got %s", c.Status)
	}
	if len(c.Members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(c.Members))
	}

	// Founder should be accepted
	for _, m := range c.Members {
		if m.Address == founder {
			if !m.Accepted {
				t.Error("founder should be auto-accepted")
			}
			if m.Role != "founder" {
				t.Errorf("founder role expected, got %s", m.Role)
			}
		} else {
			if m.Accepted {
				t.Errorf("member %s should not be accepted yet", m.Address)
			}
		}
	}

	// Accept from member 2 -> still forming
	err = k.AcceptConsortium(ctx, id, member2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c, _ = k.GetConsortium(ctx, id)
	if c.Status != types.StatusForming {
		t.Errorf("expected still forming after 1 accept, got %s", c.Status)
	}

	// Accept from member 3 -> should activate
	err = k.AcceptConsortium(ctx, id, member3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c, _ = k.GetConsortium(ctx, id)
	if c.Status != types.StatusActive {
		t.Errorf("expected active after all accept, got %s", c.Status)
	}
	if c.FormedAtBlock != 100 { // block height from setupKeeper
		t.Errorf("expected FormedAtBlock 100, got %d", c.FormedAtBlock)
	}
}

func TestConsortiumMaxMembers(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	founder := testAddr(1)

	// 12 members -> should fail
	addrs := make([]string, 12)
	acctTypes := make([]string, 12)
	shares := make([]uint64, 12)
	for i := 0; i < 12; i++ {
		addrs[i] = testAddr(i + 1)
		acctTypes[i] = "agent"
		shares[i] = 83333
	}
	acctTypes[0] = "human"
	shares[11] = 83337 // adjust last to sum to 1M

	_, err := k.CreateConsortium(ctx, founder, "Too Big",
		addrs, acctTypes, shares, 0)
	if err == nil {
		t.Error("expected error for 12 members")
	}

	// 11 members -> should succeed
	addrs = addrs[:11]
	acctTypes = acctTypes[:11]
	shares = shares[:11]
	shares[10] = 83337 + 83333 // sum to 1M
	_, err = k.CreateConsortium(ctx, founder, "Max Size",
		addrs, acctTypes, shares, 0)
	if err != nil {
		t.Fatalf("unexpected error for 11 members: %v", err)
	}
}

func TestConsortiumMinMembers(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	founder := testAddr(1)

	// 1 member -> should fail
	_, err := k.CreateConsortium(ctx, founder, "Solo",
		[]string{founder},
		[]string{"human"},
		[]uint64{1000000},
		0,
	)
	if err == nil {
		t.Error("expected error for 1 member")
	}
}

func TestConsortiumSharesValidation(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	founder := testAddr(1)
	member2 := testAddr(2)

	// Shares not summing to 1M -> error
	_, err := k.CreateConsortium(ctx, founder, "Bad Shares",
		[]string{founder, member2},
		[]string{"human", "agent"},
		[]uint64{500000, 400000}, // = 900000
		0,
	)
	if err == nil {
		t.Error("expected error for shares not summing to 1M")
	}

	// Shares summing to 1M -> success
	_, err = k.CreateConsortium(ctx, founder, "Good Shares",
		[]string{founder, member2},
		[]string{"human", "agent"},
		[]uint64{600000, 400000}, // = 1M
		0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// -----------------------------------------------------------------------
// Tests: Consortium Rewards
// -----------------------------------------------------------------------

func TestConsortiumRewardSplit(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	founder := testAddr(1)
	member2 := testAddr(2)
	member3 := testAddr(3)

	// Create and activate a consortium
	id, err := k.CreateConsortium(ctx, founder, "Reward Test",
		[]string{founder, member2, member3},
		[]string{"human", "agent", "agent"},
		[]uint64{400000, 300000, 300000},
		0, // lock tier 0 (1.0x multiplier)
	)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	// Manually activate
	c, _ := k.GetConsortium(ctx, id)
	for _, m := range c.Members {
		m.Accepted = true
	}
	c.Status = types.StatusActive
	c.FormedAtBlock = 100
	k.SetConsortium(ctx, c)

	// Distribute 10000 reward
	shares, potAdd, err := k.DistributeConsortiumReward(ctx, c, "10000", "verification")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Lock tier 0 (1.0x): adjusted = 10000
	// CommonPotShareBps=220000: pot = 10000 * 220000 / 1000000 = 2200
	// Remaining = 7800
	// Member1 (400000): 7800 * 400000 / 1000000 = 3120
	// Member2 (300000): 7800 * 300000 / 1000000 = 2340
	// Member3 (last, remainder): 7800 - 3120 - 2340 = 2340
	if potAdd != "2200" {
		t.Errorf("expected pot 2200, got %s", potAdd)
	}
	if shares[founder] != "3120" {
		t.Errorf("expected founder share 3120, got %s", shares[founder])
	}
	if shares[member2] != "2340" {
		t.Errorf("expected member2 share 2340, got %s", shares[member2])
	}
	if shares[member3] != "2340" {
		t.Errorf("expected member3 share (remainder) 2340, got %s", shares[member3])
	}

	// Verify consortium pot updated
	updated, _ := k.GetConsortium(ctx, id)
	if updated.CommonPotBalance != "2200" {
		t.Errorf("expected pot balance 2200, got %s", updated.CommonPotBalance)
	}
	if updated.TotalEarned != "10000" {
		t.Errorf("expected total earned 10000, got %s", updated.TotalEarned)
	}
}

// -----------------------------------------------------------------------
// Tests: Consortium Exit
// -----------------------------------------------------------------------

func TestConsortiumMemberExit(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	founder := testAddr(1)
	member2 := testAddr(2)
	member3 := testAddr(3)

	id, err := k.CreateConsortium(ctx, founder, "Exit Test",
		[]string{founder, member2, member3},
		[]string{"human", "agent", "agent"},
		[]uint64{400000, 300000, 300000},
		0,
	)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	// Activate
	c, _ := k.GetConsortium(ctx, id)
	for _, m := range c.Members {
		m.Accepted = true
	}
	c.Status = types.StatusActive
	c.LockExpiresAt = 50 // already expired (current=100)
	k.SetConsortium(ctx, c)

	// Member2 exits
	err = k.HandleConsortiumMemberExit(ctx, id, member2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c, _ = k.GetConsortium(ctx, id)
	if len(c.Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(c.Members))
	}
	if c.Status != types.StatusActive {
		t.Errorf("expected still active, got %s", c.Status)
	}

	// Verify shares sum to 1M after redistribution
	var totalShares uint64
	for _, m := range c.Members {
		totalShares += m.ShareBps
	}
	if totalShares != 1000000 {
		t.Errorf("expected shares to sum to 1000000, got %d", totalShares)
	}
}

func TestConsortiumDissolution(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	founder := testAddr(1)
	member2 := testAddr(2)

	id, err := k.CreateConsortium(ctx, founder, "Dissolve Test",
		[]string{founder, member2},
		[]string{"human", "agent"},
		[]uint64{500000, 500000},
		0,
	)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	// Activate
	c, _ := k.GetConsortium(ctx, id)
	for _, m := range c.Members {
		m.Accepted = true
	}
	c.Status = types.StatusActive
	c.LockExpiresAt = 50 // expired
	k.SetConsortium(ctx, c)

	// Member exits -> drops below minimum -> cooling
	err = k.HandleConsortiumMemberExit(ctx, id, member2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c, _ = k.GetConsortium(ctx, id)
	if c.Status != types.StatusCooling {
		t.Errorf("expected cooling, got %s", c.Status)
	}
	if c.ExitState == nil {
		t.Fatal("expected exit state")
	}
}

// -----------------------------------------------------------------------
// Tests: Index Queries
// -----------------------------------------------------------------------

func TestGetActiveConsortiumForAddress(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	founder := testAddr(1)
	member2 := testAddr(2)

	id, err := k.CreateConsortium(ctx, founder, "Active Test",
		[]string{founder, member2},
		[]string{"human", "agent"},
		[]uint64{500000, 500000},
		0,
	)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Activate
	c, _ := k.GetConsortium(ctx, id)
	for _, m := range c.Members {
		m.Accepted = true
	}
	c.Status = types.StatusActive
	k.SetConsortium(ctx, c)

	// Should find for both members
	got, found := k.GetActiveConsortiumForAddress(ctx, founder)
	if !found {
		t.Fatal("expected to find active consortium for founder")
	}
	if got.Id != id {
		t.Errorf("expected %s, got %s", id, got.Id)
	}

	got, found = k.GetActiveConsortiumForAddress(ctx, member2)
	if !found {
		t.Fatal("expected to find active consortium for member2")
	}
	if got.Id != id {
		t.Errorf("expected %s, got %s", id, got.Id)
	}

	// Unknown address
	_, found = k.GetActiveConsortiumForAddress(ctx, testAddr(99))
	if found {
		t.Error("expected not found for unknown address")
	}

	// Dissolved consortium -> not found
	c.Status = types.StatusDissolved
	k.SetConsortium(ctx, c)
	_, found = k.GetActiveConsortiumForAddress(ctx, founder)
	if found {
		t.Error("dissolved consortium should not be returned")
	}
}

// -----------------------------------------------------------------------
// Tests: Duplicate Prevention
// -----------------------------------------------------------------------

func TestConsortiumMemberAlreadyInPartnership(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	human := testAddr(1)
	agent := testAddr(2)
	member3 := testAddr(3)

	// Create active partnership for human+agent
	p := types.Partnership{
		Id:               "p-active",
		HumanAddr:        human,
		AgentAddr:        agent,
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
	}
	k.SetPartnership(ctx, &p)

	// Try to create consortium with the human who is in a partnership
	_, err := k.CreateConsortium(ctx, human, "Conflict",
		[]string{human, member3},
		[]string{"human", "agent"},
		[]uint64{500000, 500000},
		0,
	)
	if err == nil {
		t.Error("expected error: member already in active partnership")
	}

	// Try with agent
	_, err = k.CreateConsortium(ctx, member3, "Conflict2",
		[]string{member3, agent},
		[]string{"human", "agent"},
		[]uint64{500000, 500000},
		0,
	)
	if err == nil {
		t.Error("expected error: agent already in active partnership")
	}
}

func TestConsortiumMemberAlreadyInConsortium(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	founder := testAddr(1)
	member2 := testAddr(2)
	member3 := testAddr(3)

	// Create and activate first consortium
	id, err := k.CreateConsortium(ctx, founder, "First",
		[]string{founder, member2},
		[]string{"human", "agent"},
		[]uint64{500000, 500000},
		0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c, _ := k.GetConsortium(ctx, id)
	for _, m := range c.Members {
		m.Accepted = true
	}
	c.Status = types.StatusActive
	k.SetConsortium(ctx, c)

	// Try to create second consortium with founder -> should fail
	_, err = k.CreateConsortium(ctx, founder, "Second",
		[]string{founder, member3},
		[]string{"human", "agent"},
		[]uint64{500000, 500000},
		0,
	)
	if err == nil {
		t.Error("expected error: founder already in active consortium")
	}
}

// -----------------------------------------------------------------------
// Tests: Formation Expiry
// -----------------------------------------------------------------------

func TestConsortiumFormationExpiry(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	founder := testAddr(1)
	member2 := testAddr(2)

	id, err := k.CreateConsortium(ctx, founder, "Expiry Test",
		[]string{founder, member2},
		[]string{"human", "agent"},
		[]uint64{500000, 500000},
		0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Advance past formation window (current=100, window=154)
	ctx = ctx.WithBlockHeight(300)
	k.ExpireConsortiumFormations(ctx)

	c, _ := k.GetConsortium(ctx, id)
	if c.Status != types.StatusDissolved {
		t.Errorf("expected dissolved, got %s", c.Status)
	}
}

// -----------------------------------------------------------------------
// Tests: Cooling Settlement
// -----------------------------------------------------------------------

func TestSettleCoolingConsortia(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	founder := testAddr(1)
	member2 := testAddr(2)

	id, err := k.CreateConsortium(ctx, founder, "Cool Test",
		[]string{founder, member2},
		[]string{"human", "agent"},
		[]uint64{500000, 500000},
		0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Set to cooling with expired cooldown
	c, _ := k.GetConsortium(ctx, id)
	c.Status = types.StatusCooling
	c.ExitState = &types.ConsortiumExitState{
		InitiatedBy: founder,
		InitiatedAt: 50,
		CooldownEnd: 90, // before current block (100)
	}
	k.SetConsortium(ctx, c)

	k.SettleCoolingConsortia(ctx)

	c, _ = k.GetConsortium(ctx, id)
	if c.Status != types.StatusDissolved {
		t.Errorf("expected dissolved, got %s", c.Status)
	}
}

// -----------------------------------------------------------------------
// Tests: CRUD
// -----------------------------------------------------------------------

func TestConsortiumCRUD(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Not found initially
	_, found := k.GetConsortium(ctx, "c-1")
	if found {
		t.Error("expected not found")
	}

	// Set
	c := &types.Consortium{
		Id:                "c-1",
		Name:              "Test",
		Status:            types.StatusActive,
		CommonPotBalance:  "1000",
		TotalEarned:       "5000",
		CommonPotShareBps: 220000,
		Members: []*types.ConsortiumMember{
			{Address: testAddr(1), AccountType: "human", ShareBps: 500000, Role: "founder", Accepted: true},
			{Address: testAddr(2), AccountType: "agent", ShareBps: 500000, Role: "member", Accepted: true},
		},
	}
	k.SetConsortium(ctx, c)

	// Get
	got, found := k.GetConsortium(ctx, "c-1")
	if !found {
		t.Fatal("expected consortium found")
	}
	if got.Name != "Test" {
		t.Errorf("expected name Test, got %s", got.Name)
	}
	if got.CommonPotBalance != "1000" {
		t.Errorf("expected pot 1000, got %s", got.CommonPotBalance)
	}

	// GetAll
	all := k.GetAllConsortia(ctx)
	if len(all) != 1 {
		t.Errorf("expected 1, got %d", len(all))
	}

	// ByMember index
	ids := k.GetConsortiumsByMember(ctx, testAddr(1))
	if len(ids) != 1 || ids[0] != "c-1" {
		t.Errorf("expected [c-1], got %v", ids)
	}

	// Delete
	k.DeleteConsortium(ctx, c)
	_, found = k.GetConsortium(ctx, "c-1")
	if found {
		t.Error("expected deleted")
	}
	ids = k.GetConsortiumsByMember(ctx, testAddr(1))
	if len(ids) != 0 {
		t.Errorf("expected empty after delete, got %v", ids)
	}
}

// -----------------------------------------------------------------------
// Tests: Accept edge cases
// -----------------------------------------------------------------------

func TestAcceptConsortiumAlreadyAccepted(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	founder := testAddr(1)
	member2 := testAddr(2)

	id, err := k.CreateConsortium(ctx, founder, "Dupe Accept",
		[]string{founder, member2},
		[]string{"human", "agent"},
		[]uint64{500000, 500000},
		0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Founder tries to accept again (already auto-accepted)
	err = k.AcceptConsortium(ctx, id, founder)
	if err == nil {
		t.Error("expected already accepted error")
	}
}

func TestAcceptConsortiumNotMember(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	founder := testAddr(1)
	member2 := testAddr(2)

	id, err := k.CreateConsortium(ctx, founder, "Non-Member",
		[]string{founder, member2},
		[]string{"human", "agent"},
		[]uint64{500000, 500000},
		0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Non-member tries to accept
	err = k.AcceptConsortium(ctx, id, testAddr(99))
	if err == nil {
		t.Error("expected not consortium member error")
	}
}

// -----------------------------------------------------------------------
// Tests: Consortium Genesis
// -----------------------------------------------------------------------

func TestConsortiumGenesis(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	c := &types.Consortium{
		Id:                "c-gen",
		Name:              "Genesis Test",
		Status:            types.StatusActive,
		CommonPotBalance:  "5000",
		TotalEarned:       "20000",
		CommonPotShareBps: 220000,
		CooperationScore:  7500,
		FormedAtBlock:     50,
		Members: []*types.ConsortiumMember{
			{Address: testAddr(1), AccountType: "human", ShareBps: 500000, Role: "founder", Accepted: true},
			{Address: testAddr(2), AccountType: "agent", ShareBps: 500000, Role: "member", Accepted: true},
		},
	}
	k.SetConsortium(ctx, c)

	// Export
	exported := k.ExportConsortiumGenesis(ctx)
	if len(exported) != 1 {
		t.Errorf("expected 1, got %d", len(exported))
	}

	// Import into new keeper
	k2, ctx2, _ := setupKeeper(t)
	k2.InitConsortiumGenesis(ctx2, exported)

	// Verify
	got := k2.ExportConsortiumGenesis(ctx2)
	if len(got) != 1 {
		t.Errorf("expected 1, got %d", len(got))
	}
	if got[0].CommonPotBalance != "5000" {
		t.Errorf("expected pot 5000, got %s", got[0].CommonPotBalance)
	}
}
