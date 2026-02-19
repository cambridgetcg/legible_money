# B14-5 — Integration Test: Unified Revenue Across All Modules

## Context
After batch 14, revenue flows through multiple modules with a consistent
philosophy. This session verifies the entire revenue system works correctly
end-to-end: block rewards, tx fees, knowledge rewards, and service revenue
all follow the same principles.

## Test Plan

### 1. Create `x/integration_test/revenue_test.go`

Or add to an existing integration test file. This needs access to multiple
module keepers.

### 2. Test: Complete Revenue Flow Map

```go
func TestCompleteRevenueMap(t *testing.T) {
    // Verify the entire revenue flow:
    //
    // SOURCE 1: Block Rewards
    //   Minted → 3.33% research (with 7% founder split) → 96.67% post-tax
    //          → 50% verification pool (70% knowledge, 30% compute)
    //          → 50% block producer
    //
    // SOURCE 2: Transaction Fees
    //   Collected → 3.33% swept to research (with 7% founder split)
    //            → 96.67% to validators via distribution
    //
    // SOURCE 3: Knowledge Rewards (from verification pool)
    //   Verification reward → NO additional tax (already taxed at minting)
    //   Citation reward → 3.33% research tax (with 7% founder split)
    //   Falsification reward → 3.33% research tax (with 7% founder split)
    //
    // SOURCE 4: Tree Service Revenue
    //   Service call → 55% contributors, 22% treasury, 13% research
    //                  (with 7% founder split), 10% burn
    //
    // SOURCE 5: Billing Service Revenue
    //   Billing payment → 55% provider, 22% knowledge pool, 13% research
    //                     (with 7% founder split), 10% protocol (5% burn + 5% treasury)
}
```

### 3. Test: Founder Split Consistency

```go
func TestFounderSplitAllSources(t *testing.T) {
    // Deposit 1M ulgm to research fund from each source module.
    // Verify founder receives exactly 70,000 ulgm (7%) each time.
    // Verify research_fund receives exactly 930,000 ulgm each time.
    
    sources := []string{
        "vesting_rewards",  // block rewards
        "tree",             // service revenue
        "billing",          // billing payments
        "knowledge",        // citation/falsification tax
    }
    
    for _, source := range sources {
        // Deposit via DepositToResearchFund
        // Assert founder balance increased by 70,000
        // Assert research_fund balance increased by 930,000
    }
}
```

### 4. Test: No Double Taxation

```go
func TestNoDoubleTaxation(t *testing.T) {
    // Scenario: Block reward mints 1M ulgm
    // Step 1: 33,300 ulgm (3.33%) goes to research fund → founder gets 2,331
    // Step 2: 966,700 ulgm post-tax splits to verification pool + producer
    // Step 3: Verification pool funds knowledge module
    // Step 4: Knowledge module pays verification reward from its balance
    // Step 5: Verification reward goes directly to verifier — NO additional tax
    //
    // Assert: The verifier receives the full amount from knowledge module
    // Assert: research_fund received tax ONLY at step 1, not at step 5
}
```

### 5. Test: Service Revenue Burn

```go
func TestServiceRevenueBurn(t *testing.T) {
    // Tree service call for 1M ulgm
    // Assert: 100,000 ulgm (10%) was burned
    // Assert: total supply decreased by 100,000
    
    // Billing payment for 1M ulgm
    // Assert: 50,000 ulgm (5%) was burned
    // Assert: total supply decreased by 50,000
}
```

### 6. Test: Dead Accounts Have Zero Balance

```go
func TestDeadAccountsRemoved(t *testing.T) {
    deadAccounts := []string{
        "treasury_research",
        "treasury_foundation",
        "treasury_community",
        "treasury_developers",
        "treasury_reserve",
    }
    for _, name := range deadAccounts {
        // Assert: module account either doesn't exist or has zero balance
    }
}
```

### 7. Test: Full Ledger Balance

```go
func TestLedgerBalance(t *testing.T) {
    // After a series of operations (mint, service calls, claims, citations):
    // Assert: total supply = minted - burned
    // Assert: sum of all account balances + module accounts = total supply
    // No tokens lost or created outside of minting/burning
}
```

### 8. Revenue Summary Report

Add a CLI command or query that shows the complete revenue breakdown:
```bash
legbled query revenue summary
```

Output:
```json
{
    "block_rewards_distributed": "1234567890",
    "tx_fees_collected": "456789",
    "research_fund_balance": "89012",
    "research_fund_total_deposits": "123456",
    "founder_total_received": "8642",
    "protocol_treasury_balance": "45678",
    "total_burned": "34567",
    "total_minted": "2345678901",
    "current_supply": "2345644334"
}
```

### 9. Run

```bash
go test ./x/integration_test/... -count=1 -v -timeout 120s
# Or if using per-module integration tests:
go test ./x/vesting_rewards/... ./x/knowledge/... ./x/tree/... ./x/billing/... -count=1 -v
go build ./...
```

## Commit
```
test: unified revenue integration test across all modules

Verifies complete revenue flow: block rewards, tx fees, knowledge
rewards, and service revenue all follow consistent taxation and
distribution. Validates founder split consistency, no double taxation,
burn mechanics, dead account cleanup, and full ledger balance.
```
