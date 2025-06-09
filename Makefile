.PHONY: \
  dev stop status clean help swagger \
  seed reset-db fresh \
  docker-build docker-up docker-down docker-logs docker-logs-service \
  docker-status docker-restart docker-restart-service docker-clean \
  docker-rebuild docker-dev docker-fresh

# ---------------------------------------------------------------------
# Shared settings
# ---------------------------------------------------------------------
DOCKER := docker compose          # v2 CLI everywhere

# ---------------------------------------------------------------------
# Local development (runs binaries directly)
# ---------------------------------------------------------------------
dev:
	@echo "🚀 Starting services locally..."
	@mkdir -p .pids
	@go run api-gateway/main.go     & echo $$! > .pids/api-gateway.pid
	@sleep 1
	@go run auth-service/main.go    & echo $$! > .pids/auth-service.pid
	@sleep 1
	@go run permission-service/main.go & echo $$! > .pids/permission-service.pid
	@sleep 1
	@go run core-service/main.go    & echo $$! > .pids/core-service.pid
	@sleep 1
	@go run notification-service/main.go & echo $$! > .pids/notification-service.pid
	@sleep 1
	@go run document-service/main.go & echo $$! > .pids/document-service.pid
	@echo "✅ All local services started"; @wait

stop:
	@echo "🛑 Stopping local services..."
	@pkill -f "go run.*main.go" 2>/dev/null || true
	@sleep 2
	@for port in 8000 8001 8002 8003 8004 8005; do \
		pid=$$(lsof -ti :$$port 2>/dev/null); \
		[ -n "$$pid" ] && echo "Killing port $$port (PID $$pid)" && kill -9 $$pid; \
	done
	@rm -rf .pids 2>/dev/null || true
	@echo "✅ Local services stopped"

status:
	@echo "🔍 Health checks:"
	@for svc in 8000 8001 8002 8003 8004 8005; do \
		printf "Port %s: " $$svc; \
		curl -fsS http://localhost:$$svc/health >/dev/null && echo "✅" || echo "❌"; \
	done

# ---------------------------------------------------------------------
# Local DB helpers (direct Go run)
# ---------------------------------------------------------------------
seed:
	@echo "🌱 Seeding DB locally...";    go run cmd/seed/main.go
reset-db:
	@echo "🗑  Resetting DB locally..."; go run cmd/reset-db/main.go
fresh: reset-db seed

# ---------------------------------------------------------------------
# Swagger docs
# ---------------------------------------------------------------------
swagger:
	@echo "📝 Generating Swagger docs..."; chmod +x scripts/generate_swagger.sh && ./scripts/generate_swagger.sh

# ---------------------------------------------------------------------
# Housekeeping
# ---------------------------------------------------------------------
clean:
	@rm -rf .pids; echo "🧹 Temp files removed"

# ---------------------------------------------------------------------
# Docker workflow
# ---------------------------------------------------------------------
docker-build:
	@echo "🏗  Building Docker images..."; $(DOCKER) build; echo "✅ Build done"

docker-up:
	@echo "🚀 Bringing stack up..."; $(DOCKER) up -d; \
		echo "✅ Stack running → http://localhost:8000 (API Gateway)"

docker-down:
	@echo "🛑 Stopping stack..."; $(DOCKER) down; echo "✅ Stack stopped"

docker-logs:
	@$(DOCKER) logs -f

docker-logs-service:
	@$(DOCKER) logs -f $(SERVICE)

docker-status:
	@$(DOCKER) ps

docker-restart:
	@$(DOCKER) restart

docker-restart-service:
	@$(DOCKER) restart $(SERVICE)

docker-clean:
	@echo "🧹 Removing containers, volumes, networks..."; \
	$(DOCKER) down -v && docker system prune -f

docker-rebuild:
	@$(DOCKER) down && $(DOCKER) build --no-cache && $(DOCKER) up -d

docker-dev:
	@$(DOCKER) up -d

# ---------------------------------------------------------------------
# Docker DB seeding (uses db-seed service)
# ---------------------------------------------------------------------
docker-fresh:
	@echo "🌱 Seeding DB inside Docker..."
	@$(DOCKER) run --rm db-seed
	@echo "✅ Docker DB seeded"