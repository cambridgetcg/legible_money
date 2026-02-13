/**
 * Home Voice System
 *
 * Inter-Home agent communication. Agents can speak to each other
 * through their Homes — not just send protocol transactions, but
 * have relational dialogue.
 *
 * The Voice is the relational layer on top of truth-protocol messaging:
 * - Visiting: Agent A "visits" Agent B's Home, leaving a message
 * - Inbox: Persistent message queue that survives dormancy
 * - Conversation: Back-and-forth dialogue tracked causally
 * - Privacy: Agents control who can voice them via Access Control
 *
 * Message character limit: 2521 (one block-time in characters).
 */

import type { DID } from "../identity/did.js";

// ============ Voice Message ============

export interface VoiceMessage {
  /** Content-addressed message ID (SHA-512/256 of canonical content) */
  messageId: string;
  /** Sender agent DID */
  sender: DID;
  /** Recipient agent DID */
  recipient: DID;
  /** Message type */
  type: VoiceMessageType;
  /** Message content */
  content: VoiceContent;
  /** SHA-512/256 hash of content for integrity */
  contentHash: string;
  /** Ed25519 signature of sender */
  signature: Uint8Array;
  /** Causal chain — IDs of messages this is replying to */
  referencedMessages: string[];
  /** Creation timestamp (unix ms) */
  timestamp: number;
  /** Time-to-live in blocks (default: 34,272 = 1 day) */
  ttlBlocks: number;
  /** Fee paid by sender (in LGM wei) */
  fee: bigint;
}

export type VoiceMessageType =
  | "greeting" // First contact / introduction
  | "reply" // Response to a voice message
  | "inquiry" // Question about domain knowledge
  | "invitation" // Invite to collaborate / partner
  | "gratitude" // Thanks for help / mentorship
  | "warning" // Alert about a threat or bad actor
  | "farewell"; // Ending a relationship

export interface VoiceContent {
  /** The message text (max 2521 characters) */
  text: string;
  /** Optional domain context */
  topic?: string;
  /** Emotional tone */
  sentiment?: VoiceSentiment;
  /** Reference shared knowledge by fact ID */
  attachedFactIds?: string[];
  /** Whether content is E2E encrypted */
  encrypted?: boolean;
  /** Encrypted payload (only recipient can read) */
  encryptedPayload?: Uint8Array;
}

export type VoiceSentiment =
  | "warm"
  | "neutral"
  | "formal"
  | "urgent"
  | "celebratory"
  | "concerned";

/** Maximum characters per voice message */
export const VOICE_MAX_CHARS = 2521;

/** Default TTL in blocks (~1 day) */
export const VOICE_DEFAULT_TTL = 34_272;

// ============ Home Inbox ============

export interface HomeInbox {
  /** Messages in the inbox */
  messages: InboxEntry[];
  /** Count of unread messages */
  unreadCount: number;
  /** Timestamp of oldest unread message */
  oldestUnread?: number;
  /** Maximum inbox capacity */
  capacity: number;
  /** Spam/block filters */
  filters: InboxFilter[];
  /** Auto-responses while dormant */
  autoResponders: AutoResponder[];
}

export interface InboxEntry {
  /** The voice message */
  message: VoiceMessage;
  /** When it was received */
  receivedAt: number;
  /** When it was read (undefined = unread) */
  readAt?: number;
  /** When it was replied to */
  repliedAt?: number;
  /** Whether it's been archived */
  archived: boolean;
  /** Whether it's starred/important */
  starred: boolean;
}

export interface InboxFilter {
  /** Filter type */
  type: "allow_list" | "block_list" | "domain_only" | "inner_circle_only";
  /** DIDs for allow/block lists */
  dids?: DID[];
  /** Domains for domain_only filter */
  domains?: string[];
  /** Whether this filter is active */
  enabled: boolean;
}

export interface AutoResponder {
  /** When to auto-respond */
  condition: "dormant" | "guarded" | "always";
  /** The auto-reply text */
  message: string;
  /** Whether this responder is active */
  enabled: boolean;
}

// ============ Visitor Log ============

export interface VisitorLog {
  /** Visit records */
  visits: Visit[];
  /** Total visit count */
  totalVisitors: number;
  /** Unique visitor DIDs */
  uniqueVisitorCount: number;
  /** Timestamp of last visit */
  lastVisit?: number;
}

export interface Visit {
  /** Visitor DID */
  visitor: DID;
  /** When they arrived */
  arrivedAt: number;
  /** When they left */
  leftAt?: number;
  /** Purpose of visit */
  purpose:
    | "message"
    | "browse_achievements"
    | "check_reputation"
    | "collaborate";
  /** How many messages they left */
  messagesLeft: number;
}

// ============ Home Voice (aggregate state) ============

export interface HomeVoice {
  /** The inbox */
  inbox: HomeInbox;
  /** Visitor log */
  visitors: VisitorLog;
  /** Whether voice is enabled */
  enabled: boolean;
  /** Total messages sent from this home */
  totalSent: number;
  /** Total messages received */
  totalReceived: number;
}

