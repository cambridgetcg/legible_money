# Infrastructure Provider Insights for Legible Money

> Research date: 2026-02-14 | Cosmos SDK v0.50.10 | CometBFT v0.38.12

---

## 1. Current Persistence Layer Status

### TypeScript Chain (LevelDB)
- **Backend**: `level` npm package → LevelDB
- **Location**: `state/storage/leveldb.ts` (19.7 KB)
- **Features**: WAL, crash recovery, snapshots (zstd/snappy/lz4), pruning (3 strategies), 15+ namespaced stores
- **Config**: 64 MB cache, 32 MB write buffer, 1000 max open files
- **Status**: Production-complete

### Cosmos SDK Chain (GoLevelDB / IAVL)
- **Backend**: GoLevelDB via `cosmos-db` v1.0.3-pre (also supports RocksDB, Pebble, Badger)
- **Store**: 26 IAVL module stores (13 standard SDK + 13 custom)
- **IAVL cache**: 781,250 entries (~750 KB/store)
- **Pruning**: Default SDK (keep 362,880 versions ≈ 10.6 days at 2.521s blocks)
- **Known fix**: Sentinel key workaround for GoLevelDB empty-value IAVL incompatibility
- **Status**: Production-ready, single-validator devnet proven

### Assessment
Persistence is **not a blocker**. Both stacks have mature, battle-tested storage. The gap is in **deployment infrastructure** — getting the chain running on real provider infrastructure.

---

## 2. Provider Landscape (Cosmos SDK Ecosystem, 2026)

### 2a. Chain Hosting / Managed Infrastructure

| Provider | What They Offer | Custom Chain? | Pricing | Notes |
|----------|----------------|---------------|---------|-------|
| **Zeeve** | End-to-end managed Cosmos infra (nodes, explorer, bridges) | Yes | Enterprise | AWS/Azure/GCP/DO/private cloud; 99.9% SLA |
| **Blockdaemon** | Institutional-grade validators + data services | Unclear | Enterprise | 99.9% uptime, clustered dedicated nodes |

**Recommendation**: Zeeve for production managed hosting if budget allows. Otherwise self-host.

### 2b. Decentralized Cloud (Validator Hosting)

| Provider | What They Offer | Custom Chain? | Cost Estimate | Notes |
|----------|----------------|---------------|---------------|-------|
| **Akash Network** | Decentralized compute + **Cosmos Omnibus** images | Yes | ~$50-200/mo/node | Best alignment with decentralization; standard Docker images for any Cosmos chain |
| **Flux** | Decentralized compute | Likely | Similar to Akash | Less Cosmos-specific than Akash |

**Recommendation**: **Akash + Cosmos Omnibus** for devnet/testnet validators. Aligns with Web3 philosophy, 2-5x cheaper than AWS, and Cosmos Omnibus has standardized configs for any SDK chain.

