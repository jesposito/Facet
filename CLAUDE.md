# Claude Instructions for NotMyself.IO Development Environment

## Prime Directive

> **Rules in `.claude/rules/` are STRICTLY ENFORCED.** They override all default behavior and must be followed exactly as written. Violations break workflows, lose work, or corrupt data.

---

## Critical Rules Summary

These rules come from `.claude/rules/`. Read them in full. Here's what breaks if you violate them:

| Rule File | Critical Requirement | What Breaks If Violated |
|-----------|---------------------|------------------------|
| `tracking-github.md` | Link PRs to issues with `Fixes #N` | Issues don't auto-close, no traceability |
| `tracking-beads.md` | Sync beads before session end | Work becomes invisible, items orphaned |
| `workflow-implementation.md` | Set item + parents to `in_progress` before coding | Work invisible, parent items never close |
| `workflow-planning.md` | Never create items before plan approval | Wasted work, wrong structure |
| `scripting-hooks.md` | All hooks must be Bun TypeScript | Cross-platform compatibility fails |
| `style-csharp.md` | Use `is not null`, file-scoped namespaces (.NET 10+) | Linting errors, inconsistent codebase |

---

## Session Protocol (MANDATORY)

**Work is NOT complete until pushed to remote.**

<commands>
```bash
# Before ending ANY session:
git status                    # 1. Check what changed
git add <files>               # 2. Stage code changes
bd sync --from-main           # 3. Sync beads
git commit -m "..."           # 4. Commit code
git pull --rebase && git push # 5. Push to remote
git status                    # 6. Verify "up to date"
```
</commands>

**NEVER:**
- Stop before pushing (leaves work stranded)
- Say "ready to push when you are" (YOU must push)
- Skip `bd sync --from-main` (causes merge conflicts later)

→ Full workflow: See `workflow-implementation.md`

---

## Environment Overview

You have access to 8 plugins providing:
- **Issue tracking**: Beads (`bd` commands) → See `tracking-beads.md`
- **Code review**: `/plannotator-review`, `/code-review`
- **Git automation**: `/commit`, `/commit-push-pr`, `/clean_gone`
- **Language intelligence**: TypeScript, C#, Ruby LSPs
- **Security guidance**: Background security analysis

---

## Permissions

**Pre-approved access:**
- Git operations (except force push, hard reset)
- .NET, npm, Docker, PowerShell commands
- GitHub CLI (`gh`) for issues, PRs, actions
- MCP tools for Docker and Chrome automation

**Requires user confirmation:**
- `git push --force`, `git reset --hard`
- `docker system prune`
- `rm -rf`, `Remove-Item -Recurse -Force`

---

## Available Slash Commands

| Command | Purpose |
|---------|---------|
| `/plan-new` | Start planning workflow → See `workflow-planning.md` |
| `/commit` | Create git commit with AI message |
| `/commit-push-pr` | Commit, push, and create PR |
| `/clean_gone` | Remove branches deleted on remote |
| `/code-review` | Comprehensive code review |
| `/plannotator-review` | Interactive code review |
| `/beads` | Full beads workflow guide |
| `/ready` | Show ready-to-work issues |
| `/blocked` | Show blocked issues |
| `/stats` | Project statistics |
| `/show <id>` | Issue details |
| `/search <query>` | Search issues |

---

## Project-Specific Rules

**Location:** `.claude/rules/`

| Rule File | Applies To | Purpose |
|-----------|------------|---------|
| `tracking-github.md` | All projects | GitHub Issues, PRs, Actions workflows |
| `tracking-beads.md` | All projects | Local issue tracking, dependencies, session persistence |
| `workflow-planning.md` | All projects | Master Plan and Feature Plan workflows |
| `workflow-implementation.md` | All projects | Pre-work validation, coding, closing procedures |
| `workflow-self-learning.md` | All projects | Recall before work, record after work, review recommendations |
| `config-settings.md` | All projects | Token substitution from `.devenv/settings.json` |
| `scripting-hooks.md` | All projects | Bun TypeScript requirement, hook templates |
| `scripting-safety.md` | All projects | Pre-flight checks for scripts that modify files |
| `style-csharp.md` | .NET 10+ only | C# 14 patterns, naming, modern syntax |
| `progressive-disclosure.md` | All projects | Rule structure: concise main files, code in subdirectories |
| `versioning.md` | All projects | Semver, git tags, CHANGELOG maintenance |

---

## Frontend i18n Requirements

**All user-visible strings in the frontend MUST use the i18n system.**

| Requirement | Details |
|-------------|---------|
| Use `$t()` for all text | `{$t('admin.sidebar.dashboard')}` not `Dashboard` |
| Update ALL locale files | `en.json`, `de.json`, `elvish.json`, `klingon.json`, `lolcat.json` |
| Run validation | `cd frontend && npm run i18n:validate` |
| Import correctly | `import { t } from 'svelte-i18n';` |

**Key structure:** `{area}.{section}.{element}` (e.g., `admin.sidebar.dashboard`)

**What breaks if violated:**
- Hardcoded strings won't translate when users switch languages
- Missing keys cause fallback to English or display raw key paths
- Inconsistent locale files fail CI validation

→ Full translation guide: See `CONTRIBUTING_TRANSLATIONS.md`

**Code samples:** Command examples are in `.claude/rules/code/` subdirectories. See `progressive-disclosure.md` for the pattern.

> **Detailed workflow documentation**: See `docs/workflow.md`

---

## Error Recovery

<commands>
```bash
# Beads issues
bd doctor              # Diagnose problems
bd sync --import-only  # Fix sync divergence
bd sync --force        # Force re-sync from source

# Session context lost (after compaction or new session)
bd prime               # Reload beads context (auto-runs via hooks)
bd ready               # Find where you left off
```
</commands>

→ Full error recovery: See `tracking-beads.md` (Edge Cases section)
