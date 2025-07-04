#!/usr/bin/env bash
# agent_cleanup.sh - Clean up subagent worktrees and stale locks
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

# Parse arguments
FORCE=false
REMOVE_ALL=false

usage() {
    echo "Usage: $0 [-f|--force] [-a|--all] [-h|--help]"
    echo
    echo "Options:"
    echo "  -f, --force    Force cleanup without confirmation"
    echo "  -a, --all      Remove all subagent worktrees (not just stale ones)"
    echo "  -h, --help     Show this help message"
    echo
    echo "By default, only cleans up stale locks and dead worktrees."
}

while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--force)
            FORCE=true
            shift
            ;;
        -a|--all)
            REMOVE_ALL=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

echo -e "${BLUE}=== Subagent Cleanup ===${NC}"
echo

# First, prune any already removed worktrees
echo "Pruning git worktree list..."
git -C "$ROOT" worktree prune
echo

# Get all subagent worktrees
worktrees=$(git -C "$ROOT" worktree list --porcelain | grep "^worktree.*-subagent[0-9]" | awk '{print $2}' || true)

if [[ -z "$worktrees" ]]; then
    echo "No subagent worktrees found."
    exit 0
fi

# Track what we'll clean
to_clean=()
stale_locks=()
busy_agents=()

# Check each worktree
for worktree in $worktrees; do
    name=$(basename "$worktree")
    lockfile="$worktree/.agent_state"
    
    if [[ ! -f "$lockfile" ]]; then
        if [[ "$REMOVE_ALL" == "true" ]]; then
            to_clean+=("$worktree")
            echo -e "${YELLOW}Will remove idle worktree:${NC} $name"
        else
            echo -e "${GREEN}✓ $name${NC} - IDLE (keeping)"
        fi
    else
        # Check if lock is stale
        pid=$(grep "^pid=" "$lockfile" 2>/dev/null | cut -d= -f2 || echo "")
        
        if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
            if [[ "$REMOVE_ALL" == "true" ]]; then
                busy_agents+=("$name (PID: $pid)")
                echo -e "${RED}! $name${NC} - BUSY (PID: $pid still running)"
            else
                echo -e "${YELLOW}● $name${NC} - BUSY (keeping)"
            fi
        else
            stale_locks+=("$lockfile")
            echo -e "${RED}Will clean stale lock:${NC} $name"
            if [[ "$REMOVE_ALL" == "true" ]]; then
                to_clean+=("$worktree")
                echo -e "${YELLOW}Will also remove worktree:${NC} $name"
            fi
        fi
    fi
done

echo

# If nothing to do
if [[ ${#stale_locks[@]} -eq 0 && ${#to_clean[@]} -eq 0 ]]; then
    echo "Nothing to clean up!"
    exit 0
fi

# If trying to remove busy agents
if [[ ${#busy_agents[@]} -gt 0 && "$REMOVE_ALL" == "true" ]]; then
    echo -e "${RED}ERROR: Cannot remove worktrees with running agents:${NC}"
    for agent in "${busy_agents[@]}"; do
        echo "  - $agent"
    done
    echo
    echo "Stop these agents first or wait for them to complete."
    exit 1
fi

# Confirm action
if [[ "$FORCE" != "true" ]]; then
    echo -e "${BLUE}=== Cleanup Summary ===${NC}"
    
    if [[ ${#stale_locks[@]} -gt 0 ]]; then
        echo "Will remove ${#stale_locks[@]} stale lock file(s)"
    fi
    
    if [[ ${#to_clean[@]} -gt 0 ]]; then
        echo "Will remove ${#to_clean[@]} worktree(s)"
    fi
    
    echo
    read -p "Proceed with cleanup? [y/N] " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Cleanup cancelled."
        exit 0
    fi
fi

# Perform cleanup
echo
echo -e "${BLUE}=== Performing Cleanup ===${NC}"

# Clean stale locks
for lockfile in "${stale_locks[@]}"; do
    echo -n "Removing stale lock: $lockfile ... "
    rm -f "$lockfile"
    echo -e "${GREEN}done${NC}"
done

# Remove worktrees
for worktree in "${to_clean[@]}"; do
    name=$(basename "$worktree")
    echo -n "Removing worktree: $name ... "
    
    # Remove the worktree
    git -C "$ROOT" worktree remove --force "$worktree" 2>/dev/null || true
    
    # If git didn't remove it, force remove the directory
    if [[ -d "$worktree" ]]; then
        rm -rf "$worktree"
    fi
    
    echo -e "${GREEN}done${NC}"
done

# Final prune
if [[ ${#to_clean[@]} -gt 0 ]]; then
    echo
    echo "Running final git worktree prune..."
    git -C "$ROOT" worktree prune
fi

echo
echo -e "${GREEN}✅ Cleanup complete!${NC}"

# Show final status
echo
./plan/helpers_and_tools/agent_status.sh 2>/dev/null || true