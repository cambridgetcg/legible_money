package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/channels/types"
)

// --- Message Channel ID Counter ---

var msgChannelCounterKey = types.MsgChannelCounterKey

// GetNextMsgChannelId returns a unique message channel ID and increments the counter.
func (k Keeper) GetNextMsgChannelId(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(msgChannelCounterKey)
	var counter uint64
	if err == nil && bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	_ = store.Set(msgChannelCounterKey, newBz)
	return counter
}

// --- Message Channel CRUD ---

// SetMessageChannel stores a message channel and updates indexes.
func (k Keeper) SetMessageChannel(ctx sdk.Context, ch *types.MessageChannel) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(ch)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal message channel: %v", err))
	}
	if err := store.Set(types.MsgChannelKey(ch.ChannelId), bz); err != nil {
		panic(fmt.Sprintf("failed to set message channel: %v", err))
	}

	// Type index
	if err := store.Set(types.MsgChannelTypeKey(ch.ChannelType, ch.ChannelId), []byte(ch.ChannelId)); err != nil {
		panic(fmt.Sprintf("failed to set type index: %v", err))
	}

	// Member indexes
	for _, member := range ch.Members {
		if err := store.Set(types.MsgChannelMemberKey(member, ch.ChannelId), []byte(ch.ChannelId)); err != nil {
			panic(fmt.Sprintf("failed to set member index: %v", err))
		}
	}
}

// GetMessageChannel retrieves a message channel by ID.
func (k Keeper) GetMessageChannel(ctx sdk.Context, channelId string) (*types.MessageChannel, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.MsgChannelKey(channelId))
	if err != nil || bz == nil {
		return nil, false
	}
	var ch types.MessageChannel
	if err := json.Unmarshal(bz, &ch); err != nil {
		return nil, false
	}
	return &ch, true
}

// DeleteMessageChannel removes a message channel, its indexes, and all messages.
func (k Keeper) DeleteMessageChannel(ctx sdk.Context, ch *types.MessageChannel) {
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(types.MsgChannelKey(ch.ChannelId))
	_ = store.Delete(types.MsgChannelTypeKey(ch.ChannelType, ch.ChannelId))
	for _, member := range ch.Members {
		_ = store.Delete(types.MsgChannelMemberKey(member, ch.ChannelId))
	}
	// Delete all messages in channel
	k.deleteAllMessages(ctx, ch.ChannelId)
}

// --- Channel Creation ---

// CreateMessageChannel creates a new typed message channel.
// The creator is automatically added as the first member.
func (k Keeper) CreateMessageChannel(ctx sdk.Context, channelType, creator, name string, maxHistory uint64) (string, error) {
	if err := types.ValidateChannelType(channelType); err != nil {
		return "", types.ErrInvalidChannelType.Wrap(err.Error())
	}
	if _, err := sdk.AccAddressFromBech32(creator); err != nil {
		return "", fmt.Errorf("invalid creator address: %w", err)
	}
	if name == "" {
		return "", fmt.Errorf("channel name cannot be empty")
	}
	if maxHistory == 0 {
		maxHistory = types.DefaultMaxHistory
	}

	counter := k.GetNextMsgChannelId(ctx)
	channelId := fmt.Sprintf("mc-%d-%d", ctx.BlockHeight(), counter)

	ch := &types.MessageChannel{
		ChannelId:    channelId,
		ChannelType:  channelType,
		Creator:      creator,
		Name:         name,
		Members:      []string{creator},
		CreatedAt:    uint64(ctx.BlockHeight()),
		MaxHistory:   maxHistory,
		MessageCount: 0,
	}

	k.SetMessageChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"create_message_channel",
		sdk.NewAttribute("channel_id", channelId),
		sdk.NewAttribute("channel_type", channelType),
		sdk.NewAttribute("creator", creator),
		sdk.NewAttribute("name", name),
	))

	return channelId, nil
}

// --- Membership ---

// JoinChannel adds a member to a message channel.
func (k Keeper) JoinChannel(ctx sdk.Context, channelId, member string) error {
	if _, err := sdk.AccAddressFromBech32(member); err != nil {
		return fmt.Errorf("invalid member address: %w", err)
	}

	ch, found := k.GetMessageChannel(ctx, channelId)
	if !found {
		return types.ErrMsgChannelNotFound
	}

	// Check not already a member
	for _, m := range ch.Members {
		if m == member {
			return types.ErrAlreadyMember
		}
	}

	ch.Members = append(ch.Members, member)
	k.SetMessageChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"join_channel",
		sdk.NewAttribute("channel_id", channelId),
		sdk.NewAttribute("member", member),
	))

	return nil
}

