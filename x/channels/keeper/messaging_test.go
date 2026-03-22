package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/channels/keeper"
	"github.com/legible-money/legible/x/channels/types"
)

// ==================== Message Channel Creation ====================

func TestCreateMessageChannel_Success(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, err := k.CreateMessageChannel(ctx, types.ChannelTypeKnowledge, payer1, "test-knowledge", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if channelId == "" {
		t.Fatal("expected non-empty channel ID")
	}

	ch, found := k.GetMessageChannel(ctx, channelId)
	if !found {
		t.Fatal("channel not found after creation")
	}
	if ch.ChannelType != types.ChannelTypeKnowledge {
		t.Errorf("expected knowledge type, got %s", ch.ChannelType)
	}
	if ch.Creator != payer1 {
		t.Errorf("expected creator %s, got %s", payer1, ch.Creator)
	}
	if ch.Name != "test-knowledge" {
		t.Errorf("expected name test-knowledge, got %s", ch.Name)
	}
	if len(ch.Members) != 1 || ch.Members[0] != payer1 {
		t.Errorf("expected creator as sole member, got %v", ch.Members)
	}
	if ch.MaxHistory != 100 {
		t.Errorf("expected max history 100, got %d", ch.MaxHistory)
	}
	if ch.CreatedAt != uint64(ctx.BlockHeight()) {
		t.Errorf("expected created at block %d, got %d", ctx.BlockHeight(), ch.CreatedAt)
	}
}

func TestCreateMessageChannel_AllTypes(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	for _, ct := range []string{types.ChannelTypeKnowledge, types.ChannelTypeGovernance, types.ChannelTypeSocial, types.ChannelTypeTrade} {
		_, err := k.CreateMessageChannel(ctx, ct, payer1, "channel-"+ct, 50)
		if err != nil {
			t.Errorf("failed to create %s channel: %v", ct, err)
		}
	}

	// Verify ListChannelsByType
	knowledge := k.ListChannelsByType(ctx, types.ChannelTypeKnowledge)
	if len(knowledge) != 1 {
		t.Errorf("expected 1 knowledge channel, got %d", len(knowledge))
	}
}

func TestCreateMessageChannel_InvalidType(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	_, err := k.CreateMessageChannel(ctx, "invalid", payer1, "bad-channel", 100)
	if err == nil {
		t.Error("expected error for invalid channel type")
	}
}

func TestCreateMessageChannel_EmptyName(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	_, err := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "", 100)
	if err == nil {
		t.Error("expected error for empty channel name")
	}
}

func TestCreateMessageChannel_DefaultMaxHistory(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, err := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "default-history", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ch, _ := k.GetMessageChannel(ctx, channelId)
	if ch.MaxHistory != types.DefaultMaxHistory {
		t.Errorf("expected default max history %d, got %d", types.DefaultMaxHistory, ch.MaxHistory)
	}
}

func TestCreateMessageChannel_UniqueIDs(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	ids := make(map[string]bool)
	for i := 0; i < 5; i++ {
		id, err := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "ch", 50)
		if err != nil {
			t.Fatalf("create channel %d: %v", i, err)
		}
		if ids[id] {
			t.Errorf("duplicate channel ID: %s", id)
		}
		ids[id] = true
	}
}

// ==================== Channel CRUD ====================

func TestGetMessageChannel_NotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	_, found := k.GetMessageChannel(ctx, "nonexistent")
	if found {
		t.Error("expected channel not found")
	}
}

func TestDeleteMessageChannel(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeTrade, payer1, "deleteme", 50)
	ch, _ := k.GetMessageChannel(ctx, channelId)

	// Send a message first
	k.SendMessage(ctx, channelId, payer1, "hello")

	k.DeleteMessageChannel(ctx, ch)

	_, found := k.GetMessageChannel(ctx, channelId)
	if found {
		t.Error("channel should have been deleted")
	}

	// Messages should also be deleted
	msgs := k.GetMessageHistory(ctx, channelId, 100)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after delete, got %d", len(msgs))
	}

	// Type index should be cleared
	channels := k.ListChannelsByType(ctx, types.ChannelTypeTrade)
	if len(channels) != 0 {
		t.Errorf("expected 0 trade channels after delete, got %d", len(channels))
	}
}

// ==================== Membership ====================

func TestJoinChannel_Success(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "social-1", 50)

	err := k.JoinChannel(ctx, channelId, provider1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	members, _ := k.GetChannelMembers(ctx, channelId)
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}

	// Verify member index
	channels := k.ListChannelsByMember(ctx, provider1)
	if len(channels) != 1 {
		t.Errorf("expected 1 channel for provider1, got %d", len(channels))
	}
}