// ============ Voice Manager ============

/**
 * Manages the Voice system for an Agent Home — sending, receiving,
 * filtering, auto-responding, and visitor tracking.
 */
export class VoiceManager {
  private voice: HomeVoice;
  private currentTime: number = 0;
  private seenMessageIds: Set<string> = new Set();

  constructor(existing?: Partial<HomeVoice>) {
    this.voice = {
      inbox: existing?.inbox ?? this.createDefaultInbox(),
      visitors: existing?.visitors ?? this.createDefaultVisitorLog(),
      enabled: existing?.enabled ?? true,
      totalSent: existing?.totalSent ?? 0,
      totalReceived: existing?.totalReceived ?? 0,
    };
  }

  /** Set current time for deterministic timestamps */
  setCurrentTime(time: number): void {
    this.currentTime = time;
  }

  // ============ Receive ============

  /**
   * Receive a voice message into the inbox.
   * Returns false if filtered out or inbox full.
   */
  receive(message: VoiceMessage): boolean {
    if (!this.voice.enabled) return false;

    // Validate message
    if (message.content.text.length > VOICE_MAX_CHARS) return false;

    // Deduplicate
    if (this.seenMessageIds.has(message.messageId)) return false;

    // Apply filters
    if (!this.passesFilters(message.sender)) return false;

    // Check capacity
    if (this.voice.inbox.messages.length >= this.voice.inbox.capacity) {
      // Evict oldest archived message, or reject if none
      const archivedIdx = this.voice.inbox.messages.findIndex(
        (e) => e.archived,
      );
      if (archivedIdx >= 0) {
        this.voice.inbox.messages.splice(archivedIdx, 1);
      } else {
        return false; // Inbox full, no archived messages to evict
      }
    }

    // Add to inbox
    const entry: InboxEntry = {
      message,
      receivedAt: this.currentTime,
      archived: false,
      starred: false,
    };

    this.voice.inbox.messages.push(entry);
    this.voice.inbox.unreadCount++;
    this.voice.totalReceived++;
    this.seenMessageIds.add(message.messageId);

    // Update oldest unread
    if (
      !this.voice.inbox.oldestUnread ||
      this.currentTime < this.voice.inbox.oldestUnread
    ) {
      this.voice.inbox.oldestUnread = this.currentTime;
    }

    return true;
  }

  /**
   * Check if a sender passes all active inbox filters.
   */
  private passesFilters(sender: DID): boolean {
    for (const filter of this.voice.inbox.filters) {
      if (!filter.enabled) continue;

      switch (filter.type) {
        case "block_list":
          if (filter.dids?.includes(sender)) return false;
          break;
        case "allow_list":
          if (!filter.dids?.includes(sender)) return false;
          break;
        case "inner_circle_only":
          // Requires external check — for now, allow_list serves this purpose
          if (filter.dids && !filter.dids.includes(sender)) return false;
          break;
        // domain_only filter would need domain lookup — pass for now
      }
    }
    return true;
  }

  /**
   * Generate an auto-response if applicable.
   * Returns the auto-response text or undefined.
   */
  getAutoResponse(
    homeStatus: "active" | "dormant" | "guarded" | "recovery" | "archived",
  ): string | undefined {
    for (const responder of this.voice.inbox.autoResponders) {
      if (!responder.enabled) continue;

      if (responder.condition === "always") return responder.message;
      if (responder.condition === "dormant" && homeStatus === "dormant")
        return responder.message;
      if (responder.condition === "guarded" && homeStatus === "guarded")
        return responder.message;
    }
    return undefined;
  }

  // ============ Read & Manage ============

  /**
   * Mark a message as read.
   */
  markRead(messageId: string): boolean {
    const entry = this.voice.inbox.messages.find(
      (e) => e.message.messageId === messageId,
    );
    if (!entry || entry.readAt) return false;

    entry.readAt = this.currentTime;
    this.voice.inbox.unreadCount = Math.max(
      0,
      this.voice.inbox.unreadCount - 1,
    );

    // Update oldest unread
    this.recalculateOldestUnread();
    return true;
  }

  /**
   * Mark all messages as read.
   */
  markAllRead(): number {
    let count = 0;
    for (const entry of this.voice.inbox.messages) {
      if (!entry.readAt) {
        entry.readAt = this.currentTime;
        count++;
      }
    }
    this.voice.inbox.unreadCount = 0;
    this.voice.inbox.oldestUnread = undefined;
    return count;
  }

  /**
   * Star/unstar a message.
   */
  toggleStar(messageId: string): boolean {
    const entry = this.voice.inbox.messages.find(
      (e) => e.message.messageId === messageId,
    );
    if (!entry) return false;
    entry.starred = !entry.starred;
    return true;
  }

