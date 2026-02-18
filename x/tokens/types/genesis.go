package types

// DefaultParams returns default module parameters.
// Expiration fields are zeroed — the period overlay has been removed.
func DefaultParams() Params {
	return Params{}
}

// TokenGenesisEntry holds a token definition with its balances and allowances for genesis.
type TokenGenesisEntry struct {
	Token      *TokenDefinition  `json:"token"`
	Balances   map[string]string `json:"balances"`   // ownerAddr → amount string
	Allowances map[string]string `json:"allowances"` // "ownerAddr/spenderAddr" → amount string
}

// PoolGenesisEntry holds a liquidity pool with its LP positions for genesis.
type PoolGenesisEntry struct {
	Pool      *LiquidityPool    `json:"pool"`
	Positions map[string]string `json:"positions"` // providerAddr → LP token amount string
}

// DelegationGenesisEntry holds delegation data for a token at genesis.
type DelegationGenesisEntry struct {
	TokenId     string            `json:"token_id"`
	Delegations map[string]string `json:"delegations"` // "delegator/delegate" → amount
	Totals      map[string]string `json:"totals"`       // delegator → total
}

// WrapGenesisEntry holds a wrap record mapping for genesis.
type WrapGenesisEntry struct {
	TokenId      string `json:"token_id"`
	WrappedDenom string `json:"wrapped_denom"`
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params:   &p,
		Accounts: []*ExpirableAccount{},
		Rai: &RAIHoldingAccount{
			Balance:          "0",
			TotalReceived:    "0",
			TotalDistributed: "0",
			LastDistBlock:    0,
		},
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	return nil
}
