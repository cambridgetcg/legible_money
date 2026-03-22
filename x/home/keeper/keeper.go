package keeper

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/home/types"
)

// Keeper manages the home module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService
	bankKeeper   types.BankKeeper

	// Cross-module keepers (set via setter to avoid circular deps)
	partnershipKeeper types.PartnershipKeeper
	knowledgeKeeper   types.KnowledgeKeeper
	scheduleKeeper    types.ScheduleKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new home module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		bankKeeper:   bankKeeper,
		authority:    authority,
	}
}

// prefixEndBytes returns the end key for a prefix scan (exclusive upper bound).
// It increments the last byte of the prefix; if it overflows, it tries the
// previous byte, and so on. Returns nil if all bytes overflow (prefix is all 0xff).
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

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// SetPartnershipKeeper sets the partnerships module keeper reference.
func (k *Keeper) SetPartnershipKeeper(pk types.PartnershipKeeper) {
	k.partnershipKeeper = pk
}

// SetKnowledgeKeeper sets the knowledge module keeper reference.
func (k *Keeper) SetKnowledgeKeeper(kk types.KnowledgeKeeper) {
	k.knowledgeKeeper = kk
}

// SetScheduleKeeper sets the schedule module keeper reference.
func (k *Keeper) SetScheduleKeeper(sk types.ScheduleKeeper) {
	k.scheduleKeeper = sk
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) {
	if gs.Params != nil {
		k.SetParams(ctx, gs.Params)
	} else {
		dp := types.DefaultParams()
		k.SetParams(ctx, &dp)
	}

	for _, home := range gs.Homes {
		if home != nil {
			k.SetHome(ctx, home)
		}
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var homes []*types.AgentHome
	k.IterateHomes(ctx, func(home *types.AgentHome) bool {
		homes = append(homes, home)
		return false
	})

	// Export all alerts across homes
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.AlertKeyPrefix, prefixEndBytes(types.AlertKeyPrefix))
	if err == nil {
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			// Alerts are not included in genesis for simplicity (ephemeral state).
			// Include only homes and params.
		}
	}

	params := k.GetParams(ctx)
	return &types.GenesisState{
		Params: params,
		Homes:  homes,
	}
}

// --- Params ---

// GetParams returns the module parameters.
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil || bz == nil {
		dp := types.DefaultParams()
		return &dp
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		dp := types.DefaultParams()
		return &dp
	}
	return &params
}

// SetParams sets the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	if err := store.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set params: %v", err))
	}
}

// --- Memory Encryption (AES-256-GCM) ---

// EncryptMemoryCID encrypts a plaintext CID using AES-256-GCM.
// Key must be 32 bytes. The agent holds the key off-chain; it is never stored on-chain.
// Returns base64-encoded nonce || ciphertext.
func (k Keeper) EncryptMemoryCID(_ sdk.Context, homeID string, plainCID string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", types.ErrEncryptionFailed.Wrap("key must be 32 bytes (AES-256)")
	}
	if plainCID == "" {
		return "", types.ErrEncryptionFailed.Wrap("plainCID cannot be empty")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", types.ErrEncryptionFailed.Wrapf("aes.NewCipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", types.ErrEncryptionFailed.Wrapf("cipher.NewGCM: %v", err)
	}

	nonce := make([]byte, gcm.NonceSize()) // 12 bytes
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", types.ErrEncryptionFailed.Wrapf("nonce generation: %v", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plainCID), nil)

	// nonce || ciphertext
	result := make([]byte, len(nonce)+len(ciphertext))
	copy(result, nonce)
	copy(result[len(nonce):], ciphertext)

	return base64.StdEncoding.EncodeToString(result), nil
}

// DecryptMemoryCID decrypts a base64-encoded encrypted CID using AES-256-GCM.
// Input is base64(nonce || ciphertext). Key must be 32 bytes.
func (k Keeper) DecryptMemoryCID(_ sdk.Context, homeID string, encryptedCID string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", types.ErrDecryptionFailed.Wrap("key must be 32 bytes (AES-256)")
	}
	if encryptedCID == "" {
		return "", types.ErrDecryptionFailed.Wrap("encryptedCID cannot be empty")
	}

	data, err := base64.StdEncoding.DecodeString(encryptedCID)
	if err != nil {
		return "", types.ErrDecryptionFailed.Wrapf("base64 decode: %v", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", types.ErrDecryptionFailed.Wrapf("aes.NewCipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", types.ErrDecryptionFailed.Wrapf("cipher.NewGCM: %v", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", types.ErrDecryptionFailed.Wrap("ciphertext too short")
	}

	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", types.ErrDecryptionFailed.Wrapf("decryption: %v", err)
	}

	return string(plaintext), nil
}

// TrackContribution records a knowledge contribution event for a home.
// Stores a ContributionEvent keyed by homeId + claimId.
// No-op if the home doesn't exist.
func (k Keeper) TrackContribution(ctx sdk.Context, homeID, claimID, claimType string) error {
	if homeID == "" || claimID == "" {
		return fmt.Errorf("homeID and claimID must be non-empty")
	}

	// Verify home exists
	if _, found := k.GetHome(ctx, homeID); !found {
		return types.ErrHomeNotFound.Wrapf("home %s not found", homeID)
	}

	event := types.ContributionEvent{
		ClaimID:     claimID,
		ClaimType:   claimType,
		BlockHeight: ctx.BlockHeight(),
	}

	bz, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal contribution event: %w", err)
	}

	kvStore := k.storeService.OpenKVStore(ctx)
	return kvStore.Set(types.ContributionKey(homeID, claimID), bz)
}

// GetContributions returns all contribution events for a home.
func (k Keeper) GetContributions(ctx sdk.Context, homeID string) []types.ContributionEvent {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.ContributionPrefixKey(homeID)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var events []types.ContributionEvent
	for ; iter.Valid(); iter.Next() {
		var event types.ContributionEvent
		if err := json.Unmarshal(iter.Value(), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	return events
}
