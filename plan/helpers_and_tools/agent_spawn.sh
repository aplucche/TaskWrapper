#!/usr/bin/env bash
# agent_spawn.sh - Subagent spawner with worktree pooling and reuse
set -euo pipefail

# Configuration
ROOT=$(git rev-parse --show-toplevel)
REPO=$(basename "$ROOT")
PARENT=$(dirname "$ROOT")
MAX_SUBAGENTS=${MAX_SUBAGENTS:-2}
LOCK_TIMEOUT=${AGENT_LOCK_TIMEOUT:-7200}  # 2 hours default

# Arguments
TASK_ID=${1:?usage: $0 TASK_ID "TITLE"}
TITLE=${2:-""}

# Ensure git worktree list is clean
git -C "$ROOT" worktree prune >/dev/null 2>&1

# Function to check if a worktree is available
is_worktree_available() {
    local worktree_dir="$1"
    local lockfile="$worktree_dir/.agent_state"
    
    # No lockfile means available
    [[ ! -f "$lockfile" ]] && return 0
    
    # Check if lock is stale
    local pid=$(grep "^pid=" "$lockfile" 2>/dev/null | cut -d= -f2)
    local started=$(grep "^started=" "$lockfile" 2>/dev/null | cut -d= -f2)
    
    # If no PID or started time, consider it stale
    [[ -z "$pid" || -z "$started" ]] && return 0
    
    # Check if process is still running
    if ! kill -0 "$pid" 2>/dev/null; then
        echo "Cleaning stale lock for worktree $(basename "$worktree_dir") (dead pid: $pid)"
        rm -f "$lockfile"
        return 0
    fi
    
    # Check if lock is too old
    local now=$(date +%s)
    if (( now - started > LOCK_TIMEOUT )); then
        echo "Cleaning stale lock for worktree $(basename "$worktree_dir") (timeout exceeded)"
        rm -f "$lockfile"
        return 0
    fi
    
    return 1
}

# Function to prepare a worktree for use
prepare_worktree() {
    local worktree_dir="$1"
    local task_id="$2"
    
    cd "$worktree_dir"
    
    # Clean up any uncommitted changes
    git reset --hard HEAD >/dev/null 2>&1
    git clean -fd >/dev/null 2>&1
    
    # Just use the current main branch state
    git checkout --detach main >/dev/null 2>&1
    
    # Create task branch
    git checkout -b "task_${task_id}" >/dev/null 2>&1
}

# Clean up any stale locks first
echo "Checking for stale locks..."
stale_cleaned=0
for i in $(seq 1 "$MAX_SUBAGENTS"); do
    dir="$PARENT/${REPO}-subagent$i"
    lockfile="$dir/.agent_state"
    
    if [[ -f "$lockfile" ]]; then
        pid=$(grep "^pid=" "$lockfile" 2>/dev/null | cut -d= -f2 || echo "")
        started=$(grep "^started=" "$lockfile" 2>/dev/null | cut -d= -f2 || echo "")
        
        # Check if lock is stale (no PID, dead process, or timeout)
        should_clean=false
        if [[ -z "$pid" || -z "$started" ]]; then
            should_clean=true
        elif ! kill -0 "$pid" 2>/dev/null; then
            should_clean=true
        else
            # Check timeout
            now=$(date +%s)
            if (( now - started > LOCK_TIMEOUT )); then
                should_clean=true
            fi
        fi
        
        if [ "$should_clean" = true ]; then
            echo "Cleaning stale lock: subagent$i (pid: ${pid:-unknown})"
            rm -f "$lockfile"
            ((stale_cleaned++))
        fi
    fi
done

if [ $stale_cleaned -gt 0 ]; then
    echo "Cleaned $stale_cleaned stale lock(s)"
fi

# Find or create an available worktree
WORKTREE_DIR=""
WORKTREE_NUM=""

# First, try to find an existing available worktree
for i in $(seq 1 "$MAX_SUBAGENTS"); do
    dir="$PARENT/${REPO}-subagent$i"
    if [[ -d "$dir" ]] && is_worktree_available "$dir"; then
        WORKTREE_DIR="$dir"
        WORKTREE_NUM="$i"
        echo "Reusing existing worktree: subagent$i"
        break
    fi
done

# If no available worktree found, try to create a new one
if [[ -z "$WORKTREE_DIR" ]]; then
    # Count existing worktrees
    existing_count=$(git -C "$ROOT" worktree list --porcelain | grep -c "^worktree.*-subagent[0-9]" || true)
    
    if (( existing_count >= MAX_SUBAGENTS )); then
        echo "❌ All $MAX_SUBAGENTS subagent worktrees are busy"
        echo "Run 'plan/helpers_and_tools/agent_status.sh' to see current status"
        exit 1
    fi
    
    # Find next available number
    for i in $(seq 1 "$MAX_SUBAGENTS"); do
        dir="$PARENT/${REPO}-subagent$i"
        if [[ ! -d "$dir" ]]; then
            WORKTREE_DIR="$dir"
            WORKTREE_NUM="$i"
            echo "Creating new worktree: subagent$i"
            git -C "$ROOT" worktree add --detach "$dir" main >/dev/null 2>&1
            break
        fi
    done
fi

if [[ -z "$WORKTREE_DIR" ]]; then
    echo "❌ Failed to allocate worktree"
    exit 1
fi

# Prepare the worktree
echo "Preparing worktree for task #$TASK_ID..."
prepare_worktree "$WORKTREE_DIR" "$TASK_ID"

# Create the prompt
PROMPT="Review plan.md and task.json.
Begin task #$TASK_ID: $TITLE.

IMPORTANT: When you complete the task:
1. Do your work and commit to branch task_$TASK_ID
2. CRITICAL: Update $ROOT/plan/task.json (main branch) to change task #$TASK_ID status from 'doing' to 'pending_review'
3. The task.json status update must be on main branch so the Task Dashboard can see it immediately

Note: You're working in a separate worktree. Your task work goes on task_$TASK_ID branch, but the status update goes to main branch task.json."

# Launch the agent and capture PID
(
    cd "$WORKTREE_DIR"
    
    # Write lock file
    cat > .agent_state <<EOF
status=busy
pid=$$
task_id=$TASK_ID
task_title=$TITLE
started=$(date +%s)
started_human=$(date)
worktree=subagent$WORKTREE_NUM
EOF
    
    # Run Claude (ensure PATH includes common locations)
    export PATH="$PATH:/usr/local/bin:/Users/aplucche/.nvm/versions/node/v20.16.0/bin"
    claude "$PROMPT" --dangerously-skip-permissions
    
    # Switch back to detached main to allow branch deletion
    git checkout --detach main >/dev/null 2>&1
    
    # Clean up lock file when done
    rm -f .agent_state
) &

AGENT_PID=$!

echo "✅ Launched subagent$WORKTREE_NUM → task #$TASK_ID (pid: $AGENT_PID)"
echo "   Working in: $WORKTREE_DIR"

# Update the lock file with the actual agent PID
sleep 0.5  # Give the subshell time to create the file
if [[ -f "$WORKTREE_DIR/.agent_state" ]]; then
    sed -i.bak "s/^pid=.*/pid=$AGENT_PID/" "$WORKTREE_DIR/.agent_state"
    rm -f "$WORKTREE_DIR/.agent_state.bak"
fi