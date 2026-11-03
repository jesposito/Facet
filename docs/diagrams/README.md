# Diagrams

Editable [draw.io](https://www.drawio.com/) diagrams, each with a committed PNG beside it.

| Diagram | Shows |
|---------|-------|
| [`facet-architecture`](facet-architecture.png) ([source](facet-architecture.drawio)) | The single-container runtime: Caddy in front of SvelteKit and Go/PocketBase over SQLite, with optional outside services. The tunnel in front is one example of how an owner might expose Facet. |
| [`facet-data-model`](facet-data-model.png) ([source](facet-data-model.drawio)) | The core PocketBase collections and how they relate. A subset: the full schema is in [`backend/migrations`](../../backend/migrations). |

[Architecture](../ARCHITECTURE.md) explains the runtime in words.

## Editing

Open a `.drawio` file in draw.io, diagrams.net, or the VS Code "Draw.io Integration" extension. After editing, export a fresh PNG so the two stay in step. With the draw.io desktop app and `xvfb` installed:

```bash
F=facet-architecture   # diagram name, without extension
xvfb-run -a drawio -x -f png --scale 2 -o "$F.png" "$F.drawio" --no-sandbox
```

Facet Cloud's architecture is documented in that service's own repository, not here.
