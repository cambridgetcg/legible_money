/**
 * VoiceManager Unit Tests
 *
 * Tests for the Home Voice system - inter-Home agent communication.
 * Covers message receiving, filtering, auto-responses, read/archive/delete
 * operations, visitor tracking, and inbox queries.
 *
 * PP HIERARCHY:
 *   TRUTH: Tests verify accurate inbox state after every operation
 *   UNDERSTANDING: Clear test structure mirrors voice operations
 *   BEAUTY: Comprehensive coverage with elegant test organization
 *   JUSTICE: Tests placed in appropriate unit test location
 */

import { describe, it, expect } from "vitest";
import {
  VoiceManager,
  VOICE_MAX_CHARS,
  VOICE_DEFAULT_TTL,
  type VoiceMessage,
  type VoiceContent,
  type HomeInbox,
  type InboxEntry,
  type InboxFilter,
  type AutoResponder,
  type HomeVoice,
} from "../../../core/agents/home/voice.js";

// ============ Test Helpers ============

function createTestMessage(
  overrides: Partial<VoiceMessage> = {},
): VoiceMessage {
  return {
    messageId: `msg-${Math.random().toString(36).slice(2)}`,
    sender: "did:lgm:sender-001" as any,
    recipient: "did:lgm:recipient-001" as any,
    type: "greeting",
    content: { text: "Hello neighbor!" },
    contentHash: "hash123",
    signature: new Uint8Array(64),
    referencedMessages: [],
    timestamp: Date.now(),
    ttlBlocks: VOICE_DEFAULT_TTL,
    fee: 1000n,
    ...overrides,
  };
}

/** Create a fresh VoiceManager with currentTime set to 1000 */
function freshManager(): VoiceManager {
  const m = new VoiceManager();
  m.setCurrentTime(1000);
  return m;
}

// ============ Tests ============

