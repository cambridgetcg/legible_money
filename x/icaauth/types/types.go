package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- ValidateBasic methods for protobuf message types ---
// Struct definitions for RemoteAccount, InterchainAccountRecord, Params, and
// GenesisState are provided by the proto-generated .pb.go files (state.pb.go
// and genesis.pb.go).  Only methods and non-proto types are kept here.

// Validate validates the module parameters.
func (p *Params) Validate() error {
	if p.MaxRemoteAccountsPerDid == 0 {
		return fmt.Errorf("max_remote_accounts_per_did must be positive")
	}
	if p.RegistrationCooldown == 0 {
		return fmt.Errorf("registration_cooldown must be positive")
	}
	return nil
}

// --- Msg struct definitions, MsgServer interface, gRPC service descriptors,
// and RegisterMsgServer are provided by the proto-generated tx.pb.go and
// tx_grpc.pb.go files. Only GetSigners() and ValidateBasic() methods remain. ---

func (msg *MsgRegisterInterchainAccount) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{sender}
}

func (msg *MsgRegisterInterchainAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if msg.ConnectionId == "" {
		return fmt.Errorf("connection_id cannot be empty")
	}
	return nil
}

func (msg *MsgSendICATransaction) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{sender}
}

func (msg *MsgSendICATransaction) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if msg.ConnectionId == "" {
		return fmt.Errorf("connection_id cannot be empty")
	}
	if len(msg.MsgBytes) == 0 {
		return fmt.Errorf("msg_bytes cannot be empty")
	}
	return nil
}

// Query types (QueryServer, QueryInterchainAccountsRequest/Response, QueryParamsRequest/Response)
// are now provided by the proto-generated query.pb.go and query_grpc.pb.go files.

func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateParams) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return msg.Params.Validate()
}
