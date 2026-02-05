/**
 * Dependency Injection Container
 *
 * ARCH-3: Dependency Injection for Infrastructure
 * Domain modules MUST NOT directly instantiate infrastructure.
 * Infrastructure MUST be injected via interfaces.
 *
 * This container provides:
 * - Service registration and resolution
 * - Lifecycle management (singleton, transient, scoped)
 * - Factory function support
 * - Type-safe service keys
 */

// ============ Service Keys ============

/**
 * Symbol keys for type-safe service registration
 */
export const ServiceKeys = {
  // Infrastructure
  StateProvider: Symbol("StateProvider"),
  StateReader: Symbol("StateReader"),
  EventBus: Symbol("EventBus"),
  Storage: Symbol("Storage"),
  Network: Symbol("Network"),

  // Domain Services
  KnowledgeService: Symbol("KnowledgeService"),
  ConsensusEngine: Symbol("ConsensusEngine"),
  RewardsEngine: Symbol("RewardsEngine"),
  DisputeResolver: Symbol("DisputeResolver"),
  IdentityService: Symbol("IdentityService"),

  // Readers (read-only variants)
  KnowledgeReader: Symbol("KnowledgeReader"),
  ConsensusReader: Symbol("ConsensusReader"),
  RewardsReader: Symbol("RewardsReader"),
  DisputeReader: Symbol("DisputeReader"),
  IdentityReader: Symbol("IdentityReader"),

  // Configuration
  ConsensusParams: Symbol("ConsensusParams"),
  VestingParams: Symbol("VestingParams"),
  DisputeParams: Symbol("DisputeParams"),
} as const;

export type ServiceKey = (typeof ServiceKeys)[keyof typeof ServiceKeys];

// ============ Lifecycle Types ============

/**
 * Service lifecycle
 */
export type Lifecycle =
  | "singleton" // One instance for entire application
  | "transient" // New instance on each resolve
  | "scoped"; // One instance per scope (e.g., request)

/**
 * Service registration
 */
export interface ServiceRegistration<T> {
  key: ServiceKey;
  lifecycle: Lifecycle;
  factory: (container: Container) => T;
  instance?: T; // For singletons
}

/**
 * Scope for scoped services
 */
export interface Scope {
  id: string;
  instances: Map<ServiceKey, unknown>;
  parent?: Scope;
}

// ============ Container Interface ============

/**
 * Dependency injection container
 */
export interface Container {
  /**
   * Register a service with a factory
   */
  register<T>(
    key: ServiceKey,
    factory: (container: Container) => T,
    lifecycle?: Lifecycle,
  ): void;

  /**
   * Register a singleton instance directly
   */
  registerInstance<T>(key: ServiceKey, instance: T): void;

  /**
   * Resolve a service
   */
  resolve<T>(key: ServiceKey): T;

  /**
   * Try to resolve a service (returns undefined if not found)
   */
  tryResolve<T>(key: ServiceKey): T | undefined;

  /**
   * Check if a service is registered
   */
  has(key: ServiceKey): boolean;

  /**
   * Create a child scope
   */
  createScope(): ScopedContainer;

  /**
   * Dispose container and all services
   */
  dispose(): Promise<void>;
}

/**
 * Scoped container (for request-scoped services)
 */
export interface ScopedContainer extends Container {
  scopeId: string;
  disposeScope(): Promise<void>;
}

// ============ Container Implementation ============

/**
 * Create a new DI container
 */
