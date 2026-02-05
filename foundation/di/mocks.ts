/**
 * Mock Implementations for Testing
 *
 * In-memory mock implementations of all contracts.
 * Use these in tests to isolate domain logic from infrastructure.
 *
 * Design principle: Mocks should be simple and predictable.
 * They store state in memory and emit events synchronously.
 */

import type {
  DID,
  PublicKey,
  Amount,
  Stake,
  BlockHeight,
  MerkleRoot,
  Confidence,
} from "../types/index.js";

import type {
  StateProvider,
  ValidatorInfo,
  FactInfo,
  TransactionHandle,
  EventBus,
  ProtocolEvent,
  ProtocolEventType,
  EventHandler,
  Subscription,
} from "../contracts/index.js";

// ============ Mock State Provider ============

/**
 * In-memory mock state provider
 */
export class MockStateProvider implements StateProvider {
  private accounts = new Map<DID, MockAccount>();
  private validators = new Map<PublicKey, ValidatorInfo>();
  private facts = new Map<string, FactInfo>();
  private currentBlock: BlockHeight = 0n;
  private txCounter = 0;

  // === Account Operations ===

  async getBalance(did: DID): Promise<Amount> {
    return this.accounts.get(did)?.balance ?? 0n;
  }

  async getStakedBalance(did: DID): Promise<Stake> {
    return this.accounts.get(did)?.stakedBalance ?? 0n;
  }

  async getNonce(did: DID): Promise<bigint> {
    return this.accounts.get(did)?.nonce ?? 0n;
  }

  async accountExists(did: DID): Promise<boolean> {
    return this.accounts.has(did);
  }

  async getReputation(did: DID): Promise<number> {
    return this.accounts.get(did)?.reputation ?? 0.5;
  }

  async createAccount(did: DID, initialBalance: Amount): Promise<void> {
    this.accounts.set(did, {
      did,
      balance: initialBalance,
      stakedBalance: 0n,
      nonce: 0n,
      reputation: 0.5,
    });
  }

  async transfer(from: DID, to: DID, amount: Amount): Promise<void> {
    const fromAccount = this.accounts.get(from);
    if (!fromAccount || fromAccount.balance < amount) {
      throw new Error("Insufficient balance");
    }

    const toAccount = this.accounts.get(to);
    if (!toAccount) {
      throw new Error("Recipient account not found");
    }

    fromAccount.balance -= amount;
    toAccount.balance += amount;
  }

  async addStake(did: DID, amount: Stake): Promise<void> {
    const account = this.accounts.get(did);
    if (!account || account.balance < amount) {
      throw new Error("Insufficient balance for staking");
    }

    account.balance -= amount;
    account.stakedBalance += amount;
  }

  async removeStake(did: DID, amount: Stake): Promise<void> {
    const account = this.accounts.get(did);
    if (!account || account.stakedBalance < amount) {
      throw new Error("Insufficient staked balance");
    }

    account.stakedBalance -= amount;
    account.balance += amount;
  }

  async incrementNonce(did: DID): Promise<bigint> {
    const account = this.accounts.get(did);
    if (!account) {
      throw new Error("Account not found");
    }
    account.nonce += 1n;
    return account.nonce;
  }

  async updateAccountReputation(did: DID, delta: number): Promise<number> {
    const account = this.accounts.get(did);
    if (!account) {
      throw new Error("Account not found");
    }
    account.reputation = Math.max(0, Math.min(1, account.reputation + delta));
    return account.reputation;
  }

  // === Validator Operations ===

  async getValidator(publicKey: PublicKey): Promise<ValidatorInfo | null> {
    return this.validators.get(publicKey) ?? null;
  }

  async getValidatorByDID(did: DID): Promise<ValidatorInfo | null> {
    for (const validator of this.validators.values()) {
      if (validator.did === did) {
        return validator;
      }
    }
    return null;
  }

  async getActiveValidators(): Promise<ValidatorInfo[]> {
    return Array.from(this.validators.values()).filter((v) => v.isActive);
  }

  async getTotalStake(): Promise<Stake> {
    let total = 0n;
    for (const validator of this.validators.values()) {
      if (validator.isActive) {
        total += validator.totalStake;
      }
    }
    return total;
  }

  async isActiveValidator(publicKey: PublicKey): Promise<boolean> {
    const validator = this.validators.get(publicKey);
    return validator?.isActive ?? false;
  }

  async getValidatorCount(): Promise<number> {
    return Array.from(this.validators.values()).filter((v) => v.isActive)
      .length;
  }

