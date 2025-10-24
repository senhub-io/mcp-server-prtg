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

# Couleurs pour l'affichage
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
BLUE=\033[0;34m
NC=\033[0m # No Color

# LDFLAGS enrichis avec informations de build
LDFLAGS=-s -w \
	-X '${PACKAGE}.Version=$(VERSION)' \
	-X '${PACKAGE}.CommitHash=$(COMMIT_HASH)' \
	-X '${PACKAGE}.BuildTime=$(BUILD_TIME)' \
	-X '${PACKAGE}.GoVersion=$(GO_VERSION)'

# ========================================
# VERSION MANAGEMENT
# ========================================

version-info: ## Affiche les informations de version
	@echo "$(BLUE)ℹ️  Version information:$(NC)"
	@echo "  Version:    $(VERSION)"
	@echo "  Commit:     $(COMMIT_HASH)"
	@echo "  Build time: $(BUILD_TIME)"
	@echo "  Go version: $(GO_VERSION)"

check-version: ## Vérifie que la version est définie
	@if [ "$(VERSION)" = "" ]; then \
		echo "$(RED)❌ ERROR: No version tag found$(NC)" >&2; \
		exit 1; \
	fi
	@echo "$(GREEN)✅ Version: $(VERSION)$(NC)"

# Gestion manuelle des versions (pour développement et RC)
bump-version: ## Créer une nouvelle version tag
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

# Supprimer un tag de version (utile pour corrections)
delete-version: ## Supprimer un tag de version
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

# Build par défaut (OS courant)
build: create-build-dir ## Build pour l'OS courant
	@echo "$(GREEN)🔨 Building $(EXECUTABLE) for current OS...$(NC)"
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(EXECUTABLE) ./cmd/server
	@echo "$(GREEN)✅ Build complete: $(BUILD_DIR)/$(EXECUTABLE)$(NC)"
	@echo "$(BLUE)Version: $(VERSION) - Commit: $(COMMIT_HASH)$(NC)"

# Build toutes les plateformes
build-all: create-build-dir build-windows build-linux build-darwin ## Build pour toutes les plateformes
	@echo "$(GREEN)✅ All builds complete!$(NC)"
	@ls -lh $(BUILD_DIR)

build-windows: create-build-dir ## Build pour Windows
	@echo "$(GREEN)🪟 Building for Windows...$(NC)"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(WINDOWS) ./cmd/server
	@echo "$(GREEN)✅ Windows build complete$(NC)"

build-linux: create-build-dir ## Build pour Linux (amd64 + arm64)
	@echo "$(GREEN)🐧 Building for Linux amd64...$(NC)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(LINUX_AMD64) ./cmd/server
	@echo "$(GREEN)🐧 Building for Linux arm64...$(NC)"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(LINUX_ARM64) ./cmd/server
	@echo "$(GREEN)✅ Linux builds complete$(NC)"

build-darwin: create-build-dir ## Build pour macOS (amd64 + arm64)
	@echo "$(GREEN)🍎 Building for macOS amd64...$(NC)"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(DARWIN_AMD64) ./cmd/server
	@echo "$(GREEN)🍎 Building for macOS arm64...$(NC)"
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o $(DARWIN_ARM64) ./cmd/server
	@echo "$(GREEN)✅ macOS builds complete$(NC)"

# ========================================
# PACKAGING TARGETS
# ========================================

package: build-all ## Créer des archives ZIP pour toutes les plateformes
	@echo "$(GREEN)📦 Creating ZIP packages...$(NC)"
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_windows_amd64.zip $(EXECUTABLE)_windows_amd64.exe
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_linux_amd64.zip $(EXECUTABLE)_linux_amd64
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_linux_arm64.zip $(EXECUTABLE)_linux_arm64
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_darwin_amd64.zip $(EXECUTABLE)_darwin_amd64
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_darwin_arm64.zip $(EXECUTABLE)_darwin_arm64
	@echo "$(GREEN)✅ ZIP packages created in $(BUILD_DIR)/$(NC)"
	@ls -lh $(BUILD_DIR)/*.zip

package-windows: build-windows ## Créer archive ZIP pour Windows
	@echo "$(GREEN)📦 Creating Windows ZIP package...$(NC)"
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_windows_amd64.zip $(EXECUTABLE)_windows_amd64.exe
	@echo "$(GREEN)✅ Windows ZIP package created$(NC)"

package-linux: build-linux ## Créer archives ZIP pour Linux
	@echo "$(GREEN)📦 Creating Linux ZIP packages...$(NC)"
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_linux_amd64.zip $(EXECUTABLE)_linux_amd64
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_linux_arm64.zip $(EXECUTABLE)_linux_arm64
	@echo "$(GREEN)✅ Linux ZIP packages created$(NC)"

package-darwin: build-darwin ## Créer archives ZIP pour macOS
	@echo "$(GREEN)📦 Creating macOS ZIP packages...$(NC)"
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_darwin_amd64.zip $(EXECUTABLE)_darwin_amd64
	@cd $(BUILD_DIR) && zip -9 $(EXECUTABLE)_darwin_arm64.zip $(EXECUTABLE)_darwin_arm64
	@echo "$(GREEN)✅ macOS ZIP packages created$(NC)"

# ========================================
# DEVELOPMENT TARGETS
# ========================================

run: ## Exécuter le serveur en mode développement
	@echo "$(BLUE)🚀 Running MCP server...$(NC)"
	@echo "$(YELLOW)⚠️  Assurez-vous que PRTG_DB_PASSWORD est défini$(NC)"
	@go run ./cmd/server

# Exécuter avec configuration de test
run-dev: ## Exécuter avec variables d'environnement de dev
	@echo "$(BLUE)🚀 Running MCP server (dev mode)...$(NC)"
	LOG_LEVEL=debug go run ./cmd/server

# ========================================
# TESTING & QUALITY TARGETS
# ========================================

test: ## Exécuter les tests
	@echo "$(GREEN)🧪 Running tests...$(NC)"
	@go test ./... -v
	@echo "$(GREEN)✅ Tests passed$(NC)"

test-race: ## Tests avec détection de race conditions
	@echo "$(GREEN)🏃‍♂️ Tests avec détection de race conditions...$(NC)"
	@go test -race -v ./...
	@echo "$(GREEN)✅ Tests race terminés$(NC)"

benchmark: ## Tests de performance
	@echo "$(GREEN)⚡ Running benchmarks...$(NC)"
	@go test -bench=. -benchmem ./...
	@echo "$(GREEN)✅ Benchmarks terminés$(NC)"

coverage: ## Rapport de couverture de tests
	@echo "$(GREEN)📊 Génération du rapport de couverture...$(NC)"
	@go test -coverprofile=$(COVERAGE_FILE) ./...
	@go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "$(GREEN)✅ Rapport généré: coverage.html$(NC)"
	@echo "$(YELLOW)📈 Résumé de la couverture:$(NC)"
	@go tool cover -func=$(COVERAGE_FILE) | tail -1

lint: ## Analyse de qualité du code (golangci-lint)
	@echo "$(GREEN)🔍 Analyse de qualité du code...$(NC)"
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "$(RED)❌ golangci-lint non installé. Exécutez 'make install-tools'$(NC)"; \
		exit 1; \
	}
	@golangci-lint run --timeout=5m
	@echo "$(GREEN)✅ Analyse lint terminée$(NC)"

lint-fix: ## Corrige automatiquement les problèmes de style
	@echo "$(GREEN)🔧 Correction automatique des problèmes...$(NC)"
	@go fmt ./...
	@go mod tidy
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run --fix --timeout=5m || echo "$(YELLOW)⚠️  golangci-lint non disponible$(NC)"
	@echo "$(GREEN)✅ Corrections appliquées$(NC)"

fmt: ## Formater le code
	@echo "$(GREEN)🎨 Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✅ Code formatted$(NC)"

vet: ## Exécuter go vet
	@echo "$(GREEN)🔍 Running go vet...$(NC)"
	@go vet ./...
	@echo "$(GREEN)✅ go vet passed$(NC)"

security: ## Audit de sécurité (govulncheck + gosec)
	@echo "$(GREEN)🛡️  Audit de sécurité...$(NC)"
	@command -v govulncheck >/dev/null 2>&1 || { \
		echo "$(RED)❌ govulncheck non installé. Exécutez 'make install-tools'$(NC)"; \
		exit 1; \
	}
	@if command -v gosec >/dev/null 2>&1; then \
		echo "$(YELLOW)🔒 Analyse gosec...$(NC)"; \
		gosec -quiet ./...; \
	else \
		echo "$(YELLOW)⚠️  gosec non installé - ignoré$(NC)"; \
	fi
	@echo "$(YELLOW)🔍 Vérification des vulnérabilités...$(NC)"
	@govulncheck ./...
	@echo "$(GREEN)✅ Audit de sécurité terminé$(NC)"

install-tools: ## Installer tous les outils de développement
	@echo "$(GREEN)📦 Installation des outils de développement...$(NC)"
	@echo "$(YELLOW)Installing golangci-lint...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(YELLOW)Installing govulncheck...$(NC)"
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "$(YELLOW)Installing staticcheck...$(NC)"
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "$(YELLOW)Note: Pour gosec, installez manuellement: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest$(NC)"
	@echo "$(GREEN)✅ Outils de base installés$(NC)"

# ========================================
# WORKFLOW TARGETS
# ========================================

pre-commit: fmt vet lint-fix test ## Vérifications avant commit
	@echo "$(GREEN)✅ Prêt pour le commit !$(NC)"

quality-check: fmt vet lint test security ## Vérification complète de qualité
	@echo ""
	@echo "$(GREEN)═══════════════════════════════════════$(NC)"
	@echo "$(GREEN)🎉 CONTRÔLES DE QUALITÉ RÉUSSIS ! 🎉$(NC)"
	@echo "$(GREEN)═══════════════════════════════════════$(NC)"
	@echo "$(GREEN)✅ Formatage: OK$(NC)"
	@echo "$(GREEN)✅ go vet: OK$(NC)"
	@echo "$(GREEN)✅ Qualité du code (lint): OK$(NC)"
	@echo "$(GREEN)✅ Tests: OK$(NC)"
	@echo "$(GREEN)✅ Sécurité: OK$(NC)"
	@echo ""

release: quality-check build-all package ## Préparer une release complète
	@echo ""
	@echo "$(GREEN)═══════════════════════════════════════$(NC)"
	@echo "$(GREEN)🚀 RELEASE PRÊTE ! 🚀$(NC)"
	@echo "$(GREEN)═══════════════════════════════════════$(NC)"
	@echo "$(BLUE)Version: $(VERSION)$(NC)"
	@echo "$(BLUE)Commit:  $(COMMIT_HASH)$(NC)"
	@echo "$(BLUE)Binaires et packages dans: $(BUILD_DIR)/$(NC)"
	@ls -lh $(BUILD_DIR)
	@echo ""
	@echo "$(YELLOW)Pour publier:$(NC)"
	@echo "  1. git push origin $(VERSION)"
	@echo "  2. Créer une release GitHub avec les ZIPs"
	@echo ""

# ========================================
# UTILITY TARGETS
# ========================================

deps: ## Télécharger les dépendances
	@echo "$(GREEN)📦 Downloading dependencies...$(NC)"
	@go mod download
	@go mod tidy
	@echo "$(GREEN)✅ Dependencies updated$(NC)"

verify: fmt vet lint test ## Vérifier le code (fmt, vet, lint, test)
	@echo "$(GREEN)✅ All verifications passed!$(NC)"

clean: ## Nettoyer les artefacts de build
	@echo "$(YELLOW)🧹 Cleaning...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE)
	@rm -f coverage.html
	@go clean -testcache
	@echo "$(GREEN)✅ Clean complete$(NC)"

install: build ## Installer le binaire dans $GOPATH/bin
	@echo "$(GREEN)📥 Installing $(EXECUTABLE)...$(NC)"
	@mkdir -p $(GOPATH)/bin
	@cp $(BUILD_DIR)/$(EXECUTABLE) $(GOPATH)/bin/
	@echo "$(GREEN)✅ Installed to $(GOPATH)/bin/$(EXECUTABLE)$(NC)"

help: ## Afficher cette aide
	@echo "$(GREEN)═══════════════════════════════════════════════════════════$(NC)"
	@echo "$(GREEN)         MCP Server PRTG - Commandes disponibles$(NC)"
	@echo "$(GREEN)═══════════════════════════════════════════════════════════$(NC)"
	@echo ""
	@echo "$(YELLOW)🔨 Build & Deploy:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(build|package|install|run|create)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🧪 Tests & Qualité:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(test|lint|security|coverage|benchmark|fmt|vet)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🔄 Workflows:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(pre-commit|quality-check|release|verify)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🏷️  Version:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(version|bump|delete)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🛠️  Outils:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(install-tools|deps|clean|help)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""

.PHONY: all build build-all build-windows build-linux build-darwin package package-windows package-linux package-darwin \
	run run-dev test test-race benchmark coverage lint lint-fix fmt vet security install-tools \
	pre-commit quality-check release verify deps clean install create-build-dir \
	version-info check-version bump-version delete-version help

.DEFAULT_GOAL := help
