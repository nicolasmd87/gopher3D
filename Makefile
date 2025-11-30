.PHONY: all build test lint vet staticcheck fieldalignment clean run

# Default target
all: lint test build

# Build the editor
build:
	@echo "Building editor..."
	go build -o bin/editor.exe ./editor/cmd/...
	@echo "Build complete: bin/editor.exe"

# Run the editor
run: build
	@echo "Running editor..."
	cd bin && ./editor.exe

# Run all tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run go vet for static analysis
vet:
	@echo "Running go vet..."
	go vet ./...

# Run staticcheck if installed
staticcheck:
	@echo "Running staticcheck..."
	@which staticcheck > /dev/null 2>&1 || (echo "Installing staticcheck..." && go install honnef.co/go/tools/cmd/staticcheck@latest)
	staticcheck ./...

# Check struct field alignment for performance
fieldalignment:
	@echo "Checking struct field alignment..."
	@which fieldalignment > /dev/null 2>&1 || (echo "Installing fieldalignment..." && go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest)
	fieldalignment -fix ./... 2>/dev/null || fieldalignment ./... 2>&1 | head -50

# Run all linters
lint: vet
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null 2>&1 || (echo "golangci-lint not found, running vet only" && exit 0)
	golangci-lint run ./... || true

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f editor/cmd/cmd
	rm -f editor/cmd/cmd.exe

# Install development tools
tools:
	@echo "Installing development tools..."
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
	@echo "Tools installed"

# Check memory alignment issues (detailed report)
alignment-report:
	@echo "Generating alignment report..."
	@echo "=== Struct Alignment Analysis ===" > alignment_report.txt
	@fieldalignment ./... 2>&1 >> alignment_report.txt || true
	@echo "Report saved to alignment_report.txt"
	@cat alignment_report.txt

# Performance profiling build
profile-build:
	@echo "Building with profiling..."
	go build -gcflags="-m -m" -o bin/editor_profile.exe ./editor/cmd/... 2>&1 | head -100

# Check for race conditions
race:
	@echo "Building with race detector..."
	go build -race -o bin/editor_race.exe ./editor/cmd/...
	@echo "Race detection build: bin/editor_race.exe"

# Full CI check
ci: fmt tidy vet test build
	@echo "CI checks passed"

# Help
help:
	@echo "Available targets:"
	@echo "  all              - Run lint, test, and build"
	@echo "  build            - Build the editor"
	@echo "  run              - Build and run the editor"
	@echo "  test             - Run all tests"
	@echo "  vet              - Run go vet"
	@echo "  staticcheck      - Run staticcheck"
	@echo "  fieldalignment   - Check struct field alignment"
	@echo "  lint             - Run all linters"
	@echo "  fmt              - Format code"
	@echo "  tidy             - Tidy go.mod"
	@echo "  clean            - Remove build artifacts"
	@echo "  tools            - Install development tools"
	@echo "  alignment-report - Generate detailed alignment report"
	@echo "  profile-build    - Build with escape analysis"
	@echo "  race             - Build with race detector"
	@echo "  ci               - Run full CI checks"

