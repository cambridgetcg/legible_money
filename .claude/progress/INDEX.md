# Open-Source Release Progress Tracking

**Target**: Functional devnet with developer experience ready for open-source contributions.

---

## Critical Path Items

| Item | Status | Progress | Est. Work | Blocking |
|------|--------|----------|-----------|----------|
| [RPC Server Tests](./RPC_SERVER.md) | 🔴 Not Started | 0% | 2-3 days | Devnet API |
| [SDK Client Tests](./SDK_CLIENT.md) | 🔴 Not Started | 0% | 2-3 days | Developer adoption |
| [CLI Tool Commands](./CLI_TOOL.md) | 🟢 Complete | 100% | Done | User experience |
| [Documentation](./DOCUMENTATION.md) | 🔴 Incomplete | 30% | 2-3 days | Onboarding |

---

## Overall Progress

```
[████████████░░░░░░░░] 33%

RPC Server:    [░░░░░░░░░░░░░░░░░░░░]  0%
SDK Client:    [░░░░░░░░░░░░░░░░░░░░]  0%
CLI Tool:      [████████████████████] 100% ✓ Complete
Documentation: [██████░░░░░░░░░░░░░░] 30%
```

---

## Recommended Sequence

### Week 1: Foundation
1. **Documentation** - QUICKSTART.md, ARCHITECTURE_OVERVIEW.md
2. **RPC Server** - Server lifecycle tests, basic method tests

### Week 2: Integration
3. **SDK Client** - Connection tests, wallet tests, transaction tests
4. **RPC Server** - WebSocket tests, error handling tests

### Week 3: Completion
5. **CLI Tool** - Implement missing commands
6. **Documentation** - RPC_REFERENCE.md, DEVELOPER_GUIDE.md
7. **SDK Client** - Subscription tests, payment channel methods

---

## Quick Links

- [RPC Server Progress](./RPC_SERVER.md) - E2E tests for API server
- [SDK Client Progress](./SDK_CLIENT.md) - E2E tests for developer SDK
- [CLI Tool Progress](./CLI_TOOL.md) - Missing command implementations
- [Documentation Progress](./DOCUMENTATION.md) - Developer onboarding docs

---

## How to Update Progress

1. Open the relevant progress file
2. Check off completed items `[x]`
3. Update the Progress Log section with date and notes
4. Update this INDEX.md with new percentages

---

## Definition of Done

An item is complete when:
- [ ] All checklist items marked done
- [ ] Tests pass in CI/CD
- [ ] Code reviewed and merged
- [ ] Documentation updated (if applicable)
- [ ] Progress log updated with completion date

---

## Dependencies

```
Documentation ──────────────────────────────┐
                                            │
RPC Server ──────────► SDK Client ─────────►│──► Devnet Launch
                                            │
CLI Tool ───────────────────────────────────┘
```

- SDK Client depends on RPC Server (needs working API)
- CLI Tool can be developed in parallel
- Documentation can be written as features complete

---

## Last Updated

2026-02-05

---

## Session Continuity

When starting a new session:
1. Read this INDEX.md for overall status
2. Read the specific progress file you're working on
3. Check the Progress Log for recent updates
4. Continue from where the last session left off