export function createContainer(): Container {
  const registrations = new Map<ServiceKey, ServiceRegistration<unknown>>();
  let disposed = false;

  function assertNotDisposed(): void {
    if (disposed) {
      throw new Error("Container has been disposed");
    }
  }

  const container: Container = {
    register<T>(
      key: ServiceKey,
      factory: (container: Container) => T,
      lifecycle: Lifecycle = "singleton",
    ): void {
      assertNotDisposed();
      registrations.set(key, {
        key,
        lifecycle,
        factory: factory as (container: Container) => unknown,
      });
    },

    registerInstance<T>(key: ServiceKey, instance: T): void {
      assertNotDisposed();
      registrations.set(key, {
        key,
        lifecycle: "singleton",
        factory: () => instance,
        instance,
      });
    },

    resolve<T>(key: ServiceKey): T {
      assertNotDisposed();
      const registration = registrations.get(key);
      if (!registration) {
        throw new Error(`Service not registered: ${key.toString()}`);
      }

      if (registration.lifecycle === "singleton") {
        if (registration.instance === undefined) {
          registration.instance = registration.factory(container);
        }
        return registration.instance as T;
      }

      // Transient - always create new
      return registration.factory(container) as T;
    },

    tryResolve<T>(key: ServiceKey): T | undefined {
      assertNotDisposed();
      try {
        return container.resolve<T>(key);
      } catch {
        return undefined;
      }
    },

    has(key: ServiceKey): boolean {
      return registrations.has(key);
    },

    createScope(): ScopedContainer {
      assertNotDisposed();
      const scopeId = crypto.randomUUID();
      const scopedInstances = new Map<ServiceKey, unknown>();

      const scopedContainer: ScopedContainer = {
        scopeId,

        register: container.register.bind(container),
        registerInstance: container.registerInstance.bind(container),
        has: container.has.bind(container),
        createScope: container.createScope.bind(container),

        resolve<T>(key: ServiceKey): T {
          assertNotDisposed();
          const registration = registrations.get(key);
          if (!registration) {
            throw new Error(`Service not registered: ${key.toString()}`);
          }

          if (registration.lifecycle === "scoped") {
            if (!scopedInstances.has(key)) {
              scopedInstances.set(key, registration.factory(scopedContainer));
            }
            return scopedInstances.get(key) as T;
          }

          // Delegate to parent for singleton/transient
          return container.resolve<T>(key);
        },

        tryResolve<T>(key: ServiceKey): T | undefined {
          try {
            return scopedContainer.resolve<T>(key);
          } catch {
            return undefined;
          }
        },

        async disposeScope(): Promise<void> {
          for (const instance of scopedInstances.values()) {
            if (
              instance &&
              typeof (instance as Disposable).dispose === "function"
            ) {
              await (instance as Disposable).dispose();
            }
          }
          scopedInstances.clear();
        },

        async dispose(): Promise<void> {
          await scopedContainer.disposeScope();
        },
      };

      return scopedContainer;
    },

    async dispose(): Promise<void> {
      if (disposed) return;
      disposed = true;

      // Dispose all singleton instances
      for (const registration of registrations.values()) {
        if (
          registration.instance &&
          typeof (registration.instance as Disposable).dispose === "function"
        ) {
          await (registration.instance as Disposable).dispose();
        }
      }
      registrations.clear();
    },
  };

  return container;
}

/**
 * Disposable interface for cleanup
 */
interface Disposable {
  dispose(): void | Promise<void>;
}

// ============ Builder Pattern ============

/**
 * Container builder for fluent configuration
 */
export class ContainerBuilder {
  private registrations: Array<{
    key: ServiceKey;
    factory: (container: Container) => unknown;
    lifecycle: Lifecycle;
  }> = [];

  /**
   * Register a singleton service
   */
  addSingleton<T>(
    key: ServiceKey,
    factory: (container: Container) => T,
  ): ContainerBuilder {
    this.registrations.push({ key, factory, lifecycle: "singleton" });
    return this;
  }

  /**
   * Register a transient service
   */
  addTransient<T>(
    key: ServiceKey,
    factory: (container: Container) => T,
  ): ContainerBuilder {
    this.registrations.push({ key, factory, lifecycle: "transient" });
    return this;
  }

  /**
   * Register a scoped service
   */
  addScoped<T>(
    key: ServiceKey,
    factory: (container: Container) => T,
  ): ContainerBuilder {
    this.registrations.push({ key, factory, lifecycle: "scoped" });
    return this;
  }

  /**
   * Register an instance directly
   */
  addInstance<T>(key: ServiceKey, instance: T): ContainerBuilder {
    this.registrations.push({
      key,
      factory: () => instance,
      lifecycle: "singleton",
    });
    return this;
  }

  /**
   * Build the container
   */
  build(): Container {
    const container = createContainer();
    for (const { key, factory, lifecycle } of this.registrations) {
      container.register(key, factory, lifecycle);
    }
    return container;
  }
}

/**
 * Create a container builder
 */
export function containerBuilder(): ContainerBuilder {
  return new ContainerBuilder();
}
