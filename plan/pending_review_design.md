# Pending Review Status Design

## Overview
Add a "pending_review" status to create a proper review workflow for Claude agent completions.

## Current Workflow
1. User moves task from "todo" → "doing"
2. Claude agent launches and works on task
3. Agent updates status to "done" and commits to branch
4. User manually reviews and merges

## New Workflow
1. User moves task from "todo" → "doing"
2. Claude agent launches and works on task
3. **Agent updates status to "pending_review"** (not "done")
4. Task appears in new "Pending Review" column
5. User reviews the branch:
   - **Accept**: Move to "done" and merge branch
   - **Reject**: Create subtasks for improvements, move back to "todo"

## Required Changes

### 1. Backend (Go)
- Update status validation to include "pending_review"
- Update all status checks to handle new status
- Ensure MoveTask() supports the new status

### 2. Frontend (React)
- Add "Pending Review" column to kanban board
- Update drag-and-drop to support new column
- Add visual indicators (e.g., branch icon, review badge)

### 3. Claude Agent Prompts
Change from:
```
Update task.json status to 'done' when complete, commit to branch task_[ID], then exit.
```

To:
```
Update task.json status to 'pending_review' when complete, commit to branch task_[ID], then exit.
```

### 4. Review Actions
- **Accept**: 
  - Move task to "done"
  - Optionally trigger merge of task branch
  - Log acceptance in universal logs
  
- **Reject**:
  - Create subtasks based on review feedback
  - Move original task back to "todo" or "backlog"
  - Keep branch for reference
  - Log rejection reason in universal logs

## Benefits
1. **Clear separation** between agent completion and human approval
2. **Better visibility** of tasks awaiting review
3. **Structured feedback loop** through subtask creation
4. **Audit trail** of acceptances and rejections
5. **Quality control** before merging agent work

## Future Enhancements
- Auto-generate PR descriptions from task context
- Add review comments/notes field
- Track review metrics (acceptance rate, review time)
- Batch review operations
- Integration with GitHub PRs