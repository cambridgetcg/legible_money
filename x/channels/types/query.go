package types

import "context"

// QueryServer defines the channels module query interface.
type QueryServer interface {
	Channel(context.Context, *QueryChannelRequest) (*QueryChannelResponse, error)
	ChannelsByPayer(context.Context, *QueryChannelsByPayerRequest) (*QueryChannelsByPayerResponse, error)
	ChannelsByProvider(context.Context, *QueryChannelsByProviderRequest) (*QueryChannelsByProviderResponse, error)
	ActiveDispute(context.Context, *QueryActiveDisputeRequest) (*QueryActiveDisputeResponse, error)
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
}

// ---------- Query Types ----------

type QueryChannelRequest struct {
	ChannelId string `json:"channel_id"`
}

type QueryChannelResponse struct {
	Channel *PaymentChannel `json:"channel"`
}

type QueryChannelsByPayerRequest struct {
	Payer  string `json:"payer"`
	Limit  uint32 `json:"limit,omitempty"`
	Offset uint32 `json:"offset,omitempty"`
}

type QueryChannelsByPayerResponse struct {
	Channels []PaymentChannel `json:"channels"`
	Total    uint64           `json:"total"`
}

type QueryChannelsByProviderRequest struct {
	Provider string `json:"provider"`
	Limit    uint32 `json:"limit,omitempty"`
	Offset   uint32 `json:"offset,omitempty"`
}

type QueryChannelsByProviderResponse struct {
	Channels []PaymentChannel `json:"channels"`
	Total    uint64           `json:"total"`
}

type QueryActiveDisputeRequest struct {
	ChannelId string `json:"channel_id"`
}

type QueryActiveDisputeResponse struct {
	Dispute *ChannelDispute `json:"dispute"`
	Found   bool            `json:"found"`
}

type QueryParamsRequest struct{}

type QueryParamsResponse struct {
	Params Params `json:"params"`
}
