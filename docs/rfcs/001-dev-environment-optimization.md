# RFC-001: Dev Environment Optimization

**Date**: 2025-01-01
**Status**: Implementing
**Author**: Claude Code

## Problem Statement

The current dev environment has two critical issues:

1. **Air fails in Codespaces**: Running `air` from the repo root fails with:
   ```
   go: cannot find main module, but found .git/config in /workspaces/Me.yaml
   ```
   This happens because `go.mod` is in `./backend/`, not the repo root.

2. **8-minute rebuild loops**: Every Codespaces start runs:
   - `npm install` (~2-3 min)
   - `go mod tidy` + `go build` (~1-2 min)
   - Full dependency downloads even when unchanged

## Goals

1. Fix air to work correctly from repo root with `go.mod` in `./backend/`
2. Reduce Codespaces startup from ~8 minutes to ~15 seconds (warm start)
3. Implement lockfile hash-based caching for npm and Go modules
4. Improve developer experience with clear documentation

## Non-Goals

- Changing the monorepo structure (backend + frontend in subdirs)
- Switching from npm to pnpm (keep changes minimal)
- Adding new tooling beyond what's needed

## Analysis

### Root Cause 1: Air + Go Module Location

The root `.air.toml` has:
```toml
root = "."
cmd = "go build -o ./tmp/me-yaml ./backend"
```

When `air` runs from repo root, Go looks for `go.mod` in the current directory.
The `./backend` in the build command tells Go where source files are, but Go
still requires `go.mod` in the working directory or parent.

**Solution**: Change air's working directory to `./backend/` and update paths accordingly.

### Root Cause 2: No Dependency Caching

Current `postCreateCommand`:
```json
"postCreateCommand": "cd frontend && npm install && npx svelte-kit sync && cd ../backend && go mod tidy && go build -o ../pb_data/me-yaml ."
```

This runs EVERY time, even when dependencies haven't changed.

**Solution**:
- Move expensive operations to Dockerfile (cached in layers)
- Use hash-based scripts that skip work when lockfiles unchanged
- Persist node_modules and Go module cache with named volumes

## Implementation Plan

### Phase 0: Fix Air Bug

1. Update root `.air.toml` to work with backend subdirectory:
   - Set correct working directory semantics
   - Adjust build and run commands to be relative to backend

2. Update `scripts/start-dev.sh` to run air correctly

3. Verification: `air` starts and rebuilds on backend file changes

### Phase 1: Optimize Startup

1. Create `scripts/dev-frontend.sh`:
   - Calculate SHA256 of package-lock.json
   - Compare with cached hash in node_modules/.lockfile-hash
   - Only run `npm install` if hash differs or node_modules missing
   - Run `npm run dev -- --host`

2. Create `scripts/dev-backend.sh`:
   - Calculate SHA256 of go.mod + go.sum
   - Compare with cached hash
   - Only run `go mod download` if changed
   - Start air for hot reload

3. Update `devcontainer.json`:
   - Remove npm install from postCreateCommand
   - Remove go mod tidy from postCreateCommand
   - Keep only minimal bootstrap in postStartCommand

4. Add VS Code tasks for common operations

### Phase 2: Documentation

1. Create `docs/DEV.md` with:
   - Codespaces quick start
   - Local Docker workflow
   - Troubleshooting guide
   - Expected ports and URLs

## Risks

| Risk | Mitigation |
|------|------------|
| Volume permissions in devcontainer | Use vscode user, set proper permissions |
| Hash calculation overhead | SHA256 of lockfiles is <1ms |
| Cache invalidation edge cases | Hash comparison is deterministic |

## Rollback Plan

If issues arise:
1. Revert changes to `.air.toml`
2. Revert changes to `devcontainer.json`
3. Remove new scripts (dev-frontend.sh, dev-backend.sh)
4. Previous configuration will work (just slower)

## References

- [Air Hot Reload](https://github.com/air-verse/air) - Working directory configuration
- [VS Code Devcontainers](https://containers.dev/implementors/json_reference/) - postCreate vs postStart
- [Go Modules](https://go.dev/ref/mod) - GOPATH and module cache
