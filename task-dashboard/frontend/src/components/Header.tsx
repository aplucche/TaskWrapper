import React from 'react';
import { motion } from 'framer-motion';
import { RefreshCw, Plus, AlertCircle, CheckCircle2, EyeOff, Eye } from 'lucide-react';

interface HeaderProps {
  lastSaved: Date | null;
  error: string | null;
  onRefresh: () => void;
  onCreateTask: (title: string) => void;
  hideComplete: boolean;
  onToggleHideComplete: () => void;
}

const Header: React.FC<HeaderProps> = ({ lastSaved, error, onRefresh, onCreateTask, hideComplete, onToggleHideComplete }) => {
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

  const formatLastSaved = (date: Date) => {
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMinutes = Math.floor(diffMs / 60000);

    if (diffMinutes < 1) return 'Just now';
    if (diffMinutes < 60) return `${diffMinutes}m ago`;
    
    const diffHours = Math.floor(diffMinutes / 60);
    if (diffHours < 24) return `${diffHours}h ago`;
    
    return date.toLocaleDateString();
  };

  return (
    <header className="bg-white border-b border-gray-200 px-6 py-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <h1 className="text-2xl font-bold text-gray-900">Task Dashboard</h1>
          
          {/* Status indicator */}
          <div className="flex items-center space-x-2">
            {error ? (
              <motion.div
                initial={{ scale: 0 }}
                animate={{ scale: 1 }}
                className="flex items-center space-x-2 text-red-600"
              >
                <AlertCircle className="w-4 h-4" />
                <span className="text-sm font-medium">Error</span>
              </motion.div>
            ) : lastSaved ? (
              <motion.div
                initial={{ scale: 0 }}
                animate={{ scale: 1 }}
                className="flex items-center space-x-2 text-green-600"
              >
                <CheckCircle2 className="w-4 h-4" />
                <span className="text-sm text-gray-500">
                  Saved {formatLastSaved(lastSaved)}
                </span>
              </motion.div>
            ) : null}
          </div>
        </div>

        <div className="flex items-center space-x-3">
          {/* Hide Complete Toggle */}
          <button
            onClick={onToggleHideComplete}
            className={`flex items-center space-x-2 px-3 py-2 text-sm font-medium rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 ${
              hideComplete
                ? 'text-primary-700 bg-primary-50 border border-primary-200 hover:bg-primary-100'
                : 'text-gray-700 bg-white border border-gray-300 hover:bg-gray-50'
            }`}
          >
            {hideComplete ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
            <span>Hide Complete</span>
          </button>

          {/* Quick create task */}
          {isCreating ? (
            <motion.div
              initial={{ width: 0, opacity: 0 }}
              animate={{ width: 200, opacity: 1 }}
              exit={{ width: 0, opacity: 0 }}
              className="flex items-center space-x-2"
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
                placeholder="Task title..."
                className="px-3 py-1 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                autoFocus
              />
              <button
                onClick={handleCreateTask}
                disabled={!newTaskTitle.trim()}
                className="px-3 py-1 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Add
              </button>
            </motion.div>
          ) : (
            <button
              onClick={() => setIsCreating(true)}
              className="flex items-center space-x-2 px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2"
            >
              <Plus className="w-4 h-4" />
              <span>Add Task</span>
            </button>
          )}

          {/* Refresh button */}
          <button
            onClick={onRefresh}
            className="flex items-center space-x-2 px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2"
          >
            <RefreshCw className="w-4 h-4" />
            <span>Refresh</span>
          </button>
        </div>
      </div>

      {/* Error message */}
      {error && (
        <motion.div
          initial={{ height: 0, opacity: 0 }}
          animate={{ height: 'auto', opacity: 1 }}
          exit={{ height: 0, opacity: 0 }}
          className="mt-4 p-4 bg-red-50 border border-red-200 rounded-md"
        >
          <div className="flex items-center space-x-2">
            <AlertCircle className="w-5 h-5 text-red-500" />
            <p className="text-sm text-red-700">{error}</p>
          </div>
        </motion.div>
      )}
    </header>
  );
};

export default Header;