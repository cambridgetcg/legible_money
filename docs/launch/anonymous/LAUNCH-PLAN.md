# Anonymous Launch Plan — Legible Money

> Step-by-step sequence. Each phase must complete before the next begins.
> **Principle**: Build the anonymous identity infrastructure FIRST, then migrate the project into it.

---

## Phase 0: Preparation (Day 1)

### 0.1 Create Anonymous Communication Layer

| Step | Tool | Notes |
|------|------|-------|
| Get a VPN | Mullvad | Pay with crypto. No account needed — just a number. Always-on for ALL project work from this point forward. |
| Create project email | ProtonMail | Over VPN. No phone verification. No personal recovery email. Use a strong, unique password. |
| Verify email privacy | — | Send test email. Check headers — no IP leakage (ProtonMail strips by default). |

**From this point forward: VPN on for ALL project-related activity. No exceptions.**

### 0.2 Create Legal Entity (Optional but Recommended)

| Step | Tool | Notes |
|------|------|-------|
| Form Wyoming LLC | Northwest Registered Agent | Use project ProtonMail. They sign as organizer — your name not on public filing. Their address is the registered agent address. Pay with crypto if possible, otherwise card (the agent relationship is private). |
| Get EIN | IRS (free) | Online, instant. LLC is the applicant, not you personally. |
| Open bank account (optional) | Mercury or Relay | For LLC, if needed for hosting/infra payments. Not strictly required if paying everything with crypto. |

**Why LLC**: Domain registration, hosting accounts, and any future service agreements need an entity. The LLC provides a privacy layer — Wyoming does not publish member names.

### 0.3 Create Development Identity

| Step | How | Notes |
|------|-----|-------|
| Choose pseudonym | — | Pick something impersonal. Not clever, not memorable, not linkable. e.g., `legible-core`, `lgm-dev`, or a random name. |
| Create GitHub account | Over VPN, with ProtonMail | New username (the pseudonym). No avatar. No bio. No stars. No followers. Enable "Keep my email addresses private" + "Block command line pushes that expose my email". |
| Create GitHub org | From new account | e.g., `legible-protocol`. This is the public-facing org. |
| Generate SSH key | `ssh-keygen -t ed25519 -f ~/.ssh/legible_project -C "dev@legible.money"` | Add to new GitHub account. Never use your personal SSH key. |
| Configure SSH | See DEVICE-SECURITY.md | Separate Host alias for the new GitHub identity. |

---

## Phase 1: Domain & Infrastructure (Day 2-3)

### 1.1 Secure the Domain

| Step | Notes |
|------|-------|
| Create new Cloudflare account | Using ProtonMail, over VPN |
| Transfer `legible.money` to new Cloudflare account | Or re-register if simpler. Ensure WHOIS privacy is ON. |
| Verify WHOIS is clean | `whois legible.money` should show only Cloudflare proxy info |
| Set up DNS | Point to hosting (Cloudflare Pages, Vercel under pseudonym, or self-hosted) |
| Set up `dev@legible.money` email routing | Cloudflare Email Routing → forward to ProtonMail |

### 1.2 Hosting & Services

| Service | Account | Notes |
|---------|---------|-------|
| GitHub (org) | New pseudonymous account | Primary code hosting |
| Cloudflare | ProtonMail account | DNS, domain, Pages |
| Docker Hub (optional) | ProtonMail account | For container images |
| Discord (community) | New account, ProtonMail | Community server |

**Rule**: Every service account uses the project ProtonMail. Never use personal email for any project service.

---

## Phase 2: Repository Migration (Day 3-4)

### 2.1 Prepare Clean Codebase

On your current machine (this is local-only work, never pushed):

```bash
# 1. Create a working copy
cp -r /Users/yournameisai/Desktop/legible-money /Users/yournameisai/Desktop/legible-money-clean
cd /Users/yournameisai/Desktop/legible-money-clean

# 2. Delete sensitive files
rm -f docs/DOMAIN_REGISTRATION.md
rm -f docs/WYOMING_LLC_SETUP.md
rm -rf docs/launch/anonymous/
rm -rf prompts/
rm -rf reports/
rm -rf .claude/memory/  # Contains development session context

# 3. Scrub identity references
# Run the scrub script from IDENTITY-SCRUB-CHECKLIST.md

# 4. Verify clean
grep -rl "yournameisai\|cambridgetcg\|mynameis.you\|yournameis.ai\|mynameisyou" \
  --include="*.go" --include="*.md" --include="*.json" --include="*.yml" \
  --include="*.yaml" --include="*.sh" --include="*.ts" --include="*.js" . | grep -v ".git/"
# Must return NOTHING
```

### 2.2 Create Fresh Git History

```bash
cd /Users/yournameisai/Desktop/legible-money-clean

# Option A: Single squashed commit (simplest, most secure)
rm -rf .git
git init
git config user.name "Legible Protocol"
git config user.email "dev@legible.money"
git add -A
git commit -m "Legible Money: Proof of Truth blockchain for AI agent economies

Built on Cosmos SDK v0.50.10 with CometBFT v0.38.
28 custom modules, 2535 tests, 242 gas entries.
See README.md for architecture and quick start."

# Option B: Rewritten history (preserves commit structure)
# Only use if you want to show development history
# WARNING: Much harder to guarantee no leaks
```

