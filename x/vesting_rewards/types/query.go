package types

import "context"

// QueryServer defines the vesting_rewards module query interface.
type QueryServer interface {
	VestingSchedule(context.Context, *QueryVestingScheduleRequest) (*QueryVestingScheduleResponse, error)
	VestingSchedulesByRecipient(context.Context, *QueryVestingSchedulesByRecipientRequest) (*QueryVestingSchedulesByRecipientResponse, error)
	ActiveVestingSchedules(context.Context, *QueryActiveVestingSchedulesRequest) (*QueryActiveVestingSchedulesResponse, error)
	BlockRewardDistribution(context.Context, *QueryBlockRewardDistributionRequest) (*QueryBlockRewardDistributionResponse, error)
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
}

// ---------- Query Types ----------

type QueryVestingScheduleRequest struct {
	VestingId string `json:"vesting_id"`
}

type QueryVestingScheduleResponse struct {
	Schedule *VestingSchedule `json:"schedule"`
}

type QueryVestingSchedulesByRecipientRequest struct {
	Recipient string `json:"recipient"`
}

type QueryVestingSchedulesByRecipientResponse struct {
	Schedules []*VestingSchedule `json:"schedules"`
}

type QueryActiveVestingSchedulesRequest struct {
	Limit  uint32 `json:"limit,omitempty"`
	Offset uint32 `json:"offset,omitempty"`
}

type QueryActiveVestingSchedulesResponse struct {
	Schedules []*VestingSchedule `json:"schedules"`
	Total     uint32             `json:"total"`
}

type QueryBlockRewardDistributionRequest struct {
	BlockHeight uint64 `json:"block_height"`
}

type QueryBlockRewardDistributionResponse struct {
	Distribution *BlockRewardDistribution `json:"distribution"`
	Found        bool                     `json:"found"`
}

type QueryParamsRequest struct{}

type QueryParamsResponse struct {
	Params          *Params           `json:"params"`
	CategoryConfigs []*CategoryConfig `json:"category_configs"`
}
