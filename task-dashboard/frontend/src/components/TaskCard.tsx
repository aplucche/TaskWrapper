import React, { useState } from 'react';
import { Draggable } from '@hello-pangea/dnd';
import { MoreVertical, Edit2, Trash2, Save, X, AlertCircle } from 'lucide-react';
import { Menu, Transition } from '@headlessui/react';
import { Fragment } from 'react';
import { Task, PRIORITY_COLORS } from '../types/task';

interface TaskCardProps {
  task: Task;
  index: number;
  onUpdateTask: (task: Task) => void;
  onDeleteTask: (taskId: number) => void;
}

const CARD_STYLES = {
  base: 'group bg-white rounded-lg shadow-soft border border-gray-200 p-4 hover:shadow-medium transition-all duration-200',
  dragging: 'dragging shadow-large',
  subTask: 'ml-4 border-l-4 border-l-blue-300',
} as const;

const TaskCard: React.FC<TaskCardProps> = ({ task, index, onUpdateTask, onDeleteTask }) => {
  const [isEditing, setIsEditing] = useState(false);
  const [editTitle, setEditTitle] = useState(task.title);
  const [editPriority, setEditPriority] = useState(task.priority);

  const handleSave = () => {
    if (editTitle.trim()) {
      onUpdateTask({
        ...task,
        title: editTitle.trim(),
        priority: editPriority,
      });
      setIsEditing(false);
    }
  };

  const handleCancel = () => {
    setEditTitle(task.title);
    setEditPriority(task.priority);
    setIsEditing(false);
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSave();
    } else if (e.key === 'Escape') {
      handleCancel();
    }
  };

  const getPriorityIcon = () => {
    if (task.priority === 'high') {
      return <AlertCircle className="w-3 h-3" />;
    }
    return null;
  };

  const hasDependencies = task.deps?.length > 0;
  const isSubTask = task.parent != null;

  return (
    <Draggable draggableId={task.id.toString()} index={index}>
      {(provided, snapshot) => (
        <div
          ref={provided.innerRef}
          {...provided.draggableProps}
          {...provided.dragHandleProps}
          className={`${CARD_STYLES.base} ${
            snapshot.isDragging ? CARD_STYLES.dragging : ''
          } ${isSubTask ? CARD_STYLES.subTask : ''}`}
        >
          <div className="flex items-start justify-between">
            <div className="flex-1 min-w-0 overflow-hidden">
              {isEditing ? (
                <div className="space-y-3">
                  <textarea
                    value={editTitle}
                    onChange={(e) => setEditTitle(e.target.value)}
                    onKeyDown={handleKeyPress}
                    className="w-full p-2 text-sm border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-none"
                    rows={2}
                    autoFocus
                  />
                  
                  <div className="flex items-center space-x-3">
                    <select
                      value={editPriority}
                      onChange={(e) => setEditPriority(e.target.value as 'high' | 'medium' | 'low')}
                      className="text-xs border border-gray-300 rounded px-2 py-1 focus:outline-none focus:ring-1 focus:ring-primary-500"
                    >
                      <option value="high">High</option>
                      <option value="medium">Medium</option>
                      <option value="low">Low</option>
                    </select>
                    
                    <div className="flex items-center space-x-1 ml-auto">
                      <button
                        onClick={handleSave}
                        disabled={!editTitle.trim()}
                        className="p-1 text-green-600 hover:text-green-700 disabled:opacity-50 disabled:cursor-not-allowed"
                        title="Save"
                      >
                        <Save className="w-4 h-4" />
                      </button>
                      <button
                        onClick={handleCancel}
                        className="p-1 text-gray-500 hover:text-gray-700"
                        title="Cancel"
                      >
                        <X className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                </div>
              ) : (
                <>
                  {/* Pending Review Header */}
                  {task.status === 'pending_review' && (
                    <div className="mb-2 px-2 py-1 bg-purple-100 border border-purple-200 rounded text-xs text-purple-700 font-medium">
                      🔍 Pending Review
                    </div>
                  )}
                  <div className="flex items-start justify-between mb-2">
                    <h4 className="text-sm font-medium text-gray-900 group-hover:text-gray-700 leading-tight break-words pr-2 flex-1">
                      {task.title}
                    </h4>
                    
                    <Menu as="div" className="relative opacity-0 group-hover:opacity-100 transition-opacity">
                      <Menu.Button className="p-1 hover:bg-gray-100 rounded">
                        <MoreVertical className="w-4 h-4 text-gray-400" />
                      </Menu.Button>
                      
                      <Transition
                        as={Fragment}
                        enter="transition ease-out duration-100"
                        enterFrom="transform opacity-0 scale-95"
                        enterTo="transform opacity-100 scale-100"
                        leave="transition ease-in duration-75"
                        leaveFrom="transform opacity-100 scale-100"
                        leaveTo="transform opacity-0 scale-95"
                      >
                        <Menu.Items className="absolute right-0 top-full mt-1 z-50 w-32 bg-white rounded-md shadow-xl border border-gray-200 focus:outline-none">
                        <Menu.Item>
                          {({ active }) => (
                            <button
                              onClick={() => setIsEditing(true)}
                              className={`${
                                active ? 'bg-gray-50' : ''
                              } flex items-center space-x-2 w-full px-3 py-2 text-sm text-gray-700`}
                            >
                              <Edit2 className="w-3 h-3" />
                              <span>Edit</span>
                            </button>
                          )}
                        </Menu.Item>
                        <Menu.Item>
                          {({ active }) => (
                            <button
                              onClick={() => onDeleteTask(task.id)}
                              className={`${
                                active ? 'bg-red-50 text-red-700' : 'text-red-600'
                              } flex items-center space-x-2 w-full px-3 py-2 text-sm`}
                            >
                              <Trash2 className="w-3 h-3" />
                              <span>Delete</span>
                            </button>
                          )}
                        </Menu.Item>
                        </Menu.Items>
                      </Transition>
                    </Menu>
                  </div>

                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-2">
                      {/* Priority badge */}
                      <span className={`inline-flex items-center space-x-1 px-2 py-1 text-xs font-medium rounded-full border ${PRIORITY_COLORS[task.priority]}`}>
                        {getPriorityIcon()}
                        <span className="capitalize">{task.priority}</span>
                      </span>
                      
                      {/* Task ID */}
                      <span className="text-xs text-gray-400">#{task.id}</span>
                    </div>

                    <div className="flex items-center space-x-1">
                      {/* Dependencies indicator */}
                      {hasDependencies && (
                        <span
                          className="text-xs text-blue-600 bg-blue-100 px-2 py-1 rounded-full"
                          title={`Depends on: ${task.deps.join(', ')}`}
                        >
                          {task.deps.length} dep{task.deps.length !== 1 ? 's' : ''}
                        </span>
                      )}
                      
                      {/* Sub-task indicator */}
                      {isSubTask && (
                        <span className="text-xs text-purple-600 bg-purple-100 px-2 py-1 rounded-full">
                          Sub
                        </span>
                      )}
                    </div>
                  </div>
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </Draggable>
  );
};

export default TaskCard;