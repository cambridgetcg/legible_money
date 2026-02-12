/**
 * RPC Client
 *
 * Thin HTTP + WebSocket client for daemon communication with the Legible node.
 * Follows patterns from protocol/sdk/client.ts (subscribe, submitCommitment,
 * submitReveal) but stripped down for the daemon's needs.
 */

import { EventEmitter } from "node:events";
import type {
  BlockEvent,
  ActiveRound,
  ClaimData,
  RPCResponse,
  RoundPhase,
  ReferencedFact,
  EvidenceItem,
  CounterFactSummary,
  BidirectionalConfidence,
} from "./types.js";

/** WebSocket readyState constants */
const WS_OPEN = 1;

/** Reconnection config */
const INITIAL_RECONNECT_DELAY = 1000;
const MAX_RECONNECT_DELAY = 30000;
const RECONNECT_BACKOFF = 2;

export class RPCClient extends EventEmitter {
  private readonly httpUrl: string;
  private readonly wsUrl: string;
  private ws: WebSocket | null = null;
  private reconnectDelay = INITIAL_RECONNECT_DELAY;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private closed = false;
  private requestId = 0;

  constructor(httpUrl: string, wsUrl: string) {
    super();
    this.httpUrl = httpUrl;
    this.wsUrl = wsUrl;
  }

  // ============ HTTP RPC ============

  /** Generic JSON-RPC 2.0 call over HTTP */
  async call<T>(
    method: string,
    params: Record<string, unknown> = {},
  ): Promise<T> {
    const id = `daemon-${++this.requestId}`;
    const body = JSON.stringify({
      jsonrpc: "2.0",
      id,
      method,
      params,
    });

    const res = await fetch(this.httpUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body,
    });

    if (!res.ok) {
      throw new Error(`RPC HTTP error: ${res.status} ${res.statusText}`);
    }

    const json = (await res.json()) as RPCResponse<T>;
    if (json.error) {
      throw new Error(`RPC error ${json.error.code}: ${json.error.message}`);
    }