  async registerValidator(
    publicKey: PublicKey,
    did: DID,
    stake: Stake,
  ): Promise<void> {
    this.validators.set(publicKey, {
      publicKey,
      did,
      stake,
      delegatedStake: 0n,
      totalStake: stake,
      reputation: 0.5,
      isActive: true,
      activeSinceBlock: this.currentBlock,
      lastProposedBlock: 0n,
      blocksProposed: 0,
      verificationsCompleted: 0,
      verificationsCorrect: 0,
      slashCount: 0,
      commissionRate: 0.1,
    });
  }

  async updateStake(publicKey: PublicKey, delta: Stake): Promise<void> {
    const validator = this.validators.get(publicKey);
    if (!validator) {
      throw new Error("Validator not found");
    }
    validator.stake += delta;
    validator.totalStake = validator.stake + validator.delegatedStake;
  }

  async slash(
    publicKey: PublicKey,
    amount: Stake,
    _reason: string,
  ): Promise<void> {
    const validator = this.validators.get(publicKey);
    if (!validator) {
      throw new Error("Validator not found");
    }
    validator.stake -= amount;
    validator.totalStake = validator.stake + validator.delegatedStake;
    validator.slashCount += 1;
  }

  async updateValidatorReputation(
    publicKey: PublicKey,
    delta: number,
  ): Promise<void> {
    const validator = this.validators.get(publicKey);
    if (!validator) {
      throw new Error("Validator not found");
    }
    validator.reputation = Math.max(
      0,
      Math.min(1, validator.reputation + delta),
    );
  }

  async recordVerification(
    publicKey: PublicKey,
    correct: boolean,
  ): Promise<void> {
    const validator = this.validators.get(publicKey);
    if (!validator) {
      throw new Error("Validator not found");
    }
    validator.verificationsCompleted += 1;
    if (correct) {
      validator.verificationsCorrect += 1;
    }
  }

  async deactivate(publicKey: PublicKey): Promise<void> {
    const validator = this.validators.get(publicKey);
    if (!validator) {
      throw new Error("Validator not found");
    }
    validator.isActive = false;
  }

  // === Knowledge Operations ===

  async getFact(factId: string): Promise<FactInfo | null> {
    return this.facts.get(factId) ?? null;
  }

  async getFactsByDomain(domain: string, limit = 100): Promise<FactInfo[]> {
    const results: FactInfo[] = [];
    for (const fact of this.facts.values()) {
      if (fact.domain === domain) {
        results.push(fact);
        if (results.length >= limit) break;
      }
    }
    return results;
  }

  async getConfidence(factId: string): Promise<Confidence | null> {
    return this.facts.get(factId)?.confidence ?? null;
  }

  async factExists(factId: string): Promise<boolean> {
    return this.facts.has(factId);
  }

  async getDerivedFacts(_factId: string): Promise<string[]> {
    // Simplified - return empty array
    return [];
  }

  async addFact(fact: Omit<FactInfo, "verifiedAt">): Promise<string> {
    const factWithTime: FactInfo = {
      ...fact,
      verifiedAt: this.currentBlock,
    };
    this.facts.set(fact.id, factWithTime);
    return fact.id;
  }

  async updateConfidence(
    factId: string,
    confidence: Confidence,
  ): Promise<void> {
    const fact = this.facts.get(factId);
    if (!fact) {
      throw new Error("Fact not found");
    }
    fact.confidence = confidence;
  }

  async markChallenged(factId: string, _challengeId: string): Promise<void> {
    const fact = this.facts.get(factId);
    if (!fact) {
      throw new Error("Fact not found");
    }
    fact.status = "challenged";
    fact.challengeCount += 1;
  }

  async revokeFact(factId: string, _reason: string): Promise<void> {
    const fact = this.facts.get(factId);
    if (!fact) {
      throw new Error("Fact not found");
    }
    fact.status = "revoked";
  }

  async supersedeFact(oldFactId: string, _newFactId: string): Promise<void> {
    const fact = this.facts.get(oldFactId);
    if (!fact) {
      throw new Error("Fact not found");
    }
    fact.status = "superseded";
  }

  // === Block & Transaction ===

  async getCurrentBlock(): Promise<BlockHeight> {
    return this.currentBlock;
  }

  async getStateRoot(_block?: BlockHeight): Promise<MerkleRoot> {
    return "0".repeat(64) as MerkleRoot;
  }

