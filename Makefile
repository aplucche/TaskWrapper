# --- Makefile Template---
# variables
APP     ?= app
LOGDIR   = logs
LATEST   = $(shell ls $(LOGDIR)/universal_logs-*.log 2>/dev/null | sort | tail -1)

.PHONY: help build test run dev logs

help:   ## list targets
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | awk 'BEGIN{FS="[:#]"}{printf "%-7s %s\n", $$1, $$3}'

build:  ## compile / package
	@echo "build $(APP)"

test:   ## run tests
	@echo "test suite"

run:    ## start app
	@echo "run app"

dev:    ## live-reload / watch
	@echo "run dev"

logs:   ## tail latest log
	@plan/helpers_and_tools/log_viewer.sh