func TestJoinChannel_AlreadyMember(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "social-1", 50)

	err := k.JoinChannel(ctx, channelId, payer1)
	if err == nil {
		t.Error("expected already member error")
	}
}

func TestJoinChannel_NotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	err := k.JoinChannel(ctx, "nonexistent", provider1)
	if err == nil {
		t.Error("expected not found error")
	}
}

func TestLeaveChannel_Success(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "social-1", 50)
	k.JoinChannel(ctx, channelId, provider1)
	k.JoinChannel(ctx, channelId, payer2)

	err := k.LeaveChannel(ctx, channelId, provider1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	members, _ := k.GetChannelMembers(ctx, channelId)
	if len(members) != 2 {
		t.Errorf("expected 2 members after leave, got %d", len(members))
	}

	// Verify member index cleared
	channels := k.ListChannelsByMember(ctx, provider1)
	if len(channels) != 0 {
		t.Errorf("expected 0 channels for provider1 after leave, got %d", len(channels))
	}
}

func TestLeaveChannel_CreatorCannotLeave(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "social-1", 50)

	err := k.LeaveChannel(ctx, channelId, payer1)
	if err == nil {
		t.Error("expected error: creator cannot leave")
	}
}

func TestLeaveChannel_NotMember(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "social-1", 50)

	err := k.LeaveChannel(ctx, channelId, provider2)
	if err == nil {
		t.Error("expected not member error")
	}
}

// ==================== Send Message ====================

func TestSendMessage_Success(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeKnowledge, payer1, "knowledge-1", 100)
	k.JoinChannel(ctx, channelId, provider1)

	msg, err := k.SendMessage(ctx, channelId, payer1, "Hello from payer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Sender != payer1 {
		t.Errorf("expected sender %s, got %s", payer1, msg.Sender)
	}
	if msg.Content != "Hello from payer" {
		t.Errorf("expected content 'Hello from payer', got %s", msg.Content)
	}
	if msg.Sequence != 1 {
		t.Errorf("expected sequence 1, got %d", msg.Sequence)
	}
	if msg.SentAt != uint64(ctx.BlockHeight()) {
		t.Errorf("expected sent at %d, got %d", ctx.BlockHeight(), msg.SentAt)
	}

	// Second message
	msg2, _ := k.SendMessage(ctx, channelId, provider1, "Reply from provider")
	if msg2.Sequence != 2 {
		t.Errorf("expected sequence 2, got %d", msg2.Sequence)
	}

	// Verify message count on channel
	ch, _ := k.GetMessageChannel(ctx, channelId)
	if ch.MessageCount != 2 {
		t.Errorf("expected message count 2, got %d", ch.MessageCount)
	}
}

func TestSendMessage_NotMember(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeKnowledge, payer1, "knowledge-1", 100)

	_, err := k.SendMessage(ctx, channelId, provider1, "I'm not a member")
	if err == nil {
		t.Error("expected not member error")
	}
}

func TestSendMessage_EmptyContent(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "social-1", 100)

	_, err := k.SendMessage(ctx, channelId, payer1, "")
	if err == nil {
		t.Error("expected empty content error")
	}
}

func TestSendMessage_ChannelNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	_, err := k.SendMessage(ctx, "nonexistent", payer1, "hello")
	if err == nil {
		t.Error("expected channel not found error")
	}
}

// ==================== Acknowledge Message ====================

func TestAcknowledgeMessage_Success(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "social-1", 100)
	k.JoinChannel(ctx, channelId, provider1)

	msg, _ := k.SendMessage(ctx, channelId, payer1, "ack this")

	err := k.AcknowledgeMessage(ctx, channelId, msg.MessageId, provider1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify ack
	updated, found := k.GetMessage(ctx, channelId, msg.MessageId)
	if !found {
		t.Fatal("message not found after ack")
	}
	if len(updated.AckedBy) != 1 || updated.AckedBy[0] != provider1 {
		t.Errorf("expected acked by provider1, got %v", updated.AckedBy)
	}
}

func TestAcknowledgeMessage_AlreadyAcked(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "social-1", 100)
	k.JoinChannel(ctx, channelId, provider1)

	msg, _ := k.SendMessage(ctx, channelId, payer1, "ack this")
	k.AcknowledgeMessage(ctx, channelId, msg.MessageId, provider1)

	err := k.AcknowledgeMessage(ctx, channelId, msg.MessageId, provider1)
	if err == nil {
		t.Error("expected already acked error")
	}
}

