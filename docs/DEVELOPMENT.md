# Agent Tooling - Integrated Development Environment

This project uses three integrated tools for AI-assisted development:

## 🧠 Empirica - Epistemic Self-Assessment
Track what you know and learn throughout development sessions.

```bash
# Start session
empirica session-create --ai-id claude-code --output json

# Before work: What do I know?
empirica preflight-submit -

# After work: What did I learn?
empirica postflight-submit -
```

## 📋 Beads (bd) - Task & Issue Tracking
Distributed, git-backed graph issue tracker for structured task management.

```bash
# Find available work
bd ready

# View issue details
bd show <id>

# Claim work
bd update <id> --status in_progress

# Complete work
bd close <id>

# Sync with git
bd sync
```

## 🎨 Perles - Visual Task Management
Terminal UI for Beads with kanban boards and BQL query language.

```bash
# Launch TUI
perles

# Switch between Kanban and Search
ctrl+space

# View help
?
```

## 🔄 Integrated Workflow

1. **Session Start**
   - Run `empirica session-create` to begin tracking
   - Run `empirica preflight-submit -` to document initial knowledge
   - Run `bd ready` to identify available tasks

2. **During Development**
   - Use `bd` commands to track task progress
   - Use `perles` to visualize and manage kanban board
   - Update issue status as work progresses

3. **Session End**
   - Run `empirica postflight-submit -` to document learnings
   - Run `bd sync` to synchronize issues with git
   - See [AGENTS.md](../AGENTS.md) for mandatory landing-the-plane workflow

See [agent-instructions.md](../agent-instructions.md) for complete development guidelines (if present).
