# State Migration Tool: TypeScript Chain → Cosmos Genesis

Converts exported state from the TypeScript Legible Money chain into a valid
Cosmos SDK genesis file.

## Usage

```bash
# Phase 5 implementation. Currently a stub.
legbled migrate-genesis ./ts-state-export.json --chain-id lgm-1 --output genesis.json
```

## State Tree Mapping

| TypeScript State Tree       | Cosmos Module            | Genesis Section                    |
|-----------------------------|--------------------------|-------------------------------------|
| AccountStateTree            | x/auth + x/bank          | auth.accounts + bank.balances       |
| ValidatorStateTree          | x/staking                | staking.validators + delegations    |
| KnowledgeStateTree          | x/knowledge              | knowledge.facts + references        |
| ClaimStateTree              | x/knowledge              | knowledge.claims                    |
| EvidenceStateTree           | x/evidence_mgmt          | evidence_mgmt.evidence_items        |
| DisputeStateTree            | x/disputes               | disputes.disputes                   |
| VestingStateTree            | x/vesting_rewards         | vesting_rewards.schedules           |
| ResearchStateTree           | x/research               | research.researches + bounties      |
| GovernanceStateTree         | x/gov (extended)         | gov.proposals + votes               |
| PaymentChannelStateTree     | x/channels               | channels.channels                   |
| ProjectStateTree            | x/tree                   | tree.projects                       |
| ServiceStateTree            | x/tree                   | tree.services                       |
| TaskStateTree               | x/tree                   | tree.tasks                          |
| SeedStateTree               | x/tree                   | tree.seeds                          |
| ContractStateTree           | x/bvm                    | bvm.contracts                       |
| TokenRegistryStateTree      | x/tokens                 | tokens.tokens + pools               |
| ScheduleStateTree           | x/bvm                    | bvm.schedules                       |

## Address Conversion

DID format: `did:lgm:{first-32-hex-of-pubkey}`
Bech32 format: `lgm1{bech32(ripemd160(sha256(pubkey)))}`

The migration tool:
1. Exports all `(did, publicKey, accountState)` tuples from the TS chain
2. Derives bech32 addresses from stored public keys
3. Maintains a `did_to_bech32` mapping in genesis for backward compatibility
4. Both addressing schemes remain valid during the transition period

## Procedure

1. Halt TS chain at specific block height
2. Export full state via `StateManager.getSnapshot()`
3. Run `legbled migrate-genesis` to transform all 17 state trees
4. Validate genesis integrity
5. Distribute genesis file to validators
6. Launch Cosmos chain