func TestAcknowledgeMessage_NotMember(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "social-1", 100)
	msg, _ := k.SendMessage(ctx, channelId, payer1, "ack this")

	err := k.AcknowledgeMessage(ctx, channelId, msg.MessageId, provider1) // not a member
	if err == nil {
		t.Error("expected not member error")
	}
}

func TestAcknowledgeMessage_NotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "social-1", 100)

	err := k.AcknowledgeMessage(ctx, channelId, "nonexistent-msg", payer1)
	if err == nil {
		t.Error("expected message not found error")
	}
}

func TestAcknowledgeMessage_MultipleAckers(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "social-1", 100)
	k.JoinChannel(ctx, channelId, provider1)
	k.JoinChannel(ctx, channelId, payer2)

	msg, _ := k.SendMessage(ctx, channelId, payer1, "ack this")

	k.AcknowledgeMessage(ctx, channelId, msg.MessageId, provider1)
	k.AcknowledgeMessage(ctx, channelId, msg.MessageId, payer2)

	updated, _ := k.GetMessage(ctx, channelId, msg.MessageId)
	if len(updated.AckedBy) != 2 {
		t.Errorf("expected 2 ackers, got %d", len(updated.AckedBy))
	}
}

// ==================== Message History ====================

func TestGetMessageHistory_Ordered(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "social-1", 100)

	for i := 1; i <= 5; i++ {
		k.SendMessage(ctx, channelId, payer1, "msg")
	}

	history := k.GetMessageHistory(ctx, channelId, 100)
	if len(history) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(history))
	}

	// Verify chronological order
	for i := 0; i < len(history)-1; i++ {
		if history[i].Sequence >= history[i+1].Sequence {
			t.Errorf("messages not in chronological order: seq %d >= %d", history[i].Sequence, history[i+1].Sequence)
		}
	}
}

func TestGetMessageHistory_WithLimit(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "social-1", 100)

	for i := 1; i <= 10; i++ {
		k.SendMessage(ctx, channelId, payer1, "msg")
	}

	// Get last 3
	history := k.GetMessageHistory(ctx, channelId, 3)
	if len(history) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(history))
	}
	// Should be the last 3 (seq 8, 9, 10)
	if history[0].Sequence != 8 {
		t.Errorf("expected first message seq 8, got %d", history[0].Sequence)
	}
	if history[2].Sequence != 10 {
		t.Errorf("expected last message seq 10, got %d", history[2].Sequence)
	}
}

func TestGetMessageHistory_Empty(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "social-1", 100)

	history := k.GetMessageHistory(ctx, channelId, 100)
	if len(history) != 0 {
		t.Errorf("expected 0 messages, got %d", len(history))
	}
}

// ==================== FIFO Eviction ====================

func TestFIFOEviction(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create channel with max 5 messages
	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "fifo-test", 5)

	// Send 8 messages — first 3 should be evicted
	for i := 1; i <= 8; i++ {
		k.SendMessage(ctx, channelId, payer1, "msg")
	}

	history := k.GetMessageHistory(ctx, channelId, 100)
	if len(history) != 5 {
		t.Fatalf("expected 5 messages after eviction, got %d", len(history))
	}

	// Oldest should be seq 4 (1-3 evicted)
	if history[0].Sequence != 4 {
		t.Errorf("expected oldest message seq 4, got %d", history[0].Sequence)
	}
	// Newest should be seq 8
	if history[4].Sequence != 8 {
		t.Errorf("expected newest message seq 8, got %d", history[4].Sequence)
	}
}

func TestFIFOEviction_ExactBoundary(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "boundary", 3)

	// Send exactly 3 — no eviction
	for i := 1; i <= 3; i++ {
		k.SendMessage(ctx, channelId, payer1, "msg")
	}

	history := k.GetMessageHistory(ctx, channelId, 100)
	if len(history) != 3 {
		t.Errorf("expected 3 messages, got %d", len(history))
	}

	// Send 1 more — evicts the oldest
	k.SendMessage(ctx, channelId, payer1, "msg")

	history = k.GetMessageHistory(ctx, channelId, 100)
	if len(history) != 3 {
		t.Errorf("expected 3 messages after eviction, got %d", len(history))
	}
	if history[0].Sequence != 2 {
		t.Errorf("expected oldest seq 2 after eviction, got %d", history[0].Sequence)
	}
}

// ==================== List Queries ====================

