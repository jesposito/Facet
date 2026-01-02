# Facet Makefile
# Common commands for development and deployment

SHELL := /bin/bash

.PHONY: help dev dev-up dev-down dev-logs dev-reset build test clean docker-build docker-run seed-dev seed-clear kill stop

# Default target
help:
	@echo "Facet — Every side of you. Your way."
	@echo ""
	@echo "Development (Codespaces/Local):"
	@echo "  make dev          Start dev environment with hot reload"
	@echo "  make dev-docker   Start dev environment via Docker Compose"
	@echo "  make dev-down     Stop Docker Compose services"
	@echo "  make dev-logs     View Docker Compose logs"
	@echo "  make dev-reset    Clear caches and force reinstall"
	@echo ""
	@echo "Seed Data (development only):"
	@echo "  make seed-dev     Stop, reset & start with dev data (Jedidiah)"
	@echo "  make seed-clear   Stop & clear database (no restart)"
	@echo "  make stop         Stop all running dev processes"
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

# Kill running dev processes
kill:
	@echo "Stopping dev processes..."
	@-pkill -9 -f "air" 2>/dev/null || true
	@-pkill -9 -f "facet" 2>/dev/null || true
	@-pkill -9 -f "pocketbase" 2>/dev/null || true
	@-pkill -9 -f "vite" 2>/dev/null || true
	@-pkill -9 -f "node.*5173" 2>/dev/null || true
	@-pkill -9 -f "node.*5174" 2>/dev/null || true
	@-fuser -k 8090/tcp 2>/dev/null || true
	@-fuser -k 5173/tcp 2>/dev/null || true
	@-fuser -k 5174/tcp 2>/dev/null || true
	@sleep 2
	@echo "Done."

stop: kill

