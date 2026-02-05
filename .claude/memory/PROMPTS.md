# Prompts & Invocations

> Exact prompts that work well for this project.
> Copy-paste these for consistent results.

---

## Session Management

### Start New Session
```
Read .claude/memory/CURRENT_STATE.md and .claude/memory/BLOCKED.md to understand where Legible Money development stands. Then continue with the next priority task.
```

### Full Context Restore
```
Load complete Legible Money context:
1. Read .claude/CLAUDE.md for project overview
2. Read .claude/memory/CURRENT_STATE.md for current state
3. Read .claude/memory/DECISIONS.md for past decisions
4. Read .claude/memory/INSIGHTS.md for accumulated wisdom

Then tell me the project status and what should be done next.
```

### End Session
```
Before ending this session:
1. Update .claude/memory/CURRENT_STATE.md with what was accomplished
2. Add any new decisions to .claude/memory/DECISIONS.md
3. Add any new insights to .claude/memory/INSIGHTS.md
4. Create session log at .claude/sessions/YYYY-MM-DD.md
5. Note any blockers in .claude/memory/BLOCKED.md
```

---

## Development Tasks

### Implement New Module
```
Using Purpose Prompter methodology:
1. Assess territory (charted or uncharted?)
2. Read existing related files to understand patterns
3. Design the implementation approach
4. Implement following existing code patterns
5. Verify against PP gates

Implement [MODULE_NAME] for Legible Money.
```

### Continue Implementation
```
Read .claude/memory/CURRENT_STATE.md to see what's next.
Then implement the next pending item following existing patterns in the codebase.
Use PP methodology and track progress with tasks.
```

### Fix Bug
```
Read the relevant module files first. Then:
1. Understand the bug
2. Find root cause
3. Design minimal fix
4. Implement fix
5. Document in session log

Bug: [DESCRIPTION]
```

### Add Feature
```
For feature [NAME]:
1. Read .claude/CLAUDE.md for architecture context
2. Read related existing modules
3. Design feature following existing patterns
4. Implement with tests
5. Update CURRENT_STATE.md
6. Log decisions in DECISIONS.md
```

---

## Research & Planning

### Explore Options
```
Use PP to explore options for [TOPIC]:
1. List all viable approaches
2. Evaluate each against project constraints
3. Recommend best option with rationale
4. Document decision in DECISIONS.md
```

### Design Component
```
Using PP path-builder approach, design [COMPONENT]:
1. Map what exists (read relevant files)
2. Identify integration points
3. Design architecture
4. Document in design doc or DECISIONS.md
5. Get approval before implementing
```

### Technical Deep Dive
```
Deep dive into [TOPIC] for Legible Money:
1. Read all relevant existing code
2. Research best practices
3. Analyze trade-offs
4. Document findings in INSIGHTS.md
5. Recommend approach
```

---

## Code Operations

### Read Module Context
```
Read these files to understand [MODULE]:
- [FILE1]
- [FILE2]
- [FILE3]
Then explain how they work together.
```

### Refactor Code
```
Refactor [COMPONENT] in Legible Money:
1. Read existing implementation
2. Identify issues
3. Design cleaner approach
4. Implement incrementally
5. Ensure no behavior change
6. Document rationale in DECISIONS.md
```

### Add Tests
```
Add tests for [MODULE]:
1. Read the module implementation
2. Identify test cases (happy path, edge cases, errors)
3. Write tests following project patterns
4. Ensure all public APIs tested
5. Update CURRENT_STATE.md with test coverage
```

---

## Documentation

### Document Module
```
Create documentation for [MODULE]:
1. Read the implementation
2. Document purpose
3. Document API
4. Add usage examples
5. Note any limitations
```

### Update Memory Files
```
Update Legible Money memory files:
1. CURRENT_STATE.md - reflect current progress
2. DECISIONS.md - add any new decisions
3. INSIGHTS.md - add any new insights
4. BLOCKED.md - update blockers status
```

### Create Session Log
```
Create session log for today's work:
1. List what was attempted
2. List what was completed
3. List what wasn't completed and why
4. Note decisions made
5. Identify next priorities
Save to .claude/sessions/YYYY-MM-DD.md
```

---

## Specific Modules

### Consensus Work
```
For consensus-related work, first read:
- consensus/pot/types.ts
- consensus/pot/engine.ts
- .claude/context/CORE_TYPES.md (Consensus Types section)
Then proceed with: [TASK]
```

### State Work
```
For state-related work, first read:
- state/types.ts
- state/state_manager.ts
- .claude/context/API_PATTERNS.md (State Manager section)
Then proceed with: [TASK]
```

### Identity Work
```
For identity-related work, first read:
- agents/identity/account.ts
- agents/identity/session_keys.ts
- .claude/context/CORE_TYPES.md (Agent Types section)
Then proceed with: [TASK]
```

### Crypto Work
```
For cryptographic work, first read:
- core/crypto/hashing.ts
- core/crypto/signatures.ts
- core/crypto/merkle.ts
Then proceed with: [TASK]
```

---

## Troubleshooting

### Understand Error
```
I'm seeing this error in Legible Money:
[ERROR]

Read relevant source files and explain:
1. What's causing the error
2. Why it's happening
3. How to fix it
```

### Debug Issue
```
Debug this issue:
[DESCRIPTION]

1. Read related code
2. Trace the execution path
3. Identify the problem
4. Propose fix
5. Implement fix
```

### Performance Issue
```
Analyze performance of [COMPONENT]:
1. Read implementation
2. Identify bottlenecks
3. Propose optimizations
4. Evaluate trade-offs
5. Implement if approved
```

---

## Project Management

### Status Check
```
Read .claude/memory/CURRENT_STATE.md and give me:
1. What's complete
2. What's in progress
3. What's blocked
4. What's next
5. Overall project health
```

### Prioritize Work
```
Based on .claude/memory/CURRENT_STATE.md and BLOCKED.md:
1. List all pending work
2. Identify dependencies
3. Recommend priority order
4. Explain rationale
```

### Estimate Scope
```
For [FEATURE/TASK]:
1. Break down into subtasks
2. Identify what exists vs what's new
3. List technical challenges
4. Note any blockers
(Don't give time estimates, just scope)
```

---

## Integration

### Connect Modules
```
Connect [MODULE_A] to [MODULE_B]:
1. Read both modules
2. Identify integration points
3. Design connection
4. Implement integration
5. Test end-to-end
```

### Wire Up Flow
```
Implement the flow for [USE_CASE]:
1. Map the data flow across modules
2. Identify missing pieces
3. Implement connections
4. Add any needed types
5. Test the full flow
```

---

_Use these prompts for consistent, effective sessions._
