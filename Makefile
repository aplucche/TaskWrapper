# --- Task Dashboard Makefile ---
# variables
APP     ?= task-dashboard
LOGDIR   = logs
LATEST   = $(shell ls $(LOGDIR)/universal_logs-*.log 2>/dev/null | sort | tail -1)
WAILS_DIR = task-dashboard
PATH_WITH_GO = $(shell echo $$PATH:/usr/local/go/bin:$$HOME/go/bin)

.PHONY: help build test run dev logs install web

help:   ## list targets
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | awk 'BEGIN{FS="[:#]"}{printf "%-12s %s\n", $$1, $$3}'

install:  ## install dependencies
	cd $(WAILS_DIR) && PATH=$(PATH_WITH_GO) npm install

build:  ## compile / package wails app
	cd $(WAILS_DIR) && PATH=$(PATH_WITH_GO) wails build

test:   ## run tests
	cd $(WAILS_DIR) && PATH=$(PATH_WITH_GO) go test ./...

run:    ## start desktop app
	cd $(WAILS_DIR) && PATH=$(PATH_WITH_GO) wails build && ./build/bin/$(APP)

dev:    ## live-reload / watch (desktop + web)
	cd $(WAILS_DIR) && PATH=$(PATH_WITH_GO) wails dev

web:    ## info for web testing
	@echo "When running 'make dev', the app is available at:"
	@echo "  Desktop: Native Wails app window"
	@echo "  Web:     http://localhost:34115"
	@echo "  Playwright: Target http://localhost:34115 for testing"

logs:   ## tail latest log
	@plan/helpers_and_tools/log_viewer.sh