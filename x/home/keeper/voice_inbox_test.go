package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/home/types"
)

func TestSendAndGetInbox(t *testing.T) {
	k, ctx := setupKeeper(t)
	fromID := createTestHomeForMemory(t, k, ctx, owner1)
	toID := createTestHomeForMemory(t, k, ctx, owner2)

	msgID, err := k.SendMessage(ctx, fromID, toID, "Hello", "How are you?", 1000)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if msgID == "" {
		t.Fatal("expected non-empty messageID")
	}

	inbox := k.GetInbox(ctx, toID)
	if len(inbox) != 1 {
		t.Fatalf("expected 1 inbox message, got %d", len(inbox))
	}
	if inbox[0].Subject != "Hello" || inbox[0].Body != "How are you?" {
		t.Fatalf("unexpected message: %+v", inbox[0])
	}
	if inbox[0].FromHomeID != fromID || inbox[0].ToHomeID != toID {
		t.Fatalf("unexpected home IDs: from=%s to=%s", inbox[0].FromHomeID, inbox[0].ToHomeID)
	}
	if inbox[0].Read {
		t.Fatal("message should be unread initially")
	}
}

func TestReadMessage(t *testing.T) {
	k, ctx := setupKeeper(t)
	fromID := createTestHomeForMemory(t, k, ctx, owner1)
	toID := createTestHomeForMemory(t, k, ctx, owner2)

	msgID, _ := k.SendMessage(ctx, fromID, toID, "Subject", "Body", 1000)

	msg, found := k.ReadMessage(ctx, toID, msgID)
	if !found {
		t.Fatal("expected message to be found")
	}
	if !msg.Read {
		t.Fatal("message should be marked as read")
	}

	// Read again — should still be found and read.
	msg2, found := k.ReadMessage(ctx, toID, msgID)
	if !found || !msg2.Read {
		t.Fatal("expected message to remain found and read")
	}

	// Non-existent message.
	_, found = k.ReadMessage(ctx, toID, "nonexistent")
	if found {
		t.Fatal("expected non-existent message to not be found")
	}
}

func TestInboxExpiry(t *testing.T) {
	k, ctx := setupKeeper(t)
	fromID := createTestHomeForMemory(t, k, ctx, owner1)
	toID := createTestHomeForMemory(t, k, ctx, owner2)

	// Send message with TTL of 10 blocks at block 100.
	k.SendMessage(ctx, fromID, toID, "Expires soon", "body", 10)

	// At block 100 + 10 = 110, still visible.
	ctx110 := ctx.WithBlockHeight(110)
	inbox := k.GetInbox(ctx110, toID)
	if len(inbox) != 1 {
		t.Fatalf("expected message at block 110, got %d", len(inbox))
	}

	// Grab messageID while still visible.
	originalMsgID := inbox[0].MessageID

	// At block 111, expired.
	ctx111 := ctx.WithBlockHeight(111)
	inbox = k.GetInbox(ctx111, toID)
	if len(inbox) != 0 {
		t.Fatalf("expected expired message gone at block 111, got %d", len(inbox))
	}

	// ReadMessage should also return not-found for expired.
	_, found := k.ReadMessage(ctx111, toID, originalMsgID)
	if found {
		t.Fatal("expected expired message to not be readable")
	}
}

func TestPurgeExpired(t *testing.T) {
	k, ctx := setupKeeper(t)
	fromID := createTestHomeForMemory(t, k, ctx, owner1)
	toID := createTestHomeForMemory(t, k, ctx, owner2)

	// Send 3 messages: 2 short TTL, 1 long TTL.
	k.SendMessage(ctx.WithBlockHeight(100), fromID, toID, "Short1", "b", 5)
	k.SendMessage(ctx.WithBlockHeight(101), fromID, toID, "Short2", "b", 5)
	k.SendMessage(ctx.WithBlockHeight(100), fromID, toID, "Long", "b", 1000)

	// Purge at block 110 — Short1 (100+5=105 < 110) and Short2 (101+5=106 < 110) expired.
	ctx110 := ctx.WithBlockHeight(110)
	k.PurgeExpired(ctx110, toID)

	// Only the long-TTL message should remain.
	all := k.GetInbox(ctx110, toID)
	if len(all) != 1 {
		t.Fatalf("expected 1 message after purge, got %d", len(all))
	}
	if all[0].Subject != "Long" {
		t.Fatalf("expected Long message, got %q", all[0].Subject)
	}
}

func TestInboxFIFOEviction(t *testing.T) {
	k, ctx := setupKeeper(t)
	fromID := createTestHomeForMemory(t, k, ctx, owner1)
	toID := createTestHomeForMemory(t, k, ctx, owner2)

	// Fill inbox to capacity.
	for i := 0; i < types.MaxInboxMessages; i++ {
		bctx := ctx.WithBlockHeight(int64(i + 1))
		_, err := k.SendMessage(bctx, fromID, toID, "msg", "body", 10000)
		if err != nil {
			t.Fatalf("SendMessage %d failed: %v", i, err)
		}
	}

	inbox := k.GetInbox(ctx, toID)
	if len(inbox) != types.MaxInboxMessages {
		t.Fatalf("expected %d messages, got %d", types.MaxInboxMessages, len(inbox))
	}

	// Send one more — should evict oldest (block 1).
	bctx := ctx.WithBlockHeight(int64(types.MaxInboxMessages + 1))
	_, err := k.SendMessage(bctx, fromID, toID, "newest", "body", 10000)
	if err != nil {
		t.Fatalf("SendMessage overflow failed: %v", err)
	}

	inbox = k.GetInbox(bctx, toID)
	if len(inbox) != types.MaxInboxMessages {
		t.Fatalf("expected %d after eviction, got %d", types.MaxInboxMessages, len(inbox))
	}

	// Newest should be first.
	if inbox[0].Subject != "newest" {
		t.Fatalf("expected newest first, got %q", inbox[0].Subject)
	}
}

func TestSendMessageValidation(t *testing.T) {
	k, ctx := setupKeeper(t)
	fromID := createTestHomeForMemory(t, k, ctx, owner1)
	toID := createTestHomeForMemory(t, k, ctx, owner2)

	// Empty subject.
	_, err := k.SendMessage(ctx, fromID, toID, "", "body", 100)
	if err == nil {
		t.Fatal("expected error for empty subject")
	}

	// Zero TTL.
	_, err = k.SendMessage(ctx, fromID, toID, "subj", "body", 0)
	if err == nil {
		t.Fatal("expected error for zero TTL")
	}

	// Non-existent sender.
	_, err = k.SendMessage(ctx, "nonexistent", toID, "subj", "body", 100)
	if err == nil {
		t.Fatal("expected error for non-existent sender")
	}

	// Non-existent recipient.
	_, err = k.SendMessage(ctx, fromID, "nonexistent", "subj", "body", 100)
	if err == nil {
		t.Fatal("expected error for non-existent recipient")
	}
}
