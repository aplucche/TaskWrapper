# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **Task Dashboard** project - a standalone desktop application built with Wails (Go + React + TypeScript) that provides an editable kanban board interface for the project's task.json file. The application serves as a visual task management dashboard with drag-and-drop functionality and modern aesthetic design.

### Planning
Planning and project tracking can be found in the /plan directory. plan.md and task.md are important - use frequently to steer the project. Build simple reusable project specific tools as needed. If plan.md is not sufficiently complete insist on iteratively building it out with the user. Routinely revisit plan.md to populate task.json. Expand complex tasks into subtasks.
```
plan/
├── plan.md                          # medium to long-term horizon in plan.md
├── tasks.json                       # chunk plan.md into tasks in task.json
├── helpers_and_tools/               # Shell scripts, debug code etc. Can always be run via the shell and always includes a help flag -h
│   ├── log_viewer.sh                # reads tail of logs in log folder
```

#### Task Format (plan/task.json)
```
[
  {
    "id": 1,
    "content": "Analyze repository structure and key files",
    "status": "todo | doing | done",
    "priority": "high |medium | low",
    "dependencies": [id,…]
    "parent_id":null
  },…
]
```
 - use parent_id reference to chunk complex tasks

### Logging
Redirect logs for all applications to one log directory, partition by day. This is done to simplify observability - use it frequently.
```
logs/
├── universal_logs-<YYYY-MM-DD>.log
```

### Claude Agent Automation
When a task is moved from "todo" to "doing" in the Task Dashboard, a Claude agent is automatically launched to work on the task:

1. **Automatic Launch**: Dashboard detects todo→doing transitions and launches:
   ```bash
   claude "Review plan.md and task.json. Begin task #XX: [title]. Update task.json status to 'done' when complete, commit to branch task_XX, then exit." --dangerously-skip-permissions
   ```

2. **Agent Workflow**:
   - Reviews plan.md and task.json for context
   - Works autonomously on the task
   - Updates task.json status to "done" when complete
   - Commits changes to branch `task_XX`
   - Exits automatically

3. **Monitoring Tools** (in `plan/helpers_and_tools/`):
   - `agent_status.sh [TASK_ID] [PID]` - Quick status check
   - `monitor_claude_agent.sh [TASK_ID]` - Live progress monitoring via universal logs
   - `log_viewer.sh` - Raw universal log viewer

4. **Manual Review**: User reviews and merges task branches to main

**Note**: Only todo→doing transitions trigger agents. Direct file edits or refresh button do NOT trigger automation.

### Design Philosophy
- Prefer minimal and decoupled solutions. Lean on effective high leverage data structures to guide development
- Favor extensible patterns over precise outcomes. A design that can grow and adapt is preferred to a flawless one-shot solution.
- Test extensively and intelligently, prune ununsed code aggressively.
- **Exploration over perfection**: Don't fear ambitious attempts that exceed scope. Create exploratory branches, learn deeply, document findings, then implement scaled-down versions. Failed experiments that generate learnings are more valuable than safe implementations that teach nothing.

### Exploration Learning Workflow
When an implementation becomes too complex or hits significant obstacles, use this workflow to preserve learnings and restart effectively:

1. **Preserve the attempt**: 
   ```bash
   git checkout -b exploration_[feature_name]
   git add . && git commit -m "feat: exploration attempt for [feature_name] - preserve learnings"
   ```

2. **Return to main and document learnings**:
   ```bash
   git checkout main
   # Create plan/[feature_name]_exploration.md with:
   # - What was attempted and why
   # - Dead ends encountered and root causes  
   # - What would be done differently
   # - Scaled-down approach for next attempt
   ```

3. **Start fresh with learnings applied**:
   ```bash
   git checkout -b [feature_name]_v2
   # Implement scaled-down version using documented learnings
   ```

**Key principle**: Failed experiments that generate documented learnings create more project value than perfect implementations that teach nothing. Make pivoting psychologically easy by treating exploration branches as valuable research, not "mistakes."


## Task Dashboard Application

