import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { LayoutDashboard, FileText, Terminal, Settings } from 'lucide-react';
import KanbanBoard from './components/KanbanBoard';
import PlanView from './components/PlanView';
import CodeView from './components/CodeView';
import SettingsView from './components/SettingsView';
import RepositorySwitcher from './components/RepositorySwitcher';
import { GetConfig } from '../wailsjs/go/main/App';

type ViewType = 'tasks' | 'plan' | 'code' | 'settings';

function App() {
    const [currentView, setCurrentView] = useState<ViewType>('settings'); // Default to settings
    const [lastSaved, setLastSaved] = useState<Date | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [hasValidRepository, setHasValidRepository] = useState<boolean>(false);
    const [loading, setLoading] = useState<boolean>(true);

    const handleSave = () => {
        setLastSaved(new Date());
    };

    useEffect(() => {
        checkRepositoryStatus();
        
        // Listen for repository changes from settings
        const handleRepositoriesChanged = () => {
            checkRepositoryStatus();
        };
        
        window.addEventListener('repositoriesChanged', handleRepositoriesChanged);
        
        return () => {
            window.removeEventListener('repositoriesChanged', handleRepositoriesChanged);
        };
    }, []);

    const checkRepositoryStatus = async () => {
        try {
            const config = await GetConfig();
            // Valid if we have configured repos OR we're in a repo (not the "No Repository" fallback)
            const hasConfiguredRepos = config.repositories && config.repositories.length > 1;
            const hasValidRepo = config.repositories && config.repositories.length > 0 && 
                                config.repositories.some(repo => repo.name !== "No Repository");
            
            const isValid = hasConfiguredRepos || hasValidRepo;
            setHasValidRepository(isValid);
            
            // If we have a valid setup, default to tasks view
            if (isValid) {
                setCurrentView('tasks');
            }
        } catch (err) {
            console.error('Failed to check repository status:', err);
            setHasValidRepository(false);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="h-screen flex flex-col bg-gray-50">
            {/* Header with navigation */}
            <header className="bg-white border-b border-gray-200">
                <div className="px-6 py-4">
                    <div className="flex items-center justify-between">
                        <div className="flex items-center space-x-6">
                            <div className="flex items-center space-x-4">
                                <h1 className="text-2xl font-bold text-gray-900">TaskWrapper</h1>
                                <RepositorySwitcher />
                            </div>
                            
                            {/* Navigation tabs */}
                            <nav className="flex space-x-1">
                                <button
                                    onClick={() => hasValidRepository && setCurrentView('tasks')}
                                    disabled={!hasValidRepository}
                                    className={`flex items-center space-x-2 px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                                        !hasValidRepository
                                            ? 'text-gray-400 cursor-not-allowed'
                                            : currentView === 'tasks'
                                            ? 'bg-primary-100 text-primary-700'
                                            : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                                    }`}
                                >
                                    <LayoutDashboard className="w-4 h-4" />
                                    <span>Tasks</span>
                                </button>
                                <button
                                    onClick={() => hasValidRepository && setCurrentView('plan')}
                                    disabled={!hasValidRepository}
                                    className={`flex items-center space-x-2 px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                                        !hasValidRepository
                                            ? 'text-gray-400 cursor-not-allowed'
                                            : currentView === 'plan'
                                            ? 'bg-primary-100 text-primary-700'
                                            : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                                    }`}
                                >
                                    <FileText className="w-4 h-4" />
                                    <span>Plan</span>
                                </button>
                                <button
                                    onClick={() => setCurrentView('code')}
                                    className={`flex items-center space-x-2 px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                                        currentView === 'code'
                                            ? 'bg-primary-100 text-primary-700'
                                            : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                                    }`}
                                >
                                    <Terminal className="w-4 h-4" />
                                    <span>Code</span>
                                </button>
                                <button
                                    onClick={() => setCurrentView('settings')}
                                    className={`flex items-center space-x-2 px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                                        currentView === 'settings'
                                            ? 'bg-primary-100 text-primary-700'
                                            : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                                    }`}
                                >
                                    <Settings className="w-4 h-4" />
                                    <span>Settings</span>
                                </button>
                            </nav>
                        </div>
                        
                        {/* Page-specific actions will be rendered here by each page */}
                        <div id="page-actions"></div>
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
                    ) : currentView === 'plan' ? (
                        <PlanView onError={setError} onSave={handleSave} />
                    ) : currentView === 'code' ? (
                        <CodeView />
                    ) : (
                        <SettingsView />
                    )}
                </motion.div>
            </main>
        </div>
    );
}

export default App;
