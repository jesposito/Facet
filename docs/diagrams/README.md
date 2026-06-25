# Facet — Architecture Diagrams

Editable `.drawio` files (draw.io / diagrams.net / VS Code "Draw.io Integration" extension). Each has a committed `.png` render alongside it.

| Diagram | What it shows |
|---|---|
| `facet-architecture` | Single-container runtime: Caddy → SvelteKit + Go/PocketBase → SQLite, services, external integrations |
| `facet-data-model` | Core PocketBase collections and relationships (subset of 48) |

## Tooling

Generated with the [`drawio-skill`](https://github.com/Agents365-ai/drawio-skill) Claude Code skill (installed at `~/.claude/skills/drawio-skill`). Host deps: `graphviz`, `drawio` desktop CLI + `xvfb` for headless export.

## Edit

Open the `.drawio` in draw.io or the VS Code extension. After editing, re-validate and re-export:

```bash
F=facet-architecture   # diagram name, no extension
python3 ~/.claude/skills/drawio-skill/scripts/validate.py "$F.drawio"
xvfb-run -a drawio -x -f png --scale 2 -o "$F.png" "$F.drawio" --no-sandbox
```

Facet is the single-tenant app that Facet Cloud (`../../../facets-sh`) multi-tenants. See that repo's `docs/diagrams/` for the cloud architecture.
