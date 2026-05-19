'use client';

import { useEffect, useMemo, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { fetchEventSource } from '@microsoft/fetch-event-source';
import { ExecutionGraph } from '@/components/execution/ExecutionGraph';
import { ExecutionTimeline } from '@/components/execution/ExecutionTimeline';
import { apiClient } from '@/lib/api-client';
import { useAuthStore } from '@/store/auth';

interface WorkflowSnapshot {
  workflow?: {
    id: string;
    name: string;
    status: string;
    updated_at: string;
  };
  tasks?: Array<{
    id: string;
    name: string;
    state: string;
    task_type: string;
    depends_on?: string;
    retry_count?: number;
    error_message?: string;
  }>;
  history?: Array<{
    id: string;
    entity_type: string;
    from_state: string;
    to_state: string;
    reason?: string;
    created_at: string;
    worker_id?: string;
  }>;
}

export default function WorkflowExecutionPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated());
  const [snapshot, setSnapshot] = useState<WorkflowSnapshot>({});
  const [loading, setLoading] = useState(true);
  const [streaming, setStreaming] = useState(false);

  const workflowId = useMemo(() => String(params?.id || ''), [params]);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/auth/login');
      return;
    }

    let cancelled = false;

    const loadInitial = async () => {
      try {
        setLoading(true);
        const [statusResult, tasksResult, historyResult] = await Promise.all([
          apiClient.getWorkflowExecutionStatus(workflowId),
          apiClient.listWorkflowTasks(workflowId),
          apiClient.listWorkflowHistory(workflowId),
        ]);

        if (!cancelled) {
          setSnapshot({
            workflow: statusResult.data,
            tasks: tasksResult.data || [],
            history: historyResult.data || [],
          });
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    const startStream = async () => {
      const token = localStorage.getItem('access_token');
      if (!token) {
        return;
      }

      setStreaming(true);
      await fetchEventSource(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000'}/workflows/${workflowId}/stream`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
        onmessage(event) {
          if (cancelled) return;

          try {
            if (event.event === 'snapshot') {
              const nextSnapshot = JSON.parse(event.data) as WorkflowSnapshot;
              setSnapshot(nextSnapshot);
            } else if (event.event === 'event') {
              const executionEvent = JSON.parse(event.data);
              // Handle granular event to update state incrementally if needed
              // For now, we'll just append to history and trigger a refresh of tasks
              setSnapshot(prev => {
                const newHistory = [...(prev.history || [])];
                
                // Add the transition event to history
                newHistory.push({
                  id: executionEvent.event_id,
                  entity_type: executionEvent.task_id ? 'task' : 'workflow',
                  from_state: executionEvent.data?.from_state || 'unknown',
                  to_state: executionEvent.data?.to_state || executionEvent.event_type,
                  reason: executionEvent.error_message || executionEvent.data?.reason,
                  created_at: executionEvent.timestamp,
                  worker_id: executionEvent.worker_id,
                });

                return {
                  ...prev,
                  history: newHistory,
                };
              });
            }
          } catch (e) {
            console.error('Failed to parse stream event', e);
          }
        },
        onclose() {
          if (!cancelled) {
            setStreaming(false);
          }
        },
        onerror() {
          if (!cancelled) {
            setStreaming(false);
          }
          throw new Error('workflow stream closed');
        },
      });
    };

    loadInitial();
    void startStream().catch(() => undefined);

    return () => {
      cancelled = true;
    };
  }, [isAuthenticated, router, workflowId]);

  const tasks = snapshot.tasks || [];
  const history = snapshot.history || [];

  const handleExecute = async () => {
    await apiClient.executeWorkflow(workflowId);
  };

  const handleCancel = async () => {
    await apiClient.cancelWorkflow(workflowId);
  };

  return (
    <div className="min-h-screen bg-slate-950 text-white">
      <div className="border-b border-slate-800 bg-slate-900/70 backdrop-blur">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
          <div>
            <div className="text-xs uppercase tracking-[0.35em] text-slate-400">Execution Control Plane</div>
            <h1 className="text-2xl font-semibold">Workflow {workflowId}</h1>
            <p className="text-sm text-slate-400">
              Live DAG execution, retries, and state transitions{streaming ? ' • streaming' : ''}
            </p>
          </div>
          <div className="flex gap-3">
            <button onClick={handleExecute} className="rounded-md bg-emerald-500 px-4 py-2 text-sm font-semibold text-white hover:bg-emerald-400">
              Execute
            </button>
            <button onClick={handleCancel} className="rounded-md border border-slate-700 px-4 py-2 text-sm font-semibold text-slate-200 hover:bg-slate-800">
              Cancel
            </button>
          </div>
        </div>
      </div>

      <div className="mx-auto grid max-w-7xl gap-6 px-6 py-8 xl:grid-cols-[1.4fr_0.8fr]">
        <section className="space-y-6">
          <div className="grid gap-4 md:grid-cols-3">
            <Metric label="Status" value={snapshot.workflow?.status || 'loading'} />
            <Metric label="Tasks" value={tasks.length.toString()} />
            <Metric label="History Events" value={history.length.toString()} />
          </div>

          {loading ? (
            <div className="rounded-xl border border-slate-700 bg-slate-900/70 p-6 text-slate-400">Loading execution snapshot...</div>
          ) : (
            <ExecutionGraph tasks={tasks} />
          )}
        </section>

        <aside className="space-y-6">
          <div className="rounded-2xl border border-slate-800 bg-slate-900/80 p-5">
            <h2 className="text-lg font-semibold">Execution State</h2>
            <div className="mt-4 space-y-3 text-sm text-slate-300">
              <Row label="Workflow" value={snapshot.workflow?.name || 'Unknown'} />
              <Row label="Last updated" value={snapshot.workflow?.updated_at || 'n/a'} />
              <Row label="Streaming" value={streaming ? 'Active' : 'Idle'} />
            </div>
          </div>

          <div className="rounded-2xl border border-slate-800 bg-slate-900/80 p-5">
            <h2 className="text-lg font-semibold">Execution Timeline</h2>
            <div className="mt-4">
              <ExecutionTimeline events={history} />
            </div>
          </div>
        </aside>
      </div>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/80 p-4">
      <div className="text-xs uppercase tracking-[0.3em] text-slate-500">{label}</div>
      <div className="mt-2 text-2xl font-semibold text-white">{value}</div>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-start justify-between gap-4 border-b border-slate-800 pb-2">
      <span className="text-slate-500">{label}</span>
      <span className="text-right text-slate-200">{value}</span>
    </div>
  );
}