# Switch to dev profile (Jedidiah Esposito - for development/testing)
# Demo data (Merlin) is available via Admin > Settings > Demo Data
seed-dev: kill
	@echo "Switching to dev seed (Jedidiah Esposito)..."
	rm -rf pb_data
	@LOCAL_DEFAULT="http://localhost:8080"; \
	CODESPACES_DEFAULT=$${CODESPACE_NAME:+https://$${CODESPACE_NAME}-8080.app.github.dev}; \
	if [ -f .env ]; then set -a; source .env; set +a; fi; \
	# Step 1: choose auth mode (password-only, Google, GitHub, both) \
	AUTH_DEFAULT=1; \
	if [ -n "$$GOOGLE_CLIENT_ID" ] && [ -n "$$GOOGLE_CLIENT_SECRET" ] && [ -n "$$GITHUB_CLIENT_ID" ] && [ -n "$$GITHUB_CLIENT_SECRET" ]; then AUTH_DEFAULT=4; \
	elif [ -n "$$GOOGLE_CLIENT_ID" ] && [ -n "$$GOOGLE_CLIENT_SECRET" ]; then AUTH_DEFAULT=2; \
	elif [ -n "$$GITHUB_CLIENT_ID" ] && [ -n "$$GITHUB_CLIENT_SECRET" ]; then AUTH_DEFAULT=3; fi; \
	echo "Select auth setup:"; \
	echo "  1) Password only"; \
	echo "  2) Google"; \
	echo "  3) GitHub"; \
	echo "  4) Google + GitHub"; \
	read -r -p "Choice [$${AUTH_DEFAULT}]: " AUTH_CHOICE_INPUT; \
	AUTH_CHOICE=$${AUTH_CHOICE_INPUT:-$$AUTH_DEFAULT}; \
	GOOGLE_ENABLE=false; GITHUB_ENABLE=false; \
	case "$$AUTH_CHOICE" in \
	  2) GOOGLE_ENABLE=true ;; \
	  3) GITHUB_ENABLE=true ;; \
	  4) GOOGLE_ENABLE=true; GITHUB_ENABLE=true ;; \
	  *) AUTH_CHOICE=1 ;; \
	esac; \
	# Step 2: select APP_URL (default Codespaces if present, else localhost) \
	BASE_DEFAULT=$${APP_URL:-$${CODESPACES_DEFAULT:-$$LOCAL_DEFAULT}}; \
	if [ -n "$$CODESPACES_DEFAULT" ]; then \
		echo "Detected Codespaces. Choose APP_URL:"; \
		echo "  1) $$CODESPACES_DEFAULT (default)"; \
		echo "  2) $$LOCAL_DEFAULT (use if connecting via localhost tunnel)"; \
		read -r -p "Select [1/2] or enter custom [$$BASE_DEFAULT]: " APP_URL_CHOICE; \
		if [ "$$APP_URL_CHOICE" = "2" ]; then APP_URL_VALUE="$$LOCAL_DEFAULT"; \
		elif [ "$$APP_URL_CHOICE" = "1" ] || [ -z "$$APP_URL_CHOICE" ]; then APP_URL_VALUE="$$CODESPACES_DEFAULT"; \
		else APP_URL_VALUE="$$APP_URL_CHOICE"; fi; \
	else \
		read -r -p "APP_URL [$$BASE_DEFAULT]: " APP_URL_INPUT; \
		APP_URL_VALUE=$${APP_URL_INPUT:-$$BASE_DEFAULT}; \
	fi; \
	# Step 3: admin email \
	ADMIN_DEFAULT=$${ADMIN_EMAILS:-admin@example.com}; \
	read -r -p "Admin email for seed [$$ADMIN_DEFAULT]: " ADMIN_INPUT; \
	ADMIN_VALUE=$${ADMIN_INPUT:-$$ADMIN_DEFAULT}; \
	# Step 4: provider creds (skip prompts if already present) \
	GOOGLE_ID_INPUT="$$GOOGLE_CLIENT_ID"; GOOGLE_SECRET_INPUT="$$GOOGLE_CLIENT_SECRET"; \
	if [ "$$GOOGLE_ENABLE" = true ]; then \
		if [ -n "$$GOOGLE_ID_INPUT" ] && [ -n "$$GOOGLE_SECRET_INPUT" ]; then \
			echo "Google credentials found in .env (reusing)."; \
		else \
			read -r -p "Google Client ID (optional): " GOOGLE_ID_INPUT; \
			if [ -n "$$GOOGLE_ID_INPUT" ]; then read -r -s -p "Google Client Secret: " GOOGLE_SECRET_INPUT; echo ""; fi; \
		fi; \
	fi; \
	GITHUB_ID_INPUT="$$GITHUB_CLIENT_ID"; GITHUB_SECRET_INPUT="$$GITHUB_CLIENT_SECRET"; \
	if [ "$$GITHUB_ENABLE" = true ]; then \
		if [ -n "$$GITHUB_ID_INPUT" ] && [ -n "$$GITHUB_SECRET_INPUT" ]; then \
			echo "GitHub credentials found in .env (reusing)."; \
		else \
			read -r -p "GitHub Client ID (optional): " GITHUB_ID_INPUT; \
			if [ -n "$$GITHUB_ID_INPUT" ]; then read -r -s -p "GitHub Client Secret: " GITHUB_SECRET_INPUT; echo ""; fi; \
		fi; \
	fi; \
	if [ "$$GOOGLE_ENABLE" = true ] && { [ -z "$$GOOGLE_ID_INPUT" ] || [ -z "$$GOOGLE_SECRET_INPUT" ]; }; then \
		echo "Google credentials missing; Google auth will be disabled for this run."; \
		GOOGLE_ENABLE=false; \
	fi; \
	if [ "$$GITHUB_ENABLE" = true ] && { [ -z "$$GITHUB_ID_INPUT" ] || [ -z "$$GITHUB_SECRET_INPUT" ]; }; then \
		echo "GitHub credentials missing; GitHub auth will be disabled for this run."; \
		GITHUB_ENABLE=false; \
	fi; \
	# Build env for start-dev and persist to .env \
	VARS="APP_URL=$$APP_URL_VALUE ADMIN_EMAILS=$$ADMIN_VALUE SEED_DATA=dev"; \
	{ \
	  echo "APP_URL=$$APP_URL_VALUE"; \
	  echo "ADMIN_EMAILS=$$ADMIN_VALUE"; \
	  if [ "$$GOOGLE_ENABLE" = true ] && [ -n "$$GOOGLE_ID_INPUT" ] && [ -n "$$GOOGLE_SECRET_INPUT" ]; then \
	    VARS="GOOGLE_CLIENT_ID=$$GOOGLE_ID_INPUT GOOGLE_CLIENT_SECRET=$$GOOGLE_SECRET_INPUT $$VARS"; \
	    echo "GOOGLE_CLIENT_ID=$$GOOGLE_ID_INPUT"; \
	    echo "GOOGLE_CLIENT_SECRET=$$GOOGLE_SECRET_INPUT"; \
	  fi; \
	  if [ "$$GITHUB_ENABLE" = true ] && [ -n "$$GITHUB_ID_INPUT" ] && [ -n "$$GITHUB_SECRET_INPUT" ]; then \
	    VARS="GITHUB_CLIENT_ID=$$GITHUB_ID_INPUT GITHUB_CLIENT_SECRET=$$GITHUB_SECRET_INPUT $$VARS"; \
	    echo "GITHUB_CLIENT_ID=$$GITHUB_ID_INPUT"; \
	    echo "GITHUB_CLIENT_SECRET=$$GITHUB_SECRET_INPUT"; \
	  fi; \
	  echo "VITE_POCKETBASE_URL=http://localhost:8090"; \
	} > .env; \
	echo "Using APP_URL=$$APP_URL_VALUE"; \
	echo "Admin email: $$ADMIN_VALUE"; \
	echo "Google enabled: $$GOOGLE_ENABLE"; \
	echo "GitHub enabled: $$GITHUB_ENABLE"; \
	sh -c "$$VARS ./scripts/start-dev.sh"

# Just clear database (no restart)
seed-clear: kill
	rm -rf pb_data
	@echo "Database cleared. Run 'make dev', 'make seed-demo', or 'make seed-dev' to restart."

# =============================================================================
# Building
# =============================================================================

build: docker-build

docker-build:
	docker build -t facet:latest -f docker/Dockerfile .

docker-run:
	docker run -d \
		--name facet \
		-p 8080:3000 \
		-p 8090:8090 \
		-v $$(pwd)/data:/data \
		-e ENCRYPTION_KEY=$${ENCRYPTION_KEY:-dev-key-change-me-in-production} \
		facet:latest

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
	tar -czvf backups/facet-$$(date +%Y%m%d-%H%M%S).tar.gz pb_data/ data/ 2>/dev/null || true

# Install dependencies (manual, usually handled by scripts)
deps:
	cd backend && go mod download
	cd frontend && npm install

# Generate a secure encryption key
gen-key:
	@openssl rand -hex 32
