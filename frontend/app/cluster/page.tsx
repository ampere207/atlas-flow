'use client';

import React, { useState, useEffect } from 'react';
import axios from 'axios';

interface WorkerStatus {
  id: string;
  name: string;
  status: 'active' | 'idle' | 'offline';
  last_heartbeat: string;
  assigned_tasks_count?: number;
  total_tasks_completed?: number;
}

interface ClusterMetrics {
  total_workers: number;
  active_workers: number;
  idle_workers: number;
  offline_workers: number;
  total_tasks_in_queue: number;
  completed_tasks_total: number;
}

export default function WorkerClusterPage() {
  const [workers, setWorkers] = useState<WorkerStatus[]>([]);
  const [metrics, setMetrics] = useState<ClusterMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchClusterStatus();
    const interval = setInterval(fetchClusterStatus, 3000); // Poll every 3s for live updates
    return () => clearInterval(interval);
  }, []);

  const fetchClusterStatus = async () => {
    try {
      const token = localStorage.getItem('auth_token');
      const [workersRes, metricsRes] = await Promise.all([
        axios.get(
          `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000'}/workers?limit=100`,
          {
            headers: { Authorization: `Bearer ${token}` },
          }
        ),
        axios.get(
          `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000'}/cluster/metrics`,
          {
            headers: { Authorization: `Bearer ${token}` },
          }
        ),
      ]);

      setWorkers(workersRes.data || []);
      setMetrics(metricsRes.data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch cluster status');
    } finally {
      setLoading(false);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'bg-green-500/20 border-green-500/30 text-green-400';
      case 'idle':
        return 'bg-blue-500/20 border-blue-500/30 text-blue-400';
      case 'offline':
        return 'bg-red-500/20 border-red-500/30 text-red-400';
      default:
        return 'bg-gray-500/20 border-gray-500/30 text-gray-400';
    }
  };

  const getHeartbeatAge = (lastHeartbeat: string): string => {
    const diff = Date.now() - new Date(lastHeartbeat).getTime();
    const seconds = Math.floor(diff / 1000);
    if (seconds < 60) return `${seconds}s ago`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    return `${Math.floor(seconds / 3600)}h ago`;
  };

  if (loading && !metrics) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="w-12 h-12 border-4 border-blue-500/20 border-t-blue-500 rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-gray-400">Loading cluster status...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950">
      {/* Header */}
      <div className="border-b border-white/10 bg-slate-900/50 backdrop-blur-sm sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-6 py-6">
          <h1 className="text-3xl font-bold text-white mb-2">Worker Cluster</h1>
          <p className="text-gray-400">Real-time monitoring of distributed execution nodes</p>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-6 py-8">
        {/* Cluster Metrics Grid */}
        {metrics && (
          <div className="grid grid-cols-4 gap-4 mb-8">
            <div className="bg-slate-800/50 border border-white/10 rounded-xl p-6">
              <p className="text-xs text-gray-500 uppercase tracking-widest mb-2">Total Workers</p>
              <p className="text-3xl font-bold text-white">{metrics.total_workers}</p>
              <p className="text-xs text-gray-500 mt-2">Registered in cluster</p>
            </div>
            <div className="bg-slate-800/50 border border-green-500/20 rounded-xl p-6">
              <p className="text-xs text-gray-500 uppercase tracking-widest mb-2">Active</p>
              <p className="text-3xl font-bold text-green-400">{metrics.active_workers}</p>
              <p className="text-xs text-green-400 mt-2">Executing tasks</p>
            </div>
            <div className="bg-slate-800/50 border border-blue-500/20 rounded-xl p-6">
              <p className="text-xs text-gray-500 uppercase tracking-widest mb-2">Idle</p>
              <p className="text-3xl font-bold text-blue-400">{metrics.idle_workers}</p>
              <p className="text-xs text-blue-400 mt-2">Ready for tasks</p>
            </div>
            <div className="bg-slate-800/50 border border-white/10 rounded-xl p-6">
              <p className="text-xs text-gray-500 uppercase tracking-widest mb-2">Throughput</p>
              <p className="text-3xl font-bold text-cyan-400">{metrics.completed_tasks_total}</p>
              <p className="text-xs text-gray-500 mt-2">Total completed</p>
            </div>
          </div>
        )}

        {/* Workers List */}
        <div className="bg-slate-800/50 border border-white/10 rounded-xl overflow-hidden">
          <div className="border-b border-white/10 px-6 py-4 bg-slate-900/50">
            <h2 className="text-lg font-semibold text-white">Active Workers</h2>
          </div>

          {error && (
            <div className="px-6 py-4 bg-red-500/10 border-t border-red-500/20 text-red-400 text-sm">
              {error}
            </div>
          )}

          <div className="divide-y divide-white/5">
            {workers.length === 0 ? (
              <div className="px-6 py-8 text-center text-gray-400">
                <p>No workers registered yet</p>
                <p className="text-sm mt-1">Register a worker to start executing tasks</p>
              </div>
            ) : (
              workers.map((worker) => (
                <div
                  key={worker.id}
                  className="px-6 py-4 hover:bg-slate-700/30 transition-colors flex items-center justify-between"
                >
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-2">
                      <div className="w-2 h-2 rounded-full bg-current animate-pulse" 
                           style={{ 
                             color: worker.status === 'active' ? '#4ade80' : worker.status === 'idle' ? '#60a5fa' : '#ef4444'
                           }}>
                      </div>
                      <p className="font-medium text-white">{worker.name}</p>
                      <span
                        className={`text-xs px-2 py-1 rounded-full border ${getStatusColor(
                          worker.status
                        )}`}
                      >
                        {worker.status}
                      </span>
                    </div>
                    <p className="text-sm text-gray-500">
                      {worker.id}
                    </p>
                  </div>

                  <div className="flex items-center gap-8 text-right">
                    <div>
                      <p className="text-2xl font-bold text-white">
                        {worker.assigned_tasks_count || 0}
                      </p>
                      <p className="text-xs text-gray-500">Tasks assigned</p>
                    </div>
                    <div>
                      <p className="text-2xl font-bold text-green-400">
                        {worker.total_tasks_completed || 0}
                      </p>
                      <p className="text-xs text-gray-500">Completed</p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-400">
                        {getHeartbeatAge(worker.last_heartbeat)}
                      </p>
                      <p className="text-xs text-gray-600">Last heartbeat</p>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Refresh Indicator */}
        <div className="mt-6 text-center">
          <p className="text-xs text-gray-500">
            Auto-refreshing every 3 seconds
            <span className="ml-2 inline-block w-1.5 h-1.5 bg-green-500 rounded-full animate-pulse"></span>
          </p>
        </div>
      </div>
    </div>
  );
}
