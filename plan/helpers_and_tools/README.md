# Helper Tools & Scripts

This directory contains utility scripts for monitoring and managing the Task Dashboard ecosystem, including Claude agent automation.

## Available Tools

### 1. `log_viewer.sh` - Universal Log Viewer
Reads the tail of logs in the log folder for real-time monitoring.

**Usage:**
```bash
./log_viewer.sh
# or via Makefile:
make logs
```

**Purpose:** Monitor all application activity through the universal logging system.

---

### 2. `agent_status.sh` - Claude Agent Status Check
Quick status check for Claude agents launched by the Task Dashboard.

**Usage:**
```bash
./agent_status.sh [TASK_ID] [AGENT_PID]

# Examples:
./agent_status.sh 51          # Check status of task #51
./agent_status.sh 51 62828    # Check task #51 with specific PID
```

**Shows:**
- Current task status in task.json
- Whether task branch exists (task_XX)
- Agent process status (if PID provided)
- Recent git commits
- Recent agent activity from universal logs

**Purpose:** Quick health check of Claude agents without continuous monitoring.

---

### 3. `monitor_claude_agent.sh` - Live Claude Agent Monitor
Continuous monitoring of Claude agent progress using the universal logging system.

**Usage:**
```bash
./monitor_claude_agent.sh [TASK_ID]

# Example:
./monitor_claude_agent.sh 51   # Monitor task #51
```

**Features:**
- Follows universal logs in real-time
- Detects task status changes
- Shows relevant log entries (git, commits, errors, warnings)
- Automatically exits when task completes
- Shows final summary with branch status and commits

**Purpose:** Live monitoring of Claude agent work progress integrated with universal logs.

---

## Claude Agent Workflow

When a task is moved from "todo" to "doing" in the Task Dashboard:

1. **Automatic Launch**: Dashboard launches Claude agent with:
   ```bash
   claude "Review plan.md and task.json. Begin task #XX: [title]. Update task.json status to 'done' when complete, commit to branch task_XX, then exit." --dangerously-skip-permissions
   ```

2. **Monitoring**: Use these tools to track progress:
   ```bash
   # Quick check
   ./agent_status.sh 51
   
   # Live monitoring
   ./monitor_claude_agent.sh 51
   
   # Raw logs
   ./log_viewer.sh
   ```

3. **Agent Completion**: Agent should:
   - Update task.json status to "done"
   - Commit changes to branch `task_XX`
   - Exit automatically

4. **Manual Merge**: User reviews and merges the task branch to main

## Universal Logging Integration

All tools integrate with the universal logging system at `logs/universal_logs-YYYY-MM-DD.log`.

**Key log patterns:**
- `INFO task-dashboard: Launching Claude agent for task #XX` - Agent launch
- `INFO task-dashboard: Task XX moved from todo to doing` - Status change
- `INFO task-dashboard: Tasks saved successfully` - File updates
- `ERROR` - Any errors during operation
- `WARN` - Warnings or potential issues

## Best Practices

1. **Always monitor agents** - Use `monitor_claude_agent.sh` when launching important tasks
2. **Check logs for errors** - If an agent seems stuck, check universal logs for ERROR entries
3. **Verify branch creation** - Agents should create `task_XX` branches for their work
4. **Clean up stale agents** - If an agent runs too long, it may need manual intervention

## Troubleshooting

### Agent Not Creating Branch
- Check if agent has git access
- Verify working directory is correct
- Look for git errors in universal logs

### Agent Running Too Long
- Use `agent_status.sh` to check if still active
- Check universal logs for recent activity
- May need to manually kill process if stuck

### Task Status Not Updating
- Verify task.json has write permissions
- Check for file locking issues
- Look for save errors in logs

## Adding New Tools

When creating new helper tools:
1. Follow the naming pattern: `descriptive_name.sh`
2. Add shebang: `#!/bin/bash`
3. Make executable: `chmod +x tool_name.sh`
4. Integrate with universal logs when possible
5. Document in this README
6. Consider adding to Makefile for easy access