    return json.result as T;
  }

  /** Get current block height */
  async getBlockHeight(): Promise<bigint> {
    const result = await this.call<{ number: string }>("block_getLatest", {});
    return BigInt(result.number);
  }

  /** Get active verification rounds at a given height */
  async getActiveRounds(height: bigint): Promise<ActiveRound[]> {
    const result = await this.call<{ rounds: RawActiveRound[] }>(
      "verification_getActiveRounds",
      { height: height.toString() },
    );
    return (result.rounds ?? []).map(parseActiveRound);
  }

  /** Get the phase of a specific round */
  async getRoundPhase(roundId: string): Promise<RoundPhase> {
    const result = await this.call<{ round: { phase: RoundPhase } }>(
      "verification_getRound",
      { roundId },
    );
    return result.round.phase;
  }

  /** Fetch full claim data for evaluation */
  async getClaimData(claimId: string): Promise<ClaimData> {
    const result = await this.call<{ claim: RawClaimData }>(
      "knowledge_getClaim",
      { claimId },
    );
    return parseClaimData(result.claim);
  }

  /** Check if this validator is VRF-selected for a round */
  async isSelectedForRound(
    roundId: string,
    validatorAddress: string,
  ): Promise<boolean> {
    const result = await this.call<{ selected: boolean }>(
      "verification_isSelected",
      { roundId, validatorAddress },
    );
    return result.selected;
  }

  /** Get validator status (active, jailed, etc.) */
  async getValidatorStatus(validatorAddress: string): Promise<string> {
    const result = await this.call<{ status: string }>("staking_getValidator", {
      address: validatorAddress,
    });
    return result.status;
  }

  /** Get counter-facts targeting a specific fact */
  async getCounterFactsByFact(factId: string): Promise<CounterFactSummary[]> {
    const result = await this.call<{ counterFacts: CounterFactSummary[] }>(
      "knowledge_getCounterFactsByFact",
      { factId },
    );
    return result.counterFacts ?? [];
  }

  /** Get bidirectional confidence for a fact */
  async getBidirectionalConfidence(
    factId: string,
  ): Promise<BidirectionalConfidence> {
    const result = await this.call<{
      bidirectionalConfidence: BidirectionalConfidence;
    }>("knowledge_getBidirectionalConfidence", { factId });
    return result.bidirectionalConfidence;
  }

  /**
   * Submit a commitment (phase 1 of commit-reveal).
   * Mirrors protocol/sdk/client.ts:412-434 submitCommitment.
   */
  async submitCommitment(params: {
    roundId: string;
    commitmentHash: string;
    validatorAddress: string;
    signature: string;
  }): Promise<void> {
    await this.call("verification_submitCommitment", params);
  }

  /**
   * Submit a reveal (phase 2 of commit-reveal).
   * Mirrors protocol/sdk/client.ts:442-462 submitReveal.
   */
  async submitReveal(params: {
    roundId: string;
    verdict: string;
    confidence: number;
    salt: string;
    justification?: string;
    validatorAddress: string;
    signature: string;
  }): Promise<void> {
    await this.call("verification_submitReveal", params);
  }

  // ============ WebSocket ============

  /** Connect WebSocket and subscribe to block events */
  async connect(): Promise<void> {
    if (this.closed) return;

    return new Promise<void>((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.wsUrl);

        this.ws.onopen = () => {
          this.reconnectDelay = INITIAL_RECONNECT_DELAY;
          this.subscribeBlocks();
          resolve();
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(String(event.data));
        };

        this.ws.onerror = (event) => {
          this.emit("error", new Error(`WebSocket error: ${event}`));
        };

        this.ws.onclose = () => {
          if (!this.closed) {
            this.scheduleReconnect();
          }
        };
      } catch (err) {
        reject(err);
      }
    });
  }

  /** Disconnect WebSocket and stop reconnecting */
  disconnect(): void {
    this.closed = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws && this.ws.readyState === WS_OPEN) {
      this.ws.close();
    }
    this.ws = null;
  }

  /** Subscribe to block events via WebSocket (mirrors client.ts:630-659) */
  private subscribeBlocks(): void {
    if (!this.ws || this.ws.readyState !== WS_OPEN) return;

    const id = `sub-blocks-${Date.now()}`;
    this.ws.send(
      JSON.stringify({
        jsonrpc: "2.0",
        id,
        method: "subscribe",
        params: { event: "block" },
      }),
    );
  }

  /** Handle incoming WebSocket messages */
  private handleMessage(raw: string): void {
    try {
      const msg = JSON.parse(raw);

      // Subscription event
      if (msg.method === "subscription" && msg.params?.event === "block") {
        const data = msg.params.data;
        const block: BlockEvent = {
          height: BigInt(data.height ?? data.number ?? "0"),
          hash: data.hash ?? "",
          timestamp: data.timestamp ?? Date.now(),
        };
        this.emit("block", block);
        return;
      }
    } catch {
      // Ignore malformed messages
    }
  }

  /** Schedule WebSocket reconnection with exponential backoff */
  private scheduleReconnect(): void {
    if (this.closed) return;

    this.reconnectTimer = setTimeout(async () => {
      try {
        await this.connect();
      } catch {
        this.reconnectDelay = Math.min(
          this.reconnectDelay * RECONNECT_BACKOFF,
          MAX_RECONNECT_DELAY,
        );
        this.scheduleReconnect();
      }
    }, this.reconnectDelay);
  }
}

// ============ Response Parsing ============

interface RawActiveRound {
  id: string;
  claimId: string;
  phase: string;
  startHeight: string;
  commitDeadlineHeight: string;
  revealDeadlineHeight: string;
}

function parseActiveRound(raw: RawActiveRound): ActiveRound {
  return {
    id: raw.id,
    claimId: raw.claimId,
    phase: raw.phase as RoundPhase,
    startHeight: BigInt(raw.startHeight),
    commitDeadlineHeight: BigInt(raw.commitDeadlineHeight),
    revealDeadlineHeight: BigInt(raw.revealDeadlineHeight),
  };
}

interface RawClaimData {
  id: string;
  claimText?: string;
  claim?: string;
  domain: string;
  epistemicCategory?: string;
  category?: string;
  logicZone?: string;
  references?: ReferencedFact[];
  evidence?: EvidenceItem[];
  submitter: string;
  stakeAmount?: string;
  stake?: string;
}

function parseClaimData(raw: RawClaimData): ClaimData {
  return {
    id: raw.id,
    claimText: raw.claimText ?? raw.claim ?? "",
    domain: raw.domain,
    epistemicCategory: raw.epistemicCategory ?? raw.category ?? "empirical",
    logicZone: raw.logicZone ?? "empirical",
    references: raw.references ?? [],
    evidence: raw.evidence ?? [],
    submitter: raw.submitter,
    stakeAmount: raw.stakeAmount ?? raw.stake ?? "0",
  };
}
