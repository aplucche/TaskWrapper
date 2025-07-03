# CLAUDE.md

This file provides guidance to Claude Code when working in this repository.

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

### Design Philosophy
- Prefer minimal and decoupled solutions. Lean on effective high leverage data structures to guide development
- Favor extensible patterns over precise outcomes. A design that can grow and adapt is preferred to a flawless one-shot solution.
- Test extensively and intelligently, prune ununsed code aggressively.


### Project level commands (Makefile)
- Important project wide commands should tie to the Makefile for test, run and build as appropriate
- E.g. if `npm run dev` is used it should be triggerable from the Makefile

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