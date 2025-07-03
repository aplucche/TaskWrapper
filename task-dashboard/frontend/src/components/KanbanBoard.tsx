import React, { useState, useEffect } from 'react';
import { DragDropContext, DropResult } from '@hello-pangea/dnd';
import { motion } from 'framer-motion';
import { Task, STATUS_LABELS } from '../types/task';
import { LoadTasks, SaveTasks, MoveTask } from '../../wailsjs/go/main/App';
import Column from './Column';
import Header from './Header';

const KanbanBoard: React.FC = () => {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastSaved, setLastSaved] = useState<Date | null>(null);

  // Load tasks on component mount
  useEffect(() => {
    loadTasks();
  }, []);

  const loadTasks = async () => {
    try {
      setLoading(true);
      setError(null);
      const loadedTasks = await LoadTasks();
      setTasks(loadedTasks || []);
    } catch (err) {
      setError(`Failed to load tasks: ${err}`);
      console.error('Error loading tasks:', err);
    } finally {
      setLoading(false);
    }
  };

  const saveTasks = async (updatedTasks: Task[]) => {
    try {
      await SaveTasks(updatedTasks);
      setTasks(updatedTasks);
      setLastSaved(new Date());
      setError(null);
    } catch (err) {
      setError(`Failed to save tasks: ${err}`);
      console.error('Error saving tasks:', err);
    }
  };

  const handleDragEnd = async (result: DropResult) => {
    if (!result.destination) return;

    const { source, destination, draggableId } = result;
    
    // If dropped in the same position, do nothing
    if (
      source.droppableId === destination.droppableId &&
      source.index === destination.index
    ) {
      return;
    }

    const taskId = parseInt(draggableId);
    const newStatus = destination.droppableId as 'todo' | 'doing' | 'done';

    try {
      // Update task status via backend
      await MoveTask(taskId, newStatus);
      
      // Update local state
      const updatedTasks = tasks.map(task =>
        task.id === taskId ? { ...task, status: newStatus } : task
      );
      
      setTasks(updatedTasks);
      setLastSaved(new Date());
      setError(null);
    } catch (err) {
      setError(`Failed to move task: ${err}`);
      console.error('Error moving task:', err);
    }
  };

  const updateTask = async (updatedTask: Task) => {
    const updatedTasks = tasks.map(task =>
      task.id === updatedTask.id ? updatedTask : task
    );
    await saveTasks(updatedTasks);
  };

  const createTask = async (title: string, status: 'todo' | 'doing' | 'done' = 'todo') => {
    const maxId = tasks.reduce((max, task) => Math.max(max, task.id), 0);
    const newTask: Task = {
      id: maxId + 1,
      title,
      status,
      priority: 'medium',
      deps: [],
      parent: null,
    };

    const updatedTasks = [...tasks, newTask];
    await saveTasks(updatedTasks);
  };

  const deleteTask = async (taskId: number) => {
    const updatedTasks = tasks.filter(task => task.id !== taskId);
    await saveTasks(updatedTasks);
  };

  // Group tasks by status
  const tasksByStatus = {
    todo: tasks.filter(task => task.status === 'todo'),
    doing: tasks.filter(task => task.status === 'doing'),
    done: tasks.filter(task => task.status === 'done'),
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <motion.div
          animate={{ rotate: 360 }}
          transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
          className="w-8 h-8 border-2 border-primary-500 border-t-transparent rounded-full"
        />
        <span className="ml-3 text-gray-600">Loading tasks...</span>
      </div>
    );
  }

  return (
    <div className="h-screen flex flex-col bg-gray-50">
      <Header 
        lastSaved={lastSaved} 
        error={error} 
        onRefresh={loadTasks}
        onCreateTask={createTask}
      />
      
      <main className="flex-1 p-6 overflow-hidden">
        <DragDropContext onDragEnd={handleDragEnd}>
          <div className="grid grid-cols-3 gap-6 h-full">
            {Object.entries(STATUS_LABELS).map(([status, label]) => (
              <Column
                key={status}
                status={status as 'todo' | 'doing' | 'done'}
                title={label}
                tasks={tasksByStatus[status as keyof typeof tasksByStatus]}
                onUpdateTask={updateTask}
                onDeleteTask={deleteTask}
                onCreateTask={(title) => createTask(title, status as 'todo' | 'doing' | 'done')}
              />
            ))}
          </div>
        </DragDropContext>
      </main>
    </div>
  );
};

export default KanbanBoard;