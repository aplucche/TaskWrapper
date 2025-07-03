// Re-export the Wails-generated Task type for consistency
export type { main as WailsTypes } from '../../wailsjs/go/models';
export type Task = import('../../wailsjs/go/models').main.Task;

export interface TaskColumn {
  status: 'backlog' | 'todo' | 'doing' | 'done';
  title: string;
  tasks: Task[];
}

export interface KanbanBoard {
  columns: TaskColumn[];
}

export const PRIORITY_COLORS: Record<string, string> = {
  high: 'bg-red-100 text-red-800 border-red-200',
  medium: 'bg-orange-100 text-orange-800 border-orange-200',
  low: 'bg-green-100 text-green-800 border-green-200',
};

export const STATUS_LABELS = {
  todo: 'To Do',
  doing: 'In Progress',
  done: 'Done',
  backlog: 'Backlog',
} as const;