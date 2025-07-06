import React, { useState, useEffect } from 'react';
import { ChevronDown, GitBranch } from 'lucide-react';
import { GetRepositories, SetActiveRepository } from '../../wailsjs/go/main/App';
import { Repository } from '../types/config';

const RepositorySwitcher: React.FC = () => {
    const [repositories, setRepositories] = useState<Repository[]>([]);
    const [activeRepoId, setActiveRepoId] = useState<string>('');
    const [loading, setLoading] = useState(true);
    const [switching, setSwitching] = useState(false);

    useEffect(() => {
        loadRepositories();
        
        // Listen for repository changes
        const handleRepositoriesChanged = () => {
            loadRepositories();
        };
        
        window.addEventListener('repositoriesChanged', handleRepositoriesChanged);
        
        return () => {
            window.removeEventListener('repositoriesChanged', handleRepositoriesChanged);
        };
    }, []);

    const loadRepositories = async () => {
        try {
            const repos = await GetRepositories();
            setRepositories(repos);
            
            // Get config to find active repository
            const { GetConfig } = await import('../../wailsjs/go/main/App');
            const config = await GetConfig();
            
            // Find the active repository ID
            const activeRepo = repos.find(r => r.path === config.activeRepository);
            if (activeRepo) {
                setActiveRepoId(activeRepo.id);
            } else if (repos.length > 0) {
                setActiveRepoId(repos[0].id);
            }
        } catch (err) {
            console.error('Failed to load repositories:', err);
        } finally {
            setLoading(false);
        }
    };

    const handleSwitchRepository = async (id: string) => {
        if (id === activeRepoId || switching) return;

        try {
            setSwitching(true);
            await SetActiveRepository(id);
            setActiveRepoId(id);
            
            // Reload the entire app to switch context
            window.location.reload();
        } catch (err) {
            console.error('Failed to switch repository:', err);
        } finally {
            setSwitching(false);
        }
    };

    // Only show switcher if there are multiple repositories
    if (loading || repositories.length <= 1) {
        return null;
    }

    const activeRepo = repositories.find(r => r.id === activeRepoId);
    if (!activeRepo) return null;

    return (
        <div className="relative group">
            <button
                className="flex items-center space-x-2 px-3 py-1.5 text-sm text-gray-700 hover:text-gray-900 hover:bg-gray-100 rounded-md transition-colors"
                disabled={switching}
            >
                <GitBranch className="w-4 h-4" />
                <span className="font-medium">{activeRepo.name}</span>
                <ChevronDown className="w-3 h-3" />
            </button>

            <div className="absolute top-full left-0 mt-1 w-64 bg-white border border-gray-200 rounded-md shadow-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200">
                <div className="py-1">
                    {repositories.map(repo => (
                        <button
                            key={repo.id}
                            onClick={() => handleSwitchRepository(repo.id)}
                            className={`w-full text-left px-4 py-2 text-sm hover:bg-gray-50 ${
                                repo.id === activeRepoId ? 'bg-primary-50 text-primary-700 font-medium' : 'text-gray-700'
                            }`}
                            disabled={switching}
                        >
                            <div className="font-medium">{repo.name}</div>
                            <div className="text-xs text-gray-500 truncate">{repo.path}</div>
                        </button>
                    ))}
                </div>
            </div>
        </div>
    );
};

export default RepositorySwitcher;