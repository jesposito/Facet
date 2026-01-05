# Empirica Quick Start

Empirica is now installed and configured for your Facet project!

## What's Set Up

- **Empirica v1.2.3** installed
- **Project created**: Facet (ID: `80d960f6-9bbc-4838-9803-ce84c5376b71`)
- **Active session**: `aa34d76e-f858-4663-b295-0bc041c65838`
- **Configuration**: `.empirica/config.yaml`
- **Git integration**: Enabled (`.empirica/` is gitignored)

## Your First Development Session

### 1. Start with PREFLIGHT (Planning)

```bash
empirica preflight --session-id aa34d76e-f858-4663-b295-0bc041c65838 --prompt "Add new feature: user profile customization"
```

This assesses:
- What you know about the codebase
- What unknowns exist
- Whether you're ready to proceed

### 2. Work Phase (Track as You Go)

```bash
# Log discoveries
empirica finding-log --session-id aa34d76e-f858-4663-b295-0bc041c65838 --finding "Found profile settings in backend/hooks/profile.go"

# Track knowledge gaps
empirica unknown-log --session-id aa34d76e-f858-4663-b295-0bc041c65838 --unknown "Not sure how to update profile schema in PocketBase"

# Resolve unknowns when figured out
empirica unknown-resolve --session-id aa34d76e-f858-4663-b295-0bc041c65838 --unknown-id <ID> --resolution "Found in migrations/"
```

### 3. POSTFLIGHT (Reflection)

```bash
empirica postflight --session-id aa34d76e-f858-4663-b295-0bc041c65838 --notes "Successfully added profile customization. Learned about PocketBase migrations."
```

## Quick Commands

```bash
# View workspace
empirica workspace-overview

# List sessions
empirica sessions-list

# Create new session
empirica session-create --ai-id "claude" --subject "feature-name"

# Create checkpoint
empirica checkpoint-create --session-id <SESSION_ID> --message "Completed major milestone"

# Create goal
empirica goals-create --session-id <SESSION_ID> --title "Your goal" --priority high
```

## Documentation

Full guide: [docs/EMPIRICA_GUIDE.md](docs/EMPIRICA_GUIDE.md)

## Current Session ID

For convenience, your active session ID is:
```
aa34d76e-f858-4663-b295-0bc041c65838
```

You can use it in all commands, or create a new session for different work streams.

## Next Steps

1. Read the full guide: `cat docs/EMPIRICA_GUIDE.md`
2. Bootstrap your project to analyze the codebase:
   ```bash
   empirica project-bootstrap --project-id 80d960f6-9bbc-4838-9803-ce84c5376b71
   ```
3. Create your first goal:
   ```bash
   empirica goals-create --session-id aa34d76e-f858-4663-b295-0bc041c65838 --title "Continue Facet development" --priority high
   ```

Happy coding with Empirica!
