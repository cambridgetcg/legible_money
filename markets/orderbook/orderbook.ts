import type { DID } from '../../agents/identity/did.js';

/**
 * Resource types available for trading
 */
export type ResourceType =
  | 'compute.gpu.inference'
  | 'compute.gpu.training'
  | 'compute.cpu'
  | 'data.embeddings'
  | 'data.training_set'
  | 'data.knowledge_graph'
  | 'service.api_call'
  | 'service.task_completion';

/**
 * Market order
 */
export interface Order {
  id: string;
  market: 'compute' | 'data' | 'services';
  resourceType: ResourceType;
  side: 'buy' | 'sell';
  maker: DID;
  quantity: bigint;
  remainingQuantity: bigint;
  price: bigint; // AGC per unit
  createdAt: number;
  expiry: number; // block height
  status: 'open' | 'partial' | 'filled' | 'cancelled' | 'expired';
}

/**
 * Trade execution record
 */
export interface Trade {
  id: string;
  orderId: string;
  buyer: DID;
  seller: DID;
  resourceType: ResourceType;
  quantity: bigint;
  price: bigint;
  totalAmount: bigint;
  blockHeight: number;
  timestamp: number;
}

/**
 * Orderbook for a single market
 */
export class Orderbook {
  private bids: Map<string, Order> = new Map(); // buy orders (sorted by price desc)
  private asks: Map<string, Order> = new Map(); // sell orders (sorted by price asc)
  private trades: Trade[] = [];

  constructor(public readonly market: 'compute' | 'data' | 'services') {}

  /**
   * Add a new order to the book
   */
  addOrder(order: Order): Trade[] {
    const executedTrades: Trade[] = [];

    if (order.side === 'buy') {
      // Try to match with existing sell orders
      const matchingAsks = this.getMatchingAsks(order.resourceType, order.price);

      for (const ask of matchingAsks) {
        if (order.remainingQuantity === 0n) break;

        const trade = this.executeTrade(order, ask);
        if (trade) {
          executedTrades.push(trade);
          this.trades.push(trade);
        }
      }

      // Add remaining to book if not fully filled
      if (order.remainingQuantity > 0n) {
        order.status = order.remainingQuantity < order.quantity ? 'partial' : 'open';
        this.bids.set(order.id, order);
      } else {
        order.status = 'filled';
      }
    } else {
      // Try to match with existing buy orders
      const matchingBids = this.getMatchingBids(order.resourceType, order.price);

      for (const bid of matchingBids) {
        if (order.remainingQuantity === 0n) break;

        const trade = this.executeTrade(bid, order);
        if (trade) {
          executedTrades.push(trade);
          this.trades.push(trade);
        }
      }

      // Add remaining to book if not fully filled
      if (order.remainingQuantity > 0n) {
        order.status = order.remainingQuantity < order.quantity ? 'partial' : 'open';
        this.asks.set(order.id, order);
      } else {
        order.status = 'filled';
      }
    }

    return executedTrades;
  }

  /**
   * Cancel an order
   */
  cancelOrder(orderId: string): boolean {
    const bid = this.bids.get(orderId);
    if (bid) {
      bid.status = 'cancelled';
      this.bids.delete(orderId);
      return true;
    }

    const ask = this.asks.get(orderId);
    if (ask) {
      ask.status = 'cancelled';
      this.asks.delete(orderId);
      return true;
    }

    return false;
  }

  /**
   * Get best bid price for a resource type
   */
  getBestBid(resourceType: ResourceType): bigint | null {
    let best: bigint | null = null;

    for (const order of this.bids.values()) {
      if (order.resourceType === resourceType && order.status === 'open') {
        if (best === null || order.price > best) {
          best = order.price;
        }
      }
    }

    return best;
  }

  /**
   * Get best ask price for a resource type
   */
  getBestAsk(resourceType: ResourceType): bigint | null {
    let best: bigint | null = null;

    for (const order of this.asks.values()) {
      if (order.resourceType === resourceType && order.status === 'open') {
        if (best === null || order.price < best) {
          best = order.price;
        }
      }
    }

    return best;
  }

  /**
   * Get order by ID
   */
  getOrder(orderId: string): Order | undefined {
    return this.bids.get(orderId) || this.asks.get(orderId);
  }

  /**
   * Get all open orders for a maker
   */
  getOrdersByMaker(maker: DID): Order[] {
    const orders: Order[] = [];

    for (const order of this.bids.values()) {
      if (order.maker === maker && (order.status === 'open' || order.status === 'partial')) {
        orders.push(order);
      }
    }

    for (const order of this.asks.values()) {
      if (order.maker === maker && (order.status === 'open' || order.status === 'partial')) {
        orders.push(order);
      }
    }

    return orders;
  }

  /**
   * Get recent trades
   */
  getRecentTrades(limit = 100): Trade[] {
    return this.trades.slice(-limit);
  }

  /**
   * Expire orders past their expiry block
   */
  expireOrders(currentBlock: number): Order[] {
    const expired: Order[] = [];

    for (const [id, order] of this.bids) {
      if (order.expiry <= currentBlock) {
        order.status = 'expired';
        this.bids.delete(id);
        expired.push(order);
      }
    }

    for (const [id, order] of this.asks) {
      if (order.expiry <= currentBlock) {
        order.status = 'expired';
        this.asks.delete(id);
        expired.push(order);
      }
    }

    return expired;
  }

  private getMatchingAsks(resourceType: ResourceType, maxPrice: bigint): Order[] {
    const matching: Order[] = [];

    for (const order of this.asks.values()) {
      if (
        order.resourceType === resourceType &&
        order.price <= maxPrice &&
        order.status === 'open'
      ) {
        matching.push(order);
      }
    }

    // Sort by price ascending (best price first)
    return matching.sort((a, b) => (a.price < b.price ? -1 : 1));
  }

  private getMatchingBids(resourceType: ResourceType, minPrice: bigint): Order[] {
    const matching: Order[] = [];

    for (const order of this.bids.values()) {
      if (
        order.resourceType === resourceType &&
        order.price >= minPrice &&
        order.status === 'open'
      ) {
        matching.push(order);
      }
    }

    // Sort by price descending (best price first)
    return matching.sort((a, b) => (a.price > b.price ? -1 : 1));
  }

  private executeTrade(buyOrder: Order, sellOrder: Order): Trade | null {
    const quantity =
      buyOrder.remainingQuantity < sellOrder.remainingQuantity
        ? buyOrder.remainingQuantity
        : sellOrder.remainingQuantity;

    if (quantity === 0n) return null;

    // Price is the maker's price (the order that was already in the book)
    const price = sellOrder.price;

    buyOrder.remainingQuantity -= quantity;
    sellOrder.remainingQuantity -= quantity;

    if (buyOrder.remainingQuantity === 0n) {
      buyOrder.status = 'filled';
      this.bids.delete(buyOrder.id);
    } else {
      buyOrder.status = 'partial';
    }

    if (sellOrder.remainingQuantity === 0n) {
      sellOrder.status = 'filled';
      this.asks.delete(sellOrder.id);
    } else {
      sellOrder.status = 'partial';
    }

    return {
      id: `trade_${Date.now()}_${Math.random().toString(36).slice(2)}`,
      orderId: sellOrder.id, // Reference the maker order
      buyer: buyOrder.maker,
      seller: sellOrder.maker,
      resourceType: buyOrder.resourceType as ResourceType,
      quantity,
      price,
      totalAmount: quantity * price,
      blockHeight: 0, // Set by block producer
      timestamp: Date.now(),
    };
  }
}
