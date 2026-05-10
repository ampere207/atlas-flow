'use client';

import React, { useState, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import axios from 'axios';
import { useExecutionStream, ExecutionEvent } from '@/hooks/useExecutionStream';
import { ExecutionTimeline } from '@/components/ExecutionTimeline';

interface WorkflowExecution {
  id: string;
  name: string;
  status: string;
  definition?: any;
  created_at: string;
  updated_at: string;
}

interface TaskStatus {
  id: string;
  name: string;
  state: string;
  assigned_worker?: string;
  retry_count: number;
  error_message?: string;
}

export default function WorkflowExecutionPage() {
  const params = useParams();
  const workflowId = params.id as string;
  const router = useRouter();
  const [workflow, setWorkflow] = useState<WorkflowExecution | null>(null);
  const [tasks, setTasks] = useState<TaskStatus[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const { isConnected, events, clearEvents } = useExecutionStream({
    workflowId,
    onEvent: (event) => {
      // Update task states based on events
      if (event.task_id) {
        updateTaskFromEvent(event);
      }
    },
    onError: (err) => {
      console.error('Stream error:', err);
      setError('Failed to connect to execution stream');
    },
  });

  const updateTaskFromEvent = (event: ExecutionEvent) => {
    setTasks((prev) =>
      prev.map((task) =>
        task.id === event.task_id
          ? {
              ...task,
              state: event.event_type.includes('completed')
                ? 'completed'
                : event.event_type.includes('failed')
                  ? 'failed'
                  : event.event_type.includes('retrying')
                    ? 'retrying'
                    : event.event_type.includes('started')
                      ? 'running'
                      : event.event_type.includes('assigned')
                        ? 'assigned'
                        : task.state,
              assigned_worker: event.worker_id || task.assigned_worker,
              error_message: event.error_message || task.error_message,
            }
          : task
      )
    );
  };

  useEffect(() => {
    fetchWorkflow();
  }, [workflowId]);

  const fetchWorkflow = async () => {
    try {
      setLoading(true);
      const token = localStorage.getItem('auth_token');
      const response = await axios.get(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000'}/workflows/${workflowId}`,
        {
          headers: { Authorization: `Bearer ${token}` },
        }
      );
      setWorkflow(response.data);
      await fetchTasks();
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch workflow');
    } finally {
      setLoading(false);
    }
  };

  const fetchTasks = async () => {
    try {
      const token = localStorage.getItem('auth_token');
      const response = await axios.get(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000'}/workflows/${workflowId}/tasks`,
        {
          headers: { Authorization: `Bearer ${token}` },
        }
      );
      setTasks(response.data || []);
    } catch (err) {
      console.error('Failed to fetch tasks:', err);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'text-green-400';
      case 'failed':
        return 'text-red-400';
      case 'running':
      case 'assigned':
        return 'text-blue-400';
      case 'retrying':
        return 'text-yellow-400';
      default:
        return 'text-gray-400';
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="w-12 h-12 border-4 border-blue-500/20 border-t-blue-500 rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-gray-400">Loading workflow execution...</p>
        </div>
      </div>
    );
  }

  if (!workflow) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <p className="text-red-400 mb-4">{error || 'Workflow not found'}</p>
          <button
            onClick={() => router.back()}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg"
          >
            Go Back
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950">
      {/* Header */}
      <div className="border-b border-white/10 bg-slate-900/50 backdrop-blur-sm">
        <div className="max-w-7xl mx-auto px-6 py-6">
          <div className="flex items-center justify-between mb-4">
            <button
              onClick={() => router.back()}
              className="text-gray-400 hover:text-gray-200 transition-colors"
            >
              ← Back
            </button>
            <div className={`flex items-center gap-2 ${isConnected ? 'text-green-400' : 'text-gray-400'}`}>
              <div className="w-2 h-2 rounded-full bg-current animate-pulse"></div>
              {isConnected ? 'Live' : 'Offline'}
            </div>
          </div>
          <h1 className="text-3xl font-bold text-white mb-2">{workflow.name}</h1>
          <div className="flex items-center gap-4 text-sm text-gray-400">
            <span className={`px-3 py-1 rounded-full ${getStatusColor(workflow.status)} bg-opacity-10 border border-current`}>
              {workflow.status}
            </span>
            <span>Created: {new Date(workflow.created_at).toLocaleString()}</span>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-6 py-8">
        <div className="grid grid-cols-3 gap-6 mb-8">
          {/* Tasks Status Summary */}
          <div className="col-span-2 space-y-4">
            <div className="bg-slate-800/50 border border-white/10 rounded-xl p-6">
              <h2 className="text-lg font-semibold text-white mb-4">Task Status</h2>
              <div className="space-y-2 max-h-48 overflow-y-auto">
                {tasks.length === 0 ? (
                  <p className="text-gray-400 text-sm">No tasks defined</p>
                ) : (
                  tasks.map((task) => (
                    <div
                      key={task.id}
                      className="flex items-center justify-between p-3 bg-slate-700/30 border border-white/5 rounded-lg hover:border-white/10 transition-colors"
                    >
                      <div className="flex-1">
                        <p className="text-sm font-medium text-gray-200">{task.name}</p>
                        {task.assigned_worker && (
                          <p className="text-xs text-gray-500">Worker: {task.assigned_worker}</p>
                        )}
                        {task.error_message && (
                          <p className="text-xs text-red-400 mt-1">{task.error_message}</p>
                        )}
                      </div>
                      <div className="flex items-center gap-2">
                        {task.retry_count > 0 && (
                          <span className="text-xs px-2 py-1 bg-yellow-500/20 text-yellow-300 rounded">
                            Retried {task.retry_count}x
                          </span>
                        )}
                        <span className={`text-sm font-medium ${getStatusColor(task.state)}`}>
                          {task.state}
                        </span>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>

          {/* Key Metrics */}
          <div className="space-y-4">
            <div className="bg-slate-800/50 border border-white/10 rounded-xl p-6">
              <p className="text-xs text-gray-500 uppercase tracking-widest mb-2">Total Tasks</p>
              <p className="text-3xl font-bold text-white">{tasks.length}</p>
            </div>
            <div className="bg-slate-800/50 border border-white/10 rounded-xl p-6">
              <p className="text-xs text-gray-500 uppercase tracking-widest mb-2">Completed</p>
              <p className="text-3xl font-bold text-green-400">
                {tasks.filter((t) => t.state === 'completed').length}
              </p>
            </div>
            <div className="bg-slate-800/50 border border-white/10 rounded-xl p-6">
              <p className="text-xs text-gray-500 uppercase tracking-widest mb-2">Failed</p>
              <p className="text-3xl font-bold text-red-400">
                {tasks.filter((t) => t.state === 'failed').length}
              </p>
            </div>
          </div>
        </div>

        {/* Execution Timeline */}
        <div className="bg-slate-800/50 border border-white/10 rounded-xl p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-white">Execution Timeline</h2>
            {events.length > 0 && (
              <button
                onClick={clearEvents}
                className="text-sm text-gray-400 hover:text-gray-200 transition-colors"
              >
                Clear
              </button>
            )}
          </div>
          <ExecutionTimeline events={events} workflowId={workflowId} isLive={isConnected} />
        </div>
      </div>
    </div>
  );
}
