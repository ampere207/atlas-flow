'use client';

import Link from 'next/link';
import { useEffect, useMemo, useState } from 'react';

import Sidebar from '@/components/Sidebar';
import { apiClient } from '@/lib/api-client';
import { useAuthStore } from '@/store/auth';

interface Worker {
  worker_id: string;
  user_id: string;
  status: string;
  capabilities: string[];
  capacity: number;
  running_tasks: number;
  completed_tasks: number;
  failed_tasks: number;
  last_heartbeat: string;
}

export default function WorkerClusterPage() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated());
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!isAuthenticated) {
      return;
    }

    let active = true;

    const load = async () => {
      setLoading(true);
      try {
        const result = await apiClient.listWorkers(50, 0);
        if (active) {
          setWorkers(result || []);
        }
      } catch (error) {
        console.error('Failed to load workers:', error);
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    };

    load();
    const interval = setInterval(load, 5000);
    return () => {
      active = false;
      clearInterval(interval);
    };
  }, [isAuthenticated]);

  const activeCount = useMemo(
    () => workers.filter((worker) => worker.status === 'active' || worker.status === 'idle' || worker.status === 'connected').length,
    [workers]
  );
  const offlineCount = Math.max(workers.length - activeCount, 0);

  return (
    <div className="flex min-h-screen bg-[#050816] text-white">
      <Sidebar />
      <main className="flex-1 ml-64 overflow-auto">
        <div className="min-h-screen bg-[radial-gradient(circle_at_top_left,_rgba(59,130,246,0.18),_transparent_30%),linear-gradient(180deg,_#081120_0%,_#050816_100%)]">
          <header className="sticky top-0 z-20 border-b border-white/5 bg-[#07111d]/70 backdrop-blur-xl">
            <div className="flex items-center justify-between gap-4 px-8 py-6">
              <div>
                <p className="text-xs uppercase tracking-[0.3em] text-cyan-300/70">Workers</p>
                <h1 className="mt-2 text-3xl font-semibold text-white">Worker cluster</h1>
                <p className="mt-1 text-sm text-slate-400">Live execution nodes and heartbeat visibility.</p>
              </div>
              <Link href="/workers" className="rounded-full border border-cyan-400/20 bg-cyan-400/10 px-4 py-2 text-sm text-cyan-200 transition hover:bg-cyan-400/20">
                Register worker
              </Link>
            </div>
          </header>

          <div className="p-8 space-y-6">
            <section className="grid gap-4 md:grid-cols-3">
              <MetricCard label="Total workers" value={workers.length} tone="blue" />
              <MetricCard label="Active / idle" value={activeCount} tone="emerald" />
              <MetricCard label="Offline" value={offlineCount} tone="rose" />
            </section>

            <section className="overflow-hidden rounded-3xl border border-white/10 bg-white/5 shadow-2xl shadow-black/20 backdrop-blur-xl">
              <div className="flex items-center justify-between border-b border-white/10 px-6 py-5">
                <div>
                  <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Cluster inventory</p>
                  <h2 className="mt-2 text-xl font-semibold text-white">Registered workers</h2>
                </div>
                <span className="rounded-full bg-slate-950/60 px-3 py-1 text-xs text-slate-300">{workers.length} nodes</span>
              </div>

              <div className="p-6">
                {loading ? (
                  <div className="flex items-center justify-center py-14">
                    <div className="h-10 w-10 animate-spin rounded-full border-2 border-cyan-400 border-t-transparent" />
                  </div>
                ) : workers.length === 0 ? (
                  <EmptyState
                    title="No workers registered"
                    description="Add a worker on the registration page to begin monitoring the cluster."
                    actionHref="/workers"
                    actionLabel="Register worker"
                  />
                ) : (
                  <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                    {workers.map((worker) => (
                      <article key={worker.worker_id} className="rounded-2xl border border-white/10 bg-slate-950/40 p-5 transition hover:border-cyan-400/30 hover:bg-slate-950/60">
                        <div className="flex items-start justify-between gap-4">
                          <div className="min-w-0">
                            <h3 className="truncate text-lg font-semibold text-white">{worker.worker_id}</h3>
                            <p className="mt-1 text-xs text-slate-400">{worker.capabilities?.join(', ') || 'none'}</p>
                          </div>
                          <WorkerBadge status={worker.status} />
                        </div>

                        <div className="mt-4 grid grid-cols-3 gap-2">
                          <div className="rounded-xl border border-white/10 bg-white/5 p-3 text-center">
                            <p className="text-[10px] uppercase tracking-[0.2em] text-slate-400">Tasks</p>
                            <p className="mt-1 text-lg font-semibold text-white">{worker.running_tasks}/{worker.capacity}</p>
                          </div>
                          <div className="rounded-xl border border-white/10 bg-white/5 p-3 text-center">
                            <p className="text-[10px] uppercase tracking-[0.2em] text-slate-400">Done</p>
                            <p className="mt-1 text-lg font-semibold text-emerald-300">{worker.completed_tasks}</p>
                          </div>
                          <div className="rounded-xl border border-white/10 bg-white/5 p-3 text-center">
                            <p className="text-[10px] uppercase tracking-[0.2em] text-slate-400">Failed</p>
                            <p className="mt-1 text-lg font-semibold text-rose-300">{worker.failed_tasks}</p>
                          </div>
                        </div>

                        <div className="mt-3 rounded-xl border border-white/10 bg-white/5 p-4">
                          <p className="text-[11px] uppercase tracking-[0.25em] text-slate-400">Last heartbeat</p>
                          <p className="mt-2 text-sm text-slate-200">{new Date(worker.last_heartbeat).toLocaleString()}</p>
                        </div>
                      </article>
                    ))}
                  </div>
                )}
              </div>
            </section>
          </div>
        </div>
      </main>
    </div>
  );
}

