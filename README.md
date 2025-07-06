# TaskWrapper v0.1.0

A desktop kanban board that automatically launches Claude agents to complete tasks when moved to "In Progress".

## Key Features

- **Visual Kanban Board**: Drag-and-drop tasks between Backlog, To Do, In Progress, Pending Review, and Done
**Automatic Agent Task Execution**: Drag a task from "To Do" to "In Progress" and a Claude agent automatically spawns in an isolated git worktree to complete it autonomously. Moving task to "doing" triggers: `claude "Work on task #XX" --dangerously-skip-permissions` (Use at own risk!)
- **Isolated Workspaces**: Each agent works in dedicated git worktree (`cc_task_dash-subagent1`, etc.)
- **Review Workflow**: Agents commit to `task_XX` branches and mark tasks as "pending_review". Approve or reject from within app

## Architecture Highlights

- **Hybrid Desktop/Web**: Built with Wails (Go + React) - runs as native desktop app or web interface for testing
- **Concurrent Agent System**: Manages multiple Claude agents working in parallel using git worktree pooling
- **Unified Logging**: Single daily log file aggregates all system activity for easy observability

## v0.1.0 Release
- First alpha release with core kanban and agent automation features
- Production-ready single executable (~8MB). macOS only

## Quick Start

```bash
make install  # Install dependencies
make dev      # Start development mode (desktop + web at localhost:34115)
make build    # Build standalone executable
make run      # Run the built application
```

## Project Structure
```
taskwrapper/            # Wails desktop application
├── app.go              # Go backend with task CRUD operations
├── frontend/           # React TypeScript UI
└── build/bin/          # Compiled 8.1MB standalone app

plan/                   # Project management
├── task.json           # Single source of truth for all tasks
└── helpers_and_tools/  # Agent management scripts
    ├── agent_spawn.sh  # Manages worktree pool and launches agents
    └── monitor_claude_agent.sh  # Live agent progress monitoring
```

