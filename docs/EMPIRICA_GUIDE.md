# Empirica Integration Guide for Facet Development

## Overview

Empirica is a **Cognitive Operating System for AI Agents** that helps track what we know and don't know about the codebase, measure uncertainty, and prevent hallucinations during development. It's integrated into the Facet project to aid in continuing development.

## What is Empirica?

Empirica provides:
- **Epistemic Tracking**: Track what we know/don't know about the codebase
- **Uncertainty Measurement**: Transparent measurement of confidence levels
- **Safety Gates**: PROCEED/HALT/BRANCH/REVISE controls for critical decisions
- **Session Continuity**: Git-native storage of development context
- **Multi-Agent Coordination**: Spawn and manage multiple AI agents for complex tasks

## Quick Start

### 1. Session Management

Sessions track development work across conversations. A session has been created for you:

```bash
# List all sessions
empirica sessions-list

# Show session details
empirica sessions-show --session-id aa34d76e-f858-4663-b295-0bc041c65838

# Create a new session
empirica session-create --ai-id "claude-facet-dev" --subject "feature-name"
```

### 2. The CASCADE Workflow

Empirica uses a three-phase workflow for development tasks:

#### PREFLIGHT (Planning Phase)
Before starting work, assess what you know and what you need to learn:

```bash
empirica preflight --session-id <SESSION_ID> --prompt "Add user authentication with OAuth"
```

This generates an epistemic assessment:
- **Engagement**: How well do we understand the task?
- **Knowledge**: What do we know about relevant systems?
- **Capability**: Can we implement this?
- **Context**: What's the current state of the codebase?

#### WORK (Implementation Phase)
During implementation, track progress and uncertainties:

```bash
# Log findings as you discover them
empirica finding-log --session-id <SESSION_ID> --finding "OAuth provider configured in .env"

# Log unknowns when you encounter gaps
empirica unknown-log --session-id <SESSION_ID> --unknown "Not sure how PocketBase handles OAuth callbacks"

# Mark unknowns as resolved
empirica unknown-resolve --session-id <SESSION_ID> --unknown-id <ID> --resolution "Found in PocketBase docs"

# Log dead ends to avoid repeating mistakes
empirica deadend-log --session-id <SESSION_ID> --approach "Tried custom middleware, conflicts with PocketBase routing"
```

#### POSTFLIGHT (Review Phase)
After completion, reflect on what was learned:

```bash
empirica postflight --session-id <SESSION_ID> --notes "Successfully implemented OAuth. Learned about PocketBase auth hooks."
```

### 3. Goals and Task Management

Track development goals and subtasks:

```bash
# Create a goal
empirica goals-create --session-id <SESSION_ID> --title "Implement OAuth authentication" --priority high

# Add subtasks
empirica goals-add-subtask --goal-id <GOAL_ID> --title "Configure OAuth providers"
empirica goals-add-subtask --goal-id <GOAL_ID> --title "Update frontend login flow"

# List goals
empirica goals-list --session-id <SESSION_ID>

# Complete subtasks
empirica goals-complete-subtask --subtask-id <SUBTASK_ID>

# Mark goal as complete
empirica goals-complete --goal-id <GOAL_ID>
```

### 4. Checkpoints (Git-like Snapshots)

Create checkpoints of your epistemic state:

```bash
# Create a checkpoint
empirica checkpoint-create --session-id <SESSION_ID> --message "Completed OAuth research phase"

# List checkpoints
empirica checkpoint-list --session-id <SESSION_ID>

# Load a previous checkpoint
empirica checkpoint-load --checkpoint-id <CHECKPOINT_ID>

# Compare checkpoints
empirica checkpoint-diff --from <CHECKPOINT_ID_1> --to <CHECKPOINT_ID_2>
```

### 5. Multi-Agent Workflows

Spawn specialized agents for complex tasks:

```bash
# Spawn an agent for a subtask
empirica agent-spawn --session-id <SESSION_ID> --role "security-auditor" --task "Review OAuth implementation for security issues"

# Check agent reports
empirica agent-report --agent-id <AGENT_ID>

# Aggregate findings from multiple agents
empirica agent-aggregate --session-id <SESSION_ID>
```

## Common Workflows for Facet Development

### Adding a New Feature

```bash
# 1. Create a session
SESSION_ID=$(empirica session-create --ai-id "claude" --subject "new-feature" --output json | jq -r '.session_id')

# 2. Run preflight assessment
empirica preflight --session-id $SESSION_ID --prompt "Add dark mode support to Facet"

# 3. Create goals
empirica goals-create --session-id $SESSION_ID --title "Implement dark mode" --priority high

# 4. During implementation, log findings
empirica finding-log --session-id $SESSION_ID --finding "Tailwind CSS already has dark mode support"
empirica unknown-log --session-id $SESSION_ID --unknown "How to persist user preference"

# 5. Create checkpoint after major milestones
empirica checkpoint-create --session-id $SESSION_ID --message "Dark mode UI complete"

# 6. Run postflight when done
empirica postflight --session-id $SESSION_ID --notes "Dark mode implemented with user preference persistence"
```