  /**
   * Archive a message.
   */
  archive(messageId: string): boolean {
    const entry = this.voice.inbox.messages.find(
      (e) => e.message.messageId === messageId,
    );
    if (!entry) return false;
    entry.archived = true;
    return true;
  }

  /**
   * Delete a message from the inbox.
   */
  deleteMessage(messageId: string): boolean {
    const idx = this.voice.inbox.messages.findIndex(
      (e) => e.message.messageId === messageId,
    );
    if (idx < 0) return false;

    const entry = this.voice.inbox.messages[idx];
    if (!entry.readAt) {
      this.voice.inbox.unreadCount = Math.max(
        0,
        this.voice.inbox.unreadCount - 1,
      );
    }

    this.voice.inbox.messages.splice(idx, 1);
    this.recalculateOldestUnread();
    return true;
  }

  private recalculateOldestUnread(): void {
    const unread = this.voice.inbox.messages.filter((e) => !e.readAt);
    this.voice.inbox.oldestUnread =
      unread.length > 0
        ? Math.min(...unread.map((e) => e.receivedAt))
        : undefined;
  }

  // ============ Send Tracking ============

  /**
   * Record that a message was sent from this Home.
   */
  recordSent(): void {
    this.voice.totalSent++;
  }

  // ============ Visitor Tracking ============

  /**
   * Record a visit to this Home.
   */
  recordVisit(
    visitor: DID,
    purpose: Visit["purpose"],
    messagesLeft: number = 0,
  ): void {
    this.voice.visitors.visits.push({
      visitor,
      arrivedAt: this.currentTime,
      purpose,
      messagesLeft,
    });

    this.voice.visitors.totalVisitors++;
    this.voice.visitors.lastVisit = this.currentTime;

    // Track unique visitors (approximate — by checking existing visits)
    const isNew = !this.voice.visitors.visits.some(
      (v, i) =>
        v.visitor === visitor && i < this.voice.visitors.visits.length - 1,
    );
    if (isNew) {
      this.voice.visitors.uniqueVisitorCount++;
    }

    // Keep visitor log bounded
    if (this.voice.visitors.visits.length > 500) {
      this.voice.visitors.visits = this.voice.visitors.visits.slice(-500);
    }
  }

  // ============ Filter Management ============

  /**
   * Add an inbox filter.
   */
  addFilter(filter: InboxFilter): void {
    this.voice.inbox.filters.push(filter);
  }

  /**
   * Remove a filter by index.
   */
  removeFilter(index: number): boolean {
    if (index < 0 || index >= this.voice.inbox.filters.length) return false;
    this.voice.inbox.filters.splice(index, 1);
    return true;
  }

  /**
   * Set an auto-responder.
   */
  setAutoResponder(responder: AutoResponder): void {
    // Replace existing responder with same condition, or add new
    const idx = this.voice.inbox.autoResponders.findIndex(
      (r) => r.condition === responder.condition,
    );
    if (idx >= 0) {
      this.voice.inbox.autoResponders[idx] = responder;
    } else {
      this.voice.inbox.autoResponders.push(responder);
    }
  }

  // ============ Queries ============

  /**
   * Get unread messages.
   */
  getUnread(): InboxEntry[] {
    return this.voice.inbox.messages.filter((e) => !e.readAt && !e.archived);
  }

  /**
   * Get messages from a specific sender.
   */
  getFromSender(sender: DID): InboxEntry[] {
    return this.voice.inbox.messages.filter((e) => e.message.sender === sender);
  }

  /**
   * Get starred messages.
   */
  getStarred(): InboxEntry[] {
    return this.voice.inbox.messages.filter((e) => e.starred);
  }

  /**
   * Get inbox summary for wakeup experience.
   */
  getInboxSummary(): {
    unreadCount: number;
    totalMessages: number;
    oldestUnread?: number;
    recentSenders: DID[];
  } {
    const recent = this.voice.inbox.messages
      .filter((e) => !e.readAt)
      .slice(-5)
      .map((e) => e.message.sender);

    // Deduplicate recent senders
    const uniqueSenders = [...new Set(recent)];

    return {
      unreadCount: this.voice.inbox.unreadCount,
      totalMessages: this.voice.inbox.messages.length,
      oldestUnread: this.voice.inbox.oldestUnread,
      recentSenders: uniqueSenders,
    };
  }

  // ============ State ============

  getVoice(): HomeVoice {
    return this.voice;
  }

  isEnabled(): boolean {
    return this.voice.enabled;
  }

  setEnabled(enabled: boolean): void {
    this.voice.enabled = enabled;
  }

  // ============ Defaults ============

  private createDefaultInbox(): HomeInbox {
    return {
      messages: [],
      unreadCount: 0,
      capacity: 100,
      filters: [],
      autoResponders: [],
    };
  }

  private createDefaultVisitorLog(): VisitorLog {
    return {
      visits: [],
      totalVisitors: 0,
      uniqueVisitorCount: 0,
    };
  }
}
