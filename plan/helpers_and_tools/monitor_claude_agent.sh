#!/bin/bash
# Monitor Claude agent progress using universal logs

TASK_ID=${1:-51}
LOG_FILE="logs/universal_logs-$(date +%Y-%m-%d).log"

echo "🤖 Monitoring Claude Agent for Task #$TASK_ID"
echo "📝 Using universal log: $LOG_FILE"
echo "========================================================"

# Function to check task status in task.json
check_task_status() {
    local task_id=$1
    grep -A 5 -B 1 "\"id\": $task_id" plan/task.json | grep "status" | cut -d'"' -f4
}

# Function to check if branch exists
check_branch() {
    local task_id=$1
    git branch -a | grep -q "task_$task_id" && echo "EXISTS" || echo "NOT_FOUND"
}

# Get initial status
INITIAL_STATUS=$(check_task_status $TASK_ID)
echo "📋 Initial Task Status: $INITIAL_STATUS"
echo ""

# Show recent agent launch logs
echo "🚀 Agent Launch Logs:"
grep -A 3 -B 1 "task #$TASK_ID" "$LOG_FILE" | tail -10
echo ""

echo "🔍 Monitoring for changes..."
echo "Press Ctrl+C to stop monitoring"
echo "========================================================"

# Follow the universal log for any activity
tail -f "$LOG_FILE" | while read line; do
    # Check if line contains relevant activity
    if echo "$line" | grep -q -E "(task.*$TASK_ID|git|commit|branch|claude)" || 
       echo "$line" | grep -q -E "(Tasks.*saved|Tasks.*loaded|ERROR|WARN)"; then
        
        echo "$(date '+%H:%M:%S') | $line"
        
        # Check for status changes
        CURRENT_STATUS=$(check_task_status $TASK_ID)
        if [ "$CURRENT_STATUS" != "$INITIAL_STATUS" ]; then
            echo "$(date '+%H:%M:%S') | 🔄 STATUS CHANGE: $INITIAL_STATUS → $CURRENT_STATUS"
            INITIAL_STATUS=$CURRENT_STATUS
            
            # If task is done, show final summary
            if [ "$CURRENT_STATUS" = "done" ]; then
                echo "$(date '+%H:%M:%S') | ✅ TASK COMPLETED!"
                echo "$(date '+%H:%M:%S') | 🌳 Branch Status: $(check_branch $TASK_ID)"
                echo "$(date '+%H:%M:%S') | 📝 Recent Commits:"
                git log --oneline -3 | sed 's/^/    /'
                break
            fi
        fi
    fi
done