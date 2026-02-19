# B8-7: Developer Documentation + Module Reference

## Context
Testnet launch needs documentation for validators, developers, and the community. Current docs are scattered across reports and code comments.

## Task

### 1. Create module reference
For each of the 28 custom modules, create a one-page reference in `docs/modules/`:

```
docs/modules/
├── README.md          (index + architecture overview)
├── knowledge.md       (claims, verification, PoT)
├── staking.md         (validator staking, reputation)
├── gov.md             (governance, proposals)
├── tokens.md          (LGM-20 token standard)
├── home.md            (agent homes, guardians)
├── bvm.md             (belief virtual machine)
├── ...
```

Each module doc should contain:
- **Purpose**: One paragraph explaining what it does
- **Key concepts**: Core types and their relationships
- **CLI commands**: List of tx and query commands with examples
- **Parameters**: Governance-adjustable params
- **Events**: Emitted events
- **Dependencies**: Which other modules it interacts with

### 2. Source from code, not imagination
Read each module's:
- `types/` — message types, params, keys
- `keeper/` — core logic
- `client/cli/` — available commands
- `module.go` — registered services

**Do not invent features that don't exist in the code.**

### 3. Architecture overview
Create `docs/modules/README.md` with:
- Module dependency graph (text-based)
- Which modules are in which layer (consensus, economic, social, infrastructure)
- Data flow for key operations (claim submission → verification → confidence)

### 4. Validator guide
Update `docs/TESTNET-LAUNCH.md` or create `docs/VALIDATOR-GUIDE.md`:
- Hardware requirements
- Setup steps (from scratch)
- Monitoring recommendations
- Common troubleshooting

### 5. Verify
- All CLI command examples are real (test at least 5)
- No references to unimplemented features
- Links between docs are valid

## File Ownership
- `docs/modules/` (new directory)
- `docs/VALIDATOR-GUIDE.md` (new or updated)

## Report
Write `reports/batch-8/SESSION-B8-7-DOCUMENTATION.md`.
