/**
 * RPC Server
 *
 * JSON-RPC 2.0 server for Legible Money.
 * Supports HTTP and WebSocket connections.
 *
 * SECURITY FEATURES:
 * - Binds to localhost by default (127.0.0.1)
 * - CORS disabled by default, requires explicit origin whitelist
 * - Rate limiting per IP (HTTP) and per connection (WebSocket)
 * - Never trusts X-Forwarded-For (prevents rate limit bypass)
 * - Security headers on all responses
 * - Request size limits
 * - Production mode enforcement
 */

import { createServer, type IncomingMessage, type ServerResponse } from "http";
// @ts-expect-error - ws module lacks type declarations
import { WebSocketServer, type WebSocket } from "ws";
import { handlers, type HandlerContext } from "./handlers.js";
import type {
  RPCRequest,
  RPCResponse,
  RPCError,
  RateLimitInfo,
} from "../types.js";
import { RPC_ERROR_CODES } from "../types.js";

// ============ Security Constants ============

/** Maximum allowed origins to prevent DoS via config */
const MAX_CORS_ORIGINS = 10;

/** Methods that should never be exposed via RPC */
const BLOCKED_METHODS = ["__proto__", "constructor", "prototype"];

/** Check if running in production mode */
const IS_PRODUCTION = process.env.NODE_ENV === "production";

// ============ Server Configuration ============

export interface ServerConfig {
  port: number;
  host: string;
  enableWebSocket: boolean;
  maxRequestSize: number;
  rateLimits: RateLimitConfig;
  cors: CorsConfig;
}

export interface RateLimitConfig {
  windowMs: number;
  maxRequests: {
    free: number;
    staked100: number;
    staked1000: number;
    validator: number;
  };
}

export interface CorsConfig {
  enabled: boolean;
  origins: string[];
  /** Max age for preflight cache in seconds (default: 86400 = 24 hours) */
  maxAge?: number;
  /** Allow credentials (cookies, auth headers) - use with caution */
  credentials?: boolean;
}

/**
 * Validate CORS origin format
 * SECURITY: Ensures origins are valid URLs, not wildcards
 */
function validateCorsOrigin(origin: string): boolean {
  // Reject wildcards
  if (origin === "*" || origin.includes("*")) {
    return false;
  }
  // Must be a valid URL with protocol
  try {
    const url = new URL(origin);
    return url.protocol === "http:" || url.protocol === "https:";
  } catch {
    return false;
  }
}

/**
 * Validate entire CORS configuration
 * SECURITY: Prevents common misconfigurations
 */
function validateCorsConfig(config: CorsConfig): {
  valid: boolean;
  errors: string[];
} {
  const errors: string[] = [];

  if (config.enabled && config.origins.length === 0) {
    errors.push(
      "CORS enabled but no origins specified - this blocks all cross-origin requests",
    );
  }

  if (config.origins.length > MAX_CORS_ORIGINS) {
    errors.push(
      `Too many CORS origins (${config.origins.length}), max is ${MAX_CORS_ORIGINS}`,
    );
  }

  for (const origin of config.origins) {
    if (!validateCorsOrigin(origin)) {
      errors.push(
        `Invalid CORS origin: "${origin}" - must be valid URL without wildcards`,
      );
    }
  }

  if (config.credentials && config.origins.some((o) => o === "*")) {
    errors.push("Cannot use credentials with wildcard origin");
  }

  return { valid: errors.length === 0, errors };
}

export const DEFAULT_SERVER_CONFIG: ServerConfig = {
  port: 8545,
  host: "127.0.0.1", // SECURITY: Bind to localhost by default, not 0.0.0.0
  enableWebSocket: true,
  maxRequestSize: 1024 * 1024, // 1 MB
  rateLimits: {
    windowMs: 60000, // 1 minute
    maxRequests: {
      free: 60,
      staked100: 300,
      staked1000: 1000,
      validator: 10000,
    },
  },
  cors: {
    enabled: false, // SECURITY: CORS disabled by default - enable with explicit whitelist
    origins: [], // SECURITY: No wildcard - must explicitly whitelist origins
    maxAge: 86400, // 24 hours preflight cache
    credentials: false,
  },
};

