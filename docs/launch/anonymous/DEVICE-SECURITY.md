# Device Security for Anonymous Development

> **Machine**: Mac Studio (Apple M4 Max, 36GB) — revealed in committed `.env.development`
> **Date**: 2026-02-18

---

## Current Device Fingerprints Found

### 1. Hardware Profile in Git History
**Risk**: 🟡 MODERATE
The deleted `.env.development` was committed and contains:
```
CPU: Apple M4 Max (14 cores: 10 performance + 4 efficiency)
GPU: 32 cores (Metal 3)
RAM: 36GB unified memory
```
This narrows the owner pool significantly — M4 Max 36GB is a specific SKU.

**Fix**: Must be purged from git history in the fresh repo.

### 2. Machine Hostname in Commits
**Risk**: 🔴 CRITICAL
```
yournameisai@yournameiss-Mac-Studio.local
```
Appears as committer email in multiple commits. Contains the machine name.

**Fix**: Purged when creating fresh repo with rewritten author info.

### 3. SSH Config
**Risk**: 🟢 LOW
`~/.ssh/config` only includes Colima (Docker) config. No personal SSH hosts that would link identity.

### 4. File Paths in Prompts/Reports
**Risk**: 🟡 MODERATE
Multiple files in `prompts/batch-5/` reference `/Users/yournameisai/Desktop/legible-money`. The macOS username `yournameisai` is in the path.

**Fix**: Exclude prompts/reports from public repo OR scrub all paths.

---

## Device Hardening Checklist

### Network Layer

- [ ] **VPN always-on for project work** — Mullvad recommended (accepts crypto, no account, no logs)
  - Pay with Monero or Bitcoin (not from a KYC exchange)
  - Use WireGuard protocol for speed
  - Kill switch enabled
- [ ] **DNS over HTTPS/TLS** — Prevent ISP from seeing `legible.money` lookups
  - System Preferences → Network → DNS → use `1.1.1.1` (Cloudflare) or `9.9.9.9` (Quad9)
  - Or better: route DNS through VPN tunnel
- [ ] **Tor for sensitive operations** — registrar login, GitHub account creation, email creation
  - Tor Browser for web; `torsocks` for CLI
  - Do NOT use Tor for regular development (too slow, suspicious exit node behavior)

### Browser Isolation

- [ ] **Separate browser for project** — Firefox with a dedicated profile or Brave
  - Never log into personal accounts (Google, Apple, iCloud) from this browser
  - Install: uBlock Origin, Privacy Badger, Cookie AutoDelete
- [ ] **Firefox Multi-Account Containers** — if using one browser:
  - Container 1: Personal (Google, iCloud, banking)
  - Container 2: Project (GitHub, Cloudflare, ProtonMail)
  - Containers don't share cookies, localStorage, or sessions
- [ ] **Clear project browser data** regularly

### Git Configuration

- [ ] **Per-repo git config** (already no global config — good):
  ```bash
  cd /path/to/legible-money
  git config user.name "Legible Protocol"
  git config user.email "dev@legible.money"
  ```
- [ ] **Pre-commit hook** to block personal emails:
  ```bash
  # .git/hooks/pre-commit
  #!/bin/sh
  EMAIL=$(git config user.email)
  if echo "$EMAIL" | grep -qiE "icloud|gmail|cambridge|mynameis|yournameis"; then
    echo "ERROR: Personal email detected in git config: $EMAIL"
    echo "Set project email: git config user.email dev@legible.money"
    exit 1
  fi
  ```
- [ ] **SSH key isolation** — generate a project-specific SSH key:
  ```bash
  ssh-keygen -t ed25519 -f ~/.ssh/legible_project -C "dev@legible.money"
  ```
  Add to `~/.ssh/config`:
  ```
  Host github-legible
    HostName github.com
    User git
    IdentityFile ~/.ssh/legible_project
    IdentitiesOnly yes
  ```
  Then clone via: `git clone git@github-legible:legible-protocol/legible-money.git`

### macOS Hardening

- [ ] **FileVault** — verify full disk encryption is on:
  ```bash
  fdesetup status
  ```
- [ ] **Firewall** — enable macOS firewall:
  System Settings → Network → Firewall → ON
- [ ] **Disable Spotlight cloud sync** — prevents Apple from indexing project files:
  System Settings → Siri & Spotlight → disable "Suggestions in Spotlight"
- [ ] **Disable iCloud Drive sync for Desktop** — if enabled, your project files sync to Apple servers:
  System Settings → Apple ID → iCloud → iCloud Drive → disable "Desktop & Documents Folders"
- [ ] **Screen lock** — 1 minute timeout:
  System Settings → Lock Screen → "Require password after screen saver" → Immediately
- [ ] **Disable AirDrop** when not in use

### Development Environment

- [ ] **No personal API keys in project** — verified: `.gitignore` covers secrets, no keys in git history
- [ ] **No Homebrew analytics**: `brew analytics off`
- [ ] **No npm telemetry**: `npm config set fund false && npm config set audit false`
- [ ] **OpenClaw workspace** — this workspace (`~/.openclaw/workspace/`) contains `USER.md`, `IDENTITY.md`, `BRIEFING.md` with your full identity. This is fine for private use but:
  - Never share OpenClaw workspace files
  - Never commit them to the project repo
  - They're already in a separate directory — good

### Communication

- [ ] **Project email**: ProtonMail or Tutanota (created over VPN/Tor, no phone verification)
- [ ] **Project chat**: New Discord/Telegram account for community (pseudonymous, new email)
- [ ] **No cross-posting** between personal and project social accounts
- [ ] **Writing style**: Be aware that sustained writing creates a stylometric fingerprint. AI-generated content for public docs helps mitigate this.

---

## Metadata Leakage Points

| Vector | Risk | Status |
|--------|------|--------|
| macOS username in file paths | 🟡 | Scrub from public files |
| Apple M4 Max hardware profile | 🟡 | Remove from git history |
| Timezone in commits (GMT) | 🟢 | Accept — too common to be useful |
| Commit frequency patterns | 🟢 | Accept — weak signal |
| Go/Cosmos SDK expertise level | 🟢 | Accept — shared by thousands |
| iCloud sync to Apple servers | 🟡 | Disable for Desktop folder |
| Spotlight indexing | 🟢 | Disable cloud sync |
| Homebrew/npm telemetry | 🟢 | Disable |
