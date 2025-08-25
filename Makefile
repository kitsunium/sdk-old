# Default target when just running 'make'
.DEFAULT_GOAL := help

# ==================================================================================== #
# VARIABLES
# ==================================================================================== #

PROJECT_NAME := kitsunium-sdk
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Tools
BAZEL := bazel
GO := go
GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint
GAZELLE := gazelle
PRETTIER := prettier

# Paths
PKG_PATH := ./pkg/...
ALL_GO_FILES := $(shell find . -name "*.go" -type f)

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
MAGENTA := \033[0;35m
CYAN := \033[0;36m
NC := \033[0m # No Color

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo ''
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Development:'
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "build" "build all packages with Bazel"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "test" "run all tests with Bazel"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "test/unit" "run unit tests only"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "test/coverage" "run tests with coverage report"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "deps" "download and verify dependencies"
	@echo ''
	@echo 'Benchmarks:'
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "bench" "run and save benchmark results"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "bench/update" "download latest benchmark database"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "bench/compare" "compare two benchmark commits"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "bench/list" "list saved benchmark results"
	@echo ''
	@echo 'Quality:'
	@printf "  $(BLUE)%-20s$(NC) %s\n" "quality/analyze" "run complete code analysis"
	@printf "  $(BLUE)%-20s$(NC) %s\n" "quality/format" "format all code (Go, YAML, JSON, MD)"
	@printf "  $(BLUE)%-20s$(NC) %s\n" "quality/lint" "run linters"
	@printf "  $(BLUE)%-20s$(NC) %s\n" "quality/security" "run security analysis"
	@printf "  $(BLUE)%-20s$(NC) %s\n" "quality/fix" "automatically fix all fixable issues"
	@printf "  $(BLUE)%-20s$(NC) %s\n" "quality/validate" "validate all quality gates pass"
	@echo ''
	@echo 'Git Hooks:'
	@printf "  $(GREEN)%-20s$(NC) %s\n" "hooks/install" "install Git hooks for automatic checks"
	@printf "  $(GREEN)%-20s$(NC) %s\n" "hooks/uninstall" "remove Git hooks configuration"
	@printf "  $(GREEN)%-20s$(NC) %s\n" "hooks/status" "check Git hooks configuration status"
	@echo ''
	@echo 'Bazel:'
	@printf "  $(MAGENTA)%-20s$(NC) %s\n" "bazel/update" "update BUILD files with Gazelle"
	@printf "  $(MAGENTA)%-20s$(NC) %s\n" "bazel/clean" "clean Bazel cache"
	@printf "  $(MAGENTA)%-20s$(NC) %s\n" "bazel/info" "show Bazel workspace info"
	@echo ''
	@echo 'Operations:'
	@printf "  $(CYAN)%-20s$(NC) %s\n" "clean" "clean build artifacts and caches"
	@printf "  $(CYAN)%-20s$(NC) %s\n" "version" "show version information"
	@echo ''
	@echo 'Tools:'
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "tools/check" "check if required tools are installed"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "tools/install" "install optional development tools"
	@echo ''
	@echo 'CI/CD:'
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "ci" "run all CI checks (quality, test, build)"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "ci/validate" "validate CI pipeline locally"
	@echo ''

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## build: build all packages with Bazel
.PHONY: build
build:
	@echo "$(YELLOW)▶ Building all packages...$(NC)"
	@$(BAZEL) build //...

## test: run all tests with Bazel
.PHONY: test
test:
	@echo "$(YELLOW)▶ Running tests...$(NC)"
	@$(BAZEL) test //... --test_output=errors

## test/unit: run unit tests only
.PHONY: test/unit
test/unit:
	@echo "$(YELLOW)▶ Running unit tests...$(NC)"
	@$(BAZEL) test //... --test_output=errors --test_size_filters=small,medium

## test/coverage: run tests with coverage report
.PHONY: test/coverage
test/coverage:
	@echo "$(YELLOW)▶ Running tests with coverage...$(NC)"
	@$(BAZEL) coverage //... --combined_report=lcov
	@echo "$(GREEN)✓ Coverage report generated$(NC)"

## bench: run benchmarks and save results to SQLite database
# Usage:
#   make bench                   - run benchmarks for current commit
#   make bench <hash>            - checkout commit and run benchmarks
.PHONY: bench
bench:
	@if [ ! -f benchmarks.sqlite ]; then \
		echo "$(YELLOW)▶ No local benchmark database found, downloading from BENCH release...$(NC)"; \
		$(MAKE) bench/update; \
	fi
	@if [ -n "$(filter-out $@,$(MAKECMDGOALS))" ]; then \
		COMMIT="$(filter-out $@,$(MAKECMDGOALS))"; \
		echo "$(YELLOW)▶ Checking out commit $$COMMIT...$(NC)"; \
		STASH_OUTPUT=$$(git stash push -m "bench: auto-stash before checkout" --include-untracked 2>&1); \
		STASHED=$$?; \
		git checkout $$COMMIT || exit 1; \
		echo "$(YELLOW)▶ Running benchmarks for commit $$COMMIT...$(NC)"; \
		python3 scripts/bench_manager.py save; \
		git checkout - >/dev/null 2>&1; \
		if [ $$STASHED -eq 0 ] && echo "$$STASH_OUTPUT" | grep -q "Saved"; then \
			git stash pop --quiet 2>/dev/null || echo "$(YELLOW)Note: Could not restore stashed changes$(NC)"; \
		fi; \
	else \
		echo "$(YELLOW)▶ Running benchmarks and saving results...$(NC)"; \
		python3 scripts/bench_manager.py save; \
	fi
	@echo "$(GREEN)✓ Benchmark results saved$(NC)"


## bench/save: save benchmark results (CI mode - preserves history)
.PHONY: bench/save
bench/save:
	@echo "$(YELLOW)▶ Running benchmarks and saving results (preserving history)...$(NC)"
	@python3 scripts/bench_manager.py save --preserve-history
	@echo "$(GREEN)✓ Benchmark results saved with history preserved$(NC)"

## bench/update: fetch benchmark database from BENCH release
.PHONY: bench/update
bench/update:
	@echo "$(YELLOW)▶ Fetching benchmark database from BENCH release...$(NC)"
	@RELEASE_INFO=$$(curl -s https://api.github.com/repos/kitsunium/sdk/releases/tags/BENCH); \
	if echo "$$RELEASE_INFO" | grep -q '"tag_name".*"BENCH"'; then \
		echo "$(CYAN)→ BENCH release found, downloading benchmarks.sqlite$(NC)"; \
		ASSET_URL=$$(echo "$$RELEASE_INFO" | grep -o '"browser_download_url".*benchmarks.sqlite"' | cut -d'"' -f4); \
		if [ -n "$$ASSET_URL" ]; then \
			curl -sL "$$ASSET_URL" -o benchmarks.sqlite && \
			echo "$(GREEN)✓ Benchmark database downloaded$(NC)" || \
			echo "$(RED)❌ Failed to download benchmark database$(NC)"; \
		else \
			curl -sL https://github.com/kitsunium/sdk/releases/download/BENCH/benchmarks.sqlite -o benchmarks.sqlite && \
			echo "$(GREEN)✓ Benchmark database downloaded$(NC)" || \
			echo "$(RED)❌ Failed to download benchmark database$(NC)"; \
		fi; \
	else \
		echo "$(YELLOW)⚠ BENCH release not found, starting with empty database$(NC)"; \
	fi

## bench/compare: compare benchmark results
# Usage:
#   make bench/compare                    - compare current with main
#   make bench/compare <commit>           - compare current with <commit>
#   make bench/compare <commit1> <commit2> - compare <commit1> with <commit2>
.PHONY: bench/compare
bench/compare:
	@echo "$(YELLOW)▶ Comparing benchmarks...$(NC)"
	@args="$(filter-out $@,$(MAKECMDGOALS))"; \
	num_args=$$(echo "$$args" | tr -s ' ' | wc -w | tr -d ' '); \
	if [ -z "$$args" ] || [ "$$num_args" = "0" ]; then \
		echo "$(CYAN)→ Comparing current commit with main branch$(NC)"; \
		python3 scripts/bench_manager.py compare; \
	elif [ "$$num_args" = "1" ]; then \
		echo "$(CYAN)→ Comparing current commit with $$args$(NC)"; \
		python3 scripts/bench_manager.py compare $$args; \
	elif [ "$$num_args" = "2" ]; then \
		arg1=$$(echo $$args | cut -d' ' -f1); \
		arg2=$$(echo $$args | cut -d' ' -f2); \
		echo "$(CYAN)→ Comparing $$arg1 with $$arg2$(NC)"; \
		python3 scripts/bench_manager.py compare $$arg1 $$arg2; \
	else \
		echo "$(RED)❌ Invalid arguments. Usage:$(NC)"; \
		echo "  make bench/compare                    - compare current with main"; \
		echo "  make bench/compare <commit>           - compare current with <commit>"; \
		echo "  make bench/compare <commit1> <commit2> - compare <commit1> with <commit2>"; \
		exit 1; \
	fi

## bench/list: list all saved benchmark results
.PHONY: bench/list
bench/list:
	@python3 scripts/bench_manager.py list

# Catch-all rule for positional arguments to bench and bench/compare
%:
	@:

## deps: download and verify dependencies
.PHONY: deps
deps:
	@echo "$(YELLOW)▶ Downloading dependencies...$(NC)"
	@$(GO) mod download
	@$(GO) mod verify
	@echo "$(GREEN)✓ Dependencies verified$(NC)"

# ==================================================================================== #
# QUALITY
# ==================================================================================== #

## quality/analyze: run complete code analysis
.PHONY: quality/analyze
quality/analyze: quality/lint quality/security
	@echo "$(GREEN)✓ Code analysis complete$(NC)"


## quality/format: format all code (Go, YAML, JSON, MD)
.PHONY: quality/format
quality/format:
	@echo "$(YELLOW)▶ Formatting code...$(NC)"
	@echo "  Formatting Go files..."
	@gofmt -w -s $(shell find . -name "*.go" -not -path "./vendor/*" -not -path "./bazel-*/*")
	@if command -v goimports >/dev/null 2>&1; then \
		echo "  Running goimports..."; \
		goimports -w $(shell find . -name "*.go" -not -path "./vendor/*" -not -path "./bazel-*/*"); \
	fi
	@if command -v $(PRETTIER) >/dev/null 2>&1; then \
		$(PRETTIER) --write "**/*.{json,yaml,yml,md}" --ignore-path .prettierignore; \
	else \
		echo "$(YELLOW)⚠ Prettier not installed, skipping non-Go files$(NC)"; \
	fi
	@$(BAZEL) run //:gazelle
	@echo "$(GREEN)✓ Code formatted$(NC)"

# Alias for format
.PHONY: fmt
fmt: quality/format

## quality/lint: run linters
.PHONY: quality/lint
quality/lint:
	@echo "$(YELLOW)▶ Running linters...$(NC)"
	@if [ -f .golangci.yml ]; then \
		if command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
			$(GOLANGCI_LINT) run ./...; \
		else \
			echo "$(YELLOW)⚠ golangci-lint not installed, trying with go vet$(NC)"; \
			$(GO) vet ./...; \
		fi \
	else \
		$(GO) vet ./...; \
	fi
	@echo "$(GREEN)✓ Linting complete$(NC)"

## quality/security: run security analysis
.PHONY: quality/security
quality/security:
	@echo "$(YELLOW)▶ Running security checks...$(NC)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -config .gosec.json ./...; \
	else \
		echo "$(YELLOW)⚠ gosec not installed, install with: go install github.com/securego/gosec/v2/cmd/gosec@latest$(NC)"; \
	fi
	@echo "$(GREEN)✓ Security check complete$(NC)"

## quality/fix: automatically fix all fixable issues
.PHONY: quality/fix
quality/fix: quality/format
	@echo "$(YELLOW)▶ Fixing issues...$(NC)"
	@if command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
		$(GOLANGCI_LINT) run --fix ./...; \
	fi
	@echo "$(GREEN)✓ Issues fixed$(NC)"

## quality/validate: validate all quality gates pass
.PHONY: quality/validate
quality/validate: quality/lint test
	@echo "$(GREEN)✓ All quality gates passed$(NC)"

# ==================================================================================== #
# GIT HOOKS
# ==================================================================================== #

## hooks/install: install Git hooks for automatic checks
.PHONY: hooks/install
hooks/install:
	@echo "$(YELLOW)▶ Installing Git hooks...$(NC)"
	@chmod +x .githooks/*
	@git config core.hooksPath .githooks
	@echo "$(GREEN)✓ Git hooks installed$(NC)"
	@echo "$(CYAN)ℹ Hooks location: .githooks/$(NC)"

## hooks/uninstall: remove Git hooks configuration
.PHONY: hooks/uninstall
hooks/uninstall:
	@echo "$(YELLOW)▶ Uninstalling Git hooks...$(NC)"
	@git config --unset core.hooksPath
	@echo "$(GREEN)✓ Git hooks uninstalled$(NC)"

## hooks/status: check Git hooks configuration status
.PHONY: hooks/status
hooks/status:
	@echo "$(YELLOW)▶ Git hooks status:$(NC)"
	@if [ "$$(git config core.hooksPath)" = ".githooks" ]; then \
		echo "$(GREEN)✓ Hooks are installed and active$(NC)"; \
		echo "$(CYAN)  Location: .githooks/$(NC)"; \
		ls -la .githooks/ | grep -E "pre-commit|commit-msg|pre-push" | while read line; do \
			echo "$(CYAN)  - $$(echo $$line | awk '{print $$NF}')$(NC)"; \
		done; \
	else \
		echo "$(RED)✗ Hooks are not installed$(NC)"; \
		echo "$(YELLOW)  Run 'make hooks/install' to install them$(NC)"; \
	fi

# ==================================================================================== #
# BAZEL
# ==================================================================================== #

## bazel/update: update BUILD files with Gazelle
.PHONY: bazel/update
bazel/update:
	@echo "$(YELLOW)▶ Updating BUILD files...$(NC)"
	@$(BAZEL) run //:gazelle
	@echo "$(GREEN)✓ BUILD files updated$(NC)"

## bazel/clean: clean Bazel cache
.PHONY: bazel/clean
bazel/clean:
	@echo "$(YELLOW)▶ Cleaning Bazel cache...$(NC)"
	@$(BAZEL) clean --expunge
	@echo "$(GREEN)✓ Bazel cache cleaned$(NC)"

## bazel/info: show Bazel workspace info
.PHONY: bazel/info
bazel/info:
	@echo "$(YELLOW)▶ Bazel workspace info:$(NC)"
	@$(BAZEL) info

# ==================================================================================== #
# OPERATIONS
# ==================================================================================== #

## clean: clean build artifacts and caches
.PHONY: clean
clean:
	@echo "$(YELLOW)▶ Cleaning...$(NC)"
	@$(BAZEL) clean
	@rm -rf bazel-*
	@$(GO) clean -cache
	@echo "$(GREEN)✓ Cleaned$(NC)"

## version: show version information
.PHONY: version
version:
	@echo "$(PROJECT_NAME) version: $(VERSION)"
	@echo "Go version: $(shell go version)"
	@echo "Bazel version: $(shell bazel version --gnu_format 2>/dev/null | head -1)"

# ==================================================================================== #
# TOOLS
# ==================================================================================== #

## tools/check: check if required tools are installed
.PHONY: tools/check
tools/check:
	@echo "$(YELLOW)▶ Checking required tools...$(NC)"
	@echo -n "  Go:           "
	@if command -v $(GO) >/dev/null 2>&1; then \
		echo "$(GREEN)✓$(NC) $(shell go version | cut -d' ' -f3)"; \
	else \
		echo "$(RED)✗ Not installed$(NC)"; \
	fi
	@echo -n "  Bazel:        "
	@if command -v $(BAZEL) >/dev/null 2>&1; then \
		echo "$(GREEN)✓$(NC) $(shell bazel version --gnu_format 2>/dev/null | head -1)"; \
	else \
		echo "$(RED)✗ Not installed$(NC)"; \
	fi
	@echo -n "  Git:          "
	@if command -v git >/dev/null 2>&1; then \
		echo "$(GREEN)✓$(NC) $(shell git --version | cut -d' ' -f3)"; \
	else \
		echo "$(RED)✗ Not installed$(NC)"; \
	fi
	@echo ""
	@echo "$(YELLOW)▶ Checking optional tools...$(NC)"
	@echo -n "  golangci-lint:"
	@if command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
		echo "$(GREEN)✓$(NC) $(shell $(GOLANGCI_LINT) --version | head -1 | cut -d' ' -f4)"; \
	else \
		echo "$(YELLOW)○ Not installed (optional)$(NC)"; \
	fi
	@echo -n "  prettier:     "
	@if command -v $(PRETTIER) >/dev/null 2>&1; then \
		echo "$(GREEN)✓$(NC) $(shell $(PRETTIER) --version)"; \
	else \
		echo "$(YELLOW)○ Not installed (optional)$(NC)"; \
	fi
	@echo -n "  gosec:        "
	@if command -v gosec >/dev/null 2>&1; then \
		echo "$(GREEN)✓$(NC) $(shell gosec --version | grep Version | cut -d' ' -f2)"; \
	else \
		echo "$(YELLOW)○ Not installed (optional)$(NC)"; \
	fi

## tools/install: install optional development tools
.PHONY: tools/install
tools/install:
	@echo "$(YELLOW)▶ Installing optional tools...$(NC)"
	@echo "$(CYAN)Installing golangci-lint...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(CYAN)Installing gosec...$(NC)"
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "$(CYAN)Installing prettier...$(NC)"
	@if command -v npm >/dev/null 2>&1; then \
		npm install -g prettier; \
	else \
		echo "$(YELLOW)⚠ npm not found, skipping prettier installation$(NC)"; \
	fi
	@echo "$(GREEN)✓ Tools installed$(NC)"

# ==================================================================================== #
# CI/CD
# ==================================================================================== #

## ci: run all CI checks (quality, test, build)
.PHONY: ci
ci: quality/validate build
	@echo "$(GREEN)✓ All CI checks passed$(NC)"

## ci/validate: validate CI pipeline locally
.PHONY: ci/validate
ci/validate:
	@echo "$(YELLOW)▶ Validating CI pipeline...$(NC)"
	@$(MAKE) tools/check
	@$(MAKE) deps
	@$(MAKE) quality/analyze
	@$(MAKE) test
	@$(MAKE) build
	@echo "$(GREEN)✓ CI validation complete$(NC)"

# ==================================================================================== #
# SHORTCUTS & ALIASES
# ==================================================================================== #

.PHONY: t
t: test

.PHONY: b
b: build

.PHONY: f
f: fmt

.PHONY: l
l: quality/lint

.PHONY: q
q: quality/analyze