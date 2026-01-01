# Me.yaml Makefile
# Common commands for development and deployment

.PHONY: help dev dev-up dev-down dev-logs dev-reset build test clean docker-build docker-run seed-demo seed-dev seed-clear

# Default target
help:
	@echo "Me.yaml — You, human-readable."
	@echo ""
	@echo "Development (Codespaces/Local):"
	@echo "  make dev          Start dev environment with hot reload"
	@echo "  make dev-docker   Start dev environment via Docker Compose"
	@echo "  make dev-down     Stop Docker Compose services"
	@echo "  make dev-logs     View Docker Compose logs"
	@echo "  make dev-reset    Clear caches and force reinstall"
	@echo ""
	@echo "Seed Data (switch demo profiles):"
	@echo "  make seed-demo    Reset & start with Merlin (fun wizard demo)"
	@echo "  make seed-dev     Reset & start with Jedidiah (real-world dev)"
	@echo "  make seed-clear   Clear database only (no restart)"
	@echo ""
	@echo "Individual Services:"
	@echo "  make backend      Start backend only (with hot reload)"
	@echo "  make frontend     Start frontend only (with HMR)"
	@echo ""
	@echo "Testing & Quality:"
	@echo "  make test         Run all tests"
	@echo "  make lint         Run linters"
	@echo "  make fmt          Format code"
	@echo ""
	@echo "Production:"
	@echo "  make build        Build production Docker image"
	@echo "  make prod         Start production containers"
	@echo "  make prod-down    Stop production containers"
	@echo ""

# =============================================================================
# Development
# =============================================================================

# Main dev command - uses optimized scripts
dev:
	./scripts/start-dev.sh

dev-up: dev

# Individual services
backend:
	./scripts/dev-backend.sh

frontend:
	./scripts/dev-frontend.sh

# Docker-based dev (alternative to native)
dev-docker:
	docker compose -f docker-compose.dev.yml up

dev-down:
	docker compose -f docker-compose.dev.yml down

dev-logs:
	docker compose -f docker-compose.dev.yml logs -f

# Reset caches to force reinstall
dev-reset:
	rm -rf frontend/node_modules/.lockfile-hash
	rm -rf backend/.gomod-hash
	rm -rf pb_data
	rm -rf tmp
	rm -rf frontend/.svelte-kit
	@echo "Caches cleared. Run 'make dev' to reinstall."

# =============================================================================
# Seed Data Switching
# =============================================================================

# Switch to demo profile (Merlin Ambrosius - fun Arthurian wizard)
seed-demo:
	@echo "Switching to demo seed (Merlin Ambrosius)..."
	rm -rf pb_data
	SEED_DATA=demo ./scripts/start-dev.sh

# Switch to dev profile (Jedidiah Esposito - real-world example)
seed-dev:
	@echo "Switching to dev seed (Jedidiah Esposito)..."
	rm -rf pb_data
	SEED_DATA=dev ./scripts/start-dev.sh

# Just clear database (no restart)
seed-clear:
	rm -rf pb_data
	@echo "Database cleared. Run 'make dev', 'make seed-demo', or 'make seed-dev' to restart."

# =============================================================================
# Building
# =============================================================================

build: docker-build

docker-build:
	docker build -t me-yaml:latest -f docker/Dockerfile .

docker-run:
	docker run -d \
		--name me-yaml \
		-p 8080:3000 \
		-p 8090:8090 \
		-v $$(pwd)/data:/data \
		-e ENCRYPTION_KEY=$${ENCRYPTION_KEY:-dev-key-change-me-in-production} \
		me-yaml:latest

# =============================================================================
# Testing
# =============================================================================

test: test-backend test-frontend

test-backend:
	cd backend && go test -v ./...

test-frontend:
	cd frontend && npm run check

# =============================================================================
# Linting & Formatting
# =============================================================================

lint: lint-backend lint-frontend

lint-backend:
	cd backend && golangci-lint run || true

lint-frontend:
	cd frontend && npm run lint || true

fmt: fmt-backend fmt-frontend

fmt-backend:
	cd backend && gofmt -w .

fmt-frontend:
	cd frontend && npm run format

# =============================================================================
# Production
# =============================================================================

prod:
	docker compose up -d

prod-logs:
	docker compose logs -f

prod-down:
	docker compose down

# =============================================================================
# Utilities
# =============================================================================

# Clean build artifacts (not caches)
clean:
	rm -rf backend/tmp
	rm -rf frontend/build
	rm -rf frontend/.svelte-kit
	rm -rf tmp

# Backup data
backup:
	@mkdir -p backups
	tar -czvf backups/me-yaml-$$(date +%Y%m%d-%H%M%S).tar.gz pb_data/ data/ 2>/dev/null || true

# Install dependencies (manual, usually handled by scripts)
deps:
	cd backend && go mod download
	cd frontend && npm install

# Generate a secure encryption key
gen-key:
	@openssl rand -hex 32
