# Identity Scrub Checklist

> Every file that contains identity-linking information.
> All must be scrubbed or excluded before the public repo is created.

---

## Files Containing Direct Identity References

### 🔴 DELETE from public repo (contain personal details)

| File | Contains | Action |
|------|----------|--------|
| `docs/DOMAIN_REGISTRATION.md` | `yournameisai@icloud.com`, registrant template | DELETE |
| `docs/WYOMING_LLC_SETUP.md` | Personal email, LLC formation details | DELETE |
| `docs/launch/anonymous/*` | This entire OPSEC planning directory | DELETE |

### 🟡 SCRUB — replace identity references

| File | Contains | Replace With |
|------|----------|-------------|
| `.github/CODEOWNERS` | `@yournameisai` (34 occurrences) | `@<new-pseudonym>` or org team |
| `scripts/governance/known-agents.json` | `"yournameisai"` as human manager | `"<new-pseudonym>"` |
| `scripts/governance/test-voting-pipeline.js` | References to `yournameisai` | `"<new-pseudonym>"` |
| `docs/PR_WORKFLOW.md` | `@yournameisai` | `@<new-pseudonym>` |
| `docs/OPEN_SOURCE_DEV_FLOW.md` | `@yournameisai` | `@<new-pseudonym>` |
| `docs/GOVERNANCE_WORKFLOW.md` | `@yournameisai` | `@<new-pseudonym>` |
| `test-results/results.json` | Possibly machine-specific paths | Audit and scrub |

### 🟡 SCRUB — file paths containing username

| File | Contains | Action |
|------|----------|--------|
| `prompts/batch-4/R4-SESSION-7-FULL-BUILD-GREEN-COMMIT.md` | `/Users/yournameisai/` paths | Scrub or exclude |
| `prompts/batch-5/R5-SESSION-2-DEVNET-BOOT-TO-BLOCK.md` | `/Users/yournameisai/` paths | Scrub or exclude |
| `prompts/batch-5/R5-SESSION-5-DOCKER-MULTIVALIDATOR.md` | `/Users/yournameisai/` paths | Scrub or exclude |
| `prompts/batch-5/R5-SESSION-6-E2E-TX-LIFECYCLE.md` | `/Users/yournameisai/` paths | Scrub or exclude |
| `prompts/batch-5/R5-SESSION-7-TESTNET-READINESS.md` | `/Users/yournameisai/` paths | Scrub or exclude |

### Decision: `prompts/` and `reports/` directories

**Option A**: Exclude entirely from public repo — these are internal development artifacts
**Option B**: Include but scrub all paths — shows development methodology (AI-built chain)
**Recommendation**: Option A unless you want to make the AI development process public

---

## Git History — Cannot Be Fixed By File Editing

The following are **baked into git objects** and survive any file edit:

| Data | Where | Fix |
|------|-------|-----|
| `cambridgetcg@gmail.com` | Author/committer email in commits | Fresh repo only |
| `mynameis.you@icloud.com` | Author/committer email in commits | Fresh repo only |
| `yournameis.ai@icloud.com` | Author/committer email in commits | Fresh repo only |
| `yournameisai@yournameiss-Mac-Studio.local` | Author/committer email in commits | Fresh repo only |
| `YOU`, `mynameisyou-cmyk`, `yournameis ai` | Author names | Fresh repo only |
| M4 Max hardware profile | In deleted `.env.development` blob | Fresh repo only |
| `docs/DOMAIN_REGISTRATION.md` content | In past tree objects | Fresh repo only |
| `docs/WYOMING_LLC_SETUP.md` content | In past tree objects | Fresh repo only |

**There is no way to scrub these from the existing repo.** Even `git filter-repo` leaves traces if the repo has been pushed and forked. The only safe approach is a **fresh repo**.

---

## External References to Audit

| Location | What to Check |
|----------|---------------|
| GitHub `yournameisai` profile | Stars, repos, followers, contribution graph |
| GitHub repo forks | Anyone who forked the old repo retains full history |
| GitHub Actions logs | May contain email/username in CI output |
| Dependabot PRs | Created by bot but on the `yournameisai` repo |
| npm/pkg registries | Check if any packages published under personal account |
| Cloudflare dashboard | Domain registered under which email? |
| legible.money DNS | Currently pointing to Cloudflare — verify WHOIS privacy |
| Google cache | Search `"yournameisai" legible money` and `"cambridgetcg" legible` |
| Wayback Machine | Check if old README or repo was archived |

---

## Scrub Script (to run before creating clean repo)

```bash
#!/bin/bash
# Run from repo root BEFORE creating the clean public repo

# Delete sensitive docs
rm -f docs/DOMAIN_REGISTRATION.md
rm -f docs/WYOMING_LLC_SETUP.md
rm -rf docs/launch/anonymous/

# Optionally remove internal development artifacts
rm -rf prompts/
rm -rf reports/

# Scrub @yournameisai in remaining files
find . -type f \( -name "*.md" -o -name "*.json" -o -name "*.js" -o -name "*.yml" \) \
  -not -path "./.git/*" \
  -exec sed -i '' 's/@yournameisai/@legible-core/g' {} +

find . -type f \( -name "*.md" -o -name "*.json" -o -name "*.js" -o -name "*.yml" \) \
  -not -path "./.git/*" \
  -exec sed -i '' 's/"yournameisai"/"legible-core"/g' {} +

# Verify no leaks remain
echo "=== Remaining identity references ==="
grep -rl "yournameisai\|cambridgetcg\|mynameis.you\|yournameis.ai\|mynameisyou" \
  --include="*.go" --include="*.md" --include="*.json" --include="*.yml" \
  --include="*.yaml" --include="*.sh" --include="*.ts" --include="*.js" \
  . 2>/dev/null | grep -v ".git/"
```
