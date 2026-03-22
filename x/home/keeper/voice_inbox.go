package keeper

import (
	"encoding/json"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/types"
)

// SendMessage delivers a message to another home's inbox.
// Messages expire after ttlBlocks from the current block height.
// If the inbox is full (MaxInboxMessages), the oldest message is evicted.
func (k Keeper) SendMessage(ctx sdk.Context, fromHomeID, toHomeID, subject, body string, ttlBlocks uint64) (string, error) {
	if fromHomeID == "" || toHomeID == "" {
		return "", fmt.Errorf("fromHomeID and toHomeID must be non-empty")
	}
	if subject == "" {
		return "", fmt.Errorf("subject must be non-empty")
	}
	if ttlBlocks == 0 {
		return "", fmt.Errorf("ttlBlocks must be positive")
	}
	if _, found := k.GetHome(ctx, fromHomeID); !found {
		return "", types.ErrHomeNotFound.Wrapf("sender home %s not found", fromHomeID)
	}
	if _, found := k.GetHome(ctx, toHomeID); !found {
		return "", types.ErrHomeNotFound.Wrapf("recipient home %s not found", toHomeID)
	}

	height := uint64(ctx.BlockHeight())
	messageID := fmt.Sprintf("msg-%s-%d", fromHomeID, height)

	// Evict oldest if inbox is full.
	inbox := k.GetInbox(ctx, toHomeID)
	if len(inbox) >= types.MaxInboxMessages {
		// allInboxMessages includes expired; we need raw scan for eviction.
		allMsgs := k.allInboxMessages(ctx, toHomeID)
		if len(allMsgs) >= types.MaxInboxMessages {
			oldest := allMsgs[0]
			for _, m := range allMsgs[1:] {
				if m.SentAt < oldest.SentAt || (m.SentAt == oldest.SentAt && m.MessageID < oldest.MessageID) {
					oldest = m
				}
			}
			store := k.storeService.OpenKVStore(ctx)
			_ = store.Delete(types.InboxMessageKey(toHomeID, oldest.MessageID))
		}
	}

	msg := types.InboxMessage{
		MessageID:  messageID,
		FromHomeID: fromHomeID,
		ToHomeID:   toHomeID,
		Subject:    subject,
		Body:       body,
		SentAt:     height,
		TTLBlocks:  ttlBlocks,
		Read:       false,
	}

	bz, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal inbox message: %w", err)
	}

	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(types.InboxMessageKey(toHomeID, messageID), bz); err != nil {
		return "", fmt.Errorf("failed to store inbox message: %w", err)
	}

	return messageID, nil
}

// GetInbox returns all non-expired messages for a home, newest first.
func (k Keeper) GetInbox(ctx sdk.Context, homeID string) []types.InboxMessage {
	height := uint64(ctx.BlockHeight())
	all := k.allInboxMessages(ctx, homeID)

	var live []types.InboxMessage
	for _, m := range all {
		if m.SentAt+m.TTLBlocks >= height {
			live = append(live, m)
		}
	}

	// Sort newest first; deterministic tiebreaker on MessageID.
	sort.Slice(live, func(i, j int) bool {
		if live[i].SentAt != live[j].SentAt {
			return live[i].SentAt > live[j].SentAt
		}
		return live[i].MessageID > live[j].MessageID
	})

	return live
}

// ReadMessage marks a message as read and returns it.
func (k Keeper) ReadMessage(ctx sdk.Context, homeID, messageID string) (types.InboxMessage, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.InboxMessageKey(homeID, messageID))
	if err != nil || bz == nil {
		return types.InboxMessage{}, false
	}

	var msg types.InboxMessage
	if err := json.Unmarshal(bz, &msg); err != nil {
		return types.InboxMessage{}, false
	}

	// Check expiry.
	height := uint64(ctx.BlockHeight())
	if msg.SentAt+msg.TTLBlocks < height {
		return types.InboxMessage{}, false
	}

	if !msg.Read {
		msg.Read = true
		updated, _ := json.Marshal(msg)
		_ = store.Set(types.InboxMessageKey(homeID, messageID), updated)
	}

	return msg, true
}

// PurgeExpired removes expired messages from a home's inbox.
// Called from BeginBlocker.
func (k Keeper) PurgeExpired(ctx sdk.Context, homeID string) {
	height := uint64(ctx.BlockHeight())
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.InboxPrefixKey(homeID)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	defer iter.Close()

	var toDelete [][]byte
	for ; iter.Valid(); iter.Next() {
		var msg types.InboxMessage
		if err := json.Unmarshal(iter.Value(), &msg); err != nil {
			continue
		}
		if msg.SentAt+msg.TTLBlocks < height {
			key := make([]byte, len(iter.Key()))
			copy(key, iter.Key())
			toDelete = append(toDelete, key)
		}
	}

	for _, key := range toDelete {
		_ = store.Delete(key)
	}
}

// allInboxMessages returns all messages (including expired) for a home.
func (k Keeper) allInboxMessages(ctx sdk.Context, homeID string) []types.InboxMessage {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.InboxPrefixKey(homeID)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var msgs []types.InboxMessage
	for ; iter.Valid(); iter.Next() {
		var msg types.InboxMessage
		if err := json.Unmarshal(iter.Value(), &msg); err != nil {
			continue
		}
		msgs = append(msgs, msg)
	}
	return msgs
}
