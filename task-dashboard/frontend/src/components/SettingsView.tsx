import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { Plus, Trash2, FolderOpen, Check, X, RefreshCw } from 'lucide-react';
import { GetConfig, GetRepositories, AddRepository, RemoveRepository, SetActiveRepository, ValidateRepositoryPath, OpenDirectoryDialog } from '../../wailsjs/go/main/App';
import { Repository, RepositoryInfo } from '../types/config';

const SettingsView: React.FC = () => {
    const [repositories, setRepositories] = useState<Repository[]>([]);
    const [activeRepoId, setActiveRepoId] = useState<string>('');
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [showAddForm, setShowAddForm] = useState(false);
    const [newRepoName, setNewRepoName] = useState('');
    const [newRepoPath, setNewRepoPath] = useState('');
    const [validationInfo, setValidationInfo] = useState<RepositoryInfo | null>(null);
    const [validating, setValidating] = useState(false);
    const [saving, setSaving] = useState(false);
    const [removingId, setRemovingId] = useState<string | null>(null);

    useEffect(() => {
        loadRepositories();
    }, []);

    const loadRepositories = async () => {
        try {
            setLoading(true);
            const repos = await GetRepositories();
            setRepositories(repos);
            
            // Get config to find active repository
            const config = await GetConfig();
            
            // Find the active repository ID
            const activeRepo = repos.find(r => r.path === config.activeRepository);
            if (activeRepo) {
                setActiveRepoId(activeRepo.id);
            } else if (repos.length > 0) {
                setActiveRepoId(repos[0].id);
            }
        } catch (err) {
            setError('Failed to load repositories');
            console.error(err);
        } finally {
            setLoading(false);
        }
    };

    const handleValidatePath = async (path: string) => {
        if (!path) {
            setValidationInfo(null);
            return;
        }

        try {
            setValidating(true);
            const info = await ValidateRepositoryPath(path);
            setValidationInfo(info);
        } catch (err) {
            console.error('Validation error:', err);
            setValidationInfo(null);
        } finally {
            setValidating(false);
        }
    };

    const handleAddRepository = async () => {
        if (!newRepoPath || !validationInfo?.isValid) return;

        try {
            setSaving(true);
            const name = newRepoName || validationInfo.name;
            const newRepo = await AddRepository(name, newRepoPath);
            await loadRepositories();
            
            // Automatically switch to the newly added repository
            await SetActiveRepository(newRepo.id);
            
            // Reset form
            setShowAddForm(false);
            setNewRepoName('');
            setNewRepoPath('');
            setValidationInfo(null);
            
            // Trigger a reload of the repository switcher and app context
            window.dispatchEvent(new CustomEvent('repositoriesChanged'));
            // Reload the entire app to switch context to the new repository
            window.location.reload();
        } catch (err: any) {
            setError(err.message || 'Failed to add repository');
        } finally {
            setSaving(false);
        }
    };

    const handleRemoveRepository = async (id: string) => {
        if (repositories.length <= 1) {
            setError('Cannot remove the last repository');
            return;
        }

        try {
            setRemovingId(id);
            await RemoveRepository(id);
            await loadRepositories();
            
            // Trigger a reload of the repository switcher
            window.dispatchEvent(new CustomEvent('repositoriesChanged'));
        } catch (err: any) {
            setError(err.message || 'Failed to remove repository');
        } finally {
            setRemovingId(null);
        }
    };

    const handleSetActiveRepository = async (id: string) => {
        try {
            await SetActiveRepository(id);
            setActiveRepoId(id);
            
            // Reload the entire app to switch context
            window.location.reload();
        } catch (err: any) {
            setError(err.message || 'Failed to switch repository');
        }
    };

    const handleBrowseDirectory = async () => {
        try {
            const selectedPath = await OpenDirectoryDialog();
            if (selectedPath) {
                setNewRepoPath(selectedPath);
                handleValidatePath(selectedPath);
            }
        } catch (err: any) {
            console.error('Failed to open directory dialog:', err);
            setError('Failed to open directory dialog');
        }
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-full">
                <RefreshCw className="w-8 h-8 animate-spin text-gray-400" />
            </div>
        );
    }

    return (
        <div className="h-full overflow-y-auto bg-gray-50">
            <div className="max-w-4xl mx-auto p-6">
                <h2 className="text-2xl font-bold text-gray-900 mb-8">Settings</h2>

                {error && (
                    <motion.div
                        initial={{ opacity: 0, y: -10 }}
                        animate={{ opacity: 1, y: 0 }}
                        className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700"
                    >
                        {error}
                        <button
                            onClick={() => setError(null)}
                            className="ml-2 text-red-500 hover:text-red-700"
                        >
                            <X className="inline w-4 h-4" />
                        </button>
                    </motion.div>
                )}

                <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
                    <h3 className="text-lg font-semibold text-gray-900 mb-4">Repository Management</h3>
                    
                    {repositories.length > 1 && (
                        <div className="mb-6">
                            <label className="block text-sm font-medium text-gray-700 mb-2">
                                Active Repository
                            </label>
                            <select
                                value={activeRepoId}
                                onChange={(e) => handleSetActiveRepository(e.target.value)}
                                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
                            >
                                {repositories.map(repo => (
                                    <option key={repo.id} value={repo.id}>
                                        {repo.name} - {repo.path}
                                    </option>
                                ))}
                            </select>
                        </div>
                    )}

                    <div className="space-y-3">
                        <div className="font-medium text-sm text-gray-700 mb-2">Repositories:</div>
                        {repositories.map(repo => (
                            <motion.div
                                key={repo.id}
                                initial={{ opacity: 0, x: -20 }}
                                animate={{ opacity: 1, x: 0 }}
                                className="flex items-start justify-between p-4 bg-gray-50 rounded-lg border border-gray-200"
                            >
                                <div className="flex-1">
                                    <div className="flex items-center">
                                        {repo.id === activeRepoId && (
                                            <Check className="w-4 h-4 text-green-600 mr-2" />
                                        )}
                                        <h4 className="font-medium text-gray-900">{repo.name}</h4>
                                    </div>
                                    <p className="text-sm text-gray-600 mt-1">{repo.path}</p>
                                </div>
                                <div className="flex items-center space-x-2">
                                    {repositories.length > 1 && (
                                        <button
                                            onClick={() => handleRemoveRepository(repo.id)}
                                            disabled={removingId === repo.id}
                                            className="p-2 text-red-600 hover:bg-red-50 rounded-md transition-colors disabled:opacity-50"
                                            title="Remove repository"
                                        >
                                            {removingId === repo.id ? (
                                                <RefreshCw className="w-4 h-4 animate-spin" />
                                            ) : (
                                                <Trash2 className="w-4 h-4" />
                                            )}
                                        </button>
                                    )}
                                </div>
                            </motion.div>
                        ))}
                    </div>

                    {!showAddForm ? (
                        <button
                            onClick={() => setShowAddForm(true)}
                            className="mt-4 flex items-center px-4 py-2 bg-primary-600 text-white rounded-md hover:bg-primary-700 transition-colors"
                        >
                            <Plus className="w-4 h-4 mr-2" />
                            Add Repository
                        </button>
                    ) : (
                        <motion.div
                            initial={{ opacity: 0, y: -10 }}
                            animate={{ opacity: 1, y: 0 }}
                            className="mt-4 p-4 bg-gray-50 rounded-lg border border-gray-200"
                        >
                            <h4 className="font-medium text-gray-900 mb-3">Add New Repository</h4>
                            
                            <div className="space-y-3">
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">
                                        Repository Name (optional)
                                    </label>
                                    <input
                                        type="text"
                                        value={newRepoName}
                                        onChange={(e) => setNewRepoName(e.target.value)}
                                        placeholder="My Project"
                                        className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
                                    />
                                </div>

                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">
                                        Repository Path
                                    </label>
                                    <div className="flex space-x-2">
                                        <input
                                            type="text"
                                            value={newRepoPath}
                                            onChange={(e) => {
                                                setNewRepoPath(e.target.value);
                                                handleValidatePath(e.target.value);
                                            }}
                                            placeholder="/path/to/repository"
                                            className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
                                        />
                                        <button
                                            type="button"
                                            onClick={handleBrowseDirectory}
                                            className="px-3 py-2 border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
                                            title="Browse for folder"
                                        >
                                            <FolderOpen className="w-4 h-4 text-gray-600" />
                                        </button>
                                    </div>
                                    
                                    {validating && (
                                        <p className="mt-2 text-sm text-gray-500">Validating...</p>
                                    )}
                                    
                                    {validationInfo && !validating && (
                                        <div className={`mt-2 text-sm ${validationInfo.isValid ? 'text-green-600' : 'text-red-600'}`}>
                                            {validationInfo.isValid ? (
                                                <span className="flex items-center">
                                                    <Check className="w-4 h-4 mr-1" />
                                                    Valid repository ({validationInfo.taskCount} tasks)
                                                </span>
                                            ) : (
                                                <span className="flex items-center">
                                                    <X className="w-4 h-4 mr-1" />
                                                    {validationInfo.errorMessage}
                                                </span>
                                            )}
                                        </div>
                                    )}
                                </div>
                            </div>

                            <div className="flex justify-end space-x-2 mt-4">
                                <button
                                    onClick={() => {
                                        setShowAddForm(false);
                                        setNewRepoName('');
                                        setNewRepoPath('');
                                        setValidationInfo(null);
                                    }}
                                    className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-md transition-colors"
                                >
                                    Cancel
                                </button>
                                <button
                                    onClick={handleAddRepository}
                                    disabled={!validationInfo?.isValid || saving}
                                    className="px-4 py-2 bg-primary-600 text-white rounded-md hover:bg-primary-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                >
                                    {saving ? 'Adding...' : 'Add Repository'}
                                </button>
                            </div>
                        </motion.div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default SettingsView;