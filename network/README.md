# Network Module

P2P networking layer using libp2p.

## Overview

The network module provides:

- **libp2p integration** - Full P2P networking stack
- **GossipSub** - Message broadcasting (blocks, transactions, evidence)
- **Kademlia DHT** - Peer discovery
- **Custom protocols** - Block sync, state sync, light sync

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     LegibleNode                          │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │
│  │   PubSub    │  │    DHT      │  │    Protocols    │  │
│  │  (GossipSub)│  │  (Kademlia) │  │  (Sync, etc.)   │  │
│  └─────────────┘  └─────────────┘  └─────────────────┘  │
├─────────────────────────────────────────────────────────┤
│                      libp2p Core                         │
│  ┌───────┐  ┌───────┐  ┌───────┐  ┌───────────────┐    │
│  │  TCP  │  │  WS   │  │ Noise │  │    Yamux      │    │
│  └───────┘  └───────┘  └───────┘  └───────────────┘    │
└─────────────────────────────────────────────────────────┘
```

## LegibleNode

Main network node class.

```typescript
import { LegibleNode } from './network';
import { PUBSUB_TOPICS } from './network/config';

const node = new LegibleNode({
  listenAddresses: ['/ip4/0.0.0.0/tcp/9000'],
  bootstrapPeers: [
    '/ip4/bootnode.legible.money/tcp/9000/p2p/12D3Koo...',
  ],
  maxConnections: 50,
  minConnections: 5,
});

// Start node
await node.start();
console.log('Peer ID:', node.getPeerId());
console.log('Listening:', node.getListenAddresses());

// Connect to peer
await node.connect('/ip4/192.168.1.10/tcp/9000/p2p/12D3Koo...');

// Get connected peers
const peers = node.getConnectedPeers();
console.log('Connected:', peers.length);

// Stop node
await node.stop();
```

## PubSub Topics

Message broadcasting using GossipSub.

```typescript
// Subscribe to topic
await node.subscribe(PUBSUB_TOPICS.BLOCKS);

// Broadcast to topic
await node.broadcast(PUBSUB_TOPICS.TRANSACTIONS, message);

// Unsubscribe
await node.unsubscribe(PUBSUB_TOPICS.BLOCKS);

// Listen for messages via events
node.on('message:received', ({ topic, message, from }) => {
  console.log(`Message on ${topic} from ${from}`);
});
```

### Available Topics

| Topic | Description |
|-------|-------------|
| `/lgm/blocks/1.0.0` | New block announcements |
| `/lgm/transactions/1.0.0` | Transaction propagation |
| `/lgm/claims/1.0.0` | Claim submissions |
| `/lgm/verifications/1.0.0` | Verification results |
| `/lgm/consensus/1.0.0` | Consensus messages |
| `/lgm/validators/1.0.0` | Validator coordination |
| `/lgm/evidence/1.0.0` | Evidence submissions |
| `/lgm/evidence-requests/1.0.0` | Evidence requests with bounties |
| `/lgm/attestations/1.0.0` | Third-party attestations |
| `/lgm/challenges/1.0.0` | Challenge announcements |
| `/lgm/resolutions/1.0.0` | Final resolution outcomes |
| `/lgm/acknowledgments/1.0.0` | Delivery confirmations |

## Topic Manager

Route and validate messages using `TopicManager`:

```typescript
import { TopicManager, createBlockTopicHandler } from './network/pubsub/topics';

const topicManager = new TopicManager();

// Register handlers
topicManager.registerHandler(createBlockTopicHandler(
  async (block) => validateBlock(block),
  async (block) => processBlock(block)
));

// Wire to node
node.setTopicManager(topicManager);
```

## Consensus Bridge

Wire consensus engine to network broadcasts:

```typescript
import { createConsensusBridge } from './network/adapters/consensus-bridge';

const bridge = createConsensusBridge({
  node,
  consensus: potEngine,
  mempool,
});

// Broadcasts happen automatically via events:
// - consensus 'block_proposed' → bridge.broadcastBlock()
// - mempool 'tx:added' → bridge.broadcastTransaction()

// Or broadcast manually:
await bridge.broadcastBlock(block);
await bridge.broadcastCommitment(roundId, commitment);
```

## Sync Protocols

### Block Sync (`/lgm/sync/1.0.0`)

Full block synchronization between nodes.

```typescript
import { SyncManager, SyncProtocolHandler, SYNC_PROTOCOL } from './network/protocols/sync';

