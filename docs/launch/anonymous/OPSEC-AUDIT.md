# Anonymous Launch OPSEC Audit

> **Purpose**: Identify and eliminate every link between Yu's real identity and the Legible Money project before public launch.
> **Threat model**: A motivated investigator (journalist, doxxer, state actor) attempts to identify the founder.
> **Date**: 2026-02-18

---

## 🔴 CRITICAL — Direct Identity Exposure

### 1. Git History — Email Addresses
**Risk**: CRITICAL — Git commits contain real email addresses in every commit object. Even if you rewrite recent commits, forks and caches may preserve them.

**Exposed emails in git log**:
- `cambridgetcg@gmail.com` — links to Cambridge TCG (trading card business)
- `mynameis.you@icloud.com` — personal iCloud
- `yournameis.ai@icloud.com` — project-specific but still iCloud (Apple account = real identity)
- `yournameisai@yournameiss-Mac-Studio.local` — contains machine hostname

**Fix**:
- [ ] Create a **new GitHub account** with a pseudonymous username and a **burner email** (ProtonMail or Tutanota, registered over VPN/Tor)
- [ ] Create a **fresh repo** with `git filter-repo` or start from a squashed commit using the new identity
- [ ] **Never push from the current repo** — the existing GitHub repo (`yournameisai/legible_money`) is permanently burned
- [ ] Delete or archive the old repo after migration

### 2. GitHub Account — `yournameisai`
**Risk**: CRITICAL — The GitHub username, profile, stars, followers, other repos, and contribution graph all create a linkable identity.

**Fix**:
- [ ] New GitHub org (e.g., `legible-protocol` or similar) owned by a new pseudonymous account
- [ ] No personal repos, stars, or follows on the new account
- [ ] New SSH key generated on a clean machine or VM
- [ ] GitHub account email must be the burner email (not iCloud)
- [ ] Enable "Keep my email addresses private" + "Block command line pushes that expose my email" in GitHub settings

### 3. CODEOWNERS File
**Risk**: CRITICAL — `.github/CODEOWNERS` references `@yournameisai` on every line.

**Fix**:
- [ ] Replace with new pseudonymous GitHub username or org team
- [ ] Or remove CODEOWNERS entirely for public launch

### 4. Domain Registration — `legible.money`
**Risk**: HIGH — WHOIS records may contain personal details depending on registrar and privacy settings.

**Fix**:
- [ ] Verify WHOIS privacy is enabled (Cloudflare provides this by default)
- [ ] If registered with personal info, **transfer to a new registrar account** created under the Wyoming LLC
- [ ] Check: `whois legible.money` should show only registrar/privacy proxy info, not personal details

### 5. `docs/DOMAIN_REGISTRATION.md`
**Risk**: CRITICAL — Contains `yournameisai@icloud.com` as the registrant email, and a template with "[Your Full Name]" placeholders suggesting personal info was used.

**Fix**:
- [ ] Delete this file from the repo AND from git history
- [ ] Re-register domain under LLC/pseudonym