func TestListChannelsByType(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.CreateMessageChannel(ctx, types.ChannelTypeKnowledge, payer1, "k1", 50)
	k.CreateMessageChannel(ctx, types.ChannelTypeKnowledge, payer1, "k2", 50)
	k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "s1", 50)

	knowledge := k.ListChannelsByType(ctx, types.ChannelTypeKnowledge)
	if len(knowledge) != 2 {
		t.Errorf("expected 2 knowledge channels, got %d", len(knowledge))
	}

	social := k.ListChannelsByType(ctx, types.ChannelTypeSocial)
	if len(social) != 1 {
		t.Errorf("expected 1 social channel, got %d", len(social))
	}

	trade := k.ListChannelsByType(ctx, types.ChannelTypeTrade)
	if len(trade) != 0 {
		t.Errorf("expected 0 trade channels, got %d", len(trade))
	}
}

func TestListChannelsByMember(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	id1, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "ch1", 50)
	id2, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "ch2", 50)
	k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer2, "ch3", 50) // payer2 is creator

	k.JoinChannel(ctx, id1, provider1)
	k.JoinChannel(ctx, id2, provider1)

	// payer1 is member of ch1 and ch2 (creator)
	p1channels := k.ListChannelsByMember(ctx, payer1)
	if len(p1channels) != 2 {
		t.Errorf("expected 2 channels for payer1, got %d", len(p1channels))
	}

	// provider1 joined ch1 and ch2
	pr1channels := k.ListChannelsByMember(ctx, provider1)
	if len(pr1channels) != 2 {
		t.Errorf("expected 2 channels for provider1, got %d", len(pr1channels))
	}

	// payer2 created ch3 only
	p2channels := k.ListChannelsByMember(ctx, payer2)
	if len(p2channels) != 1 {
		t.Errorf("expected 1 channel for payer2, got %d", len(p2channels))
	}
}

// ==================== IterateMessageChannels ====================

func TestIterateMessageChannels(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "ch1", 50)
	k.CreateMessageChannel(ctx, types.ChannelTypeKnowledge, payer1, "ch2", 50)
	k.CreateMessageChannel(ctx, types.ChannelTypeTrade, payer2, "ch3", 50)

	var count int
	k.IterateMessageChannels(ctx, func(ch *types.MessageChannel) bool {
		count++
		return false
	})
	if count != 3 {
		t.Errorf("expected 3 channels, got %d", count)
	}
}

func TestIterateMessageChannels_EarlyStop(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "ch1", 50)
	k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "ch2", 50)
	k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "ch3", 50)

	var count int
	k.IterateMessageChannels(ctx, func(ch *types.MessageChannel) bool {
		count++
		return count >= 2 // stop after 2
	})
	if count != 2 {
		t.Errorf("expected early stop at 2, got %d", count)
	}
}

// ==================== GetMessage ====================

func TestGetMessage_Success(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "ch1", 50)
	sent, _ := k.SendMessage(ctx, channelId, payer1, "find me")

	got, found := k.GetMessage(ctx, channelId, sent.MessageId)
	if !found {
		t.Fatal("message not found")
	}
	if got.Content != "find me" {
		t.Errorf("expected content 'find me', got %s", got.Content)
	}
}

func TestGetMessage_NotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "ch1", 50)

	_, found := k.GetMessage(ctx, channelId, "nonexistent-msg")
	if found {
		t.Error("expected message not found")
	}
}

// ==================== Edge Cases ====================

func TestSendMessage_AfterMemberLeaves(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "social-1", 100)
	k.JoinChannel(ctx, channelId, provider1)

	// Provider sends a message (success)
	_, err := k.SendMessage(ctx, channelId, provider1, "hello")
	if err != nil {
		t.Fatalf("should succeed: %v", err)
	}

	// Provider leaves
	k.LeaveChannel(ctx, channelId, provider1)

	// Provider tries to send again (should fail)
	_, err = k.SendMessage(ctx, channelId, provider1, "can't send")
	if err == nil {
		t.Error("expected error: left member should not be able to send")
	}
}

func TestMultipleChannels_Isolation(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	id1, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "ch1", 50)
	id2, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "ch2", 50)

	// Send messages to ch1
	k.SendMessage(ctx, id1, payer1, "msg in ch1 #1")
	k.SendMessage(ctx, id1, payer1, "msg in ch1 #2")

	// Send messages to ch2
	k.SendMessage(ctx, id2, payer1, "msg in ch2 #1")

	// Verify isolation
	h1 := k.GetMessageHistory(ctx, id1, 100)
	if len(h1) != 2 {
		t.Errorf("expected 2 messages in ch1, got %d", len(h1))
	}

	h2 := k.GetMessageHistory(ctx, id2, 100)
	if len(h2) != 1 {
		t.Errorf("expected 1 message in ch2, got %d", len(h2))
	}
}

