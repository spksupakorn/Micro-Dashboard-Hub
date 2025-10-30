.PHONY: help build up down logs clean restart test

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build all Docker images
	docker-compose build

up: ## Start all services
	docker-compose up -d
	@echo "Services starting..."
	@echo "AuthService:    http://localhost:8080"
	@echo "ProfileService: http://localhost:8081"
	@echo "BillingService: http://localhost:8082"

down: ## Stop all services
	docker-compose down

logs: ## Show logs from all services
	docker-compose logs -f

logs-auth: ## Show logs from AuthService
	docker-compose logs -f auth_service

logs-profile: ## Show logs from ProfileService
	docker-compose logs -f profile_service

logs-billing: ## Show logs from BillingService
	docker-compose logs -f billing_service

logs-db: ## Show logs from PostgreSQL
	docker-compose logs -f postgres

clean: ## Stop and remove all containers, volumes, and images
	docker-compose down -v --rmi all

restart: ## Restart all services
	docker-compose restart

rebuild: ## Rebuild and restart all services
	docker-compose down
	docker-compose up --build -d

ps: ## Show running containers
	docker-compose ps

test: ## Run SSO test flow
	@echo "Testing SSO Flow..."
	@echo "1. Opening ProfileService (should redirect to login)..."
	@open http://localhost:8081/dashboard || xdg-open http://localhost:8081/dashboard 2>/dev/null
	@echo "2. After login, try BillingService (should auto-login)..."
	@sleep 5
	@open http://localhost:8082/invoices || xdg-open http://localhost:8082/invoices 2>/dev/null

db-shell: ## Access PostgreSQL shell
	docker-compose exec postgres psql -U sso_user -d sso_db

status: ## Check service health
	@echo "Checking service status..."
	@curl -s -o /dev/null -w "AuthService:    %{http_code}\n" http://localhost:8080 || echo "AuthService:    DOWN"
	@curl -s -o /dev/null -w "ProfileService: %{http_code}\n" http://localhost:8081 || echo "ProfileService: DOWN"
	@curl -s -o /dev/null -w "BillingService: %{http_code}\n" http://localhost:8082 || echo "BillingService: DOWN"
