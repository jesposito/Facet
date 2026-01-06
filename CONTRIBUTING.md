# Contributing to Facet

Thanks for your interest in contributing! This document covers how to get started.

## Getting Started

1. Fork the repository
2. Clone your fork
3. Set up local development (see [docs/DEV.md](docs/DEV.md))

```bash
# Install prerequisites
# - Go 1.24+
# - Node.js 20+
# - Air (go install github.com/air-verse/air@v1.61.7)

# Start development
make dev
```

## Making Changes

### Before You Start

- Check existing issues to avoid duplicate work
- For large features, open an issue first to discuss the approach
- Read [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) to understand the codebase

### Code Standards

**Backend (Go):**
- Follow standard Go formatting (`go fmt`)
- Add tests for new functionality
- Keep hooks focused and services reusable

**Frontend (SvelteKit/TypeScript):**
- Use TypeScript strictly (no `any` types)
- Follow existing component patterns
- Use Tailwind for styling

**General:**
- No commented-out code
- Meaningful commit messages
- Update docs if you change behavior

### Testing

```bash
# Run all tests
make test

# Run specific test suites
cd frontend && npm run test:public
```

## Submitting Changes

1. Create a branch from `main`
2. Make your changes with clear commits
3. Ensure tests pass
4. Push to your fork
5. Open a Pull Request

### PR Guidelines

- Describe what changed and why
- Link related issues
- Keep PRs focused (one feature/fix per PR)
- Be responsive to review feedback

## Reporting Issues

- Search existing issues first
- Include reproduction steps
- Specify your environment (OS, versions)
- For security issues, email directly instead of opening a public issue

## Questions?

Open a discussion on GitHub if you need help or want to propose something.
