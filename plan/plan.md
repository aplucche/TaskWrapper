
# Task Dashboard Project Plan

## Project Overview
A standalone desktop application built with Wails (Go + React + TypeScript) that provides an editable kanban board interface for the project's task.json file. The application serves as a visual task management dashboard with drag-and-drop functionality, modern aesthetic design, and dual-mode operation (desktop app and web browser for testing).

## Requirements

### Core Features
- **Task Management**: Read/write operations on `plan/task.json`
- **Kanban Board**: Visual representation with columns (To Do, In Progress, Done)
- **Drag & Drop**: Move tasks between columns with smooth animations
- **Real-time Updates**: File watching and auto-save functionality
- **Web Testing**: Accessible via browser at `localhost:34115` for Playwright testing

### Technical Requirements
- **Frontend**: React + TypeScript with modern UI components
- **Backend**: Go with Wails framework for desktop integration
- **Build**: Single executable binary
- **Testing**: Web-compatible for automated testing
- **Aesthetics**: Modern, clean design with smooth animations

## Architecture

### Technology Stack
- **Desktop Framework**: Wails v2
- **Backend**: Go (embedded in Wails)
- **Frontend**: React 18 + TypeScript + Vite
- **UI Components**: Tailwind CSS, Headless UI, Framer Motion
- **Drag & Drop**: @hello-pangea/dnd
- **Icons**: Lucide React, Heroicons

### Data Flow
1. Go backend reads `plan/task.json` on startup
2. React frontend displays tasks as kanban columns
3. User interactions (drag/drop, edit) trigger Go API calls
4. Changes saved atomically to `plan/task.json`
5. File watching detects external changes
6. All operations logged to `logs/universal_logs-*.log`

### Project Structure
```
task-dashboard/
├── main.go              # Wails app entry point
├── app.go               # Go backend API
├── frontend/
│   ├── src/
│   │   ├── App.tsx      # Main React component
│   │   ├── components/  # Kanban board components
│   │   └── types/       # TypeScript definitions
│   └── package.json
└── wails.json           # Wails configuration
```

## Milestones

### Phase 1: Foundation ✅
- [x] Initialize Wails project with React TypeScript
- [x] Update Makefile with proper commands
- [x] Install UI dependencies (Tailwind, Headless UI, Framer Motion, etc.)
- [x] Set up project structure and configure Tailwind CSS

### Phase 2: Backend API ✅
- [x] Create Go API for task CRUD operations
- [x] Implement LoadTasks(), SaveTasks(), UpdateTask(), MoveTask()
- [x] Add atomic file operations with backup system
- [x] Integrate with universal logging system
- [x] Thread-safe operations with mutex protection
- [x] Task validation and error handling

### Phase 3: Frontend UI ✅
- [x] Build kanban board component architecture
- [x] Create KanbanBoard, Column, TaskCard, Header components
- [x] Implement drag & drop functionality with @hello-pangea/dnd
- [x] Add task editing and creation UI
- [x] Style with modern aesthetic design (Tailwind CSS)
- [x] Add task priority visualization and metadata
- [x] Implement error handling and loading states

### Phase 4: Polish & Testing ✅
- [x] Wails TypeScript bindings for Go API
- [x] Add animations and visual feedback
- [x] Test web mode compatibility
- [x] Verify Playwright testing capability
- [x] Build single executable
- [x] Fix path resolution for standalone app
- [ ] File watching for external changes (optional enhancement)

### Phase 5: Testing Infrastructure 🚧
- [ ] Document testing strategy and approach
- [ ] Assess current toolchain for testing compatibility  
- [x] Implement Go backend test suite (8 comprehensive tests)
- [ ] Setup Playwright E2E infrastructure
- [ ] Implement core E2E workflow tests
- [ ] Add testing commands to Makefile

## Risks

### Technical Risks
- **File Concurrency**: Multiple processes accessing task.json simultaneously
- **Cross-platform**: Ensuring consistent behavior across OS
- **Performance**: Large task lists affecting UI responsiveness

### Mitigations
- Implement file locking and atomic operations
- Test on target platforms early
- Virtualize large lists if needed

