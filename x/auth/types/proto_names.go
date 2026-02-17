package types

// XXX_MessageName methods for hand-written types — provides unique proto
// message names so the Cosmos SDK interface registry derives distinct type URLs.

func (*MsgRegisterAccount) XXX_MessageName() string              { return "legible.auth.v1.MsgRegisterAccount" }
func (*MsgRegisterAccountResponse) XXX_MessageName() string      { return "legible.auth.v1.MsgRegisterAccountResponse" }
func (*MsgFreezeAccount) XXX_MessageName() string                { return "legible.auth.v1.MsgFreezeAccount" }
func (*MsgFreezeAccountResponse) XXX_MessageName() string        { return "legible.auth.v1.MsgFreezeAccountResponse" }
func (*MsgUnfreezeAccount) XXX_MessageName() string              { return "legible.auth.v1.MsgUnfreezeAccount" }
func (*MsgUnfreezeAccountResponse) XXX_MessageName() string      { return "legible.auth.v1.MsgUnfreezeAccountResponse" }
func (*MsgSetRecoveryConfig) XXX_MessageName() string            { return "legible.auth.v1.MsgSetRecoveryConfig" }
func (*MsgSetRecoveryConfigResponse) XXX_MessageName() string    { return "legible.auth.v1.MsgSetRecoveryConfigResponse" }
func (*MsgInitiateRecovery) XXX_MessageName() string             { return "legible.auth.v1.MsgInitiateRecovery" }
func (*MsgInitiateRecoveryResponse) XXX_MessageName() string     { return "legible.auth.v1.MsgInitiateRecoveryResponse" }
func (*MsgSubmitRecoveryShard) XXX_MessageName() string          { return "legible.auth.v1.MsgSubmitRecoveryShard" }
func (*MsgSubmitRecoveryShardResponse) XXX_MessageName() string  { return "legible.auth.v1.MsgSubmitRecoveryShardResponse" }
func (*MsgChallengeRecovery) XXX_MessageName() string            { return "legible.auth.v1.MsgChallengeRecovery" }
func (*MsgChallengeRecoveryResponse) XXX_MessageName() string    { return "legible.auth.v1.MsgChallengeRecoveryResponse" }
func (*MsgExecuteRecovery) XXX_MessageName() string              { return "legible.auth.v1.MsgExecuteRecovery" }
func (*MsgExecuteRecoveryResponse) XXX_MessageName() string      { return "legible.auth.v1.MsgExecuteRecoveryResponse" }
