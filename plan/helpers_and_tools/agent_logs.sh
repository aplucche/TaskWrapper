#!/usr/bin/env bash
# agent_logs.sh - View detailed logs from subagents
set -euo pipefail

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
ROOT=$(git rev-parse --show-toplevel)
REPO=$(basename "$ROOT")
PARENT=$(dirname "$ROOT")

usage() {
    echo "Usage: $0 [OPTIONS] [TASK_ID]"
    echo
    echo "View detailed logs from subagents"
    echo
    echo "Options:"
    echo "  -f, --follow     Follow logs in real-time (like tail -f)"
    echo "  -n N             Show last N lines (default: 50)"
    echo "  -a, --all        Show logs from all agents"
    echo "  -h, --help       Show this help"
    echo
    echo "Examples:"
    echo "  $0               Show recent logs from all agents"
    echo "  $0 -f            Follow all agent logs in real-time"
    echo "  $0 62            Show logs for task #62"
    echo "  $0 -f 62         Follow logs for task #62"
    echo "  $0 -n 100        Show last 100 lines from all agents"
}

# Parse arguments
FOLLOW=false
LINES=50
TASK_ID=""
SHOW_ALL=true

while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--follow)
            FOLLOW=true
            shift
            ;;
        -n)
            LINES="$2"
            shift 2
            ;;
        -a|--all)
            SHOW_ALL=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        -*)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            TASK_ID="$1"
            SHOW_ALL=false
            shift
            ;;
    esac
done

# Find log files
UNIVERSAL_LOG=$(ls logs/universal_logs-*.log 2>/dev/null | sort | tail -1)

if [[ ! -f "$UNIVERSAL_LOG" ]]; then
    echo -e "${RED}Error: No universal log found${NC}"
    exit 1
fi

echo -e "${BLUE}=== Subagent Logs ===${NC}"

if [[ "$SHOW_ALL" == "true" ]]; then
    echo "Showing logs from all subagents..."
    
    if [[ "$FOLLOW" == "true" ]]; then
        echo -e "${YELLOW}Following logs in real-time (press Ctrl+C to exit)...${NC}"
        echo
        tail -f "$UNIVERSAL_LOG" | grep --line-buffered -E "(agent|Claude|task #|spawn)" -i --color=always
    else
        echo "Last $LINES lines containing agent activity:"
        echo
        tail -n "$LINES" "$UNIVERSAL_LOG" | grep -E "(agent|Claude|task #|spawn)" -i --color=always
    fi
else
    echo "Showing logs for task #$TASK_ID..."
    
    if [[ "$FOLLOW" == "true" ]]; then
        echo -e "${YELLOW}Following logs for task #$TASK_ID in real-time (press Ctrl+C to exit)...${NC}"
        echo
        tail -f "$UNIVERSAL_LOG" | grep --line-buffered -E "task #?$TASK_ID" -i --color=always
    else
        echo "Last $LINES lines for task #$TASK_ID:"
        echo
        tail -n "$LINES" "$UNIVERSAL_LOG" | grep -E "task #?$TASK_ID" -i --color=always
    fi
fi

# If no results found
if [[ $? -ne 0 ]] && [[ "$FOLLOW" == "false" ]]; then
    echo -e "${YELLOW}No agent logs found${NC}"
    echo
    echo "Try:"
    echo "  - Check if any agents are running: make agent-status"
    echo "  - View general logs: make logs"
    echo "  - Increase line count: $0 -n 200"
fi