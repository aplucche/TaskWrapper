import React, { useState, useEffect, useCallback } from 'react';
import { motion } from 'framer-motion';
import { Save, Edit3, Eye, AlertCircle, CheckCircle2 } from 'lucide-react';
import { LoadPlan, SavePlan } from '../../wailsjs/go/main/App';

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
              <button
                onClick={() => setIsEditing(true)}
                className="flex items-center space-x-2 px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2"
              >
                <Edit3 className="w-4 h-4" />
                <span>Edit</span>
              </button>
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
    </div>
  );
};

export default PlanView;