# Task Dashboard

A desktop kanban board that automatically launches Claude agents to complete tasks when moved to "In Progress".

## Key Innovation

**Automatic task execution**: Drag a task from "To Do" to "In Progress" and a Claude agent automatically spawns in an isolated git worktree to complete it autonomously.

## Architecture Highlights

- **Hybrid Desktop/Web**: Built with Wails (Go + React) - runs as native desktop app or web interface for testing
- **Concurrent Agent System**: Manages multiple Claude agents working in parallel using git worktree pooling
- **Atomic File Operations**: All task updates use temp file + atomic rename with automatic backups
- **Unified Logging**: Single daily log file aggregates all system activity for easy observability

## Quick Start

```bash
make install  # Install dependencies
make dev      # Start development mode (desktop + web at localhost:34115)
make build    # Build standalone executable
```

## Core Features

- **Visual Kanban Board**: Drag-and-drop tasks between Backlog, To Do, In Progress, Pending Review, and Done
- **Agent Automation**: Moving task to "doing" triggers: `claude "Work on task #XX" --dangerously-skip-permissions`
- **Isolated Workspaces**: Each agent works in dedicated git worktree (`cc_task_dash-subagent1`, etc.)
- **Review Workflow**: Agents commit to `task_XX` branches and mark tasks as "pending_review"

## Project Structure

```
task-dashboard/          # Wails desktop application
├── app.go              # Go backend with task CRUD operations
├── frontend/           # React TypeScript UI
└── build/bin/          # Compiled 8.1MB standalone app

plan/                   # Project management
├── task.json           # Single source of truth for all tasks
└── helpers_and_tools/  # Agent management scripts
    ├── agent_spawn.sh  # Manages worktree pool and launches agents
    └── monitor_claude_agent.sh  # Live agent progress monitoring
```

## Technical Stack

- **Backend**: Go with mutex-protected file operations
- **Frontend**: React 18, TypeScript, Tailwind CSS, Framer Motion
- **Drag & Drop**: @hello-pangea/dnd for smooth kanban interactions
- **Logging**: Centralized daily logs in `logs/universal_logs-YYYY-MM-DD.log`

## Development Philosophy

- **Exploration over perfection**: Failed experiments that generate learnings > safe implementations
- **Minimal and decoupled**: Simple tools composed effectively
- **Observable by default**: Everything logs to one place

Built to transform task management from manual coordination to autonomous execution.