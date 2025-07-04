import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { LayoutDashboard, FileText } from 'lucide-react';
import KanbanBoard from './components/KanbanBoard';
import PlanView from './components/PlanView';

type ViewType = 'tasks' | 'plan';

function App() {
    const [currentView, setCurrentView] = useState<ViewType>('tasks');
    const [lastSaved, setLastSaved] = useState<Date | null>(null);
    const [error, setError] = useState<string | null>(null);

    const handleSave = () => {
        setLastSaved(new Date());
    };

    return (
        <div className="h-screen flex flex-col bg-gray-50">
            {/* Header with navigation */}
            <header className="bg-white border-b border-gray-200">
                <div className="px-6 py-4">
                    <div className="flex items-center justify-between">
                        <h1 className="text-2xl font-bold text-gray-900">Task Dashboard</h1>
                        
                        {/* Navigation tabs */}
                        <nav className="flex space-x-1">
                            <button
                                onClick={() => setCurrentView('tasks')}
                                className={`flex items-center space-x-2 px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                                    currentView === 'tasks'
                                        ? 'bg-primary-100 text-primary-700'
                                        : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                                }`}
                            >
                                <LayoutDashboard className="w-4 h-4" />
                                <span>Tasks</span>
                            </button>
                            <button
                                onClick={() => setCurrentView('plan')}
                                className={`flex items-center space-x-2 px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                                    currentView === 'plan'
                                        ? 'bg-primary-100 text-primary-700'
                                        : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                                }`}
                            >
                                <FileText className="w-4 h-4" />
                                <span>Plan</span>
                            </button>
                        </nav>
                    </div>
                </div>
            </header>

            {/* Main content */}
            <main className="flex-1 overflow-hidden">
                <motion.div
                    key={currentView}
                    initial={{ opacity: 0, x: 20 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: -20 }}
                    transition={{ duration: 0.2 }}
                    className="h-full"
                >
                    {currentView === 'tasks' ? (
                        <KanbanBoard />
                    ) : (
                        <PlanView onError={setError} onSave={handleSave} />
                    )}
                </motion.div>
            </main>
        </div>
    );
}

export default App;