function MetricCard({ label, value, tone }: { label: string; value: number; tone: 'blue' | 'emerald' | 'rose' }) {
  const classes =
    tone === 'blue'
      ? 'from-blue-500/20 to-cyan-500/10 text-cyan-200'
      : tone === 'emerald'
        ? 'from-emerald-500/20 to-green-500/10 text-emerald-200'
        : 'from-rose-500/20 to-red-500/10 text-rose-200';

  return (
    <div className={`rounded-3xl border border-white/10 bg-gradient-to-br ${classes} p-5 shadow-xl shadow-black/10`}>
      <p className="text-[11px] uppercase tracking-[0.25em] text-slate-300/70">{label}</p>
      <p className="mt-3 text-4xl font-semibold text-white">{value}</p>
    </div>
  );
}

function WorkerBadge({ status }: { status: string }) {
  const key = status.toLowerCase();
  const classes =
    key === 'active' || key === 'connected'
      ? 'bg-emerald-500/15 text-emerald-200 border-emerald-400/20'
      : key === 'idle'
        ? 'bg-cyan-500/15 text-cyan-200 border-cyan-400/20'
        : 'bg-rose-500/15 text-rose-200 border-rose-400/20';

  return <span className={`shrink-0 rounded-full border px-3 py-1 text-xs font-medium capitalize ${classes}`}>{status}</span>;
}

function EmptyState({ title, description, actionHref, actionLabel }: { title: string; description: string; actionHref: string; actionLabel: string }) {
  return (
    <div className="rounded-2xl border border-dashed border-white/10 bg-slate-950/35 px-6 py-12 text-center">
      <p className="text-sm font-medium text-white">{title}</p>
      <p className="mt-2 text-sm text-slate-400">{description}</p>
      <Link href={actionHref} className="mt-4 inline-flex rounded-full bg-cyan-500/15 px-4 py-2 text-sm font-medium text-cyan-200 transition hover:bg-cyan-500/25">
        {actionLabel}
      </Link>
    </div>
  );
}