**Recommendation**: Option A. A single initial commit is normal for open-source launches. It's the safest approach.

### 2.3 Push to New Org

```bash
# Configure SSH for new identity
export GIT_SSH_COMMAND="ssh -i ~/.ssh/legible_project -o IdentitiesOnly=yes"

# Add remote and push
git remote add origin git@github-legible:legible-protocol/legible-money.git
git push -u origin main
```

### 2.4 Verify the Push

- [ ] Check new repo on GitHub — no personal info visible
- [ ] Check contributor graph — only "Legible Protocol" appears
- [ ] Check commit emails — only `dev@legible.money`
- [ ] Run `git log --format="%an <%ae> | %cn <%ce>"` — all clean

---

## Phase 3: Old Repo Cleanup (Day 4-5)

### 3.1 Archive Old Repo

| Step | Notes |
|------|-------|
| Make `yournameisai/legible_money` private | GitHub Settings → Danger Zone → Change visibility |
| Remove all collaborators | If any |
| Disable GitHub Actions | Settings → Actions → Disable |
| Consider deletion | After 30 days of private, delete entirely |

### 3.2 Check for Traces

| Check | How |
|-------|-----|
| Google: `"yournameisai" "legible"` | Search and request removal if cached |
| Google: `"cambridgetcg" "legible"` | Search and request removal if cached |
| Wayback Machine | Check `web.archive.org/web/*/github.com/yournameisai/legible*` — submit removal request if found |
| GitHub forks | Check if anyone forked the repo — if so, those forks retain full history |
| npm/Go proxy | Check `pkg.go.dev/github.com/legible-money/legible` — the module path is clean (uses org, not personal) but go proxy may have cached under old path |

---

## Phase 4: Public Launch (Day 5+)

### 4.1 Pre-Launch Checklist

- [ ] New GitHub org has clean repo with zero identity links
- [ ] Domain WHOIS shows only privacy proxy
- [ ] `dev@legible.money` email works (routed through Cloudflare → ProtonMail)
- [ ] Website deployed (no personal info, no analytics that track you)
- [ ] README links to `legible.money` not to any personal domain
- [ ] Old repo is private or deleted
- [ ] Google/Wayback traces cleaned
- [ ] VPN has been on for all activity since Phase 0

### 4.2 Launch Sequence

1. **Publish repo** — Make the new org repo public
2. **Deploy website** — legible.money goes live
3. **Announce** — From pseudonymous accounts only (Discord, Twitter/X, Reddit, Cosmos forums)
4. **Community channels** — Open Discord server, Telegram group (pseudonymous admin)
5. **Testnet** — Public testnet with faucet

### 4.3 Ongoing OPSEC

- **Never** log into personal accounts from the same browser/session as project accounts
- **Never** reference your personal life, location, or business in project communications
- **Never** push from a git config with personal email (pre-commit hook prevents this)
- **Review** every PR and commit for accidental identity leaks before merging
- **Assume** everything public is permanent — act accordingly

---

## Risk Summary

| Risk | Severity | Mitigation | Status |
|------|----------|-----------|--------|
| Git history emails | 🔴 CRITICAL | Fresh repo | Planned |
| GitHub `yournameisai` account | 🔴 CRITICAL | New account + make old private | Planned |
| CODEOWNERS `@yournameisai` | 🔴 CRITICAL | Scrub | Planned |
| Domain registration | 🟡 MODERATE | Transfer to LLC/new Cloudflare | Planned |
| `DOMAIN_REGISTRATION.md` | 🔴 CRITICAL | Delete from repo + history | Planned |
| `WYOMING_LLC_SETUP.md` | 🔴 CRITICAL | Delete from repo + history | Planned |
| Machine hostname in commits | 🔴 CRITICAL | Fresh repo | Planned |
| File paths with username | 🟡 MODERATE | Exclude prompts/reports | Planned |
| Hardware profile in .env | 🟡 MODERATE | Fresh repo | Planned |
| iCloud sync | 🟡 MODERATE | Disable for Desktop | Action needed |
| Timezone | 🟢 LOW | Accept | — |
| Writing style | 🟢 LOW | AI-generated public content | — |

---

## Timeline

| Day | Phase | Key Actions |
|-----|-------|-------------|
| 1 | Prep | VPN, ProtonMail, pseudonym |
| 2 | Entity | Wyoming LLC (optional), GitHub account/org |
| 3 | Infra | Domain transfer, Cloudflare setup, SSH keys |
| 3-4 | Migration | Scrub codebase, fresh repo, push |
| 4-5 | Cleanup | Archive old repo, check Google/Wayback traces |
| 5+ | Launch | Public repo, website, announce, testnet |

**Total time to anonymous launch readiness: ~5 days**