// LeaveChannel removes a member from a message channel.
// The creator cannot leave (must delete the channel instead).
func (k Keeper) LeaveChannel(ctx sdk.Context, channelId, member string) error {
	ch, found := k.GetMessageChannel(ctx, channelId)
	if !found {
		return types.ErrMsgChannelNotFound
	}

	if member == ch.Creator {
		return types.ErrNotChannelCreator.Wrap("creator cannot leave; delete the channel instead")
	}

	// Remove member
	store := k.storeService.OpenKVStore(ctx)
	found = false
	newMembers := make([]string, 0, len(ch.Members))
	for _, m := range ch.Members {
		if m == member {
			found = true
			// Remove member index
			_ = store.Delete(types.MsgChannelMemberKey(member, channelId))
			continue
		}
		newMembers = append(newMembers, m)
	}

	if !found {
		return types.ErrNotMember
	}

	ch.Members = newMembers
	k.SetMessageChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"leave_channel",
		sdk.NewAttribute("channel_id", channelId),
		sdk.NewAttribute("member", member),
	))

	return nil
}

// GetChannelMembers returns the members of a message channel.
func (k Keeper) GetChannelMembers(ctx sdk.Context, channelId string) ([]string, error) {
	ch, found := k.GetMessageChannel(ctx, channelId)
	if !found {
		return nil, types.ErrMsgChannelNotFound
	}
	return ch.Members, nil
}

// isMember checks if an address is a member of a channel.
func isMember(ch *types.MessageChannel, addr string) bool {
	for _, m := range ch.Members {
		if m == addr {
			return true
		}
	}
	return false
}

// --- Message Send / Receive / Acknowledge ---

// SendMessage sends a message to a channel. Sender must be a member.
// Enforces FIFO eviction if message count exceeds MaxHistory.
func (k Keeper) SendMessage(ctx sdk.Context, channelId, sender, content string) (*types.ChannelMessage, error) {
	if content == "" {
		return nil, types.ErrEmptyContent
	}
	if _, err := sdk.AccAddressFromBech32(sender); err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	ch, found := k.GetMessageChannel(ctx, channelId)
	if !found {
		return nil, types.ErrMsgChannelNotFound
	}

	if !isMember(ch, sender) {
		return nil, types.ErrNotMember
	}

	ch.MessageCount++
	seq := ch.MessageCount

	msg := &types.ChannelMessage{
		MessageId: fmt.Sprintf("%s-msg-%d", channelId, seq),
		ChannelId: channelId,
		Sender:    sender,
		Content:   content,
		SentAt:    uint64(ctx.BlockHeight()),
		Sequence:  seq,
	}

	k.setMessage(ctx, msg)
	k.SetMessageChannel(ctx, ch)

	// FIFO eviction: if we have more messages than MaxHistory, evict oldest
	k.evictOldMessages(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"send_message",
		sdk.NewAttribute("channel_id", channelId),
		sdk.NewAttribute("message_id", msg.MessageId),
		sdk.NewAttribute("sender", sender),
		sdk.NewAttribute("sequence", fmt.Sprintf("%d", seq)),
	))

	return msg, nil
}

// AcknowledgeMessage marks a message as acknowledged by the given address.
// Acker must be a member of the channel.
func (k Keeper) AcknowledgeMessage(ctx sdk.Context, channelId, messageId, acker string) error {
	if _, err := sdk.AccAddressFromBech32(acker); err != nil {
		return fmt.Errorf("invalid acker address: %w", err)
	}

	ch, found := k.GetMessageChannel(ctx, channelId)
	if !found {
		return types.ErrMsgChannelNotFound
	}

	if !isMember(ch, acker) {
		return types.ErrNotMember
	}

	msg, found := k.GetMessage(ctx, channelId, messageId)
	if !found {
		return types.ErrMessageNotFound
	}

	// Check not already acked
	for _, a := range msg.AckedBy {
		if a == acker {
			return types.ErrAlreadyAcked
		}
	}

	msg.AckedBy = append(msg.AckedBy, acker)
	k.setMessage(ctx, msg)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"ack_message",
		sdk.NewAttribute("channel_id", channelId),
		sdk.NewAttribute("message_id", messageId),
		sdk.NewAttribute("acker", acker),
	))

	return nil
}

