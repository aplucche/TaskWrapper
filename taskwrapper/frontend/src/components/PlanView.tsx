import React, { useState, useEffect, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Save, Edit3, Eye, AlertCircle, CheckCircle2, Sparkles, X, Check } from 'lucide-react';
import { LoadPlan, SavePlan, SuggestTasks, AddSuggestedTasks } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

interface PlanViewProps {
  onError: (error: string | null) => void;
  onSave: () => void;
}

const PlanView: React.FC<PlanViewProps> = ({ onError, onSave }) => {
  const [content, setContent] = useState('');
  const [originalContent, setOriginalContent] = useState('');
  const [isEditing, setIsEditing] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [isDirty, setIsDirty] = useState(false);

  // Suggestion modal state
  const [showGuidanceModal, setShowGuidanceModal] = useState(false);
  const [guidanceText, setGuidanceText] = useState('');
  const [showSuggestionModal, setShowSuggestionModal] = useState(false);
  const [suggestedTasks, setSuggestedTasks] = useState<main.SuggestedTask[]>([]);
  const [selectedSuggestions, setSelectedSuggestions] = useState<Set<number>>(new Set());
  const [loadingSuggestions, setLoadingSuggestions] = useState(false);
  const [addingSuggestions, setAddingSuggestions] = useState(false);

  // Load plan content on mount
  useEffect(() => {
    loadPlanContent();
  }, []);

  const loadPlanContent = async () => {
    try {
      setLoading(true);
      onError(null);
      const planContent = await LoadPlan();
      setContent(planContent);
      setOriginalContent(planContent);
      setIsDirty(false);
    } catch (err) {
      onError(`Failed to load plan: ${err}`);
      console.error('Error loading plan:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    try {
      setSaving(true);
      onError(null);
      await SavePlan(content);
      setOriginalContent(content);
      setIsDirty(false);
      setIsEditing(false); // Return to view mode after successful save
      onSave();
    } catch (err) {
      onError(`Failed to save plan: ${err}`);
      console.error('Error saving plan:', err);
    } finally {
      setSaving(false);
    }
  };

  const handleCancel = () => {
    setContent(originalContent);
    setIsDirty(false);
    setIsEditing(false);
  };

  const handleContentChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setContent(e.target.value);
    setIsDirty(e.target.value !== originalContent);
  };

  const handleSuggestTasks = () => {
    // Show guidance modal first
    setShowGuidanceModal(true);
    setGuidanceText(''); // Reset guidance text
  };

  const handleGenerateSuggestions = async (guidance: string) => {
    try {
      setLoadingSuggestions(true);
      setShowGuidanceModal(false);
      onError(null);
      const suggestions = await SuggestTasks(guidance);
      setSuggestedTasks(suggestions || []);
      setSelectedSuggestions(new Set(suggestions.map((_, i) => i))); // Select all by default
      setShowSuggestionModal(true);
    } catch (err) {
      onError(`Failed to generate suggestions: ${err}`);
      console.error('Error generating suggestions:', err);
    } finally {
      setLoadingSuggestions(false);
    }
  };

  const handleToggleSuggestion = (index: number) => {
    const newSelected = new Set(selectedSuggestions);
    if (newSelected.has(index)) {
      newSelected.delete(index);
    } else {
      newSelected.add(index);
    }
    setSelectedSuggestions(newSelected);
  };

  const handleAddSelectedTasks = async () => {
    try {
      setAddingSuggestions(true);
      onError(null);

      // Filter to only selected suggestions
      const tasksToAdd = suggestedTasks.filter((_, i) => selectedSuggestions.has(i));

      if (tasksToAdd.length === 0) {
        onError('Please select at least one task to add');
        return;
      }

      await AddSuggestedTasks(tasksToAdd);

      // Close modal and reset state
      setShowSuggestionModal(false);
      setSuggestedTasks([]);
      setSelectedSuggestions(new Set());

      // Notify success (could add a toast here)
      console.log(`Added ${tasksToAdd.length} tasks successfully`);
    } catch (err) {
      onError(`Failed to add tasks: ${err}`);
      console.error('Error adding tasks:', err);
    } finally {
      setAddingSuggestions(false);
    }
  };

  const handleCancelSuggestions = () => {
    setShowSuggestionModal(false);
    setSuggestedTasks([]);
    setSelectedSuggestions(new Set());
  };

  const renderMarkdown = (markdown: string) => {
    // Simple markdown rendering (you could use a library like react-markdown for better results)
    return markdown.split('\n').map((line, i) => {
      // Headers
      if (line.startsWith('### ')) {
        return <h3 key={i} className="text-lg font-semibold mt-4 mb-2">{line.substring(4)}</h3>;
      }
      if (line.startsWith('## ')) {
        return <h2 key={i} className="text-xl font-bold mt-6 mb-3">{line.substring(3)}</h2>;
      }
      if (line.startsWith('# ')) {
        return <h1 key={i} className="text-2xl font-bold mt-8 mb-4">{line.substring(2)}</h1>;
      }
      
      // Bullets
      if (line.startsWith('- ')) {
        return <li key={i} className="ml-4 mb-1">{line.substring(2)}</li>;
      }
      
      // Code blocks (simple)
      if (line.startsWith('```')) {
        return <div key={i} className="font-mono text-sm bg-gray-100 p-1 rounded"></div>;
      }
      
      // Regular paragraphs
      if (line.trim()) {
        return <p key={i} className="mb-2">{line}</p>;
      }
      
      // Empty lines
      return <br key={i} />;
    });
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-gray-500">Loading plan...</div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full bg-gray-50">
      {/* Toolbar */}
      <div className="bg-white border-b border-gray-200 px-6 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <h2 className="text-lg font-semibold text-gray-900">Plan</h2>
            {isDirty && (
              <motion.span
                initial={{ scale: 0 }}
                animate={{ scale: 1 }}
                className="text-sm text-orange-600"
              >
                • Unsaved changes
              </motion.span>
            )}
          </div>
          
          <div className="flex items-center space-x-3">
            {isEditing ? (
              <>
                <button
                  onClick={handleCancel}
                  className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2"
                >
                  Cancel
                </button>
                <button
                  onClick={handleSave}
                  disabled={!isDirty || saving}
                  className="flex items-center space-x-2 px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <Save className="w-4 h-4" />
                  <span>{saving ? 'Saving...' : 'Save'}</span>
                </button>
              </>
            ) : (
              <>
                <button
                  onClick={handleSuggestTasks}
                  disabled={loadingSuggestions}
                  className="flex items-center space-x-2 px-4 py-2 text-sm font-medium text-purple-700 bg-purple-50 border border-purple-300 rounded-md hover:bg-purple-100 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <Sparkles className="w-4 h-4" />
                  <span>{loadingSuggestions ? 'Suggesting...' : 'Suggest Tasks'}</span>
                </button>
                <button
                  onClick={() => setIsEditing(true)}
                  className="flex items-center space-x-2 px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2"
                >
                  <Edit3 className="w-4 h-4" />
                  <span>Edit</span>
                </button>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Content area */}
      <div className="flex-1 overflow-auto p-6">
        <div className="max-w-4xl mx-auto bg-white rounded-lg shadow-sm border border-gray-200">
          {isEditing ? (
            <textarea
              value={content}
              onChange={handleContentChange}
              className="w-full h-full min-h-[600px] p-6 font-mono text-sm border-0 focus:outline-none focus:ring-0 resize-none"
              placeholder="Enter your plan in markdown format..."
            />
          ) : (
            <div className="p-6 prose prose-sm max-w-none">
              {content ? (
                <div className="text-gray-700 leading-relaxed">
                  {renderMarkdown(content)}
                </div>
              ) : (
                <div className="text-gray-400 text-center py-8">
                  No plan content yet. Click Edit to start writing.
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Guidance Input Modal */}
      <AnimatePresence>
        {showGuidanceModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center">
            {/* Backdrop */}
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              onClick={() => setShowGuidanceModal(false)}
              className="absolute inset-0 bg-black bg-opacity-50"
            />

            {/* Modal */}
            <motion.div
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.95 }}
              className="relative bg-white rounded-lg shadow-xl max-w-lg w-full mx-4"
            >
              {/* Header */}
              <div className="px-6 py-4 border-b border-gray-200">
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-2">
                    <Sparkles className="w-5 h-5 text-purple-600" />
                    <h3 className="text-lg font-semibold text-gray-900">Suggest Tasks</h3>
                  </div>
                  <button
                    onClick={() => setShowGuidanceModal(false)}
                    className="text-gray-400 hover:text-gray-600 focus:outline-none"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </div>
                <p className="mt-1 text-sm text-gray-500">
                  Optionally provide guidance for task suggestions
                </p>
              </div>

              {/* Content */}
              <div className="px-6 py-4">
                <label htmlFor="guidance" className="block text-sm font-medium text-gray-700 mb-2">
                  Focus area (optional)
                </label>
                <input
                  id="guidance"
                  type="text"
                  value={guidanceText}
                  onChange={(e) => setGuidanceText(e.target.value)}
                  placeholder="e.g., UI polish, testing, documentation..."
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                  autoFocus
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      handleGenerateSuggestions(guidanceText.trim());
                    }
                  }}
                />
                <p className="mt-2 text-xs text-gray-500">
                  Leave blank to let AI choose based on the plan
                </p>
              </div>

              {/* Footer */}
              <div className="px-6 py-4 border-t border-gray-200 flex items-center justify-end space-x-3">
                <button
                  onClick={() => setShowGuidanceModal(false)}
                  className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2"
                >
                  Cancel
                </button>
                <button
                  onClick={() => handleGenerateSuggestions(guidanceText.trim())}
                  disabled={loadingSuggestions}
                  className="px-4 py-2 text-sm font-medium text-white bg-purple-600 rounded-md hover:bg-purple-700 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {loadingSuggestions ? 'Generating...' : 'Generate'}
                </button>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>

      {/* Suggestion Modal */}
      <AnimatePresence>
        {showSuggestionModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center">
            {/* Backdrop */}
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              onClick={handleCancelSuggestions}
              className="absolute inset-0 bg-black bg-opacity-50"
            />

            {/* Modal */}
            <motion.div
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.95 }}
              className="relative bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[80vh] overflow-hidden"
            >
              {/* Header */}
              <div className="px-6 py-4 border-b border-gray-200">
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-2">
                    <Sparkles className="w-5 h-5 text-purple-600" />
                    <h3 className="text-lg font-semibold text-gray-900">Suggested Tasks</h3>
                  </div>
                  <button
                    onClick={handleCancelSuggestions}
                    className="text-gray-400 hover:text-gray-600 focus:outline-none"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </div>
                <p className="mt-1 text-sm text-gray-500">
                  Select tasks to add to your board
                </p>
              </div>

              {/* Content */}
              <div className="px-6 py-4 overflow-y-auto max-h-[calc(80vh-180px)]">
                {suggestedTasks.length === 0 ? (
                  <div className="text-center py-8 text-gray-500">
                    No suggestions generated
                  </div>
                ) : (
                  <div className="space-y-3">
                    {suggestedTasks.map((task, index) => (
                      <div
                        key={index}
                        onClick={() => handleToggleSuggestion(index)}
                        className={`p-4 border rounded-lg cursor-pointer transition-all ${
                          selectedSuggestions.has(index)
                            ? 'border-purple-500 bg-purple-50'
                            : 'border-gray-200 hover:border-gray-300'
                        }`}
                      >
                        <div className="flex items-start space-x-3">
                          {/* Checkbox */}
                          <div className="flex-shrink-0 mt-1">
                            <div
                              className={`w-5 h-5 rounded border-2 flex items-center justify-center ${
                                selectedSuggestions.has(index)
                                  ? 'bg-purple-600 border-purple-600'
                                  : 'border-gray-300'
                              }`}
                            >
                              {selectedSuggestions.has(index) && (
                                <Check className="w-3 h-3 text-white" />
                              )}
                            </div>
                          </div>

                          {/* Task details */}
                          <div className="flex-1">
                            <div className="flex items-center space-x-2">
                              <h4 className="font-medium text-gray-900">{task.title}</h4>
                              <span
                                className={`px-2 py-0.5 text-xs font-medium rounded ${
                                  task.priority === 'high'
                                    ? 'bg-red-100 text-red-800'
                                    : task.priority === 'medium'
                                    ? 'bg-yellow-100 text-yellow-800'
                                    : 'bg-green-100 text-green-800'
                                }`}
                              >
                                {task.priority}
                              </span>
                            </div>
                            <p className="mt-1 text-sm text-gray-600">{task.reason}</p>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Footer */}
              <div className="px-6 py-4 border-t border-gray-200 flex items-center justify-between">
                <div className="text-sm text-gray-600">
                  {selectedSuggestions.size} of {suggestedTasks.length} selected
                </div>
                <div className="flex items-center space-x-3">
                  <button
                    onClick={handleCancelSuggestions}
                    className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleAddSelectedTasks}
                    disabled={selectedSuggestions.size === 0 || addingSuggestions}
                    className="px-4 py-2 text-sm font-medium text-white bg-purple-600 rounded-md hover:bg-purple-700 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {addingSuggestions ? 'Adding...' : `Add ${selectedSuggestions.size} Task${selectedSuggestions.size !== 1 ? 's' : ''}`}
                  </button>
                </div>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>
    </div>
  );
};

export default PlanView;