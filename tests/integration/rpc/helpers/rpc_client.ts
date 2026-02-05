/**
 * RPC Test Client
 *
 * Test utilities for making HTTP and WebSocket RPC calls.
 * Used by RPC server integration tests.
 */

import { WebSocket } from "ws";
import type {
  RPCRequest,
  RPCResponse,
  RPCError,
} from "../../../../protocol/types.js";

// ============ HTTP Client ============

/**
 * Error thrown when RPC call returns an error
 */
export class RPCCallError extends Error {
  constructor(
    public code: number,
    message: string,
    public data?: unknown,
  ) {
    super(message);
    this.name = "RPCCallError";
  }
}

/**
 * HTTP-based RPC test client
 */
export class TestRPCClient {
  private requestId = 0;

  constructor(private baseUrl: string) {}

  /**
   * Make an RPC call and return the result
   * Throws RPCCallError if the response contains an error
   */
  async call<T>(method: string, params?: unknown): Promise<T> {
    const response = await this.callRaw({
      jsonrpc: "2.0",
      id: ++this.requestId,
      method,
      params,
    });

    if (response.error) {
      throw new RPCCallError(
        response.error.code,
        response.error.message,
        response.error.data,
      );
    }

    return response.result as T;
  }

  /**
   * Make a raw RPC call and return the full response
   * Does not throw on error responses
   */
  async callRaw(request: RPCRequest): Promise<RPCResponse> {
    const response = await fetch(this.baseUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(request),
    });

    return response.json();
  }

  /**
   * Make a raw HTTP request (for testing non-JSON-RPC scenarios)
   */
  async rawRequest(options: {
    method?: string;
    headers?: Record<string, string>;
    body?: string;
  }): Promise<Response> {
    return fetch(this.baseUrl, {
      method: options.method ?? "POST",
      headers: options.headers ?? { "Content-Type": "application/json" },
      body: options.body,
    });
  }

  /**
   * Get rate limit info from last response headers
   */
  async getRateLimitInfo(): Promise<{
    limit: number;
    remaining: number;
    reset: number;
    tier: string;
  }> {
    const response = await fetch(this.baseUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        jsonrpc: "2.0",
        id: ++this.requestId,
        method: "network_health",
      }),
    });

    return {
      limit: parseInt(response.headers.get("X-RateLimit-Limit") || "0"),
      remaining: parseInt(response.headers.get("X-RateLimit-Remaining") || "0"),
      reset: parseInt(response.headers.get("X-RateLimit-Reset") || "0"),
      tier: response.headers.get("X-RateLimit-Tier") || "unknown",
    };
  }
}

// ============ WebSocket Client ============

/**
 * WebSocket-based RPC test client
 */
export class TestWSClient {
  private ws: WebSocket | null = null;
  private requestId = 0;
  private pendingRequests = new Map<
    number,
    {
      resolve: (value: RPCResponse) => void;
      reject: (error: Error) => void;
    }
  >();
  private eventHandlers: Array<(event: unknown) => void> = [];
  private subscriptions = new Map<string, boolean>();

  constructor(private wsUrl: string) {}

  /**
   * Connect to the WebSocket server
   */
  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(this.wsUrl);

      this.ws.on("open", () => {
        resolve();
      });

      this.ws.on("error", (error) => {
        reject(error);
      });

      this.ws.on("message", (data) => {
        try {
          const message = JSON.parse(data.toString());

          // Check if this is a response to a pending request
          if (
            message.id !== undefined &&
            this.pendingRequests.has(message.id)
          ) {
            const pending = this.pendingRequests.get(message.id)!;
            this.pendingRequests.delete(message.id);
            pending.resolve(message);
            return;
          }

          // Otherwise, it's a subscription event
          if (message.method === "subscription") {
            for (const handler of this.eventHandlers) {
              handler(message.params);
            }
          }
        } catch (error) {
          console.error("Failed to parse WebSocket message:", error);
        }
      });

      this.ws.on("close", () => {
        // Reject all pending requests
        for (const [id, pending] of this.pendingRequests) {
          pending.reject(new Error("WebSocket closed"));
          this.pendingRequests.delete(id);
        }
      });
    });
  }

  /**
   * Disconnect from the WebSocket server
   */
  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.pendingRequests.clear();
    this.eventHandlers = [];
    this.subscriptions.clear();
  }

  /**
   * Make an RPC call over WebSocket
   */
  async call<T>(method: string, params?: unknown): Promise<T> {
    const response = await this.callRaw({
      jsonrpc: "2.0",
      id: ++this.requestId,
      method,
      params,
    });

    if (response.error) {
      throw new RPCCallError(
        response.error.code,
        response.error.message,
        response.error.data,
      );
    }

    return response.result as T;
  }

  /**
   * Make a raw RPC call over WebSocket
   */
  callRaw(request: RPCRequest): Promise<RPCResponse> {
    return new Promise((resolve, reject) => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        reject(new Error("WebSocket not connected"));
        return;
      }

      this.pendingRequests.set(request.id as number, { resolve, reject });
      this.ws.send(JSON.stringify(request));

      // Timeout after 10 seconds
      setTimeout(() => {
        if (this.pendingRequests.has(request.id as number)) {
          this.pendingRequests.delete(request.id as number);
          reject(new Error("Request timeout"));
        }
      }, 10000);
    });
  }

  /**
   * Subscribe to an event
   */
  async subscribe(
    event: string,
  ): Promise<{ subscribed: boolean; event: string }> {
    const result = await this.call<{ subscribed: boolean; event: string }>(
      "subscribe",
      { event },
    );
    if (result.subscribed) {
      this.subscriptions.set(event, true);
    }
    return result;
  }

  /**
   * Unsubscribe from events
   */
  async unsubscribe(
    subscriptionId?: string,
  ): Promise<{ unsubscribed: boolean }> {
    const result = await this.call<{ unsubscribed: boolean }>("unsubscribe", {
      subscriptionId,
    });
    if (result.unsubscribed) {
      this.subscriptions.clear();
    }
    return result;
  }

  /**
   * Register an event handler for subscription events
   */
  onEvent(handler: (event: unknown) => void): () => void {
    this.eventHandlers.push(handler);
    return () => {
      const index = this.eventHandlers.indexOf(handler);
      if (index !== -1) {
        this.eventHandlers.splice(index, 1);
      }
    };
  }

  /**
   * Wait for a specific event
   */
  waitForEvent(timeout = 5000): Promise<unknown> {
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        cleanup();
        reject(new Error("Event timeout"));
      }, timeout);

      const cleanup = this.onEvent((event) => {
        clearTimeout(timer);
        cleanup();
        resolve(event);
      });
    });
  }

  /**
   * Check if connected
   */
  get isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }
}

// ============ Utilities ============

/**
 * Get the server port from address info
 */
export function getServerPort(server: {
  address: () => { port: number } | string | null;
}): number {
  const addr = server.address();
  if (typeof addr === "object" && addr !== null) {
    return addr.port;
  }
  throw new Error("Unable to get server port");
}
