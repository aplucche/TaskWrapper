import React from 'react';
import { Droppable } from '@hello-pangea/dnd';
import { motion } from 'framer-motion';
import { Plus } from 'lucide-react';
import { Task } from '../types/task';
import TaskCard from './TaskCard';

interface ColumnProps {
  status: 'backlog' | 'todo' | 'doing' | 'done';
  title: string;
  tasks: Task[];
  onUpdateTask: (task: Task) => void;
  onDeleteTask: (taskId: number) => void;
  onCreateTask: (title: string) => void;
}

const Column: React.FC<ColumnProps> = ({
  status,
  title,
  tasks,
  onUpdateTask,
  onDeleteTask,
  onCreateTask,
}) => {
  const [isCreating, setIsCreating] = React.useState(false);
  const [newTaskTitle, setNewTaskTitle] = React.useState('');

  const handleCreateTask = () => {
    if (newTaskTitle.trim()) {
      onCreateTask(newTaskTitle.trim());
      setNewTaskTitle('');
      setIsCreating(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleCreateTask();
    } else if (e.key === 'Escape') {
      setNewTaskTitle('');
      setIsCreating(false);
    }
  };

  const getColumnColor = () => {
    switch (status) {
      case 'backlog':
        return 'bg-gray-50 border-gray-200';
      case 'todo':
        return 'bg-blue-50 border-blue-200';
      case 'doing':
        return 'bg-yellow-50 border-yellow-200';
      case 'done':
        return 'bg-green-50 border-green-200';
      default:
        return 'bg-gray-50 border-gray-200';
    }
  };

  const getHeaderColor = () => {
    switch (status) {
      case 'backlog':
        return 'text-gray-700 bg-gray-100';
      case 'todo':
        return 'text-blue-700 bg-blue-100';
      case 'doing':
        return 'text-yellow-700 bg-yellow-100';
      case 'done':
        return 'text-green-700 bg-green-100';
      default:
        return 'text-gray-700 bg-gray-100';
    }
  };

  return (
    <div className={`flex flex-col rounded-lg border-2 border-dashed ${getColumnColor()}`}>
      {/* Column Header */}
      <div className={`px-4 py-3 rounded-t-lg ${getHeaderColor()}`}>
        <div className="flex items-center justify-between">
          <h3 className="font-semibold text-lg">{title}</h3>
          <div className="flex items-center space-x-2">
            <span className="px-2 py-1 text-xs font-medium bg-white bg-opacity-70 rounded-full">
              {tasks.length}
            </span>
            <button
              onClick={() => setIsCreating(true)}
              className="p-1 hover:bg-white hover:bg-opacity-50 rounded-full transition-colors"
              title="Add task"
            >
              <Plus className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>

      {/* Column Content */}
      <Droppable droppableId={status}>
        {(provided, snapshot) => (
          <div
            ref={provided.innerRef}
            {...provided.droppableProps}
            className={`p-4 space-y-3 transition-colors ${
              snapshot.isDraggingOver ? 'bg-opacity-50' : ''
            }`}
          >
            {/* Quick create task input */}
            {isCreating && (
              <motion.div
                initial={{ opacity: 0, y: -10 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -10 }}
                className="p-3 bg-white rounded-lg shadow-soft border border-gray-200"
              >
                <input
                  type="text"
                  value={newTaskTitle}
                  onChange={(e) => setNewTaskTitle(e.target.value)}
                  onKeyDown={handleKeyPress}
                  onBlur={() => {
                    if (!newTaskTitle.trim()) {
                      setIsCreating(false);
                    }
                  }}
                  placeholder="Enter task title..."
                  className="w-full px-0 py-1 text-sm border-0 border-b border-gray-200 focus:outline-none focus:border-primary-500 bg-transparent"
                  autoFocus
                />
                <div className="flex items-center justify-end space-x-2 mt-2">
                  <button
                    onClick={() => {
                      setNewTaskTitle('');
                      setIsCreating(false);
                    }}
                    className="px-2 py-1 text-xs text-gray-500 hover:text-gray-700"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleCreateTask}
                    disabled={!newTaskTitle.trim()}
                    className="px-3 py-1 text-xs font-medium text-white bg-primary-600 rounded hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Add
                  </button>
                </div>
              </motion.div>
            )}

            {/* Tasks */}
            {tasks.map((task, index) => (
              <TaskCard
                key={task.id}
                task={task}
                index={index}
                onUpdateTask={onUpdateTask}
                onDeleteTask={onDeleteTask}
              />
            ))}

            {provided.placeholder}

            {/* Empty state */}
            {tasks.length === 0 && !isCreating && (
              <div className="flex flex-col items-center justify-center py-12 text-gray-400">
                <div className="w-16 h-16 mb-4 rounded-full bg-gray-100 flex items-center justify-center">
                  <Plus className="w-8 h-8" />
                </div>
                <p className="text-sm text-center">
                  No tasks yet.<br />
                  <button
                    onClick={() => setIsCreating(true)}
                    className="text-primary-600 hover:text-primary-700 font-medium"
                  >
                    Add one
                  </button>
                </p>
              </div>
            )}
          </div>
        )}
      </Droppable>
    </div>
  );
};

export default Column;