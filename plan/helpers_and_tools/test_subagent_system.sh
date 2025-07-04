#!/usr/bin/env bash
# test_subagent_system.sh - Non-destructive tests for subagent system
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== Subagent System Validation ===${NC}"
echo

# Get to project root
cd "$(git rev-parse --show-toplevel)"

PASS=0
FAIL=0

test_check() {
    local description="$1"
    local command="$2"
    
    echo -n "Testing: $description ... "
    
    if eval "$command" >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
        ((PASS++))
    else
        echo -e "${RED}✗${NC}"
        ((FAIL++))
    fi
}

echo "1. Validating required scripts exist and are executable..."
test_check "agent_spawn.sh exists and executable" "[ -x plan/helpers_and_tools/agent_spawn.sh ]"
test_check "agent_status.sh exists and executable" "[ -x plan/helpers_and_tools/agent_status.sh ]"
test_check "agent_cleanup.sh exists and executable" "[ -x plan/helpers_and_tools/agent_cleanup.sh ]"

echo -e "\n2. Validating script syntax..."
test_check "agent_spawn.sh syntax valid" "bash -n plan/helpers_and_tools/agent_spawn.sh"
test_check "agent_status.sh syntax valid" "bash -n plan/helpers_and_tools/agent_status.sh"
test_check "agent_cleanup.sh syntax valid" "bash -n plan/helpers_and_tools/agent_cleanup.sh"

echo -e "\n3. Validating help output..."
test_check "agent_status.sh runs without error" "plan/helpers_and_tools/agent_status.sh"
test_check "agent_cleanup.sh shows help" "plan/helpers_and_tools/agent_cleanup.sh --help"

echo -e "\n4. Validating Go backend integration..."
test_check "Go backend compiles" "cd task-dashboard && go build -o /tmp/test_build ."
test_check "strconv import present" "grep -q 'strconv' task-dashboard/app.go"
test_check "agent_spawn.sh path in code" "grep -q 'agent_spawn.sh' task-dashboard/app.go"

echo -e "\n5. Validating worktree setup..."
test_check "Git worktrees exist" "git worktree list | grep -q subagent"
test_check "MAX_SUBAGENTS variable set" "grep -q 'MAX_SUBAGENTS' Makefile"
test_check "add_subagent target exists" "grep -q 'add_subagent:' Makefile"

echo -e "\n6. Validating environment..."
test_check "Claude CLI available" "which claude"
test_check "Git repository valid" "git status"
test_check "Task file exists" "[ -f plan/task.json ]"

echo -e "\n7. Testing script parameters (dry run)..."
# Test parameter validation without actually running
test_check "agent_spawn.sh requires task ID" "! plan/helpers_and_tools/agent_spawn.sh 2>/dev/null"

# Clean up test build
rm -f /tmp/test_build

echo -e "\n${BLUE}=== Test Summary ===${NC}"
echo -e "Passed: ${GREEN}$PASS${NC}"
echo -e "Failed: ${RED}$FAIL${NC}"

if [ $FAIL -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All subagent system components validated successfully!${NC}"
    echo -e "\nThe system is ready for:"
    echo "  • Moving tasks from 'todo' to 'doing' in the UI"
    echo "  • Automatic agent spawning with worktree reuse"
    echo "  • Monitoring with ./plan/helpers_and_tools/agent_status.sh"
    echo "  • Cleanup with ./plan/helpers_and_tools/agent_cleanup.sh"
    exit 0
else
    echo -e "\n${RED}❌ Some components failed validation${NC}"
    echo "Please fix the failing components before using the subagent system."
    exit 1
fi