const syncManager = new SyncManager(
  currentBlock,
  getBlock,
  saveBlock,
  validateBlock
);

// Register protocol handler
const handler = new SyncProtocolHandler(syncManager);
await node.registerProtocol(SYNC_PROTOCOL, (stream, peerId) =>
  handler.handleStream(stream, peerId)
);
```

### State Sync (`/lgm/state-sync/1.0.0`)

Snapshot-based state sync for new validators.

```typescript
import { StateSyncManager, STATE_SYNC_PROTOCOL } from './network/protocols/state_sync';

const stateSyncManager = new StateSyncManager(snapshotManager);

// Check if state sync is needed
if (stateSyncManager.shouldUseStateSync(currentBlock, targetBlock)) {
  await stateSyncManager.startStateSync();
}
```

### Light Sync (`/lgm/light-sync/1.0.0`)

Lightweight sync for chatnode instances (headers + knowledge tree only).

```typescript
import { LightSyncManager } from './network/protocols/light_sync';

const lightSync = new LightSyncManager(
  currentBlock,
  saveHeader,
  getHeader,
  validateHeader
);

// Get knowledge proof on demand
const proof = await lightSync.getKnowledgeProof(factId);
```

## Peer Discovery

### Bootstrap Peers

```typescript
import { BootstrapDiscovery } from './network/discovery/bootstrap';

const discovery = new BootstrapDiscovery({
  peers: [{ address: '/ip4/.../tcp/9000/p2p/12D3Koo...', failureCount: 0 }],
  validatePeers: true, // VULN-018: validates multiaddress format
});

const connected = await discovery.discover(async (addr) => {
  const peer = await node.connect(addr);
  return peer !== null;
});
```

### mDNS (Local Discovery)

```typescript
const config = {
  enableMdns: true,  // Auto-discover peers on local network
};
```

### Kademlia DHT

```typescript
const config = {
  dht: {
    enabled: true,
    clientMode: false,  // Serve DHT queries
  },
};
```

## Configuration

See `NetworkConfig` in `config.ts` for full options. Key settings:

```typescript
const DEFAULT_NETWORK_CONFIG = {
  listenAddresses: ['/ip4/0.0.0.0/tcp/9000', '/ip4/0.0.0.0/tcp/9001/ws'],
  bootstrapPeers: [],
  enableMdns: true,
  maxConnections: 100,
  minConnections: 10,
  connectionTimeout: 30000,
  dialTimeout: 10000,
  pubsub: { enabled: true },
  dht: { enabled: true, clientMode: false },
};
```

## Security

### Noise Protocol

All connections encrypted with Noise protocol framework.

### Peer Scoring

GossipSub peer scoring prevents spam (configured via `PeerScoreConfig`):

```typescript
pubsub: {
  scoreThresholds: {
    gossipThreshold: -100,    // Below this: no gossip
    publishThreshold: -1000,  // Below this: no publish
    graylistThreshold: -10000,// Below this: disconnect
  },
}
```

### Message Validation (VULN-019)

All PubSub messages pass base validation before handler-specific validation:
- Payload size limit (1 MB)
- Message freshness (5 min max age)
- Sender verification
- Structure validation

### Multiaddress Validation (VULN-018)

Bootstrap peer addresses are validated before connection attempts.

## Testing

```bash
# Run network tests
npm run test -- tests/unit/network/
```

## Files

| File | Description |
|------|-------------|
| `config.ts` | Network configuration, protocol IDs, BigInt serialization |
| `node.ts` | LegibleNode class |
| `pubsub/topics.ts` | TopicManager, base message validation |
| `pubsub/truth_topics.ts` | Truth Message System topic handlers |
| `protocols/sync.ts` | Block sync protocol |
| `protocols/state_sync.ts` | Snapshot-based state sync |
| `protocols/light_sync.ts` | Light sync for chatnodes |
| `discovery/bootstrap.ts` | Bootstrap peer discovery |
| `adapters/consensus-bridge.ts` | Consensus-network bridge |

## See Also

- [RUNNING_A_NODE.md](../docs/RUNNING_A_NODE.md) - Node operation
- [GLOSSARY.md](../docs/GLOSSARY.md) - Terminology
