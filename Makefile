PROJECT_NAME := observability-sandbox

SERVICES := services/user-service services/order-service services/api-gateway
MONITORING := monitoring/prometheus monitoring/grafana monitoring/jaeger
SHARED := shared

MODULES := $(SERVICES) $(MONITORING) shared

.PHONY: all clean setup init-services init-shared init-monitoring build test distclean

# Default workflow: safe init + build + test
all: build test

# Clean generated files and Go modules
clean:
	@echo "\n🧹 Cleaning project..."
	-rm -vf go.mod go.sum
	-rm -vf *.out *.test coverage*.txt
	-rm -rf bin build vendor
	@for d in $(MODULES); do \
		if [ -d $$d ]; then \
			echo " - Cleaning $$d"; \
			(cd $$d && rm -vf go.mod go.sum *.out *.test coverage*.txt && rm -rf bin build vendor); \
		fi \
	done

# Setup folder structure from scratch
setup:
	@echo "\n⚠️ Rebuilding project structure from scratch (this removes existing folders)..."
	-rm -Rvf services monitoring shared
	mkdir -p $(SERVICES)
	mkdir -p $(MONITORING)
	mkdir -p $(SHARED)
	@if [ ! -f go.mod ]; then go mod init $(PROJECT_NAME); else echo "go.mod already exists at root, skipping init"; fi
	@echo "\n📂 Project structure created:"
	@command -v tree >/dev/null 2>&1 && tree || echo "Install 'tree' to view project structure"

# Initialize workspace
workspace:
	go work init ./shared ./services/user-service ./services/order-service ./services/api-gateway

# Initialize services modules
init-services:
	@echo "\n⚙️ Initializing services..."
	@for d in $(SERVICES); do \
		echo "Initializing $$d"; \
		if [ ! -f $$d/go.mod ]; then \
			(cd $$d && go mod init $(notdir $$d)); \
		else \
			echo "$$d already initialized"; \
		fi; \
		(cd $$d && go get github.com/gin-gonic/gin \
			&& go get github.com/prometheus/client_golang/prometheus/promhttp \
			&& go get github.com/stretchr/testify/assert \
			&& go get github.com/stretchr/testify/mock \
			&& go mod tidy); \
	done

# Initialize shared module
init-shared:
	@echo "\n⚙️ Initializing shared..."
	@for d in $(SHARED); do \
		if [ ! -f $$d/go.mod ]; then \
			(cd $$d && go mod init shared); \
		else \
			echo "$$d already initialized"; \
		fi; \
		(cd $$d && go get github.com/gin-gonic/gin \
			&& go get github.com/stretchr/testify/assert \
			&& go get github.com/stretchr/testify/mock \
			&& go get github.com/prometheus/client_golang/prometheus \
			&& go get github.com/sirupsen/logrus \
			&& go get go.opentelemetry.io/otel \
			&& go get go.opentelemetry.io/otel/exporters/jaeger \
			&& go get go.opentelemetry.io/otel/sdk/trace \
			&& go get go.opentelemetry.io/otel/semconv/v1.4.0 \
			&& go mod tidy); \
	done

# Initialize monitoring modules
init-monitoring:
	@echo "\n⚙️ Initializing monitoring..."
	@for d in $(MONITORING); do \
		if [ ! -f $$d/go.mod ]; then \
			(cd $$d && go mod init $(notdir $$d)); \
		else \
			echo "$$d already initialized"; \
		fi; \
		(cd $$d && go mod tidy); \
	done

# Build requires all modules initialized
build: init-services init-shared init-monitoring
	@echo "\n🔨 Building all modules..."
	@for d in $(MODULES); do \
		if [ -d $$d ]; then \
			echo "▶️ Building $$d"; \
			(cd $$d && go build ./...); \
		fi \
	done

# Test requires build
test: build
	@echo "\n🧪 Running tests for all modules..."
	@for d in $(MODULES); do \
		if [ -d $$d ]; then \
			echo "▶️ Testing $$d"; \
			(cd $$d && go test ./... -race -cover); \
		fi \
	done

# Distclean: fully reset project (manual destructive target)
distclean:
	@echo "\n⚠️ distclean target is to be implemented. It will remove all folders, go modules, binaries, and generated files."
