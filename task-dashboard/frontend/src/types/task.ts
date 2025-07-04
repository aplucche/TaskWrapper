// Task type from Wails-generated models
export type Task = import('../../wailsjs/go/models').main.Task;

export interface TaskColumn {
  status: 'backlog' | 'todo' | 'doing' | 'pending_review' | 'done';
  title: string;
  tasks: Task[];
}

export const PRIORITY_COLORS: Record<string, string> = {
  high: 'bg-red-100 text-red-800 border-red-200',
  medium: 'bg-orange-100 text-orange-800 border-orange-200',
  low: 'bg-green-100 text-green-800 border-green-200',
};

export const STATUS_LABELS = {
  backlog: 'Backlog',
  todo: 'To Do',
  doing: 'In Progress',
  done: 'Done',
} as const;