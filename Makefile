# me.yaml Makefile
# Common commands for development and deployment

.PHONY: help dev build test clean docker-build docker-run

# Default target
help:
	@echo "me.yaml - A self-hosted, human-readable profile and portfolio"
	@echo ""
	@echo "Usage:"
	@echo "  make dev          Start development environment"
	@echo "  make build        Build production images"
	@echo "  make test         Run tests"
	@echo "  make clean        Clean build artifacts"
	@echo "  make docker-build Build Docker image"
	@echo "  make docker-run   Run Docker container"
	@echo ""
	@echo "Development:"
	@echo "  make backend      Start backend only"
	@echo "  make frontend     Start frontend only"
	@echo "  make lint         Run linters"
	@echo "  make fmt          Format code"
	@echo ""

# Development
dev:
	docker-compose -f docker-compose.dev.yml up

dev-down:
	docker-compose -f docker-compose.dev.yml down

backend:
	cd backend && go run .

frontend:
	cd frontend && npm run dev

# Build
build: docker-build

docker-build:
	docker build -t me-yaml:latest -f docker/Dockerfile .

docker-run:
	docker run -d \
		--name me-yaml \
		-p 8080:3000 \
		-p 8090:8090 \
		-v ./data:/data \
		-e ENCRYPTION_KEY=$${ENCRYPTION_KEY:-dev-key-change-me-in-production} \
		me-yaml:latest

# Testing
test: test-backend test-frontend

test-backend:
	cd backend && go test ./...

test-frontend:
	cd frontend && npm run check

# Linting
lint: lint-backend lint-frontend

lint-backend:
	cd backend && golangci-lint run

lint-frontend:
	cd frontend && npm run lint

# Formatting
fmt: fmt-backend fmt-frontend

fmt-backend:
	cd backend && gofmt -w .

fmt-frontend:
	cd frontend && npm run format

# Clean
clean:
	rm -rf backend/tmp
	rm -rf frontend/build
	rm -rf frontend/.svelte-kit
	rm -rf frontend/node_modules/.cache

# Production
prod:
	docker-compose up -d

prod-logs:
	docker-compose logs -f

prod-down:
	docker-compose down

# Backup
backup:
	@mkdir -p backups
	tar -czvf backups/me-yaml-$$(date +%Y%m%d-%H%M%S).tar.gz data/

# Install dependencies
deps:
	cd backend && go mod download
	cd frontend && npm install

# Generate encryption key
gen-key:
	@openssl rand -hex 32
