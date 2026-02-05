# Phase 1: Foundation Extraction - COMPLETE

**Date**: 2026-02-03
**Status**: COMPLETE

## Summary

The foundation layer has been created as the base layer with no internal dependencies. All other layers will depend on this.

## Created Files

```
foundation/
├── index.ts              # Main entry point
├── types/
│   ├── primitives.ts     # Branded primitive types
│   ├── errors.ts         # Error hierarchy
│   └── index.ts          # Types public API
├── crypto/
│   └── index.ts          # Crypto facade (re-exports core/crypto)
└── utils/
    └── index.ts          # Pure utility functions
```

## Foundation Layer Contents

### Types (`foundation/types/`)

**Primitive Types** (branded for type safety):
- `PrivateKey`, `PublicKey`, `Signature` - Cryptographic
- `Hash`, `MerkleRoot`, `VRFOutput`, `VRFProof`, `Commitment` - Hashing
- `DID`, `Address` - Identity
- `BlockHeight`, `Timestamp`, `BlockDuration`, `Epoch` - Temporal
- `Amount`, `Fee`, `Stake`, `Confidence` - Economic

**Protocol Constants**:
- `PROTOCOL_CONSTANTS.TARGET_BLOCK_TIME_MS` = 2521
- `PROTOCOL_CONSTANTS.FINALITY_DEPTH` = 19
- `ONE_LGM` = 10^18 base units

**Error Classes**:
- `LegibleError` - Base error with layer/code/details
- `CryptoError` - Cryptographic failures
- `ValidationError` - Input validation failures
- `SerializationError` - Encoding/decoding failures

### Crypto (`foundation/crypto/`)

Re-exports from `core/crypto/` with foundation-typed interfaces:
- Key generation and signing
- Domain-separated hashing
- Merkle trees (sparse and compact)
- VRF operations
- Commitment schemes

### Utils (`foundation/utils/`)

Pure utility functions:
- Byte operations (hex conversion, concatenation)
- BigInt operations (min, max, clamp, percent)
- String operations (normalize, truncate)
- Array operations (chunk, unique, groupBy)
- Object operations (deepFreeze, pick, omit)
- Time operations (sleep, now, msToBlocks)
- Serialization (deterministic JSON, bigint handling)

## Architecture Invariants Satisfied

- [x] **ARCH-1**: Layer Isolation - Foundation has no internal dependencies
- [x] **ARCH-2**: Module Boundary Contracts - Single index.ts entry point
- [x] **TYPE-1**: Shared Types in Foundation - All primitive types defined here

## Migration Strategy

The foundation layer uses a **facade pattern** for crypto:
- `foundation/crypto/index.ts` re-exports from `core/crypto/*`
- Existing code continues to work without changes
- New code can import from `@legible/foundation`
- Gradual migration: update imports module by module

## Verification

```bash
# Type check passes
npx tsc --noEmit -p tsconfig.json 2>&1 | grep "foundation/"
# (no errors)

# Property tests pass
npx vitest run tests/property/crypto/signatures.prop.ts
# ✓ 16 tests passed
```

## Next Steps (Phase 2-4)

### Phase 2: Interface Definition
- Define contracts for each domain module
- Create interface files for:
  - `KnowledgeService`
  - `ConsensusEngine`
  - `RewardsEngine`
  - `DisputeResolver`

### Phase 3: Dependency Inversion
- Inject dependencies instead of importing
- Create factory functions
- Update tests to use mocks

### Phase 4: Event-Driven Refactor
- Implement event bus
- Replace direct calls with events
- Decouple modules

## Usage

```typescript
// Import from foundation
import {
  type PublicKey,
  type DID,
  generateKeypair,
  sign,
  verify,
  ONE_LGM,
  LegibleError,
  bytesToHex,
} from './foundation/index.js';

// Generate a keypair
const { privateKey, publicKey } = generateKeypair();

// Sign a message
const message = new TextEncoder().encode('Hello, Legible Money!');
const signature = sign(message, privateKey);

// Verify
const valid = verify(signature, message, publicKey);
```
