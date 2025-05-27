# Stripe Payment Backend Makefile
# =================================

# Variables
APP_NAME := stripe-backend
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION := $(shell go version | cut -d " " -f 3)
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Docker variables
DOCKER_IMAGE := $(APP_NAME)
DOCKER_TAG := $(VERSION)
COMPOSE_FILE := docker-compose.yml

# Database variables
DB_HOST := localhost
DB_PORT := 5432
DB_USER := stripe_user
DB_PASSWORD := stripe_password123
DB_NAME := stripe_payments
DB_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Colors for output
CYAN := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
RESET := \033[0m
BOLD := \033[1m

# Default target
.DEFAULT_GOAL := help

# =================================
# 📋 HELP & INFO
# =================================

.PHONY: help
help: ## Show this help message
	@echo "$(BOLD)$(CYAN)$(APP_NAME) - Development Makefile$(RESET)"
	@echo "$(YELLOW)Version: $(VERSION)$(RESET)"
	@echo ""
	@echo "$(BOLD)Available commands:$(RESET)"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { \
		printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2 \
	}' $(MAKEFILE_LIST) | sort
	@echo ""
	@echo "$(BOLD)Examples:$(RESET)"
	@echo "  make dev          # Start development environment"
	@echo "  make test         # Run all tests"
	@echo "  make docker-up    # Start with Docker"
	@echo "  make deploy       # Deploy to production"

.PHONY: info
info: ## Show project information
	@echo "$(BOLD)$(CYAN)Project Information$(RESET)"
	@echo "$(YELLOW)App Name:$(RESET)     $(APP_NAME)"
	@echo "$(YELLOW)Version:$(RESET)      $(VERSION)"
	@echo "$(YELLOW)Build Time:$(RESET)   $(BUILD_TIME)"
	@echo "$(YELLOW)Go Version:$(RESET)   $(GO_VERSION)"
	@echo "$(YELLOW)Docker Image:$(RESET) $(DOCKER_IMAGE):$(DOCKER_TAG)"
	@echo "$(YELLOW)Database URL:$(RESET) $(DB_URL)"

# =================================
# 🔧 DEVELOPMENT
# =================================

.PHONY: setup
setup: ## Initial project setup
	@echo "$(CYAN)Setting up project...$(RESET)"
	@go mod download
	@go mod tidy
	@mkdir -p logs tmp data
	@cp .env.example .env || cp .env.docker .env || touch .env
	@echo "$(GREEN)✓ Project setup complete$(RESET)"
	@echo "$(YELLOW)⚠ Don't forget to edit .env with your configuration$(RESET)"

.PHONY: deps
deps: ## Download and verify dependencies
	@echo "$(CYAN)Downloading dependencies...$(RESET)"
	@go mod download
	@go mod verify
	@go mod tidy
	@echo "$(GREEN)✓ Dependencies updated$(RESET)"

.PHONY: build
build: ## Build the application
	@echo "$(CYAN)Building $(APP_NAME)...$(RESET)"
	@go build $(LDFLAGS) -o bin/$(APP_NAME) .
	@echo "$(GREEN)✓ Build complete: bin/$(APP_NAME)$(RESET)"

.PHONY: run
run: ## Run the application locally
	@echo "$(CYAN)Starting $(APP_NAME)...$(RESET)"
	@go run $(LDFLAGS) .

.PHONY: dev
dev: ## Start development server with hot reload
	@echo "$(CYAN)Starting development server...$(RESET)"
	@echo "$(YELLOW)💡 Install 'air' for hot reload: go install github.com/cosmtrek/air@latest$(RESET)"
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "$(YELLOW)⚠ Hot reload not available, using regular run$(RESET)"; \
		make run; \
	fi

.PHONY: watch
watch: dev ## Alias for dev (hot reload)

# =================================
# 🧪 TESTING & QUALITY
# =================================

.PHONY: test
test: ## Run all tests
	@echo "$(CYAN)Running tests...$(RESET)"
	@go test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "$(CYAN)Running tests with coverage...$(RESET)"
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report: coverage.html$(RESET)"

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "$(CYAN)Running integration tests...$(RESET)"
	@go test -v -tags=integration ./tests/...

.PHONY: benchmark
benchmark: ## Run benchmark tests
	@echo "$(CYAN)Running benchmarks...$(RESET)"
	@go test -bench=. -benchmem ./...

.PHONY: lint
lint: ## Run linters
	@echo "$(CYAN)Running linters...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "$(YELLOW)⚠ golangci-lint not found, using basic checks$(RESET)"; \
		go vet ./...; \
		go fmt ./...; \
	fi

.PHONY: fmt
fmt: ## Format code
	@echo "$(CYAN)Formatting code...$(RESET)"
	@go fmt ./...
	@echo "$(GREEN)✓ Code formatted$(RESET)"

.PHONY: vet
vet: ## Run go vet
	@echo "$(CYAN)Running go vet...$(RESET)"
	@go vet ./...

.PHONY: check
check: fmt vet lint test ## Run all quality checks

# =================================
# 🐳 DOCKER OPERATIONS
# =================================

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "$(CYAN)Building Docker image...$(RESET)"
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest
	@echo "$(GREEN)✓ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)$(RESET)"

.PHONY: docker-run
docker-run: ## Run application in Docker container
	@echo "$(CYAN)Running Docker container...$(RESET)"
	@docker run --rm -p 8080:8080 --env-file .env $(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-up
docker-up: ## Start all services with Docker Compose
	@echo "$(CYAN)Starting Docker services...$(RESET)"
	@docker-compose up -d
	@echo "$(GREEN)✓ Services started$(RESET)"
	@make docker-status

.PHONY: docker-down
docker-down: ## Stop all Docker services
	@echo "$(CYAN)Stopping Docker services...$(RESET)"
	@docker-compose down
	@echo "$(GREEN)✓ Services stopped$(RESET)"

.PHONY: docker-restart
docker-restart: docker-down docker-up ## Restart Docker services

.PHONY: docker-logs
docker-logs: ## Show Docker logs
	@docker-compose logs -f

.PHONY: docker-status
docker-status: ## Show Docker services status
	@echo "$(BOLD)$(CYAN)Docker Services Status:$(RESET)"
	@docker-compose ps
	@echo ""
	@echo "$(BOLD)$(CYAN)Service URLs:$(RESET)"
	@echo "  $(YELLOW)API:$(RESET)       http://localhost:8080"
	@echo "  $(YELLOW)Health:$(RESET)    http://localhost:8080/health"
	@echo "  $(YELLOW)pgAdmin:$(RESET)   http://localhost:5050"
	@echo "  $(YELLOW)Database:$(RESET)  postgresql://localhost:5432/$(DB_NAME)"

.PHONY: docker-shell
docker-shell: ## Open shell in running container
	@docker-compose exec $(APP_NAME) sh

.PHONY: docker-clean
docker-clean: ## Clean Docker resources
	@echo "$(CYAN)Cleaning Docker resources...$(RESET)"
	@docker-compose down -v --remove-orphans
	@docker system prune -f
	@echo "$(GREEN)✓ Docker cleanup complete$(RESET)"

# =================================
# 🗄️ DATABASE OPERATIONS
# =================================

.PHONY: db-create
db-create: ## Create database
	@echo "$(CYAN)Creating database...$(RESET)"
	@docker-compose exec postgres createdb -U $(DB_USER) $(DB_NAME) || true
	@echo "$(GREEN)✓ Database created$(RESET)"

.PHONY: db-drop
db-drop: ## Drop database
	@echo "$(RED)⚠ This will delete all data!$(RESET)"
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ]
	@docker-compose exec postgres dropdb -U $(DB_USER) $(DB_NAME) || true
	@echo "$(GREEN)✓ Database dropped$(RESET)"

.PHONY: db-reset
db-reset: db-drop db-create db-migrate ## Reset database (drop, create, migrate)

.PHONY: db-migrate
db-migrate: ## Run database migrations
	@echo "$(CYAN)Running database migrations...$(RESET)"
	@echo "$(YELLOW)💡 Migrations are auto-applied via init scripts$(RESET)"
	@docker-compose exec postgres psql -U $(DB_USER) -d $(DB_NAME) -f /docker-entrypoint-initdb.d/01-create-tables.sql
	@echo "$(GREEN)✓ Migrations complete$(RESET)"

.PHONY: db-shell
db-shell: ## Open database shell
	@echo "$(CYAN)Opening database shell...$(RESET)"
	@docker-compose exec postgres psql -U $(DB_USER) -d $(DB_NAME)

.PHONY: db-backup
db-backup: ## Backup database
	@echo "$(CYAN)Creating database backup...$(RESET)"
	@mkdir -p backups
	@docker-compose exec postgres pg_dump -U $(DB_USER) $(DB_NAME) > backups/backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "$(GREEN)✓ Backup created in backups/$(RESET)"

.PHONY: db-restore
db-restore: ## Restore database (requires BACKUP_FILE variable)
	@if [ -z "$(BACKUP_FILE)" ]; then \
		echo "$(RED)Error: BACKUP_FILE variable required$(RESET)"; \
		echo "Usage: make db-restore BACKUP_FILE=backups/backup_20231201_120000.sql"; \
		exit 1; \
	fi
	@echo "$(CYAN)Restoring database from $(BACKUP_FILE)...$(RESET)"
	@docker-compose exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) < $(BACKUP_FILE)
	@echo "$(GREEN)✓ Database restored$(RESET)"

# =================================
# 🚀 DEPLOYMENT
# =================================

.PHONY: deploy-dev
deploy-dev: docker-build docker-up ## Deploy to development environment
	@echo "$(GREEN)✓ Development deployment complete$(RESET)"

.PHONY: deploy-staging
deploy-staging: check docker-build ## Deploy to staging environment
	@echo "$(CYAN)Deploying to staging...$(RESET)"
	@echo "$(YELLOW)💡 Implement your staging deployment logic here$(RESET)"
	@echo "$(GREEN)✓ Staging deployment complete$(RESET)"

.PHONY: deploy-prod
deploy-prod: ## Deploy to production environment
	@echo "$(RED)⚠ Production deployment$(RESET)"
	@read -p "Are you sure you want to deploy to production? (y/N): " confirm && [ "$$confirm" = "y" ]
	@echo "$(CYAN)Deploying to production...$(RESET)"
	@make check
	@echo "$(YELLOW)💡 Implement your production deployment logic here$(RESET)"
	@echo "$(GREEN)✓ Production deployment complete$(RESET)"

.PHONY: deploy
deploy: deploy-dev ## Default deployment (development)

# =================================
# 🧹 UTILITIES
# =================================

.PHONY: clean
clean: ## Clean build artifacts and temporary files
	@echo "$(CYAN)Cleaning up...$(RESET)"
	@rm -rf bin/ dist/ tmp/ logs/*.log
	@rm -f coverage.out coverage.html
	@go clean -cache -testcache -modcache
	@echo "$(GREEN)✓ Cleanup complete$(RESET)"

.PHONY: update
update: ## Update dependencies
	@echo "$(CYAN)Updating dependencies...$(RESET)"
	@go get -u ./...
	@go mod tidy
	@echo "$(GREEN)✓ Dependencies updated$(RESET)"

.PHONY: security
security: ## Run security checks
	@echo "$(CYAN)Running security checks...$(RESET)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "$(YELLOW)💡 Install gosec: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest$(RESET)"; \
	fi

.PHONY: docs
docs: ## Generate documentation
	@echo "$(CYAN)Generating documentation...$(RESET)"
	@go doc -all . > docs/api.md || echo "$(YELLOW)⚠ No docs generated$(RESET)"
	@echo "$(GREEN)✓ Documentation generated$(RESET)"

.PHONY: tools
tools: ## Install development tools
	@echo "$(CYAN)Installing development tools...$(RESET)"
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@echo "$(GREEN)✓ Development tools installed$(RESET)"

# =================================
# 🎯 SHORTCUTS & ALIASES
# =================================

.PHONY: up
up: docker-up ## Alias for docker-up

.PHONY: down
down: docker-down ## Alias for docker-down

.PHONY: logs
logs: docker-logs ## Alias for docker-logs

.PHONY: status
status: docker-status ## Alias for docker-status

.PHONY: shell
shell: docker-shell ## Alias for docker-shell

.PHONY: db
db: db-shell ## Alias for db-shell

# =================================
# 🔄 CI/CD TARGETS
# =================================

.PHONY: ci
ci: deps check test ## CI pipeline
	@echo "$(GREEN)✓ CI pipeline complete$(RESET)"

.PHONY: release
release: ## Create a new release
	@if [ -z "$(TAG)" ]; then \
		echo "$(RED)Error: TAG variable required$(RESET)"; \
		echo "Usage: make release TAG=v1.0.0"; \
		exit 1; \
	fi
	@echo "$(CYAN)Creating release $(TAG)...$(RESET)"
	@git tag $(TAG)
	@git push origin $(TAG)
	@echo "$(GREEN)✓ Release $(TAG) created$(RESET)"

# =================================
# 📝 FILE OPERATIONS
# =================================

# Create .env from template if it doesn't exist
.env:
	@echo "$(YELLOW)Creating .env file from template...$(RESET)"
	@cp .env.example .env 2>/dev/null || cp .env.docker .env 2>/dev/null || touch .env
	@echo "$(GREEN)✓ .env file created$(RESET)"
	@echo "$(YELLOW)⚠ Please edit .env with your configuration$(RESET)"

# Create necessary directories
directories:
	@mkdir -p bin logs tmp data backups

# =================================
# 🎨 ADVANCED FEATURES
# =================================

.PHONY: load-test
load-test: ## Run load tests
	@echo "$(CYAN)Running load tests...$(RESET)"
	@if command -v hey >/dev/null 2>&1; then \
		hey -n 1000 -c 10 http://localhost:8080/health; \
	else \
		echo "$(YELLOW)💡 Install hey: go install github.com/rakyll/hey@latest$(RESET)"; \
	fi

.PHONY: monitor
monitor: ## Monitor application (requires running services)
	@echo "$(CYAN)Monitoring application...$(RESET)"
	@echo "$(YELLOW)Press Ctrl+C to stop monitoring$(RESET)"
	@while true; do \
		echo "$(CYAN)[$(shell date)] Health Check:$(RESET)"; \
		curl -s http://localhost:8080/health | jq . || echo "Service unavailable"; \
		sleep 5; \
	done

# Keep intermediate files
.PRECIOUS: %.go

# Prevent make from trying to build files that match target names
.PHONY: help info setup deps build run dev watch test test-coverage test-integration \
        benchmark lint fmt vet check docker-build docker-run docker-up docker-down \
        docker-restart docker-logs docker-status docker-shell docker-clean \
        db-create db-drop db-reset db-migrate db-shell db-backup db-restore \
        deploy-dev deploy-staging deploy-prod deploy clean update security docs tools \
        up down logs status shell db ci release load-test monitor