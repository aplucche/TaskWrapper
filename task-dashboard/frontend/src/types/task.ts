export interface Task {
  id: number;
  title: string;
  status: 'todo' | 'doing' | 'done';
  priority: 'high' | 'medium' | 'low';
  deps: number[];
  parent: number | null;
}

export interface TaskColumn {
  status: 'todo' | 'doing' | 'done';
  title: string;
  tasks: Task[];
}

export interface KanbanBoard {
  columns: TaskColumn[];
}

export const PRIORITY_COLORS = {
  high: 'bg-red-100 text-red-800 border-red-200',
  medium: 'bg-orange-100 text-orange-800 border-orange-200',
  low: 'bg-green-100 text-green-800 border-green-200',
} as const;

export const STATUS_LABELS = {
  todo: 'To Do',
  doing: 'In Progress',
  done: 'Done',
} as const;