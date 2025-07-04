#!/usr/bin/env bash
# agent_status.sh - Monitor status of all subagent worktrees
set -euo pipefail

# Configuration
ROOT=$(git rev-parse --show-toplevel)
REPO=$(basename "$ROOT")
PARENT=$(dirname "$ROOT")

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Subagent Worktree Status ===${NC}"
echo

# Get all worktrees
worktrees=$(git -C "$ROOT" worktree list --porcelain | grep "^worktree.*-subagent[0-9]" | awk '{print $2}' | sort -V)

if [[ -z "$worktrees" ]]; then
    echo "No subagent worktrees found."
    echo "Run 'make add_subagent' to create one."
    exit 0
fi

# Track totals
total=0
busy=0
idle=0

# Check each worktree
for worktree in $worktrees; do
    ((total++))
    name=$(basename "$worktree")
    lockfile="$worktree/.agent_state"
    
    if [[ ! -f "$lockfile" ]]; then
        echo -e "${GREEN}✓ $name${NC} - IDLE"
        ((idle++))
    else
        # Read lock file
        status=$(grep "^status=" "$lockfile" 2>/dev/null | cut -d= -f2 || echo "unknown")
        pid=$(grep "^pid=" "$lockfile" 2>/dev/null | cut -d= -f2 || echo "?")
        task_id=$(grep "^task_id=" "$lockfile" 2>/dev/null | cut -d= -f2 || echo "?")
        task_title=$(grep "^task_title=" "$lockfile" 2>/dev/null | cut -d= -f2 || echo "?")
        started=$(grep "^started_human=" "$lockfile" 2>/dev/null | cut -d= -f2- || echo "?")
        
        # Check if process is still running
        if [[ "$pid" != "?" ]] && kill -0 "$pid" 2>/dev/null; then
            echo -e "${YELLOW}● $name${NC} - BUSY"
            echo "    Task: #$task_id - $task_title"
            echo "    PID: $pid (running)"
            echo "    Started: $started"
            ((busy++))
        else
            echo -e "${RED}! $name${NC} - STALE LOCK"
            echo "    Task: #$task_id - $task_title"
            echo "    PID: $pid (dead)"
            echo "    Lock file should be cleaned up"
        fi
    fi
    echo
done

# Summary
echo -e "${BLUE}=== Summary ===${NC}"
echo "Total worktrees: $total"
echo -e "Idle: ${GREEN}$idle${NC}"
echo -e "Busy: ${YELLOW}$busy${NC}"

# Check against MAX_SUBAGENTS
MAX_SUBAGENTS=${MAX_SUBAGENTS:-2}
if (( total < MAX_SUBAGENTS )); then
    available=$((MAX_SUBAGENTS - total))
    echo -e "Available slots: ${GREEN}$available${NC} (can create $available more worktrees)"
fi

# Git worktree summary
echo
echo -e "${BLUE}=== Git Worktree List ===${NC}"
git -C "$ROOT" worktree list | grep -E "(main|subagent)" || true