import React, { useState, useEffect } from 'react';
import { Terminal as TerminalIcon, Activity } from 'lucide-react';
import { GetAgentStatus } from '../../wailsjs/go/main/App';
import Terminal from './Terminal';

interface AgentWorktree {
    name: string;
    status: 'idle' | 'busy' | 'stale';
    taskId?: string;
    taskTitle?: string;
    pid?: string;
    started?: string;
}

interface AgentStatusInfo {
    worktrees: AgentWorktree[];
    totalWorktrees: number;
    idleCount: number;
    busyCount: number;
    maxSubagents: number;
}

export default function CodeView() {
    const [agentStatus, setAgentStatus] = useState<AgentStatusInfo | null>(null);

    // Fetch agent status periodically
    useEffect(() => {
        const fetchAgentStatus = async () => {
            try {
                const status = await GetAgentStatus();
                setAgentStatus(status as AgentStatusInfo);
            } catch (error) {
                console.error('Failed to fetch agent status:', error);
            }
        };

        fetchAgentStatus();
        const interval = setInterval(fetchAgentStatus, 5000); // Update every 5 seconds

        return () => clearInterval(interval);
    }, []);

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'idle':
                return 'text-green-600';
            case 'busy':
                return 'text-yellow-600';
            case 'stale':
                return 'text-red-600';
            default:
                return 'text-gray-600';
        }
    };

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'idle':
                return '✓';
            case 'busy':
                return '●';
            case 'stale':
                return '!';
            default:
                return '?';
        }
    };

    return (
        <div className="h-full flex flex-col bg-gray-50">
            <div className="flex-1 flex gap-6 p-6 min-h-0 overflow-hidden">
                {/* Terminal Section */}
                <div className="flex-1 flex flex-col min-h-0">
                    <div className="mb-4 flex-shrink-0">
                        <h2 className="text-lg font-semibold flex items-center space-x-2">
                            <TerminalIcon className="w-5 h-5" />
                            <span>Interactive Terminal</span>
                        </h2>
                        <p className="text-sm text-gray-600 mt-1">
                            Run Claude CLI and other commands in this full-featured terminal
                        </p>
                    </div>
                    
                    <Terminal className="flex-1 min-h-0" />
                </div>

                {/* Agent Status Section */}
                <div className="w-80 flex-shrink-0 flex flex-col bg-white rounded-lg border border-gray-200 max-h-full">
                    <div className="px-6 py-4 border-b border-gray-200 flex-shrink-0">
                        <h2 className="text-lg font-semibold flex items-center space-x-2">
                            <Activity className="w-5 h-5" />
                            <span>Agent Monitor</span>
                        </h2>
                    </div>
                    
                    <div className="flex-1 p-6 overflow-y-auto">
                        {agentStatus ? (
                            <div className="space-y-6">
                                {/* Summary */}
                                <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                                    <h3 className="font-semibold text-blue-900 mb-2">Summary</h3>
                                    <div className="space-y-1 text-sm">
                                        <div>Total worktrees: {agentStatus.totalWorktrees}</div>
                                        <div className="flex items-center space-x-4">
                                            <span className="text-green-600">Idle: {agentStatus.idleCount}</span>
                                            <span className="text-yellow-600">Busy: {agentStatus.busyCount}</span>
                                        </div>
                                        <div>Available slots: {agentStatus.maxSubagents - agentStatus.totalWorktrees}</div>
                                    </div>
                                </div>

                                {/* Worktrees */}
                                <div className="space-y-4">
                                    <h3 className="font-semibold">Active Worktrees</h3>
                                    {agentStatus.worktrees.length > 0 ? (
                                        agentStatus.worktrees.map((worktree, index) => (
                                            <div key={index} className="border border-gray-200 rounded-lg p-4">
                                                <div className="flex items-center justify-between mb-2">
                                                    <span className="font-medium">{worktree.name}</span>
                                                    <span className={`flex items-center space-x-1 ${getStatusColor(worktree.status)}`}>
                                                        <span>{getStatusIcon(worktree.status)}</span>
                                                        <span className="uppercase text-xs font-semibold">{worktree.status}</span>
                                                    </span>
                                                </div>
                                                
                                                {worktree.status !== 'idle' && (
                                                    <div className="text-sm text-gray-600 space-y-1">
                                                        {worktree.taskId && (
                                                            <div>Task: #{worktree.taskId} - {worktree.taskTitle}</div>
                                                        )}
                                                        {worktree.pid && (
                                                            <div>PID: {worktree.pid}</div>
                                                        )}
                                                        {worktree.started && (
                                                            <div>Started: {worktree.started}</div>
                                                        )}
                                                    </div>
                                                )}
                                            </div>
                                        ))
                                    ) : (
                                        <div className="text-gray-500 text-center py-8">
                                            No active worktrees
                                        </div>
                                    )}
                                </div>
                            </div>
                        ) : (
                            <div className="text-gray-500 text-center py-8">Loading agent status...</div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}