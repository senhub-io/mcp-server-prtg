EXECUTABLE=mcp-server-prtg
BUILD_DIR=build
WINDOWS=$(BUILD_DIR)/$(EXECUTABLE)_windows_amd64.exe
LINUX_AMD64=$(BUILD_DIR)/$(EXECUTABLE)_linux_amd64
LINUX_ARM64=$(BUILD_DIR)/$(EXECUTABLE)_linux_arm64
DARWIN_AMD64=$(BUILD_DIR)/$(EXECUTABLE)_darwin_amd64
DARWIN_ARM64=$(BUILD_DIR)/$(EXECUTABLE)_darwin_arm64
VERSION=$(shell git describe --tags --always --abbrev=0 2>/dev/null || echo "v1.0.0")
COMMIT_HASH=$(shell git describe --tags --always --long --dirty 2>/dev/null || echo "unknown")

# Package to set version variable (use 'main' for main package)
PACKAGE=main

BUILD_TIME=$(shell date +%FT%T%z)
GO_VERSION=$(shell go version | cut -d' ' -f3)
COVERAGE_FILE=coverage.out

# Colors for display
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
BLUE=\033[0;34m
NC=\033[0m # No Color

# LDFLAGS enriched with build information
LDFLAGS=-s -w \
	-X '${PACKAGE}.Version=$(VERSION)' \
	-X '${PACKAGE}.CommitHash=$(COMMIT_HASH)' \
	-X '${PACKAGE}.BuildTime=$(BUILD_TIME)' \
	-X '${PACKAGE}.GoVersion=$(GO_VERSION)'

# ========================================
# VERSION MANAGEMENT
# ========================================

version-info: ## Display version information
	@echo "$(BLUE)ℹ️  Version information:$(NC)"
	@echo "  Version:    $(VERSION)"
	@echo "  Commit:     $(COMMIT_HASH)"
	@echo "  Build time: $(BUILD_TIME)"
	@echo "  Go version: $(GO_VERSION)"

check-version: ## Verify that version is defined
	@if [ "$(VERSION)" = "" ]; then \
		echo "$(RED)❌ ERROR: No version tag found$(NC)" >&2; \
		exit 1; \
	fi
	@echo "$(GREEN)✅ Version: $(VERSION)$(NC)"

# Manual version management (for development and RC)
bump-version: ## Create a new version tag
	@current_version=$$(git tag -l | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+' | sort -V | tail -n1 | sed 's/v//'); \
	if [ -z "$$current_version" ]; then \
		current_version="1.0.0"; \
	fi; \
	echo "$(YELLOW)Current version: v$$current_version$(NC)"; \
	read -p "Enter new version [$$current_version]: " new_version; \
	: "$${new_version:=$$current_version}"; \
	if [[ ! "$$new_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-rc[0-9]+)?$$ ]]; then \
		echo "$(RED)❌ Invalid version format. Use X.Y.Z or X.Y.Z-rcN$(NC)"; \
		exit 1; \
	fi; \
	echo "$(GREEN)Creating new version: v$$new_version$(NC)"; \
	git tag -a "v$$new_version" -m "Version $$new_version"; \
	echo "$(GREEN)✅ Tag created. Push with: git push origin v$$new_version$(NC)"

# Delete a version tag (useful for corrections)
delete-version: ## Delete a version tag
	@echo "$(YELLOW)Current tags:$(NC)"; \
	git tag -l | grep -E '^v[0-9]+'; \
	read -p "Enter tag to delete (without v): " version_to_delete; \
	git tag -d "v$$version_to_delete"; \
	echo "$(GREEN)✅ Tag deleted locally. Push deletion with: git push origin :refs/tags/v$$version_to_delete$(NC)"

# ========================================
# BUILD TARGETS
# ========================================

# Create build directory
create-build-dir:
	@mkdir -p $(BUILD_DIR)

# Default build (current OS)
build: create-build-dir ## Build for current OS
	@echo "$(GREEN)🔨 Building $(EXECUTABLE) for current OS...$(NC)"
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(EXECUTABLE) ./cmd/server
	@echo "$(GREEN)✅ Build complete: $(BUILD_DIR)/$(EXECUTABLE)$(NC)"
	@echo "$(BLUE)Version: $(VERSION) - Commit: $(COMMIT_HASH)$(NC)"

# Build all platforms
build-all: create-build-dir build-windows build-linux build-darwin ## Build for all platforms
	@echo "$(GREEN)✅ All builds complete!$(NC)"
	@ls -lh $(BUILD_DIR)

build-windows: create-build-dir ## Build for Windows
	@echo "$(GREEN)🪟 Building for Windows...$(NC)"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(WINDOWS) ./cmd/server
	@echo "$(GREEN)✅ Windows build complete$(NC)"

build-linux: create-build-dir ## Build for Linux (amd64 + arm64)
	@echo "$(GREEN)🐧 Building for Linux amd64...$(NC)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(LINUX_AMD64) ./cmd/server
	@echo "$(GREEN)🐧 Building for Linux arm64...$(NC)"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(LINUX_ARM64) ./cmd/server
	@echo "$(GREEN)✅ Linux builds complete$(NC)"

build-darwin: create-build-dir ## Build for macOS (amd64 + arm64)
	@echo "$(GREEN)🍎 Building for macOS amd64...$(NC)"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(DARWIN_AMD64) ./cmd/server
	@echo "$(GREEN)🍎 Building for macOS arm64...$(NC)"
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(DARWIN_ARM64) ./cmd/server
	@echo "$(GREEN)✅ macOS builds complete$(NC)"

# ========================================
# PACKAGING TARGETS
# ========================================

package: build-all ## Create ZIP archives for all platforms
	@echo "$(GREEN)📦 Creating ZIP packages...$(NC)"
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_windows_amd64.zip $(EXECUTABLE)_windows_amd64.exe
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_linux_amd64.zip $(EXECUTABLE)_linux_amd64
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_linux_arm64.zip $(EXECUTABLE)_linux_arm64
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_darwin_amd64.zip $(EXECUTABLE)_darwin_amd64
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_darwin_arm64.zip $(EXECUTABLE)_darwin_arm64
	@echo "$(GREEN)✅ ZIP packages created in $(BUILD_DIR)/$(NC)"
	@ls -lh $(BUILD_DIR)/*.zip

package-windows: build-windows ## Create ZIP archive for Windows
	@echo "$(GREEN)📦 Creating Windows ZIP package...$(NC)"
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_windows_amd64.zip $(EXECUTABLE)_windows_amd64.exe
	@echo "$(GREEN)✅ Windows ZIP package created$(NC)"

package-linux: build-linux ## Create ZIP archives for Linux
	@echo "$(GREEN)📦 Creating Linux ZIP packages...$(NC)"
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_linux_amd64.zip $(EXECUTABLE)_linux_amd64
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_linux_arm64.zip $(EXECUTABLE)_linux_arm64
	@echo "$(GREEN)✅ Linux ZIP packages created$(NC)"

package-darwin: build-darwin ## Create ZIP archives for macOS
	@echo "$(GREEN)📦 Creating macOS ZIP packages...$(NC)"
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_darwin_amd64.zip $(EXECUTABLE)_darwin_amd64
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_darwin_arm64.zip $(EXECUTABLE)_darwin_arm64
	@echo "$(GREEN)✅ macOS ZIP packages created$(NC)"

# ========================================
# DEVELOPMENT TARGETS
# ========================================

run: ## Run server in development mode
	@echo "$(BLUE)🚀 Running MCP server...$(NC)"
	@echo "$(YELLOW)⚠️  Make sure PRTG_DB_PASSWORD is set$(NC)"
	@go run ./cmd/server

# Run with test configuration
run-dev: ## Run with development environment variables
	@echo "$(BLUE)🚀 Running MCP server (dev mode)...$(NC)"
	LOG_LEVEL=debug go run ./cmd/server

# ========================================
# TESTING & QUALITY TARGETS
# ========================================

test: ## Run tests
	@echo "$(GREEN)🧪 Running tests...$(NC)"
	@go test ./... -v
	@echo "$(GREEN)✅ Tests passed$(NC)"

test-race: ## Tests with race condition detection
	@echo "$(GREEN)🏃‍♂️ Tests with race condition detection...$(NC)"
	@go test -race -v ./...
	@echo "$(GREEN)✅ Race tests complete$(NC)"

benchmark: ## Performance benchmarks
	@echo "$(GREEN)⚡ Running benchmarks...$(NC)"
	@go test -bench=. -benchmem ./...
	@echo "$(GREEN)✅ Benchmarks complete$(NC)"

coverage: ## Test coverage report
	@echo "$(GREEN)📊 Generating coverage report...$(NC)"
	@go test -coverprofile=$(COVERAGE_FILE) ./...
	@go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "$(GREEN)✅ Report generated: coverage.html$(NC)"
	@echo "$(YELLOW)📈 Coverage summary:$(NC)"
	@go tool cover -func=$(COVERAGE_FILE) | tail -1

lint: ## Code quality analysis (golangci-lint)
	@echo "$(GREEN)🔍 Code quality analysis...$(NC)"
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "$(RED)❌ golangci-lint not installed. Run 'make install-tools'$(NC)"; \
		exit 1; \
	}
	@golangci-lint run --timeout=5m
	@echo "$(GREEN)✅ Lint analysis complete$(NC)"

lint-fix: ## Automatically fix style issues
	@echo "$(GREEN)🔧 Automatically fixing issues...$(NC)"
	@go fmt ./...
	@go mod tidy
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run --fix --timeout=5m || echo "$(YELLOW)⚠️  golangci-lint not available$(NC)"
	@echo "$(GREEN)✅ Fixes applied$(NC)"

fmt: ## Format code
	@echo "$(GREEN)🎨 Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✅ Code formatted$(NC)"

vet: ## Run go vet
	@echo "$(GREEN)🔍 Running go vet...$(NC)"
	@go vet ./...
	@echo "$(GREEN)✅ go vet passed$(NC)"

security: ## Security audit (govulncheck + gosec)
	@echo "$(GREEN)🛡️  Security audit...$(NC)"
	@command -v govulncheck >/dev/null 2>&1 || { \
		echo "$(RED)❌ govulncheck not installed. Run 'make install-tools'$(NC)"; \
		exit 1; \
	}
	@if command -v gosec >/dev/null 2>&1; then \
		echo "$(YELLOW)🔒 gosec analysis...$(NC)"; \
		gosec -quiet ./...; \
	else \
		echo "$(YELLOW)⚠️  gosec not installed - skipped$(NC)"; \
	fi
	@echo "$(YELLOW)🔍 Vulnerability check...$(NC)"
	@govulncheck ./...
	@echo "$(GREEN)✅ Security audit complete$(NC)"

install-tools: ## Install all development tools
	@echo "$(GREEN)📦 Installing development tools...$(NC)"
	@echo "$(YELLOW)Installing golangci-lint...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(YELLOW)Installing govulncheck...$(NC)"
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "$(YELLOW)Installing staticcheck...$(NC)"
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "$(YELLOW)Note: For gosec, install manually: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest$(NC)"
	@echo "$(GREEN)✅ Core tools installed$(NC)"

# ========================================
# WORKFLOW TARGETS
# ========================================

pre-commit: fmt vet lint-fix test ## Pre-commit checks
	@echo "$(GREEN)✅ Ready for commit!$(NC)"

quality-check: fmt vet lint test security ## Complete quality check
	@echo ""
	@echo "$(GREEN)═══════════════════════════════════════$(NC)"
	@echo "$(GREEN)🎉 QUALITY CHECKS PASSED! 🎉$(NC)"
	@echo "$(GREEN)═══════════════════════════════════════$(NC)"
	@echo "$(GREEN)✅ Formatting: OK$(NC)"
	@echo "$(GREEN)✅ go vet: OK$(NC)"
	@echo "$(GREEN)✅ Code quality (lint): OK$(NC)"
	@echo "$(GREEN)✅ Tests: OK$(NC)"
	@echo "$(GREEN)✅ Security: OK$(NC)"
	@echo ""

release: quality-check build-all package ## Prepare complete release
	@echo ""
	@echo "$(GREEN)═══════════════════════════════════════$(NC)"
	@echo "$(GREEN)🚀 RELEASE READY! 🚀$(NC)"
	@echo "$(GREEN)═══════════════════════════════════════$(NC)"
	@echo "$(BLUE)Version: $(VERSION)$(NC)"
	@echo "$(BLUE)Commit:  $(COMMIT_HASH)$(NC)"
	@echo "$(BLUE)Binaries and packages in: $(BUILD_DIR)/$(NC)"
	@ls -lh $(BUILD_DIR)
	@echo ""
	@echo "$(YELLOW)To publish:$(NC)"
	@echo "  1. git push origin $(VERSION)"
	@echo "  2. Create GitHub release with ZIPs"
	@echo ""

# ========================================
# UTILITY TARGETS
# ========================================

deps: ## Download dependencies
	@echo "$(GREEN)📦 Downloading dependencies...$(NC)"
	@go mod download
	@go mod tidy
	@echo "$(GREEN)✅ Dependencies updated$(NC)"

verify: fmt vet lint test ## Verify code (fmt, vet, lint, test)
	@echo "$(GREEN)✅ All verifications passed!$(NC)"

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE)
	@rm -f coverage.html
	@go clean -testcache
	@echo "$(GREEN)✅ Clean complete$(NC)"

install: build ## Install binary to $GOPATH/bin
	@echo "$(GREEN)📥 Installing $(EXECUTABLE)...$(NC)"
	@mkdir -p $(GOPATH)/bin
	@cp $(BUILD_DIR)/$(EXECUTABLE) $(GOPATH)/bin/
	@echo "$(GREEN)✅ Installed to $(GOPATH)/bin/$(EXECUTABLE)$(NC)"

help: ## Display this help
	@echo "$(GREEN)═══════════════════════════════════════════════════════════$(NC)"
	@echo "$(GREEN)         MCP Server PRTG - Available Commands$(NC)"
	@echo "$(GREEN)═══════════════════════════════════════════════════════════$(NC)"
	@echo ""
	@echo "$(YELLOW)🔨 Build & Deploy:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(build|package|install|run|create)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🧪 Tests & Quality:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(test|lint|security|coverage|benchmark|fmt|vet)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🔄 Workflows:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(pre-commit|quality-check|release|verify)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🏷️  Version:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(version|bump|delete)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🛠️  Tools:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(install-tools|deps|clean|help)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""

.PHONY: all build build-all build-windows build-linux build-darwin package package-windows package-linux package-darwin \
	run run-dev test test-race benchmark coverage lint lint-fix fmt vet security install-tools \
	pre-commit quality-check release verify deps clean install create-build-dir \
	version-info check-version bump-version delete-version help

.DEFAULT_GOAL := help