func TestMsgChannelIdCounter(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	id1 := k.GetNextMsgChannelId(ctx)
	id2 := k.GetNextMsgChannelId(ctx)
	id3 := k.GetNextMsgChannelId(ctx)

	if id1 != 1 || id2 != 2 || id3 != 3 {
		t.Errorf("expected 1,2,3, got %d,%d,%d", id1, id2, id3)
	}
}

// ==================== OpenClaw Adversarial Probes ====================

// OC-MCH-1: Non-member sends message
func TestOC_MCH1_NonMemberSend(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "guarded", 50)

	_, err := k.SendMessage(ctx, channelId, provider1, "sneak in")
	if err == nil {
		t.Fatal("OC-MCH-1 FAIL: non-member was able to send message")
	}

	msgs := k.GetMessageHistory(ctx, channelId, 100)
	if len(msgs) != 0 {
		t.Fatal("OC-MCH-1 FAIL: message stored from non-member")
	}
}

// OC-MCH-2: Non-member acknowledges message
func TestOC_MCH2_NonMemberAck(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "guarded", 50)
	msg, _ := k.SendMessage(ctx, channelId, payer1, "ack me")

	err := k.AcknowledgeMessage(ctx, channelId, msg.MessageId, provider1) // not member
	if err == nil {
		t.Fatal("OC-MCH-2 FAIL: non-member was able to acknowledge message")
	}
}

// OC-MCH-3: Creator tries to leave (should be rejected)
func TestOC_MCH3_CreatorLeave(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "owned", 50)
	k.JoinChannel(ctx, channelId, provider1)

	err := k.LeaveChannel(ctx, channelId, payer1)
	if err == nil {
		t.Fatal("OC-MCH-3 FAIL: creator was able to leave channel")
	}

	// Channel should still have creator
	ch, _ := k.GetMessageChannel(ctx, channelId)
	if ch.Members[0] != payer1 {
		t.Fatal("OC-MCH-3 FAIL: creator removed from members")
	}
}

// OC-MCH-4: Double join
func TestOC_MCH4_DoubleJoin(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "no-dupes", 50)
	k.JoinChannel(ctx, channelId, provider1)

	err := k.JoinChannel(ctx, channelId, provider1)
	if err == nil {
		t.Fatal("OC-MCH-4 FAIL: duplicate join accepted")
	}

	members, _ := k.GetChannelMembers(ctx, channelId)
	if len(members) != 2 {
		t.Errorf("OC-MCH-4 FAIL: expected 2 members, got %d", len(members))
	}
}

// OC-MCH-5: FIFO eviction preserves newest messages
func TestOC_MCH5_FIFOPreservesNewest(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	channelId, _ := k.CreateMessageChannel(ctx, types.ChannelTypeSocial, payer1, "fifo", 3)

	// Send 10 messages
	for i := 1; i <= 10; i++ {
		k.SendMessage(ctx, channelId, payer1, "msg")
	}

	history := k.GetMessageHistory(ctx, channelId, 100)
	if len(history) != 3 {
		t.Fatalf("OC-MCH-5 FAIL: expected 3 messages, got %d", len(history))
	}

	// Must be sequences 8, 9, 10
	if history[0].Sequence != 8 || history[1].Sequence != 9 || history[2].Sequence != 10 {
		t.Errorf("OC-MCH-5 FAIL: wrong sequences: %d, %d, %d",
			history[0].Sequence, history[1].Sequence, history[2].Sequence)
	}
}

// OC-MCH-6: Invalid channel type rejected
func TestOC_MCH6_InvalidType(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	_, err := k.CreateMessageChannel(ctx, "malicious", payer1, "evil-channel", 50)
	if err == nil {
		t.Fatal("OC-MCH-6 FAIL: invalid channel type accepted")
	}

	_, err = k.CreateMessageChannel(ctx, "", payer1, "empty-type", 50)
	if err == nil {
		t.Fatal("OC-MCH-6 FAIL: empty channel type accepted")
	}
}

// Verify that the messaging keeper methods exist and are callable from tests.
// This serves as a compile-time check of the full interface.
func TestMessagingInterface_Compiles(t *testing.T) {
	k, _, _ := setupKeeper(t)
	_ = keeper.NewMsgServerImpl(k) // ensures keeper is compatible with msg server
}
