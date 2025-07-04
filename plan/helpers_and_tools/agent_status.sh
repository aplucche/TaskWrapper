#!/bin/bash
# Quick agent status check

TASK_ID=${1:-51}
AGENT_PID=${2}

echo "🤖 Claude Agent Status Check"
echo "============================="

# Check task status
TASK_STATUS=$(grep -A 5 -B 1 "\"id\": $TASK_ID" plan/task.json | grep "status" | cut -d'"' -f4)
echo "📋 Task #$TASK_ID Status: $TASK_STATUS"

# Check if branch exists
if git branch -a | grep -q "task_$TASK_ID"; then
    echo "🌳 Branch task_$TASK_ID: EXISTS"
else
    echo "🌳 Branch task_$TASK_ID: NOT FOUND"
fi

# Check if PID provided and running
if [ -n "$AGENT_PID" ]; then
    if ps -p $AGENT_PID > /dev/null 2>&1; then
        echo "⚡ Agent PID $AGENT_PID: RUNNING"
    else
        echo "💀 Agent PID $AGENT_PID: STOPPED"
    fi
fi

# Check recent commits
echo ""
echo "📝 Recent Commits:"
git log --oneline -3

# Check recent universal log entries for this task
echo ""
echo "📊 Recent Agent Activity:"
LOG_FILE="logs/universal_logs-$(date +%Y-%m-%d).log"
grep "task.*$TASK_ID\|claude.*$TASK_ID" "$LOG_FILE" | tail -3