  async beginTransaction(): Promise<TransactionHandle> {
    return {
      id: `tx-${++this.txCounter}`,
      startedAt: Date.now(),
    };
  }

  async commitTransaction(_handle: TransactionHandle): Promise<void> {
    // No-op for mock
  }

  async rollbackTransaction(_handle: TransactionHandle): Promise<void> {
    // No-op for mock
  }

  // === Test Helpers ===

  advanceBlock(count = 1n): void {
    this.currentBlock += count;
  }

  reset(): void {
    this.accounts.clear();
    this.validators.clear();
    this.facts.clear();
    this.currentBlock = 0n;
    this.txCounter = 0;
  }
}

interface MockAccount {
  did: DID;
  balance: Amount;
  stakedBalance: Stake;
  nonce: bigint;
  reputation: number;
}

// ============ Mock Event Bus ============

/**
 * In-memory mock event bus
 */
export class MockEventBus implements EventBus {
  private handlers = new Map<
    ProtocolEventType,
    Set<EventHandler<ProtocolEventType>>
  >();
  private history: ProtocolEvent[] = [];
  private subscriptionCounter = 0;

  emit<T extends ProtocolEventType>(event: ProtocolEvent<T>): void {
    this.history.push(event as ProtocolEvent);

    const typeHandlers = this.handlers.get(event.type);
    if (typeHandlers) {
      for (const handler of typeHandlers) {
        try {
          handler(event as ProtocolEvent<ProtocolEventType>);
        } catch (error) {
          console.error(`Event handler error for ${event.type}:`, error);
        }
      }
    }
  }

  on<T extends ProtocolEventType>(
    eventType: T,
    handler: EventHandler<T>,
  ): Subscription {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, new Set());
    }
    this.handlers
      .get(eventType)!
      .add(handler as EventHandler<ProtocolEventType>);

    const subscriptionId = `sub-${++this.subscriptionCounter}`;

    return {
      id: subscriptionId,
      eventType,
      unsubscribe: () => {
        this.handlers
          .get(eventType)
          ?.delete(handler as EventHandler<ProtocolEventType>);
      },
    };
  }

  once<T extends ProtocolEventType>(
    eventType: T,
    handler: EventHandler<T>,
  ): Subscription {
    const wrappedHandler: EventHandler<T> = (event: ProtocolEvent<T>) => {
      subscription.unsubscribe();
      handler(event);
    };
    const subscription = this.on(eventType, wrappedHandler);
    return subscription;
  }

  onMany(
    eventTypes: ProtocolEventType[],
    handler: EventHandler<ProtocolEventType>,
  ): Subscription[] {
    return eventTypes.map((type) => this.on(type, handler));
  }

  off(eventType: ProtocolEventType): void {
    this.handlers.delete(eventType);
  }

  async waitFor<T extends ProtocolEventType>(
    eventType: T,
    predicate?: (event: ProtocolEvent<T>) => boolean,
    timeoutMs = 5000,
  ): Promise<ProtocolEvent<T>> {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        subscription.unsubscribe();
        reject(new Error(`Timeout waiting for event: ${eventType}`));
      }, timeoutMs);

      const subscription = this.on(eventType, (event: ProtocolEvent<T>) => {
        if (!predicate || predicate(event as ProtocolEvent<T>)) {
          clearTimeout(timeout);
          subscription.unsubscribe();
          resolve(event as ProtocolEvent<T>);
        }
      });
    });
  }

  getRecentEvents(count = 100): ProtocolEvent[] {
    return this.history.slice(-count);
  }

  clearHistory(): void {
    this.history = [];
  }

  // === Test Helpers ===

  getEmittedEvents<T extends ProtocolEventType>(type?: T): ProtocolEvent<T>[] {
    if (type) {
      return this.history.filter((e) => e.type === type) as ProtocolEvent<T>[];
    }
    return this.history as ProtocolEvent<T>[];
  }

  reset(): void {
    this.handlers.clear();
    this.history = [];
    this.subscriptionCounter = 0;
  }
}

// ============ Mock Factory ============

/**
 * Create a test environment with mocks
 */
export interface MockEnvironment {
  state: MockStateProvider;
  events: MockEventBus;
  reset(): void;
}

/**
 * Create a fresh mock environment for testing
 */
export function createMockEnvironment(): MockEnvironment {
  const state = new MockStateProvider();
  const events = new MockEventBus();

  return {
    state,
    events,
    reset() {
      state.reset();
      events.reset();
    },
  };
}