// --- Message Storage ---

// setMessage stores a channel message.
func (k Keeper) setMessage(ctx sdk.Context, msg *types.ChannelMessage) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(msg)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal channel message: %v", err))
	}
	if err := store.Set(types.ChannelMsgKey(msg.ChannelId, msg.Sequence), bz); err != nil {
		panic(fmt.Sprintf("failed to set channel message: %v", err))
	}
}

// GetMessage retrieves a specific message by channel ID and message ID.
// Scans messages in the channel to find by ID.
func (k Keeper) GetMessage(ctx sdk.Context, channelId, messageId string) (*types.ChannelMessage, bool) {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.ChannelMsgPrefix(channelId)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil, false
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var msg types.ChannelMessage
		if err := json.Unmarshal(iter.Value(), &msg); err != nil {
			continue
		}
		if msg.MessageId == messageId {
			return &msg, true
		}
	}
	return nil, false
}

// GetMessageHistory returns the most recent messages in a channel, up to limit.
// Messages are returned in chronological order (oldest first).
func (k Keeper) GetMessageHistory(ctx sdk.Context, channelId string, limit uint64) []*types.ChannelMessage {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.ChannelMsgPrefix(channelId)

	// Collect all messages first (they're already in sequence order due to key layout)
	var all []*types.ChannelMessage
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var msg types.ChannelMessage
		if err := json.Unmarshal(iter.Value(), &msg); err != nil {
			continue
		}
		all = append(all, &msg)
	}

	if limit > 0 && uint64(len(all)) > limit {
		all = all[uint64(len(all))-limit:]
	}

	return all
}

// evictOldMessages removes oldest messages when count exceeds MaxHistory.
func (k Keeper) evictOldMessages(ctx sdk.Context, ch *types.MessageChannel) {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.ChannelMsgPrefix(ch.ChannelId)

	// Count current messages
	var keys [][]byte
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	for ; iter.Valid(); iter.Next() {
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())
		keys = append(keys, key)
	}
	iter.Close()

	// Evict oldest (keys are sorted by sequence due to big-endian encoding)
	if uint64(len(keys)) > ch.MaxHistory {
		toEvict := uint64(len(keys)) - ch.MaxHistory
		for i := uint64(0); i < toEvict; i++ {
			_ = store.Delete(keys[i])
		}
	}
}

// deleteAllMessages removes all messages for a channel.
func (k Keeper) deleteAllMessages(ctx sdk.Context, channelId string) {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.ChannelMsgPrefix(channelId)

	var keys [][]byte
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	for ; iter.Valid(); iter.Next() {
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())
		keys = append(keys, key)
	}
	iter.Close()

	for _, key := range keys {
		_ = store.Delete(key)
	}
}

// --- Query Helpers ---

// ListChannelsByType returns all message channels of a given type.
func (k Keeper) ListChannelsByType(ctx sdk.Context, channelType string) []*types.MessageChannel {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.MsgChannelTypeKeyPrefix(channelType)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var channels []*types.MessageChannel
	for ; iter.Valid(); iter.Next() {
		channelId := string(iter.Value())
		ch, found := k.GetMessageChannel(ctx, channelId)
		if found {
			channels = append(channels, ch)
		}
	}
	return channels
}

// ListChannelsByMember returns all message channels a given address belongs to.
func (k Keeper) ListChannelsByMember(ctx sdk.Context, member string) []*types.MessageChannel {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.MsgChannelMemberKeyPrefix(member)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var channels []*types.MessageChannel
	for ; iter.Valid(); iter.Next() {
		channelId := string(iter.Value())
		ch, found := k.GetMessageChannel(ctx, channelId)
		if found {
			channels = append(channels, ch)
		}
	}
	return channels
}

// IterateMessageChannels iterates over all message channels. Return true from cb to stop.
func (k Keeper) IterateMessageChannels(ctx sdk.Context, cb func(*types.MessageChannel) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.MsgChannelKeyPrefix, prefixEndBytes(types.MsgChannelKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var ch types.MessageChannel
		if err := json.Unmarshal(iter.Value(), &ch); err != nil {
			continue
		}
		if cb(&ch) {
			break
		}
	}
}
