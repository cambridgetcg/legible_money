# Session: GIT-COMMIT-ORGANIZE — Clean commit history for testnet branch

You are working in the legible_money Cosmos SDK chain codebase at ~/Desktop/legible_money.

## Task: Organize all batch 1-7 changes into a clean, reviewable commit history

### Context
Multiple Claude Code sessions across 7 batches have made changes. Some may be committed, some may be staged, some may be uncommitted. We need a clean git history before testnet launch.

### Step 1: Assess current state
```bash
git status
git log --oneline -20
git diff --stat HEAD
git diff --stat --cached
```

### Step 2: Identify uncommitted changes
```bash
git diff --name-only HEAD
```
Categorize each uncommitted file by which batch/session created it.

### Step 3: Create logical commits
Group changes by topic, not by session. Use conventional commit format:

Suggested commit plan (adjust based on actual uncommitted state):

```
fix(app): add 6 missing module accounts to maccPerms

  ontology, research, schedule, discovery, bvm, treasury_protocol
  all need module accounts for bank keeper coin operations.
  Without these, runtime panics occur on first coin transfer.
```

```
fix(staking,gov): wire CLI commands and fix gRPC query paths

  - staking: GetTxCmd/GetQueryCmd returned nil despite CLI code existing
  - gov: gRPC query path used lgm_gov instead of gov proto package
  - gov: GetTxCmd/GetQueryCmd returned nil
```

```
feat(tokens): add --mintable/--burnable/--pausable/--wrappable flags

  create-token CLI now populates TokenFeatures from flags.
  Without these, all tokens created as non-mintable/non-burnable.
```

```
feat(genesis): add production testnet genesis ceremony

  - testnet-genesis.sh: full ceremony script (init/add-validator/finalize)
  - testnet-genesis-config.json: 410+ params across 28 modules
  - docs/TESTNET-LAUNCH.md: launch guide
```

```
fix(knowledge): add missing events + Burner permission

  - ChallengeProvisionalFact and UpdateParams events added
  - knowledge module needs Burner for challenge bond slashing
```

```
chore(build): set binary version via ldflags
```

```
chore(reports): batch-6 and batch-7 reports
```

### Step 4: Execute
For each commit:
- `git add <specific files>`
- `git commit -m "<message>"`
- Verify build still passes after each commit

### Step 5: Final verification
```bash
git log --oneline -15
go build ./...
go test ./... -count=1 -timeout 600s
```

## Report format
Write to reports/batch-7/SESSION-B7-5-GIT-ORGANIZE.md:
- Final commit list with hashes and messages
- Files per commit
- Build verification after last commit
- Any files intentionally left uncommitted and why