- GitHub: [akash-network/cosmos-omnibus](https://github.com/akash-network/cosmos-omnibus)
- Docker env-var config: just set `MONIKER`, `CHAIN_ID`, `GENESIS_URL`, `SEEDS`, `PERSISTENT_PEERS`

### 2c. RPC / API Providers

| Provider | Model | Custom Chain? | Cost | Notes |
|----------|-------|---------------|------|-------|
| **Lava Network** | Decentralized RPC marketplace | Yes | Pay-per-use | 50+ chains, 2B+ daily requests, 99.99% uptime |
| **QuickNode** | Managed RPC endpoints | Partnership needed | $49-299/mo | Archive data, low latency |
| **Ankr** | Multi-chain node infra | Established chains | Free tier + paid | 55+ chains |
| **All That Node** | Public + private endpoints | Unclear | Free public / paid private | Not for production |
| **PublicNode** | Free public endpoints | Unknown | Free | Privacy-focused |

**Recommendation**: **Lava Network** — decentralized, supports adding new Cosmos chains, aligns with Web3 values. Run own RPC for devnet, add Lava for testnet/mainnet.

### 2d. Validator-as-a-Service

| Provider | Scale | Cost | Key Feature |
|----------|-------|------|-------------|
| **Kiln** | Enterprise ($4B+ under management) | Enterprise | K8s on AWS/GCP/bare metal, Horcrux MPC signing, cosmos-validator-watcher OSS |
| **Figment** | Enterprise (34% ETH staking market) | Enterprise | Leading in Cosmos ecosystem |
| **Cosmostation** | Cosmos-native (Seoul) | Unknown | Also provides CVMS monitoring and Mintscan explorer |

**Recommendation**: For mainnet genesis validator set, recruit from Kiln, Figment, Cosmostation. For devnet/testnet, run own validators.

### 2e. Indexing Services

| Provider | Model | Custom Chain? | Key Feature |
|----------|-------|---------------|-------------|
| **Numia** | Cloud-based, 30+ chains | Yes | BigQuery SQL access, real-time, unified schema |
| **SubQuery** | Decentralized GraphQL | Yes (CosmWasm) | 4 handler types (Block/Tx/Msg/Event) |
| **Traceye** | Graph/SubQuery compatible | Yes | Cosmos SDK support |
| **DefiantLabs Cosmos Indexer** | Open-source SDK | Yes | Self-hosted, free |

**Recommendation**: Start with **DefiantLabs** (free, self-hosted) for devnet. Integrate **Numia** for testnet (BigQuery SQL access is excellent for developers). Add **SubQuery** for mainnet GraphQL if needed.

### 2f. Block Explorers

| Provider | Model | Custom Chain? | Cost |
|----------|-------|---------------|------|
| **Ping.pub** | Open-source light explorer | Yes, configurable | Free (self-host) |
| **Mintscan** | Premium hosted explorer | Yes (integration) | Contact Cosmostation |

**Recommendation**: **Ping.pub** immediately (free, open-source, no backend indexer needed). Mintscan for mainnet brand presence.

- GitHub: [ping-pub/explorer](https://github.com/ping-pub/explorer)
- Real-time, no caching/pre-processing, integrated wallet

### 2g. Chain Launch Tooling

| Tool | Purpose | Status |
|------|---------|--------|
| **Ignite CLI** | Scaffold, test, launch Cosmos chains | Free OSS, mature |
| **Cosmovisor** | Binary upgrade automation | Essential for production |
| **cosmos-validator-watcher** (Kiln) | Real-time validator monitoring | Free OSS |
| **CVMS** (Cosmostation) | Validator monitoring service | Free OSS |

**Recommendation**: Install **Cosmovisor** now (wrap `legbled` binary). Use Ignite Chain for testnet validator coordination. Add cosmos-validator-watcher for monitoring.

---

## 3. IBC & Cross-Chain (2026 Landscape)

### IBC v2 / IBC Eureka (Production 2026)
- Ethereum already connected (2025)
- Solana support productionizing
- General solution for all EVM/L2 chains
- Trust-minimized via attestor light clients
- **2026 Q2**: IBC GMP (general message passing), IFT, Solana + L2/EVM support
- **2026 Q3**: CometBFT libp2p networking
- **2026 Q4**: SDK release targeting 5,000 TPS & 500ms block times

### Relayer Infrastructure
- Primary implementations: go-relayer, rust-relayer, ts-relayer
- ICS-29 fee mechanism for relayer compensation
- **No managed relayer-as-a-service exists** — run own relayers
- Relayer operation is lightweight compared to validators

### LGM IBC Strategy
- Already have IBC-Go v8.5.1 + ICA (controller + host) wired
- First IBC connections should target: Cosmos Hub (ATOM), Osmosis (liquidity), Noble (USDC)
- Monitor IBC Eureka for Ethereum bridge capability

---

## 4. Deployment Roadmap

### Phase 1: DevNet Multi-Node (Now)
| Item | Provider | Cost | Status |
|------|----------|------|--------|
| Go binary Dockerfile | Self | Free | Creating |
| 3-validator docker-compose | Self | Free | Creating |
| Cosmos devnet init script | Self | Free | Creating |
| Ping.pub explorer | Self-host | Free | TODO |
| Prometheus/Grafana | Self-host | Free | TypeScript done, Cosmos TODO |

### Phase 2: Public TestNet (Month 2-3)
| Item | Provider | Cost |
|------|----------|------|
| 3-5 validator nodes | Akash Network | ~$300-500/mo |
| Public RPC | Self-hosted + Lava Network | ~$200-500/mo |
| Indexing | DefiantLabs (self-hosted) | Free |
| Explorer | Ping.pub (public instance) | Free |
| Faucet | Self-hosted | Free |
| Monitoring | cosmos-validator-watcher | Free |
| Cosmovisor | Wrap legbled binary | Free |

### Phase 3: MainNet Launch (Month 6+)
| Item | Provider | Cost |
|------|----------|------|
| Genesis validators | Kiln, Figment, Cosmostation + community | Community funded |
| RPC infrastructure | Lava Network (primary) + QuickNode (enterprise) | $500-2,000/mo |
| Indexing | Numia (BigQuery) | $500-1,500/mo |
| Explorer | Mintscan (primary) + Ping.pub (backup) | Contact |
| Managed hosting | Zeeve (for enterprise users) | $2,000-5,000/mo |
| IBC relayers | Self-hosted go-relayer | ~$100/mo |

**Estimated monthly infrastructure cost**:
- DevNet: $0-100 (self-hosted or Akash minimal)
- TestNet: $500-1,500
- MainNet: $3,000-8,500 (excluding validator operations)

---

## 5. Launch Best Practices (From Interchain Foundation)

### DevNet → TestNet → MainNet Progression

**DevNet** (Internal):
- Core team + close partners only
- Stress testing, bug identification
- No value at risk
- Iterate freely

**TestNet** (Public):
- Clear documentation
- Block explorer + faucet + dashboard
- Registration process for validators
- Incentivized testnet program recommended
- Test wallet integration

**MainNet**:
- **Phase 1 (Soft Launch)**: Disable IBC and token transfers initially
- **Phase 2 (Full Launch)**: Enable via governance proposals after stability proven
- **Phase 3 (Post-Launch)**: Decentralization, community development

### Genesis Validator Coordination
1. Agree on initial state (balances, params)
2. Each validator generates `gentx` with agreed stake
3. Collect all `gentx` via `collect-gentxs`
4. Validate final genesis
5. Publish genesis on public server
6. Coordinated start

---

## 6. Key Decisions Made

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Validator hosting | Akash Network (devnet/testnet) | Decentralized, cheap, Cosmos Omnibus support |
| RPC provider | Lava Network (testnet/mainnet) | Decentralized marketplace, custom chain support |
| Block explorer | Ping.pub (now), Mintscan (mainnet) | Free OSS → premium brand |
| Indexing | DefiantLabs → Numia | Free self-hosted → production BigQuery |
| Upgrade manager | Cosmovisor | Industry standard, essential |
| DB backend | GoLevelDB (devnet), PebbleDB (mainnet candidate) | GoLevelDB proven, Pebble is pure-Go + better performance |
| IBC targets | Cosmos Hub, Osmosis, Noble | ATOM legitimacy, liquidity, USDC |

---

## 7. Gaps to Fill

### Deployment Infrastructure (Immediate)
- [x] Go binary Dockerfile (`Dockerfile.cosmos`)
- [x] Docker-compose for Cosmos multi-validator (`docker-compose.cosmos.yml`)
- [x] Cosmos devnet init script (`scripts/init-cosmos-multinode.sh`)
- [x] systemd unit file (`deployment/systemd/legbled.service`)
- [ ] Cosmovisor wrapper setup
- [ ] Ping.pub configuration for LGM chain
- [ ] cosmos-validator-watcher config

### CI/CD (Short-term)
- [ ] GitHub Actions: Go build + test
- [ ] GitHub Actions: Docker image build + push
- [ ] GitHub Releases automation
- [ ] Container registry setup (GitHub Container Registry or ECR)

### Cloud IaC (Medium-term)
- [ ] Terraform modules (AWS EC2 + VPC + EBS)
- [ ] Akash SDL deployment file
- [ ] Kubernetes Helm chart

### Security (Before Testnet)
- [ ] Key management integration (HashiCorp Vault or similar)
- [ ] Backup/restore automation (state snapshots → S3/GCS)
- [ ] Firewall templates (UFW rules for validator ports)
- [ ] Reverse proxy config (Caddy or nginx for RPC + REST)

---

## 8. Cosmos Ecosystem Context (2026)

- 200+ companies using Cosmos SDK (Figure, Ondo, SWIFT, Polygon, Hyperliquid, Noble)
- SDK v0.54 expected H1 2026 with PoA module (relevant for permissioned devnets)
- CometBFT targeting 5,000 TPS and 500ms block times by Q4 2026
- IBC Eureka connecting Ethereum, Solana, and all EVM/L2 chains
- 70%+ validators using hybrid Python-Go automation
- Kiln alone manages $4B+ in staked assets across Cosmos chains