### 6. `docs/WYOMING_LLC_SETUP.md`
**Risk**: HIGH — References personal email, Northwest Registered Agent details, and the structure of the LLC. While the LLC itself provides privacy (Wyoming doesn't publish member names), the *document* in the repo links the project to the LLC formation process and email.

**Fix**:
- [ ] Delete from repo AND git history
- [ ] Keep private notes locally, never commit

---

## 🟡 MODERATE — Indirect Linking Vectors

### 7. Git Author Names
**Risk**: MODERATE — Author names like "YOU", "yournameis ai", "mynameisyou-cmyk" are pseudonymous but form a pattern. An investigator cross-referencing GitHub accounts could link them.

**Fix**:
- [ ] Fresh repo with single pseudonymous author name
- [ ] `git config user.name "Legible Protocol"` or similar impersonal name

### 8. `scripts/governance/known-agents.json`
**Risk**: MODERATE — Contains `"yournameisai"` as the human manager.

**Fix**:
- [ ] Replace with pseudonymous identity

### 9. `docs/PR_WORKFLOW.md`, `docs/OPEN_SOURCE_DEV_FLOW.md`, `docs/GOVERNANCE_WORKFLOW.md`
**Risk**: LOW-MODERATE — Reference `@yournameisai` and GitHub-specific workflow details.

**Fix**:
- [ ] Search and replace all `@yournameisai` references with new pseudonym

### 10. `test-results/results.json`
**Risk**: LOW — May contain machine-specific paths or usernames.

**Fix**:
- [ ] Audit contents, scrub if needed

### 11. Commit Message Content
**Risk**: LOW — Commit messages reference "R3+R4 batch", "OpenClaw", session prompts — these reveal the development methodology (AI-assisted) but not personal identity directly.

**Fix**:
- [ ] Squash history OR rewrite commit messages for public repo
- [ ] Decision: do you *want* the AI-developed nature to be public? If yes, leave. If no, squash.

### 12. `prompts/` and `reports/` directories
**Risk**: LOW — These contain Claude Code session prompts and reports. They reveal development process but not personal identity. However, `prompts/batch-5/R5-SESSION-*.md` reference file paths containing `yournameisai`.

**Fix**:
- [ ] Scrub file paths in prompts/reports OR exclude these directories from the public repo

---

## 🟢 LOW — Behavioral / Metadata Vectors

### 13. Coding Style / Writing Style
**Risk**: LOW — Stylometry analysis could theoretically link writing patterns across projects. Mitigated by the fact that most code was AI-generated.

**Fix**: No action needed unless extremely paranoid.

### 14. Timezone / Commit Patterns
**Risk**: LOW — Commit timestamps reveal GMT/London timezone working hours.

**Fix**:
- [ ] If concerned, use `GIT_AUTHOR_DATE` and `GIT_COMMITTER_DATE` to randomize timestamps
- [ ] Or accept that timezone alone is a weak signal

### 15. Go Module Path
**Risk**: NONE — `github.com/legible-money/legible` is a project org, not personal.

### 16. LICENSE
**Risk**: NONE — Says "Legible Money Contributors", no personal names.

### 17. README / Trustpaper
**Risk**: NONE — No personal names found.

---

## 🔒 External Service Audit

### Services That May Link Identity

| Service | Risk | Current Status | Fix |
|---------|------|---------------|-----|
| **GitHub** (`yournameisai`) | 🔴 | Public repo, email in commits | New account + fresh repo |
| **Cloudflare** (domain) | 🟡 | Likely registered with personal email | Re-register under LLC |
| **iCloud** (`yournameis.ai@icloud.com`) | 🔴 | Apple account = real identity via payment | Stop using for project |
| **Gmail** (`cambridgetcg@gmail.com`) | 🔴 | Links to TCG business | Stop using for project |
| **npm** (if published) | 🟡 | Check if any packages published under personal account | Audit |
| **Discord** (if community) | 🟡 | Depends on account used | Use pseudonymous account |
| **Twitter/X** (if any) | 🟡 | Check if project was mentioned from personal account | Audit |
| **Cloudflare Pages/Workers** | 🟡 | legible.money hosting | Verify under LLC account |

---

## ✅ Recommended Launch Sequence

### Phase 1: Identity Infrastructure (Before Any Public Action)

1. **Create burner email** — ProtonMail over VPN, no phone verification, no personal recovery email
2. **Create Wyoming LLC** via Northwest Registered Agent using burner email
3. **Register new GitHub account** using burner email — generic pseudonym, no avatar, no bio
4. **Create new GitHub org** (e.g., `legible-protocol`) owned by pseudonymous account
5. **Transfer domain** to Cloudflare account registered under LLC email

### Phase 2: Repository Migration

6. **Prepare clean repo**:
   ```bash
   # Option A: Squash everything into one commit
   git checkout --orphan clean
   git add -A
   git commit --author="Legible Protocol <dev@legible.money>" -m "Initial commit"
   
   # Option B: Rewrite history (preserves commits, changes author)
   git filter-repo --email-callback 'return b"dev@legible.money"' \
                    --name-callback 'return b"Legible Protocol"'
   ```
7. **Scrub sensitive files** before committing to clean repo:
   - Delete: `docs/DOMAIN_REGISTRATION.md`, `docs/WYOMING_LLC_SETUP.md`
   - Scrub: All files containing `yournameisai`, `cambridgetcg`, `mynameis.you`
   - Scrub: `CODEOWNERS`, `known-agents.json`
   - Decision: include or exclude `prompts/` and `reports/`
8. **Push to new org** repo
9. **Archive old repo** (make private, then delete after confirming migration)

### Phase 3: Ongoing OPSEC

10. **All project work** through VPN (Mullvad or similar, paid with crypto)
11. **All project communication** through pseudonymous accounts only
12. **Never cross-contaminate** — don't log into personal accounts from the same browser session as project accounts
13. **Separate browser profiles** — Firefox containers or separate browsers entirely
14. **Git config** — always verify `user.name` and `user.email` before committing:
    ```bash
    # In the project directory
    git config user.name "Legible Protocol"
    git config user.email "dev@legible.money"
    ```

---

## Summary

| Priority | Count | Key Actions |
|----------|:-----:|-------------|
| 🔴 CRITICAL | 6 | New GitHub, fresh repo, scrub emails, delete DOMAIN_REGISTRATION.md + WYOMING_LLC_SETUP.md |
| 🟡 MODERATE | 6 | Replace @yournameisai refs, scrub file paths, audit external services |
| 🟢 LOW | 5 | Timezone, writing style, commit messages — accept or mitigate |

**The single most important action**: Create a fresh repository with rewritten git history. The current repo is irrecoverably linked to your identity through commit emails. No amount of file editing fixes this — git objects are immutable and cached by GitHub, forks, and mirrors.
