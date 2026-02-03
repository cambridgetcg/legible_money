/**
 * RPC Server
 *
 * JSON-RPC 2.0 server for Legible Money.
 * Supports HTTP and WebSocket connections.
 */

import { createServer, type IncomingMessage, type ServerResponse } from 'http';
import { WebSocketServer, type WebSocket } from 'ws';
import { handlers, type HandlerContext } from './handlers.js';
import type {
  RPCRequest,
  RPCResponse,
  RPCError,
  RateLimitInfo,
} from '../types.js';
import { RPC_ERROR_CODES } from '../types.js';

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
}

export const DEFAULT_SERVER_CONFIG: ServerConfig = {
  port: 8545,
  host: '0.0.0.0',
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
    enabled: true,
    origins: ['*'],
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
        this.wsServer.on('connection', (ws, req) => {
          this.handleWebSocketConnection(ws, req);
        });
      }

      // Start listening
      this.httpServer.listen(this.config.port, this.config.host, () => {
        console.log(`RPC server listening on ${this.config.host}:${this.config.port}`);
        if (this.config.enableWebSocket) {
          console.log(`WebSocket enabled on ws://${this.config.host}:${this.config.port}`);
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

  // ============ HTTP Handling ============

  private async handleHttpRequest(req: IncomingMessage, res: ServerResponse): Promise<void> {
    // CORS headers
    if (this.config.cors.enabled) {
      res.setHeader('Access-Control-Allow-Origin', this.config.cors.origins.join(', '));
      res.setHeader('Access-Control-Allow-Methods', 'POST, OPTIONS');
      res.setHeader('Access-Control-Allow-Headers', 'Content-Type');
    }

    // Handle preflight
    if (req.method === 'OPTIONS') {
      res.writeHead(204);
      res.end();
      return;
    }

    // Only accept POST
    if (req.method !== 'POST') {
      res.writeHead(405, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Method not allowed' }));
      return;
    }

    // Get client IP for rate limiting
    const clientIp = this.getClientIp(req);

    // Check rate limit
    const rateLimit = this.checkRateLimit(clientIp);
    res.setHeader('X-RateLimit-Limit', rateLimit.limit.toString());
    res.setHeader('X-RateLimit-Remaining', rateLimit.remaining.toString());
    res.setHeader('X-RateLimit-Reset', rateLimit.reset.toString());
    res.setHeader('X-RateLimit-Tier', rateLimit.tier);

    if (rateLimit.remaining <= 0) {
      res.writeHead(429, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(this.createErrorResponse(
        0,
        RPC_ERROR_CODES.RATE_LIMITED,
        'Rate limit exceeded',
        { retryAfter: rateLimit.reset - Date.now() }
      )));
      return;
    }

    // Parse body
    try {
      const body = await this.readBody(req);
      const request: RPCRequest = JSON.parse(body);

      // Process request
      const response = await this.processRequest(request);

      // Send response
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(response));

    } catch (error) {
      res.writeHead(400, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(this.createErrorResponse(
        0,
        RPC_ERROR_CODES.PARSE_ERROR,
        'Invalid JSON'
      )));
    }
  }

  private async readBody(req: IncomingMessage): Promise<string> {
    return new Promise((resolve, reject) => {
      let body = '';
      let size = 0;

      req.on('data', (chunk) => {
        size += chunk.length;
        if (size > this.config.maxRequestSize) {
          reject(new Error('Request too large'));
          return;
        }
        body += chunk;
      });

      req.on('end', () => resolve(body));
      req.on('error', reject);
    });
  }

  // ============ WebSocket Handling ============

  private handleWebSocketConnection(ws: WebSocket, req: IncomingMessage): void {
    const clientIp = this.getClientIp(req);

    ws.on('message', async (data) => {
      try {
        const message = JSON.parse(data.toString());

        if (message.method === 'subscribe') {
          this.handleSubscribe(ws, message);
        } else if (message.method === 'unsubscribe') {
          this.handleUnsubscribe(ws, message);
        } else {
          // Regular RPC call
          const response = await this.processRequest(message);
          ws.send(JSON.stringify(response));
        }
      } catch (error) {
        ws.send(JSON.stringify(this.createErrorResponse(
          0,
          RPC_ERROR_CODES.PARSE_ERROR,
          'Invalid message'
        )));
      }
    });

    ws.on('close', () => {
      // Remove from all subscriptions
      for (const subscribers of this.subscriptions.values()) {
        subscribers.delete(ws);
      }
    });
  }

  private handleSubscribe(ws: WebSocket, message: RPCRequest): void {
    const event = (message.params as any)?.event;
    if (!event) {
      ws.send(JSON.stringify(this.createErrorResponse(
        message.id,
        RPC_ERROR_CODES.INVALID_PARAMS,
        'Event required'
      )));
      return;
    }

    if (!this.subscriptions.has(event)) {
      this.subscriptions.set(event, new Set());
    }
    this.subscriptions.get(event)!.add(ws);

    ws.send(JSON.stringify({
      jsonrpc: '2.0',
      id: message.id,
      result: { subscribed: true, event },
    }));
  }

  private handleUnsubscribe(ws: WebSocket, message: RPCRequest): void {
    const subscriptionId = (message.params as any)?.subscriptionId;
    for (const subscribers of this.subscriptions.values()) {
      subscribers.delete(ws);
    }

    ws.send(JSON.stringify({
      jsonrpc: '2.0',
      id: message.id,
      result: { unsubscribed: true },
    }));
  }

  /**
   * Broadcast event to subscribers
   */
  broadcast(event: string, data: unknown): void {
    const subscribers = this.subscriptions.get(event);
    if (!subscribers) return;

    const message = JSON.stringify({
      jsonrpc: '2.0',
      method: 'subscription',
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
    // Validate request
    if (request.jsonrpc !== '2.0') {
      return this.createErrorResponse(request.id, RPC_ERROR_CODES.INVALID_REQUEST, 'Invalid JSON-RPC version');
    }

    if (!request.method) {
      return this.createErrorResponse(request.id, RPC_ERROR_CODES.INVALID_REQUEST, 'Method required');
    }

    // Find handler
    const handler = handlers[request.method];
    if (!handler) {
      return this.createErrorResponse(request.id, RPC_ERROR_CODES.METHOD_NOT_FOUND, `Method '${request.method}' not found`);
    }

    // Execute handler
    try {
      const result = await handler(request.params || {}, this.context);
      return {
        jsonrpc: '2.0',
        id: request.id,
        result,
      };
    } catch (error) {
      if (isRPCError(error)) {
        return this.createErrorResponse(request.id, error.code, error.message, error.data);
      }

      console.error('Handler error:', error);
      return this.createErrorResponse(
        request.id,
        RPC_ERROR_CODES.INTERNAL_ERROR,
        'Internal error'
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
        tier: 'free' as const,
      };
      this.rateLimitStore.set(clientIp, entry);
    }

    entry.count++;

    const limit = this.config.rateLimits.maxRequests[entry.tier];

    return {
      limit,
      remaining: Math.max(0, limit - entry.count),
      reset: entry.resetAt,
      tier: entry.tier,
    };
  }

  // ============ Helpers ============

  private getClientIp(req: IncomingMessage): string {
    const forwarded = req.headers['x-forwarded-for'];
    if (typeof forwarded === 'string') {
      return forwarded.split(',')[0].trim();
    }
    return req.socket.remoteAddress || 'unknown';
  }

  private createErrorResponse(
    id: string | number,
    code: number,
    message: string,
    data?: unknown
  ): RPCResponse {
    return {
      jsonrpc: '2.0',
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

interface RateLimitEntry {
  count: number;
  resetAt: number;
  tier: 'free' | 'staked100' | 'staked1000' | 'validator';
}

function isRPCError(error: unknown): error is RPCError {
  return typeof error === 'object' && error !== null && 'code' in error && 'message' in error;
}

// ============ Factory ============

/**
 * Create and start RPC server
 */
export async function createRPCServer(
  context: HandlerContext,
  config: Partial<ServerConfig> = {}
): Promise<RPCServer> {
  const server = new RPCServer(context, config);
  await server.start();
  return server;
}
