.PHONY: build test test-race lint coverage bench clean fmt tidy help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOLINT=golangci-lint

# Packages
PKGS=./pkg/...
BENCH_PKG=./benchmark/...
TEST_PKG=./pkg/... ./benchmark/...

# Build
build:
	$(GOBUILD) ./...

# Format code
fmt:
	$(GOFMT) ./pkg/...

# Tidy modules
tidy:
	$(GOMOD) tidy

# Unit tests
test:
	$(GOTEST) -count=1 $(TEST_PKG)

# Unit tests with race detector
test-race:
	$(GOTEST) -count=1 -race -timeout 120s $(TEST_PKG)

# Integration tests (requires -tags=integration)
test-integration:
	$(GOTEST) -tags=integration -count=1 -timeout 300s ./tests/...

# Fuzz tests (60s per target)
test-fuzz:
	$(GOTEST) -fuzz=FuzzLimiterKey -fuzztime=60s ./pkg/ruleflow/core/
	$(GOTEST) -fuzz=FuzzEvalWithRandomData -fuzztime=60s ./pkg/ruleflow/core/
	$(GOTEST) -fuzz=FuzzPanicRecovery -fuzztime=60s ./pkg/ruleflow/core/

# Lint
lint:
	$(GOLINT) run --timeout=10m ./pkg/...

# Coverage
coverage:
	$(GOTEST) -coverprofile=coverage.out -covermode=atomic $(PKGS)
	$(GOCMD) tool cover -func=coverage.out | tail -1

# Coverage HTML report
coverage-html:
	$(GOTEST) -coverprofile=coverage.out -covermode=atomic $(PKGS)
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Benchmarks
bench:
	$(GOTEST) -bench=. -benchtime=1s -benchmem -run=^$ $(BENCH_PKG)

# Clean
clean:
	rm -f coverage.out coverage.html
	rm -f bench.txt

# Help
help:
	@echo "RuleFlow Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build        Build all packages"
	@echo "  make test         Run unit tests"
	@echo "  make test-race    Run tests with race detector"
	@echo "  make test-integration Run integration tests"
	@echo "  make test-fuzz    Run fuzz tests"
	@echo "  make lint         Run golangci-lint"
	@echo "  make coverage     Show coverage summary"
	@echo "  make coverage-html Generate HTML coverage report"
	@echo "  make bench        Run benchmarks"
	@echo "  make fmt          Format code"
	@echo "  make tidy         Tidy go modules"
	@echo "  make clean        Clean generated files"