## Parking Lot (Future Ideas)

### Enhancements
- Task dependencies visualization
- Due date tracking and reminders
- Task filtering and search
- Export to other formats (CSV, PDF)
- Team collaboration features
- Dark/light theme toggle
- Keyboard shortcuts
- Task time tracking

### Integration Ideas
- GitHub Issues synchronization
- Calendar integration
- Slack/Discord notifications
- CI/CD pipeline integration

## Troubleshooting & Debugging

### Common Issues

#### App Won't Open on macOS
- **"App is damaged or incomplete"**: Run `xattr -d com.apple.quarantine path/to/task-dashboard.app`
- **Missing executable**: Clean rebuild with `rm -rf build/ && make build`
- **Permission denied**: Check if app bundle has proper executable permissions

#### Task Loading Issues
- **"Failed to load tasks"**: Check file paths in logs at `logs/universal_logs-*.log`
- **Permission denied on directories**: App creates fallback in `~/Documents/TaskDashboard/`
- **Corrupted task.json**: Check backup files with `.backup.YYYYMMDD_HHMMSS` extension

#### Development Issues
- **Frontend won't compile**: Check Node.js version (requires 15+), run `npm install` in frontend/
- **Go compilation errors**: Verify Go 1.18+ and PATH includes `/usr/local/go/bin`
- **Wails binding issues**: Delete `frontend/wailsjs/` and run `wails dev` to regenerate

### Debugging Tools

#### Logging
- **Location**: `logs/universal_logs-YYYY-MM-DD.log`
- **View live**: `make logs` or `tail -f logs/universal_logs-*.log`
- **Levels**: INFO (normal operations), ERROR (failures)

#### Development Commands
```bash
make dev          # Start with hot reload + web access
make build        # Production build
make test         # Run all tests
wails doctor      # Check system requirements
go build -v       # Test Go compilation
npm run build     # Test frontend compilation
```

#### Web Development
- **URL**: `http://localhost:5173/` during `make dev`
- **Browser DevTools**: Full access to React DevTools
- **Network inspection**: Monitor API calls to Go backend
- **Console logging**: Frontend errors and state changes

### File Locations

#### Development
- **Task file**: `plan/task.json` (in project root)
- **Logs**: `logs/universal_logs-*.log`
- **Backups**: `plan/task.json.backup.*`

#### Standalone App
- **Primary**: `~/repos/cc_task_dash/plan/task.json` (if exists)
- **Fallback**: `~/Documents/TaskDashboard/task.json`
- **Logs**: Adjacent to task file in `logs/` subdirectory

## Future Enhancements

### Priority Features
- **File watching**: Auto-refresh when task.json changes externally
- **Undo/Redo**: Task operation history with Ctrl+Z/Ctrl+Y
- **Search & Filter**: Find tasks by title, priority, or status
- **Bulk operations**: Select multiple tasks for batch updates
- **Export options**: PDF reports, CSV exports, JSON backup

### UI/UX Improvements
- **Dark mode**: Toggle between light/dark themes
- **Keyboard shortcuts**: Arrow key navigation, quick task creation
- **Column customization**: Reorder columns, custom statuses
- **Task templates**: Predefined task structures
- **Progress visualization**: Charts and completion statistics

### Advanced Features
- **Time tracking**: Start/stop timers for tasks
- **Dependencies visualization**: Graphical dependency tree
- **Team collaboration**: Multi-user support with conflict resolution
- **Integration APIs**: REST endpoints for external tools
- **Custom fields**: User-defined task properties

### Technical Improvements
- **Database backend**: Optional SQLite/PostgreSQL support
- **Sync services**: Cloud backup (iCloud, Dropbox, Google Drive)
- **Plugin system**: Extensible architecture for custom features
- **Multi-platform**: Windows and Linux builds
- **Mobile companion**: React Native or web responsive design

### Performance Optimizations
- **Virtual scrolling**: Handle thousands of tasks efficiently
- **Lazy loading**: Load task details on demand
- **Caching layer**: Reduce file I/O with smart caching
- **Compression**: Smaller app bundle with UPX compression