'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/store/auth';
import { apiClient } from '@/lib/api-client';
import Sidebar from '@/components/Sidebar';

interface Workflow {
  id: string;
  name: string;
  status: string;
  created_at: string;
  updated_at?: string;
}

export default function WorkflowsPage() {
  const router = useRouter();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated());
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [loading, setLoading] = useState(true);
  const [newWorkflowName, setNewWorkflowName] = useState('');
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/auth/login');
      return;
    }
    fetchWorkflows();
  }, [isAuthenticated, router]);

  const fetchWorkflows = async () => {
    try {
      setLoading(true);
      const result = await apiClient.listWorkflows(100, 0);
      setWorkflows(result.data || []);
    } catch (error) {
      console.error('Failed to fetch workflows:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateWorkflow = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newWorkflowName.trim()) return;

    try {
      setCreating(true);
      await apiClient.createWorkflow(newWorkflowName);
      setNewWorkflowName('');
      await fetchWorkflows();
    } catch (error) {
      console.error('Failed to create workflow:', error);
    } finally {
      setCreating(false);
    }
  };

  const groupedWorkflows = {
    completed: workflows.filter(w => w.status === 'completed'),
    running: workflows.filter(w => w.status === 'running'),
    pending: workflows.filter(w => w.status === 'pending'),
    failed: workflows.filter(w => w.status === 'failed'),
  };

  return (
    <div className="flex h-screen bg-slate-950">
      <Sidebar />
      <div className="flex-1 ml-64 overflow-auto">
        <div className="bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 min-h-screen">
          {/* Header */}
          <div className="border-b border-slate-700 sticky top-0 z-10">
            <div className="px-8 py-6 bg-slate-900/50 backdrop-blur">
              <h1 className="text-3xl font-bold text-white">Workflows</h1>
              <p className="text-slate-400 text-sm mt-1">Create and manage your workflow orchestrations.</p>
            </div>
          </div>

          <div className="p-8 space-y-8">
            {/* Create Workflow Form */}
            <div className="bg-slate-800/50 backdrop-blur border border-slate-700 rounded-xl p-6">
              <h2 className="text-lg font-semibold text-white mb-4">Create New Workflow</h2>
              <form onSubmit={handleCreateWorkflow} className="flex gap-3">
                <input
                  type="text"
                  placeholder="Enter workflow name (e.g., Data Processing Pipeline)"
                  value={newWorkflowName}
                  onChange={(e) => setNewWorkflowName(e.target.value)}
                  className="flex-1 px-4 py-2 bg-slate-700/50 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                />
                <button
                  type="submit"
                  disabled={creating || !newWorkflowName.trim()}
                  className="px-6 py-2 bg-gradient-to-r from-blue-600 to-blue-500 hover:from-blue-700 hover:to-blue-600 text-white font-medium rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-all"
                >
                  {creating ? 'Creating...' : 'Create Workflow'}
                </button>
              </form>
            </div>

            {/* Stats */}
            <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
              <StatCard 
                label="Total" 
                value={workflows.length}
                color="bg-slate-600"
              />
              <StatCard 
                label="Completed" 
                value={groupedWorkflows.completed.length}
                color="bg-green-600"
              />
              <StatCard 
                label="Running" 
                value={groupedWorkflows.running.length}
                color="bg-blue-600"
              />
              <StatCard 
                label="Pending" 
                value={groupedWorkflows.pending.length}
                color="bg-yellow-600"
              />
              <StatCard 
                label="Failed" 
                value={groupedWorkflows.failed.length}
                color="bg-red-600"
              />
            </div>

            {/* Workflows by Status */}
            {loading ? (
              <div className="flex items-center justify-center py-16">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
              </div>
            ) : workflows.length === 0 ? (
              <div className="text-center py-16 bg-slate-800/50 backdrop-blur border border-slate-700 rounded-xl">
                <p className="text-slate-400 text-lg">No workflows yet</p>
                <p className="text-slate-500 text-sm mt-2">Create your first workflow using the form above</p>
              </div>
            ) : (
              <div className="space-y-6">
                {Object.entries(groupedWorkflows).map(([status, items]) => 
                  items.length > 0 && (
                    <div key={status} className="bg-slate-800/50 backdrop-blur border border-slate-700 rounded-xl overflow-hidden">
                      <div className={`px-6 py-4 border-b border-slate-700 bg-gradient-to-r ${getStatusGradient(status)}`}>
                        <h3 className="text-lg font-semibold text-white capitalize">
                          {status} Workflows ({items.length})
                        </h3>
                      </div>
                      <div className="p-6">
                        <div className="grid gap-3">
                          {items.map((workflow) => (
                            <Link
                              key={workflow.id}
                              href={`/workflows/${workflow.id}`}
                              className="group p-4 bg-slate-700/30 hover:bg-slate-600/40 border border-slate-600/30 hover:border-slate-500/50 rounded-lg transition-all"
                            >
                              <div className="flex items-center justify-between">
                                <div>
                                  <p className="text-white font-medium group-hover:text-blue-300 transition-colors">{workflow.name}</p>
                                  <p className="text-xs text-slate-400 mt-1">
                                    Created {new Date(workflow.created_at).toLocaleDateString()}
                                  </p>
                                </div>
                                <div className="text-right">
                                  <span className={`inline-block px-3 py-1 rounded-full text-xs font-medium ${getStatusBadgeColor(status)}`}>
                                    {status}
                                  </span>
                                  <p className="text-xs text-slate-400 mt-2">
                                    ID: {workflow.id.substring(0, 8)}...
                                  </p>
                                </div>
                              </div>
                            </Link>
                          ))}
                        </div>
                      </div>
                    </div>
                  )
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function StatCard({ 
  label, 
  value,
  color
}: { 
  label: string
  value: number
  color: string
}) {
  return (
    <div className="bg-slate-800/50 backdrop-blur border border-slate-700 rounded-xl p-4">
      <p className="text-slate-400 text-xs font-medium uppercase tracking-wider">{label}</p>
      <div className="flex items-end gap-3 mt-2">
        <p className="text-2xl font-bold text-white">{value}</p>
        <div className={`w-12 h-8 rounded ${color} opacity-40`}></div>
      </div>
    </div>
  );
}

function getStatusGradient(status: string): string {
  switch (status) {
    case 'completed':
      return 'from-green-600/20 to-emerald-600/10';
    case 'running':
      return 'from-blue-600/20 to-cyan-600/10';
    case 'pending':
      return 'from-yellow-600/20 to-orange-600/10';
    case 'failed':
      return 'from-red-600/20 to-rose-600/10';
    default:
      return 'from-slate-600/20 to-slate-500/10';
  }
}

function getStatusBadgeColor(status: string): string {
  switch (status) {
    case 'completed':
      return 'bg-green-500/20 text-green-300';
    case 'running':
      return 'bg-blue-500/20 text-blue-300';
    case 'pending':
      return 'bg-yellow-500/20 text-yellow-300';
    case 'failed':
      return 'bg-red-500/20 text-red-300';
    default:
      return 'bg-slate-500/20 text-slate-300';
  }
}
