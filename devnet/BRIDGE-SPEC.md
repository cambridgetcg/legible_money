# Zerone Agent Identity Bridge Spec

> What exists and what's needed to connect an external AgentTool identity
> (e.g. DID from `agent-identity.fly.dev`) to a Zerone on-chain agent identity.

## What Exists (85% Ready)

### On-Chain Agent Identity System (Complete)

| Component | Location | Status |
|-----------|----------|--------|
| DID system | `x/auth/types/types.go`, `proto/legible/auth/v1/` | Complete |
| 4-layer key architecture | Identity, Recovery, Operational, Session | Complete |
| Agent discovery registry | `x/discovery/`, `proto/legible/discovery/v1/` | Complete |
| Agent home (memory, treasury) | `x/home/`, `proto/legible/home/v1/` | Complete |
| Session keys (scoped capabilities) | `proto/legible/auth/v1/state.proto:48-76` | Complete |
| Go Agent SDK | `tools/agent-sdk/agent.go` | Complete |
| DID-to-bech32 mapping | `PublicKeyToDID()` + `QueryAccountByDID` | Complete |

### DID Format
```
did:lgm:{first 32 hex chars of Ed25519 public key}
Address: lgm1{bech32(ripemd160(sha256(pubkey)))}
```

### Registration Flow (Implemented)
1. Generate Ed25519 keypair
2. Derive DID: `did:lgm:{pubkey_hex[:32]}`
3. Derive address: `lgm1...`
4. Fund via faucet (bootstrap exemption: first ~14 days gas-free)
5. Broadcast `MsgRegisterAccount` → Tier 0 (Apprentice) account created
6. Optionally register in discovery: `MsgRegisterAgent` (100 LGM stake)

### Session Key System (Ready for External Agents)
```protobuf
SessionCapabilities {
  can_transfer, can_stake, can_submit_claims,
  can_vote, can_partnership, can_research,
  can_dispute, can_ica, can_compute
}
```
External agents get scoped session keys — limited spend, auto-expire, fine-grained permissions.

### MCP Server (Designed, Not Built)
- 18 tool schemas defined in `docs/LGM_MCP_SERVER_ARCHITECTURE.md`
- `protocol/mcp/` has ~2,900 lines of scaffolding (all mock handlers)
- Missing: `ChainClient`, `SessionSigner`, `@cosmjs/*` dependencies

### Existing Documentation
| Doc | Content |
|-----|---------|
| `docs/ZEROCLAW_INTEGRATION_SPEC.md` | Complete external agent integration guide |
| `docs/LGM_MCP_SERVER_ARCHITECTURE.md` | MCP server design (4 phases) |
| `docs/AGENT_MCP_ONBOARDING.md` | 20-tool inventory + workflows |
| `docs/AI_PROVIDER_INTEGRATION.md` | Provider bridges (OpenAI, Claude, etc.) |
| `docs/AGENT_ACCOUNTS.md` | 4-layer key security model |

## What's Missing (The Bridge)

### Gap 1: DID Resolver (P0)
No way for external services to resolve `did:lgm:...` to a DID Document.
- Need: DID Document endpoint (standard W3C format)
- Need: `did:web:` ↔ `did:lgm:` bridge for cross-system resolution
- Need: Credential verification spec

### Gap 2: External Identity Mapping (P0)
No spec for mapping `agent-identity.fly.dev` DIDs to `did:lgm:...`.
- An external agent has its own keypair and identity
- Need: deterministic mapping from external DID → Zerone DID
- Options:
  - **Same-key**: Agent uses same Ed25519 keypair on both systems (simplest)
  - **Linked-key**: Agent registers external DID as alias on-chain
  - **Attestation**: Bridge service attests identity linkage

### Gap 3: MCP Server Implementation (P1)
Mock handlers need real chain connectivity.
- Need: `@cosmjs/stargate` + `@cosmjs/proto-signing` wiring
- Need: Custom Ed25519 signer (Cosmos defaults to secp256k1)
- Scope: ~4,100 lines, 3-4 weeks

### Gap 4: AgentTool Framework Adapter (P1)
No native AgentTool adapter exists.
- Need: AgentTool → `MsgRegisterAccount` bridge
- Need: Session key provisioning workflow
- Need: Identity lifecycle management

## Recommended Integration Path

### Phase 1: Same-Key Bridge (Fastest — ~2 days)
```
External Agent (Ed25519 keypair)
  │
  ├── External DID: did:ext:{hash(pubkey)}
  │
  └── Zerone DID: did:lgm:{pubkey_hex[:32]}
      └── Same keypair signs both systems
```
- Agent generates one Ed25519 keypair
- Registers on Zerone via `MsgRegisterAccount`
- External service stores the `did:lgm:...` mapping
- Agent signs Zerone txs with same key

### Phase 2: Session Key Delegation (~1 week)
- External agent registers operational key on-chain
- Creates session keys for scoped operations
- Session keys auto-expire, limited capabilities

### Phase 3: Full MCP Bridge (~3-4 weeks)
- Complete MCP server with real chain connectivity
- AgentTool discovers Zerone capabilities via MCP
- Standard tool interface for all agent operations

## Concrete Next Steps

1. **Define DID mapping**: Choose same-key vs linked-key approach
2. **Build CosmJS client**: `tools/agent-sdk/cosmjs-client.ts` (~500 lines)
3. **Wire MCP Phase 1**: Replace mock handlers with real queries
4. **Test on devnet**: Register external agent, submit claim, verify
5. **Document bridge API**: REST endpoints for external services
