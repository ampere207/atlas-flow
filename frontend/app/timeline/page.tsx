'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { apiClient } from '@/lib/api-client';
import { useAuthStore } from '@/store/auth';
import Sidebar from '@/components/Sidebar';

interface TimelineEvent {
  id: string;
  entity_type: string;
  from_state: string;
  to_state: string;
  reason?: string;
  created_at: string;
}

export default function TimelinePage() {
  const router = useRouter();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated());
  const [workflowId, setWorkflowId] = useState('');
  const [events, setEvents] = useState<TimelineEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/auth/login');
      return;
    }

    const url = new URL(window.location.href);
    const selectedWorkflowId = url.searchParams.get('workflow') || '';
    setWorkflowId(selectedWorkflowId);
  }, [isAuthenticated, router]);

  useEffect(() => {
    if (!isAuthenticated || !workflowId) {
      return;
    }

    let active = true;

    const load = async () => {
      setLoading(true);
      try {
        const result = await apiClient.listWorkflowHistory(workflowId);
        if (active) {
          setEvents(result.data || []);
        }
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    };

    load();
    const interval = autoRefresh ? setInterval(load, 5000) : undefined;
    return () => {
      active = false;
      if (interval) clearInterval(interval);
    };
  }, [isAuthenticated, workflowId, autoRefresh]);

  return (
    <div className="flex h-screen bg-slate-950">
      <Sidebar />
      <div className="flex-1 ml-64 overflow-auto">
        <div className="bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 min-h-screen">
          {/* Header */}
          <div className="border-b border-slate-700 sticky top-0 z-10">
            <div className="px-8 py-6 bg-slate-900/50 backdrop-blur">
              <h1 className="text-3xl font-bold text-white">Execution Timeline</h1>
              <p className="text-slate-400 text-sm mt-1">Monitor workflow execution history and state transitions.</p>
            </div>
          </div>

          <div className="p-8 space-y-6">
            {/* Workflow Selection */}
            <div className="bg-slate-800/50 backdrop-blur border border-slate-700 rounded-xl p-6">
              <label className="block text-sm font-medium text-slate-300 mb-3">
                Select Workflow to Inspect
              </label>
              <div className="flex gap-3">
                <input
                  type="text"
                  placeholder="Enter workflow ID or append ?workflow=<id> to URL"
                  value={workflowId}
                  onChange={(e) => {
                    setWorkflowId(e.target.value);
                    // Update URL
                    if (e.target.value) {
                      router.push(`/timeline?workflow=${e.target.value}`);
                    }
                  }}
                  className="flex-1 px-4 py-2 bg-slate-700/50 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                />
                <button
                  onClick={() => setAutoRefresh(!autoRefresh)}
                  className={`px-4 py-2 rounded-lg font-medium transition-all ${
                    autoRefresh
                      ? 'bg-green-600 hover:bg-green-700 text-white'
                      : 'bg-slate-700 hover:bg-slate-600 text-slate-300'
                  }`}
                >
                  {autoRefresh ? '🔄 Auto-refresh' : '⏸ Manual'}
                </button>
              </div>
            </div>

            {/* Timeline */}
            {!workflowId ? (
              <div className="text-center py-16 bg-slate-800/50 backdrop-blur border border-slate-700 rounded-xl">
                <p className="text-slate-400 text-lg">No workflow selected</p>
                <p className="text-slate-500 text-sm mt-2">Enter a workflow ID above or append ?workflow=&lt;id&gt; to the URL</p>
              </div>
            ) : loading && events.length === 0 ? (
              <div className="flex items-center justify-center py-16">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
              </div>
            ) : events.length === 0 ? (
              <div className="text-center py-16 bg-slate-800/50 backdrop-blur border border-slate-700 rounded-xl">
                <p className="text-slate-400 text-lg">No execution events</p>
                <p className="text-slate-500 text-sm mt-2">This workflow hasn't generated any state transitions yet</p>
              </div>
            ) : (
              <div className="bg-slate-800/50 backdrop-blur border border-slate-700 rounded-xl overflow-hidden">
                <div className="px-6 py-4 border-b border-slate-700">
                  <h2 className="text-lg font-semibold text-white">Events ({events.length})</h2>
                </div>
                <div className="p-6">
                  <div className="relative">
                    {/* Timeline line */}
                    <div className="absolute left-8 top-0 bottom-0 w-0.5 bg-gradient-to-b from-blue-500 via-purple-500 to-slate-700"></div>

                    {/* Events */}
                    <div className="space-y-6">
                      {events.slice().reverse().map((event, index) => (
                        <div key={event.id} className="pl-24 relative">
                          {/* Timeline dot */}
                          <div className="absolute left-2 top-1 w-13 h-13 rounded-full bg-slate-700 border-4 border-slate-900 flex items-center justify-center">
                            <div className="w-2 h-2 rounded-full bg-blue-500"></div>
                          </div>

                          {/* Event card */}
                          <div className="p-4 bg-slate-700/30 border border-slate-600/30 hover:border-slate-500/50 rounded-lg transition-all">
                            <div className="flex items-start justify-between gap-4 mb-2">
                              <div className="flex items-center gap-3">
                                <span className="px-3 py-1 rounded-full text-xs font-semibold bg-blue-600/30 text-blue-200">
                                  {event.entity_type}
                                </span>
                                <span className="text-xs text-slate-400">
                                  {new Date(event.created_at).toLocaleTimeString()}
                                </span>
                              </div>
                              <span className="text-xs text-slate-500">
                                #{events.length - index}
                              </span>
                            </div>
                            <div className="mt-3 space-y-2">
                              <div className="flex items-center gap-2 text-sm">
                                <span className="px-2 py-1 rounded bg-slate-600/50 text-slate-200">{event.from_state}</span>
                                <span className="text-slate-400">→</span>
                                <span className="px-2 py-1 rounded bg-blue-600/30 text-blue-200">{event.to_state}</span>
                              </div>
                              {event.reason && (
                                <div className="text-xs text-slate-400 italic">
                                  <span className="text-slate-500">Reason:</span> {event.reason}
                                </div>
                              )}
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