// ============ RPC Server ============

export class RPCServer {
  private config: ServerConfig;
  private context: HandlerContext;
  private httpServer: ReturnType<typeof createServer> | null = null;
  private wsServer: WebSocketServer | null = null;
  private rateLimitStore: Map<string, RateLimitEntry> = new Map();
  private subscriptions: Map<string, Set<WebSocket>> = new Map();

  constructor(context: HandlerContext, config: Partial<ServerConfig> = {}) {
    this.config = { ...DEFAULT_SERVER_CONFIG, ...config };
    this.context = context;

    // SECURITY: Validate configuration
    this.validateConfig();
  }

  /**
   * SECURITY: Validate server configuration
   * Throws in production mode for dangerous configurations
   */
  private validateConfig(): void {
    const warnings: string[] = [];
    const errors: string[] = [];

    // Check host binding
    if (this.config.host === "0.0.0.0") {
      const msg =
        "Server bound to 0.0.0.0 (all interfaces) - use 127.0.0.1 for local-only access";
      if (IS_PRODUCTION) {
        errors.push(msg);
      } else {
        warnings.push(msg);
      }
    }

    // Validate CORS config
    if (this.config.cors.enabled) {
      const corsValidation = validateCorsConfig(this.config.cors);
      if (!corsValidation.valid) {
        if (IS_PRODUCTION) {
          errors.push(...corsValidation.errors);
        } else {
          warnings.push(...corsValidation.errors);
        }
      }
    }

    // Check for overly permissive rate limits
    if (this.config.rateLimits.maxRequests.free > 1000) {
      warnings.push(
        "Free tier rate limit is very high (>1000/min) - potential DoS risk",
      );
    }

    // Log warnings
    for (const warning of warnings) {
      console.warn(`[RPC Security Warning] ${warning}`);
    }

    // Throw errors in production
    if (errors.length > 0) {
      throw new Error(
        `[RPC Security Error] Configuration rejected:\n  - ${errors.join("\n  - ")}`,
      );
    }
  }

  /**
   * Start the server
   */
  async start(): Promise<void> {
    return new Promise((resolve) => {
      // Create HTTP server
      this.httpServer = createServer(async (req, res) => {
        await this.handleHttpRequest(req, res);
      });

      // Create WebSocket server if enabled
      if (this.config.enableWebSocket) {
        this.wsServer = new WebSocketServer({ server: this.httpServer });
        this.wsServer.on(
          "connection",
          (ws: WebSocket, req: IncomingMessage) => {
            this.handleWebSocketConnection(ws, req);
          },
        );
      }

      // Start listening
      this.httpServer.listen(this.config.port, this.config.host, () => {
        console.log(
          `RPC server listening on ${this.config.host}:${this.config.port}`,
        );
        if (this.config.enableWebSocket) {
          console.log(
            `WebSocket enabled on ws://${this.config.host}:${this.config.port}`,
          );
        }
        resolve();
      });
    });
  }

  /**
   * Stop the server
   */
  async stop(): Promise<void> {
    return new Promise((resolve) => {
      if (this.wsServer) {
        this.wsServer.close();
      }
      if (this.httpServer) {
        this.httpServer.close(() => resolve());
      } else {
        resolve();
      }
    });
  }

  /**
   * Get the server's bound address info
   * Useful for tests using port 0 (random port)
   */
  getAddress(): { host: string; port: number } | null {
    if (!this.httpServer) return null;
    const addr = this.httpServer.address();
    if (typeof addr === "object" && addr !== null) {
      return { host: addr.address, port: addr.port };
    }
    return null;
  }

  // ============ HTTP Handling ============

