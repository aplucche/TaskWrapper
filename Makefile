# --- TaskWrapper Makefile ---
# variables
APP     ?= taskwrapper
LOGDIR   = logs
LATEST   = $(shell ls $(LOGDIR)/universal_logs-*.log 2>/dev/null | sort | tail -1)
WAILS_DIR = taskwrapper
PATH_WITH_GO = $(shell echo $$PATH:/usr/local/go/bin:$$HOME/go/bin)
MAX_SUBAGENTS ?= 2

.PHONY: help build clean rebuild test run dev logs install web agent-status agent-watch agent-cleanup agent-cleanup-force agent-test agent-logs agent-logs-follow add_subagent

help:   ## list targets
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | awk 'BEGIN{FS=" *## *"}{printf "%-20s %s\n", $$1, $$2}' | sed 's/:.*##//'

install:  ## install dependencies
	cd $(WAILS_DIR)/frontend && npm install

build:  ## compile / package wails app
	@echo "Preparing build environment..."
	@cd $(WAILS_DIR)/frontend && npm install --silent
	@cd $(WAILS_DIR) && PATH=$(PATH_WITH_GO) go mod tidy
	@echo "Building application..."
	cd $(WAILS_DIR) && PATH=$(PATH_WITH_GO) wails build
	@echo "Build completed successfully!"
	@ls -la $(WAILS_DIR)/build/bin/

clean:  ## clean build artifacts and regenerate bindings  
	@echo "Cleaning build artifacts..."
	@rm -rf $(WAILS_DIR)/build/
	@rm -rf $(WAILS_DIR)/frontend/dist/
	@rm -rf $(WAILS_DIR)/frontend/wailsjs/
	@echo "Clean completed!"

rebuild:  ## clean build artifacts and rebuild from scratch
	@$(MAKE) clean
	@$(MAKE) build

test:   ## run all tests
	cd $(WAILS_DIR) && PATH=$(PATH_WITH_GO) go test ./...
	@echo "\nValidating subagent system..."
	@./plan/helpers_and_tools/test_subagent_system.sh

test-go:  ## run Go backend tests
	cd $(WAILS_DIR) && PATH=$(PATH_WITH_GO) go test -v

run:    ## start desktop app
	cd $(WAILS_DIR) && PATH=$(PATH_WITH_GO) wails build && open ./build/bin/$(APP).app

dev:    ## live-reload / watch (desktop + web)
	cd $(WAILS_DIR) && PATH=$(PATH_WITH_GO) wails dev

web:    ## info for web testing
	@echo "When running 'make dev', the app is available at:"
	@echo "  Desktop: Native Wails app window"
	@echo "  Web:     http://localhost:34115"
	@echo "  Playwright: Target http://localhost:34115 for testing"

logs:   ## tail latest log
	@plan/helpers_and_tools/log_viewer.sh

agent-status: ## show current agent status
	@plan/helpers_and_tools/agent_status.sh

agent-watch: ## watch agent status (refreshes every 5s, ctrl+c to exit)
	@echo "Watching agent status (press Ctrl+C to exit)..."
	@while true; do \
		clear; \
		plan/helpers_and_tools/agent_status.sh; \
		sleep 5; \
	done

agent-cleanup: ## clean up stale agent locks
	@plan/helpers_and_tools/agent_cleanup.sh

agent-cleanup-force: ## force clean up stale agent locks
	@plan/helpers_and_tools/agent_cleanup.sh --force

agent-test: ## validate agent system components
	@plan/helpers_and_tools/test_subagent_system.sh

agent-logs: ## show detailed logs from subagents
	@echo "Showing recent subagent activity from universal logs..."
	@tail -n 100 logs/universal_logs-*.log 2>/dev/null | grep -E "(subagent|agent)" --color=always || echo "No agent logs found"

agent-logs-follow: ## follow subagent logs in real-time
	@echo "Following subagent logs in real-time (press Ctrl+C to exit)..."
	@tail -f logs/universal_logs-*.log 2>/dev/null | grep --line-buffered -E "(subagent|agent)" --color=always

add_subagent: ## create ../<repo>-subagentN worktree (detached, capped)
	@set -e; \
	root=$$(git rev-parse --show-toplevel); \
	git -C "$$root" worktree prune >/dev/null; \
	repo=$$(basename "$$root"); \
	parent=$$(dirname  "$$root"); \
	n=$$(git -C "$$root" worktree list --porcelain | \
	     awk '/^worktree/ && $$2 ~ /-subagent[0-9]+$$/ {sub(/.*-subagent/, "", $$2); if($$2+0>m) m=$$2} END{print m+1}' m=0); \
	if [ "$$n" -gt $(MAX_SUBAGENTS) ]; then \
	    echo "🚫  Max $(MAX_SUBAGENTS) subagents reached—no new worktree made."; exit 1; \
	fi; \
	path=$$parent/$${repo}-subagent$${n}; \
	git -C "$$root" worktree add --detach -f "$$path" main; \
	echo "✅  Worktree $$path ready (subagent $$n, detached HEAD)" \
	echo; echo "Current worktrees:"; \
	git -C "$$root" worktree list