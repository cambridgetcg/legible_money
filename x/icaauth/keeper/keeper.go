package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/icaauth/types"
)

// Keeper manages the icaauth module state.
type Keeper struct {
	cdc           codec.Codec
	storeService  store.KVStoreService
	legibleAuth   types.LegibleAuthKeeper
	icaController types.ICAControllerKeeper
	authority     string
}

// NewKeeper creates a new Keeper instance.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	legibleAuth types.LegibleAuthKeeper,
	icaController types.ICAControllerKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		legibleAuth:   legibleAuth,
		icaController: icaController,
		authority:     authority,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// prefixEndBytes returns the end key for a prefix scan.
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}

// --- State Operations ---

// SetRecord stores an InterchainAccountRecord for a DID.
func (k Keeper) SetRecord(ctx sdk.Context, record types.InterchainAccountRecord) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&record)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal ICA record: %v", err))
	}
	if err := kvStore.Set(types.InterchainAccountKey(record.Did), bz); err != nil {
		panic(fmt.Sprintf("failed to store ICA record: %v", err))
	}
}

// GetRecord retrieves an InterchainAccountRecord for a DID.
func (k Keeper) GetRecord(ctx sdk.Context, did string) (types.InterchainAccountRecord, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.InterchainAccountKey(did))
	if err != nil || bz == nil {
		return types.InterchainAccountRecord{}, false
	}
	var record types.InterchainAccountRecord
	if err := proto.Unmarshal(bz, &record); err != nil {
		return types.InterchainAccountRecord{}, false
	}
	return record, true
}

// GetRemoteAccountsForDID returns all remote accounts for a DID.
func (k Keeper) GetRemoteAccountsForDID(ctx sdk.Context, did string) []*types.RemoteAccount {
	record, found := k.GetRecord(ctx, did)
	if !found {
		return nil
	}
	return record.Accounts
}

// AddRemoteAccount adds a remote account to a DID's record.
func (k Keeper) AddRemoteAccount(ctx sdk.Context, did string, account *types.RemoteAccount) error {
	params := k.GetParams(ctx)
	record, found := k.GetRecord(ctx, did)
	if !found {
		record = types.InterchainAccountRecord{
			Did:      did,
			Accounts: []*types.RemoteAccount{},
		}
	}

	if uint32(len(record.Accounts)) >= params.MaxRemoteAccountsPerDid {
		return types.ErrMaxAccountsReached
	}

	// Check for duplicate connection
	for _, existing := range record.Accounts {
		if existing.ConnectionId == account.ConnectionId {
			return types.ErrAlreadyRegistered
		}
	}

	record.Accounts = append(record.Accounts, account)
	k.SetRecord(ctx, record)
	return nil
}

// GetRemoteAccountByConnection returns the remote account for a DID on a specific connection.
func (k Keeper) GetRemoteAccountByConnection(ctx sdk.Context, did, connectionId string) (*types.RemoteAccount, bool) {
	record, found := k.GetRecord(ctx, did)
	if !found {
		return nil, false
	}
	for _, acc := range record.Accounts {
		if acc.ConnectionId == connectionId {
			return acc, true
		}
	}
	return nil, false
}

// UpdateRemoteAccountAddress sets the remote address once the ICA is registered on the host.
func (k Keeper) UpdateRemoteAccountAddress(ctx sdk.Context, did, connectionId, remoteAddress string) {
	record, found := k.GetRecord(ctx, did)
	if !found {
		return
	}
	for i, acc := range record.Accounts {
		if acc.ConnectionId == connectionId {
			record.Accounts[i].RemoteAddress = remoteAddress
			record.Accounts[i].Active = true
			break
		}
	}
	k.SetRecord(ctx, record)
}

// --- Params ---

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}
	return kvStore.Set(types.ParamsKey, bz)
}

// GetParams retrieves module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// --- Genesis ---

// InitGenesis initializes the module state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) error {
	if data.Params != nil {
		if err := k.SetParams(ctx, *data.Params); err != nil {
			return fmt.Errorf("failed to set params: %w", err)
		}
	}
	for _, record := range data.Records {
		if record != nil {
			k.SetRecord(ctx, *record)
		}
	}
	return nil
}

// ExportGenesis exports the module state for genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.InterchainAccountPrefix, prefixEndBytes(types.InterchainAccountPrefix))
	if err != nil {
		return &types.GenesisState{Params: &params}
	}
	defer iter.Close()

	var records []*types.InterchainAccountRecord
	for ; iter.Valid(); iter.Next() {
		var record types.InterchainAccountRecord
		if err := proto.Unmarshal(iter.Value(), &record); err != nil {
			continue
		}
		records = append(records, &record)
	}

	return &types.GenesisState{
		Params:  &params,
		Records: records,
	}
}