  private async handleHttpRequest(
    req: IncomingMessage,
    res: ServerResponse,
  ): Promise<void> {
    // SECURITY: Set security headers on ALL responses
    this.setSecurityHeaders(res);

    // CORS handling
    const corsResult = this.handleCors(req, res);
    if (!corsResult.allowed) {
      // Origin not in whitelist - don't set CORS headers, browser will block
      // Continue processing anyway (same-origin requests still work)
    }

    // Handle preflight
    if (req.method === "OPTIONS") {
      if (corsResult.allowed) {
        res.writeHead(204);
      } else {
        res.writeHead(403);
      }
      res.end();
      return;
    }

    // DEVNET FIX: Handle /health endpoint for Docker health checks
    if (
      req.method === "GET" &&
      (req.url === "/health" || req.url === "/health/")
    ) {
      await this.handleHealthCheck(res);
      return;
    }

    // Only accept POST for RPC
    if (req.method !== "POST") {
      res.writeHead(405, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ error: "Method not allowed" }));
      return;
    }

    // Get client IP for rate limiting
    const clientIp = this.getClientIp(req);

    // Check rate limit
    const rateLimit = this.checkRateLimit(clientIp);
    res.setHeader("X-RateLimit-Limit", rateLimit.limit.toString());
    res.setHeader("X-RateLimit-Remaining", rateLimit.remaining.toString());
    res.setHeader("X-RateLimit-Reset", rateLimit.reset.toString());
    res.setHeader("X-RateLimit-Tier", rateLimit.tier);

    if (rateLimit.remaining <= 0) {
      res.writeHead(429, { "Content-Type": "application/json" });
      res.end(
        JSON.stringify(
          this.createErrorResponse(
            0,
            RPC_ERROR_CODES.RATE_LIMITED,
            "Rate limit exceeded",
            { retryAfter: rateLimit.reset - Date.now() },
          ),
        ),
      );
      return;
    }

    // Parse body
    try {
      const body = await this.readBody(req);
      const request: RPCRequest = JSON.parse(body);

      // Process request
      const response = await this.processRequest(request);

      // Serialize response (handle BigInt values that may have leaked through)
      let responseJson: string;
      try {
        responseJson = JSON.stringify(response, (_, v) =>
          typeof v === "bigint" ? v.toString() : v,
        );
      } catch (serializeError) {
        console.error("Response serialization error:", serializeError);
        res.writeHead(500, { "Content-Type": "application/json" });
        res.end(
          JSON.stringify(
            this.createErrorResponse(
              request.id,
              RPC_ERROR_CODES.INTERNAL_ERROR,
              "Response serialization failed",
            ),
          ),
        );
        return;
      }

      // Send response
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(responseJson);
    } catch (error) {
      // Guard against headers already sent
      if (res.headersSent) {
        console.error("Error after headers sent:", error);
        return;
      }
      res.writeHead(400, { "Content-Type": "application/json" });
      res.end(
        JSON.stringify(
          this.createErrorResponse(
            0,
            RPC_ERROR_CODES.PARSE_ERROR,
            "Invalid JSON",
          ),
        ),
      );
    }
  }

  /**
   * SECURITY: Set security headers on all responses
   */
  private setSecurityHeaders(res: ServerResponse): void {
    // Prevent MIME type sniffing
    res.setHeader("X-Content-Type-Options", "nosniff");

    // Prevent clickjacking
    res.setHeader("X-Frame-Options", "DENY");

    // Strict transport security (only effective over HTTPS)
    res.setHeader(
      "Strict-Transport-Security",
      "max-age=31536000; includeSubDomains",
    );

    // Disable caching for RPC responses
    res.setHeader("Cache-Control", "no-store, no-cache, must-revalidate");
    res.setHeader("Pragma", "no-cache");

    // Content security policy for JSON API
    res.setHeader(
      "Content-Security-Policy",
      "default-src 'none'; frame-ancestors 'none'",
    );
  }

  /**
   * SECURITY: Handle CORS with proper origin validation
   * Returns whether the request origin is allowed
   */
  private handleCors(
    req: IncomingMessage,
    res: ServerResponse,
  ): { allowed: boolean } {
    if (!this.config.cors.enabled) {
      return { allowed: true }; // CORS not enabled, same-origin only
    }

    const requestOrigin = req.headers.origin;

    // No origin header = same-origin request, allow
    if (!requestOrigin) {
      return { allowed: true };
    }

    // Check if origin is in whitelist
    const isAllowed = this.config.cors.origins.includes(requestOrigin);

    if (isAllowed) {
      // SECURITY: Reflect the specific allowed origin, never use wildcard or join
      res.setHeader("Access-Control-Allow-Origin", requestOrigin);
      res.setHeader("Access-Control-Allow-Methods", "POST, OPTIONS");
      res.setHeader(
        "Access-Control-Allow-Headers",
        "Content-Type, Authorization",
      );
      res.setHeader(
        "Access-Control-Max-Age",
        String(this.config.cors.maxAge || 86400),
      );

      // Vary header is critical for caching with multiple origins
      res.setHeader("Vary", "Origin");

      if (this.config.cors.credentials) {
        res.setHeader("Access-Control-Allow-Credentials", "true");
      }
    }

    return { allowed: isAllowed };
  }

  private async readBody(req: IncomingMessage): Promise<string> {
    return new Promise((resolve, reject) => {
      let body = "";
      let size = 0;

      req.on("data", (chunk) => {
        size += chunk.length;
        if (size > this.config.maxRequestSize) {
          reject(new Error("Request too large"));
          return;
        }
        body += chunk;
      });

      req.on("end", () => resolve(body));
      req.on("error", reject);
    });
  }

  // ============ Health Check ============

  /**
   * DEVNET FIX: Handle /health HTTP GET endpoint
   * Used by Docker health checks and monitoring tools
   */
  private async handleHealthCheck(res: ServerResponse): Promise<void> {
    try {
      // Get node status from context
      const blockNumber = this.context.state.getBlockNumber();
      const nodeVersion = this.context.nodeVersion;

      // Determine sync status
      // Consider synced if we have at least block 0 and consensus is running
      const isSynced = blockNumber >= 0n;

      // Build health response
      const health = {
        status: "healthy" as const,
        blockHeight: blockNumber.toString(),
        isSynced,
        peerCount: this.context.network?.getPeerCount() ?? 0,
        version: nodeVersion,
        timestamp: Date.now(),
      };

      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify(health));
    } catch (error) {
      // If we can't even get basic info, node is unhealthy
      const unhealthy = {
        status: "unhealthy" as const,
        error: error instanceof Error ? error.message : "Unknown error",
        timestamp: Date.now(),
      };

      res.writeHead(503, { "Content-Type": "application/json" });
      res.end(JSON.stringify(unhealthy));
    }
  }

  // ============ WebSocket Handling ============

  // SECURITY: Track WebSocket rate limits per connection
  private wsRateLimits: WeakMap<WebSocket, { count: number; resetAt: number }> =
    new WeakMap();

  private handleWebSocketConnection(ws: WebSocket, req: IncomingMessage): void {
    // Get client IP for logging (not used for rate limiting to prevent bypass)
    const clientIp = this.getClientIp(req);
    console.log(`[RPC] WebSocket connection from ${clientIp}`);

    // SECURITY: Initialize rate limit tracking for this WebSocket (per-connection, not per-IP)
    this.wsRateLimits.set(ws, {
      count: 0,
      resetAt: Date.now() + this.config.rateLimits.windowMs,
    });

    ws.on("message", async (data: unknown) => {
      // SECURITY: Check WebSocket rate limit before processing
      const rateLimit = this.checkWebSocketRateLimit(ws);
      if (rateLimit.remaining <= 0) {
        ws.send(
          JSON.stringify(
            this.createErrorResponse(
              0,
              RPC_ERROR_CODES.RATE_LIMITED,
              "WebSocket rate limit exceeded",
              { retryAfter: rateLimit.reset - Date.now() },
            ),
          ),
        );
        return;
      }

      try {
        const message = JSON.parse(String(data));

        if (message.method === "subscribe") {
          this.handleSubscribe(ws, message);
        } else if (message.method === "unsubscribe") {
          this.handleUnsubscribe(ws, message);
        } else {
          // Regular RPC call
          const response = await this.processRequest(message);
          ws.send(JSON.stringify(response));
        }
      } catch (error) {
        ws.send(
          JSON.stringify(
            this.createErrorResponse(
              0,
              RPC_ERROR_CODES.PARSE_ERROR,
              "Invalid message",
            ),
          ),
        );
      }
    });

    ws.on("close", () => {
      // Remove from all subscriptions
      for (const subscribers of this.subscriptions.values()) {
        subscribers.delete(ws);
      }
      // Rate limit cleanup happens automatically via WeakMap
    });
  }

  /**
   * SECURITY: Check rate limit for WebSocket connection
   */
  private checkWebSocketRateLimit(ws: WebSocket): RateLimitInfo {
    const now = Date.now();
    let entry = this.wsRateLimits.get(ws);

    if (!entry || now > entry.resetAt) {
      entry = {
        count: 0,
        resetAt: now + this.config.rateLimits.windowMs,
      };
      this.wsRateLimits.set(ws, entry);
    }

    entry.count++;

    // Use same rate limits as HTTP (free tier)
    const limit = this.config.rateLimits.maxRequests.free;

    return {
      limit,
      remaining: Math.max(0, limit - entry.count),
      reset: entry.resetAt,
      tier: "free",
    };
  }

  private handleSubscribe(ws: WebSocket, message: RPCRequest): void {
    const event = (message.params as any)?.event;
    if (!event) {
      ws.send(
        JSON.stringify(
          this.createErrorResponse(
            message.id,
            RPC_ERROR_CODES.INVALID_PARAMS,
            "Event required",
          ),
        ),
      );
      return;
    }

    if (!this.subscriptions.has(event)) {
      this.subscriptions.set(event, new Set());
    }
    this.subscriptions.get(event)!.add(ws);

    ws.send(
      JSON.stringify({
        jsonrpc: "2.0",
        id: message.id,
        result: { subscribed: true, event },
      }),
    );
  }

  private handleUnsubscribe(ws: WebSocket, message: RPCRequest): void {
    // subscriptionId available at (message.params as any)?.subscriptionId for future per-subscription removal
    for (const subscribers of this.subscriptions.values()) {
      subscribers.delete(ws);
    }

    ws.send(
      JSON.stringify({
        jsonrpc: "2.0",
        id: message.id,
        result: { unsubscribed: true },
      }),
    );
  }

  /**
   * Broadcast event to subscribers
   */
  broadcast(event: string, data: unknown): void {
    const subscribers = this.subscriptions.get(event);
    if (!subscribers) return;

    const message = JSON.stringify({
      jsonrpc: "2.0",
      method: "subscription",
      params: {
        subscriptionId: event,
        event: data,
      },
    });

    for (const ws of subscribers) {
      if (ws.readyState === ws.OPEN) {
        ws.send(message);
      }
    }
  }

  // ============ Request Processing ============

  private async processRequest(request: RPCRequest): Promise<RPCResponse> {
    // Validate request structure
    if (request.jsonrpc !== "2.0") {
      return this.createErrorResponse(
        request.id,
        RPC_ERROR_CODES.INVALID_REQUEST,
        "Invalid JSON-RPC version",
      );
    }

    if (!request.method || typeof request.method !== "string") {
      return this.createErrorResponse(
        request.id,
        RPC_ERROR_CODES.INVALID_REQUEST,
        "Method required and must be string",
      );
    }

    // SECURITY: Validate method name format
    if (request.method.length > 100) {
      return this.createErrorResponse(
        request.id,
        RPC_ERROR_CODES.INVALID_REQUEST,
        "Method name too long",
      );
    }

    // SECURITY: Block prototype pollution attempts
    if (
      BLOCKED_METHODS.includes(request.method) ||
      request.method.startsWith("_")
    ) {
      return this.createErrorResponse(
        request.id,
        RPC_ERROR_CODES.METHOD_NOT_FOUND,
        "Method not found",
      );
    }

    // SECURITY: Validate request ID (prevent huge IDs that could cause issues)
    if (request.id !== undefined && request.id !== null) {
      if (typeof request.id === "string" && request.id.length > 100) {
        return this.createErrorResponse(
          0,
          RPC_ERROR_CODES.INVALID_REQUEST,
          "Request ID too long",
        );
      }
      if (typeof request.id === "number" && !Number.isFinite(request.id)) {
        return this.createErrorResponse(
          0,
          RPC_ERROR_CODES.INVALID_REQUEST,
          "Invalid request ID",
        );
      }
    }

    // Find handler - use hasOwnProperty to prevent prototype access
    if (!Object.prototype.hasOwnProperty.call(handlers, request.method)) {
      return this.createErrorResponse(
        request.id,
        RPC_ERROR_CODES.METHOD_NOT_FOUND,
        `Method '${request.method}' not found`,
      );
    }
    const handler = handlers[request.method];

    // Execute handler
    try {
      const result = await handler(request.params || {}, this.context);
      return {
        jsonrpc: "2.0",
        id: request.id,
        result,
      };
    } catch (error) {
      if (isRPCError(error)) {
        return this.createErrorResponse(
          request.id,
          error.code,
          error.message,
          error.data,
        );
      }

      console.error("Handler error:", error);
      return this.createErrorResponse(
        request.id,
        RPC_ERROR_CODES.INTERNAL_ERROR,
        `Internal error: ${error instanceof Error ? error.message : String(error)}`,
      );
    }
  }

  // ============ Rate Limiting ============

  private checkRateLimit(clientIp: string): RateLimitInfo {
    const now = Date.now();
    let entry = this.rateLimitStore.get(clientIp);

    if (!entry || now > entry.resetAt) {
      entry = {
        count: 0,
        resetAt: now + this.config.rateLimits.windowMs,
        tier: "free" as const,
      };
      this.rateLimitStore.set(clientIp, entry);
    }

    entry.count++;

    const limit = this.config.rateLimits.maxRequests[entry.tier];

    return {
      limit,
      remaining: Math.max(0, limit - entry.count),
      reset: entry.resetAt,
      tier: toApiTier(entry.tier),
    };
  }

  // ============ Helpers ============

  private getClientIp(req: IncomingMessage): string {
    // SECURITY: Never trust X-Forwarded-For header - it can be spoofed to bypass rate limits.
    // If running behind a reverse proxy, configure the proxy to set a trusted header
    // and verify the connection comes from the proxy's known IP.
    return req.socket.remoteAddress || "unknown";
  }

  private createErrorResponse(
    id: string | number,
    code: number,
    message: string,
    data?: unknown,
  ): RPCResponse {
    return {
      jsonrpc: "2.0",
      id,
      error: {
        code,
        message,
        data: data as any,
      },
    };
  }
}

// ============ Types ============

/** Internal tier representation (matches config keys) */
type InternalTier = "free" | "staked100" | "staked1000" | "validator";

/** API tier representation (matches RateLimitInfo) */
type ApiTier = "free" | "staked-100" | "staked-1000" | "validator";

interface RateLimitEntry {
  count: number;
  resetAt: number;
  tier: InternalTier;
}

/** Convert internal tier name to API tier name */
function toApiTier(internal: InternalTier): ApiTier {
  switch (internal) {
    case "staked100":
      return "staked-100";
    case "staked1000":
      return "staked-1000";
    default:
      return internal;
  }
}

function isRPCError(error: unknown): error is RPCError {
  return (
    typeof error === "object" &&
    error !== null &&
    "code" in error &&
    "message" in error
  );
}

// ============ Factory ============

/**
 * Create and start RPC server
 */
export async function createRPCServer(
  context: HandlerContext,
  config: Partial<ServerConfig> = {},
): Promise<RPCServer> {
  const server = new RPCServer(context, config);
  await server.start();
  return server;
}
