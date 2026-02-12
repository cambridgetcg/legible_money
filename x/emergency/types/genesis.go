package types

import "fmt"

// DefaultParams returns the default emergency module parameters.
func DefaultParams() Params {
	return Params{
		HaltQuorum:                      750000,  // 75%
		RevertQuorum:                    800000,  // 80%
		ResumeQuorum:                    800000,  // 80%
		HaltPrevoteBlocks:               11,
		HaltPrecommitBlocks:             11,
		HaltTimeoutBlocks:               44,
		RevertPrevoteBlocks:             22,
		RevertPrecommitBlocks:           22,
		RevertTimeoutBlocks:             111,
		ResumePrevoteBlocks:             22,
		ResumePrecommitBlocks:           22,
		ResumeTimeoutBlocks:             111,
		MaxProposalsPerEpoch:            3,
		MaxProposalsPerGuardianPerEpoch: 1,
		CooldownBlocks:                  111,
		MinGuardianStake:                "111111000000", // 111,111 LGM in ulgm
		MinDistinctVoters:               4,
		MaxRevertDepth:                  111111,
		EpochBlocks:                     34272, // ~1 day at 2521ms blocks
		GenesisCouncil:                  []string{},
		CouncilExpiryBlock:              0,
		CouncilVirtualStake:             "11111000000", // 11,111 LGM in ulgm
	}
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params: &p,
		Status: string(StatusNormal),
	}
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return gs.Params.Validate()
}