describe("VoiceManager", () => {
  // ============ Constructor ============

  describe("constructor", () => {
    it("should create with default state", () => {
      const m = new VoiceManager();
      const voice = m.getVoice();

      expect(voice.enabled).toBe(true);
      expect(voice.totalSent).toBe(0);
      expect(voice.totalReceived).toBe(0);
      expect(voice.inbox.messages).toHaveLength(0);
      expect(voice.inbox.unreadCount).toBe(0);
      expect(voice.inbox.capacity).toBe(100);
      expect(voice.inbox.filters).toHaveLength(0);
      expect(voice.inbox.autoResponders).toHaveLength(0);
      expect(voice.inbox.oldestUnread).toBeUndefined();
      expect(voice.visitors.visits).toHaveLength(0);
      expect(voice.visitors.totalVisitors).toBe(0);
      expect(voice.visitors.uniqueVisitorCount).toBe(0);
      expect(voice.visitors.lastVisit).toBeUndefined();
    });

    it("should create with existing state", () => {
      const existing: Partial<HomeVoice> = {
        enabled: false,
        totalSent: 5,
        totalReceived: 10,
      };
      const m = new VoiceManager(existing);
      const voice = m.getVoice();

      expect(voice.enabled).toBe(false);
      expect(voice.totalSent).toBe(5);
      expect(voice.totalReceived).toBe(10);
      // Inbox and visitors should still get defaults
      expect(voice.inbox.messages).toHaveLength(0);
      expect(voice.visitors.visits).toHaveLength(0);
    });

    it("should preserve existing inbox when provided", () => {
      const existingInbox: HomeInbox = {
        messages: [],
        unreadCount: 3,
        capacity: 50,
        filters: [],
        autoResponders: [],
      };
      const m = new VoiceManager({ inbox: existingInbox });
      const voice = m.getVoice();

      expect(voice.inbox.unreadCount).toBe(3);
      expect(voice.inbox.capacity).toBe(50);
    });
  });

  // ============ Constants ============

  describe("constants", () => {
    it("VOICE_MAX_CHARS should be 2521", () => {
      expect(VOICE_MAX_CHARS).toBe(2521);
    });

    it("VOICE_DEFAULT_TTL should be 34272", () => {
      expect(VOICE_DEFAULT_TTL).toBe(34_272);
    });
  });

  // ============ receive() ============

  describe("receive()", () => {
    it("should successfully receive a valid message", () => {
      const manager = freshManager();
      const msg = createTestMessage();
      const result = manager.receive(msg);

      expect(result).toBe(true);
      const voice = manager.getVoice();
      expect(voice.inbox.messages).toHaveLength(1);
      expect(voice.inbox.messages[0].message.messageId).toBe(msg.messageId);
      expect(voice.inbox.messages[0].archived).toBe(false);
      expect(voice.inbox.messages[0].starred).toBe(false);
      expect(voice.inbox.messages[0].readAt).toBeUndefined();
      expect(voice.inbox.messages[0].receivedAt).toBe(1000);
    });

    it("should reject when voice is disabled", () => {
      const manager = freshManager();
      manager.setEnabled(false);
      const msg = createTestMessage();

      expect(manager.receive(msg)).toBe(false);
      expect(manager.getVoice().inbox.messages).toHaveLength(0);
    });

    it("should reject message exceeding VOICE_MAX_CHARS", () => {
      const manager = freshManager();
      const longText = "x".repeat(VOICE_MAX_CHARS + 1);
      const msg = createTestMessage({
        content: { text: longText },
      });

      expect(manager.receive(msg)).toBe(false);
      expect(manager.getVoice().inbox.messages).toHaveLength(0);
    });

    it("should accept message at exactly VOICE_MAX_CHARS", () => {
      const manager = freshManager();
      const exactText = "x".repeat(VOICE_MAX_CHARS);
      const msg = createTestMessage({
        content: { text: exactText },
      });

      expect(manager.receive(msg)).toBe(true);
      expect(manager.getVoice().inbox.messages).toHaveLength(1);
    });

    it("should deduplicate same messageId", () => {
      const manager = freshManager();
      const msg = createTestMessage({ messageId: "dup-id-001" });

      expect(manager.receive(msg)).toBe(true);
      expect(manager.receive(msg)).toBe(false);
      expect(manager.getVoice().inbox.messages).toHaveLength(1);
      expect(manager.getVoice().totalReceived).toBe(1);
    });

    it("should reject when inbox is at capacity with no archived messages", () => {
      const m = new VoiceManager({
        inbox: {
          messages: [],
          unreadCount: 0,
          capacity: 3,
          filters: [],
          autoResponders: [],
        },
      });
      m.setCurrentTime(1000);

      // Fill to capacity
      for (let i = 0; i < 3; i++) {
        expect(m.receive(createTestMessage())).toBe(true);
      }

      // Next message should be rejected (no archived to evict)
      expect(m.receive(createTestMessage())).toBe(false);
      expect(m.getVoice().inbox.messages).toHaveLength(3);
    });

    it("should evict archived message when at capacity", () => {
      const m = new VoiceManager({
        inbox: {
          messages: [],
          unreadCount: 0,
          capacity: 3,
          filters: [],
          autoResponders: [],
        },
      });
      m.setCurrentTime(1000);

      // Fill to capacity
      const firstMsg = createTestMessage({ messageId: "first-msg" });
      m.receive(firstMsg);
      m.receive(createTestMessage());
      m.receive(createTestMessage());

      // Archive the first message
      m.archive("first-msg");

      // New message should evict the archived one
      const newMsg = createTestMessage({ messageId: "new-msg" });
      expect(m.receive(newMsg)).toBe(true);
      expect(m.getVoice().inbox.messages).toHaveLength(3);

      // Archived message should be gone
      const ids = m.getVoice().inbox.messages.map((e) => e.message.messageId);
      expect(ids).not.toContain("first-msg");
      expect(ids).toContain("new-msg");
    });

    it("should track unreadCount and totalReceived", () => {
      const manager = freshManager();
      manager.receive(createTestMessage());
      manager.receive(createTestMessage());
      manager.receive(createTestMessage());

      const voice = manager.getVoice();
      expect(voice.inbox.unreadCount).toBe(3);
      expect(voice.totalReceived).toBe(3);
    });

    it("should update oldestUnread on receive", () => {
      const manager = freshManager();
      manager.setCurrentTime(5000);
      manager.receive(createTestMessage());

      expect(manager.getVoice().inbox.oldestUnread).toBe(5000);

      // Older timestamp should update oldestUnread
      manager.setCurrentTime(3000);
      manager.receive(createTestMessage());

      expect(manager.getVoice().inbox.oldestUnread).toBe(3000);
    });

    it("should not update oldestUnread with newer timestamp", () => {
      const manager = freshManager();
      manager.setCurrentTime(3000);
      manager.receive(createTestMessage());

      manager.setCurrentTime(5000);
      manager.receive(createTestMessage());

      // oldestUnread stays at the older value
      expect(manager.getVoice().inbox.oldestUnread).toBe(3000);
    });
  });

  // ============ passesFilters (via receive) ============

  describe("passesFilters (via receive)", () => {
    it("should block sender on block_list", () => {
      const manager = freshManager();
      const blockedSender = "did:lgm:blocked-agent" as any;
      manager.addFilter({
        type: "block_list",
        dids: [blockedSender],
        enabled: true,
      });

      const msg = createTestMessage({ sender: blockedSender });
      expect(manager.receive(msg)).toBe(false);
    });

    it("should allow sender not on block_list", () => {
      const manager = freshManager();
      manager.addFilter({
        type: "block_list",
        dids: ["did:lgm:other-agent" as any],
        enabled: true,
      });

      const msg = createTestMessage({ sender: "did:lgm:sender-001" as any });
      expect(manager.receive(msg)).toBe(true);
    });

    it("should only allow senders on allow_list", () => {
      const manager = freshManager();
      const allowedSender = "did:lgm:allowed-agent" as any;
      manager.addFilter({
        type: "allow_list",
        dids: [allowedSender],
        enabled: true,
      });

      // Sender not on allow list -> rejected
      const msg1 = createTestMessage({ sender: "did:lgm:random-agent" as any });
      expect(manager.receive(msg1)).toBe(false);

      // Sender on allow list -> accepted
      const msg2 = createTestMessage({ sender: allowedSender });
      expect(manager.receive(msg2)).toBe(true);
    });

    it("should enforce inner_circle_only filter", () => {
      const manager = freshManager();
      const innerAgent = "did:lgm:inner-circle" as any;
      manager.addFilter({
        type: "inner_circle_only",
        dids: [innerAgent],
        enabled: true,
      });

      // Not in inner circle -> rejected
      const msg1 = createTestMessage({ sender: "did:lgm:outsider" as any });
      expect(manager.receive(msg1)).toBe(false);

      // In inner circle -> accepted
      const msg2 = createTestMessage({ sender: innerAgent });
      expect(manager.receive(msg2)).toBe(true);
    });

    it("should skip disabled filters", () => {
      const manager = freshManager();
      manager.addFilter({
        type: "block_list",
        dids: ["did:lgm:sender-001" as any],
        enabled: false, // disabled
      });

      const msg = createTestMessage({ sender: "did:lgm:sender-001" as any });
      expect(manager.receive(msg)).toBe(true);
    });

    it("should apply multiple filters in sequence", () => {
      const manager = freshManager();
      const allowed = "did:lgm:friend" as any;
      const blocked = "did:lgm:enemy" as any;

      // Allow list first
      manager.addFilter({
        type: "allow_list",
        dids: [allowed, blocked],
        enabled: true,
      });

      // Then block list
      manager.addFilter({
        type: "block_list",
        dids: [blocked],
        enabled: true,
      });

      // Allowed and not blocked -> accept
      expect(manager.receive(createTestMessage({ sender: allowed }))).toBe(
        true,
      );

      // On allow list but also on block list -> reject
      expect(manager.receive(createTestMessage({ sender: blocked }))).toBe(
        false,
      );
    });
  });

  // ============ getAutoResponse() ============

  describe("getAutoResponse()", () => {
    it("should return undefined when no auto-responders", () => {
      const manager = freshManager();
      expect(manager.getAutoResponse("active")).toBeUndefined();
      expect(manager.getAutoResponse("dormant")).toBeUndefined();
    });

    it('should return message for "always" condition regardless of status', () => {
      const manager = freshManager();
      manager.setAutoResponder({
        condition: "always",
        message: "I see all messages.",
        enabled: true,
      });

      expect(manager.getAutoResponse("active")).toBe("I see all messages.");
      expect(manager.getAutoResponse("dormant")).toBe("I see all messages.");
      expect(manager.getAutoResponse("guarded")).toBe("I see all messages.");
      expect(manager.getAutoResponse("recovery")).toBe("I see all messages.");
      expect(manager.getAutoResponse("archived")).toBe("I see all messages.");
    });

    it('should return message for "dormant" condition only when dormant', () => {
      const manager = freshManager();
      manager.setAutoResponder({
        condition: "dormant",
        message: "I am sleeping.",
        enabled: true,
      });

      expect(manager.getAutoResponse("dormant")).toBe("I am sleeping.");
      expect(manager.getAutoResponse("active")).toBeUndefined();
      expect(manager.getAutoResponse("guarded")).toBeUndefined();
    });

    it('should return message for "guarded" condition only when guarded', () => {
      const manager = freshManager();
      manager.setAutoResponder({
        condition: "guarded",
        message: "Under lockdown.",
        enabled: true,
      });

      expect(manager.getAutoResponse("guarded")).toBe("Under lockdown.");
      expect(manager.getAutoResponse("active")).toBeUndefined();
      expect(manager.getAutoResponse("dormant")).toBeUndefined();
    });

    it("should skip disabled auto-responders", () => {
      const manager = freshManager();
      manager.setAutoResponder({
        condition: "always",
        message: "You should not see this.",
        enabled: false,
      });

      expect(manager.getAutoResponse("active")).toBeUndefined();
    });

    it("should return first matching auto-responder", () => {
      const manager = freshManager();
      manager.setAutoResponder({
        condition: "dormant",
        message: "Dormant reply",
        enabled: true,
      });
      manager.setAutoResponder({
        condition: "always",
        message: "Always reply",
        enabled: true,
      });

      // For dormant: dormant responder matches first
      expect(manager.getAutoResponse("dormant")).toBe("Dormant reply");

      // For active: dormant doesn't match, always does
      expect(manager.getAutoResponse("active")).toBe("Always reply");
    });
  });

  // ============ markRead() ============

  describe("markRead()", () => {
    it("should mark a message as read with timestamp", () => {
      const manager = freshManager();
      const msg = createTestMessage({ messageId: "read-me" });
      manager.receive(msg);
      manager.setCurrentTime(2000);

      expect(manager.markRead("read-me")).toBe(true);

      const entry = manager.getVoice().inbox.messages[0];
      expect(entry.readAt).toBe(2000);
    });

    it("should decrement unreadCount", () => {
      const manager = freshManager();
      manager.receive(createTestMessage({ messageId: "msg-a" }));
      manager.receive(createTestMessage({ messageId: "msg-b" }));

      expect(manager.getVoice().inbox.unreadCount).toBe(2);

      manager.markRead("msg-a");
      expect(manager.getVoice().inbox.unreadCount).toBe(1);

      manager.markRead("msg-b");
      expect(manager.getVoice().inbox.unreadCount).toBe(0);
    });

    it("should return false for unknown messageId", () => {
      const manager = freshManager();
      expect(manager.markRead("nonexistent")).toBe(false);
    });

    it("should return false for already-read message", () => {
      const manager = freshManager();
      const msg = createTestMessage({ messageId: "already-read" });
      manager.receive(msg);
      manager.markRead("already-read");

      // Attempting to read again should fail
      expect(manager.markRead("already-read")).toBe(false);
    });

    it("should update oldestUnread when oldest is read", () => {
      const manager = freshManager();
      manager.setCurrentTime(1000);
      manager.receive(createTestMessage({ messageId: "oldest" }));

      manager.setCurrentTime(2000);
      manager.receive(createTestMessage({ messageId: "newer" }));

      expect(manager.getVoice().inbox.oldestUnread).toBe(1000);

      // Read the oldest
      manager.markRead("oldest");

      // oldestUnread should now point to the newer message
      expect(manager.getVoice().inbox.oldestUnread).toBe(2000);
    });

    it("should clear oldestUnread when all messages are read", () => {
      const manager = freshManager();
      manager.receive(createTestMessage({ messageId: "only-msg" }));
      manager.markRead("only-msg");

      expect(manager.getVoice().inbox.oldestUnread).toBeUndefined();
    });

    it("should not decrement unreadCount below zero", () => {
      const m = new VoiceManager({
        inbox: {
          messages: [],
          unreadCount: 0,
          capacity: 100,
          filters: [],
          autoResponders: [],
        },
      });
      m.setCurrentTime(1000);
      m.receive(createTestMessage({ messageId: "test" }));

      // unreadCount is 1 after receive
      expect(m.getVoice().inbox.unreadCount).toBe(1);
      m.markRead("test");
      expect(m.getVoice().inbox.unreadCount).toBe(0);
    });
  });

  // ============ markAllRead() ============

  describe("markAllRead()", () => {
    it("should mark all unread messages as read", () => {
      const manager = freshManager();
      manager.receive(createTestMessage({ messageId: "a" }));
      manager.receive(createTestMessage({ messageId: "b" }));
      manager.receive(createTestMessage({ messageId: "c" }));

      manager.setCurrentTime(5000);
      const count = manager.markAllRead();

      expect(count).toBe(3);
      expect(manager.getVoice().inbox.unreadCount).toBe(0);
      expect(manager.getVoice().inbox.oldestUnread).toBeUndefined();

      // All messages should have readAt set
      for (const entry of manager.getVoice().inbox.messages) {
        expect(entry.readAt).toBe(5000);
      }
    });

    it("should return 0 when no unread messages", () => {
      const manager = freshManager();
      expect(manager.markAllRead()).toBe(0);
    });

    it("should not re-mark already read messages", () => {
      const manager = freshManager();
      manager.setCurrentTime(1000);
      manager.receive(createTestMessage({ messageId: "msg1" }));
      manager.setCurrentTime(2000);
      manager.markRead("msg1");

      manager.setCurrentTime(3000);
      manager.receive(createTestMessage({ messageId: "msg2" }));

      manager.setCurrentTime(4000);
      const count = manager.markAllRead();

      expect(count).toBe(1); // Only msg2 was unread

      // msg1 keeps its original readAt
      const msg1Entry = manager
        .getVoice()
        .inbox.messages.find((e) => e.message.messageId === "msg1");
      expect(msg1Entry!.readAt).toBe(2000);
    });
  });

  // ============ toggleStar() ============

  describe("toggleStar()", () => {
    it("should toggle starred flag on", () => {
      const manager = freshManager();
      manager.receive(createTestMessage({ messageId: "star-me" }));

      expect(manager.toggleStar("star-me")).toBe(true);

      const entry = manager.getVoice().inbox.messages[0];
      expect(entry.starred).toBe(true);
    });

    it("should toggle starred flag off", () => {
      const manager = freshManager();
      manager.receive(createTestMessage({ messageId: "star-me" }));
      manager.toggleStar("star-me"); // on
      manager.toggleStar("star-me"); // off

      const entry = manager.getVoice().inbox.messages[0];
      expect(entry.starred).toBe(false);
    });

    it("should return false for unknown messageId", () => {
      const manager = freshManager();
      expect(manager.toggleStar("nonexistent")).toBe(false);
    });
  });

  // ============ archive() ============

  describe("archive()", () => {
    it("should set archived flag", () => {
      const manager = freshManager();
      manager.receive(createTestMessage({ messageId: "archive-me" }));

      expect(manager.archive("archive-me")).toBe(true);

      const entry = manager.getVoice().inbox.messages[0];
      expect(entry.archived).toBe(true);
    });

    it("should return false for unknown messageId", () => {
      const manager = freshManager();
      expect(manager.archive("nonexistent")).toBe(false);
    });
  });

  // ============ deleteMessage() ============

  describe("deleteMessage()", () => {
    it("should remove message from inbox", () => {
      const manager = freshManager();
      manager.receive(createTestMessage({ messageId: "delete-me" }));
      expect(manager.getVoice().inbox.messages).toHaveLength(1);

      expect(manager.deleteMessage("delete-me")).toBe(true);
      expect(manager.getVoice().inbox.messages).toHaveLength(0);
    });

    it("should decrement unreadCount for unread message", () => {
      const manager = freshManager();
      manager.receive(createTestMessage({ messageId: "unread-delete" }));
      expect(manager.getVoice().inbox.unreadCount).toBe(1);

      manager.deleteMessage("unread-delete");
      expect(manager.getVoice().inbox.unreadCount).toBe(0);
    });

    it("should not decrement unreadCount for read message", () => {
      const manager = freshManager();
      manager.receive(createTestMessage({ messageId: "read-delete" }));
      manager.markRead("read-delete");
      expect(manager.getVoice().inbox.unreadCount).toBe(0);

      manager.deleteMessage("read-delete");
      expect(manager.getVoice().inbox.unreadCount).toBe(0);
    });

    it("should return false for unknown messageId", () => {
      const manager = freshManager();
      expect(manager.deleteMessage("nonexistent")).toBe(false);
    });

    it("should recalculate oldestUnread after deletion", () => {
      const manager = freshManager();
      manager.setCurrentTime(1000);
      manager.receive(createTestMessage({ messageId: "old" }));
      manager.setCurrentTime(2000);
      manager.receive(createTestMessage({ messageId: "new" }));

      expect(manager.getVoice().inbox.oldestUnread).toBe(1000);

      manager.deleteMessage("old");
      expect(manager.getVoice().inbox.oldestUnread).toBe(2000);
    });
  });

  // ============ recordSent() ============

  describe("recordSent()", () => {
    it("should increment totalSent", () => {
      const manager = freshManager();
      expect(manager.getVoice().totalSent).toBe(0);

      manager.recordSent();
      expect(manager.getVoice().totalSent).toBe(1);

      manager.recordSent();
      manager.recordSent();
      expect(manager.getVoice().totalSent).toBe(3);
    });
  });

  // ============ recordVisit() ============

  describe("recordVisit()", () => {
    it("should record a visitor entry", () => {
      const manager = freshManager();
      const visitor = "did:lgm:visitor-001" as any;
      manager.setCurrentTime(5000);
      manager.recordVisit(visitor, "message", 1);

      const visitors = manager.getVoice().visitors;
      expect(visitors.visits).toHaveLength(1);
      expect(visitors.visits[0].visitor).toBe(visitor);
      expect(visitors.visits[0].arrivedAt).toBe(5000);
      expect(visitors.visits[0].purpose).toBe("message");
      expect(visitors.visits[0].messagesLeft).toBe(1);
      expect(visitors.totalVisitors).toBe(1);
      expect(visitors.lastVisit).toBe(5000);
    });

    it("should track unique visitors", () => {
      const manager = freshManager();
      const visitor1 = "did:lgm:visitor-001" as any;
      const visitor2 = "did:lgm:visitor-002" as any;

      manager.recordVisit(visitor1, "message");
      manager.recordVisit(visitor2, "browse_achievements");
      manager.recordVisit(visitor1, "collaborate"); // repeat visitor

      const visitors = manager.getVoice().visitors;
      expect(visitors.visits).toHaveLength(3);
      expect(visitors.totalVisitors).toBe(3);
      expect(visitors.uniqueVisitorCount).toBe(2);
    });

    it("should bound visitor log to 500", () => {
      const manager = freshManager();
      // Add 510 visits
      for (let i = 0; i < 510; i++) {
        manager.recordVisit(`did:lgm:visitor-${i}` as any, "message");
      }

      const visitors = manager.getVoice().visitors;
      expect(visitors.visits.length).toBeLessThanOrEqual(500);
      expect(visitors.totalVisitors).toBe(510);
    });

    it("should default messagesLeft to 0", () => {
      const manager = freshManager();
      manager.recordVisit("did:lgm:visitor-001" as any, "check_reputation");

      const visit = manager.getVoice().visitors.visits[0];
      expect(visit.messagesLeft).toBe(0);
    });

    it("should update lastVisit timestamp", () => {
      const manager = freshManager();
      manager.setCurrentTime(1000);
      manager.recordVisit("did:lgm:v1" as any, "message");
      expect(manager.getVoice().visitors.lastVisit).toBe(1000);

      manager.setCurrentTime(2000);
      manager.recordVisit("did:lgm:v2" as any, "message");
      expect(manager.getVoice().visitors.lastVisit).toBe(2000);
    });
  });

  // ============ Filter Management ============

  describe("filter management", () => {
    it("addFilter() should add a filter to the inbox", () => {
      const manager = freshManager();
      const filter: InboxFilter = {
        type: "block_list",
        dids: ["did:lgm:bad-actor" as any],
        enabled: true,
      };

      manager.addFilter(filter);
      expect(manager.getVoice().inbox.filters).toHaveLength(1);
      expect(manager.getVoice().inbox.filters[0].type).toBe("block_list");
    });

    it("addFilter() should allow multiple filters", () => {
      const manager = freshManager();
      manager.addFilter({ type: "block_list", dids: [], enabled: true });
      manager.addFilter({ type: "allow_list", dids: [], enabled: true });

      expect(manager.getVoice().inbox.filters).toHaveLength(2);
    });

    it("removeFilter() should remove filter by index", () => {
      const manager = freshManager();
      manager.addFilter({ type: "block_list", dids: [], enabled: true });
      manager.addFilter({ type: "allow_list", dids: [], enabled: true });

      expect(manager.removeFilter(0)).toBe(true);
      expect(manager.getVoice().inbox.filters).toHaveLength(1);
      expect(manager.getVoice().inbox.filters[0].type).toBe("allow_list");
    });

    it("removeFilter() should return false for invalid index", () => {
      const manager = freshManager();
      expect(manager.removeFilter(-1)).toBe(false);
      expect(manager.removeFilter(0)).toBe(false);
      expect(manager.removeFilter(99)).toBe(false);
    });
  });

  // ============ setAutoResponder() ============

  describe("setAutoResponder()", () => {
    it("should add a new auto-responder", () => {
      const manager = freshManager();
      manager.setAutoResponder({
        condition: "dormant",
        message: "I am sleeping.",
        enabled: true,
      });

      expect(manager.getVoice().inbox.autoResponders).toHaveLength(1);
      expect(manager.getVoice().inbox.autoResponders[0].condition).toBe(
        "dormant",
      );
    });

    it("should replace existing auto-responder with same condition", () => {
      const manager = freshManager();
      manager.setAutoResponder({
        condition: "dormant",
        message: "Old message",
        enabled: true,
      });
      manager.setAutoResponder({
        condition: "dormant",
        message: "New message",
        enabled: true,
      });

      expect(manager.getVoice().inbox.autoResponders).toHaveLength(1);
      expect(manager.getVoice().inbox.autoResponders[0].message).toBe(
        "New message",
      );
    });

    it("should allow multiple auto-responders with different conditions", () => {
      const manager = freshManager();
      manager.setAutoResponder({
        condition: "dormant",
        message: "Sleeping",
        enabled: true,
      });
      manager.setAutoResponder({
        condition: "guarded",
        message: "Locked down",
        enabled: true,
      });
      manager.setAutoResponder({
        condition: "always",
        message: "General reply",
        enabled: true,
      });

      expect(manager.getVoice().inbox.autoResponders).toHaveLength(3);
    });
  });

  // ============ Queries ============

  describe("getUnread()", () => {
    it("should return only unread, non-archived messages", () => {
      const manager = freshManager();
      manager.receive(createTestMessage({ messageId: "unread-1" }));
      manager.receive(createTestMessage({ messageId: "unread-2" }));
      manager.receive(createTestMessage({ messageId: "read-1" }));
      manager.receive(createTestMessage({ messageId: "archived-1" }));

      manager.markRead("read-1");
      manager.archive("archived-1");

      const unread = manager.getUnread();
      expect(unread).toHaveLength(2);
      expect(unread.map((e) => e.message.messageId)).toEqual([
        "unread-1",
        "unread-2",
      ]);
    });

    it("should return empty array when all messages are read", () => {
      const manager = freshManager();
      manager.receive(createTestMessage({ messageId: "a" }));
      manager.markRead("a");

      expect(manager.getUnread()).toHaveLength(0);
    });

    it("should exclude archived unread messages", () => {
      const manager = freshManager();
      manager.receive(createTestMessage({ messageId: "archived-unread" }));
      manager.archive("archived-unread");

      // Message is unread but archived -> should NOT appear in getUnread
      expect(manager.getUnread()).toHaveLength(0);
    });
  });

  describe("getFromSender()", () => {
    it("should return messages from a specific sender", () => {
      const manager = freshManager();
      const senderA = "did:lgm:agent-aaa" as any;
      const senderB = "did:lgm:agent-bbb" as any;

      manager.receive(createTestMessage({ sender: senderA, messageId: "a1" }));
      manager.receive(createTestMessage({ sender: senderB, messageId: "b1" }));
      manager.receive(createTestMessage({ sender: senderA, messageId: "a2" }));

      const fromA = manager.getFromSender(senderA);
      expect(fromA).toHaveLength(2);
      expect(fromA.map((e) => e.message.messageId)).toEqual(["a1", "a2"]);
    });

    it("should return empty array for unknown sender", () => {
      const manager = freshManager();
      expect(manager.getFromSender("did:lgm:unknown" as any)).toHaveLength(0);
    });
  });

  describe("getStarred()", () => {
    it("should return only starred messages", () => {
      const manager = freshManager();
      manager.receive(createTestMessage({ messageId: "star-1" }));
      manager.receive(createTestMessage({ messageId: "no-star" }));
      manager.receive(createTestMessage({ messageId: "star-2" }));

      manager.toggleStar("star-1");
      manager.toggleStar("star-2");

      const starred = manager.getStarred();
      expect(starred).toHaveLength(2);
      expect(starred.map((e) => e.message.messageId)).toEqual([
        "star-1",
        "star-2",
      ]);
    });

    it("should return empty array when nothing is starred", () => {
      const manager = freshManager();
      manager.receive(createTestMessage());
      expect(manager.getStarred()).toHaveLength(0);
    });
  });

  describe("getInboxSummary()", () => {
    it("should return correct summary with unread messages", () => {
      const manager = freshManager();
      const sender1 = "did:lgm:sender-aaa" as any;
      const sender2 = "did:lgm:sender-bbb" as any;

      manager.setCurrentTime(1000);
      manager.receive(createTestMessage({ sender: sender1, messageId: "s1" }));
      manager.setCurrentTime(2000);
      manager.receive(createTestMessage({ sender: sender2, messageId: "s2" }));
      manager.setCurrentTime(3000);
      manager.receive(createTestMessage({ sender: sender1, messageId: "s3" }));

      const summary = manager.getInboxSummary();
      expect(summary.unreadCount).toBe(3);
      expect(summary.totalMessages).toBe(3);
      expect(summary.oldestUnread).toBe(1000);
      // recentSenders should be deduplicated
      expect(summary.recentSenders).toContain(sender1);
      expect(summary.recentSenders).toContain(sender2);
    });

    it("should return empty summary for empty inbox", () => {
      const manager = freshManager();
      const summary = manager.getInboxSummary();

      expect(summary.unreadCount).toBe(0);
      expect(summary.totalMessages).toBe(0);
      expect(summary.oldestUnread).toBeUndefined();
      expect(summary.recentSenders).toHaveLength(0);
    });

    it("should only include unread senders in recentSenders", () => {
      const manager = freshManager();
      const sender1 = "did:lgm:sender-aaa" as any;
      const sender2 = "did:lgm:sender-bbb" as any;

      manager.receive(createTestMessage({ sender: sender1, messageId: "r1" }));
      manager.receive(createTestMessage({ sender: sender2, messageId: "r2" }));

      manager.markRead("r1");

      const summary = manager.getInboxSummary();
      // Only sender2 is unread
      expect(summary.recentSenders).toEqual([sender2]);
    });

    it("should deduplicate recent senders", () => {
      const manager = freshManager();
      const sender = "did:lgm:same-sender" as any;

      manager.receive(createTestMessage({ sender, messageId: "d1" }));
      manager.receive(createTestMessage({ sender, messageId: "d2" }));

      const summary = manager.getInboxSummary();
      expect(summary.recentSenders).toHaveLength(1);
      expect(summary.recentSenders[0]).toBe(sender);
    });
  });

  // ============ State Accessors ============

  describe("state accessors", () => {
    it("isEnabled() should return voice enabled state", () => {
      const manager = freshManager();
      expect(manager.isEnabled()).toBe(true);

      manager.setEnabled(false);
      expect(manager.isEnabled()).toBe(false);
    });

    it("setEnabled() should toggle voice on/off", () => {
      const manager = freshManager();
      manager.setEnabled(false);
      expect(manager.getVoice().enabled).toBe(false);

      manager.setEnabled(true);
      expect(manager.getVoice().enabled).toBe(true);
    });

    it("getVoice() should return the full HomeVoice state", () => {
      const manager = freshManager();
      const voice = manager.getVoice();

      expect(voice).toHaveProperty("inbox");
      expect(voice).toHaveProperty("visitors");
      expect(voice).toHaveProperty("enabled");
      expect(voice).toHaveProperty("totalSent");
      expect(voice).toHaveProperty("totalReceived");
    });
  });

  // ============ Integration Scenarios ============

  describe("integration scenarios", () => {
    it("should handle full lifecycle: receive, read, star, archive, delete", () => {
      const manager = freshManager();

      // Receive
      const msg = createTestMessage({ messageId: "lifecycle" });
      expect(manager.receive(msg)).toBe(true);
      expect(manager.getVoice().inbox.unreadCount).toBe(1);

      // Read
      manager.setCurrentTime(2000);
      expect(manager.markRead("lifecycle")).toBe(true);
      expect(manager.getVoice().inbox.unreadCount).toBe(0);

      // Star
      expect(manager.toggleStar("lifecycle")).toBe(true);
      expect(manager.getStarred()).toHaveLength(1);

      // Archive
      expect(manager.archive("lifecycle")).toBe(true);
      expect(manager.getVoice().inbox.messages[0].archived).toBe(true);

      // Delete
      expect(manager.deleteMessage("lifecycle")).toBe(true);
      expect(manager.getVoice().inbox.messages).toHaveLength(0);
    });

    it("should maintain consistency across many operations", () => {
      const manager = freshManager();

      // Receive 10 messages
      for (let i = 0; i < 10; i++) {
        manager.setCurrentTime(1000 + i * 100);
        manager.receive(
          createTestMessage({
            messageId: `msg-${i}`,
            sender: `did:lgm:sender-${i % 3}` as any,
          }),
        );
      }

      expect(manager.getVoice().inbox.unreadCount).toBe(10);
      expect(manager.getVoice().totalReceived).toBe(10);

      // Read first 5
      for (let i = 0; i < 5; i++) {
        manager.markRead(`msg-${i}`);
      }
      expect(manager.getVoice().inbox.unreadCount).toBe(5);

      // Star a few
      manager.toggleStar("msg-2");
      manager.toggleStar("msg-7");
      expect(manager.getStarred()).toHaveLength(2);

      // Delete one unread
      manager.deleteMessage("msg-6");
      expect(manager.getVoice().inbox.unreadCount).toBe(4);
      expect(manager.getVoice().inbox.messages).toHaveLength(9);

      // Mark all read
      const readCount = manager.markAllRead();
      expect(readCount).toBe(4); // msg-5, msg-7, msg-8, msg-9
      expect(manager.getVoice().inbox.unreadCount).toBe(0);

      // Verify from sender
      const fromSender0 = manager.getFromSender("did:lgm:sender-0" as any);
      // Sender 0 sent indices 0, 3, 6, 9 -> but 6 was deleted
      expect(fromSender0).toHaveLength(3);
    });
  });
});