### Investigating a Bug

```bash
# 1. Create investigation session
empirica session-create --ai-id "claude" --subject "bug-investigation"

# 2. Log the investigation
empirica investigate-log --session-id <SESSION_ID> --issue "Share links not working behind reverse proxy"

# 3. Create a branch for investigation
empirica investigate-create-branch --session-id <SESSION_ID> --branch-name "investigate-share-links"

# 4. Log mistakes to avoid repeating them
empirica mistake-log --session-id <SESSION_ID> --mistake "Assumed X-Forwarded-Host was always present"

# 5. Checkpoint when you find the root cause
empirica checkpoint-create --session-id <SESSION_ID> --message "Found root cause: TRUST_PROXY not set"
```

### Code Refactoring

```bash
# 1. Create session
empirica session-create --ai-id "claude" --subject "refactoring"

# 2. Assess the component
empirica assess-component --session-id <SESSION_ID> --file "backend/hooks/view.go"

# 3. Compare before/after
empirica assess-compare --session-id <SESSION_ID> --before <CHECKPOINT_ID_1> --after <CHECKPOINT_ID_2>
```

## Project-Level Commands

### Workspace Overview

```bash
# Get an overview of all projects and sessions
empirica workspace-overview

# Map the workspace structure
empirica workspace-map
```

### Project Management

```bash
# List all projects
empirica project-list

# Bootstrap a project (analyze codebase structure)
empirica project-bootstrap --project-id 80d960f6-9bbc-4838-9803-ce84c5376b71

# Search across project
empirica project-search --project-id <PROJECT_ID> --query "OAuth"

# Handoff project to another developer
empirica project-handoff --project-id <PROJECT_ID> --output handoff.json
```

## Integration with Git

Empirica integrates with Git to track epistemic state alongside code changes:

```bash
# Empirica automatically detects the git repository
# Session data is stored in .empirica/ (already gitignored)

# Create checkpoints that align with git commits
git add .
git commit -m "Add OAuth support"
empirica checkpoint-create --session-id <SESSION_ID> --message "OAuth implementation complete"
```

## Facet Project Context

**Project ID**: `80d960f6-9bbc-4838-9803-ce84c5376b71`

**Tech Stack**:
- Backend: Go 1.24 + PocketBase
- Frontend: SvelteKit + TypeScript + Tailwind CSS
- Database: SQLite

**Key Areas**:
- `/backend/hooks/`: Custom API endpoints (10K+ lines)
- `/backend/services/`: Business logic
- `/frontend/src/routes/`: Page routes
- `/frontend/src/components/`: UI components

## Best Practices

1. **Start Every Development Session with PREFLIGHT**: Assess what you know before diving in
2. **Log Unknowns Immediately**: Don't let knowledge gaps go untracked
3. **Create Checkpoints at Milestones**: Save your epistemic state after major achievements
4. **Use Specific Session Subjects**: Makes it easier to resume later
5. **Run POSTFLIGHT to Solidify Learning**: Document what you learned for future reference

## Useful Queries

```bash
# Show all unknowns across all sessions
empirica unknown-log --session-id <SESSION_ID> --list

# Check epistemic drift (how much has changed)
empirica check-drift --session-id <SESSION_ID>

# Monitor trajectory (where the project is heading)
empirica trajectory-project --project-id <PROJECT_ID>

# Generate efficiency report
empirica efficiency-report --session-id <SESSION_ID>
```

## Configuration

Configuration is stored in [.empirica/config.yaml](.empirica/config.yaml):

```yaml
version: '2.0'
root: /workspaces/Facet/.empirica
paths:
  sessions: sessions/sessions.db
  identity: identity/
  messages: messages/
  metrics: metrics/
  personas: personas/
settings:
  auto_checkpoint: true
  git_integration: true
  log_level: info
```

## Further Reading

- [Empirica GitHub Repository](https://github.com/Nubaeon/empirica)
- [Epistemic Vectors Documentation](https://github.com/Nubaeon/empirica#core-features)
- [CASCADE Workflow Guide](https://github.com/Nubaeon/empirica#setup-instructions)

## Quick Reference

| Command | Purpose |
|---------|---------|
| `empirica session-create` | Start a new development session |
| `empirica preflight` | Assess knowledge before starting work |
| `empirica finding-log` | Record discoveries |
| `empirica unknown-log` | Track knowledge gaps |
| `empirica checkpoint-create` | Save epistemic state |
| `empirica goals-create` | Define development goals |
| `empirica postflight` | Reflect on completed work |
| `empirica workspace-overview` | See all projects and sessions |

---

**Note**: Empirica is installed and configured in this repository. Session data is stored in `.empirica/` (gitignored by default).
