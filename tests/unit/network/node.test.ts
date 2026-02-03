/**
 * Unit Tests for LegibleNode
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { LegibleNode } from '../../../network/node.js';

describe('LegibleNode', () => {
  let node: LegibleNode;

  beforeEach(() => {
    // Create node with test configuration
    node = new LegibleNode({
      listenAddresses: ['/ip4/127.0.0.1/tcp/0'], // Random port
      enableMdns: false,
      bootstrapPeers: [],
      dht: { enabled: false, clientMode: true },
    });
  });

  afterEach(async () => {
    if (node.isStarted()) {
      await node.stop();
    }
  });

  describe('lifecycle', () => {
    it('should start and stop cleanly', async () => {
      expect(node.isStarted()).toBe(false);

      await node.start();

      expect(node.isStarted()).toBe(true);
      expect(node.getPeerId()).toBeTruthy();
      expect(node.getPeerId()?.startsWith('12D3')).toBe(true);

      await node.stop();

      expect(node.isStarted()).toBe(false);
    });

    it('should emit started event with peerId', async () => {
      const startedHandler = vi.fn();
      node.on('started', startedHandler);

      await node.start();

      expect(startedHandler).toHaveBeenCalledTimes(1);
      expect(startedHandler).toHaveBeenCalledWith(
        expect.objectContaining({
          peerId: expect.any(String)
        })
      );
    });

    it('should emit stopped event', async () => {
      const stoppedHandler = vi.fn();
      node.on('stopped', stoppedHandler);

      await node.start();
      await node.stop();

      expect(stoppedHandler).toHaveBeenCalledTimes(1);
    });

    it('should throw if started twice', async () => {
      await node.start();

      await expect(node.start()).rejects.toThrow('Node already started');
    });

    it('should do nothing if stopped when not running', async () => {
      await expect(node.stop()).resolves.not.toThrow();
    });
  });

  describe('peer management', () => {
    it('should start with no peers', async () => {
      await node.start();

      expect(node.getPeerCount()).toBe(0);
      expect(node.getConnectedPeers()).toHaveLength(0);
    });

    it('should return listen addresses after start', async () => {
      await node.start();

      const addrs = node.getListenAddresses();
      expect(addrs.length).toBeGreaterThan(0);
      expect(addrs[0]).toContain('/ip4/127.0.0.1/tcp/');
    });
  });

  describe('pubsub', () => {
    it('should subscribe to topics', async () => {
      await node.start();

      // Should not throw
      await expect(node.subscribe('/test/topic/1.0.0')).resolves.not.toThrow();
    });

    it('should unsubscribe from topics', async () => {
      await node.start();
      await node.subscribe('/test/topic/1.0.0');

      // Should not throw
      await expect(node.unsubscribe('/test/topic/1.0.0')).resolves.not.toThrow();
    });

    it('should throw if subscribe before start', async () => {
      await expect(node.subscribe('/test/topic')).rejects.toThrow('Node not initialized');
    });
  });

  describe('statistics', () => {
    it('should return initial stats', async () => {
      const stats = node.getStats();

      expect(stats.connectedPeers).toBe(0);
      expect(stats.messagesReceived).toBe(0);
      expect(stats.messagesSent).toBe(0);
      expect(stats.bytesReceived).toBe(0n);
      expect(stats.bytesSent).toBe(0n);
    });

    it('should have peerId after start', async () => {
      await node.start();
      const stats = node.getStats();

      expect(stats.peerId).toBeTruthy();
      expect(stats.uptime).toBeGreaterThan(0);
    });
  });

  describe('component integration', () => {
    it('should accept topic manager', async () => {
      const mockTopicManager = {
        getHandler: vi.fn().mockReturnValue(null),
      };

      node.setTopicManager(mockTopicManager as any);

      // Should not throw
      await node.start();
    });

    it('should accept sync manager', async () => {
      const mockSyncManager = {};

      node.setSyncManager(mockSyncManager as any);

      // Should not throw
      await node.start();
    });
  });

  describe('error handling', () => {
    it('should handle broadcast before start', async () => {
      await expect(node.broadcast('/topic', {} as any)).rejects.toThrow('Node not started');
    });

    it('should handle send before start', async () => {
      await expect(node.send('peer', '/proto', {} as any)).rejects.toThrow('Node not started');
    });

    it('should handle request before start', async () => {
      await expect(node.request('peer', '/proto', {} as any)).rejects.toThrow('Node not started');
    });

    it('should handle connect before start', async () => {
      await expect(node.connect('/ip4/1.2.3.4/tcp/9000/p2p/xyz')).rejects.toThrow('Node not initialized');
    });
  });
});

describe('LegibleNode factory functions', () => {
  it('should create devnet node', async () => {
    const { createDevnetNode } = await import('../../../network/node.js');
    const node = createDevnetNode(9999);

    expect(node).toBeInstanceOf(LegibleNode);
  });

  it('should create testnet node', async () => {
    const { createTestnetNode } = await import('../../../network/node.js');
    const node = createTestnetNode(['/ip4/1.2.3.4/tcp/9000/p2p/12D3KooWtest']);

    expect(node).toBeInstanceOf(LegibleNode);
  });
});
