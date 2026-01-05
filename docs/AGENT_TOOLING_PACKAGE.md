# Agent Tooling Package

The [`/agent-tooling-setup`](../agent-tooling-setup/) directory contains a standalone, distributable installer for the integrated agent development environment used in Facet.

## What It Does

Installs and configures three integrated tools for AI-assisted development:

- 🧠 **Empirica** - Epistemic self-assessment
- 📋 **Beads** - Git-backed issue tracker
- 🎨 **Perles** - Terminal UI with kanban boards

## Why It's Here

This package was created during Facet's development and can be:

1. **Used as-is** in this repo
2. **Extracted** as a standalone repository
3. **Distributed** to other projects
4. **Customized** for specific workflows

## Quick Start

### Install in Another Project

```bash
cd /path/to/your/project
/path/to/Facet/agent-tooling-setup/install.sh
```

### Extract as Standalone Repo

See [`/agent-tooling-setup/DISTRIBUTION.md`](../agent-tooling-setup/DISTRIBUTION.md) for detailed instructions.

## Files

```
agent-tooling-setup/
├── install.sh              # Main installer script
├── uninstall.sh           # Removal script
├── README.md              # User documentation
├── DISTRIBUTION.md        # Distribution guide
├── LICENSE                # MIT license
├── Makefile              # Build/test automation
└── .github/
    └── workflows/
        └── test.yml       # CI testing
```

## What Gets Installed

The installer creates:

```
your-project/
├── .beads/               # Beads database
│   ├── beads.db
│   └── config.yaml
├── .claude/
│   └── CLAUDE.md        # Quick reference
├── AGENTS.md            # Workflow guide
├── .gitattributes       # Git merge strategy
└── agent-instructions.md # Full guidelines (optional)
```

## Testing

```bash
cd /workspaces/Facet/agent-tooling-setup
make test
```

## Relationship to Facet

- **Facet uses this setup** - The `.beads/`, `.claude/`, and `AGENTS.md` in Facet root were created with this installer
- **Self-contained** - The package has no dependencies on Facet and can be used standalone
- **Customizable** - You can modify the package without affecting Facet

## Distribution Options

1. **Direct use**: Point users to this directory in Facet repo
2. **Standalone repo**: Extract and publish as separate project
3. **Tarball**: `make package` creates distributable archive
4. **One-liner**: Host on GitHub for `curl | bash` install

See [DISTRIBUTION.md](../agent-tooling-setup/DISTRIBUTION.md) for details.

## Maintenance

When updating the package:

1. Test: `make test`
2. Update version in `install.sh`
3. Document changes in `README.md`
4. Consider extracting to standalone repo if widely used

## License

MIT - Same as Facet

## See Also

- [AGENTS.md](../AGENTS.md) - Workflow created by installer
- [.claude/CLAUDE.md](../.claude/CLAUDE.md) - Quick reference
- [agent-instructions.md](../agent-instructions.md) - Full guidelines
- [Beads](https://github.com/steveyegge/beads)
- [Perles](https://github.com/zjrosen/perles)
