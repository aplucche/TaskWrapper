
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
- [x] Install UI dependencies
- [x] Set up project structure

### Phase 2: Backend API (In Progress)
- [ ] Create Go API for task CRUD operations
- [ ] Implement file watching for external changes
- [ ] Add atomic file operations with backups
- [ ] Integrate with universal logging system

### Phase 3: Frontend UI
- [ ] Build kanban board component architecture
- [ ] Implement drag & drop functionality
- [ ] Add task editing and creation
- [ ] Style with modern aesthetic design

### Phase 4: Polish & Testing
- [ ] Add animations and visual feedback
- [ ] Test web mode compatibility
- [ ] Verify Playwright testing capability
- [ ] Build single executable

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