### Architecture
- **Backend**: Go with Wails framework for desktop integration
- **Frontend**: React 18 + TypeScript + Vite
- **UI Framework**: Tailwind CSS, Headless UI, Framer Motion
- **Drag & Drop**: @hello-pangea/dnd
- **Icons**: Lucide React, Heroicons

### Enhanced Review System
Integrated review workflow: `pending_review` tasks appear in Done column with approve/reject buttons. Approve merges `task_X` branch to main; reject deletes branch with "NOT MERGED:" prefix. Backend handles git operations with comprehensive error handling and logging.

### Key Files and Directories
```
task-dashboard/
├── main.go                      # Wails app entry point
├── app.go                       # Go backend API (LoadTasks, SaveTasks, etc.)
├── frontend/
│   ├── src/
│   │   ├── App.tsx              # Main React component
│   │   ├── components/          # Kanban board components
│   │   │   ├── KanbanBoard.tsx  # Main board component
│   │   │   ├── Column.tsx       # Kanban columns
│   │   │   ├── TaskCard.tsx     # Individual task cards
│   │   │   └── Header.tsx       # App header
│   │   └── types/task.ts        # TypeScript definitions
│   ├── wailsjs/                 # Auto-generated Wails bindings
│   └── package.json
├── build/bin/                   # Built executables
│   └── task-dashboard.app       # macOS app bundle
└── wails.json                   # Wails configuration
```

### Development Commands (Makefile)
All commands use the Makefile for consistency:

```bash
make help      # List all available commands
make install   # Install frontend dependencies
make dev       # Start development mode (desktop + web at localhost:5173)
make build     # Build production executable
make test      # Run tests
make run       # Run built executable
make web       # Show web testing information
make logs      # View application logs
```

### Data Flow
1. Go backend reads `plan/task.json` on startup
2. React frontend displays tasks as kanban columns (To Do, In Progress, Done)
3. User interactions (drag/drop, edit, approve/reject) trigger Go API calls
4. Changes saved atomically to `plan/task.json` with backup
5. All operations logged to `logs/universal_logs-*.log`

### Web Testing
- **Development**: `make dev` serves both desktop app and web version
- **Web URL**: `http://localhost:5173/` (Vite dev server)
- **Wails Dev Server**: `http://localhost:34115/` (full backend integration)
- **Playwright Testing**: Use `http://localhost:34115/` for complete functionality

### Building and Distribution
- **Single Executable**: `make build` creates `task-dashboard.app` (8.1MB)
- **Platform**: macOS (darwin/arm64) optimized for Apple Silicon
- **Dependencies**: All assets embedded, no external requirements

### Project Level Commands (Makefile)
The Makefile provides all essential commands for development and deployment. All Wails-specific commands are properly configured with PATH handling for Go and Node.js.

### Language and Tool Specific
- **[python]** only use uv, never pip
- **[git]** never mention co-authored-by or similar
- **[git]** never mention the tool used to create the commit message
- **[git]** never use emojis

### UI/UX Philosophy
- **Minimal**: Clean interface without visual clutter
- **Responsive**: Works on various desktop sizes
- **Accessible**: Keyboard shortcuts and clear visual feedback

### Large Codebase Analysis with Gemini CLI

When analyzing large codebases or multiple files that might exceed Claude's context limits, use the Gemini CLI with its massive context window. Use for large refactoring tasks, architecture analysis, complex debugging across many files, project wide document generation, migration planning

```bash
# Single file analysis
gemini -p "@src/main.py Explain this file's purpose and structure"
# Multiple files
gemini -p "@package.json @src/index.js Analyze the dependencies used in the code"
# Entire directory
gemini -p "@src/ Summarize the architecture of this codebase"
# Multiple directories
gemini -p "@src/ @tests/ Analyze test coverage for the source code"
# Current directory and subdirectories
gemini -p "@./ Give me an overview of this entire project"
# Or use --all_files flag for everything
gemini --all_files -p "Analyze the project structure and dependencies"
```

### Development Process Reminders

- **Branch Merge Reminder**: 
  - After merging branches make sure to update the related todo.json task from pending_review to done

### Development Best Practices
- Check if the dev server is running before starting. Use nohup for long running processes including the dev server