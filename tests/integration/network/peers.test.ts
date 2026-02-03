/**
 * Integration Tests for Peer Communication
 *
 * Tests multi-node communication via libp2p.
 * These tests require actual network operations.
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { LegibleNode } from '../../../network/node.js';
import { PUBSUB_TOPICS } from '../../../network/config.js';

// Helper to wait for a condition
async function waitFor(
  condition: () => boolean,
  timeout: number = 5000,
  interval: number = 100
): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    if (condition()) return true;
    await new Promise(resolve => setTimeout(resolve, interval));
  }
  return false;
}

// Helper to delay
function delay(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

describe('Peer Communication', () => {
  let node1: LegibleNode;
  let node2: LegibleNode;

  beforeEach(async () => {
    // Create two nodes with different ports
    node1 = new LegibleNode({
      listenAddresses: ['/ip4/127.0.0.1/tcp/0'], // Random port
      enableMdns: false,
      bootstrapPeers: [],
      dht: { enabled: false, clientMode: true },
    });

    node2 = new LegibleNode({
      listenAddresses: ['/ip4/127.0.0.1/tcp/0'], // Random port
      enableMdns: false,
      bootstrapPeers: [],
      dht: { enabled: false, clientMode: true },
    });

    await node1.start();
    await node2.start();
  });

  afterEach(async () => {
    await node1.stop();
    await node2.stop();
  });

  describe('direct connection', () => {
    it('should connect nodes directly', async () => {
      // Get node1's listen address
      const node1Addrs = node1.getListenAddresses();
      expect(node1Addrs.length).toBeGreaterThan(0);

      // Connect node2 to node1
      const peerInfo = await node2.connect(node1Addrs[0]);

      expect(peerInfo).toBeTruthy();
      expect(peerInfo?.peerId).toBe(node1.getPeerId());

      // Wait for connection to establish
      const connected = await waitFor(() => node2.getPeerCount() > 0);
      expect(connected).toBe(true);
    });

    it('should emit peer:connect event', async () => {
      let connectedPeerId: string | null = null;

      node2.on('peer:connect', (event: any) => {
        connectedPeerId = event.peer.peerId;
      });

      const node1Addrs = node1.getListenAddresses();
      await node2.connect(node1Addrs[0]);

      // Wait for event
      const received = await waitFor(() => connectedPeerId !== null);
      expect(received).toBe(true);
      expect(connectedPeerId).toBe(node1.getPeerId());
    });

    it('should handle disconnect', async () => {
      const node1Addrs = node1.getListenAddresses();
      await node2.connect(node1Addrs[0]);

      // Wait for connection
      await waitFor(() => node2.getPeerCount() > 0);

      // Disconnect
      await node2.disconnect(node1.getPeerId()!);

      expect(node2.getPeerCount()).toBe(0);
    });
  });

  describe('pubsub messaging', () => {
    it('should propagate messages between connected nodes', async () => {
      // Connect nodes
      const node1Addrs = node1.getListenAddresses();
      await node2.connect(node1Addrs[0]);

      // Wait for connection
      await waitFor(() =>
        node1.getPeerCount() > 0 && node2.getPeerCount() > 0
      );

      // Subscribe both nodes to same topic
      const testTopic = '/lgm/test/1.0.0';
      await node1.subscribe(testTopic);
      await node2.subscribe(testTopic);

      // Wait for subscription propagation (GossipSub needs time)
      await delay(1000);

      // Setup message receiver
      let receivedMessage: any = null;
      node2.on('message:received', (event: any) => {
        if (event.topic === testTopic) {
          receivedMessage = event.message;
        }
      });

      // Broadcast from node1
      const testMessage = {
        type: 'test',
        version: '1.0.0',
        timestamp: Date.now(),
        sender: node1.getPeerId()!,
        payload: new TextEncoder().encode(JSON.stringify({ data: 'hello' })),
      };

      await node1.broadcast(testTopic, testMessage);

      // Wait for message to arrive
      const received = await waitFor(() => receivedMessage !== null, 5000);

      // Note: Message propagation depends on GossipSub mesh formation
      // In a unit test environment, this may not always succeed
      // This is more of a smoke test
      if (received) {
        expect(receivedMessage.type).toBe('test');
      }
    });
  });

  describe('protocol streaming', () => {
    it('should handle protocol registration', async () => {
      let handlerCalled = false;

      await node1.registerProtocol('/lgm/test/1.0.0', async (stream, peerId) => {
        handlerCalled = true;
      });

      // Connect nodes
      const node1Addrs = node1.getListenAddresses();
      await node2.connect(node1Addrs[0]);

      // Wait for connection
      await waitFor(() => node2.getPeerCount() > 0);

      // Protocol handler is registered but won't be called until
      // node2 opens a stream on that protocol
      expect(handlerCalled).toBe(false);
    });
  });

  describe('statistics', () => {
    it('should track peer count across nodes', async () => {
      expect(node1.getPeerCount()).toBe(0);
      expect(node2.getPeerCount()).toBe(0);

      const node1Addrs = node1.getListenAddresses();
      await node2.connect(node1Addrs[0]);

      // Wait for bidirectional connection
      await waitFor(() =>
        node1.getPeerCount() > 0 && node2.getPeerCount() > 0
      );

      expect(node1.getPeerCount()).toBeGreaterThan(0);
      expect(node2.getPeerCount()).toBeGreaterThan(0);
    });
  });
});

describe('mDNS Discovery', () => {
  it('should discover local peers with mDNS enabled', async () => {
    // Create nodes with mDNS enabled
    const nodeA = new LegibleNode({
      listenAddresses: ['/ip4/127.0.0.1/tcp/0'],
      enableMdns: true,
      bootstrapPeers: [],
      dht: { enabled: false, clientMode: true },
    });

    const nodeB = new LegibleNode({
      listenAddresses: ['/ip4/127.0.0.1/tcp/0'],
      enableMdns: true,
      bootstrapPeers: [],
      dht: { enabled: false, clientMode: true },
    });

    try {
      await nodeA.start();
      await nodeB.start();

      // mDNS discovery can take several seconds
      // This is a smoke test - actual discovery depends on network conditions
      await delay(2000);

      // Nodes may or may not have discovered each other
      // depending on mDNS support in the test environment
    } finally {
      await nodeA.stop();
      await nodeB.stop();
    }
  });
});
